package orchestration

import (
	"context"
	"errors"
	"iter"
	"testing"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/tool"
)

// mockAgent is a test double for agent.Agent.
type mockAgent struct {
	id      string
	persona agent.Persona
	result  string
	err     error
}

func (m *mockAgent) ID() string                    { return m.id }
func (m *mockAgent) Persona() agent.Persona        { return m.persona }
func (m *mockAgent) Tools() []tool.Tool             { return nil }
func (m *mockAgent) Children() []agent.Agent        { return nil }

func (m *mockAgent) Invoke(_ context.Context, _ string, _ ...agent.Option) (string, error) {
	return m.result, m.err
}

func (m *mockAgent) Stream(_ context.Context, _ string, _ ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		if m.err != nil {
			yield(agent.Event{}, m.err)
			return
		}
		if !yield(agent.Event{Type: agent.EventText, Text: m.result, AgentID: m.id}, nil) {
			return
		}
		yield(agent.Event{Type: agent.EventDone, AgentID: m.id}, nil)
	}
}

func TestSupervisor_StrategyDelegation(t *testing.T) {
	a1 := &mockAgent{id: "math", result: "42"}
	a2 := &mockAgent{id: "code", result: "print('hi')"}

	// Always pick the first agent.
	strategy := func(_ context.Context, _ any, agents []agent.Agent) (agent.Agent, error) {
		return agents[0], nil
	}

	s := NewSupervisor(strategy, a1, a2)
	result, err := s.Invoke(context.Background(), "compute")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "42" {
		t.Fatalf("expected 42, got %v", result)
	}
}

func TestSupervisor_RoundRobin(t *testing.T) {
	a1 := &mockAgent{id: "a1", result: "r1"}
	a2 := &mockAgent{id: "a2", result: "r2"}
	a3 := &mockAgent{id: "a3", result: "r3"}

	strategy := RoundRobin()
	s := NewSupervisor(strategy, a1, a2, a3)

	// First call should pick a1.
	result, err := s.Invoke(context.Background(), "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "r1" {
		t.Fatalf("expected r1, got %v", result)
	}

	// Second call should pick a2.
	result, err = s.Invoke(context.Background(), "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "r2" {
		t.Fatalf("expected r2, got %v", result)
	}

	// Third call should pick a3.
	result, err = s.Invoke(context.Background(), "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "r3" {
		t.Fatalf("expected r3, got %v", result)
	}

	// Fourth call wraps back to a1.
	result, err = s.Invoke(context.Background(), "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "r1" {
		t.Fatalf("expected r1 (wrap-around), got %v", result)
	}
}

func TestSupervisor_MaxRounds(t *testing.T) {
	callCount := 0
	a1 := &mockAgent{id: "a1", result: "result"}

	strategy := func(_ context.Context, _ any, agents []agent.Agent) (agent.Agent, error) {
		callCount++
		return agents[0], nil
	}

	s := NewSupervisor(strategy, a1).WithMaxRounds(3)
	_, err := s.Invoke(context.Background(), "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 3 {
		t.Fatalf("expected 3 rounds, got %d", callCount)
	}
}

func TestSupervisor_NilAgentStop(t *testing.T) {
	callCount := 0
	a1 := &mockAgent{id: "a1", result: "done"}

	strategy := func(_ context.Context, _ any, _ []agent.Agent) (agent.Agent, error) {
		callCount++
		if callCount > 1 {
			return nil, nil // Stop after first round.
		}
		return a1, nil
	}

	s := NewSupervisor(strategy, a1).WithMaxRounds(10)
	result, err := s.Invoke(context.Background(), "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "done" {
		t.Fatalf("expected done, got %v", result)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 strategy calls, got %d", callCount)
	}
}

func TestSupervisor_NoAgents(t *testing.T) {
	strategy := func(_ context.Context, _ any, _ []agent.Agent) (agent.Agent, error) {
		return nil, nil
	}

	s := NewSupervisor(strategy)
	_, err := s.Invoke(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error for no agents")
	}
}

func TestSupervisor_StrategyError(t *testing.T) {
	errStrategy := errors.New("strategy error")
	a1 := &mockAgent{id: "a1", result: "r"}

	strategy := func(_ context.Context, _ any, _ []agent.Agent) (agent.Agent, error) {
		return nil, errStrategy
	}

	s := NewSupervisor(strategy, a1)
	_, err := s.Invoke(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, errStrategy) {
		t.Fatalf("expected strategy error, got %v", err)
	}
}

func TestSupervisor_AgentError(t *testing.T) {
	errAgent := errors.New("agent failed")
	a1 := &mockAgent{id: "a1", err: errAgent}

	strategy := func(_ context.Context, _ any, agents []agent.Agent) (agent.Agent, error) {
		return agents[0], nil
	}

	s := NewSupervisor(strategy, a1)
	_, err := s.Invoke(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, errAgent) {
		t.Fatalf("expected agent error, got %v", err)
	}
}

func TestSupervisor_DelegateBySkill(t *testing.T) {
	mathAgent := &mockAgent{
		id:      "math",
		persona: agent.Persona{Goal: "solve math equations and calculations"},
		result:  "42",
	}
	codeAgent := &mockAgent{
		id:      "code",
		persona: agent.Persona{Goal: "write code and programs"},
		result:  "print(42)",
	}

	strategy := DelegateBySkill()
	s := NewSupervisor(strategy, mathAgent, codeAgent)

	// "solve math" should match math agent's goal.
	result, err := s.Invoke(context.Background(), "solve math equation")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "42" {
		t.Fatalf("expected 42, got %v", result)
	}
}

func TestSupervisor_LoadBalanced(t *testing.T) {
	a1 := &mockAgent{id: "a1", result: "r1"}
	a2 := &mockAgent{id: "a2", result: "r2"}

	strategy := LoadBalanced()
	s := NewSupervisor(strategy, a1, a2)

	// First call should pick a1 (both at 0, a1 is first).
	result, err := s.Invoke(context.Background(), "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "r1" {
		t.Fatalf("expected r1, got %v", result)
	}

	// Second call should pick a2 (a1=1, a2=0).
	result, err = s.Invoke(context.Background(), "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "r2" {
		t.Fatalf("expected r2, got %v", result)
	}
}

func TestSupervisor_Stream(t *testing.T) {
	a1 := &mockAgent{id: "a1", result: "streamed"}

	strategy := func(_ context.Context, _ any, agents []agent.Agent) (agent.Agent, error) {
		return agents[0], nil
	}

	s := NewSupervisor(strategy, a1)
	var events []agent.Event
	for val, err := range s.Stream(context.Background(), "x") {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		events = append(events, val.(agent.Event))
	}
	if len(events) == 0 {
		t.Fatal("expected at least one event")
	}
	if events[0].Text != "streamed" {
		t.Fatalf("expected streamed, got %v", events[0].Text)
	}
}

func TestSupervisor_Stream_NoAgents(t *testing.T) {
	strategy := func(_ context.Context, _ any, _ []agent.Agent) (agent.Agent, error) {
		return nil, nil
	}
	s := NewSupervisor(strategy)

	for _, err := range s.Stream(context.Background(), "x") {
		if err == nil {
			t.Fatal("expected error for no agents")
		}
		return
	}
	t.Fatal("expected at least one stream result")
}

func TestSupervisor_Stream_StrategyError(t *testing.T) {
	errStrategy := errors.New("strategy failed")
	a1 := &mockAgent{id: "a1", result: "r"}

	strategy := func(_ context.Context, _ any, _ []agent.Agent) (agent.Agent, error) {
		return nil, errStrategy
	}

	s := NewSupervisor(strategy, a1)
	for _, err := range s.Stream(context.Background(), "x") {
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, errStrategy) {
			t.Fatalf("expected strategy error, got %v", err)
		}
		return
	}
	t.Fatal("expected at least one stream result")
}

func TestSupervisor_Stream_StrategyNilStop(t *testing.T) {
	callCount := 0
	a1 := &mockAgent{id: "a1", result: "done"}

	strategy := func(_ context.Context, _ any, agents []agent.Agent) (agent.Agent, error) {
		callCount++
		if callCount > 1 {
			return nil, nil // Stop.
		}
		return agents[0], nil
	}

	s := NewSupervisor(strategy, a1).WithMaxRounds(10)
	var results []any
	for val, err := range s.Stream(context.Background(), "x") {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		results = append(results, val)
	}
	// Strategy returns nil on 2nd call, so we get the result from round 1.
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
}

func TestSupervisor_Stream_MultiRound(t *testing.T) {
	a1 := &mockAgent{id: "a1", result: "streamed"}

	strategy := func(_ context.Context, _ any, agents []agent.Agent) (agent.Agent, error) {
		return agents[0], nil
	}

	// 3 rounds: first 2 invoke, last streams.
	s := NewSupervisor(strategy, a1).WithMaxRounds(3)
	var results []any
	for val, err := range s.Stream(context.Background(), "x") {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		results = append(results, val)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
}

func TestSupervisor_Stream_InvokeError(t *testing.T) {
	errAgent := errors.New("invoke failed")
	a1 := &mockAgent{id: "a1", err: errAgent}

	strategy := func(_ context.Context, _ any, agents []agent.Agent) (agent.Agent, error) {
		return agents[0], nil
	}

	// 2 rounds: first round invokes (and fails).
	s := NewSupervisor(strategy, a1).WithMaxRounds(2)
	for _, err := range s.Stream(context.Background(), "x") {
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, errAgent) {
			t.Fatalf("expected agent error, got %v", err)
		}
		return
	}
	t.Fatal("expected at least one stream result")
}

func TestSupervisor_Stream_StreamError(t *testing.T) {
	errStream := errors.New("stream failed")
	a1 := &mockAgent{id: "a1", err: errStream}

	strategy := func(_ context.Context, _ any, agents []agent.Agent) (agent.Agent, error) {
		return agents[0], nil
	}

	// Single round streams the agent (which has error).
	s := NewSupervisor(strategy, a1)
	for _, err := range s.Stream(context.Background(), "x") {
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, errStream) {
			t.Fatalf("expected stream error, got %v", err)
		}
		return
	}
	t.Fatal("expected at least one stream result")
}

func TestSupervisor_Stream_ConsumerBreak(t *testing.T) {
	a1 := &mockAgent{id: "a1", result: "data"}

	strategy := func(_ context.Context, _ any, agents []agent.Agent) (agent.Agent, error) {
		return agents[0], nil
	}

	s := NewSupervisor(strategy, a1)
	count := 0
	for _, err := range s.Stream(context.Background(), "x") {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count++
		break // Consumer breaks early.
	}
	if count != 1 {
		t.Fatalf("expected 1 event before break, got %d", count)
	}
}

func TestSupervisor_WithMaxRounds_Zero(t *testing.T) {
	a1 := &mockAgent{id: "a1", result: "r"}
	strategy := func(_ context.Context, _ any, agents []agent.Agent) (agent.Agent, error) {
		return agents[0], nil
	}

	// WithMaxRounds(0) should keep the default (1).
	s := NewSupervisor(strategy, a1).WithMaxRounds(0)
	if s.maxRounds != 1 {
		t.Fatalf("expected maxRounds=1, got %d", s.maxRounds)
	}
}

func TestRoundRobin_EmptyAgents(t *testing.T) {
	strategy := RoundRobin()
	selected, err := strategy(context.Background(), "x", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected != nil {
		t.Fatal("expected nil for empty agents")
	}
}

func TestLoadBalanced_EmptyAgents(t *testing.T) {
	strategy := LoadBalanced()
	selected, err := strategy(context.Background(), "x", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected != nil {
		t.Fatal("expected nil for empty agents")
	}
}

func TestDelegateBySkill_NoMatch(t *testing.T) {
	a1 := &mockAgent{
		id:      "generic",
		persona: agent.Persona{Goal: "handle general tasks"},
		result:  "generic-result",
	}

	strategy := DelegateBySkill()
	selected, err := strategy(context.Background(), "ab", []agent.Agent{a1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Short words (<=2 chars) don't match, so falls back to first agent.
	if selected.ID() != "generic" {
		t.Fatalf("expected generic (fallback), got %v", selected.ID())
	}
}
