package orchestration

import (
	"context"
	"errors"
	"iter"
	"testing"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/tool"
)

// bbMockAgent is a test double for blackboard tests with configurable behavior.
type bbMockAgent struct {
	id       string
	invokeFn func(ctx context.Context, input string) (string, error)
}

func (m *bbMockAgent) ID() string                    { return m.id }
func (m *bbMockAgent) Persona() agent.Persona        { return agent.Persona{} }
func (m *bbMockAgent) Tools() []tool.Tool             { return nil }
func (m *bbMockAgent) Children() []agent.Agent        { return nil }

func (m *bbMockAgent) Invoke(ctx context.Context, input string, _ ...agent.Option) (string, error) {
	if m.invokeFn != nil {
		return m.invokeFn(ctx, input)
	}
	return "default-" + m.id, nil
}

func (m *bbMockAgent) Stream(_ context.Context, input string, _ ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		result, _ := m.Invoke(context.Background(), input)
		yield(agent.Event{Type: agent.EventText, Text: result, AgentID: m.id}, nil)
	}
}

func TestBlackboard_MultiAgent(t *testing.T) {
	a1 := &bbMockAgent{id: "analyzer", invokeFn: func(_ context.Context, _ string) (string, error) {
		return "analyzed", nil
	}}
	a2 := &bbMockAgent{id: "synthesizer", invokeFn: func(_ context.Context, _ string) (string, error) {
		return "synthesized", nil
	}}

	roundCount := 0
	termination := func(board map[string]any) bool {
		roundCount++
		return roundCount > 1 // Run exactly 1 round.
	}

	bb := NewBlackboard(termination, a1, a2)
	result, err := bb.Invoke(context.Background(), "problem")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	board, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}

	if board["input"] != "problem" {
		t.Fatalf("expected input=problem, got %v", board["input"])
	}
	if board["analyzer"] != "analyzed" {
		t.Fatalf("expected analyzer=analyzed, got %v", board["analyzer"])
	}
	if board["synthesizer"] != "synthesized" {
		t.Fatalf("expected synthesizer=synthesized, got %v", board["synthesizer"])
	}
}

func TestBlackboard_TerminationCondition(t *testing.T) {
	a1 := &bbMockAgent{id: "worker"}

	termination := func(board map[string]any) bool {
		_, done := board["worker"]
		return done
	}

	bb := NewBlackboard(termination, a1)
	result, err := bb.Invoke(context.Background(), "start")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	board := result.(map[string]any)
	// After first round, "worker" key exists, so termination triggers on 2nd check.
	if _, ok := board["worker"]; !ok {
		t.Fatal("expected worker key in board")
	}
}

func TestBlackboard_MaxRounds(t *testing.T) {
	callCount := 0
	a1 := &bbMockAgent{id: "counter", invokeFn: func(_ context.Context, _ string) (string, error) {
		callCount++
		return "called", nil
	}}

	// Never terminate.
	termination := func(_ map[string]any) bool { return false }

	bb := NewBlackboard(termination, a1).WithMaxRounds(3)
	_, err := bb.Invoke(context.Background(), "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 3 {
		t.Fatalf("expected 3 calls, got %d", callCount)
	}
}

func TestBlackboard_BoardState(t *testing.T) {
	bb := NewBlackboard(func(_ map[string]any) bool { return true })
	bb.Set("key1", "value1")
	bb.Set("key2", 42)

	v, ok := bb.Get("key1")
	if !ok || v != "value1" {
		t.Fatalf("expected value1, got %v", v)
	}

	v, ok = bb.Get("key2")
	if !ok || v != 42 {
		t.Fatalf("expected 42, got %v", v)
	}

	_, ok = bb.Get("nonexistent")
	if ok {
		t.Fatal("expected nonexistent key to return false")
	}
}

func TestBlackboard_NoAgents(t *testing.T) {
	termination := func(_ map[string]any) bool { return false }
	bb := NewBlackboard(termination)

	_, err := bb.Invoke(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error for no agents")
	}
}

func TestBlackboard_AgentError(t *testing.T) {
	errAgent := errors.New("agent failed")
	a1 := &bbMockAgent{id: "broken", invokeFn: func(_ context.Context, _ string) (string, error) {
		return "", errAgent
	}}

	termination := func(_ map[string]any) bool { return false }
	bb := NewBlackboard(termination, a1)

	_, err := bb.Invoke(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, errAgent) {
		t.Fatalf("expected agent error, got %v", err)
	}
}

func TestBlackboard_Stream(t *testing.T) {
	a1 := &bbMockAgent{id: "worker"}

	roundCount := 0
	termination := func(_ map[string]any) bool {
		roundCount++
		return roundCount > 2 // Run 2 rounds.
	}

	bb := NewBlackboard(termination, a1)
	var results []any
	for val, err := range bb.Stream(context.Background(), "x") {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		results = append(results, val)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
}

func TestBlackboard_ImmediateTermination(t *testing.T) {
	a1 := &bbMockAgent{id: "worker"}
	callCount := 0

	// Terminate immediately (before any agent runs).
	termination := func(_ map[string]any) bool { return true }

	a1.invokeFn = func(_ context.Context, _ string) (string, error) {
		callCount++
		return "called", nil
	}

	bb := NewBlackboard(termination, a1)
	result, err := bb.Invoke(context.Background(), "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 0 {
		t.Fatalf("expected 0 agent calls for immediate termination, got %d", callCount)
	}
	board := result.(map[string]any)
	if board["input"] != "x" {
		t.Fatalf("expected input=x, got %v", board["input"])
	}
}

func TestBlackboard_SetOverwrite(t *testing.T) {
	bb := NewBlackboard(func(_ map[string]any) bool { return true })
	bb.Set("key", "v1")
	bb.Set("key", "v2")

	v, ok := bb.Get("key")
	if !ok || v != "v2" {
		t.Fatalf("expected v2, got %v", v)
	}
}

func TestBlackboard_Stream_NoAgents(t *testing.T) {
	termination := func(_ map[string]any) bool { return false }
	bb := NewBlackboard(termination)

	for _, err := range bb.Stream(context.Background(), "x") {
		if err == nil {
			t.Fatal("expected error for no agents")
		}
		return
	}
	t.Fatal("expected at least one stream result")
}

func TestBlackboard_Stream_AgentError(t *testing.T) {
	errAgent := errors.New("agent failed")
	a1 := &bbMockAgent{id: "broken", invokeFn: func(_ context.Context, _ string) (string, error) {
		return "", errAgent
	}}

	termination := func(_ map[string]any) bool { return false }
	bb := NewBlackboard(termination, a1)

	for _, err := range bb.Stream(context.Background(), "x") {
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

func TestBlackboard_Stream_ImmediateTermination(t *testing.T) {
	a1 := &bbMockAgent{id: "worker"}
	termination := func(_ map[string]any) bool { return true }

	bb := NewBlackboard(termination, a1)
	var results []any
	for val, err := range bb.Stream(context.Background(), "x") {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		results = append(results, val)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result (terminal board state)")
	}
}

func TestBlackboard_Stream_ConsumerBreak(t *testing.T) {
	a1 := &bbMockAgent{id: "worker"}
	termination := func(_ map[string]any) bool { return false }

	bb := NewBlackboard(termination, a1).WithMaxRounds(10)
	count := 0
	for _, err := range bb.Stream(context.Background(), "x") {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count++
		break // Consumer breaks early.
	}
	if count != 1 {
		t.Fatalf("expected 1 result before break, got %d", count)
	}
}

func TestBlackboard_WithMaxRounds_Zero(t *testing.T) {
	termination := func(_ map[string]any) bool { return true }
	bb := NewBlackboard(termination).WithMaxRounds(0)
	if bb.maxRounds != 10 { // default should be preserved
		t.Fatalf("expected maxRounds=10, got %d", bb.maxRounds)
	}
}
