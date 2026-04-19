package teambuilder

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/llm"
	"github.com/lookatitude/beluga-ai/v2/schema"
	"github.com/lookatitude/beluga-ai/v2/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Test mocks ---

// mockAgent is a minimal agent.Agent implementation for tests.
type mockAgent struct {
	id      string
	persona agent.Persona
}

var _ agent.Agent = (*mockAgent)(nil)

func (m *mockAgent) ID() string              { return m.id }
func (m *mockAgent) Persona() agent.Persona  { return m.persona }
func (m *mockAgent) Tools() []tool.Tool      { return nil }
func (m *mockAgent) Children() []agent.Agent { return nil }
func (m *mockAgent) Invoke(_ context.Context, input string, _ ...agent.Option) (string, error) {
	return fmt.Sprintf("[%s] %s", m.id, input), nil
}
func (m *mockAgent) Stream(_ context.Context, input string, _ ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		yield(agent.Event{Type: agent.EventText, Text: input, AgentID: m.id}, nil)
	}
}

func newMockAgent(id string, role, goal string, caps ...string) *mockAgent {
	_ = caps // capabilities are registered via pool, not agent
	return &mockAgent{
		id:      id,
		persona: agent.Persona{Role: role, Goal: goal},
	}
}

// mockChatModel satisfies llm.ChatModel for LLMSelector tests.
type mockChatModel struct {
	mu         sync.Mutex
	generateFn func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error)
	calls      int
}

var _ llm.ChatModel = (*mockChatModel)(nil)

func (m *mockChatModel) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
	m.mu.Lock()
	m.calls++
	fn := m.generateFn
	m.mu.Unlock()
	if fn != nil {
		return fn(ctx, msgs, opts...)
	}
	return &schema.AIMessage{}, nil
}

func (m *mockChatModel) Stream(_ context.Context, _ []schema.Message, _ ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	return func(yield func(schema.StreamChunk, error) bool) {}
}

func (m *mockChatModel) BindTools(_ []schema.ToolDefinition) llm.ChatModel {
	return m
}

func (m *mockChatModel) ModelID() string { return "mock-model" }

// mockSelector is a configurable Selector for testing the builder.
type mockSelector struct {
	selectFn func(ctx context.Context, task string, candidates []PoolEntry) ([]PoolEntry, error)
}

var _ Selector = (*mockSelector)(nil)

func (ms *mockSelector) Select(ctx context.Context, task string, candidates []PoolEntry) ([]PoolEntry, error) {
	return ms.selectFn(ctx, task, candidates)
}

// mockScoredSelector is a configurable ScoredSelector for testing the builder.
type mockScoredSelector struct {
	scoredFn func(ctx context.Context, task string, candidates []PoolEntry) ([]ScoredPoolEntry, error)
}

var _ ScoredSelector = (*mockScoredSelector)(nil)

func (ms *mockScoredSelector) Select(ctx context.Context, task string, candidates []PoolEntry) ([]PoolEntry, error) {
	scored, err := ms.scoredFn(ctx, task, candidates)
	if err != nil {
		return nil, err
	}
	out := make([]PoolEntry, len(scored))
	for i, s := range scored {
		out[i] = s.Entry
	}
	return out, nil
}

func (ms *mockScoredSelector) SelectScored(ctx context.Context, task string, candidates []PoolEntry) ([]ScoredPoolEntry, error) {
	return ms.scoredFn(ctx, task, candidates)
}

// --- AgentPool tests ---

func TestAgentPool_Register(t *testing.T) {
	tests := []struct {
		name    string
		agent   agent.Agent
		caps    []string
		wantErr bool
		errCode core.ErrorCode
	}{
		{
			name:  "successful registration",
			agent: newMockAgent("a1", "coder", "write code"),
			caps:  []string{"golang", "testing"},
		},
		{
			name:    "nil agent",
			agent:   nil,
			wantErr: true,
			errCode: core.ErrInvalidInput,
		},
		{
			name:    "empty ID",
			agent:   &mockAgent{id: "", persona: agent.Persona{Role: "test"}},
			wantErr: true,
			errCode: core.ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewAgentPool()
			err := pool.Register(tt.agent, tt.caps...)
			if tt.wantErr {
				require.Error(t, err)
				var coreErr *core.Error
				require.True(t, errors.As(err, &coreErr))
				assert.Equal(t, tt.errCode, coreErr.Code)
				return
			}
			require.NoError(t, err)
			entries := pool.List()
			assert.Len(t, entries, 1)
			assert.Equal(t, tt.caps, entries[0].Capabilities)
		})
	}
}

func TestAgentPool_RegisterDuplicate(t *testing.T) {
	pool := NewAgentPool()
	a := newMockAgent("a1", "coder", "write code")
	require.NoError(t, pool.Register(a, "go"))
	err := pool.Register(a, "python")
	require.Error(t, err)
	var coreErr *core.Error
	require.True(t, errors.As(err, &coreErr))
	assert.Equal(t, core.ErrInvalidInput, coreErr.Code)
}

func TestAgentPool_Unregister(t *testing.T) {
	pool := NewAgentPool()
	a := newMockAgent("a1", "coder", "write code")
	require.NoError(t, pool.Register(a, "go"))

	err := pool.Unregister("a1")
	require.NoError(t, err)
	assert.Empty(t, pool.List())

	// Unregister nonexistent.
	err = pool.Unregister("a1")
	require.Error(t, err)
	var coreErr *core.Error
	require.True(t, errors.As(err, &coreErr))
	assert.Equal(t, core.ErrNotFound, coreErr.Code)
}

func TestAgentPool_Get(t *testing.T) {
	pool := NewAgentPool()
	a := newMockAgent("a1", "coder", "write code")
	require.NoError(t, pool.Register(a, "go"))

	entry, ok := pool.Get("a1")
	assert.True(t, ok)
	assert.Equal(t, "a1", entry.Agent.ID())

	_, ok = pool.Get("nonexistent")
	assert.False(t, ok)
}

func TestAgentPool_Select(t *testing.T) {
	pool := NewAgentPool()
	a1 := newMockAgent("coder", "software engineer", "write golang code")
	a2 := newMockAgent("writer", "tech writer", "write documentation")
	require.NoError(t, pool.Register(a1, "golang", "testing"))
	require.NoError(t, pool.Register(a2, "documentation", "markdown"))

	t.Run("nil selector", func(t *testing.T) {
		_, err := pool.Select(context.Background(), "task", nil)
		require.Error(t, err)
	})

	t.Run("empty pool", func(t *testing.T) {
		emptyPool := NewAgentPool()
		sel := NewKeywordSelector()
		_, err := emptyPool.Select(context.Background(), "task", sel)
		require.Error(t, err)
	})

	t.Run("keyword select", func(t *testing.T) {
		sel := NewKeywordSelector()
		agents, err := pool.Select(context.Background(), "write golang unit tests", sel)
		require.NoError(t, err)
		require.NotEmpty(t, agents)
		// Coder should be first since "golang" and "testing" match.
		assert.Equal(t, "coder", agents[0].ID())
	})
}

func TestAgentPool_ConcurrentAccess(t *testing.T) {
	pool := NewAgentPool()
	var wg sync.WaitGroup

	// Concurrent registrations.
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			a := newMockAgent(fmt.Sprintf("agent-%d", idx), "role", "goal")
			pool.Register(a, "cap")
		}(i)
	}
	wg.Wait()

	assert.Len(t, pool.List(), 50)

	// Concurrent reads.
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pool.List()
		}()
	}
	wg.Wait()
}

// --- AgentMetrics tests ---

func TestAgentMetrics_RecordAndSnapshot(t *testing.T) {
	m := NewAgentMetrics()

	m.RecordSuccess(100 * time.Millisecond)
	m.RecordSuccess(200 * time.Millisecond)
	m.RecordFailure(50 * time.Millisecond)

	snap := m.Snapshot()
	assert.Equal(t, 3, snap.Invocations)
	assert.Equal(t, 2, snap.Successes)
	assert.Equal(t, 1, snap.Failures)
	assert.Equal(t, 350*time.Millisecond, snap.TotalLatency)
	assert.InDelta(t, 116.666, float64(snap.AvgLatency.Milliseconds()), 1)
	assert.False(t, snap.LastUsed.IsZero())
}

func TestMetricsSnapshot_SuccessRate(t *testing.T) {
	tests := []struct {
		name        string
		invocations int
		successes   int
		want        float64
	}{
		{"no invocations", 0, 0, 0},
		{"all success", 10, 10, 1.0},
		{"half success", 10, 5, 0.5},
		{"none success", 10, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := MetricsSnapshot{Invocations: tt.invocations, Successes: tt.successes}
			assert.InDelta(t, tt.want, s.SuccessRate(), 0.001)
		})
	}
}

func TestAgentMetrics_ConcurrentAccess(t *testing.T) {
	m := NewAgentMetrics()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			m.RecordSuccess(time.Millisecond)
		}()
		go func() {
			defer wg.Done()
			m.Snapshot()
		}()
	}
	wg.Wait()
	assert.Equal(t, 100, m.Snapshot().Invocations)
}

// --- KeywordSelector tests ---

func TestKeywordSelector_Select(t *testing.T) {
	coder := PoolEntry{
		Agent:        newMockAgent("coder", "software engineer", "write golang code and tests"),
		Capabilities: []string{"golang", "testing", "refactoring"},
		Metrics:      NewAgentMetrics(),
	}
	writer := PoolEntry{
		Agent:        newMockAgent("writer", "tech writer", "write documentation and tutorials"),
		Capabilities: []string{"documentation", "markdown", "tutorials"},
		Metrics:      NewAgentMetrics(),
	}
	designer := PoolEntry{
		Agent:        newMockAgent("designer", "ui designer", "design user interfaces"),
		Capabilities: []string{"figma", "css", "design"},
		Metrics:      NewAgentMetrics(),
	}

	candidates := []PoolEntry{coder, writer, designer}

	tests := []struct {
		name     string
		task     string
		minScore int
		wantIDs  []string
	}{
		{
			name:    "matches golang",
			task:    "write unit tests for golang code",
			wantIDs: []string{"coder"},
		},
		{
			name:    "matches documentation",
			task:    "write markdown documentation",
			wantIDs: []string{"writer"},
		},
		{
			name:    "matches design",
			task:    "design a new user interface with figma",
			wantIDs: []string{"designer"},
		},
		{
			name:    "empty task",
			task:    "",
			wantIDs: nil,
		},
		{
			name:    "no matches",
			task:    "xyz abc",
			wantIDs: nil,
		},
		{
			name:     "high min score filters out",
			task:     "golang",
			minScore: 10,
			wantIDs:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts []KeywordOption
			if tt.minScore > 0 {
				opts = append(opts, WithMinScore(tt.minScore))
			}
			sel := NewKeywordSelector(opts...)
			result, err := sel.Select(context.Background(), tt.task, candidates)
			require.NoError(t, err)

			if tt.wantIDs == nil {
				assert.Empty(t, result)
				return
			}

			gotIDs := make([]string, len(result))
			for i, e := range result {
				gotIDs[i] = e.Agent.ID()
			}
			// First result should match expected best.
			for i, wantID := range tt.wantIDs {
				if i < len(gotIDs) {
					assert.Equal(t, wantID, gotIDs[i])
				}
			}
		})
	}
}

func TestKeywordSelector_Ranking(t *testing.T) {
	// Both agents match, but coder should rank higher for a golang task.
	coder := PoolEntry{
		Agent:        newMockAgent("coder", "engineer", "write golang code"),
		Capabilities: []string{"golang", "testing", "code"},
		Metrics:      NewAgentMetrics(),
	}
	generalist := PoolEntry{
		Agent:        newMockAgent("generalist", "generalist", "help with anything including golang"),
		Capabilities: []string{"general", "support"},
		Metrics:      NewAgentMetrics(),
	}

	sel := NewKeywordSelector()
	result, err := sel.Select(context.Background(), "write golang tests", []PoolEntry{generalist, coder})
	require.NoError(t, err)
	require.NotEmpty(t, result)
	assert.Equal(t, "coder", result[0].Agent.ID())
}

func TestKeywordSelector_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Already cancelled.

	sel := NewKeywordSelector()
	// KeywordSelector doesn't use ctx, but it should still work.
	result, err := sel.Select(ctx, "task", []PoolEntry{
		{
			Agent:        newMockAgent("a", "r", "g"),
			Capabilities: []string{"task"},
			Metrics:      NewAgentMetrics(),
		},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

// --- LLMSelector tests ---

func TestLLMSelector_Select(t *testing.T) {
	candidates := []PoolEntry{
		{
			Agent:        newMockAgent("coder", "engineer", "write golang code"),
			Capabilities: []string{"golang"},
			Metrics:      NewAgentMetrics(),
		},
		{
			Agent:        newMockAgent("writer", "writer", "documentation"),
			Capabilities: []string{"docs"},
			Metrics:      NewAgentMetrics(),
		},
	}

	t.Run("successful selection", func(t *testing.T) {
		resp := selectionResponse{
			Selections: []agentSelection{
				{AgentID: "coder", Score: 0.9, Reasoning: "best match"},
				{AgentID: "writer", Score: 0.4, Reasoning: "partial match"},
			},
		}
		respJSON, _ := json.Marshal(resp)

		model := &mockChatModel{
			generateFn: func(_ context.Context, _ []schema.Message, _ ...llm.GenerateOption) (*schema.AIMessage, error) {
				return schema.NewAIMessage(string(respJSON)), nil
			},
		}

		sel := NewLLMSelector(model)
		result, err := sel.Select(context.Background(), "write golang tests", candidates)
		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Equal(t, "coder", result[0].Agent.ID())
	})

	t.Run("LLM error", func(t *testing.T) {
		model := &mockChatModel{
			generateFn: func(_ context.Context, _ []schema.Message, _ ...llm.GenerateOption) (*schema.AIMessage, error) {
				return nil, errors.New("model unavailable")
			},
		}

		sel := NewLLMSelector(model, WithLLMMaxRetries(0))
		_, err := sel.Select(context.Background(), "task", candidates)
		require.Error(t, err)
	})

	t.Run("empty candidates", func(t *testing.T) {
		model := &mockChatModel{}
		sel := NewLLMSelector(model)
		result, err := sel.Select(context.Background(), "task", nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("filters low scores", func(t *testing.T) {
		resp := selectionResponse{
			Selections: []agentSelection{
				{AgentID: "coder", Score: 0.2, Reasoning: "poor match"},
				{AgentID: "writer", Score: 0.1, Reasoning: "no match"},
			},
		}
		respJSON, _ := json.Marshal(resp)

		model := &mockChatModel{
			generateFn: func(_ context.Context, _ []schema.Message, _ ...llm.GenerateOption) (*schema.AIMessage, error) {
				return schema.NewAIMessage(string(respJSON)), nil
			},
		}

		sel := NewLLMSelector(model)
		result, err := sel.Select(context.Background(), "task", candidates)
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

// --- Selector Registry tests ---

func TestSelectorRegistry(t *testing.T) {
	// "keyword" is registered via init()
	names := ListSelectors()
	assert.Contains(t, names, "keyword")

	sel, err := NewSelector("keyword", nil)
	require.NoError(t, err)
	assert.NotNil(t, sel)

	_, err = NewSelector("nonexistent", nil)
	require.Error(t, err)
	var coreErr *core.Error
	require.True(t, errors.As(err, &coreErr))
	assert.Equal(t, core.ErrNotFound, coreErr.Code)
}

// --- TeamBuilder tests ---

func TestTeamBuilder_Build(t *testing.T) {
	pool := NewAgentPool()
	a1 := newMockAgent("coder", "engineer", "write golang code")
	a2 := newMockAgent("writer", "writer", "write documentation")
	a3 := newMockAgent("tester", "qa", "test software")
	require.NoError(t, pool.Register(a1, "golang", "code"))
	require.NoError(t, pool.Register(a2, "documentation", "markdown"))
	require.NoError(t, pool.Register(a3, "testing", "qa"))

	t.Run("basic build", func(t *testing.T) {
		tb := NewTeamBuilder(pool,
			WithSelector(NewKeywordSelector()),
		)
		team, err := tb.Build(context.Background(), "write golang unit tests")
		require.NoError(t, err)
		require.NotNil(t, team)
		assert.NotEmpty(t, team.Children())
	})

	t.Run("with max agents", func(t *testing.T) {
		// Use a selector that returns all agents.
		allSelector := &mockSelector{
			selectFn: func(_ context.Context, _ string, candidates []PoolEntry) ([]PoolEntry, error) {
				return candidates, nil
			},
		}
		tb := NewTeamBuilder(pool,
			WithSelector(allSelector),
			WithMaxAgents(2),
		)
		team, err := tb.Build(context.Background(), "do everything")
		require.NoError(t, err)
		assert.LessOrEqual(t, len(team.Children()), 2)
	})

	t.Run("empty task", func(t *testing.T) {
		tb := NewTeamBuilder(pool)
		_, err := tb.Build(context.Background(), "")
		require.Error(t, err)
		var coreErr *core.Error
		require.True(t, errors.As(err, &coreErr))
		assert.Equal(t, core.ErrInvalidInput, coreErr.Code)
	})

	t.Run("nil pool", func(t *testing.T) {
		tb := NewTeamBuilder(nil)
		_, err := tb.Build(context.Background(), "task")
		require.Error(t, err)
	})

	t.Run("empty pool", func(t *testing.T) {
		tb := NewTeamBuilder(NewAgentPool())
		_, err := tb.Build(context.Background(), "task")
		require.Error(t, err)
		var coreErr *core.Error
		require.True(t, errors.As(err, &coreErr))
		assert.Equal(t, core.ErrNotFound, coreErr.Code)
	})

	t.Run("selector returns empty", func(t *testing.T) {
		emptySelector := &mockSelector{
			selectFn: func(_ context.Context, _ string, _ []PoolEntry) ([]PoolEntry, error) {
				return nil, nil
			},
		}
		tb := NewTeamBuilder(pool, WithSelector(emptySelector))
		_, err := tb.Build(context.Background(), "impossible task xyz")
		require.Error(t, err)
		var coreErr *core.Error
		require.True(t, errors.As(err, &coreErr))
		assert.Equal(t, core.ErrNotFound, coreErr.Code)
	})

	t.Run("selector returns error", func(t *testing.T) {
		errSelector := &mockSelector{
			selectFn: func(_ context.Context, _ string, _ []PoolEntry) ([]PoolEntry, error) {
				return nil, errors.New("selection failure")
			},
		}
		tb := NewTeamBuilder(pool, WithSelector(errSelector))
		_, err := tb.Build(context.Background(), "task")
		require.Error(t, err)
	})

	t.Run("with custom team ID", func(t *testing.T) {
		allSelector := &mockSelector{
			selectFn: func(_ context.Context, _ string, candidates []PoolEntry) ([]PoolEntry, error) {
				return candidates[:1], nil
			},
		}
		tb := NewTeamBuilder(pool,
			WithSelector(allSelector),
			WithTeamID("custom-team"),
		)
		team, err := tb.Build(context.Background(), "task")
		require.NoError(t, err)
		assert.Equal(t, "custom-team", team.ID())
	})
}

func TestTeamBuilder_Hooks(t *testing.T) {
	pool := NewAgentPool()
	a := newMockAgent("coder", "engineer", "write code")
	require.NoError(t, pool.Register(a, "code"))

	t.Run("OnTeamFormed and OnAgentSelected fire", func(t *testing.T) {
		var teamFormedCalled bool
		var selectedAgents []string

		hooks := Hooks{
			OnTeamFormed: func(_ context.Context, task string, agents []agent.Agent) {
				teamFormedCalled = true
				assert.Equal(t, "write code", task)
			},
			OnAgentSelected: func(_ context.Context, _ string, entry PoolEntry, score float64) {
				selectedAgents = append(selectedAgents, entry.Agent.ID())
				assert.Greater(t, score, 0.0)
			},
		}

		allSelector := &mockSelector{
			selectFn: func(_ context.Context, _ string, candidates []PoolEntry) ([]PoolEntry, error) {
				return candidates, nil
			},
		}

		tb := NewTeamBuilder(pool,
			WithSelector(allSelector),
			WithHooks(hooks),
		)
		_, err := tb.Build(context.Background(), "write code")
		require.NoError(t, err)
		assert.True(t, teamFormedCalled)
		assert.Contains(t, selectedAgents, "coder")
	})

	t.Run("OnSelectionFailed fires on error", func(t *testing.T) {
		var failedCalled bool
		hooks := Hooks{
			OnSelectionFailed: func(_ context.Context, _ string, err error) {
				failedCalled = true
				assert.Error(t, err)
			},
		}

		errSelector := &mockSelector{
			selectFn: func(_ context.Context, _ string, _ []PoolEntry) ([]PoolEntry, error) {
				return nil, errors.New("fail")
			},
		}

		tb := NewTeamBuilder(pool,
			WithSelector(errSelector),
			WithHooks(hooks),
		)
		_, err := tb.Build(context.Background(), "task")
		require.Error(t, err)
		assert.True(t, failedCalled)
	})

	t.Run("OnSelectionFailed fires on empty result", func(t *testing.T) {
		var failedCalled bool
		hooks := Hooks{
			OnSelectionFailed: func(_ context.Context, _ string, err error) {
				failedCalled = true
			},
		}

		emptySelector := &mockSelector{
			selectFn: func(_ context.Context, _ string, _ []PoolEntry) ([]PoolEntry, error) {
				return nil, nil
			},
		}

		tb := NewTeamBuilder(pool,
			WithSelector(emptySelector),
			WithHooks(hooks),
		)
		_, _ = tb.Build(context.Background(), "task")
		assert.True(t, failedCalled)
	})
}

// --- Hooks tests ---

func TestComposeHooks(t *testing.T) {
	var order []string

	h1 := Hooks{
		OnTeamFormed: func(_ context.Context, _ string, _ []agent.Agent) {
			order = append(order, "h1")
		},
	}
	h2 := Hooks{
		OnTeamFormed: func(_ context.Context, _ string, _ []agent.Agent) {
			order = append(order, "h2")
		},
	}

	composed := ComposeHooks(h1, h2)
	require.NotNil(t, composed.OnTeamFormed)
	composed.OnTeamFormed(context.Background(), "task", nil)
	assert.Equal(t, []string{"h1", "h2"}, order)
}

func TestComposeHooks_NilFields(t *testing.T) {
	h1 := Hooks{} // All nil.
	h2 := Hooks{
		OnAgentSelected: func(_ context.Context, _ string, _ PoolEntry, _ float64) {},
	}

	composed := ComposeHooks(h1, h2)
	assert.Nil(t, composed.OnTeamFormed)
	assert.Nil(t, composed.OnSelectionFailed)
	assert.NotNil(t, composed.OnAgentSelected)
}

// --- Helper tests ---

func TestTokenize(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"hello world", []string{"hello", "world"}},
		{"a b c", []string{}}, // All too short.
		{"Write GOLANG tests!", []string{"write", "golang", "tests"}},
		{"", []string{}},
	}
	for _, tt := range tests {
		got := tokenize(tt.input)
		assert.Equal(t, tt.want, got, "tokenize(%q)", tt.input)
	}
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "hello", truncate("hello", 10))
	assert.Equal(t, "hel...", truncate("hello world", 3))
	assert.Equal(t, "", truncate("", 5))
}

// --- ScoredSelector tests ---

func TestKeywordSelector_SelectScored(t *testing.T) {
	coder := PoolEntry{
		Agent:        newMockAgent("coder", "engineer", "write golang code and tests"),
		Capabilities: []string{"golang", "testing"},
		Metrics:      NewAgentMetrics(),
	}
	writer := PoolEntry{
		Agent:        newMockAgent("writer", "tech writer", "write documentation"),
		Capabilities: []string{"documentation"},
		Metrics:      NewAgentMetrics(),
	}

	sel := NewKeywordSelector()
	scored, err := sel.SelectScored(context.Background(), "write golang tests", []PoolEntry{writer, coder})
	require.NoError(t, err)
	require.NotEmpty(t, scored)
	// Coder should rank first with a concrete normalized score in (0, 1].
	assert.Equal(t, "coder", scored[0].Entry.Agent.ID())
	assert.Greater(t, scored[0].Score, 0.0)
	assert.LessOrEqual(t, scored[0].Score, 1.0)
	// Scores must be ordered descending.
	for i := 1; i < len(scored); i++ {
		assert.GreaterOrEqual(t, scored[i-1].Score, scored[i].Score)
	}
}

func TestKeywordSelector_SelectScored_EmptyTask(t *testing.T) {
	sel := NewKeywordSelector()
	scored, err := sel.SelectScored(context.Background(), "", []PoolEntry{
		{Agent: newMockAgent("a", "r", "g"), Capabilities: []string{"x"}, Metrics: NewAgentMetrics()},
	})
	require.NoError(t, err)
	assert.Nil(t, scored)
}

func TestLLMSelector_SelectScored(t *testing.T) {
	candidates := []PoolEntry{
		{Agent: newMockAgent("coder", "engineer", "write code"), Capabilities: []string{"golang"}, Metrics: NewAgentMetrics()},
		{Agent: newMockAgent("writer", "writer", "docs"), Capabilities: []string{"docs"}, Metrics: NewAgentMetrics()},
	}

	resp := selectionResponse{
		Selections: []agentSelection{
			{AgentID: "coder", Score: 0.87, Reasoning: "best"},
			{AgentID: "writer", Score: 0.42, Reasoning: "ok"},
		},
	}
	respJSON, _ := json.Marshal(resp)

	model := &mockChatModel{
		generateFn: func(_ context.Context, _ []schema.Message, _ ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage(string(respJSON)), nil
		},
	}

	sel := NewLLMSelector(model)
	scored, err := sel.SelectScored(context.Background(), "write golang tests", candidates)
	require.NoError(t, err)
	require.Len(t, scored, 2)
	// Real LLM scores must be forwarded, not positional estimates.
	assert.Equal(t, "coder", scored[0].Entry.Agent.ID())
	assert.InDelta(t, 0.87, scored[0].Score, 0.001)
	assert.Equal(t, "writer", scored[1].Entry.Agent.ID())
	assert.InDelta(t, 0.42, scored[1].Score, 0.001)
}

func TestLLMSelector_SelectScored_ClampsOutOfRange(t *testing.T) {
	candidates := []PoolEntry{
		{Agent: newMockAgent("a", "r", "g"), Capabilities: nil, Metrics: NewAgentMetrics()},
		{Agent: newMockAgent("b", "r", "g"), Capabilities: nil, Metrics: NewAgentMetrics()},
	}
	resp := selectionResponse{
		Selections: []agentSelection{
			{AgentID: "a", Score: 1.5},
			{AgentID: "b", Score: 0.9},
		},
	}
	respJSON, _ := json.Marshal(resp)
	model := &mockChatModel{
		generateFn: func(_ context.Context, _ []schema.Message, _ ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage(string(respJSON)), nil
		},
	}
	sel := NewLLMSelector(model)
	scored, err := sel.SelectScored(context.Background(), "task", candidates)
	require.NoError(t, err)
	require.Len(t, scored, 2)
	assert.LessOrEqual(t, scored[0].Score, 1.0)
	assert.GreaterOrEqual(t, scored[0].Score, 0.0)
}

func TestTeamBuilder_ForwardsRealScoresFromScoredSelector(t *testing.T) {
	pool := NewAgentPool()
	a := newMockAgent("coder", "engineer", "write code")
	b := newMockAgent("writer", "writer", "docs")
	require.NoError(t, pool.Register(a, "code"))
	require.NoError(t, pool.Register(b, "docs"))

	// Scored selector returns non-positional, non-decreasing scores.
	scorer := &mockScoredSelector{
		scoredFn: func(_ context.Context, _ string, candidates []PoolEntry) ([]ScoredPoolEntry, error) {
			return []ScoredPoolEntry{
				{Entry: candidates[0], Score: 0.73},
				{Entry: candidates[1], Score: 0.55},
			}, nil
		},
	}

	type scored struct {
		id    string
		score float64
	}
	var observed []scored

	hooks := Hooks{
		OnAgentSelected: func(_ context.Context, _ string, entry PoolEntry, score float64) {
			observed = append(observed, scored{id: entry.Agent.ID(), score: score})
		},
	}

	tb := NewTeamBuilder(pool,
		WithSelector(scorer),
		WithHooks(hooks),
	)
	_, err := tb.Build(context.Background(), "task")
	require.NoError(t, err)
	require.Len(t, observed, 2)
	// Exact scores must be forwarded, not 1.0/0.9 positional approximation.
	assert.InDelta(t, 0.73, observed[0].score, 0.001)
	assert.InDelta(t, 0.55, observed[1].score, 0.001)
}

func TestTeamBuilder_FallsBackToPositionalScoresForPlainSelector(t *testing.T) {
	pool := NewAgentPool()
	a := newMockAgent("a", "r", "g")
	b := newMockAgent("b", "r", "g")
	require.NoError(t, pool.Register(a, "x"))
	require.NoError(t, pool.Register(b, "y"))

	plain := &mockSelector{
		selectFn: func(_ context.Context, _ string, candidates []PoolEntry) ([]PoolEntry, error) {
			return candidates, nil
		},
	}

	var scores []float64
	hooks := Hooks{
		OnAgentSelected: func(_ context.Context, _ string, _ PoolEntry, score float64) {
			scores = append(scores, score)
		},
	}

	tb := NewTeamBuilder(pool, WithSelector(plain), WithHooks(hooks))
	_, err := tb.Build(context.Background(), "task")
	require.NoError(t, err)
	require.Len(t, scores, 2)
	// Positional fallback: first is 1.0, each subsequent drops by 0.1.
	assert.InDelta(t, 1.0, scores[0], 0.001)
	assert.InDelta(t, 0.9, scores[1], 0.001)
}

// --- mapSelectionsToEntries tests ---

func TestMapSelectionsToEntries(t *testing.T) {
	candidates := []PoolEntry{
		{Agent: newMockAgent("a", "r", "g"), Metrics: NewAgentMetrics()},
		{Agent: newMockAgent("b", "r", "g"), Metrics: NewAgentMetrics()},
	}

	t.Run("orders by score", func(t *testing.T) {
		selections := []agentSelection{
			{AgentID: "b", Score: 0.9},
			{AgentID: "a", Score: 0.5},
		}
		result, err := mapSelectionsToEntries(selections, candidates)
		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Equal(t, "b", result[0].Agent.ID())
		assert.Equal(t, "a", result[1].Agent.ID())
	})

	t.Run("filters low scores", func(t *testing.T) {
		selections := []agentSelection{
			{AgentID: "a", Score: 0.2},
			{AgentID: "b", Score: 0.1},
		}
		result, err := mapSelectionsToEntries(selections, candidates)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("ignores unknown agent IDs", func(t *testing.T) {
		selections := []agentSelection{
			{AgentID: "unknown", Score: 0.9},
		}
		result, err := mapSelectionsToEntries(selections, candidates)
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

// --- buildSelectionPrompt test ---

func TestBuildSelectionPrompt(t *testing.T) {
	candidates := []PoolEntry{
		{
			Agent:        newMockAgent("coder", "engineer", "write code"),
			Capabilities: []string{"golang", "testing"},
			Metrics:      NewAgentMetrics(),
		},
	}
	prompt := buildSelectionPrompt("write tests", candidates)
	assert.Contains(t, prompt, "## Task")
	assert.Contains(t, prompt, "write tests")
	assert.Contains(t, prompt, "coder")
	assert.Contains(t, prompt, "golang, testing")
}

// --- Benchmark ---

func BenchmarkKeywordSelector_Select(b *testing.B) {
	// Build a pool of 100 agents.
	candidates := make([]PoolEntry, 100)
	for i := 0; i < 100; i++ {
		candidates[i] = PoolEntry{
			Agent:        newMockAgent(fmt.Sprintf("agent-%d", i), "role", fmt.Sprintf("goal for area %d", i)),
			Capabilities: []string{fmt.Sprintf("skill-%d", i), fmt.Sprintf("area-%d", i%10)},
			Metrics:      NewAgentMetrics(),
		}
	}

	sel := NewKeywordSelector()
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sel.Select(ctx, "find agent with skill-50 in area-5", candidates)
	}
}

func BenchmarkAgentPool_RegisterAndList(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		pool := NewAgentPool()
		for j := 0; j < 50; j++ {
			a := newMockAgent(fmt.Sprintf("a-%d", j), "r", "g")
			pool.Register(a, "cap")
		}
		pool.List()
	}
}
