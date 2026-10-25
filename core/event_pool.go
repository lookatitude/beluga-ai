package core

import (
	"reflect"
	"sync"
)

// eventPools holds one sync.Pool per Event payload type, keyed by the
// reflect.Type of T. A single shared pool cannot recycle Event[T] values for
// more than one T, because a pooled *Event[A] is useless to a caller asking
// for *Event[B].
var eventPools sync.Map // map[reflect.Type]*sync.Pool

// poolFor returns the pool for Event[T], creating it on first use.
func poolFor[T any]() *sync.Pool {
	key := reflect.TypeFor[T]()
	if p, ok := eventPools.Load(key); ok {
		return p.(*sync.Pool)
	}
	p, _ := eventPools.LoadOrStore(key, &sync.Pool{
		New: func() any { return &Event[T]{} },
	})
	return p.(*sync.Pool)
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

// AcquireEvent retrieves a pre-allocated *Event[T] from the pool for type T.
// The caller must call [ReleaseEvent] when done to return the Event for reuse.
//
// Each payload type T has its own pool, so in steady state (after a matching
// ReleaseEvent) AcquireEvent returns a recycled instance for any T. sync.Pool
// may still drop pooled items at garbage collection, in which case a fresh
// Event is allocated.
func AcquireEvent[T any]() *Event[T] {
	return poolFor[T]().Get().(*Event[T])
}

// ReleaseEvent resets e and returns it to the pool for type T.
// The caller must not access e after calling ReleaseEvent.
func ReleaseEvent[T any](e *Event[T]) {
	if e == nil {
		return
	}
	e.Reset()
	poolFor[T]().Put(e)
}
