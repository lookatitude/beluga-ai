package audit

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/core"
)

// Compile-time check that InMemoryStore satisfies Store.
var _ Store = (*InMemoryStore)(nil)

// InMemoryStore is a thread-safe, in-memory implementation of [Store].
// It is intended for development and testing. All entries are stored in a
// slice protected by a read/write mutex.
type InMemoryStore struct {
	mu      sync.RWMutex
	entries []Entry
}

// InMemoryOption is a functional option for [NewInMemoryStore].
type InMemoryOption func(*InMemoryStore)

// NewInMemoryStore creates a new [InMemoryStore] with the given options applied.
func NewInMemoryStore(opts ...InMemoryOption) *InMemoryStore {
	s := &InMemoryStore{}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Log records the entry, enriching it with a generated ID and current
// timestamp when those fields are zero.
//
// Log returns an error if:
//   - the context is cancelled before the entry is stored, or
//   - ID generation fails.
func (s *InMemoryStore) Log(ctx context.Context, entry Entry) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("audit: Log: %w", err)
	}

	enriched, err := enrichEntry(entry)
	if err != nil {
		return core.NewError("audit.Log", core.ErrInvalidInput, "failed to generate entry ID", err)
	}

	s.mu.Lock()
	s.entries = append(s.entries, enriched)
	s.mu.Unlock()
	return nil
}

// Query returns all stored entries that match the filter. Entries are returned
// in insertion order. When filter.Limit is positive the result is capped at
// that many entries.
//
// Query returns an error if the context is cancelled.
func (s *InMemoryStore) Query(ctx context.Context, filter Filter) ([]Entry, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("audit: Query: %w", err)
	}

	s.mu.RLock()
	snapshot := make([]Entry, len(s.entries))
	copy(snapshot, s.entries)
	s.mu.RUnlock()

	var result []Entry
	for _, e := range snapshot {
		if !matchesFilter(e, filter) {
			continue
		}
		result = append(result, e)
		if filter.Limit > 0 && len(result) >= filter.Limit {
			break
		}
	}
	return result, nil
}

// matchesFilter reports whether e satisfies all non-zero fields of filter.
func matchesFilter(e Entry, f Filter) bool {
	if f.TenantID != "" && e.TenantID != f.TenantID {
		return false
	}
	if f.AgentID != "" && e.AgentID != f.AgentID {
		return false
	}
	if f.SessionID != "" && e.SessionID != f.SessionID {
		return false
	}
	if f.Action != "" && e.Action != f.Action {
		return false
	}
	if !f.Since.IsZero() && e.Timestamp.Before(f.Since) {
		return false
	}
	if !f.Until.IsZero() && e.Timestamp.After(f.Until) {
		return false
	}
	return true
}

func init() {
	Register("inmemory", func(cfg Config) (Store, error) {
		return NewInMemoryStore(), nil
	})
}
