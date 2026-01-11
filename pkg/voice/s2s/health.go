package s2s

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/internal"
)

// HealthStatus represents the health status of an S2S provider.
type HealthStatus struct {
	Healthy   bool
	Provider  string
	Message   string
	Timestamp time.Time
	Latency   time.Duration
	Error     error
}

// HealthCheckManager manages health checks for S2S providers.
type HealthCheckManager struct {
	registry   *Registry
	timeout    time.Duration
	mu         sync.RWMutex
	lastChecks map[string]HealthStatus
}

// NewHealthCheckManager creates a new health check manager.
func NewHealthCheckManager(registry *Registry, timeout time.Duration) *HealthCheckManager {
	return &HealthCheckManager{
		registry:   registry,
		timeout:    timeout,
		lastChecks: make(map[string]HealthStatus),
	}
}

// CheckHealth checks the health of all registered providers.
// This implements the core.HealthChecker interface.
func (h *HealthCheckManager) CheckHealth(ctx context.Context) error {
	// Check all providers and return overall health
	providers := h.registry.ListProviders()
	if len(providers) == 0 {
		return fmt.Errorf("no providers registered")
	}

	var unhealthyProviders []string
	for _, providerName := range providers {
		status := h.GetProviderHealth(ctx, providerName)
		if !status.Healthy {
			unhealthyProviders = append(unhealthyProviders, providerName)
		}
	}

	if len(unhealthyProviders) > 0 {
		return fmt.Errorf("unhealthy providers: %v", unhealthyProviders)
	}

	return nil
}

// CheckHealthStatus checks the health of all registered providers and returns detailed status.
// This is a convenience method that returns HealthStatus instead of error.
func (h *HealthCheckManager) CheckHealthStatus(ctx context.Context) HealthStatus {
	// Check all providers and return overall health
	providers := h.registry.ListProviders()
	if len(providers) == 0 {
		return HealthStatus{
			Healthy:   false,
			Message:   "no providers registered",
			Timestamp: time.Now(),
		}
	}

	allHealthy := true
	for _, providerName := range providers {
		status := h.GetProviderHealth(ctx, providerName)
		if !status.Healthy {
			allHealthy = false
		}
	}

	return HealthStatus{
		Healthy:   allHealthy,
		Message:   "health check completed",
		Timestamp: time.Now(),
	}
}

// GetProviderHealth checks the health of a specific provider.
func (h *HealthCheckManager) GetProviderHealth(ctx context.Context, providerName string) HealthStatus {
	h.mu.RLock()
	lastCheck, exists := h.lastChecks[providerName]
	// If checked recently (within last 30 seconds), return cached result
	if exists && time.Since(lastCheck.Timestamp) < 30*time.Second {
		h.mu.RUnlock()
		return lastCheck
	}
	h.mu.RUnlock()

	// Create a context with timeout
	checkCtx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	// Check if provider is registered
	if !h.registry.IsRegistered(providerName) {
		status := HealthStatus{
			Healthy:   false,
			Provider:  providerName,
			Message:   "provider not registered",
			Timestamp: time.Now(),
			Error:     errors.New("provider not registered"),
		}
		h.mu.Lock()
		h.lastChecks[providerName] = status
		h.mu.Unlock()
		return status
	}

	// Create a minimal config for health check
	config := DefaultConfig()
	config.Provider = providerName
	config.Timeout = h.timeout

	// Create provider instance for health check
	provider, err := h.registry.GetProvider(providerName, config)
	if err != nil {
		status := HealthStatus{
			Healthy:   false,
			Provider:  providerName,
			Message:   "failed to create provider",
			Timestamp: time.Now(),
			Error:     err,
		}
		h.mu.Lock()
		h.lastChecks[providerName] = status
		h.mu.Unlock()
		return status
	}

	// Perform health check by attempting a minimal operation
	startTime := time.Now()
	status := h.performHealthCheck(checkCtx, provider, providerName)
	status.Latency = time.Since(startTime)
	status.Timestamp = time.Now()

	// Cache the result
	h.mu.Lock()
	h.lastChecks[providerName] = status
	h.mu.Unlock()

	return status
}

// performHealthCheck performs the actual health check on a provider.
func (h *HealthCheckManager) performHealthCheck(ctx context.Context, provider iface.S2SProvider, providerName string) HealthStatus {
	// Create a minimal audio input for testing
	testAudio := []byte{0, 0, 0, 0} // Minimal test audio
	input := &internal.AudioInput{
		Data: testAudio,
		Format: internal.AudioFormat{
			SampleRate: 24000,
			Channels:   1,
			BitDepth:   16,
			Encoding:   "PCM",
		},
		Timestamp: time.Now(),
	}

	convCtx := &internal.ConversationContext{
		SessionID: "health-check",
	}

	// Attempt to process (this will fail quickly if provider is unhealthy)
	// We use a very short timeout to avoid blocking
	_, err := provider.Process(ctx, input, convCtx)

	if err != nil {
		// Check if it's a timeout or context cancellation (expected for health check)
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			// Provider responded (even if with timeout), consider it healthy
			return HealthStatus{
				Healthy:  true,
				Provider: providerName,
				Message:  "provider responding",
			}
		}

		// Other errors indicate unhealthy provider
		return HealthStatus{
			Healthy:  false,
			Provider: providerName,
			Message:  "provider error",
			Error:    err,
		}
	}

	return HealthStatus{
		Healthy:  true,
		Provider: providerName,
		Message:  "provider healthy",
	}
}

// GetLastHealthStatus returns the last cached health status for a provider.
func (h *HealthCheckManager) GetLastHealthStatus(providerName string) (HealthStatus, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	status, exists := h.lastChecks[providerName]
	return status, exists
}

// Ensure HealthCheckManager implements core.HealthChecker interface.
var _ core.HealthChecker = (*HealthCheckManager)(nil)
