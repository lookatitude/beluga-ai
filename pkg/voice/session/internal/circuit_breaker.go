package internal

import (
	"fmt"
	"sync"
	"time"
)

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	CircuitBreakerClosed CircuitBreakerState = iota
	CircuitBreakerOpen
	CircuitBreakerHalfOpen
)

// CircuitBreaker implements circuit breaker pattern for provider failures
type CircuitBreaker struct {
	mu                 sync.RWMutex
	state              CircuitBreakerState
	failureCount       int
	successCount       int
	maxFailures        int
	halfOpenMaxSuccess int
	resetTimeout       time.Duration
	lastFailureTime    time.Time
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(maxFailures, halfOpenMaxSuccess int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:              CircuitBreakerClosed,
		maxFailures:        maxFailures,
		halfOpenMaxSuccess: halfOpenMaxSuccess,
		resetTimeout:       resetTimeout,
	}
}

// Call executes a function with circuit breaker protection
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	state := cb.state
	cb.mu.Unlock()

	// Check if circuit is open
	if state == CircuitBreakerOpen {
		cb.mu.Lock()
		// Check if reset timeout has passed
		if time.Since(cb.lastFailureTime) >= cb.resetTimeout {
			cb.state = CircuitBreakerHalfOpen
			cb.successCount = 0
		}
		cb.mu.Unlock()

		if cb.state == CircuitBreakerOpen {
			return fmt.Errorf("circuit breaker is open")
		}
	}

	// Execute function
	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.recordFailure()
		return err
	}

	cb.recordSuccess()
	return nil
}

// recordFailure records a failure
func (cb *CircuitBreaker) recordFailure() {
	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.state == CircuitBreakerHalfOpen {
		// Half-open -> Open on failure
		cb.state = CircuitBreakerOpen
		cb.successCount = 0
	} else if cb.failureCount >= cb.maxFailures {
		// Closed -> Open on too many failures
		cb.state = CircuitBreakerOpen
	}
}

// recordSuccess records a success
func (cb *CircuitBreaker) recordSuccess() {
	cb.failureCount = 0

	if cb.state == CircuitBreakerHalfOpen {
		cb.successCount++
		if cb.successCount >= cb.halfOpenMaxSuccess {
			// Half-open -> Closed on enough successes
			cb.state = CircuitBreakerClosed
			cb.successCount = 0
		}
	}
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}
