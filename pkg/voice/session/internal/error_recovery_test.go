package internal

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewErrorRecovery(t *testing.T) {
	er := NewErrorRecovery(3, 100*time.Millisecond)
	assert.NotNil(t, er)
	assert.Equal(t, 3, er.maxRetries)
	assert.Equal(t, 100*time.Millisecond, er.retryDelay)
	assert.True(t, er.recoveryEnabled)
}

func TestErrorRecovery_ShouldRetry_RetryableError(t *testing.T) {
	er := NewErrorRecovery(3, 100*time.Millisecond)
	err := errors.New("timeout error")

	result := er.ShouldRetry("test-op", err)
	assert.True(t, result)
}

func TestErrorRecovery_ShouldRetry_NonRetryableError(t *testing.T) {
	er := NewErrorRecovery(3, 100*time.Millisecond)
	err := errors.New("invalid input")

	result := er.ShouldRetry("test-op", err)
	assert.False(t, result)
}

func TestErrorRecovery_ShouldRetry_MaxRetriesExceeded(t *testing.T) {
	er := NewErrorRecovery(2, 100*time.Millisecond)
	err := errors.New("timeout error")

	// Retry twice
	assert.True(t, er.ShouldRetry("test-op", err))
	assert.True(t, er.ShouldRetry("test-op", err))

	// Should not retry after max retries
	result := er.ShouldRetry("test-op", err)
	assert.False(t, result)
}

func TestErrorRecovery_RetryWithBackoff_Success(t *testing.T) {
	ctx := context.Background()
	er := NewErrorRecovery(3, 10*time.Millisecond)

	attempts := 0
	err := er.RetryWithBackoff(ctx, "test-op", func() error {
		attempts++
		if attempts < 2 {
			return errors.New("timeout error") // Must be retryable
		}
		return nil
	})

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, attempts, 2)
}

func TestErrorRecovery_RetryWithBackoff_AllFailures(t *testing.T) {
	ctx := context.Background()
	er := NewErrorRecovery(2, 10*time.Millisecond)

	err := er.RetryWithBackoff(ctx, "test-op", func() error {
		return errors.New("timeout error")
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed after")
}

func TestErrorRecovery_RetryWithBackoff_ContextCancellation(t *testing.T) {
	er := NewErrorRecovery(3, 10*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay to allow first attempt
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()

	err := er.RetryWithBackoff(ctx, "test-op", func() error {
		return errors.New("timeout error")
	})

	assert.Error(t, err)
	// Error should be context.Canceled or wrapped
	assert.True(t, err == context.Canceled || err == ctx.Err())
}

func TestErrorRecovery_Reset(t *testing.T) {
	er := NewErrorRecovery(3, 100*time.Millisecond)
	err := errors.New("timeout error")

	// Retry once
	er.ShouldRetry("test-op", err)

	// Reset
	er.Reset("test-op")

	// Should be able to retry again
	result := er.ShouldRetry("test-op", err)
	assert.True(t, result)
}

func TestErrorRecovery_isRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "timeout error",
			err:      errors.New("timeout occurred"),
			expected: true,
		},
		{
			name:     "deadline error",
			err:      errors.New("deadline exceeded"),
			expected: true,
		},
		{
			name:     "network error",
			err:      errors.New("network failure"),
			expected: true,
		},
		{
			name:     "connection error",
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "non-retryable error",
			err:      errors.New("invalid input"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			er := NewErrorRecovery(3, 100*time.Millisecond)
			result := er.ShouldRetry("test-op", tt.err)
			// ShouldRetry checks isRetryableError internally
			// For non-retryable errors, result should be false
			if !tt.expected {
				assert.False(t, result)
			}
		})
	}
}
