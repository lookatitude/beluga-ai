package orchestration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRetryExecutor(t *testing.T) {
	config := DefaultRetryConfig()
	executor := NewRetryExecutor(config)
	require.NotNil(t, executor)
	assert.Equal(t, config, executor.config)
}

func TestRetryExecutor_ExecuteWithRetry_Success(t *testing.T) {
	config := RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  10 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		BackoffFactor: 2.0,
	}

	executor := NewRetryExecutor(config)
	callCount := 0

	operation := func() error {
		callCount++
		if callCount < 2 {
			return errors.New("temporary error")
		}
		return nil
	}

	ctx := context.Background()
	err := executor.ExecuteWithRetry(ctx, operation)

	assert.NoError(t, err)
	assert.Equal(t, 2, callCount)
}

func TestRetryExecutor_ExecuteWithRetry_ExhaustRetries(t *testing.T) {
	config := RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  10 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		BackoffFactor: 2.0,
	}

	executor := NewRetryExecutor(config)
	callCount := 0

	operation := func() error {
		callCount++
		return errors.New("persistent error")
	}

	ctx := context.Background()
	err := executor.ExecuteWithRetry(ctx, operation)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed after 3 attempts")
	assert.Equal(t, 3, callCount)
}

func TestRetryExecutor_ExecuteWithRetry_ContextCancelled(t *testing.T) {
	config := RetryConfig{
		MaxAttempts:   5,
		InitialDelay:  50 * time.Millisecond,
		MaxDelay:      200 * time.Millisecond,
		BackoffFactor: 2.0,
	}

	executor := NewRetryExecutor(config)
	callCount := 0

	operation := func() error {
		callCount++
		return errors.New("error")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Millisecond)
	defer cancel()

	err := executor.ExecuteWithRetry(ctx, operation)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled")
	assert.True(t, callCount >= 1)
}

func TestRetryExecutor_CalculateDelay(t *testing.T) {
	config := RetryConfig{
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      1000 * time.Millisecond,
		BackoffFactor: 2.0,
		JitterFactor:  0.0, // Disable jitter for predictable testing
	}

	executor := NewRetryExecutor(config)

	// Test first retry (attempt 1)
	delay1 := executor.calculateDelay(1)
	assert.Equal(t, 100*time.Millisecond, delay1) // 100 * 2^(1-1) = 100 * 1

	// Test second retry (attempt 2)
	delay2 := executor.calculateDelay(2)
	assert.Equal(t, 200*time.Millisecond, delay2) // 100 * 2^(2-1) = 100 * 2

	// Test third retry (attempt 3)
	delay3 := executor.calculateDelay(3)
	assert.Equal(t, 400*time.Millisecond, delay3) // 100 * 2^(3-1) = 100 * 4

	// Test max delay cap (attempt 10)
	delay10 := executor.calculateDelay(10)
	assert.Equal(t, 1000*time.Millisecond, delay10) // capped at MaxDelay
}

func TestRetryExecutor_IsRetryableError(t *testing.T) {
	config := RetryConfig{
		RetryableErrors: []error{errors.New("network_error"), errors.New("timeout")},
	}

	executor := NewRetryExecutor(config)

	// Test retryable errors
	assert.True(t, executor.isRetryableError(errors.New("network_error")))
	assert.True(t, executor.isRetryableError(errors.New("timeout")))

	// Test non-retryable error
	assert.False(t, executor.isRetryableError(errors.New("validation_error")))

	// Test empty retryable errors (should retry all)
	configEmpty := RetryConfig{}
	executorEmpty := NewRetryExecutor(configEmpty)
	assert.True(t, executorEmpty.isRetryableError(errors.New("any_error")))
}

func TestCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(2, 100*time.Millisecond)

	// Test initial state
	assert.Equal(t, CircuitState(StateClosed), cb.GetState())

	// Test successful calls
	err := cb.Call(func() error { return nil })
	assert.NoError(t, err)
	assert.Equal(t, CircuitState(StateClosed), cb.GetState())

	// Test failures leading to open state
	err = cb.Call(func() error { return errors.New("failure") })
	assert.Error(t, err)
	assert.Equal(t, CircuitState(StateClosed), cb.GetState())

	err = cb.Call(func() error { return errors.New("failure") })
	assert.Error(t, err)
	assert.Equal(t, CircuitState(StateOpen), cb.GetState())

	// Test circuit breaker open
	err = cb.Call(func() error { return nil })
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker is open")

	// Wait for reset timeout
	time.Sleep(110 * time.Millisecond)

	// Test half-open state (should allow one request)
	err = cb.Call(func() error { return nil })
	assert.NoError(t, err)
	assert.Equal(t, CircuitState(StateClosed), cb.GetState())
}

func TestBulkhead(t *testing.T) {
	bulkhead := NewBulkhead(2)
	ctx := context.Background()

	// Fill the bulkhead capacity
	done := make(chan bool, 2)
	for i := 0; i < 2; i++ {
		go func() {
			bulkhead.Execute(ctx, func() error {
				<-done // Wait to be released
				return nil
			})
		}()
	}

	// Give goroutines time to acquire capacity
	time.Sleep(10 * time.Millisecond)

	// Test capacity exceeded - should fail immediately
	err := bulkhead.Execute(ctx, func() error { return nil })
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bulkhead capacity exceeded")

	// Release capacity
	done <- true
	done <- true

	// Now should be able to execute again
	time.Sleep(10 * time.Millisecond)
	err = bulkhead.Execute(ctx, func() error { return nil })
	assert.NoError(t, err)
}

func TestBulkhead_SerialExecution(t *testing.T) {
	bulkhead := NewBulkhead(1)
	ctx := context.Background()

	// Test serial execution
	for i := 0; i < 3; i++ {
		err := bulkhead.Execute(ctx, func() error {
			time.Sleep(10 * time.Millisecond)
			return nil
		})
		assert.NoError(t, err)
	}
}

func TestBulkhead_GetStats(t *testing.T) {
	bulkhead := NewBulkhead(5)

	assert.Equal(t, 5, bulkhead.GetCapacity())
	assert.Equal(t, 0, bulkhead.GetCurrentConcurrency())

	ctx := context.Background()
	done := make(chan bool)

	go func() {
		bulkhead.Execute(ctx, func() error {
			time.Sleep(50 * time.Millisecond)
			done <- true
			return nil
		})
	}()

	// Give some time for goroutine to start
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 1, bulkhead.GetCurrentConcurrency())

	<-done
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 0, bulkhead.GetCurrentConcurrency())
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	assert.Equal(t, 3, config.MaxAttempts)
	assert.Equal(t, 100*time.Millisecond, config.InitialDelay)
	assert.Equal(t, 30*time.Second, config.MaxDelay)
	assert.Equal(t, 2.0, config.BackoffFactor)
	assert.Equal(t, 0.1, config.JitterFactor)
	assert.Nil(t, config.RetryableErrors)
}
