// Package health provides health monitoring implementations
package health

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/monitoring/iface"
	"github.com/lookatitude/beluga-ai/pkg/monitoring/internal/logger"
)

// HealthStatus represents the current health status of a component.
type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusDegraded  HealthStatus = "degraded"
	StatusUnhealthy HealthStatus = "unhealthy"
	StatusUnknown   HealthStatus = "unknown"
)

// HealthCheckResult contains the result of a health check.
type HealthCheckResult struct {
	Status      HealthStatus
	Message     string
	Timestamp   time.Time
	Details     map[string]any
	CheckName   string
	ComponentID string
}

// HealthCheckFunc is a function that performs a health check and returns a result.
type HealthCheckFunc func() *HealthCheckResult

// AlertFunc is a function called when a health check status changes.
type AlertFunc func(result *HealthCheckResult)

// HealthCheck defines a periodic health check mechanism.
type HealthCheck struct {
	Logger      iface.Logger
	Check       HealthCheckFunc
	StopChan    chan struct{}
	LastResult  *HealthCheckResult
	Name        string
	ComponentID string
	Alerts      []AlertFunc
	Interval    time.Duration
	Timeout     time.Duration
	MaxRetries  int
	RetryDelay  time.Duration
	mutex       sync.RWMutex
}

// NewHealthCheck creates a new health check.
func NewHealthCheck(name, componentID string, interval time.Duration, check HealthCheckFunc) *HealthCheck {
	return &HealthCheck{
		Name:        name,
		ComponentID: componentID,
		Interval:    interval,
		Timeout:     time.Second * 10,
		Check:       check,
		StopChan:    make(chan struct{}),
		LastResult: &HealthCheckResult{
			Status:      StatusUnknown,
			Message:     "Health check not started",
			Timestamp:   time.Now(),
			CheckName:   name,
			ComponentID: componentID,
			Details:     make(map[string]any),
		},
		MaxRetries: 3,
		RetryDelay: time.Second * 2,
		Alerts:     make([]AlertFunc, 0),
		Logger:     logger.NewStructuredLogger("health_check_" + name),
	}
}

// Start begins the periodic health check.
func (hc *HealthCheck) Start() {
	hc.Logger.Info(context.Background(), "Starting health check",
		map[string]any{
			"name":         hc.Name,
			"component_id": hc.ComponentID,
			"interval":     hc.Interval,
		})

	go func() {
		ticker := time.NewTicker(hc.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				hc.RunCheck()
			case <-hc.StopChan:
				hc.Logger.Info(context.Background(), "Health check stopped",
					map[string]any{
						"name":         hc.Name,
						"component_id": hc.ComponentID,
					})
				return
			}
		}
	}()
}

// RunCheck executes the health check once.
func (hc *HealthCheck) RunCheck() {
	var result *HealthCheckResult
	var prevStatus HealthStatus

	hc.mutex.RLock()
	if hc.LastResult != nil {
		prevStatus = hc.LastResult.Status
	}
	hc.mutex.RUnlock()

	// Create a timeout context for the check
	checkComplete := make(chan struct{})
	var checkResult *HealthCheckResult

	go func() {
		checkResult = hc.Check()
		close(checkComplete)
	}()

	// Wait for check to complete or timeout
	select {
	case <-checkComplete:
		result = checkResult
	case <-time.After(hc.Timeout):
		result = &HealthCheckResult{
			Status:      StatusUnhealthy,
			Message:     fmt.Sprintf("Health check timed out after %v", hc.Timeout),
			Timestamp:   time.Now(),
			CheckName:   hc.Name,
			ComponentID: hc.ComponentID,
		}
	}

	if result == nil {
		result = &HealthCheckResult{
			Status:      StatusUnhealthy,
			Message:     "Health check returned nil result",
			Timestamp:   time.Now(),
			CheckName:   hc.Name,
			ComponentID: hc.ComponentID,
		}
	}

	// Retry logic for failed checks
	attempts := 1
	for result.Status == StatusUnhealthy && attempts <= hc.MaxRetries {
		hc.Logger.Warning(context.Background(), "Health check failed, retrying",
			map[string]any{
				"name":         hc.Name,
				"component_id": hc.ComponentID,
				"attempts":     attempts,
				"max_retries":  hc.MaxRetries,
			})
		time.Sleep(hc.RetryDelay)

		retryResult := hc.Check()
		if retryResult != nil && retryResult.Status != StatusUnhealthy {
			result = retryResult
			result.Message = fmt.Sprintf("Recovered on retry %d: %s", attempts, result.Message)
			break
		}
		attempts++
	}

	// Update last result
	hc.mutex.Lock()
	hc.LastResult = result
	hc.mutex.Unlock()

	switch result.Status {
	case StatusUnhealthy:
		hc.Logger.Error(context.Background(), "Health check failed",
			map[string]any{
				"name":         hc.Name,
				"component_id": hc.ComponentID,
				"status":       string(result.Status),
				"message":      result.Message,
			})
	case StatusDegraded:
		hc.Logger.Warning(context.Background(), "Health check degraded",
			map[string]any{
				"name":         hc.Name,
				"component_id": hc.ComponentID,
				"status":       string(result.Status),
				"message":      result.Message,
			})
	default:
		hc.Logger.Info(context.Background(), "Health check passed",
			map[string]any{
				"name":         hc.Name,
				"component_id": hc.ComponentID,
				"status":       string(result.Status),
				"message":      result.Message,
			})
	}

	// Trigger alerts if status changed
	if prevStatus != result.Status {
		hc.Logger.Info(context.Background(), "Health status changed",
			map[string]any{
				"name":            hc.Name,
				"component_id":    hc.ComponentID,
				"previous_status": prevStatus,
				"current_status":  result.Status,
			})
		hc.triggerAlerts(result)
	}
}

// triggerAlerts notifies all registered alert handlers.
func (hc *HealthCheck) triggerAlerts(result *HealthCheckResult) {
	for _, alert := range hc.Alerts {
		go func(alertFunc AlertFunc) {
			defer func() {
				if r := recover(); r != nil {
					hc.Logger.Error(context.Background(), "Alert handler panicked",
						map[string]any{
							"error": r,
						})
				}
			}()
			alertFunc(result)
		}(alert)
	}
}

// RegisterAlert adds an alert handler to be called when health status changes.
func (hc *HealthCheck) RegisterAlert(alertFunc AlertFunc) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()
	hc.Alerts = append(hc.Alerts, alertFunc)
}

// GetLastResult returns the most recent health check result.
func (hc *HealthCheck) GetLastResult() *HealthCheckResult {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()
	return hc.LastResult
}

// Stop halts the periodic health check.
func (hc *HealthCheck) Stop() {
	close(hc.StopChan)
}

// HealthCheckManager manages multiple health checks.
type HealthCheckManager struct {
	Logger iface.Logger
	checks map[string]*HealthCheck
	mutex  sync.RWMutex
}

// NewHealthCheckManager creates a new health check manager.
func NewHealthCheckManager() *HealthCheckManager {
	return &HealthCheckManager{
		checks: make(map[string]*HealthCheck),
		Logger: logger.NewStructuredLogger("health_check_manager"),
	}
}

// AddCheck registers a health check with the manager.
func (hcm *HealthCheckManager) AddCheck(check *HealthCheck) error {
	hcm.mutex.Lock()
	defer hcm.mutex.Unlock()

	checkID := check.ComponentID + ":" + check.Name
	if _, exists := hcm.checks[checkID]; exists {
		return fmt.Errorf("health check with ID %s already exists", checkID)
	}

	hcm.checks[checkID] = check
	hcm.Logger.Info(context.Background(), "Added health check",
		map[string]any{
			"name":         check.Name,
			"component_id": check.ComponentID,
		})
	return nil
}

// RemoveCheck unregisters a health check from the manager.
func (hcm *HealthCheckManager) RemoveCheck(componentID, name string) error {
	checkID := componentID + ":" + name

	hcm.mutex.Lock()
	defer hcm.mutex.Unlock()

	check, exists := hcm.checks[checkID]
	if !exists {
		return fmt.Errorf("health check with ID %s not found", checkID)
	}

	check.Stop()
	delete(hcm.checks, checkID)
	hcm.Logger.Info(context.Background(), "Removed health check",
		map[string]any{
			"name":         name,
			"component_id": componentID,
		})
	return nil
}

// StartAllChecks begins all registered health checks.
func (hcm *HealthCheckManager) StartAllChecks() {
	hcm.mutex.RLock()
	defer hcm.mutex.RUnlock()

	for _, check := range hcm.checks {
		check.Start()
	}
	hcm.Logger.Info(context.Background(), "Started all health checks",
		map[string]any{
			"total_checks": len(hcm.checks),
		})
}

// StopAllChecks halts all registered health checks.
func (hcm *HealthCheckManager) StopAllChecks() {
	hcm.mutex.RLock()
	defer hcm.mutex.RUnlock()

	for _, check := range hcm.checks {
		check.Stop()
	}
	hcm.Logger.Info(context.Background(), "Stopped all health checks", nil)
}

// GetCheckResults returns the results of all health checks.
func (hcm *HealthCheckManager) GetCheckResults() map[string]*HealthCheckResult {
	hcm.mutex.RLock()
	defer hcm.mutex.RUnlock()

	results := make(map[string]*HealthCheckResult, len(hcm.checks))
	for id, check := range hcm.checks {
		results[id] = check.GetLastResult()
	}
	return results
}

// CheckSystemHealth returns the overall health status of the system.
func (hcm *HealthCheckManager) CheckSystemHealth() (HealthStatus, map[string]*HealthCheckResult) {
	results := hcm.GetCheckResults()

	overallStatus := StatusHealthy
	for _, result := range results {
		if result.Status == StatusUnhealthy {
			return StatusUnhealthy, results
		}
		if result.Status == StatusDegraded {
			overallStatus = StatusDegraded
		}
	}

	return overallStatus, results
}

// CreateAgentHealthCheckFunc creates a health check function for an agent.
func CreateAgentHealthCheckFunc(getHealthFunc func() map[string]any) HealthCheckFunc {
	return func() *HealthCheckResult {
		health := getHealthFunc()

		status := StatusHealthy
		message := "Agent is healthy"

		// Check agent state
		if agentState, ok := health["state"].(string); ok {
			switch agentState {
			case "error":
				status = StatusUnhealthy
				message = "Agent is in error state"
			case "paused":
				status = StatusDegraded
				message = "Agent is paused"
			}
		}

		// Check error count
		if errorCount, ok := health["error_count"].(int); ok && errorCount > 0 {
			if errorCount > 5 {
				status = StatusUnhealthy
				message = fmt.Sprintf("Agent has high error count: %d", errorCount)
			} else {
				status = StatusDegraded
				message = fmt.Sprintf("Agent has errors: %d", errorCount)
			}
		}

		return &HealthCheckResult{
			Status:      status,
			Message:     message,
			Timestamp:   time.Now(),
			Details:     health,
			CheckName:   "agent_health",
			ComponentID: health["name"].(string),
		}
	}
}
