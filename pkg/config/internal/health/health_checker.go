package health

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/config/iface"
)

// HealthChecker implements health monitoring for configuration providers
type HealthChecker struct {
	mu               sync.RWMutex
	providers        map[string]iface.Provider
	lastHealthChecks map[string]HealthStatus
	enabled          bool
	checkInterval    time.Duration
	stopChan         chan struct{}
	callbacks        []HealthCallback
}

// HealthStatus represents the health state of a component
type HealthStatus struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Details   map[string]interface{} `json:"details,omitempty"`
	LastError string                 `json:"last_error,omitempty"`
	CheckID   string                 `json:"check_id"`
}

// HealthCallback is called when health status changes
type HealthCallback func(status HealthStatus)

// HealthCheckerOptions configures the health checker
type HealthCheckerOptions struct {
	CheckInterval time.Duration
	Enabled       bool
	MaxHistory    int
}

// NewHealthChecker creates a new health checker with the given options
func NewHealthChecker(options HealthCheckerOptions) (*HealthChecker, error) {
	if options.CheckInterval == 0 {
		options.CheckInterval = 30 * time.Second
	}
	if options.MaxHistory == 0 {
		options.MaxHistory = 100
	}

	return &HealthChecker{
		providers:        make(map[string]iface.Provider),
		lastHealthChecks: make(map[string]HealthStatus),
		enabled:          options.Enabled,
		checkInterval:    options.CheckInterval,
		stopChan:         make(chan struct{}),
		callbacks:        make([]HealthCallback, 0),
	}, nil
}

// HealthCheck performs a health check and returns current status
func (hc *HealthChecker) HealthCheck(ctx context.Context) HealthStatus {
	status := HealthStatus{
		Timestamp: time.Now(),
		CheckID:   fmt.Sprintf("check-%d", time.Now().UnixNano()),
		Details:   make(map[string]interface{}),
	}

	// Check if context is cancelled
	select {
	case <-ctx.Done():
		status.Status = "unhealthy"
		status.LastError = "context cancelled"
		status.Details["error"] = ctx.Err().Error()
		return status
	default:
	}

	// Perform actual health check
	hc.mu.RLock()
	providerCount := len(hc.providers)
	enabled := hc.enabled
	hc.mu.RUnlock()

	if !enabled {
		status.Status = "disabled"
		status.Details["message"] = "health checking disabled"
		return status
	}

	if providerCount == 0 {
		status.Status = "unknown"
		status.Details["message"] = "no providers to check"
		return status
	}

	// Check providers
	healthyCount := 0
	degradedCount := 0
	unhealthyCount := 0

	hc.mu.RLock()
	for name, provider := range hc.providers {
		providerStatus := hc.checkProvider(ctx, name, provider)
		
		switch providerStatus {
		case "healthy":
			healthyCount++
		case "degraded":
			degradedCount++
		case "unhealthy":
			unhealthyCount++
		}
	}
	hc.mu.RUnlock()

	// Determine overall status
	totalCount := healthyCount + degradedCount + unhealthyCount
	status.Details["healthy_count"] = healthyCount
	status.Details["degraded_count"] = degradedCount
	status.Details["unhealthy_count"] = unhealthyCount
	status.Details["total_count"] = totalCount

	if unhealthyCount > 0 {
		status.Status = "unhealthy"
	} else if degradedCount > 0 {
		status.Status = "degraded"
	} else if healthyCount > 0 {
		status.Status = "healthy"
	} else {
		status.Status = "unknown"
	}

	// Store result and notify callbacks
	hc.mu.Lock()
	hc.lastHealthChecks["system"] = status
	hc.mu.Unlock()

	hc.notifyCallbacks(status)

	return status
}

// checkProvider checks the health of a specific provider
func (hc *HealthChecker) checkProvider(ctx context.Context, name string, provider iface.Provider) string {
	// For providers that implement health checking, call their method
	if healthCheckProvider, ok := provider.(interface {
		HealthCheck(ctx context.Context) HealthStatus
	}); ok {
		providerHealth := healthCheckProvider.HealthCheck(ctx)
		
		hc.mu.Lock()
		hc.lastHealthChecks[name] = providerHealth
		hc.mu.Unlock()
		
		return providerHealth.Status
	}

	// For providers without health checking, do basic validation
	err := provider.Validate()
	if err != nil {
		providerStatus := HealthStatus{
			Status:    "unhealthy",
			Timestamp: time.Now(),
			LastError: err.Error(),
			Details:   map[string]interface{}{"provider": name},
		}
		
		hc.mu.Lock()
		hc.lastHealthChecks[name] = providerStatus
		hc.mu.Unlock()
		
		return "unhealthy"
	}

	// Provider is responsive
	providerStatus := HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Details:   map[string]interface{}{"provider": name},
	}
	
	hc.mu.Lock()
	hc.lastHealthChecks[name] = providerStatus
	hc.mu.Unlock()
	
	return "healthy"
}

// StartHealthChecks begins periodic health checking
func (hc *HealthChecker) StartHealthChecks(ctx context.Context, interval time.Duration) error {
	hc.mu.Lock()
	if !hc.enabled {
		hc.mu.Unlock()
		return fmt.Errorf("health checker is disabled")
	}
	
	hc.checkInterval = interval
	hc.mu.Unlock()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				_ = hc.HealthCheck(ctx) // Ignore errors in periodic checks
			case <-hc.stopChan:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// StopHealthChecks stops periodic health checking
func (hc *HealthChecker) StopHealthChecks() error {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	select {
	case hc.stopChan <- struct{}{}:
	default:
		// Channel might be closed or blocking
	}

	return nil
}

// GetHealthHistory returns historical health data
func (hc *HealthChecker) GetHealthHistory(limit int) []HealthStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	var history []HealthStatus
	for _, status := range hc.lastHealthChecks {
		history = append(history, status)
	}

	// Sort by timestamp (newest first)
	for i := 0; i < len(history)-1; i++ {
		for j := i + 1; j < len(history); j++ {
			if history[i].Timestamp.Before(history[j].Timestamp) {
				history[i], history[j] = history[j], history[i]
			}
		}
	}

	// Apply limit
	if limit > 0 && len(history) > limit {
		history = history[:limit]
	}

	return history
}

// RegisterHealthCallback registers a callback for health status changes
func (hc *HealthChecker) RegisterHealthCallback(callback HealthCallback) error {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.callbacks = append(hc.callbacks, callback)
	return nil
}

// notifyCallbacks notifies all registered callbacks of health status change
func (hc *HealthChecker) notifyCallbacks(status HealthStatus) {
	hc.mu.RLock()
	callbacks := make([]HealthCallback, len(hc.callbacks))
	copy(callbacks, hc.callbacks)
	hc.mu.RUnlock()

	for _, callback := range callbacks {
		go func(cb HealthCallback) {
			defer func() {
				if r := recover(); r != nil {
					// Log panic in callback but don't crash health checker
				}
			}()
			cb(status)
		}(callback)
	}
}

// AddProvider adds a provider to be monitored
func (hc *HealthChecker) AddProvider(name string, provider iface.Provider) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	hc.providers[name] = provider
}

// RemoveProvider removes a provider from monitoring
func (hc *HealthChecker) RemoveProvider(name string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	delete(hc.providers, name)
	delete(hc.lastHealthChecks, name)
}

// GetProviderHealth returns the last health check result for a specific provider
func (hc *HealthChecker) GetProviderHealth(name string) HealthStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	
	if status, exists := hc.lastHealthChecks[name]; exists {
		return status
	}
	
	return HealthStatus{
		Status:    "unknown",
		Timestamp: time.Now(),
		Details:   map[string]interface{}{"provider": name, "message": "no health data available"},
	}
}
