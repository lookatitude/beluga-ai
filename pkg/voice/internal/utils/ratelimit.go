package utils

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RateLimiter provides rate limiting functionality.
type RateLimiter struct {
	mu          sync.Mutex
	rate        int           // Number of requests allowed
	per         time.Duration // Per time period
	requests    []time.Time   // Timestamps of recent requests
	windowStart time.Time
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(rate int, per time.Duration) *RateLimiter {
	return &RateLimiter{
		rate:        rate,
		per:         per,
		requests:    make([]time.Time, 0, rate),
		windowStart: time.Now(),
	}
}

// Allow checks if a request is allowed under the rate limit.
func (rl *RateLimiter) Allow(ctx context.Context) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Remove requests outside the time window
	cutoff := now.Add(-rl.per)
	validRequests := rl.requests[:0]
	for _, req := range rl.requests {
		if req.After(cutoff) {
			validRequests = append(validRequests, req)
		}
	}
	rl.requests = validRequests

	// Check if we're at the rate limit
	if len(rl.requests) >= rl.rate {
		// Calculate wait time until the oldest request expires
		oldest := rl.requests[0]
		waitTime := oldest.Add(rl.per).Sub(now)
		if waitTime > 0 {
			return fmt.Errorf("rate limit exceeded, wait %v", waitTime)
		}
	}

	// Add current request
	rl.requests = append(rl.requests, now)
	return nil
}

// Wait waits until a request can be made under the rate limit.
func (rl *RateLimiter) Wait(ctx context.Context) error {
	for {
		err := rl.Allow(ctx)
		if err == nil {
			return nil
		}

		// Extract wait time from error message (simplified - in production, use a better approach)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Millisecond):
			// Retry after a short delay
		}
	}
}
