package cognitive

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock Agent ---

type mockAgent struct {
	id      string
	result  string
	err     error
	events  []agent.Event
	calls   int
	mu      sync.Mutex
	delay   time.Duration
	persona agent.Persona
}

func newMockAgent(id, result string) *mockAgent {
	return &mockAgent{
		id:     id,
		result: result,
		persona: agent.Persona{
			Role: id,
			Goal: "mock agent for testing",
		},
	}
}

func (m *mockAgent) ID() string              { return m.id }
func (m *mockAgent) Persona() agent.Persona  { return m.persona }
func (m *mockAgent) Tools() []tool.Tool      { return nil }
func (m *mockAgent) Children() []agent.Agent { return nil }
func (m *mockAgent) CallCount() int          { m.mu.Lock(); defer m.mu.Unlock(); return m.calls }

func (m *mockAgent) Invoke(ctx context.Context, input string, opts ...agent.Option) (string, error) {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()

	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	return m.result, m.err
}

func (m *mockAgent) Stream(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		m.mu.Lock()
		m.calls++
		m.mu.Unlock()

		if m.err != nil {
			yield(agent.Event{Type: agent.EventError, AgentID: m.id}, m.err)
			return
		}

		if len(m.events) > 0 {
			for _, e := range m.events {
				if !yield(e, nil) {
					return
				}
			}
			return
		}

		// Default: emit text chunks
		words := strings.Fields(m.result)
		for _, w := range words {
			select {
			case <-ctx.Done():
				yield(agent.Event{}, ctx.Err())
				return
			default:
			}
			if !yield(agent.Event{
				Type:    agent.EventText,
				AgentID: m.id,
				Text:    w + " ",
			}, nil) {
				return
			}
		}
		yield(agent.Event{Type: agent.EventDone, AgentID: m.id}, nil)
	}
}

// --- Mock Scorer ---

type mockScorer struct {
	score ComplexityScore
	err   error
}

func (s *mockScorer) Score(_ context.Context, _ string) (ComplexityScore, error) {
	return s.score, s.err
}

// --- Tests ---

func TestComplexityLevel_String(t *testing.T) {
	tests := []struct {
		level ComplexityLevel
		want  string
	}{
		{Simple, "simple"},
		{Moderate, "moderate"},
		{Complex, "complex"},
		{ComplexityLevel(99), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.level.String())
		})
	}
}

func TestHeuristicScorer(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantLevel   ComplexityLevel
		wantMinConf float64
	}{
		{
			name:        "simple greeting",
			input:       "hello",
			wantLevel:   Simple,
			wantMinConf: 0.7,
		},
		{
			name:        "simple question",
			input:       "What is the capital of France?",
			wantLevel:   Simple,
			wantMinConf: 0.7,
		},
		{
			name:        "complex keyword - step by step",
			input:       "Explain step by step how photosynthesis works",
			wantLevel:   Moderate,
			wantMinConf: 0.5,
		},
		{
			name:        "complex keyword - analyze",
			input:       "Analyze the economic implications of this policy",
			wantLevel:   Moderate,
			wantMinConf: 0.5,
		},
		{
			name:        "math expression",
			input:       "Solve 2 + 3 * 5 = x",
			wantLevel:   Moderate,
			wantMinConf: 0.5,
		},
		{
			name:        "multiple questions",
			input:       "What is X? How does Y work? Why is Z important?",
			wantLevel:   Moderate,
			wantMinConf: 0.5,
		},
		{
			name:        "long complex input",
			input:       strings.Repeat("Analyze the comprehensive trade-offs ", 20),
			wantLevel:   Complex,
			wantMinConf: 0.6,
		},
		{
			name:        "empty input",
			input:       "",
			wantLevel:   Simple,
			wantMinConf: 0.7,
		},
	}

	scorer := NewHeuristicScorer()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, err := scorer.Score(context.Background(), tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.wantLevel, score.Level, "level mismatch")
			assert.GreaterOrEqual(t, score.Confidence, tt.wantMinConf, "confidence too low")
			assert.NotEmpty(t, score.Reason)
		})
	}
}

func TestHeuristicScorer_CustomThresholds(t *testing.T) {
	scorer := NewHeuristicScorer(
		WithModerateThreshold(5),
		WithComplexThreshold(10),
	)

	// A short input that would normally be simple should be moderate with low threshold
	score, err := scorer.Score(context.Background(), "What are the key differences between X and Y?")
	require.NoError(t, err)
	// With threshold of 5, this should at least be moderate due to token count
	assert.GreaterOrEqual(t, int(score.Level), int(Moderate))
}

func TestHeuristicScorer_InvalidThresholds(t *testing.T) {
	scorer := NewHeuristicScorer(
		WithModerateThreshold(-1),
		WithComplexThreshold(0),
	)
	// Negative/zero thresholds should be ignored, keeping defaults
	assert.Equal(t, 30, scorer.moderateThreshold)
	assert.Equal(t, 100, scorer.complexThreshold)
}

func TestDualProcessAgent_New(t *testing.T) {
	s1 := newMockAgent("s1", "fast")
	s2 := newMockAgent("s2", "slow")

	t.Run("success", func(t *testing.T) {
		a, err := New("test", s1, s2)
		require.NoError(t, err)
		assert.Equal(t, "test", a.ID())
		assert.Equal(t, 2, len(a.Children()))
		assert.Nil(t, a.Tools())
		assert.NotEmpty(t, a.Persona().Role)
	})

	t.Run("nil s1", func(t *testing.T) {
		_, err := New("test", nil, s2)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system 1")
	})

	t.Run("nil s2", func(t *testing.T) {
		_, err := New("test", s1, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system 2")
	})
}

func TestDualProcessAgent_InvokeSimple(t *testing.T) {
	s1 := newMockAgent("s1", "fast answer")
	s2 := newMockAgent("s2", "slow answer")

	scorer := &mockScorer{score: ComplexityScore{
		Level:      Simple,
		Confidence: 0.9,
	}}

	a, err := New("test", s1, s2, WithScorer(scorer))
	require.NoError(t, err)

	result, err := a.Invoke(context.Background(), "hello")
	require.NoError(t, err)
	assert.Equal(t, "fast answer", result)
	assert.Equal(t, 1, s1.CallCount())
	assert.Equal(t, 0, s2.CallCount())

	// Check metrics
	assert.Equal(t, int64(1), a.Metrics().S1Count())
	assert.Equal(t, int64(0), a.Metrics().S2Count())
}

func TestDualProcessAgent_InvokeComplex(t *testing.T) {
	s1 := newMockAgent("s1", "fast answer")
	s2 := newMockAgent("s2", "deliberate answer")

	scorer := &mockScorer{score: ComplexityScore{
		Level:      Complex,
		Confidence: 0.85,
	}}

	a, err := New("test", s1, s2, WithScorer(scorer))
	require.NoError(t, err)

	result, err := a.Invoke(context.Background(), "analyze the trade-offs")
	require.NoError(t, err)
	assert.Equal(t, "deliberate answer", result)
	assert.Equal(t, 0, s1.CallCount()) // S1 never called for confident complex
	assert.Equal(t, 1, s2.CallCount())

	assert.Equal(t, int64(0), a.Metrics().S1Count())
	assert.Equal(t, int64(1), a.Metrics().S2Count())
}

func TestDualProcessAgent_InvokeEscalation(t *testing.T) {
	s1 := newMockAgent("s1", "shallow answer")
	s2 := newMockAgent("s2", "deep answer")

	// Moderate with low confidence triggers escalation
	scorer := &mockScorer{score: ComplexityScore{
		Level:      Moderate,
		Confidence: 0.5,
	}}

	var escalated bool
	hooks := Hooks{
		OnEscalated: func(_ context.Context, _ string, s1Output string, reason string) {
			escalated = true
			assert.Equal(t, "shallow answer", s1Output)
			assert.Contains(t, reason, "below threshold")
		},
	}

	a, err := New("test", s1, s2, WithScorer(scorer), WithCognitiveHooks(hooks))
	require.NoError(t, err)

	result, err := a.Invoke(context.Background(), "moderate question")
	require.NoError(t, err)
	assert.Equal(t, "deep answer", result)
	assert.True(t, escalated)
	assert.Equal(t, 1, s1.CallCount())
	assert.Equal(t, 1, s2.CallCount())
	assert.Equal(t, int64(1), a.Metrics().EscalationCount())
}

func TestDualProcessAgent_InvokeS1Error(t *testing.T) {
	s1 := newMockAgent("s1", "")
	s1.err = errors.New("s1 failed")
	s2 := newMockAgent("s2", "fallback answer")

	scorer := &mockScorer{score: ComplexityScore{
		Level:      Simple,
		Confidence: 0.9,
	}}

	a, err := New("test", s1, s2, WithScorer(scorer))
	require.NoError(t, err)

	result, err := a.Invoke(context.Background(), "hello")
	require.NoError(t, err)
	assert.Equal(t, "fallback answer", result)
	assert.Equal(t, int64(1), a.Metrics().EscalationCount())
}

func TestDualProcessAgent_InvokeScorerError(t *testing.T) {
	s1 := newMockAgent("s1", "fast")
	s2 := newMockAgent("s2", "slow")

	scorer := &mockScorer{err: errors.New("scorer broken")}

	a, err := New("test", s1, s2, WithScorer(scorer))
	require.NoError(t, err)

	_, err = a.Invoke(context.Background(), "hello")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scoring failed")
}

func TestDualProcessAgent_StreamSimple(t *testing.T) {
	s1 := newMockAgent("s1", "fast stream")
	s2 := newMockAgent("s2", "slow stream")

	scorer := &mockScorer{score: ComplexityScore{
		Level:      Simple,
		Confidence: 0.9,
	}}

	a, err := New("test", s1, s2, WithScorer(scorer))
	require.NoError(t, err)

	var texts []string
	for event, err := range a.Stream(context.Background(), "hello") {
		require.NoError(t, err)
		if event.Type == agent.EventText && event.Text != "" {
			texts = append(texts, event.Text)
		}
	}

	assert.Equal(t, 1, s1.CallCount())
	assert.Equal(t, 0, s2.CallCount())
	assert.Equal(t, int64(1), a.Metrics().S1Count())
}

func TestDualProcessAgent_StreamComplex(t *testing.T) {
	s1 := newMockAgent("s1", "fast")
	s2 := newMockAgent("s2", "deliberate stream")

	scorer := &mockScorer{score: ComplexityScore{
		Level:      Complex,
		Confidence: 0.9,
	}}

	a, err := New("test", s1, s2, WithScorer(scorer))
	require.NoError(t, err)

	var eventCount int
	for _, err := range a.Stream(context.Background(), "complex question") {
		require.NoError(t, err)
		eventCount++
	}

	assert.Greater(t, eventCount, 0)
	assert.Equal(t, 0, s1.CallCount())
	assert.Equal(t, 1, s2.CallCount())
	assert.Equal(t, int64(1), a.Metrics().S2Count())
}

func TestDualProcessAgent_StreamContextCancellation(t *testing.T) {
	s1 := newMockAgent("s1", "word1 word2 word3 word4 word5")

	scorer := &mockScorer{score: ComplexityScore{
		Level:      Simple,
		Confidence: 0.9,
	}}

	a, err := New("test", s1, newMockAgent("s2", ""), WithScorer(scorer))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	count := 0
	for _, err := range a.Stream(ctx, "hello") {
		count++
		if count == 2 {
			cancel()
		}
		if err != nil {
			assert.ErrorIs(t, err, context.Canceled)
			break
		}
	}
	assert.LessOrEqual(t, count, 4) // should stop near cancellation
}

func TestDualProcessAgent_StreamScorerError(t *testing.T) {
	s1 := newMockAgent("s1", "fast")
	s2 := newMockAgent("s2", "slow")

	scorer := &mockScorer{err: errors.New("scorer broken")}

	a, err := New("test", s1, s2, WithScorer(scorer))
	require.NoError(t, err)

	var gotErr error
	for _, err := range a.Stream(context.Background(), "hello") {
		if err != nil {
			gotErr = err
			break
		}
	}
	require.Error(t, gotErr)
	assert.Contains(t, gotErr.Error(), "scoring failed")
}

func TestDualProcessAgent_Hooks(t *testing.T) {
	s1 := newMockAgent("s1", "fast")
	s2 := newMockAgent("s2", "slow")

	scorer := &mockScorer{score: ComplexityScore{
		Level:      Simple,
		Confidence: 0.9,
	}}

	var routedCalled, completedCalled bool
	var routedTarget, completedTier string

	hooks := Hooks{
		OnRouted: func(_ context.Context, _ string, _ ComplexityLevel, target string) {
			routedCalled = true
			routedTarget = target
		},
		OnCompleted: func(_ context.Context, tier string, latency time.Duration) {
			completedCalled = true
			completedTier = tier
			assert.Greater(t, latency, time.Duration(0))
		},
	}

	a, err := New("test", s1, s2, WithScorer(scorer), WithCognitiveHooks(hooks))
	require.NoError(t, err)

	_, err = a.Invoke(context.Background(), "hello")
	require.NoError(t, err)

	assert.True(t, routedCalled)
	assert.Equal(t, "s1", routedTarget)
	assert.True(t, completedCalled)
	assert.Equal(t, "s1", completedTier)
}

func TestDualProcessAgent_CustomThreshold(t *testing.T) {
	s1 := newMockAgent("s1", "fast")
	s2 := newMockAgent("s2", "slow")

	// Moderate with 0.6 confidence
	scorer := &mockScorer{score: ComplexityScore{
		Level:      Moderate,
		Confidence: 0.6,
	}}

	// Low threshold - should NOT escalate
	a1, err := New("test", s1, s2, WithScorer(scorer), WithThreshold(0.5))
	require.NoError(t, err)
	result, err := a1.Invoke(context.Background(), "question")
	require.NoError(t, err)
	assert.Equal(t, "fast", result)

	// Reset
	s1 = newMockAgent("s1", "fast")
	s2 = newMockAgent("s2", "slow")

	// High threshold - should escalate
	a2, err := New("test", s1, s2, WithScorer(scorer), WithThreshold(0.9))
	require.NoError(t, err)
	result, err = a2.Invoke(context.Background(), "question")
	require.NoError(t, err)
	assert.Equal(t, "slow", result)
}

func TestComposeHooks(t *testing.T) {
	var order []int

	h1 := Hooks{
		OnRouted: func(_ context.Context, _ string, _ ComplexityLevel, _ string) {
			order = append(order, 1)
		},
	}
	h2 := Hooks{
		OnRouted: func(_ context.Context, _ string, _ ComplexityLevel, _ string) {
			order = append(order, 2)
		},
	}

	composed := ComposeHooks(h1, h2)
	composed.OnRouted(context.Background(), "", Simple, "s1")
	assert.Equal(t, []int{1, 2}, order)
}

func TestComposeHooks_NilFields(t *testing.T) {
	// Composing hooks with nil fields should not panic
	h1 := Hooks{}
	h2 := Hooks{
		OnCompleted: func(_ context.Context, _ string, _ time.Duration) {},
	}

	composed := ComposeHooks(h1, h2)
	// Should not panic
	composed.OnRouted(context.Background(), "", Simple, "s1")
	composed.OnEscalated(context.Background(), "", "", "")
	composed.OnCompleted(context.Background(), "s1", time.Second)
}

// --- Metrics Tests ---

func TestRoutingMetrics(t *testing.T) {
	m := &RoutingMetrics{}

	m.RecordS1(100*time.Millisecond, 0.01)
	m.RecordS1(200*time.Millisecond, 0.02)
	m.RecordS2(500*time.Millisecond, 0.10)
	m.RecordEscalation()

	assert.Equal(t, int64(2), m.S1Count())
	assert.Equal(t, int64(1), m.S2Count())
	assert.Equal(t, int64(1), m.EscalationCount())
	assert.Equal(t, 300*time.Millisecond, m.S1TotalLatency())
	assert.Equal(t, 500*time.Millisecond, m.S2TotalLatency())
}

func TestRoutingMetrics_EscalationRate(t *testing.T) {
	m := &RoutingMetrics{}

	assert.Equal(t, 0.0, m.EscalationRate()) // no requests

	m.RecordS1(0, 0)
	m.RecordS1(0, 0)
	m.RecordS2(0, 0)
	m.RecordEscalation()

	rate := m.EscalationRate()
	assert.InDelta(t, 1.0/3.0, rate, 0.01)
}

func TestRoutingMetrics_CostSavings(t *testing.T) {
	m := &RoutingMetrics{}

	m.RecordS1(0, 0.01) // cheap S1
	m.RecordS1(0, 0.01)
	m.RecordS2(0, 0.10) // expensive S2

	// If all 3 went to S2 at 0.10 each = 0.30
	// Actual cost = 0.01 + 0.01 + 0.10 = 0.12
	// Savings = 0.30 - 0.12 = 0.18
	savings := m.CostSavings(0.10)
	assert.InDelta(t, 0.18, savings, 0.001)
}

func TestRoutingMetrics_CostSavings_NoRequests(t *testing.T) {
	m := &RoutingMetrics{}
	assert.Equal(t, 0.0, m.CostSavings(0.10))
}

func TestRoutingMetrics_Snapshot(t *testing.T) {
	m := &RoutingMetrics{}
	m.RecordS1(100*time.Millisecond, 0.05)
	m.RecordS2(200*time.Millisecond, 0.10)
	m.RecordEscalation()

	snap := m.Snapshot()
	assert.Equal(t, int64(1), snap.S1Count)
	assert.Equal(t, int64(1), snap.S2Count)
	assert.Equal(t, int64(1), snap.EscalationCount)
	assert.Equal(t, 100*time.Millisecond, snap.S1TotalLatency)
	assert.Equal(t, 200*time.Millisecond, snap.S2TotalLatency)
	assert.InDelta(t, 0.05, snap.S1TotalCost, 0.001)
	assert.InDelta(t, 0.10, snap.S2TotalCost, 0.001)
}

func TestRoutingMetrics_Concurrent(t *testing.T) {
	m := &RoutingMetrics{}
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(3)
		go func() { defer wg.Done(); m.RecordS1(time.Millisecond, 0.01) }()
		go func() { defer wg.Done(); m.RecordS2(time.Millisecond, 0.02) }()
		go func() { defer wg.Done(); m.RecordEscalation() }()
	}
	wg.Wait()

	assert.Equal(t, int64(100), m.S1Count())
	assert.Equal(t, int64(100), m.S2Count())
	assert.Equal(t, int64(100), m.EscalationCount())
}

// --- Registry Tests ---

func TestScorerRegistry(t *testing.T) {
	// "heuristic" is registered in init()
	scorers := ListScorers()
	assert.Contains(t, scorers, "heuristic")

	scorer, err := NewScorer("heuristic", ScorerConfig{})
	require.NoError(t, err)
	assert.NotNil(t, scorer)

	// Test unknown scorer
	_, err = NewScorer("nonexistent", ScorerConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not registered")
}

func TestScorerRegistry_Custom(t *testing.T) {
	RegisterScorer("test-custom", func(_ ScorerConfig) (ComplexityScorer, error) {
		return &mockScorer{score: ComplexityScore{Level: Complex}}, nil
	})

	scorer, err := NewScorer("test-custom", ScorerConfig{})
	require.NoError(t, err)

	score, err := scorer.Score(context.Background(), "anything")
	require.NoError(t, err)
	assert.Equal(t, Complex, score.Level)
}

// --- Estimate Tokens Tests ---

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"hi", 1},
		{"hello world", 3},
		{strings.Repeat("a", 400), 100},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("len=%d", len(tt.input)), func(t *testing.T) {
			got := estimateTokens(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- WithMetrics Option Test ---

func TestDualProcessAgent_WithMetrics(t *testing.T) {
	shared := &RoutingMetrics{}
	s1 := newMockAgent("s1", "fast")
	s2 := newMockAgent("s2", "slow")
	scorer := &mockScorer{score: ComplexityScore{Level: Simple, Confidence: 0.9}}

	a, err := New("test", s1, s2, WithScorer(scorer), WithMetrics(shared))
	require.NoError(t, err)

	_, err = a.Invoke(context.Background(), "hello")
	require.NoError(t, err)

	assert.Equal(t, int64(1), shared.S1Count())
	assert.Same(t, shared, a.Metrics())
}

// --- WithScorer nil test ---

func TestDualProcessAgent_WithNilScorer(t *testing.T) {
	s1 := newMockAgent("s1", "fast")
	s2 := newMockAgent("s2", "slow")

	a, err := New("test", s1, s2, WithScorer(nil))
	require.NoError(t, err)

	// Should use default HeuristicScorer
	result, err := a.Invoke(context.Background(), "hello")
	require.NoError(t, err)
	assert.Equal(t, "fast", result)
}

// --- WithThreshold boundary tests ---

func TestDualProcessAgent_WithThresholdBoundary(t *testing.T) {
	s1 := newMockAgent("s1", "fast")
	s2 := newMockAgent("s2", "slow")

	// Invalid thresholds are ignored
	a, err := New("test", s1, s2, WithThreshold(-0.1))
	require.NoError(t, err)
	assert.Equal(t, 0.7, a.threshold) // default preserved

	a, err = New("test", s1, s2, WithThreshold(1.5))
	require.NoError(t, err)
	assert.Equal(t, 0.7, a.threshold) // default preserved

	// Valid boundaries
	a, err = New("test", s1, s2, WithThreshold(0.0))
	require.NoError(t, err)
	assert.Equal(t, 0.0, a.threshold)

	a, err = New("test", s1, s2, WithThreshold(1.0))
	require.NoError(t, err)
	assert.Equal(t, 1.0, a.threshold)
}
