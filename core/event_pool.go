package core

import "sync"

// eventPool is the backing sync.Pool for zero-allocation event recycling.
// It stores *Event[any] as the common pool element; AcquireEvent and
// ReleaseEvent cast to the correct generic instantiation.
var eventPool = sync.Pool{
	New: func() any { return &Event[any]{} },
}

// Reset clears all fields on e so that the Event can be safely returned to
// the pool and reused without leaking data from a previous iteration.
func (e *Event[T]) Reset() {
	var zero T
	e.Type = ""
	e.Payload = zero
	e.Err = nil
	e.Meta = nil
}

// AcquireEvent retrieves a pre-allocated *Event[T] from the shared pool.
// The caller must call [ReleaseEvent] when done to return the Event for reuse.
//
// Because sync.Pool is untyped internally, a new Event[T] is allocated on the
// first call for a given type T; subsequent calls in steady state return
// recycled instances with zero allocations.
func AcquireEvent[T any]() *Event[T] {
	// Attempt to reuse a pooled *Event[T] directly.
	if v := eventPool.Get(); v != nil {
		if e, ok := v.(*Event[T]); ok {
			return e
		}
	}
	// Pool held a differently-typed Event (or the pool was empty).
	// Allocate a fresh one; the discarded item will be GC'd.
	return &Event[T]{}
}

// ReleaseEvent resets e and returns it to the shared pool for reuse.
// The caller must not access e after calling ReleaseEvent.
func ReleaseEvent[T any](e *Event[T]) {
	if e == nil {
		return
	}
	e.Reset()
	eventPool.Put(e)
}
