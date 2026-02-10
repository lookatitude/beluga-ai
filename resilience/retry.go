package resilience

import (
	"context"
	"errors"
	"math"
	"math/rand/v2"
	"time"

	"github.com/lookatitude/beluga-ai/core"
)

// RetryPolicy configures how retries are performed. Zero-value fields fall back
// to the defaults returned by DefaultRetryPolicy.
type RetryPolicy struct {
	// MaxAttempts is the total number of attempts (including the first call).
	// A value of 1 means no retries.
	MaxAttempts int

	// InitialBackoff is the delay before the first retry.
	InitialBackoff time.Duration

	// MaxBackoff caps the delay between retries.
	MaxBackoff time.Duration

	// BackoffFactor is the multiplier applied to the backoff after each retry.
	BackoffFactor float64

	// Jitter adds random ±25 % variation to the computed backoff when true.
	Jitter bool

	// RetryableErrors restricts retries to these error codes. When empty,
	// core.IsRetryable decides whether an error is retryable.
	RetryableErrors []core.ErrorCode
}

// DefaultRetryPolicy returns a sensible default retry policy: 3 attempts,
// 500 ms initial backoff, 30 s max backoff, 2× multiplier, jitter enabled.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts:    3,
		InitialBackoff: 500 * time.Millisecond,
		MaxBackoff:     30 * time.Second,
		BackoffFactor:  2.0,
		Jitter:         true,
	}
}

// Retry executes fn up to policy.MaxAttempts times. On each retryable failure
// it waits with exponential backoff (optionally jittered) before retrying. If
// the context is cancelled the function returns immediately with the context
// error.
func Retry[T any](ctx context.Context, policy RetryPolicy, fn func(ctx context.Context) (T, error)) (T, error) {
	policy = normalizePolicy(policy)

	retryableSet := buildRetryableSet(policy.RetryableErrors)

	var lastErr error
	backoff := policy.InitialBackoff

	for attempt := 0; attempt < policy.MaxAttempts; attempt++ {
		result, err := fn(ctx)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Don't wait after the last attempt.
		if attempt == policy.MaxAttempts-1 {
			break
		}

		if !isRetryable(err, retryableSet) {
			break
		}

		// Compute jittered delay.
		delay := backoff
		if policy.Jitter {
			delay = jitter(delay)
		}

		select {
		case <-ctx.Done():
			var zero T
			return zero, ctx.Err()
		case <-time.After(delay):
		}

		// Grow backoff for next iteration, capped at MaxBackoff.
		backoff = time.Duration(float64(backoff) * policy.BackoffFactor)
		if backoff > policy.MaxBackoff {
			backoff = policy.MaxBackoff
		}
	}

	var zero T
	return zero, lastErr
}

// normalizePolicy fills zero-value fields with their defaults.
func normalizePolicy(p RetryPolicy) RetryPolicy {
	d := DefaultRetryPolicy()
	if p.MaxAttempts <= 0 {
		p.MaxAttempts = d.MaxAttempts
	}
	if p.InitialBackoff <= 0 {
		p.InitialBackoff = d.InitialBackoff
	}
	if p.MaxBackoff <= 0 {
		p.MaxBackoff = d.MaxBackoff
	}
	if p.BackoffFactor <= 0 {
		p.BackoffFactor = d.BackoffFactor
	}
	return p
}

// buildRetryableSet converts the slice of error codes to a fast-lookup map.
// Returns nil when the slice is empty (meaning "use core.IsRetryable").
func buildRetryableSet(codes []core.ErrorCode) map[core.ErrorCode]bool {
	if len(codes) == 0 {
		return nil
	}
	m := make(map[core.ErrorCode]bool, len(codes))
	for _, c := range codes {
		m[c] = true
	}
	return m
}

// isRetryable decides whether err should be retried. When retryableSet is
// non-nil those specific codes plus the default retryable codes qualify;
// otherwise core.IsRetryable alone is used.
func isRetryable(err error, retryableSet map[core.ErrorCode]bool) bool {
	if retryableSet == nil {
		return core.IsRetryable(err)
	}
	// Always honour the built-in retryable codes.
	if core.IsRetryable(err) {
		return true
	}
	// Check the caller-specified set.
	var e *core.Error
	if errors.As(err, &e) {
		return retryableSet[e.Code]
	}
	return false
}

// jitter applies random ±25 % variation to d.
func jitter(d time.Duration) time.Duration {
	if d <= 0 {
		return d
	}
	// factor in [0.75, 1.25)
	factor := 0.75 + rand.Float64()*0.5
	ns := float64(d.Nanoseconds()) * factor
	// Guard against overflow.
	if ns > math.MaxInt64 {
		return d
	}
	return time.Duration(int64(ns))
}
