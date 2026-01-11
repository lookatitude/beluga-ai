package internal

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
)

// RetryConfig holds retry configuration.
type RetryConfig struct {
	MaxRetries int
	Delay      time.Duration
	Backoff    float64
}

// DefaultRetryConfig returns default retry configuration.
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries: 3,
		Delay:      time.Second,
		Backoff:    2.0,
	}
}

// RetryWithBackoff executes a function with exponential backoff retry logic.
func RetryWithBackoff(ctx context.Context, config *RetryConfig, operation string, fn func() error) error {
	var lastErr error
	delay := config.Delay

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				delay = time.Duration(float64(delay) * config.Backoff)
			}
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't retry if the error is not retryable
		if !backend.IsRetryableError(err) {
			break
		}
	}

	return backend.WrapError(operation, lastErr)
}
