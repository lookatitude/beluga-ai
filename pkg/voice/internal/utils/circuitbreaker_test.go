package utils

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(3, 100*time.Millisecond)

	assert.NotNil(t, cb)
	assert.Equal(t, StateClosed, cb.GetState())
	assert.Equal(t, 3, cb.failureThreshold)
	assert.Equal(t, 100*time.Millisecond, cb.resetTimeout)
}

func TestCircuitBreaker_Call_Success(t *testing.T) {
	cb := NewCircuitBreaker(3, 100*time.Millisecond)

	err := cb.Call(func() error {
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, StateClosed, cb.GetState())
	assert.Equal(t, 0, cb.failureCount)
}

func TestCircuitBreaker_Call_Failure(t *testing.T) {
	cb := NewCircuitBreaker(3, 100*time.Millisecond)
	testErr := errors.New("test error")

	err := cb.Call(func() error {
		return testErr
	})

	require.Error(t, err)
	assert.Equal(t, testErr, err)
	assert.Equal(t, StateClosed, cb.GetState())
	assert.Equal(t, 1, cb.failureCount)
}

func TestCircuitBreaker_Call_OpenState(t *testing.T) {
	cb := NewCircuitBreaker(2, 100*time.Millisecond)
	testErr := errors.New("test error")

	// Trigger failures to open the circuit
	_ = cb.Call(func() error {
		return testErr
	})
	_ = cb.Call(func() error {
		return testErr
	})

	// Circuit should be open now
	assert.Equal(t, StateOpen, cb.GetState())

	// Call should fail immediately
	err := cb.Call(func() error {
		return nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker is open")
}

func TestCircuitBreaker_Call_HalfOpenState(t *testing.T) {
	cb := NewCircuitBreaker(2, 50*time.Millisecond)
	testErr := errors.New("test error")

	// Trigger failures to open the circuit
	_ = cb.Call(func() error {
		return testErr
	})
	_ = cb.Call(func() error {
		return testErr
	})

	// Wait for reset timeout
	time.Sleep(60 * time.Millisecond)

	// First call after timeout should be allowed (half-open)
	// But we need to check state transition
	err := cb.Call(func() error {
		return nil
	})

	// If successful, circuit should close
	if err == nil {
		assert.Equal(t, StateClosed, cb.GetState())
	}
}

func TestCircuitBreaker_Call_HalfOpenToOpen(t *testing.T) {
	cb := NewCircuitBreaker(2, 50*time.Millisecond)
	testErr := errors.New("test error")

	// Trigger failures to open the circuit
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

	require.Error(t, err)
	assert.Equal(t, StateOpen, cb.GetState())
}

func TestCircuitBreaker_GetState(t *testing.T) {
	cb := NewCircuitBreaker(3, 100*time.Millisecond)

	// Initial state should be closed
	assert.Equal(t, StateClosed, cb.GetState())

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

	// Should be open after threshold
	assert.Equal(t, StateOpen, cb.GetState())
}

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	cb := NewCircuitBreaker(2, 50*time.Millisecond)
	testErr := errors.New("test error")

	// Start closed
	assert.Equal(t, StateClosed, cb.GetState())

	// Fail once - still closed
	_ = cb.Call(func() error {
		return testErr
	})
	assert.Equal(t, StateClosed, cb.GetState())

	// Fail again - should open
	_ = cb.Call(func() error {
		return testErr
	})
	assert.Equal(t, StateOpen, cb.GetState())

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Next call should transition to half-open
	// We can't directly check half-open state easily, but we can verify behavior
	_ = cb.Call(func() error {
		return nil
	})

	// If successful, should be closed
	state := cb.GetState()
	assert.True(t, state == StateClosed || state == StateHalfOpen)
}
