package resilience

import (
	"context"
	"sync"
	"time"
)

// ProviderLimits describes the rate-limiting constraints for a specific LLM
// provider or model.
type ProviderLimits struct {
	// RPM is the maximum number of requests per minute. Zero means unlimited.
	RPM int

	// TPM is the maximum number of tokens per minute. Zero means unlimited.
	TPM int

	// MaxConcurrent is the maximum number of in-flight requests. Zero means
	// unlimited.
	MaxConcurrent int

	// CooldownOnRetry is the additional delay to apply when a request must be
	// retried due to rate limiting.
	CooldownOnRetry time.Duration
}

// RateLimiter enforces provider-specific rate limits using a token-bucket
// algorithm for RPM and TPM, and a semaphore for concurrency.
type RateLimiter struct {
	limits ProviderLimits

	mu            sync.Mutex
	// RPM token bucket state.
	rpmTokens     float64
	rpmLastRefill time.Time

	// TPM token bucket state.
	tpmTokens     float64
	tpmLastRefill time.Time

	// Concurrency tracking.
	concurrent int
}

// NewRateLimiter creates a RateLimiter enforcing the given limits.
func NewRateLimiter(limits ProviderLimits) *RateLimiter {
	now := time.Now()
	rl := &RateLimiter{
		limits:        limits,
		rpmLastRefill: now,
		tpmLastRefill: now,
	}
	if limits.RPM > 0 {
		rl.rpmTokens = float64(limits.RPM)
	}
	if limits.TPM > 0 {
		rl.tpmTokens = float64(limits.TPM)
	}
	return rl
}

// Allow blocks until the rate limiter permits a new request, or the context
// is cancelled. It reserves 1 RPM token and checks the concurrency limit.
func (rl *RateLimiter) Allow(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		rl.mu.Lock()
		ok, wait := rl.tryAcquire()
		rl.mu.Unlock()

		if ok {
			return nil
		}

		// Wait until tokens refill or context expires.
		if wait <= 0 {
			wait = 10 * time.Millisecond
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
}

// Release signals that an in-flight request has completed, freeing a
// concurrency slot.
func (rl *RateLimiter) Release() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if rl.concurrent > 0 {
		rl.concurrent--
	}
}

// Wait blocks for the cooldown duration configured for retry scenarios, or
// until ctx is cancelled.
func (rl *RateLimiter) Wait(ctx context.Context) error {
	if rl.limits.CooldownOnRetry <= 0 {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(rl.limits.CooldownOnRetry):
		return nil
	}
}

// ConsumeTokens deducts count tokens from the TPM bucket. Call this after
// learning how many tokens a request consumed. It blocks until enough tokens
// are available or the context is cancelled.
func (rl *RateLimiter) ConsumeTokens(ctx context.Context, count int) error {
	if rl.limits.TPM <= 0 || count <= 0 {
		return nil
	}

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		rl.mu.Lock()
		rl.refillTPM()
		if rl.tpmTokens >= float64(count) {
			rl.tpmTokens -= float64(count)
			rl.mu.Unlock()
			return nil
		}
		deficit := float64(count) - rl.tpmTokens
		rate := float64(rl.limits.TPM) / 60.0 // tokens per second
		wait := time.Duration(deficit/rate*1e9) * time.Nanosecond
		rl.mu.Unlock()

		if wait <= 0 {
			wait = 10 * time.Millisecond
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
}

// tryAcquire attempts to take 1 RPM token and 1 concurrency slot. Returns
// (true, 0) on success, or (false, suggestedWait) when the caller should
// back off. Caller must hold rl.mu.
func (rl *RateLimiter) tryAcquire() (ok bool, wait time.Duration) {
	// Check concurrency.
	if rl.limits.MaxConcurrent > 0 && rl.concurrent >= rl.limits.MaxConcurrent {
		return false, 10 * time.Millisecond
	}

	// Check RPM.
	if rl.limits.RPM > 0 {
		rl.refillRPM()
		if rl.rpmTokens < 1.0 {
			// Compute how long until one token is available.
			rate := float64(rl.limits.RPM) / 60.0 // tokens per second
			deficit := 1.0 - rl.rpmTokens
			return false, time.Duration(deficit/rate*1e9) * time.Nanosecond
		}
		rl.rpmTokens--
	}

	rl.concurrent++
	return true, 0
}

// refillRPM replenishes RPM tokens based on elapsed time. Caller must hold
// rl.mu.
func (rl *RateLimiter) refillRPM() {
	now := time.Now()
	elapsed := now.Sub(rl.rpmLastRefill).Seconds()
	if elapsed <= 0 {
		return
	}
	rate := float64(rl.limits.RPM) / 60.0
	rl.rpmTokens += elapsed * rate
	if rl.rpmTokens > float64(rl.limits.RPM) {
		rl.rpmTokens = float64(rl.limits.RPM)
	}
	rl.rpmLastRefill = now
}

// refillTPM replenishes TPM tokens based on elapsed time. Caller must hold
// rl.mu.
func (rl *RateLimiter) refillTPM() {
	now := time.Now()
	elapsed := now.Sub(rl.tpmLastRefill).Seconds()
	if elapsed <= 0 {
		return
	}
	rate := float64(rl.limits.TPM) / 60.0
	rl.tpmTokens += elapsed * rate
	if rl.tpmTokens > float64(rl.limits.TPM) {
		rl.tpmTokens = float64(rl.limits.TPM)
	}
	rl.tpmLastRefill = now
}
