package metacognitive

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// SelfModelStore persists and retrieves self-models across sessions.
type SelfModelStore interface {
	// Load retrieves the self-model for the given agent ID.
	// Returns a new empty model if none exists.
	Load(ctx context.Context, agentID string) (*SelfModel, error)

	// Save persists the self-model.
	Save(ctx context.Context, model *SelfModel) error

	// SearchHeuristics finds the k most relevant heuristics for a query.
	// Implementations may use embedding similarity or keyword matching.
	SearchHeuristics(ctx context.Context, agentID, query string, k int) ([]Heuristic, error)
}

// Compile-time check.
var _ SelfModelStore = (*InMemoryStore)(nil)

// InMemoryStore is a thread-safe in-memory implementation of SelfModelStore.
// Suitable for testing and single-process deployments.
type InMemoryStore struct {
	mu     sync.RWMutex
	models map[string]*SelfModel
}

// NewInMemoryStore creates a new InMemoryStore.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		models: make(map[string]*SelfModel),
	}
}

// Load retrieves the self-model for the given agent ID. If no model exists,
// a new empty model is created and returned (but not persisted until Save).
func (s *InMemoryStore) Load(ctx context.Context, agentID string) (*SelfModel, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if agentID == "" {
		return nil, core.NewError("metacognitive.store.load", core.ErrInvalidInput, "agent ID must not be empty", nil)
	}

	s.mu.RLock()
	m, ok := s.models[agentID]
	s.mu.RUnlock()

	if !ok {
		return NewSelfModel(agentID), nil
	}

	// Return a copy to prevent data races on the caller side.
	return s.copyModel(m), nil
}

// Save persists the self-model. Overwrites any existing model for the agent.
func (s *InMemoryStore) Save(ctx context.Context, model *SelfModel) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if model == nil {
		return core.NewError("metacognitive.store.save", core.ErrInvalidInput, "model must not be nil", nil)
	}
	if model.AgentID == "" {
		return core.NewError("metacognitive.store.save", core.ErrInvalidInput, "agent ID must not be empty", nil)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.models[model.AgentID] = s.copyModel(model)
	return nil
}

// SearchHeuristics performs keyword-based search over an agent's heuristics.
// It scores each heuristic by counting query term matches in the content and
// task type, then returns the top k results sorted by relevance (match count
// * utility).
func (s *InMemoryStore) SearchHeuristics(ctx context.Context, agentID, query string, k int) ([]Heuristic, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if agentID == "" {
		return nil, core.NewError("metacognitive.store.search", core.ErrInvalidInput, "agent ID must not be empty", nil)
	}
	if k <= 0 {
		return nil, nil
	}

	// Use a full Lock so we can safely bump UsageCount on matched heuristics.
	s.mu.Lock()
	defer s.mu.Unlock()

	m, ok := s.models[agentID]
	if !ok || len(m.Heuristics) == 0 {
		return nil, nil
	}

	queryLower := strings.ToLower(query)
	terms := strings.Fields(queryLower)
	if len(terms) == 0 {
		// No query terms; return most useful heuristics.
		top := s.topByUtility(m.Heuristics, k)
		bumpUsageCountLocked(m, top)
		return top, nil
	}

	type scored struct {
		idx   int
		h     Heuristic
		score float64
	}

	var results []scored
	for i, h := range m.Heuristics {
		contentLower := strings.ToLower(h.Content)
		taskLower := strings.ToLower(h.TaskType)
		matches := 0
		for _, term := range terms {
			if strings.Contains(contentLower, term) || strings.Contains(taskLower, term) {
				matches++
			}
		}
		if matches > 0 {
			utility := h.Utility
			if utility <= 0 {
				utility = 0.1
			}
			results = append(results, scored{idx: i, h: h, score: float64(matches) * utility})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	out := make([]Heuristic, 0, k)
	for i := 0; i < len(results) && i < k; i++ {
		// Increment usage count on the stored heuristic so retrieval
		// statistics remain accurate across Search calls.
		m.Heuristics[results[i].idx].UsageCount++
		// Return a copy that reflects the updated UsageCount.
		h := m.Heuristics[results[i].idx]
		out = append(out, h)
	}
	return out, nil
}

// bumpUsageCountLocked increments UsageCount for every heuristic present in
// selected. Caller must hold s.mu in write mode.
func bumpUsageCountLocked(m *SelfModel, selected []Heuristic) {
	byID := make(map[string]struct{}, len(selected))
	for _, h := range selected {
		byID[h.ID] = struct{}{}
	}
	for i := range m.Heuristics {
		if _, ok := byID[m.Heuristics[i].ID]; ok {
			m.Heuristics[i].UsageCount++
		}
	}
}

// topByUtility returns the top k heuristics sorted by utility descending.
func (s *InMemoryStore) topByUtility(hs []Heuristic, k int) []Heuristic {
	sorted := make([]Heuristic, len(hs))
	copy(sorted, hs)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Utility > sorted[j].Utility
	})
	if len(sorted) > k {
		sorted = sorted[:k]
	}
	return sorted
}

// copyModel creates a deep copy of a SelfModel.
func (s *InMemoryStore) copyModel(m *SelfModel) *SelfModel {
	cp := &SelfModel{
		AgentID:      m.AgentID,
		UpdatedAt:    m.UpdatedAt,
		Capabilities: make(map[string]*CapabilityScore, len(m.Capabilities)),
	}
	if len(m.Heuristics) > 0 {
		cp.Heuristics = make([]Heuristic, len(m.Heuristics))
		copy(cp.Heuristics, m.Heuristics)
	}
	for k, v := range m.Capabilities {
		vc := *v
		cp.Capabilities[k] = &vc
	}
	return cp
}
