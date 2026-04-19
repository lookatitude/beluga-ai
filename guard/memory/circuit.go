package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/resilience"
)

// InterAgentCircuitBreaker manages per-agent-pair circuit breakers that
// isolate writers from shared memory when poisoning is detected. Each
// writer-reader pair gets its own circuit breaker, allowing fine-grained
// isolation of compromised agents without affecting healthy communication
// paths.
type InterAgentCircuitBreaker struct {
	failureThreshold int
	resetTimeout     time.Duration
	hooks            Hooks

	mu       sync.RWMutex
	breakers map[string]*resilience.CircuitBreaker
}

// CircuitBreakerOption configures an InterAgentCircuitBreaker.
type CircuitBreakerOption func(*InterAgentCircuitBreaker)

// WithFailureThreshold sets the number of poisoning detections before the
// circuit trips. Default is 3.
func WithFailureThreshold(n int) CircuitBreakerOption {
	return func(cb *InterAgentCircuitBreaker) {
		if n > 0 {
			cb.failureThreshold = n
		}
	}
}

// WithResetTimeout sets how long a tripped circuit stays open before
// transitioning to half-open. Default is 5 minutes.
func WithResetTimeout(d time.Duration) CircuitBreakerOption {
	return func(cb *InterAgentCircuitBreaker) {
		if d > 0 {
			cb.resetTimeout = d
		}
	}
}

// WithCircuitHooks sets hooks for circuit breaker events.
func WithCircuitHooks(h Hooks) CircuitBreakerOption {
	return func(cb *InterAgentCircuitBreaker) {
		cb.hooks = h
	}
}

// NewInterAgentCircuitBreaker creates a new circuit breaker manager for
// inter-agent memory access.
func NewInterAgentCircuitBreaker(opts ...CircuitBreakerOption) *InterAgentCircuitBreaker {
	cb := &InterAgentCircuitBreaker{
		failureThreshold: 3,
		resetTimeout:     5 * time.Minute,
		breakers:         make(map[string]*resilience.CircuitBreaker),
	}
	for _, opt := range opts {
		opt(cb)
	}
	return cb
}

// pairKey returns a deterministic, collision-free key for a writer-reader
// agent pair. The encoding is length-prefixed ("<len(writer)>:<writer>|<reader>")
// so that distinct writer-reader pairs can never produce the same key even if
// their identifiers contain the delimiter characters.
func pairKey(writer, reader string) string {
	return fmt.Sprintf("%d:%s|%s", len(writer), writer, reader)
}

// getOrCreateBreaker returns the circuit breaker for the given agent pair,
// creating one if it does not exist.
func (iacb *InterAgentCircuitBreaker) getOrCreateBreaker(writer, reader string) *resilience.CircuitBreaker {
	key := pairKey(writer, reader)

	iacb.mu.RLock()
	cb, ok := iacb.breakers[key]
	iacb.mu.RUnlock()
	if ok {
		return cb
	}

	iacb.mu.Lock()
	defer iacb.mu.Unlock()

	// Double-check after acquiring write lock.
	if cb, ok = iacb.breakers[key]; ok {
		return cb
	}

	cb = resilience.NewCircuitBreaker(iacb.failureThreshold, iacb.resetTimeout)
	iacb.breakers[key] = cb
	return cb
}

// Allow checks whether the writer is allowed to write to memory that the
// reader will consume. Returns an error if the circuit is open.
func (iacb *InterAgentCircuitBreaker) Allow(ctx context.Context, writer, reader string) error {
	cb := iacb.getOrCreateBreaker(writer, reader)
	state := cb.State()

	if state == resilience.StateOpen {
		return core.NewError(
			"guard/memory.circuit",
			core.ErrGuardBlocked,
			fmt.Sprintf("circuit open for %s -> %s", writer, reader),
			resilience.ErrCircuitOpen,
		)
	}

	return nil
}

// RecordPoisoning records a poisoning detection event for the writer-reader
// pair. This may trip the circuit breaker, isolating the writer.
func (iacb *InterAgentCircuitBreaker) RecordPoisoning(ctx context.Context, writer, reader string) {
	cb := iacb.getOrCreateBreaker(writer, reader)

	prevState := cb.State()

	// If the circuit is already open, Execute would short-circuit with
	// ErrCircuitOpen and never record the failure, letting the reset timer
	// expire despite ongoing poisoning. Trip() bumps the last-failure
	// timestamp and keeps the breaker open.
	if prevState == resilience.StateOpen {
		cb.Trip()
		return
	}

	// Record failure by executing a function that always fails.
	_, _ = cb.Execute(ctx, func(_ context.Context) (any, error) {
		return nil, core.Errorf(core.ErrGuardBlocked, "poisoning detected")
	})

	newState := cb.State()

	// Fire hook if circuit just tripped.
	if prevState != resilience.StateOpen && newState == resilience.StateOpen {
		if iacb.hooks.OnCircuitTripped != nil {
			iacb.hooks.OnCircuitTripped(ctx, writer, reader)
		}
	}
}

// RecordSuccess records a successful (clean) write for the writer-reader pair.
// In half-open state this may close the circuit.
func (iacb *InterAgentCircuitBreaker) RecordSuccess(ctx context.Context, writer, reader string) {
	cb := iacb.getOrCreateBreaker(writer, reader)
	_, _ = cb.Execute(ctx, func(_ context.Context) (any, error) {
		return nil, nil
	})
}

// State returns the current circuit breaker state for a writer-reader pair.
// Returns StateClosed if no breaker exists for the pair.
func (iacb *InterAgentCircuitBreaker) State(writer, reader string) resilience.State {
	key := pairKey(writer, reader)

	iacb.mu.RLock()
	cb, ok := iacb.breakers[key]
	iacb.mu.RUnlock()

	if !ok {
		return resilience.StateClosed
	}
	return cb.State()
}

// Reset manually resets the circuit breaker for a writer-reader pair.
func (iacb *InterAgentCircuitBreaker) Reset(writer, reader string) {
	key := pairKey(writer, reader)

	iacb.mu.RLock()
	cb, ok := iacb.breakers[key]
	iacb.mu.RUnlock()

	if ok {
		cb.Reset()
	}
}

// ListTripped returns the keys of all agent pairs whose circuits are currently
// open, sorted alphabetically.
func (iacb *InterAgentCircuitBreaker) ListTripped() []string {
	iacb.mu.RLock()
	defer iacb.mu.RUnlock()

	var tripped []string
	for key, cb := range iacb.breakers {
		if cb.State() == resilience.StateOpen {
			tripped = append(tripped, key)
		}
	}
	sort.Strings(tripped)
	return tripped
}
