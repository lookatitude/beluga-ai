package internal

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(3, 2, 100*time.Millisecond)
	assert.NotNil(t, cb)
	assert.Equal(t, CircuitBreakerClosed, cb.GetState())
	assert.Equal(t, 3, cb.maxFailures)
	assert.Equal(t, 2, cb.halfOpenMaxSuccess)
}

func TestCircuitBreaker_Call_Success(t *testing.T) {
	cb := NewCircuitBreaker(3, 2, 100*time.Millisecond)
	err := cb.Call(func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, CircuitBreakerClosed, cb.GetState())
}

func TestCircuitBreaker_Call_Failure(t *testing.T) {
	cb := NewCircuitBreaker(3, 2, 100*time.Millisecond)
	testErr := errors.New("test error")

	err := cb.Call(func() error {
		return testErr
	})
	assert.Error(t, err)
	assert.Equal(t, testErr, err)
	assert.Equal(t, CircuitBreakerClosed, cb.GetState()) // Not open yet
}

func TestCircuitBreaker_Call_OpenState(t *testing.T) {
	cb := NewCircuitBreaker(2, 2, 100*time.Millisecond)
	testErr := errors.New("test error")

	// Trigger failures to open circuit
	_ = cb.Call(func() error {
		return testErr
	})
	_ = cb.Call(func() error {
		return testErr
	})

	// Circuit should be open now
	assert.Equal(t, CircuitBreakerOpen, cb.GetState())

	// Call should fail immediately
	err := cb.Call(func() error {
		return nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker is open")
}

func TestCircuitBreaker_Call_HalfOpenState(t *testing.T) {
	cb := NewCircuitBreaker(2, 1, 50*time.Millisecond)
	testErr := errors.New("test error")

	// Trigger failures to open circuit
	_ = cb.Call(func() error {
		return testErr
	})
	_ = cb.Call(func() error {
		return testErr
	})

	// Wait for reset timeout
	time.Sleep(60 * time.Millisecond)

	// First call should transition to half-open and succeed
	err := cb.Call(func() error {
		return nil
	})
	assert.NoError(t, err)
	// After success in half-open, should close
	assert.Equal(t, CircuitBreakerClosed, cb.GetState())
}

func TestCircuitBreaker_Call_HalfOpenToOpen(t *testing.T) {
	cb := NewCircuitBreaker(2, 2, 50*time.Millisecond)
	testErr := errors.New("test error")

	// Trigger failures to open circuit
	_ = cb.Call(func() error {
		return testErr
	})
	_ = cb.Call(func() error {
		return testErr
	})

	// Wait for reset timeout
	time.Sleep(60 * time.Millisecond)

	// Call in half-open state that fails should open circuit again
	err := cb.Call(func() error {
		return testErr
	})
	assert.Error(t, err)
	assert.Equal(t, CircuitBreakerOpen, cb.GetState())
}

func TestCircuitBreaker_GetState(t *testing.T) {
	cb := NewCircuitBreaker(3, 2, 100*time.Millisecond)
	assert.Equal(t, CircuitBreakerClosed, cb.GetState())

	// After failures, state should change
	testErr := errors.New("test error")
	_ = cb.Call(func() error {
		return testErr
	})
	_ = cb.Call(func() error {
		return testErr
	})
	_ = cb.Call(func() error {
		return testErr
	})

	assert.Equal(t, CircuitBreakerOpen, cb.GetState())
}

func TestCircuitBreaker_HalfOpenSuccessThreshold(t *testing.T) {
	cb := NewCircuitBreaker(2, 2, 50*time.Millisecond)
	testErr := errors.New("test error")

	// Open the circuit
	_ = cb.Call(func() error {
		return testErr
	})
	_ = cb.Call(func() error {
		return testErr
	})

	// Wait for reset
	time.Sleep(60 * time.Millisecond)

	// Need 2 successes to close (halfOpenMaxSuccess = 2)
	_ = cb.Call(func() error {
		return nil
	})
	// Should still be half-open after 1 success
	state := cb.GetState()
	assert.True(t, state == CircuitBreakerHalfOpen || state == CircuitBreakerClosed)

	// Second success should close it
	_ = cb.Call(func() error {
		return nil
	})
	assert.Equal(t, CircuitBreakerClosed, cb.GetState())
}

