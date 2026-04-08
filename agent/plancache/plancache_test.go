package plancache

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEndToEnd_CostReduction verifies that repeated similar tasks reuse cached
// plans and reduce the number of inner planner calls. For 10 similar tasks,
// the mock planner should be called at most 3 times (first call + at most 2
// near misses).
func TestEndToEnd_CostReduction(t *testing.T) {
	var plannerCalls atomic.Int32

	inner := &mockPlanner{
		planFn: func(ctx context.Context, state agent.PlannerState) ([]agent.Action, error) {
			plannerCalls.Add(1)
			return []agent.Action{
				{Type: agent.ActionTool, ToolCall: &schema.ToolCall{Name: "search_db", Arguments: `{"query":"` + state.Input + `"}`}},
				{Type: agent.ActionTool, ToolCall: &schema.ToolCall{Name: "format_results"}},
				{Type: agent.ActionFinish, Message: "Here are the results"},
			}, nil
		},
	}

	store := NewInMemoryStore(100)
	matcher := mustNewMatcher("keyword")
	cp := Wrap(inner, store, matcher, WithMinScore(0.5))

	ctx := context.Background()

	// 10 similar tasks with overlapping keywords.
	inputs := []string{
		"search database for user records",
		"search database for user profiles",
		"search database for user accounts",
		"search database for user data",
		"search database for user information",
		"search database for user details",
		"search database for user entries",
		"search database for user items",
		"search database for user results",
		"search database for user documents",
	}

	for _, input := range inputs {
		state := agent.PlannerState{
			Input:    input,
			Metadata: map[string]any{"agent_id": "cost-test-agent"},
		}
		actions, err := cp.Plan(ctx, state)
		require.NoError(t, err)
		assert.NotEmpty(t, actions, "actions should not be empty for input: %s", input)
	}

	calls := plannerCalls.Load()
	assert.LessOrEqual(t, calls, int32(3),
		"inner planner should be called at most 3 times for 10 similar tasks, got %d", calls)
	t.Logf("Inner planner called %d times for %d similar inputs", calls, len(inputs))
}

// TestEndToEnd_DifferentTasks verifies that different tasks produce separate
// templates.
func TestEndToEnd_DifferentTasks(t *testing.T) {
	inner := &mockPlanner{
		planFn: func(ctx context.Context, state agent.PlannerState) ([]agent.Action, error) {
			return []agent.Action{
				{Type: agent.ActionTool, ToolCall: &schema.ToolCall{Name: "tool_" + state.Input[:4]}},
				{Type: agent.ActionFinish, Message: "done"},
			}, nil
		},
	}

	store := NewInMemoryStore(100)
	matcher := mustNewMatcher("keyword")
	cp := Wrap(inner, store, matcher, WithMinScore(0.8))

	ctx := context.Background()

	// Completely different tasks.
	tasks := []string{
		"deploy application production server",
		"analyze weather forecast data",
		"compile financial quarterly report",
	}

	for _, task := range tasks {
		_, err := cp.Plan(ctx, agent.PlannerState{
			Input:    task,
			Metadata: map[string]any{"agent_id": "diverse-agent"},
		})
		require.NoError(t, err)
	}

	templates, err := store.List(ctx, "diverse-agent")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(templates), 2, "different tasks should produce separate templates")
}

// TestEndToEnd_FullLifecycle tests the complete lifecycle: cache miss, cache
// hit, deviation tracking, and eviction.
func TestEndToEnd_FullLifecycle(t *testing.T) {
	var (
		hits      int
		misses    int
		extracted int
		evicted   int
	)

	inner := &mockPlanner{}
	store := NewInMemoryStore(100)
	matcher := mustNewMatcher("keyword")

	cp := Wrap(inner, store, matcher,
		WithMinScore(0.5),
		WithEvictionThreshold(0.3),
		WithHooks(Hooks{
			OnCacheHit:          func(ctx context.Context, tmpl *Template, score float64) { hits++ },
			OnCacheMiss:         func(ctx context.Context, input string) { misses++ },
			OnTemplateExtracted: func(ctx context.Context, tmpl *Template) { extracted++ },
			OnTemplateEvicted:   func(ctx context.Context, tmpl *Template) { evicted++ },
		}),
	)

	ctx := context.Background()
	state := newTestState("search database records")

	// Step 1: Cache miss + extraction.
	_, err := cp.Plan(ctx, state)
	require.NoError(t, err)
	assert.Equal(t, 1, misses)
	assert.Equal(t, 1, extracted)
	assert.Equal(t, 0, hits)

	// Step 2: Cache hit.
	_, err = cp.Plan(ctx, state)
	require.NoError(t, err)
	assert.Equal(t, 1, hits)

	// Step 3: Multiple replans to trigger eviction.
	for i := 0; i < 5; i++ {
		_, err = cp.Replan(ctx, state)
		require.NoError(t, err)
	}
	assert.Greater(t, evicted, 0, "template should be evicted after repeated deviation")
}

// TestEndToEnd_ConcurrentAccess verifies thread safety under concurrent load.
func TestEndToEnd_ConcurrentAccess(t *testing.T) {
	inner := &mockPlanner{}
	store := NewInMemoryStore(100)
	matcher := mustNewMatcher("keyword")
	cp := Wrap(inner, store, matcher, WithMinScore(0.5))

	ctx := context.Background()
	errs := make(chan error, 50)

	for i := 0; i < 50; i++ {
		go func(n int) {
			state := agent.PlannerState{
				Input:    fmt.Sprintf("search database for records type %d", n%5),
				Metadata: map[string]any{"agent_id": "concurrent-agent"},
			}
			_, err := cp.Plan(ctx, state)
			errs <- err
		}(i)
	}

	for i := 0; i < 50; i++ {
		assert.NoError(t, <-errs)
	}
}
