package internal

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ErrorRecovery manages automatic error recovery and retry logic
type ErrorRecovery struct {
	mu              sync.RWMutex
	maxRetries      int
	retryDelay      time.Duration
	retryCount      map[string]int // Track retries per operation
	lastError       map[string]error
	recoveryEnabled bool
}

// NewErrorRecovery creates a new error recovery manager
func NewErrorRecovery(maxRetries int, retryDelay time.Duration) *ErrorRecovery {
	return &ErrorRecovery{
		maxRetries:      maxRetries,
		retryDelay:      retryDelay,
		retryCount:      make(map[string]int),
		lastError:       make(map[string]error),
		recoveryEnabled: true,
	}
}

// ShouldRetry determines if an error should be retried
func (er *ErrorRecovery) ShouldRetry(op string, err error) bool {
	if !er.recoveryEnabled {
		return false
	}

	// Check if error is retryable
	if !isRetryableError(err) {
		return false
	}

	er.mu.Lock()
	defer er.mu.Unlock()

	count := er.retryCount[op]
	if count >= er.maxRetries {
		return false
	}

	er.retryCount[op] = count + 1
	er.lastError[op] = err
	return true
}

// RetryWithBackoff executes a function with retry logic and exponential backoff
func (er *ErrorRecovery) RetryWithBackoff(ctx context.Context, op string, fn func() error) error {
	var lastErr error
	backoff := er.retryDelay

	for attempt := 0; attempt <= er.maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				backoff = time.Duration(float64(backoff) * 1.5) // Exponential backoff
			}
		}

		err := fn()
		if err == nil {
			// Success - reset retry count
			er.mu.Lock()
			delete(er.retryCount, op)
			delete(er.lastError, op)
			er.mu.Unlock()
			return nil
		}

		lastErr = err
		if !er.ShouldRetry(op, err) {
			break
		}
	}

	return fmt.Errorf("operation %s failed after %d attempts: %w", op, er.maxRetries+1, lastErr)
}

// Reset resets retry counts for an operation
func (er *ErrorRecovery) Reset(op string) {
	er.mu.Lock()
	defer er.mu.Unlock()
	delete(er.retryCount, op)
	delete(er.lastError, op)
}

// isRetryableError checks if an error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for timeout errors
	errStr := err.Error()
	if contains(errStr, "timeout") || contains(errStr, "deadline") {
		return true
	}

	// Check for network errors
	if contains(errStr, "network") || contains(errStr, "connection") {
		return true
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
