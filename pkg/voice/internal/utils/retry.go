// Package utils provides shared utilities for the voice package.
package utils

import (
	"context"
	"fmt"
	"math"
	"time"
)

// RetryConfig holds configuration for retry behavior.
type RetryConfig struct {
	MaxAttempts   int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
	JitterFactor  float64
}

// DefaultRetryConfig returns a default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		JitterFactor:  0.1,
	}
}

// RetryExecutor handles retry logic with exponential backoff.
type RetryExecutor struct {
	config RetryConfig
}

// NewRetryExecutor creates a new retry executor.
func NewRetryExecutor(config RetryConfig) *RetryExecutor {
	return &RetryExecutor{config: config}
}

// ExecuteWithRetry executes a function with retry logic.
func (re *RetryExecutor) ExecuteWithRetry(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt <= re.config.MaxAttempts; attempt++ {
		if attempt > 0 {
			delay := re.calculateDelay(attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err
	}

	return fmt.Errorf("operation failed after %d attempts: %w", re.config.MaxAttempts+1, lastErr)
}

// calculateDelay calculates the delay for the given attempt using exponential backoff.
func (re *RetryExecutor) calculateDelay(attempt int) time.Duration {
	// Exponential backoff: initialDelay * (backoffFactor ^ (attempt - 1))
	delay := float64(re.config.InitialDelay) * math.Pow(re.config.BackoffFactor, float64(attempt-1))

	// Cap at max delay
	if delay > float64(re.config.MaxDelay) {
		delay = float64(re.config.MaxDelay)
	}

	// Add jitter to prevent thundering herd
	jitter := delay * re.config.JitterFactor * float64(time.Now().UnixNano()%1000) / 1000.0

	return time.Duration(delay + jitter)
}
