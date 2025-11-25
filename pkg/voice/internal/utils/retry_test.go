package utils

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	assert.Equal(t, 3, config.MaxAttempts)
	assert.Equal(t, 100*time.Millisecond, config.InitialDelay)
	assert.Equal(t, 30*time.Second, config.MaxDelay)
	assert.Equal(t, 2.0, config.BackoffFactor)
	assert.Equal(t, 0.1, config.JitterFactor)
}

func TestNewRetryExecutor(t *testing.T) {
	config := DefaultRetryConfig()
	executor := NewRetryExecutor(config)

	assert.NotNil(t, executor)
	assert.Equal(t, config, executor.config)
}

func TestRetryExecutor_ExecuteWithRetry_Success(t *testing.T) {
	config := DefaultRetryConfig()
	executor := NewRetryExecutor(config)
	ctx := context.Background()

	err := executor.ExecuteWithRetry(ctx, func() error {
		return nil
	})

	assert.NoError(t, err)
}

func TestRetryExecutor_ExecuteWithRetry_SuccessAfterRetries(t *testing.T) {
	config := DefaultRetryConfig()
	executor := NewRetryExecutor(config)
	ctx := context.Background()

	attempts := 0
	err := executor.ExecuteWithRetry(ctx, func() error {
		attempts++
		if attempts < 2 {
			return errors.New("temporary error")
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 2, attempts)
}

func TestRetryExecutor_ExecuteWithRetry_AllAttemptsFail(t *testing.T) {
	config := DefaultRetryConfig()
	executor := NewRetryExecutor(config)
	testErr := errors.New("persistent error")
	ctx := context.Background()

	err := executor.ExecuteWithRetry(ctx, func() error {
		return testErr
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "operation failed after")
	assert.Contains(t, err.Error(), "attempts")
}

func TestRetryExecutor_ExecuteWithRetry_ContextCancellation(t *testing.T) {
	config := DefaultRetryConfig()
	executor := NewRetryExecutor(config)
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context immediately
	cancel()

	err := executor.ExecuteWithRetry(ctx, func() error {
		return errors.New("should not be called")
	})

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestRetryExecutor_ExecuteWithRetry_ContextCancellationDuringRetry(t *testing.T) {
	config := DefaultRetryConfig()
	executor := NewRetryExecutor(config)
	ctx, cancel := context.WithCancel(context.Background())

	attempts := 0
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := executor.ExecuteWithRetry(ctx, func() error {
		attempts++
		return errors.New("temporary error")
	})

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	assert.Greater(t, attempts, 0)
}

func TestRetryExecutor_calculateDelay(t *testing.T) {
	config := RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
		JitterFactor:  0.1,
	}
	executor := NewRetryExecutor(config)

	// Test exponential backoff
	delay1 := executor.calculateDelay(1)
	delay2 := executor.calculateDelay(2)
	delay3 := executor.calculateDelay(3)

	// Delay should increase exponentially
	assert.Greater(t, delay2, delay1)
	assert.Greater(t, delay3, delay2)

	// But should be capped at MaxDelay
	assert.LessOrEqual(t, delay1, config.MaxDelay)
	assert.LessOrEqual(t, delay2, config.MaxDelay)
	assert.LessOrEqual(t, delay3, config.MaxDelay)
}

func TestRetryExecutor_calculateDelay_MaxDelayCap(t *testing.T) {
	config := RetryConfig{
		MaxAttempts:   10,
		InitialDelay:  1 * time.Second,
		MaxDelay:      2 * time.Second,
		BackoffFactor: 2.0,
		JitterFactor:  0.1,
	}
	executor := NewRetryExecutor(config)

	// Even with high attempt number, delay should be capped (accounting for jitter)
	delay := executor.calculateDelay(10)
	// Jitter can add up to 10% (JitterFactor), so allow for that
	maxAllowed := config.MaxDelay + time.Duration(float64(config.MaxDelay)*config.JitterFactor)
	assert.LessOrEqual(t, delay, maxAllowed)
}

func TestRetryExecutor_ExecuteWithRetry_RetryCount(t *testing.T) {
	ctx := context.Background()
	config := RetryConfig{
		MaxAttempts:   2,
		InitialDelay:  10 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		BackoffFactor: 2.0,
		JitterFactor:  0.1,
	}
	executor := NewRetryExecutor(config)

	attempts := 0
	err := executor.ExecuteWithRetry(ctx, func() error {
		attempts++
		return errors.New("always fails")
	})

	assert.Error(t, err)
	// Should have tried MaxAttempts + 1 times (initial + retries)
	assert.Equal(t, config.MaxAttempts+1, attempts)
}

func TestRetryExecutor_ExecuteWithRetry_NoDelayOnFirstAttempt(t *testing.T) {
	config := RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
		JitterFactor:  0.1,
	}
	executor := NewRetryExecutor(config)
	ctx := context.Background()

	start := time.Now()
	err := executor.ExecuteWithRetry(ctx, func() error {
		return nil
	})
	duration := time.Since(start)

	assert.NoError(t, err)
	// First attempt should be immediate (no delay)
	assert.Less(t, duration, 50*time.Millisecond)
}
