package workflow

import (
	"context"
	"errors"
	"iter"
	"strings"
	"sync"
	"testing"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/tool"
)

// parMockAgent is a mock agent for parallel tests.
type parMockAgent struct {
	id       string
	invokeFn func(ctx context.Context, input string) (string, error)
	streamFn func(ctx context.Context, input string) iter.Seq2[agent.Event, error]
}

func (a *parMockAgent) ID() string                 { return a.id }
func (a *parMockAgent) Persona() agent.Persona     { return agent.Persona{} }
func (a *parMockAgent) Tools() []tool.Tool         { return nil }
func (a *parMockAgent) Children() []agent.Agent    { return nil }
func (a *parMockAgent) Invoke(ctx context.Context, input string, opts ...agent.Option) (string, error) {
	if a.invokeFn != nil {
		return a.invokeFn(ctx, input)
	}
	return a.id + ":" + input, nil
}
func (a *parMockAgent) Stream(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
	if a.streamFn != nil {
		return a.streamFn(ctx, input)
	}
	return func(yield func(agent.Event, error) bool) {
		yield(agent.Event{Type: agent.EventText, Text: a.id + ":" + input, AgentID: a.id}, nil)
	}
}

var _ agent.Agent = (*parMockAgent)(nil)

func TestParallelAgent_Invoke_AllChildrenRun(t *testing.T) {
	children := []agent.Agent{
		&parMockAgent{
			id: "a",
			invokeFn: func(ctx context.Context, input string) (string, error) {
				return "result-a", nil
			},
		},
		&parMockAgent{
			id: "b",
			invokeFn: func(ctx context.Context, input string) (string, error) {
				return "result-b", nil
			},
		},
	}

	pa := NewParallelAgent("parallel", children)
	result, err := pa.Invoke(context.Background(), "input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Results are joined with newline.
	if !strings.Contains(result, "result-a") {
		t.Errorf("result should contain 'result-a', got: %q", result)
	}
	if !strings.Contains(result, "result-b") {
		t.Errorf("result should contain 'result-b', got: %q", result)
	}
}

func TestParallelAgent_Invoke_ErrorFromChild(t *testing.T) {
	children := []agent.Agent{
		&parMockAgent{
			id: "ok",
			invokeFn: func(ctx context.Context, input string) (string, error) {
				return "ok", nil
			},
		},
		&parMockAgent{
			id: "fail",
			invokeFn: func(ctx context.Context, input string) (string, error) {
				return "", errors.New("child failed")
			},
		},
	}

	pa := NewParallelAgent("parallel", children)
	_, err := pa.Invoke(context.Background(), "input")
	if err == nil {
		t.Fatal("expected error from failing child")
	}
	if !strings.Contains(err.Error(), "fail") {
		t.Errorf("error should mention the failing child, got: %v", err)
	}
}

func TestParallelAgent_Invoke_EmptyChildren(t *testing.T) {
	pa := NewParallelAgent("empty", nil)
	result, err := pa.Invoke(context.Background(), "input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty result, got %q", result)
	}
}

func TestParallelAgent_Invoke_AllChildrenGetSameInput(t *testing.T) {
	var mu sync.Mutex
	var inputs []string
	children := []agent.Agent{
		&parMockAgent{
			id: "a",
			invokeFn: func(ctx context.Context, input string) (string, error) {
				mu.Lock()
				inputs = append(inputs, input)
				mu.Unlock()
				return "a", nil
			},
		},
		&parMockAgent{
			id: "b",
			invokeFn: func(ctx context.Context, input string) (string, error) {
				mu.Lock()
				inputs = append(inputs, input)
				mu.Unlock()
				return "b", nil
			},
		},
	}

	pa := NewParallelAgent("parallel", children)
	_, _ = pa.Invoke(context.Background(), "shared-input")

	mu.Lock()
	defer mu.Unlock()
	for _, in := range inputs {
		if in != "shared-input" {
			t.Errorf("expected 'shared-input', got %q", in)
		}
	}
}

func TestParallelAgent_ID(t *testing.T) {
	pa := NewParallelAgent("my-par", nil)
	if pa.ID() != "my-par" {
		t.Errorf("ID() = %q, want %q", pa.ID(), "my-par")
	}
}

func TestParallelAgent_Persona(t *testing.T) {
	pa := NewParallelAgent("par", nil)
	p := pa.Persona()
	if p.Role == "" {
		t.Error("expected non-empty persona role")
	}
}

func TestParallelAgent_Tools_ReturnsNil(t *testing.T) {
	pa := NewParallelAgent("par", nil)
	if pa.Tools() != nil {
		t.Error("expected nil tools")
	}
}

func TestParallelAgent_Children_ReturnsChildren(t *testing.T) {
	children := []agent.Agent{
		&parMockAgent{id: "a"},
		&parMockAgent{id: "b"},
	}
	pa := NewParallelAgent("par", children)
	got := pa.Children()
	if len(got) != 2 {
		t.Fatalf("expected 2 children, got %d", len(got))
	}
}

func TestParallelAgent_Stream_ReceivesEventsFromAllChildren(t *testing.T) {
	children := []agent.Agent{
		&parMockAgent{
			id: "a",
			streamFn: func(ctx context.Context, input string) iter.Seq2[agent.Event, error] {
				return func(yield func(agent.Event, error) bool) {
					yield(agent.Event{Type: agent.EventText, Text: "from-a", AgentID: "a"}, nil)
				}
			},
		},
		&parMockAgent{
			id: "b",
			streamFn: func(ctx context.Context, input string) iter.Seq2[agent.Event, error] {
				return func(yield func(agent.Event, error) bool) {
					yield(agent.Event{Type: agent.EventText, Text: "from-b", AgentID: "b"}, nil)
				}
			},
		},
	}

	pa := NewParallelAgent("parallel", children)
	texts := make(map[string]bool)
	var doneCount int
	for event, err := range pa.Stream(context.Background(), "input") {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if event.Type == agent.EventText {
			texts[event.Text] = true
		}
		if event.Type == agent.EventDone {
			doneCount++
		}
	}

	if !texts["from-a"] {
		t.Error("missing text event from agent a")
	}
	if !texts["from-b"] {
		t.Error("missing text event from agent b")
	}
	if doneCount != 1 {
		t.Errorf("expected 1 done event, got %d", doneCount)
	}
}

func TestParallelAgent_Stream_ErrorPropagation(t *testing.T) {
	children := []agent.Agent{
		&parMockAgent{
			id: "fail",
			streamFn: func(ctx context.Context, input string) iter.Seq2[agent.Event, error] {
				return func(yield func(agent.Event, error) bool) {
					yield(agent.Event{}, errors.New("stream error"))
				}
			},
		},
	}

	pa := NewParallelAgent("parallel", children)
	var gotErr error
	for _, err := range pa.Stream(context.Background(), "input") {
		if err != nil {
			gotErr = err
			break
		}
	}
	if gotErr == nil {
		t.Fatal("expected error from stream")
	}
}
