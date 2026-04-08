package metacognitive

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryStore_CompileTimeCheck(t *testing.T) {
	var _ SelfModelStore = (*InMemoryStore)(nil)
}

func TestInMemoryStore_LoadNewAgent(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	model, err := store.Load(ctx, "agent-new")
	require.NoError(t, err)
	require.NotNil(t, model)
	assert.Equal(t, "agent-new", model.AgentID)
	assert.Empty(t, model.Heuristics)
	assert.NotNil(t, model.Capabilities)
}

func TestInMemoryStore_SaveAndLoad(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	model := NewSelfModel("agent-1")
	model.Heuristics = []Heuristic{
		{ID: "h1", Content: "test heuristic", Source: "failure", TaskType: "search", Utility: 0.7},
	}
	model.Capabilities["search"] = &CapabilityScore{
		TaskType:    "search",
		SuccessRate: 0.8,
		SampleCount: 5,
	}

	err := store.Save(ctx, model)
	require.NoError(t, err)

	loaded, err := store.Load(ctx, "agent-1")
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, "agent-1", loaded.AgentID)
	assert.Len(t, loaded.Heuristics, 1)
	assert.Equal(t, "test heuristic", loaded.Heuristics[0].Content)
	assert.InDelta(t, 0.8, loaded.Capabilities["search"].SuccessRate, 0.001)
}

func TestInMemoryStore_SaveOverwrite(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	model1 := NewSelfModel("agent-1")
	model1.Heuristics = []Heuristic{{ID: "h1", Content: "first"}}
	require.NoError(t, store.Save(ctx, model1))

	model2 := NewSelfModel("agent-1")
	model2.Heuristics = []Heuristic{{ID: "h2", Content: "second"}}
	require.NoError(t, store.Save(ctx, model2))

	loaded, err := store.Load(ctx, "agent-1")
	require.NoError(t, err)
	require.Len(t, loaded.Heuristics, 1)
	assert.Equal(t, "second", loaded.Heuristics[0].Content)
}

func TestInMemoryStore_LoadReturnsCopy(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	model := NewSelfModel("agent-1")
	model.Heuristics = []Heuristic{{ID: "h1", Content: "original"}}
	require.NoError(t, store.Save(ctx, model))

	loaded, err := store.Load(ctx, "agent-1")
	require.NoError(t, err)
	loaded.Heuristics[0].Content = "mutated"

	loaded2, err := store.Load(ctx, "agent-1")
	require.NoError(t, err)
	assert.Equal(t, "original", loaded2.Heuristics[0].Content, "mutation must not affect stored model")
}

func TestInMemoryStore_EmptyAgentID(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	_, err := store.Load(ctx, "")
	assert.Error(t, err)

	err = store.Save(ctx, &SelfModel{})
	assert.Error(t, err)

	_, err = store.SearchHeuristics(ctx, "", "query", 5)
	assert.Error(t, err)
}

func TestInMemoryStore_SaveNilModel(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	err := store.Save(ctx, nil)
	assert.Error(t, err)
}

func TestInMemoryStore_ContextCancellation(t *testing.T) {
	store := NewInMemoryStore()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := store.Load(ctx, "agent-1")
	assert.Error(t, err)

	err = store.Save(ctx, NewSelfModel("agent-1"))
	assert.Error(t, err)

	_, err = store.SearchHeuristics(ctx, "agent-1", "query", 5)
	assert.Error(t, err)
}

func TestInMemoryStore_SearchHeuristics(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	model := NewSelfModel("agent-1")
	model.Heuristics = []Heuristic{
		{ID: "h1", Content: "Avoid: timeout errors in search tasks", Source: "failure", TaskType: "search", Utility: 0.9},
		{ID: "h2", Content: "Prefer: use caching for repeated queries", Source: "success", TaskType: "search", Utility: 0.8},
		{ID: "h3", Content: "Avoid: parsing errors in code generation", Source: "failure", TaskType: "coding", Utility: 0.7},
		{ID: "h4", Content: "Prefer: validate input before processing", Source: "success", TaskType: "general", Utility: 0.6},
	}
	require.NoError(t, store.Save(ctx, model))

	tests := []struct {
		name    string
		query   string
		k       int
		wantLen int
		wantIDs []string // expected IDs in order
	}{
		{
			name:    "search by task type",
			query:   "search",
			k:       5,
			wantLen: 2,
			wantIDs: []string{"h1", "h2"},
		},
		{
			name:    "search by keyword",
			query:   "timeout",
			k:       5,
			wantLen: 1,
			wantIDs: []string{"h1"},
		},
		{
			name:    "limit results",
			query:   "search errors",
			k:       1,
			wantLen: 1,
		},
		{
			name:    "no matches",
			query:   "nonexistent xyz",
			k:       5,
			wantLen: 0,
		},
		{
			name:    "k is zero",
			query:   "search",
			k:       0,
			wantLen: 0,
		},
		{
			name:    "empty query returns top by utility",
			query:   "",
			k:       2,
			wantLen: 2,
			wantIDs: []string{"h1", "h2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := store.SearchHeuristics(ctx, "agent-1", tt.query, tt.k)
			require.NoError(t, err)
			assert.Len(t, results, tt.wantLen)

			if len(tt.wantIDs) > 0 && len(results) > 0 {
				gotIDs := make([]string, len(results))
				for i, h := range results {
					gotIDs[i] = h.ID
				}
				for _, wantID := range tt.wantIDs {
					assert.Contains(t, gotIDs, wantID)
				}
			}
		})
	}
}

func TestInMemoryStore_SearchHeuristics_AgentNotFound(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	results, err := store.SearchHeuristics(ctx, "nonexistent", "query", 5)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestInMemoryStore_ConcurrentAccess(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()
	const n = 50

	var wg sync.WaitGroup
	wg.Add(n * 3)

	// Concurrent saves.
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			m := NewSelfModel("agent-1")
			m.Heuristics = []Heuristic{{ID: "h", Content: "test", Utility: 0.5}}
			_ = store.Save(ctx, m)
		}()
	}

	// Concurrent loads.
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_, _ = store.Load(ctx, "agent-1")
		}()
	}

	// Concurrent searches.
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_, _ = store.SearchHeuristics(ctx, "agent-1", "test", 3)
		}()
	}

	wg.Wait()
}

func TestInMemoryStore_SearchHeuristics_ZeroUtility(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	model := NewSelfModel("agent-1")
	model.Heuristics = []Heuristic{
		{ID: "h1", Content: "zero utility heuristic about search", Utility: 0.0, TaskType: "search"},
		{ID: "h2", Content: "positive utility about search", Utility: 0.5, TaskType: "search"},
	}
	require.NoError(t, store.Save(ctx, model))

	results, err := store.SearchHeuristics(ctx, "agent-1", "search", 5)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	// Higher utility should come first.
	assert.Equal(t, "h2", results[0].ID)
}

func TestInMemoryStore_MultipleAgents(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	m1 := NewSelfModel("agent-1")
	m1.Heuristics = []Heuristic{{ID: "h1", Content: "agent 1 heuristic"}}
	require.NoError(t, store.Save(ctx, m1))

	m2 := NewSelfModel("agent-2")
	m2.Heuristics = []Heuristic{{ID: "h2", Content: "agent 2 heuristic"}}
	require.NoError(t, store.Save(ctx, m2))

	loaded1, err := store.Load(ctx, "agent-1")
	require.NoError(t, err)
	assert.Len(t, loaded1.Heuristics, 1)
	assert.Equal(t, "agent 1 heuristic", loaded1.Heuristics[0].Content)

	loaded2, err := store.Load(ctx, "agent-2")
	require.NoError(t, err)
	assert.Len(t, loaded2.Heuristics, 1)
	assert.Equal(t, "agent 2 heuristic", loaded2.Heuristics[0].Content)
}

func TestInMemoryStore_CapabilityScoreDeepCopy(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	model := NewSelfModel("agent-1")
	model.Capabilities["search"] = &CapabilityScore{
		TaskType:    "search",
		SuccessRate: 0.8,
		SampleCount: 5,
		AvgLatency:  time.Second,
		LastUpdated: time.Now(),
	}
	require.NoError(t, store.Save(ctx, model))

	loaded, err := store.Load(ctx, "agent-1")
	require.NoError(t, err)
	loaded.Capabilities["search"].SuccessRate = 0.1

	loaded2, err := store.Load(ctx, "agent-1")
	require.NoError(t, err)
	assert.InDelta(t, 0.8, loaded2.Capabilities["search"].SuccessRate, 0.001,
		"mutation of loaded capability must not affect stored model")
}
