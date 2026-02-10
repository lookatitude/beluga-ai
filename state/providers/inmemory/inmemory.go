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

// Store is a thread-safe in-memory implementation of state.Store.
type Store struct {
	mu       sync.RWMutex
	data     map[string]any
	watchers map[string][]chan state.StateChange
	closed   bool
	done     chan struct{} // closed on Close() to unblock context goroutines
}

// New creates a new in-memory Store.
func New() *Store {
	return &Store{
		data:     make(map[string]any),
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

	val, ok := s.data[key]
	if !ok {
		return nil, nil
	}
	return val, nil
}

// Set stores a value under the given key.
func (s *Store) Set(ctx context.Context, key string, value any) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("state/set: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("state/set: store is closed")
	}

	oldValue := s.data[key]
	s.data[key] = value

	s.broadcast(state.StateChange{
		Key:      key,
		OldValue: oldValue,
		Value:    value,
		Op:       state.OpSet,
	})

	return nil
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

	oldValue, exists := s.data[key]
	if !exists {
		return nil
	}

	delete(s.data, key)

	s.broadcast(state.StateChange{
		Key:      key,
		OldValue: oldValue,
		Value:    nil,
		Op:       state.OpDelete,
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

// Ensure Store implements state.Store at compile time.
var _ state.Store = (*Store)(nil)
