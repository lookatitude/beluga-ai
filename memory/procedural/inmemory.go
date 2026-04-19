package procedural

import (
	"context"
	"strings"
	"sync"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

// InMemoryStore is a thread-safe, map-based skill store for testing and
// development. It performs simple substring matching for search rather than
// embedding-based retrieval.
type InMemoryStore struct {
	mu     sync.RWMutex
	skills map[string]*schema.Skill
}

// NewInMemoryStore creates a new InMemoryStore.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		skills: make(map[string]*schema.Skill),
	}
}

// Save stores a skill in the map, keyed by its ID.
func (s *InMemoryStore) Save(_ context.Context, skill *schema.Skill) error {
	if skill.ID == "" {
		return core.Errorf(core.ErrInvalidInput, "procedural/inmemory: skill ID is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Store a copy to prevent external mutation.
	cp := *skill
	s.skills[skill.ID] = &cp
	return nil
}

// Get retrieves a skill by ID. Returns nil and no error if not found.
func (s *InMemoryStore) Get(_ context.Context, id string) (*schema.Skill, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sk, ok := s.skills[id]
	if !ok {
		return nil, nil
	}
	cp := *sk
	return &cp, nil
}

// Search performs case-insensitive substring matching of the query against
// skill names, descriptions, triggers, and tags. Returns at most k results.
func (s *InMemoryStore) Search(_ context.Context, query string, k int) ([]*schema.Skill, error) {
	if k <= 0 {
		k = 10
	}
	q := strings.ToLower(query)

	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*schema.Skill
	for _, sk := range s.skills {
		if matchesSkill(sk, q) {
			cp := *sk
			results = append(results, &cp)
			if len(results) >= k {
				break
			}
		}
	}
	return results, nil
}

// Delete removes a skill by ID. Returns no error if the skill does not exist.
func (s *InMemoryStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.skills, id)
	return nil
}

// All returns all skills in the store. Used for testing.
func (s *InMemoryStore) All() []*schema.Skill {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*schema.Skill, 0, len(s.skills))
	for _, sk := range s.skills {
		cp := *sk
		result = append(result, &cp)
	}
	return result
}

// matchesSkill checks if a skill's text fields contain the query substring.
func matchesSkill(sk *schema.Skill, query string) bool {
	if strings.Contains(strings.ToLower(sk.Name), query) {
		return true
	}
	if strings.Contains(strings.ToLower(sk.Description), query) {
		return true
	}
	for _, t := range sk.Triggers {
		if strings.Contains(strings.ToLower(t), query) {
			return true
		}
	}
	for _, tag := range sk.Tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return true
		}
	}
	return false
}
