package replay

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/lookatitude/beluga-ai/core"
)

// InMemoryStore is a thread-safe in-memory implementation of CheckpointStore.
// It is suitable for testing and development; production use cases should use
// a persistent store.
type InMemoryStore struct {
	mu          sync.RWMutex
	checkpoints map[string]*Checkpoint
}

// Compile-time interface check.
var _ CheckpointStore = (*InMemoryStore)(nil)

// NewInMemoryStore creates a new empty InMemoryStore.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		checkpoints: make(map[string]*Checkpoint),
	}
}

// Save persists a checkpoint in memory. If a checkpoint with the same ID
// already exists, it is overwritten.
func (s *InMemoryStore) Save(_ context.Context, cp *Checkpoint) error {
	if cp == nil {
		return core.NewError("replay.store.save", core.ErrInvalidInput, "checkpoint must not be nil", nil)
	}
	if cp.ID == "" {
		return core.NewError("replay.store.save", core.ErrInvalidInput, "checkpoint ID must not be empty", nil)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkpoints[cp.ID] = cp
	return nil
}

// Get retrieves a checkpoint by its ID. Returns a core.Error with code
// ErrNotFound if no checkpoint with the given ID exists.
func (s *InMemoryStore) Get(_ context.Context, id string) (*Checkpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cp, ok := s.checkpoints[id]
	if !ok {
		return nil, core.NewError("replay.store.get", core.ErrNotFound,
			fmt.Sprintf("checkpoint %q not found", id), nil)
	}
	return cp, nil
}

// List returns all checkpoint IDs for the given session, ordered by TurnIndex
// ascending.
func (s *InMemoryStore) List(_ context.Context, sessionID string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	type entry struct {
		id        string
		turnIndex int
	}
	var entries []entry
	for _, cp := range s.checkpoints {
		if cp.SessionID == sessionID {
			entries = append(entries, entry{id: cp.ID, turnIndex: cp.TurnIndex})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].turnIndex < entries[j].turnIndex
	})

	ids := make([]string, len(entries))
	for i, e := range entries {
		ids[i] = e.id
	}
	return ids, nil
}

// Delete removes a checkpoint by its ID. Returns a core.Error with code
// ErrNotFound if no checkpoint with the given ID exists.
func (s *InMemoryStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.checkpoints[id]; !ok {
		return core.NewError("replay.store.delete", core.ErrNotFound,
			fmt.Sprintf("checkpoint %q not found", id), nil)
	}
	delete(s.checkpoints, id)
	return nil
}
