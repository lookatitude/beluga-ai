package utils

import (
	"fmt"
	"sync"
	"time"
)

// CircuitBreaker provides circuit breaker functionality.
type CircuitBreaker struct {
	mu               sync.RWMutex
	failureThreshold int
	resetTimeout     time.Duration
	failureCount     int
	lastFailureTime  time.Time
	state            CircuitState
}

// CircuitState represents the state of the circuit breaker.
type CircuitState int

const (
	// StateClosed indicates the circuit is closed and requests are allowed.
	StateClosed CircuitState = iota
	// StateOpen indicates the circuit is open and requests are blocked.
	StateOpen
	// StateHalfOpen indicates the circuit is half-open and testing recovery.
	StateHalfOpen
)

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
		state:            StateClosed,
	}
}

// Call executes a function with circuit breaker protection.
func (cb *CircuitBreaker) Call(operation func() error) error {
	cb.mu.Lock()

	switch cb.state {
	case StateOpen:
		if time.Since(cb.lastFailureTime) < cb.resetTimeout {
			cb.mu.Unlock()
			return fmt.Errorf("circuit breaker is open")
		}
		cb.state = StateHalfOpen
	case StateHalfOpen:
		// Allow one request through
	}

	cb.mu.Unlock()

	err := operation()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failureCount++
		cb.lastFailureTime = time.Now()

		if cb.failureCount >= cb.failureThreshold {
			cb.state = StateOpen
		} else if cb.state == StateHalfOpen {
			cb.state = StateOpen
		}

		return err
	}

	// Success
	if cb.state == StateHalfOpen {
		cb.state = StateClosed
	}
	cb.failureCount = 0

	return nil
}

// GetState returns the current state of the circuit breaker.
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}
