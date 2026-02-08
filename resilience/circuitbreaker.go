package resilience

import (
	"context"
	"errors"
	"sync"
	"time"
)

// State represents the current state of a CircuitBreaker.
type State string

const (
	// StateClosed is the normal operating state where requests flow through.
	StateClosed State = "closed"

	// StateOpen indicates the circuit is tripped; all requests fail immediately.
	StateOpen State = "open"

	// StateHalfOpen allows a single probe request to determine recovery.
	StateHalfOpen State = "half_open"
)

// ErrCircuitOpen is returned when the circuit breaker is in the open state and
// the reset timeout has not yet elapsed.
var ErrCircuitOpen = errors.New("resilience: circuit breaker is open")

// CircuitBreaker implements the circuit-breaker stability pattern. It wraps
// function calls and short-circuits when a failure threshold is exceeded,
// giving the downstream dependency time to recover.
//
// State transitions:
//
//	closed  → open      after failureThreshold consecutive failures
//	open    → half-open after resetTimeout elapses
//	half-open → closed  on a successful probe call
//	half-open → open    on a failed probe call
type CircuitBreaker struct {
	failureThreshold int
	resetTimeout     time.Duration

	mu          sync.Mutex
	state       State
	failures    int
	lastFailure time.Time
	// successes tracks consecutive successes in half-open state.
	successes int
}

// NewCircuitBreaker creates a CircuitBreaker that opens after
// failureThreshold consecutive failures and stays open for resetTimeout
// before transitioning to half-open.
func NewCircuitBreaker(failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	if failureThreshold <= 0 {
		failureThreshold = 5
	}
	if resetTimeout <= 0 {
		resetTimeout = 30 * time.Second
	}
	return &CircuitBreaker{
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
		state:            StateClosed,
	}
}

// State returns the current state of the circuit breaker. If the breaker is
// open and the reset timeout has elapsed, it reports StateHalfOpen.
func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.stateLocked()
}

// stateLocked returns the effective state, promoting open → half-open when
// the reset timeout has elapsed. Caller must hold cb.mu.
func (cb *CircuitBreaker) stateLocked() State {
	if cb.state == StateOpen && time.Since(cb.lastFailure) >= cb.resetTimeout {
		cb.state = StateHalfOpen
		cb.successes = 0
	}
	return cb.state
}

// Reset manually resets the circuit breaker to the closed state, clearing
// all failure counters.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
	cb.lastFailure = time.Time{}
}

// Execute runs fn through the circuit breaker. If the circuit is open,
// ErrCircuitOpen is returned without calling fn. In half-open state a single
// probe call is allowed.
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(ctx context.Context) (any, error)) (any, error) {
	cb.mu.Lock()
	s := cb.stateLocked()
	if s == StateOpen {
		cb.mu.Unlock()
		return nil, ErrCircuitOpen
	}
	cb.mu.Unlock()

	result, err := fn(ctx)

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.recordFailure()
		return result, err
	}

	cb.recordSuccess()
	return result, nil
}

// recordFailure increments the failure counter and may trip the breaker.
// Caller must hold cb.mu.
func (cb *CircuitBreaker) recordFailure() {
	cb.failures++
	cb.lastFailure = time.Now()
	cb.successes = 0

	switch cb.state {
	case StateClosed:
		if cb.failures >= cb.failureThreshold {
			cb.state = StateOpen
		}
	case StateHalfOpen:
		// A single failure in half-open sends us back to open.
		cb.state = StateOpen
	}
}

// recordSuccess resets failure counters and may close the breaker.
// Caller must hold cb.mu.
func (cb *CircuitBreaker) recordSuccess() {
	cb.successes++
	cb.failures = 0

	if cb.state == StateHalfOpen {
		cb.state = StateClosed
		cb.lastFailure = time.Time{}
	}
}
