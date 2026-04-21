package debate

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/schema"
	"github.com/lookatitude/beluga-ai/v2/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAgent is a minimal agent for testing.
type mockAgent struct {
	id       string
	invokeFn func(ctx context.Context, input string) (string, error)
}

var _ agent.Agent = (*mockAgent)(nil)

func (m *mockAgent) ID() string              { return m.id }
func (m *mockAgent) Persona() agent.Persona  { return agent.Persona{Role: m.id} }
func (m *mockAgent) Tools() []tool.Tool      { return nil }
func (m *mockAgent) Children() []agent.Agent { return nil }
func (m *mockAgent) Invoke(ctx context.Context, input string, opts ...agent.Option) (string, error) {
	if m.invokeFn != nil {
		return m.invokeFn(ctx, input)
	}
	return fmt.Sprintf("response from %s", m.id), nil
}
func (m *mockAgent) Stream(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		text, err := m.Invoke(ctx, input, opts...)
		if err != nil {
			yield(agent.Event{}, err)
			return
		}
		yield(agent.Event{Type: agent.EventText, Text: text, AgentID: m.id}, nil)
	}
}

func newMockAgent(id string) *mockAgent {
	return &mockAgent{id: id}
}

func newMockAgentWithFn(id string, fn func(ctx context.Context, input string) (string, error)) *mockAgent {
	return &mockAgent{id: id, invokeFn: fn}
}

// --- DebateState tests ---

func TestDebateState_LastRound(t *testing.T) {
	tests := []struct {
		name  string
		state DebateState
		wantN int
	}{
		{
			name:  "empty rounds",
			state: DebateState{},
			wantN: 0,
		},
		{
			name: "one round",
			state: DebateState{
				Rounds: []Round{{Number: 1}},
			},
			wantN: 1,
		},
		{
			name: "multiple rounds returns last",
			state: DebateState{
				Rounds: []Round{{Number: 1}, {Number: 2}, {Number: 3}},
			},
			wantN: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.state.LastRound()
			assert.Equal(t, tt.wantN, got.Number)
		})
	}
}

// --- ComposeHooks tests ---

func TestComposeHooks(t *testing.T) {
	var calls []string

	h1 := Hooks{
		BeforeRound: func(_ context.Context, _ DebateState) error {
			calls = append(calls, "h1-before")
			return nil
		},
	}
	h2 := Hooks{
		BeforeRound: func(_ context.Context, _ DebateState) error {
			calls = append(calls, "h2-before")
			return nil
		},
	}

	composed := ComposeHooks(h1, h2)
	require.NotNil(t, composed.BeforeRound)

	err := composed.BeforeRound(context.Background(), DebateState{})
	require.NoError(t, err)
	assert.Equal(t, []string{"h1-before", "h2-before"}, calls)
}

func TestComposeHooks_ErrorShortCircuits(t *testing.T) {
	sentinel := errors.New("hook error")
	var called bool

	h1 := Hooks{
		BeforeRound: func(_ context.Context, _ DebateState) error {
			return sentinel
		},
	}
	h2 := Hooks{
		BeforeRound: func(_ context.Context, _ DebateState) error {
			called = true
			return nil
		},
	}

	composed := ComposeHooks(h1, h2)
	err := composed.BeforeRound(context.Background(), DebateState{})
	assert.ErrorIs(t, err, sentinel)
	assert.False(t, called, "second hook should not be called after error")
}

// --- Protocol registry tests ---

func TestProtocolRegistry(t *testing.T) {
	protocols := ListProtocols()
	assert.Contains(t, protocols, "roundrobin")
	assert.Contains(t, protocols, "adversarial")
	assert.Contains(t, protocols, "judged")

	p, err := NewProtocol("roundrobin", nil)
	require.NoError(t, err)
	require.NotNil(t, p)

	_, err = NewProtocol("nonexistent", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown protocol")
}

// --- Detector registry tests ---

func TestDetectorRegistry(t *testing.T) {
	detectors := ListDetectors()
	assert.Contains(t, detectors, "stability")
	assert.Contains(t, detectors, "agreement")
	assert.Contains(t, detectors, "maxrounds")

	d, err := NewDetector("stability", map[string]any{"threshold": 0.9})
	require.NoError(t, err)
	require.NotNil(t, d)

	_, err = NewDetector("nonexistent", nil)
	require.Error(t, err)
}

// --- RoundRobinProtocol tests ---

func TestRoundRobinProtocol(t *testing.T) {
	p := NewRoundRobinProtocol()
	ctx := context.Background()

	t.Run("basic prompts", func(t *testing.T) {
		state := DebateState{
			Topic:        "test topic",
			AgentIDs:     []string{"a1", "a2"},
			CurrentRound: 0,
		}
		prompts, err := p.NextRound(ctx, state)
		require.NoError(t, err)
		assert.Len(t, prompts, 2)
		assert.Contains(t, prompts["a1"], "test topic")
		assert.Contains(t, prompts["a2"], "Round 1")
	})

	t.Run("with history", func(t *testing.T) {
		state := DebateState{
			Topic:        "test topic",
			AgentIDs:     []string{"a1", "a2"},
			CurrentRound: 1,
			Rounds: []Round{{
				Number: 1,
				Contributions: []Contribution{
					{AgentID: "a1", Content: "first response"},
				},
			}},
		}
		prompts, err := p.NextRound(ctx, state)
		require.NoError(t, err)
		assert.Contains(t, prompts["a1"], "first response")
	})

	t.Run("no agents", func(t *testing.T) {
		state := DebateState{Topic: "test"}
		_, err := p.NextRound(ctx, state)
		require.Error(t, err)
	})
}

// --- AdversarialProtocol tests ---

func TestAdversarialProtocol(t *testing.T) {
	p := NewAdversarialProtocol()
	ctx := context.Background()

	t.Run("assigns roles", func(t *testing.T) {
		state := DebateState{
			Topic:        "test topic",
			AgentIDs:     []string{"a1", "a2", "a3"},
			CurrentRound: 0,
		}
		prompts, err := p.NextRound(ctx, state)
		require.NoError(t, err)
		assert.Contains(t, prompts["a1"], "PRO")
		assert.Contains(t, prompts["a2"], "PRO")
		assert.Contains(t, prompts["a3"], "CON")
	})

	t.Run("too few agents", func(t *testing.T) {
		state := DebateState{
			Topic:    "test",
			AgentIDs: []string{"a1"},
		}
		_, err := p.NextRound(ctx, state)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least 2 agents")
	})
}

// --- JudgedProtocol tests ---

func TestJudgedProtocol(t *testing.T) {
	ctx := context.Background()

	t.Run("judge role assigned", func(t *testing.T) {
		p := NewJudgedProtocol("a2")
		state := DebateState{
			Topic:        "test topic",
			AgentIDs:     []string{"a1", "a2", "a3"},
			CurrentRound: 0,
		}
		prompts, err := p.NextRound(ctx, state)
		require.NoError(t, err)
		// Judge is excluded from the first pass; only participants get prompts.
		_, judgeInFirstPass := prompts["a2"]
		assert.False(t, judgeInFirstPass, "judge should not receive a first-pass prompt")
		assert.Contains(t, prompts["a1"], "test topic")
		assert.NotContains(t, prompts["a1"], "JUDGE")

		// Judge prompt is produced by FollowUp with current-round contributions.
		currentRound := Round{
			Number: 1,
			Contributions: []Contribution{
				{AgentID: "a1", Content: "argument one"},
				{AgentID: "a3", Content: "argument three"},
			},
		}
		followUp, err := p.FollowUp(ctx, state, currentRound)
		require.NoError(t, err)
		assert.Contains(t, followUp["a2"], "JUDGE")
		assert.Contains(t, followUp["a2"], "argument one")
		assert.Contains(t, followUp["a2"], "argument three")
	})

	t.Run("default judge is first agent", func(t *testing.T) {
		p := NewJudgedProtocol("")
		state := DebateState{
			Topic:        "test",
			AgentIDs:     []string{"a1", "a2"},
			CurrentRound: 0,
		}
		prompts, err := p.NextRound(ctx, state)
		require.NoError(t, err)
		// a1 is the default judge; it must be absent from the first pass.
		_, judgeInFirstPass := prompts["a1"]
		assert.False(t, judgeInFirstPass, "default judge should not receive a first-pass prompt")

		followUp, err := p.FollowUp(ctx, state, Round{Number: 1, Contributions: []Contribution{{AgentID: "a2", Content: "x"}}})
		require.NoError(t, err)
		assert.Contains(t, followUp["a1"], "JUDGE")
	})

	t.Run("too few agents", func(t *testing.T) {
		p := NewJudgedProtocol("")
		state := DebateState{
			Topic:    "test",
			AgentIDs: []string{"a1"},
		}
		_, err := p.NextRound(ctx, state)
		require.Error(t, err)
	})
}

// --- StabilityDetector tests ---

func TestStabilityDetector(t *testing.T) {
	ctx := context.Background()

	t.Run("not enough rounds", func(t *testing.T) {
		d := NewStabilityDetector(0.8)
		state := DebateState{Rounds: []Round{{Number: 1}}}
		r, err := d.Check(ctx, state)
		require.NoError(t, err)
		assert.False(t, r.Converged)
	})

	t.Run("identical rounds converge", func(t *testing.T) {
		d := NewStabilityDetector(0.8)
		state := DebateState{
			Rounds: []Round{
				{Number: 1, Contributions: []Contribution{{AgentID: "a1", Content: "the same content"}}},
				{Number: 2, Contributions: []Contribution{{AgentID: "a1", Content: "the same content"}}},
			},
		}
		r, err := d.Check(ctx, state)
		require.NoError(t, err)
		assert.True(t, r.Converged)
		assert.Equal(t, 1.0, r.Score)
	})

	t.Run("different rounds do not converge", func(t *testing.T) {
		d := NewStabilityDetector(0.99)
		state := DebateState{
			Rounds: []Round{
				{Number: 1, Contributions: []Contribution{{AgentID: "a1", Content: "alpha beta gamma"}}},
				{Number: 2, Contributions: []Contribution{{AgentID: "a1", Content: "completely different text xyz"}}},
			},
		}
		r, err := d.Check(ctx, state)
		require.NoError(t, err)
		assert.False(t, r.Converged)
	})

	t.Run("threshold clamping", func(t *testing.T) {
		d := NewStabilityDetector(1.5)
		assert.Equal(t, 1.0, d.Threshold)
		d = NewStabilityDetector(-0.5)
		assert.Equal(t, 0.0, d.Threshold)
	})
}

// --- AgreementDetector tests ---

func TestAgreementDetector(t *testing.T) {
	ctx := context.Background()

	t.Run("no rounds", func(t *testing.T) {
		d := NewAgreementDetector(0.6)
		r, err := d.Check(ctx, DebateState{})
		require.NoError(t, err)
		assert.False(t, r.Converged)
	})

	t.Run("identical contributions converge", func(t *testing.T) {
		d := NewAgreementDetector(0.5)
		state := DebateState{
			Rounds: []Round{{
				Number: 1,
				Contributions: []Contribution{
					{AgentID: "a1", Content: "the answer is yes"},
					{AgentID: "a2", Content: "the answer is yes"},
					{AgentID: "a3", Content: "something entirely different"},
				},
			}},
		}
		r, err := d.Check(ctx, state)
		require.NoError(t, err)
		assert.True(t, r.Converged)
	})

	t.Run("all different do not converge with high threshold", func(t *testing.T) {
		d := NewAgreementDetector(0.8)
		state := DebateState{
			Rounds: []Round{{
				Number: 1,
				Contributions: []Contribution{
					{AgentID: "a1", Content: "alpha"},
					{AgentID: "a2", Content: "beta"},
					{AgentID: "a3", Content: "gamma"},
				},
			}},
		}
		r, err := d.Check(ctx, state)
		require.NoError(t, err)
		assert.False(t, r.Converged)
	})
}

// --- MaxRoundsDetector tests ---

func TestMaxRoundsDetector(t *testing.T) {
	ctx := context.Background()
	d := &MaxRoundsDetector{}

	t.Run("not at max", func(t *testing.T) {
		r, err := d.Check(ctx, DebateState{CurrentRound: 0, MaxRounds: 5})
		require.NoError(t, err)
		assert.False(t, r.Converged)
	})

	t.Run("at max", func(t *testing.T) {
		r, err := d.Check(ctx, DebateState{CurrentRound: 4, MaxRounds: 5})
		require.NoError(t, err)
		assert.True(t, r.Converged)
	})
}

// --- BigramSimilarity tests ---

func TestBigramSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want float64
	}{
		{"identical", "hello", "hello", 1.0},
		{"empty", "", "", 0.0},
		{"one char", "a", "b", 0.0},
		{"same string", "test", "test", 1.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bigramSimilarity(tt.a, tt.b)
			assert.InDelta(t, tt.want, got, 0.01)
		})
	}
}

// --- DebateOrchestrator tests ---

func TestDebateOrchestrator_Invoke(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		agents := []agent.Agent{
			newMockAgent("a1"),
			newMockAgent("a2"),
		}
		d := NewDebateOrchestrator(agents, WithMaxRounds(2))
		result, err := d.Invoke(context.Background(), "test topic")
		require.NoError(t, err)

		dr, ok := result.(*DebateResult)
		require.True(t, ok)
		assert.Equal(t, "test topic", dr.Topic)
		assert.Equal(t, 2, dr.Metrics.TotalRounds)
		assert.Equal(t, 4, dr.Metrics.TotalContributions)
	})

	t.Run("convergence stops early", func(t *testing.T) {
		callCount := 0
		agents := []agent.Agent{
			newMockAgentWithFn("a1", func(_ context.Context, _ string) (string, error) {
				callCount++
				return "same response", nil
			}),
			newMockAgentWithFn("a2", func(_ context.Context, _ string) (string, error) {
				callCount++
				return "same response", nil
			}),
		}
		d := NewDebateOrchestrator(agents,
			WithMaxRounds(10),
			WithConvergenceDetector(NewStabilityDetector(0.8)),
		)
		result, err := d.Invoke(context.Background(), "test topic")
		require.NoError(t, err)

		dr := result.(*DebateResult)
		assert.True(t, dr.Convergence.Converged)
		assert.Less(t, dr.Metrics.TotalRounds, 10)
	})

	t.Run("invalid input type", func(t *testing.T) {
		agents := []agent.Agent{newMockAgent("a1"), newMockAgent("a2")}
		d := NewDebateOrchestrator(agents)
		_, err := d.Invoke(context.Background(), 123)
		require.Error(t, err)
		var coreErr *core.Error
		assert.True(t, errors.As(err, &coreErr))
		assert.Equal(t, core.ErrInvalidInput, coreErr.Code)
	})

	t.Run("empty topic", func(t *testing.T) {
		agents := []agent.Agent{newMockAgent("a1"), newMockAgent("a2")}
		d := NewDebateOrchestrator(agents)
		_, err := d.Invoke(context.Background(), "   ")
		require.Error(t, err)
	})

	t.Run("too few agents", func(t *testing.T) {
		agents := []agent.Agent{newMockAgent("a1")}
		d := NewDebateOrchestrator(agents)
		_, err := d.Invoke(context.Background(), "test")
		require.Error(t, err)
	})

	t.Run("agent error propagates", func(t *testing.T) {
		agents := []agent.Agent{
			newMockAgentWithFn("a1", func(_ context.Context, _ string) (string, error) {
				return "", errors.New("agent failed")
			}),
			newMockAgent("a2"),
		}
		d := NewDebateOrchestrator(agents, WithMaxRounds(1))
		_, err := d.Invoke(context.Background(), "test")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "agent failed")
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		agents := []agent.Agent{newMockAgent("a1"), newMockAgent("a2")}
		d := NewDebateOrchestrator(agents, WithMaxRounds(5))
		_, err := d.Invoke(ctx, "test")
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("hooks are called", func(t *testing.T) {
		var beforeCalls, afterCalls, convCalls int
		hooks := Hooks{
			BeforeRound: func(_ context.Context, _ DebateState) error {
				beforeCalls++
				return nil
			},
			AfterRound: func(_ context.Context, _ DebateState) error {
				afterCalls++
				return nil
			},
			OnConvergence: func(_ context.Context, _ ConvergenceResult) error {
				convCalls++
				return nil
			},
		}

		agents := []agent.Agent{newMockAgent("a1"), newMockAgent("a2")}
		d := NewDebateOrchestrator(agents, WithMaxRounds(2), WithHooks(hooks))
		_, err := d.Invoke(context.Background(), "test")
		require.NoError(t, err)
		assert.Equal(t, 2, beforeCalls)
		assert.Equal(t, 2, afterCalls)
		assert.Equal(t, 2, convCalls)
	})

	t.Run("hook error stops debate", func(t *testing.T) {
		hooks := Hooks{
			BeforeRound: func(_ context.Context, _ DebateState) error {
				return errors.New("hook stopped it")
			},
		}

		agents := []agent.Agent{newMockAgent("a1"), newMockAgent("a2")}
		d := NewDebateOrchestrator(agents, WithMaxRounds(5), WithHooks(hooks))
		_, err := d.Invoke(context.Background(), "test")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "hook stopped it")
	})
}

func TestDebateOrchestrator_Stream(t *testing.T) {
	t.Run("yields events in order", func(t *testing.T) {
		agents := []agent.Agent{newMockAgent("a1"), newMockAgent("a2")}
		d := NewDebateOrchestrator(agents, WithMaxRounds(1))

		var events []DebateEventType
		for val, err := range d.Stream(context.Background(), "test") {
			require.NoError(t, err)
			event, ok := val.(DebateEvent)
			require.True(t, ok)
			events = append(events, event.Type)
		}

		assert.Equal(t, []DebateEventType{
			EventRoundStart,
			EventContribution,
			EventContribution,
			EventRoundEnd,
			EventConvergence,
			EventComplete,
		}, events)
	})

	t.Run("context cancellation stops stream", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		slowAgent := newMockAgentWithFn("slow", func(ctx context.Context, _ string) (string, error) {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(10 * time.Second):
				return "too slow", nil
			}
		})

		agents := []agent.Agent{slowAgent, newMockAgent("a2")}
		d := NewDebateOrchestrator(agents, WithMaxRounds(5))

		cancel()
		for _, err := range d.Stream(ctx, "test") {
			if err != nil {
				assert.ErrorIs(t, err, context.Canceled)
				break
			}
		}
	})

	t.Run("invalid input yields error", func(t *testing.T) {
		agents := []agent.Agent{newMockAgent("a1"), newMockAgent("a2")}
		d := NewDebateOrchestrator(agents)

		for _, err := range d.Stream(context.Background(), 42) {
			require.Error(t, err)
			break
		}
	})
}

// --- GeneratorEvaluator tests ---

func TestGeneratorEvaluator_Invoke(t *testing.T) {
	t.Run("approved on first try", func(t *testing.T) {
		gen := newMockAgentWithFn("gen", func(_ context.Context, _ string) (string, error) {
			return "good response", nil
		})
		evaluators := []EvaluatorFunc{
			func(_ context.Context, _, _ string) (Critique, error) {
				return Critique{Approved: true, Score: 1.0, Feedback: "perfect"}, nil
			},
		}

		ge := NewGeneratorEvaluator(gen, evaluators)
		result, err := ge.Invoke(context.Background(), "write something")
		require.NoError(t, err)

		r, ok := result.(*GeneratorEvaluatorResult)
		require.True(t, ok)
		assert.Equal(t, "good response", r.FinalResponse)
		assert.True(t, r.Approved)
		assert.Equal(t, 1, r.Iterations)
	})

	t.Run("rejected then approved", func(t *testing.T) {
		callCount := 0
		gen := newMockAgentWithFn("gen", func(_ context.Context, _ string) (string, error) {
			callCount++
			if callCount == 1 {
				return "bad response", nil
			}
			return "good response", nil
		})
		evaluators := []EvaluatorFunc{
			func(_ context.Context, _, response string) (Critique, error) {
				if response == "good response" {
					return Critique{Approved: true, Score: 1.0}, nil
				}
				return Critique{Approved: false, Score: 0.3, Feedback: "needs work"}, nil
			},
		}

		ge := NewGeneratorEvaluator(gen, evaluators, WithGEMaxIterations(5))
		result, err := ge.Invoke(context.Background(), "write something")
		require.NoError(t, err)

		r := result.(*GeneratorEvaluatorResult)
		assert.True(t, r.Approved)
		assert.Equal(t, 2, r.Iterations)
	})

	t.Run("max iterations exhausted", func(t *testing.T) {
		gen := newMockAgent("gen")
		evaluators := []EvaluatorFunc{
			func(_ context.Context, _, _ string) (Critique, error) {
				return Critique{Approved: false, Score: 0.1, Feedback: "nope"}, nil
			},
		}

		ge := NewGeneratorEvaluator(gen, evaluators, WithGEMaxIterations(2))
		result, err := ge.Invoke(context.Background(), "test")
		require.NoError(t, err)

		r := result.(*GeneratorEvaluatorResult)
		assert.False(t, r.Approved)
		assert.Equal(t, 2, r.Iterations)
	})

	t.Run("majority approval", func(t *testing.T) {
		gen := newMockAgent("gen")
		evaluators := []EvaluatorFunc{
			func(_ context.Context, _, _ string) (Critique, error) {
				return Critique{Approved: true, Score: 0.8}, nil
			},
			func(_ context.Context, _, _ string) (Critique, error) {
				return Critique{Approved: true, Score: 0.7}, nil
			},
			func(_ context.Context, _, _ string) (Critique, error) {
				return Critique{Approved: false, Score: 0.3}, nil
			},
		}

		ge := NewGeneratorEvaluator(gen, evaluators, WithGEApprovalStrategy(ApprovalMajority))
		result, err := ge.Invoke(context.Background(), "test")
		require.NoError(t, err)

		r := result.(*GeneratorEvaluatorResult)
		assert.True(t, r.Approved)
		assert.Equal(t, 1, r.Iterations)
	})

	t.Run("all approval rejects partial", func(t *testing.T) {
		gen := newMockAgent("gen")
		evaluators := []EvaluatorFunc{
			func(_ context.Context, _, _ string) (Critique, error) {
				return Critique{Approved: true}, nil
			},
			func(_ context.Context, _, _ string) (Critique, error) {
				return Critique{Approved: false}, nil
			},
		}

		ge := NewGeneratorEvaluator(gen, evaluators, WithGEMaxIterations(1))
		result, err := ge.Invoke(context.Background(), "test")
		require.NoError(t, err)

		r := result.(*GeneratorEvaluatorResult)
		assert.False(t, r.Approved)
	})

	t.Run("invalid input", func(t *testing.T) {
		gen := newMockAgent("gen")
		ge := NewGeneratorEvaluator(gen, nil)
		_, err := ge.Invoke(context.Background(), 123)
		require.Error(t, err)
	})

	t.Run("empty prompt", func(t *testing.T) {
		gen := newMockAgent("gen")
		ge := NewGeneratorEvaluator(gen, nil)
		_, err := ge.Invoke(context.Background(), "")
		require.Error(t, err)
	})

	t.Run("nil generator", func(t *testing.T) {
		ge := NewGeneratorEvaluator(nil, nil)
		_, err := ge.Invoke(context.Background(), "test")
		require.Error(t, err)
	})

	t.Run("evaluator error propagates", func(t *testing.T) {
		gen := newMockAgent("gen")
		evaluators := []EvaluatorFunc{
			func(_ context.Context, _, _ string) (Critique, error) {
				return Critique{}, errors.New("eval failed")
			},
		}

		ge := NewGeneratorEvaluator(gen, evaluators)
		_, err := ge.Invoke(context.Background(), "test")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "eval failed")
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		gen := newMockAgent("gen")
		ge := NewGeneratorEvaluator(gen, nil, WithGEMaxIterations(5))
		_, err := ge.Invoke(ctx, "test")
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})
}

func TestGeneratorEvaluator_Stream(t *testing.T) {
	t.Run("yields events in order", func(t *testing.T) {
		gen := newMockAgent("gen")
		evaluators := []EvaluatorFunc{
			func(_ context.Context, _, _ string) (Critique, error) {
				return Critique{Approved: true, Score: 1.0}, nil
			},
		}

		ge := NewGeneratorEvaluator(gen, evaluators)
		var eventTypes []GeneratorEvaluatorEventType
		for val, err := range ge.Stream(context.Background(), "test") {
			require.NoError(t, err)
			event, ok := val.(GeneratorEvaluatorEvent)
			require.True(t, ok)
			eventTypes = append(eventTypes, event.Type)
		}

		assert.Equal(t, []GeneratorEvaluatorEventType{
			GEEventGenerate,
			GEEventEvaluate,
			GEEventApproved,
			GEEventComplete,
		}, eventTypes)
	})

	t.Run("rejected then approved", func(t *testing.T) {
		callCount := 0
		gen := newMockAgentWithFn("gen", func(_ context.Context, _ string) (string, error) {
			callCount++
			return fmt.Sprintf("response %d", callCount), nil
		})
		evaluators := []EvaluatorFunc{
			func(_ context.Context, _, response string) (Critique, error) {
				if response == "response 2" {
					return Critique{Approved: true}, nil
				}
				return Critique{Approved: false, Feedback: "try again"}, nil
			},
		}

		ge := NewGeneratorEvaluator(gen, evaluators, WithGEMaxIterations(5))
		var eventTypes []GeneratorEvaluatorEventType
		for val, err := range ge.Stream(context.Background(), "test") {
			require.NoError(t, err)
			event := val.(GeneratorEvaluatorEvent)
			eventTypes = append(eventTypes, event.Type)
		}

		assert.Contains(t, eventTypes, GEEventRejected)
		assert.Contains(t, eventTypes, GEEventApproved)
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		gen := newMockAgent("gen")
		ge := NewGeneratorEvaluator(gen, nil, WithGEMaxIterations(5))

		for _, err := range ge.Stream(ctx, "test") {
			if err != nil {
				assert.ErrorIs(t, err, context.Canceled)
				break
			}
		}
	})
}

// --- Team pattern adapter tests ---

func TestDebatePattern(t *testing.T) {
	agents := []agent.Agent{newMockAgent("a1"), newMockAgent("a2")}
	p := DebatePattern(agents, WithMaxRounds(1))
	assert.Equal(t, "debate", p.Name())

	result, err := p.Invoke(context.Background(), "test topic")
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestGeneratorEvaluatorPattern(t *testing.T) {
	gen := newMockAgent("gen")
	evaluators := []EvaluatorFunc{
		func(_ context.Context, _, _ string) (Critique, error) {
			return Critique{Approved: true}, nil
		},
	}
	p := GeneratorEvaluatorPattern(gen, evaluators)
	assert.Equal(t, "generator_evaluator", p.Name())

	result, err := p.Invoke(context.Background(), "test")
	require.NoError(t, err)
	require.NotNil(t, result)
}

// --- Middleware tests ---

func TestApplyMiddleware(t *testing.T) {
	agents := []agent.Agent{newMockAgent("a1"), newMockAgent("a2")}
	d := NewDebateOrchestrator(agents, WithMaxRounds(1))

	var called bool
	mw := func(next core.Runnable) core.Runnable {
		return &middlewareWrapper{next: next, onInvoke: func() { called = true }}
	}

	wrapped := ApplyMiddleware(d, Middleware(mw))
	_, err := wrapped.Invoke(context.Background(), "test")
	require.NoError(t, err)
	assert.True(t, called)
}

func TestAsOrchestrationPattern(t *testing.T) {
	agents := []agent.Agent{newMockAgent("a1"), newMockAgent("a2")}
	d := NewDebateOrchestrator(agents, WithMaxRounds(1))

	p := AsOrchestrationPattern("custom-debate", d)
	assert.Equal(t, "custom-debate", p.Name())

	result, err := p.Invoke(context.Background(), "test")
	require.NoError(t, err)
	require.NotNil(t, result)
}

// middlewareWrapper is a test helper for middleware testing.
type middlewareWrapper struct {
	next     core.Runnable
	onInvoke func()
}

func (w *middlewareWrapper) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	if w.onInvoke != nil {
		w.onInvoke()
	}
	return w.next.Invoke(ctx, input, opts...)
}

func (w *middlewareWrapper) Stream(ctx context.Context, input any, opts ...core.Option) iter.Seq2[any, error] {
	return w.next.Stream(ctx, input, opts...)
}

// --- Compile-time interface checks ---

var (
	_ core.Runnable       = (*DebateOrchestrator)(nil)
	_ core.Runnable       = (*GeneratorEvaluator)(nil)
	_ Protocol            = (*RoundRobinProtocol)(nil)
	_ Protocol            = (*AdversarialProtocol)(nil)
	_ Protocol            = (*JudgedProtocol)(nil)
	_ ConvergenceDetector = (*StabilityDetector)(nil)
	_ ConvergenceDetector = (*AgreementDetector)(nil)
	_ ConvergenceDetector = (*MaxRoundsDetector)(nil)
)

// Unused import guard for schema package.
var _ = schema.SystemMessage{}
