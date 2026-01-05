package s2s

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/internal"
)

// ProviderFallback manages fallback switching between S2S providers.
type ProviderFallback struct {
	primary       iface.S2SProvider
	fallbacks     []iface.S2SProvider
	breaker       *CircuitBreaker
	usingFallback bool
	currentIndex  int // Index of current provider (0 = primary, 1+ = fallback)
	mu            sync.RWMutex
}

// NewProviderFallback creates a new S2S provider fallback manager.
func NewProviderFallback(primary iface.S2SProvider, fallbacks []iface.S2SProvider, breaker *CircuitBreaker) *ProviderFallback {
	return &ProviderFallback{
		primary:       primary,
		fallbacks:     fallbacks,
		breaker:       breaker,
		usingFallback: false,
		currentIndex:  0,
	}
}

// GetProvider returns the current provider (primary or fallback).
func (pf *ProviderFallback) GetProvider() iface.S2SProvider {
	pf.mu.RLock()
	defer pf.mu.RUnlock()

	if pf.currentIndex == 0 {
		return pf.primary
	}

	// Return fallback provider at index-1 (since index 0 is primary)
	fallbackIndex := pf.currentIndex - 1
	if fallbackIndex < len(pf.fallbacks) {
		return pf.fallbacks[fallbackIndex]
	}

	// Fallback to primary if index is out of range
	return pf.primary
}

// SwitchToFallback switches to the next fallback provider.
func (pf *ProviderFallback) SwitchToFallback() bool {
	pf.mu.Lock()
	defer pf.mu.Unlock()

	// Try next fallback provider
	if pf.currentIndex < len(pf.fallbacks) {
		pf.currentIndex++
		pf.usingFallback = pf.currentIndex > 0
		return true
	}

	// No more fallbacks available
	return false
}

// SwitchToPrimary switches back to the primary provider.
func (pf *ProviderFallback) SwitchToPrimary() {
	pf.mu.Lock()
	defer pf.mu.Unlock()
	pf.currentIndex = 0
	pf.usingFallback = false
}

// IsUsingFallback returns whether fallback is currently active.
func (pf *ProviderFallback) IsUsingFallback() bool {
	pf.mu.RLock()
	defer pf.mu.RUnlock()
	return pf.usingFallback
}

// GetCurrentProviderName returns the name of the current provider.
func (pf *ProviderFallback) GetCurrentProviderName() string {
	provider := pf.GetProvider()
	if provider != nil {
		return provider.Name()
	}
	return "unknown"
}

// ProcessWithFallback processes audio with automatic fallback on failure.
func (pf *ProviderFallback) ProcessWithFallback(ctx context.Context, input *internal.AudioInput, convCtx *internal.ConversationContext, opts ...internal.STSOption) (*internal.AudioOutput, error) {
	var lastErr error
	providers := []iface.S2SProvider{pf.primary}
	providers = append(providers, pf.fallbacks...)

	// Try each provider in order
	for i, provider := range providers {
		if provider == nil {
			continue
		}

		// Try primary first with circuit breaker
		if i == 0 {
			var output *internal.AudioOutput
			err := pf.breaker.Call(func() error {
				var callErr error
				output, callErr = provider.Process(ctx, input, convCtx, opts...)
				return callErr
			})

			if err == nil && output != nil {
				// Success - switch back to primary if we were using fallback
				if pf.IsUsingFallback() {
					pf.SwitchToPrimary()
				}
				return output, nil
			}
			lastErr = err
			continue
		}

		// Try fallback providers
		output, err := provider.Process(ctx, input, convCtx, opts...)
		if err == nil && output != nil {
			// Success - switch to this fallback
			pf.mu.Lock()
			pf.currentIndex = i
			pf.usingFallback = true
			pf.mu.Unlock()
			return output, nil
		}
		lastErr = err
	}

	// All providers failed
	return nil, fmt.Errorf("all S2S providers failed: %w", lastErr)
}

// CircuitBreaker implements a simple circuit breaker pattern for provider resilience.
type CircuitBreaker struct {
	failureThreshold int
	resetTimeout     time.Duration
	failureCount     int
	lastFailureTime  time.Time
	state            CircuitBreakerState
	mu               sync.RWMutex
}

// CircuitBreakerState represents the state of the circuit breaker.
type CircuitBreakerState int

const (
	// CircuitBreakerStateClosed means the circuit is closed (normal operation).
	CircuitBreakerStateClosed CircuitBreakerState = iota
	// CircuitBreakerStateOpen means the circuit is open (failing, not trying).
	CircuitBreakerStateOpen
	// CircuitBreakerStateHalfOpen means the circuit is half-open (testing recovery).
	CircuitBreakerStateHalfOpen
)

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(failureThreshold int, resetTimeoutMs int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
		failureCount:     0,
		state:            CircuitBreakerStateClosed,
	}
}

// Call executes a function with circuit breaker protection.
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()

	// Check if we should attempt recovery
	if cb.state == CircuitBreakerStateOpen {
		if time.Since(cb.lastFailureTime) >= cb.resetTimeout {
			cb.state = CircuitBreakerStateHalfOpen
			cb.failureCount = 0
		} else {
			cb.mu.Unlock()
			return fmt.Errorf("circuit breaker is open")
		}
	}

	cb.mu.Unlock()

	// Execute the function
	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failureCount++
		cb.lastFailureTime = time.Now()

		if cb.failureCount >= cb.failureThreshold {
			cb.state = CircuitBreakerStateOpen
		} else if cb.state == CircuitBreakerStateHalfOpen {
			cb.state = CircuitBreakerStateOpen
		}
		return err
	}

	// Success - reset failure count and close circuit
	cb.failureCount = 0
	if cb.state == CircuitBreakerStateHalfOpen {
		cb.state = CircuitBreakerStateClosed
	}

	return nil
}

// GetState returns the current circuit breaker state.
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}
