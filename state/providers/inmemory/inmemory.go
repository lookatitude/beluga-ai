package inmemory

import (
	"context"
	"fmt"
	"iter"
	"sync"

	"github.com/lookatitude/beluga-ai/v2/state"
)

func init() {
	state.Register("inmemory", func(cfg state.Config) (state.Store, error) {
		return New(), nil
	})
}

// entry holds a value and its monotonic version counter.
type entry struct {
	value   any
	version uint64
}

// Store is a thread-safe in-memory implementation of state.VersionedStore.
type Store struct {
	mu       sync.RWMutex
	data     map[string]entry
	watchers map[string][]chan state.StateChange
	closed   bool
	done     chan struct{} // closed on Close() to unblock context goroutines
}

// New creates a new in-memory Store.
func New() *Store {
	return &Store{
		data:     make(map[string]entry),
		watchers: make(map[string][]chan state.StateChange),
		done:     make(chan struct{}),
	}
}

// Get retrieves the value for the given key. Returns nil, nil if the key
// does not exist.
func (s *Store) Get(ctx context.Context, key string) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("state/get: %w", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, fmt.Errorf("state/get: store is closed")
	}

	e, ok := s.data[key]
	if !ok {
		return nil, nil
	}
	return e.value, nil
}

// GetVersioned retrieves the value and current version for the given key.
// Returns (nil, 0, nil) if the key does not exist.
func (s *Store) GetVersioned(ctx context.Context, key string) (any, uint64, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, fmt.Errorf("state/get_versioned: %w", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, 0, fmt.Errorf("state/get_versioned: store is closed")
	}

	e, ok := s.data[key]
	if !ok {
		return nil, 0, nil
	}
	return e.value, e.version, nil
}

// Set stores a value under the given key, incrementing the version counter.
func (s *Store) Set(ctx context.Context, key string, value any) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("state/set: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("state/set: store is closed")
	}

	e := s.data[key]
	oldValue := e.value
	e.version++
	e.value = value
	s.data[key] = e

	s.broadcast(state.StateChange{
		Key:      key,
		OldValue: oldValue,
		Value:    value,
		Op:       state.OpSet,
		Version:  e.version,
	})

	return nil
}

// CompareAndSwap atomically sets the value for key only if the current version
// matches expectedVersion. Returns the new version on success. For new keys,
// expectedVersion must be 0.
func (s *Store) CompareAndSwap(ctx context.Context, key string, expectedVersion uint64, value any) (uint64, error) {
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("state/cas: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return 0, fmt.Errorf("state/cas: store is closed")
	}

	e, exists := s.data[key]
	currentVersion := e.version
	if !exists {
		currentVersion = 0
	}

	if currentVersion != expectedVersion {
		return currentVersion, state.ErrVersionMismatch
	}

	oldValue := e.value
	e.version = currentVersion + 1
	e.value = value
	s.data[key] = e

	s.broadcast(state.StateChange{
		Key:      key,
		OldValue: oldValue,
		Value:    value,
		Op:       state.OpSet,
		Version:  e.version,
	})

	return e.version, nil
}

// Delete removes the given key. Deleting a non-existent key is a no-op.
func (s *Store) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("state/delete: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("state/delete: store is closed")
	}

	e, exists := s.data[key]
	if !exists {
		return nil
	}

	delete(s.data, key)

	s.broadcast(state.StateChange{
		Key:      key,
		OldValue: e.value,
		Value:    nil,
		Op:       state.OpDelete,
		Version:  e.version + 1,
	})

	return nil
}

// Watch returns an iter.Seq2 stream of StateChange notifications for the
// given key. The subscription is established eagerly before Watch returns,
// so events produced after this call but before the caller starts iterating
// are buffered (capacity 16) and will be delivered on the first iteration.
// Events that arrive while the buffer is full are dropped.
//
// The iterator ends when ctx is cancelled, the store is closed, or the
// caller breaks out of the loop. Initial-subscription errors (ctx already
// cancelled, store already closed) are reported by yielding a zero-value
// StateChange together with a non-nil error.
func (s *Store) Watch(ctx context.Context, key string) iter.Seq2[state.StateChange, error] {
	if err := ctx.Err(); err != nil {
		wrapped := fmt.Errorf("state/watch: %w", err)
		return func(yield func(state.StateChange, error) bool) {
			yield(state.StateChange{}, wrapped)
		}
	}

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		err := fmt.Errorf("state/watch: store is closed")
		return func(yield func(state.StateChange, error) bool) {
			yield(state.StateChange{}, err)
		}
	}

	ch := make(chan state.StateChange, 16)
	s.watchers[key] = append(s.watchers[key], ch)
	s.mu.Unlock()

	var unsubOnce sync.Once
	unsub := func() {
		unsubOnce.Do(func() {
			s.removeWatcherOrphan(key, ch)
		})
	}

	// Background goroutine unsubscribes when ctx is cancelled or the store
	// is closed, so callers who never iterate still release the watcher slot.
	go func() {
		select {
		case <-ctx.Done():
			unsub()
		case <-s.done:
			unsub()
		}
	}()

	return func(yield func(state.StateChange, error) bool) {
		defer unsub()
		for {
			select {
			case <-ctx.Done():
				return
			case <-s.done:
				return
			case change, ok := <-ch:
				if !ok {
					return
				}
				if !yield(change, nil) {
					return
				}
			}
		}
	}
}

// Close releases resources and signals all active watcher iterators to exit
// by closing the done channel. Individual watcher channels are not closed —
// iterators observe the done signal via select and unsubscribe themselves.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	close(s.done)

	for key := range s.watchers {
		delete(s.watchers, key)
	}

	return nil
}

// broadcast sends a change to all watchers for the given key.
// Must be called with s.mu held.
func (s *Store) broadcast(change state.StateChange) {
	chs, ok := s.watchers[change.Key]
	if !ok {
		return
	}

	for _, ch := range chs {
		select {
		case ch <- change:
		default:
			// Drop if the watcher is not keeping up.
		}
	}
}

// removeWatcherOrphan removes a specific channel from the watchers list for
// a key without closing it. The channel is orphaned; any pending broadcast
// uses a non-blocking send (see broadcast) so there is no panic risk from
// the unclosed channel.
func (s *Store) removeWatcherOrphan(key string, target chan state.StateChange) {
	s.mu.Lock()
	defer s.mu.Unlock()

	chs, ok := s.watchers[key]
	if !ok {
		return
	}

	for i, ch := range chs {
		if ch == target {
			s.watchers[key] = append(chs[:i], chs[i+1:]...)
			break
		}
	}

	if len(s.watchers[key]) == 0 {
		delete(s.watchers, key)
	}
}

// Compile-time checks.
var _ state.Store = (*Store)(nil)
var _ state.VersionedStore = (*Store)(nil)
