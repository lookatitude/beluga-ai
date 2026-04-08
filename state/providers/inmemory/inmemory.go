package inmemory

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/state"
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

// Watch returns a channel that receives StateChange notifications for the
// given key. The channel is buffered with capacity 16 to reduce blocking.
// The channel is closed when the store is closed or the context is cancelled.
func (s *Store) Watch(ctx context.Context, key string) (<-chan state.StateChange, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("state/watch: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("state/watch: store is closed")
	}

	ch := make(chan state.StateChange, 16)
	s.watchers[key] = append(s.watchers[key], ch)

	// Close the channel when the context is cancelled or the store is closed.
	go func() {
		select {
		case <-ctx.Done():
			s.removeWatcher(key, ch)
		case <-s.done:
			// Store was closed; channels already closed by Close().
		}
	}()

	return ch, nil
}

// Close releases resources and closes all watcher channels.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	close(s.done)

	for key, chs := range s.watchers {
		for _, ch := range chs {
			close(ch)
		}
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

// removeWatcher removes a specific channel from the watchers for a key.
func (s *Store) removeWatcher(key string, target chan state.StateChange) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return // channels already closed by Close()
	}

	chs, ok := s.watchers[key]
	if !ok {
		return
	}

	for i, ch := range chs {
		if ch == target {
			s.watchers[key] = append(chs[:i], chs[i+1:]...)
			close(ch)
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
