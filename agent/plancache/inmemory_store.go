package plancache

import (
	"context"
	"sync"
	"time"
)

// InMemoryStore is a thread-safe, bounded-capacity, in-memory Store
// implementation with LRU eviction.
type InMemoryStore struct {
	mu       sync.RWMutex
	data     map[string]*Template
	order    []string // LRU order: most recently used at the end
	capacity int
}

var _ Store = (*InMemoryStore)(nil)

// NewInMemoryStore creates a new InMemoryStore with the given capacity.
// When the store reaches capacity, the least recently used template is evicted.
func NewInMemoryStore(capacity int) *InMemoryStore {
	if capacity <= 0 {
		capacity = 100
	}
	return &InMemoryStore{
		data:     make(map[string]*Template),
		order:    make([]string, 0, capacity),
		capacity: capacity,
	}
}

// Save persists a template. If a template with the same ID already exists, it
// is updated. If the store is at capacity and the template is new, the least
// recently used template is evicted.
func (s *InMemoryStore) Save(ctx context.Context, tmpl *Template) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if tmpl == nil {
		return newCacheError("inmemory.Save", "template must not be nil", nil)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	if existing, ok := s.data[tmpl.ID]; ok {
		// Update existing: preserve creation time, bump version.
		tmpl.CreatedAt = existing.CreatedAt
		tmpl.Version = existing.Version + 1
		tmpl.UpdatedAt = now
		s.data[tmpl.ID] = tmpl
		s.touchLocked(tmpl.ID)
		return nil
	}

	// New template: evict LRU if at capacity.
	if len(s.data) >= s.capacity {
		s.evictLRULocked()
	}

	if tmpl.CreatedAt.IsZero() {
		tmpl.CreatedAt = now
	}
	tmpl.UpdatedAt = now
	s.data[tmpl.ID] = tmpl
	s.order = append(s.order, tmpl.ID)
	return nil
}

// Get retrieves a template by ID. Returns an error with code
// ErrTemplateNotFound if the template does not exist.
func (s *InMemoryStore) Get(ctx context.Context, id string) (*Template, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tmpl, ok := s.data[id]
	if !ok {
		return nil, newNotFoundError("inmemory.Get", id)
	}
	s.touchLocked(id)
	return copyTemplate(tmpl), nil
}

// List returns all templates for the given agent ID. Returns an empty slice
// if no templates are found.
func (s *InMemoryStore) List(ctx context.Context, agentID string) ([]*Template, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Template
	for _, tmpl := range s.data {
		if tmpl.AgentID == agentID {
			result = append(result, copyTemplate(tmpl))
		}
	}
	return result, nil
}

// Delete removes a template by ID. Returns nil if the template does not exist.
func (s *InMemoryStore) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.data[id]; !ok {
		return nil
	}
	delete(s.data, id)
	s.removeFromOrderLocked(id)
	return nil
}

// touchLocked moves an ID to the end of the LRU order (most recently used).
// Must be called with mu held.
func (s *InMemoryStore) touchLocked(id string) {
	s.removeFromOrderLocked(id)
	s.order = append(s.order, id)
}

// removeFromOrderLocked removes an ID from the LRU order slice.
// Must be called with mu held.
func (s *InMemoryStore) removeFromOrderLocked(id string) {
	for i, oid := range s.order {
		if oid == id {
			s.order = append(s.order[:i], s.order[i+1:]...)
			return
		}
	}
}

// evictLRULocked removes the least recently used template.
// Must be called with mu held.
func (s *InMemoryStore) evictLRULocked() {
	if len(s.order) == 0 {
		return
	}
	oldest := s.order[0]
	s.order = s.order[1:]
	delete(s.data, oldest)
}

// Len returns the number of templates currently stored. Useful for testing.
func (s *InMemoryStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}

// copyTemplate returns a shallow copy of the template with copied slices to
// prevent data races when callers mutate the returned value.
func copyTemplate(t *Template) *Template {
	cp := *t
	if t.Keywords != nil {
		cp.Keywords = make([]string, len(t.Keywords))
		copy(cp.Keywords, t.Keywords)
	}
	if t.Actions != nil {
		cp.Actions = make([]TemplateAction, len(t.Actions))
		copy(cp.Actions, t.Actions)
	}
	return &cp
}
