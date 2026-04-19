package metacognitive

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/runtime"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

func TestPlugin_CompileTimeCheck(t *testing.T) {
	var _ runtime.Plugin = (*Plugin)(nil)
}

func TestPlugin_Name(t *testing.T) {
	store := NewInMemoryStore()
	p := NewPlugin(store)
	assert.Equal(t, "metacognitive", p.Name())
}

func TestPlugin_Defaults(t *testing.T) {
	store := NewInMemoryStore()
	p := NewPlugin(store)
	assert.Equal(t, DefaultMaxHeuristics, p.maxHeuristics)
	assert.InDelta(t, DefaultEMAAlpha, p.emaAlpha, 0.001)
	assert.NotNil(t, p.extractor)
	assert.NotNil(t, p.monitor)
}

func TestPlugin_Options(t *testing.T) {
	store := NewInMemoryStore()
	ext := NewSimpleExtractor()

	var hooksCalled bool
	p := NewPlugin(store,
		WithExtractor(ext),
		WithMaxHeuristics(10),
		WithEMAAlpha(0.5),
		WithHooks(Hooks{
			OnSelfModelLoaded: func(m *SelfModel) { hooksCalled = true },
		}),
	)

	assert.Equal(t, 10, p.maxHeuristics)
	assert.InDelta(t, 0.5, p.emaAlpha, 0.001)

	// Trigger the hook via a BeforeTurn with saved model.
	ctx := context.Background()
	sess := runtime.NewSession("s1", "agent-1")
	m := NewSelfModel("agent-1")
	m.Heuristics = []Heuristic{{ID: "h1", Content: "test search", Utility: 0.5, TaskType: "search"}}
	require.NoError(t, store.Save(ctx, m))

	_, err := p.BeforeTurn(ctx, sess, schema.NewHumanMessage("search query"))
	require.NoError(t, err)
	assert.True(t, hooksCalled)
}

func TestPlugin_WithExtractorNil(t *testing.T) {
	store := NewInMemoryStore()
	p := NewPlugin(store, WithExtractor(nil))
	// Should keep the default extractor.
	assert.NotNil(t, p.extractor)
}

func TestPlugin_WithMaxHeuristicsInvalid(t *testing.T) {
	store := NewInMemoryStore()
	p := NewPlugin(store, WithMaxHeuristics(-1))
	assert.Equal(t, DefaultMaxHeuristics, p.maxHeuristics)

	p = NewPlugin(store, WithMaxHeuristics(0))
	assert.Equal(t, DefaultMaxHeuristics, p.maxHeuristics)
}

func TestPlugin_WithEMAAlphaInvalid(t *testing.T) {
	store := NewInMemoryStore()
	p := NewPlugin(store, WithEMAAlpha(0))
	assert.InDelta(t, DefaultEMAAlpha, p.emaAlpha, 0.001)

	p = NewPlugin(store, WithEMAAlpha(1.5))
	assert.InDelta(t, DefaultEMAAlpha, p.emaAlpha, 0.001)
}

func TestPlugin_BeforeTurn_NilSession(t *testing.T) {
	store := NewInMemoryStore()
	p := NewPlugin(store)

	msg, err := p.BeforeTurn(context.Background(), nil, schema.NewHumanMessage("hi"))
	require.NoError(t, err)
	assert.Equal(t, "human", string(msg.GetRole()))
}

func TestPlugin_BeforeTurn_EmptyAgentID(t *testing.T) {
	store := NewInMemoryStore()
	p := NewPlugin(store)
	sess := runtime.NewSession("s1", "")

	msg, err := p.BeforeTurn(context.Background(), sess, schema.NewHumanMessage("hi"))
	require.NoError(t, err)
	assert.Equal(t, "human", string(msg.GetRole()))
}

func TestPlugin_BeforeTurn_StoresContextInSession(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	// Seed the store with heuristics.
	m := NewSelfModel("agent-1")
	m.Heuristics = []Heuristic{
		{ID: "h1", Content: "Avoid: timeout in search", Source: "failure", TaskType: "search", Utility: 0.9},
	}
	require.NoError(t, store.Save(ctx, m))

	p := NewPlugin(store)
	sess := runtime.NewSession("s1", "agent-1")

	_, err := p.BeforeTurn(ctx, sess, schema.NewHumanMessage("search for documents"))
	require.NoError(t, err)

	// Check that metacognitive context was stored in session state.
	ctxStr, ok := sess.State["metacognitive.context"].(string)
	assert.True(t, ok, "metacognitive context should be stored in session")
	assert.Contains(t, ctxStr, "timeout")
}

func TestPlugin_BeforeTurn_NoHeuristics(t *testing.T) {
	store := NewInMemoryStore()
	p := NewPlugin(store)
	sess := runtime.NewSession("s1", "agent-1")

	msg, err := p.BeforeTurn(context.Background(), sess, schema.NewHumanMessage("hi"))
	require.NoError(t, err)
	assert.Equal(t, "human", string(msg.GetRole()))

	// No context stored since no heuristics found.
	_, ok := sess.State["metacognitive.context"]
	assert.False(t, ok)
}

func TestPlugin_AfterTurn_ExtractsAndSaves(t *testing.T) {
	store := NewInMemoryStore()

	var extractedHeuristics []Heuristic
	var mu sync.Mutex
	p := NewPlugin(store, WithHooks(Hooks{
		OnHeuristicExtracted: func(h Heuristic) {
			mu.Lock()
			defer mu.Unlock()
			extractedHeuristics = append(extractedHeuristics, h)
		},
	}))

	ctx := context.Background()
	sess := runtime.NewSession("s1", "agent-1")

	// Simulate a failed turn with events.
	events := []agent.Event{
		{Type: agent.EventError, Text: "API timeout occurred"},
	}

	evts, err := p.AfterTurn(ctx, sess, events)
	require.NoError(t, err)
	assert.Len(t, evts, 1, "events must be passed through unchanged")

	// Verify the model was saved.
	model, err := store.Load(ctx, "agent-1")
	require.NoError(t, err)
	assert.NotEmpty(t, model.Heuristics, "heuristics should be extracted and saved")

	mu.Lock()
	assert.NotEmpty(t, extractedHeuristics, "OnHeuristicExtracted hook should fire")
	mu.Unlock()
}

func TestPlugin_AfterTurn_UpdatesCapabilityScore(t *testing.T) {
	store := NewInMemoryStore()

	var updatedScores []CapabilityScore
	p := NewPlugin(store, WithHooks(Hooks{
		OnCapabilityUpdated: func(taskType string, score CapabilityScore) {
			updatedScores = append(updatedScores, score)
		},
	}))

	ctx := context.Background()
	sess := runtime.NewSession("s1", "agent-1")
	sess.State["metacognitive.task_type"] = "search"

	// First successful turn.
	_, err := p.AfterTurn(ctx, sess, []agent.Event{
		{Type: agent.EventText, Text: "success result"},
	})
	require.NoError(t, err)

	model, err := store.Load(ctx, "agent-1")
	require.NoError(t, err)
	score, ok := model.Capabilities["search"]
	require.True(t, ok)
	assert.Equal(t, 1, score.SampleCount)

	assert.NotEmpty(t, updatedScores)
}

func TestPlugin_AfterTurn_NilSession(t *testing.T) {
	store := NewInMemoryStore()
	p := NewPlugin(store)

	events := []agent.Event{{Type: agent.EventText, Text: "hi"}}
	evts, err := p.AfterTurn(context.Background(), nil, events)
	require.NoError(t, err)
	assert.Len(t, evts, 1)
}

func TestPlugin_OnError_Passthrough(t *testing.T) {
	store := NewInMemoryStore()
	p := NewPlugin(store)

	tests := []struct {
		name string
		err  error
	}{
		{"nil error", nil},
		{"non-nil error", assert.AnError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.OnError(context.Background(), tt.err)
			assert.Equal(t, tt.err, got)
		})
	}
}

func TestPlugin_CapabilityScore_EMA(t *testing.T) {
	store := NewInMemoryStore()
	p := NewPlugin(store, WithEMAAlpha(0.5))
	ctx := context.Background()

	model := NewSelfModel("agent-1")

	// First observation: success.
	p.updateCapabilityScore(model, MonitoringSignals{
		TaskType:     "search",
		Success:      true,
		TotalLatency: 100 * time.Millisecond,
	})
	assert.InDelta(t, 1.0, model.Capabilities["search"].SuccessRate, 0.001)
	assert.Equal(t, 1, model.Capabilities["search"].SampleCount)

	// Second observation: failure.
	p.updateCapabilityScore(model, MonitoringSignals{
		TaskType:     "search",
		Success:      false,
		TotalLatency: 200 * time.Millisecond,
	})
	// EMA: 1.0 + 0.5 * (0.0 - 1.0) = 0.5
	assert.InDelta(t, 0.5, model.Capabilities["search"].SuccessRate, 0.001)
	assert.Equal(t, 2, model.Capabilities["search"].SampleCount)

	// Third observation: success.
	p.updateCapabilityScore(model, MonitoringSignals{
		TaskType:     "search",
		Success:      true,
		TotalLatency: 50 * time.Millisecond,
	})
	// EMA: 0.5 + 0.5 * (1.0 - 0.5) = 0.75
	assert.InDelta(t, 0.75, model.Capabilities["search"].SuccessRate, 0.001)
	assert.Equal(t, 3, model.Capabilities["search"].SampleCount)

	_ = ctx // used in setup
}

func TestPlugin_CapabilityScore_DefaultTaskType(t *testing.T) {
	store := NewInMemoryStore()
	p := NewPlugin(store)

	model := NewSelfModel("agent-1")
	p.updateCapabilityScore(model, MonitoringSignals{
		Success:      true,
		TotalLatency: 100 * time.Millisecond,
	})

	_, ok := model.Capabilities["general"]
	assert.True(t, ok, "empty task type should default to 'general'")
}

func TestPlugin_Monitor(t *testing.T) {
	store := NewInMemoryStore()
	p := NewPlugin(store)
	m := p.Monitor()
	require.NotNil(t, m)
	assert.IsType(t, &Monitor{}, m)
}

func TestPlugin_EnrichSignalsFromEvents(t *testing.T) {
	tests := []struct {
		name        string
		signals     MonitoringSignals
		events      []agent.Event
		wantOutcome string
		wantSuccess bool
	}{
		{
			name:    "enriches from text events",
			signals: MonitoringSignals{Success: true},
			events: []agent.Event{
				{Type: agent.EventText, Text: "hello "},
				{Type: agent.EventText, Text: "world"},
			},
			wantOutcome: "hello world",
			wantSuccess: true,
		},
		{
			name:    "marks failure on error event",
			signals: MonitoringSignals{Success: true},
			events: []agent.Event{
				{Type: agent.EventError, Text: "failed"},
			},
			wantOutcome: "",
			wantSuccess: false,
		},
		{
			name:        "preserves existing outcome",
			signals:     MonitoringSignals{Outcome: "existing", Success: true},
			events:      []agent.Event{{Type: agent.EventText, Text: "new"}},
			wantOutcome: "existing",
			wantSuccess: true,
		},
		{
			name:    "collects tool calls from events",
			signals: MonitoringSignals{Success: true},
			events: []agent.Event{
				{Type: agent.EventToolCall, ToolCall: &schema.ToolCall{Name: "search"}},
			},
			wantOutcome: "",
			wantSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := enrichSignalsFromEvents(tt.signals, tt.events)
			assert.Equal(t, tt.wantOutcome, result.Outcome)
			assert.Equal(t, tt.wantSuccess, result.Success)
		})
	}
}

func TestPlugin_BuildHeuristicContext(t *testing.T) {
	heuristics := []Heuristic{
		{Source: "failure", Content: "Avoid: timeout errors"},
		{Source: "success", Content: "Prefer: caching"},
	}

	ctx := buildHeuristicContext(heuristics)
	assert.Contains(t, ctx, "[Metacognitive Heuristics]")
	assert.Contains(t, ctx, "1. [failure] Avoid: timeout errors")
	assert.Contains(t, ctx, "2. [success] Prefer: caching")
}

func TestPlugin_ExtractTextFromMessage(t *testing.T) {
	tests := []struct {
		name string
		msg  schema.Message
		want string
	}{
		{
			name: "human message",
			msg:  schema.NewHumanMessage("hello world"),
			want: "hello world",
		},
		{
			name: "nil message",
			msg:  nil,
			want: "",
		},
		{
			name: "system message",
			msg:  schema.NewSystemMessage("system prompt"),
			want: "system prompt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTextFromMessage(tt.msg)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPlugin_FullRoundTrip(t *testing.T) {
	store := NewInMemoryStore()
	p := NewPlugin(store, WithMaxHeuristics(3))
	ctx := context.Background()
	sess := runtime.NewSession("s1", "agent-1")

	// Turn 1: failure, should extract heuristics.
	_, err := p.BeforeTurn(ctx, sess, schema.NewHumanMessage("search for data"))
	require.NoError(t, err)

	p.Monitor().Reset()
	hooks := p.Monitor().Hooks()
	_ = hooks.OnStart(ctx, "search for data")
	_ = hooks.OnError(ctx, assert.AnError)
	hooks.OnEnd(ctx, "", assert.AnError)

	_, err = p.AfterTurn(ctx, sess, []agent.Event{
		{Type: agent.EventError, Text: "search failed"},
	})
	require.NoError(t, err)

	// Verify heuristics were saved.
	model, err := store.Load(ctx, "agent-1")
	require.NoError(t, err)
	assert.NotEmpty(t, model.Heuristics)

	// Turn 2: should retrieve heuristics from turn 1.
	_, err = p.BeforeTurn(ctx, sess, schema.NewHumanMessage("search again"))
	require.NoError(t, err)

	ctxStr, ok := sess.State["metacognitive.context"].(string)
	assert.True(t, ok, "should have metacognitive context from previous heuristics")
	assert.Contains(t, ctxStr, "Heuristics")
}
