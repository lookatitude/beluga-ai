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
	normalizeRetryPolicy(&policy)

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

		if err := sleepWithJitter(ctx, interval, policy.MaxInterval); err != nil {
			return err
		}

		interval = nextInterval(interval, policy.BackoffCoefficient, policy.MaxInterval)
	}

	return fmt.Errorf("workflow/retry: max attempts (%d) exceeded: %w", policy.MaxAttempts, lastErr)
}

// normalizeRetryPolicy fills in zero-valued fields with sensible defaults.
func normalizeRetryPolicy(p *RetryPolicy) {
	if p.MaxAttempts <= 0 {
		p.MaxAttempts = 1
	}
	if p.InitialInterval <= 0 {
		p.InitialInterval = 100 * time.Millisecond
	}
	if p.BackoffCoefficient <= 0 {
		p.BackoffCoefficient = 2.0
	}
}

// sleepWithJitter waits for interval with jitter applied, respecting ctx and maxInterval.
func sleepWithJitter(ctx context.Context, interval, maxInterval time.Duration) error {
	jitter := time.Duration(float64(interval) * (0.5 + rand.Float64()))
	if maxInterval > 0 && jitter > maxInterval {
		jitter = maxInterval
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(jitter):
		return nil
	}
}

// nextInterval computes the next backoff interval, capped by maxInterval.
func nextInterval(current time.Duration, coefficient float64, maxInterval time.Duration) time.Duration {
	next := time.Duration(float64(current) * coefficient)
	if maxInterval > 0 && next > maxInterval {
		return maxInterval
	}
	return next
}

// computeInterval calculates the retry interval for a given attempt.
func computeInterval(policy RetryPolicy, attempt int) time.Duration {
	interval := float64(policy.InitialInterval) * math.Pow(policy.BackoffCoefficient, float64(attempt))
	if policy.MaxInterval > 0 && time.Duration(interval) > policy.MaxInterval {
		return policy.MaxInterval
	}
	return time.Duration(interval)
}
