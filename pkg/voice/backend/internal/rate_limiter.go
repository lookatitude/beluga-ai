package internal

import (
	"context"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

// FrameworkRateLimiter provides a framework fallback rate limiter implementation.
// This is used when providers don't implement their own rate limiting (FR-024).
type FrameworkRateLimiter struct {
	// Rate limit configuration
	requestsPerSecond int
	burstSize         int

	// Token bucket state
	tokens     float64
	lastUpdate time.Time
	mu         sync.Mutex
}

// NewFrameworkRateLimiter creates a new framework rate limiter.
// Defaults: 100 requests/second, burst size of 10.
func NewFrameworkRateLimiter(requestsPerSecond int, burstSize int) *FrameworkRateLimiter {
	if requestsPerSecond <= 0 {
		requestsPerSecond = 100 // Default: 100 requests/second
	}
	if burstSize <= 0 {
		burstSize = 10 // Default: burst size of 10
	}

	return &FrameworkRateLimiter{
		requestsPerSecond: requestsPerSecond,
		burstSize:         burstSize,
		tokens:            float64(burstSize),
		lastUpdate:        time.Now(),
	}
}

// Allow checks if a request is allowed based on a key (token bucket algorithm).
func (rl *FrameworkRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Update tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(rl.lastUpdate).Seconds()
	rl.tokens += elapsed * float64(rl.requestsPerSecond)

	// Cap tokens at burst size
	if rl.tokens > float64(rl.burstSize) {
		rl.tokens = float64(rl.burstSize)
	}

	rl.lastUpdate = now

	// Check if we have enough tokens
	if rl.tokens >= 1.0 {
		rl.tokens -= 1.0
		return true, nil
	}

	return false, nil
}

// Wait waits until a request is allowed based on a key.
func (rl *FrameworkRateLimiter) Wait(ctx context.Context, key string) error {
	// Try to allow immediately
	allowed, err := rl.Allow(ctx, key)
	if err != nil {
		return err
	}

	if allowed {
		return nil
	}

	// Calculate wait time based on token refill rate
	rl.mu.Lock()
	tokensNeeded := 1.0 - rl.tokens
	refillRate := float64(rl.requestsPerSecond)
	waitTime := time.Duration(tokensNeeded/refillRate*float64(time.Second)) + 10*time.Millisecond
	rl.mu.Unlock()

	// Wait with context cancellation support
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitTime):
		// Try again after waiting
		allowed, err := rl.Allow(ctx, key)
		if err != nil {
			return err
		}
		if !allowed {
			// Still not allowed, return error
			return backend.NewBackendError("Wait", backend.ErrCodeRateLimitExceeded,
				backend.NewBackendError("Wait", "rate_limit_exceeded", nil))
		}
		return nil
	}
}

// GetOrCreateRateLimiter returns a rate limiter from config or creates a framework fallback.
func GetOrCreateRateLimiter(config *vbiface.Config) vbiface.RateLimiter {
	if config.RateLimiter != nil {
		return config.RateLimiter
	}

	// Create framework fallback rate limiter
	// Default: 100 requests/second, burst size of 10
	return NewFrameworkRateLimiter(100, 10)
}
