package workflow

import (
	"context"
	"errors"
	"iter"
	"testing"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/tool"
)

// loopMockAgent is a mock agent for loop tests.
type loopMockAgent struct {
	id       string
	invokeFn func(ctx context.Context, input string) (string, error)
	streamFn func(ctx context.Context, input string) iter.Seq2[agent.Event, error]
}

func (a *loopMockAgent) ID() string              { return a.id }
func (a *loopMockAgent) Persona() agent.Persona  { return agent.Persona{} }
func (a *loopMockAgent) Tools() []tool.Tool      { return nil }
func (a *loopMockAgent) Children() []agent.Agent { return nil }
func (a *loopMockAgent) Invoke(ctx context.Context, input string, opts ...agent.Option) (string, error) {
	if a.invokeFn != nil {
		return a.invokeFn(ctx, input)
	}
	return a.id + ":" + input, nil
}
func (a *loopMockAgent) Stream(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
	if a.streamFn != nil {
		return a.streamFn(ctx, input)
	}
	return func(yield func(agent.Event, error) bool) {
		yield(agent.Event{Type: agent.EventText, Text: a.id + ":" + input, AgentID: a.id}, nil)
	}
}

var _ agent.Agent = (*loopMockAgent)(nil)

func TestNewLoopAgent_Defaults(t *testing.T) {
	child := &loopMockAgent{id: "child"}
	la := NewLoopAgent("loop", child)

	if la.ID() != "loop" {
		t.Errorf("ID() = %q, want %q", la.ID(), "loop")
	}
	if la.maxIterations != 10 {
		t.Errorf("maxIterations = %d, want 10", la.maxIterations)
	}
	if la.condition != nil {
		t.Error("condition should be nil by default")
	}
}

func TestNewLoopAgent_WithOptions(t *testing.T) {
	child := &loopMockAgent{id: "child"}
	cond := func(iteration int, lastResult string) bool { return true }
	la := NewLoopAgent("loop", child,
		WithLoopMaxIterations(5),
		WithLoopCondition(cond),
	)

	if la.maxIterations != 5 {
		t.Errorf("maxIterations = %d, want 5", la.maxIterations)
	}
	if la.condition == nil {
		t.Error("condition should not be nil")
	}
}

func TestWithLoopMaxIterations_IgnoresNonPositive(t *testing.T) {
	child := &loopMockAgent{id: "child"}
	la := NewLoopAgent("loop", child, WithLoopMaxIterations(0))
	if la.maxIterations != 10 {
		t.Errorf("maxIterations = %d, want 10 (default)", la.maxIterations)
	}

	la = NewLoopAgent("loop", child, WithLoopMaxIterations(-1))
	if la.maxIterations != 10 {
		t.Errorf("maxIterations = %d, want 10 (default)", la.maxIterations)
	}
}

func TestLoopAgent_ID(t *testing.T) {
	la := NewLoopAgent("my-loop", &loopMockAgent{id: "child"})
	if la.ID() != "my-loop" {
		t.Errorf("ID() = %q, want %q", la.ID(), "my-loop")
	}
}

func TestLoopAgent_Persona(t *testing.T) {
	la := NewLoopAgent("loop", &loopMockAgent{id: "child"})
	p := la.Persona()
	if p.Role == "" {
		t.Error("expected non-empty persona role")
	}
}

func TestLoopAgent_Tools_ReturnsNil(t *testing.T) {
	la := NewLoopAgent("loop", &loopMockAgent{id: "child"})
	if la.Tools() != nil {
		t.Error("expected nil tools")
	}
}

func TestLoopAgent_Children(t *testing.T) {
	child := &loopMockAgent{id: "child"}
	la := NewLoopAgent("loop", child)
	children := la.Children()
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	if children[0].ID() != "child" {
		t.Errorf("child ID = %q, want %q", children[0].ID(), "child")
	}
}

func TestLoopAgent_Invoke_RunsUntilMaxIterations(t *testing.T) {
	callCount := 0
	child := &loopMockAgent{
		id: "child",
		invokeFn: func(ctx context.Context, input string) (string, error) {
			callCount++
			return input + "+iter", nil
		},
	}

	la := NewLoopAgent("loop", child, WithLoopMaxIterations(3))
	result, err := la.Invoke(context.Background(), "start")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if callCount != 3 {
		t.Errorf("child called %d times, want 3", callCount)
	}
	// Each iteration appends "+iter" to the result.
	if result != "start+iter+iter+iter" {
		t.Errorf("result = %q, want %q", result, "start+iter+iter+iter")
	}
}

func TestLoopAgent_Invoke_StopsOnCondition(t *testing.T) {
	callCount := 0
	child := &loopMockAgent{
		id: "child",
		invokeFn: func(ctx context.Context, input string) (string, error) {
			callCount++
			return "DONE", nil
		},
	}

	la := NewLoopAgent("loop", child,
		WithLoopMaxIterations(10),
		WithLoopCondition(func(iteration int, lastResult string) bool {
			return lastResult == "DONE"
		}),
	)

	result, err := la.Invoke(context.Background(), "start")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if callCount != 1 {
		t.Errorf("child called %d times, want 1 (stopped by condition)", callCount)
	}
	if result != "DONE" {
		t.Errorf("result = %q, want %q", result, "DONE")
	}
}

func TestLoopAgent_Invoke_ChainsOutput(t *testing.T) {
	child := &loopMockAgent{
		id: "child",
		invokeFn: func(ctx context.Context, input string) (string, error) {
			return input + "→refined", nil
		},
	}

	la := NewLoopAgent("loop", child, WithLoopMaxIterations(3))
	result, err := la.Invoke(context.Background(), "raw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Each iteration takes previous output as input.
	want := "raw→refined→refined→refined"
	if result != want {
		t.Errorf("result = %q, want %q", result, want)
	}
}

func TestLoopAgent_Invoke_Error(t *testing.T) {
	child := &loopMockAgent{
		id: "child",
		invokeFn: func(ctx context.Context, input string) (string, error) {
			return "", errors.New("child failed")
		},
	}

	la := NewLoopAgent("loop", child)
	_, err := la.Invoke(context.Background(), "start")
	if err == nil {
		t.Fatal("expected error from child")
	}
}

func TestLoopAgent_Invoke_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	child := &loopMockAgent{
		id: "child",
		invokeFn: func(ctx context.Context, input string) (string, error) {
			return "ok", nil
		},
	}

	la := NewLoopAgent("loop", child)
	_, err := la.Invoke(ctx, "start")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestLoopAgent_Stream_EmitsEvents(t *testing.T) {
	child := &loopMockAgent{
		id: "child",
		streamFn: func(ctx context.Context, input string) iter.Seq2[agent.Event, error] {
			return func(yield func(agent.Event, error) bool) {
				yield(agent.Event{Type: agent.EventText, Text: "text:" + input, AgentID: "child"}, nil)
			}
		},
	}

	la := NewLoopAgent("loop", child, WithLoopMaxIterations(2))

	var texts []string
	var doneCount int
	for event, err := range la.Stream(context.Background(), "start") {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if event.Type == agent.EventText {
			texts = append(texts, event.Text)
		}
		if event.Type == agent.EventDone {
			doneCount++
		}
	}

	if len(texts) < 2 {
		t.Fatalf("expected at least 2 text events, got %d: %v", len(texts), texts)
	}
	if doneCount != 1 {
		t.Errorf("expected 1 done event, got %d", doneCount)
	}
}

func TestLoopAgent_Stream_Error(t *testing.T) {
	child := &loopMockAgent{
		id: "child",
		streamFn: func(ctx context.Context, input string) iter.Seq2[agent.Event, error] {
			return func(yield func(agent.Event, error) bool) {
				yield(agent.Event{}, errors.New("stream failed"))
			}
		},
	}

	la := NewLoopAgent("loop", child)

	var gotErr error
	for _, err := range la.Stream(context.Background(), "start") {
		if err != nil {
			gotErr = err
			break
		}
	}
	if gotErr == nil {
		t.Fatal("expected error from stream")
	}
}

func TestLoopAgent_Stream_StopsOnCondition(t *testing.T) {
	iterCount := 0
	child := &loopMockAgent{
		id: "child",
		streamFn: func(ctx context.Context, input string) iter.Seq2[agent.Event, error] {
			iterCount++
			return func(yield func(agent.Event, error) bool) {
				yield(agent.Event{Type: agent.EventText, Text: "STOP", AgentID: "child"}, nil)
			}
		},
	}

	la := NewLoopAgent("loop", child,
		WithLoopMaxIterations(10),
		WithLoopCondition(func(iteration int, lastResult string) bool {
			return lastResult == "STOP"
		}),
	)

	for event, err := range la.Stream(context.Background(), "start") {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_ = event
	}

	if iterCount != 1 {
		t.Errorf("child streamed %d times, want 1 (stopped by condition)", iterCount)
	}
}

func TestLoopAgent_Stream_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	child := &loopMockAgent{id: "child"}
	la := NewLoopAgent("loop", child)

	var gotErr error
	for _, err := range la.Stream(ctx, "start") {
		if err != nil {
			gotErr = err
			break
		}
	}
	if gotErr == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestLoopAgent_ImplementsAgent(t *testing.T) {
	var _ agent.Agent = (*LoopAgent)(nil)
}
