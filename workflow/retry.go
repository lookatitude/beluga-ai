package workflow

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"time"
)

// RetryPolicy configures retry behavior for activities.
type RetryPolicy struct {
	// MaxAttempts is the maximum number of attempts (including the initial try).
	MaxAttempts int
	// InitialInterval is the delay before the first retry.
	InitialInterval time.Duration
	// BackoffCoefficient is the multiplier applied to the interval after each retry.
	BackoffCoefficient float64
	// MaxInterval is the maximum delay between retries.
	MaxInterval time.Duration
}

// DefaultRetryPolicy returns a sensible default retry policy.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts:        3,
		InitialInterval:   100 * time.Millisecond,
		BackoffCoefficient: 2.0,
		MaxInterval:        10 * time.Second,
	}
}

// executeWithRetry executes the given function with retries according to the policy.
func executeWithRetry(ctx context.Context, policy RetryPolicy, fn func(ctx context.Context) error) error {
	if policy.MaxAttempts <= 0 {
		policy.MaxAttempts = 1
	}
	if policy.InitialInterval <= 0 {
		policy.InitialInterval = 100 * time.Millisecond
	}
	if policy.BackoffCoefficient <= 0 {
		policy.BackoffCoefficient = 2.0
	}

	var lastErr error
	interval := policy.InitialInterval

	for attempt := 0; attempt < policy.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		lastErr = fn(ctx)
		if lastErr == nil {
			return nil
		}

		if attempt+1 >= policy.MaxAttempts {
			break
		}

		// Add jitter: 0.5x to 1.5x the interval.
		jitter := time.Duration(float64(interval) * (0.5 + rand.Float64()))
		if policy.MaxInterval > 0 && jitter > policy.MaxInterval {
			jitter = policy.MaxInterval
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(jitter):
		}

		// Exponential backoff.
		interval = time.Duration(float64(interval) * policy.BackoffCoefficient)
		if policy.MaxInterval > 0 && interval > policy.MaxInterval {
			interval = policy.MaxInterval
		}
	}

	return fmt.Errorf("workflow/retry: max attempts (%d) exceeded: %w", policy.MaxAttempts, lastErr)
}

// computeInterval calculates the retry interval for a given attempt.
func computeInterval(policy RetryPolicy, attempt int) time.Duration {
	interval := float64(policy.InitialInterval) * math.Pow(policy.BackoffCoefficient, float64(attempt))
	if policy.MaxInterval > 0 && time.Duration(interval) > policy.MaxInterval {
		return policy.MaxInterval
	}
	return time.Duration(interval)
}
