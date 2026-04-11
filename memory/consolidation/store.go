package consolidation

import (
	"context"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/core"
)

// ConsolidationStore is the storage interface used by the consolidation
// worker to read, update, and delete memory records. Implementations must
// be safe for concurrent use.
type ConsolidationStore interface {
	// ListRecords returns up to limit records starting from offset.
	ListRecords(ctx context.Context, offset, limit int) ([]Record, error)

	// DeleteRecords removes the records with the given IDs.
	DeleteRecords(ctx context.Context, ids []string) error

	// UpdateRecords persists changes to the given records.
	UpdateRecords(ctx context.Context, records []Record) error

	// RecordAccess increments the access counter and updates the last
	// accessed timestamp for the given record ID.
	RecordAccess(ctx context.Context, id string) error
}

// InMemoryConsolidationStore is a thread-safe, in-memory implementation of
// ConsolidationStore suitable for testing and lightweight use cases.
type InMemoryConsolidationStore struct {
	mu      sync.RWMutex
	records map[string]Record
	order   []string // insertion order for deterministic listing
}

// Compile-time interface check.
var _ ConsolidationStore = (*InMemoryConsolidationStore)(nil)

// NewInMemoryConsolidationStore creates an empty in-memory store.
func NewInMemoryConsolidationStore() *InMemoryConsolidationStore {
	return &InMemoryConsolidationStore{
		records: make(map[string]Record),
	}
}

// Add inserts a record into the store. If a record with the same ID already
// exists it is overwritten.
func (s *InMemoryConsolidationStore) Add(record Record) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.records[record.ID]; !exists {
		s.order = append(s.order, record.ID)
	}
	s.records[record.ID] = record
}

// Len returns the number of records in the store.
func (s *InMemoryConsolidationStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.records)
}

// ListRecords returns up to limit records starting from offset, in insertion
// order.
func (s *InMemoryConsolidationStore) ListRecords(ctx context.Context, offset, limit int) ([]Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if offset < 0 {
		offset = 0
	}
	if offset >= len(s.order) {
		return nil, nil
	}

	end := offset + limit
	if end > len(s.order) {
		end = len(s.order)
	}

	result := make([]Record, 0, end-offset)
	for _, id := range s.order[offset:end] {
		if r, ok := s.records[id]; ok {
			result = append(result, r)
		}
	}
	return result, nil
}

// DeleteRecords removes records by ID.
func (s *InMemoryConsolidationStore) DeleteRecords(ctx context.Context, ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		idSet[id] = struct{}{}
		delete(s.records, id)
	}

	// Compact the order slice.
	n := 0
	for _, id := range s.order {
		if _, removed := idSet[id]; !removed {
			s.order[n] = id
			n++
		}
	}
	s.order = s.order[:n]
	return nil
}

// UpdateRecords overwrites existing records. Records that do not exist are
// silently ignored.
func (s *InMemoryConsolidationStore) UpdateRecords(ctx context.Context, records []Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range records {
		if _, exists := s.records[r.ID]; exists {
			s.records[r.ID] = r
		}
	}
	return nil
}

// RecordAccess increments the access counter and updates LastAccessedAt for
// the given record ID.
func (s *InMemoryConsolidationStore) RecordAccess(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.records[id]
	if !ok {
		return core.Errorf(core.ErrNotFound, "consolidation: record %q not found", id)
	}
	r.Utility.AccessCount++
	r.Utility.LastAccessedAt = time.Now()
	s.records[id] = r
	return nil
}

// Get retrieves a single record by ID. It returns false if the record does
// not exist.
func (s *InMemoryConsolidationStore) Get(id string) (Record, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.records[id]
	return r, ok
}
