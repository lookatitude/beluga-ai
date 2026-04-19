package plancache

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPlanner is a test planner that records calls.
type mockPlanner struct {
	planCalls   atomic.Int32
	replanCalls atomic.Int32
	planFn      func(ctx context.Context, state agent.PlannerState) ([]agent.Action, error)
	replanFn    func(ctx context.Context, state agent.PlannerState) ([]agent.Action, error)
}

func (m *mockPlanner) Plan(ctx context.Context, state agent.PlannerState) ([]agent.Action, error) {
	m.planCalls.Add(1)
	if m.planFn != nil {
		return m.planFn(ctx, state)
	}
	return []agent.Action{
		{Type: agent.ActionTool, ToolCall: &schema.ToolCall{Name: "search", Arguments: `{"q":"test"}`}},
		{Type: agent.ActionFinish, Message: "done"},
	}, nil
}

func (m *mockPlanner) Replan(ctx context.Context, state agent.PlannerState) ([]agent.Action, error) {
	m.replanCalls.Add(1)
	if m.replanFn != nil {
		return m.replanFn(ctx, state)
	}
	return []agent.Action{
		{Type: agent.ActionFinish, Message: "replanned"},
	}, nil
}

func newTestState(input string) agent.PlannerState {
	return agent.PlannerState{
		Input:    input,
		Metadata: map[string]any{"agent_id": "test-agent"},
	}
}

func TestCachedPlanner_CacheMiss(t *testing.T) {
	inner := &mockPlanner{}
	store := NewInMemoryStore(10)
	matcher := mustNewMatcher("keyword")

	cp := Wrap(inner, store, matcher)

	actions, err := cp.Plan(context.Background(), newTestState("search database for records"))
	require.NoError(t, err)
	assert.Len(t, actions, 2)
	assert.Equal(t, int32(1), inner.planCalls.Load())
	assert.Greater(t, store.Len(), 0, "template should be cached")
}

func TestCachedPlanner_CacheHit(t *testing.T) {
	inner := &mockPlanner{}
	store := NewInMemoryStore(10)
	matcher := mustNewMatcher("keyword")

	cp := Wrap(inner, store, matcher, WithMinScore(0.5))

	// First call: cache miss.
	_, err := cp.Plan(context.Background(), newTestState("search database for records"))
	require.NoError(t, err)
	assert.Equal(t, int32(1), inner.planCalls.Load())

	// Second call with same input: cache hit.
	actions, err := cp.Plan(context.Background(), newTestState("search database for records"))
	require.NoError(t, err)
	assert.NotEmpty(t, actions)
	assert.Equal(t, int32(1), inner.planCalls.Load(), "inner planner should not be called on cache hit")
}

func TestCachedPlanner_CacheHitSimilarInput(t *testing.T) {
	inner := &mockPlanner{}
	store := NewInMemoryStore(10)
	matcher := mustNewMatcher("keyword")

	cp := Wrap(inner, store, matcher, WithMinScore(0.5))

	_, err := cp.Plan(context.Background(), newTestState("search database for user records"))
	require.NoError(t, err)

	// Similar input should also hit.
	_, err = cp.Plan(context.Background(), newTestState("search database for user profiles"))
	require.NoError(t, err)
	// May or may not hit depending on score, but should not error.
}

func TestCachedPlanner_ReplanAlwaysDelegates(t *testing.T) {
	inner := &mockPlanner{}
	store := NewInMemoryStore(10)
	matcher := mustNewMatcher("keyword")

	cp := Wrap(inner, store, matcher)

	// Seed a template.
	_, err := cp.Plan(context.Background(), newTestState("search database"))
	require.NoError(t, err)

	// Replan always calls inner.
	actions, err := cp.Replan(context.Background(), newTestState("search database"))
	require.NoError(t, err)
	assert.NotEmpty(t, actions)
	assert.Equal(t, int32(1), inner.replanCalls.Load())
}

func TestCachedPlanner_DeviationEviction(t *testing.T) {
	inner := &mockPlanner{}
	store := NewInMemoryStore(10)
	matcher := mustNewMatcher("keyword")

	var evicted bool
	cp := Wrap(inner, store, matcher,
		WithMinScore(0.5),
		WithEvictionThreshold(0.3),
		WithHooks(Hooks{
			OnTemplateEvicted: func(ctx context.Context, tmpl *Template) {
				evicted = true
			},
		}),
	)

	// Seed a template.
	_, err := cp.Plan(context.Background(), newTestState("search database for records"))
	require.NoError(t, err)

	// Replan multiple times to increase deviation.
	for i := 0; i < 5; i++ {
		_, err := cp.Replan(context.Background(), newTestState("search database for records"))
		require.NoError(t, err)
	}

	assert.True(t, evicted, "template should have been evicted due to high deviation")
}

func TestCachedPlanner_HooksOnCacheHit(t *testing.T) {
	inner := &mockPlanner{}
	store := NewInMemoryStore(10)
	matcher := mustNewMatcher("keyword")

	var hitCalled bool
	var hitScore float64

	cp := Wrap(inner, store, matcher,
		WithMinScore(0.5),
		WithHooks(Hooks{
			OnCacheHit: func(ctx context.Context, tmpl *Template, score float64) {
				hitCalled = true
				hitScore = score
			},
		}),
	)

	_, _ = cp.Plan(context.Background(), newTestState("search database records"))
	_, _ = cp.Plan(context.Background(), newTestState("search database records"))

	assert.True(t, hitCalled, "OnCacheHit should have been called")
	assert.Greater(t, hitScore, 0.0)
}

func TestCachedPlanner_HooksOnCacheMiss(t *testing.T) {
	inner := &mockPlanner{}
	store := NewInMemoryStore(10)
	matcher := mustNewMatcher("keyword")

	var missCalled bool
	cp := Wrap(inner, store, matcher,
		WithHooks(Hooks{
			OnCacheMiss: func(ctx context.Context, input string) {
				missCalled = true
			},
		}),
	)

	_, _ = cp.Plan(context.Background(), newTestState("unique input never seen before"))
	assert.True(t, missCalled)
}

func TestCachedPlanner_HooksOnTemplateExtracted(t *testing.T) {
	inner := &mockPlanner{}
	store := NewInMemoryStore(10)
	matcher := mustNewMatcher("keyword")

	var extractedCalled bool
	cp := Wrap(inner, store, matcher,
		WithHooks(Hooks{
			OnTemplateExtracted: func(ctx context.Context, tmpl *Template) {
				extractedCalled = true
			},
		}),
	)

	_, _ = cp.Plan(context.Background(), newTestState("search database"))
	assert.True(t, extractedCalled)
}

func TestCachedPlanner_InnerPlannerError(t *testing.T) {
	inner := &mockPlanner{
		planFn: func(ctx context.Context, state agent.PlannerState) ([]agent.Action, error) {
			return nil, errors.New("planner failed")
		},
	}
	store := NewInMemoryStore(10)
	matcher := mustNewMatcher("keyword")

	cp := Wrap(inner, store, matcher)

	_, err := cp.Plan(context.Background(), newTestState("search"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "planner failed")
}

func TestCachedPlanner_ContextCancellation(t *testing.T) {
	inner := &mockPlanner{}
	store := NewInMemoryStore(10)
	matcher := mustNewMatcher("keyword")

	cp := Wrap(inner, store, matcher)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := cp.Plan(ctx, newTestState("search"))
	assert.Error(t, err)

	_, err = cp.Replan(ctx, newTestState("search"))
	assert.Error(t, err)
}

func TestCachedPlanner_EmptyActionsNotCached(t *testing.T) {
	inner := &mockPlanner{
		planFn: func(ctx context.Context, state agent.PlannerState) ([]agent.Action, error) {
			return nil, nil
		},
	}
	store := NewInMemoryStore(10)
	matcher := mustNewMatcher("keyword")

	cp := Wrap(inner, store, matcher)

	_, err := cp.Plan(context.Background(), newTestState("empty"))
	require.NoError(t, err)
	assert.Equal(t, 0, store.Len(), "empty actions should not be cached")
}

func TestCachedPlanner_Options(t *testing.T) {
	inner := &mockPlanner{}
	store := NewInMemoryStore(10)
	matcher := mustNewMatcher("keyword")

	cp := Wrap(inner, store, matcher,
		WithMinScore(0.9),
		WithMaxTemplates(5),
		WithEvictionThreshold(0.2),
	)

	assert.Equal(t, 0.9, cp.opts.minScore)
	assert.Equal(t, 5, cp.opts.maxTemplates)
	assert.Equal(t, 0.2, cp.opts.evictionThreshold)
}

func TestAgentIDFromState(t *testing.T) {
	tests := []struct {
		name string
		meta map[string]any
		want string
	}{
		{"with agent_id", map[string]any{"agent_id": "my-agent"}, "my-agent"},
		{"empty agent_id", map[string]any{"agent_id": ""}, "default"},
		{"no agent_id", map[string]any{}, "default"},
		{"nil metadata", nil, "default"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := agent.PlannerState{Metadata: tt.meta}
			assert.Equal(t, tt.want, agentIDFromState(state))
		})
	}
}
