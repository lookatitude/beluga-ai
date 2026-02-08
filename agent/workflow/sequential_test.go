package workflow

import (
	"context"
	"errors"
	"iter"
	"testing"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/tool"
)

// seqMockAgent is a mock agent for sequential tests.
type seqMockAgent struct {
	id       string
	invokeFn func(ctx context.Context, input string) (string, error)
	streamFn func(ctx context.Context, input string) iter.Seq2[agent.Event, error]
}

func (a *seqMockAgent) ID() string                 { return a.id }
func (a *seqMockAgent) Persona() agent.Persona     { return agent.Persona{} }
func (a *seqMockAgent) Tools() []tool.Tool         { return nil }
func (a *seqMockAgent) Children() []agent.Agent    { return nil }
func (a *seqMockAgent) Invoke(ctx context.Context, input string, opts ...agent.Option) (string, error) {
	if a.invokeFn != nil {
		return a.invokeFn(ctx, input)
	}
	return a.id + ":" + input, nil
}
func (a *seqMockAgent) Stream(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
	if a.streamFn != nil {
		return a.streamFn(ctx, input)
	}
	return func(yield func(agent.Event, error) bool) {
		yield(agent.Event{Type: agent.EventText, Text: a.id + ":" + input, AgentID: a.id}, nil)
	}
}

var _ agent.Agent = (*seqMockAgent)(nil)

func TestSequentialAgent_Invoke_ChainsOutput(t *testing.T) {
	children := []agent.Agent{
		&seqMockAgent{
			id: "step1",
			invokeFn: func(ctx context.Context, input string) (string, error) {
				return input + " -> step1", nil
			},
		},
		&seqMockAgent{
			id: "step2",
			invokeFn: func(ctx context.Context, input string) (string, error) {
				return input + " -> step2", nil
			},
		},
		&seqMockAgent{
			id: "step3",
			invokeFn: func(ctx context.Context, input string) (string, error) {
				return input + " -> step3", nil
			},
		},
	}

	sa := NewSequentialAgent("pipeline", children)
	result, err := sa.Invoke(context.Background(), "start")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "start -> step1 -> step2 -> step3"
	if result != want {
		t.Errorf("result = %q, want %q", result, want)
	}
}

func TestSequentialAgent_Invoke_StopsOnError(t *testing.T) {
	step2Err := errors.New("step2 failed")
	step3Called := false

	children := []agent.Agent{
		&seqMockAgent{
			id: "step1",
			invokeFn: func(ctx context.Context, input string) (string, error) {
				return "ok", nil
			},
		},
		&seqMockAgent{
			id: "step2",
			invokeFn: func(ctx context.Context, input string) (string, error) {
				return "", step2Err
			},
		},
		&seqMockAgent{
			id: "step3",
			invokeFn: func(ctx context.Context, input string) (string, error) {
				step3Called = true
				return "ok", nil
			},
		},
	}

	sa := NewSequentialAgent("pipeline", children)
	_, err := sa.Invoke(context.Background(), "start")
	if err == nil {
		t.Fatal("expected error from step2")
	}
	if step3Called {
		t.Error("step3 should not have been called")
	}
}

func TestSequentialAgent_Invoke_EmptyChildren(t *testing.T) {
	sa := NewSequentialAgent("empty", nil)
	result, err := sa.Invoke(context.Background(), "input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "input" {
		t.Errorf("expected input passthrough, got %q", result)
	}
}

func TestSequentialAgent_ID(t *testing.T) {
	sa := NewSequentialAgent("my-seq", nil)
	if sa.ID() != "my-seq" {
		t.Errorf("ID() = %q, want %q", sa.ID(), "my-seq")
	}
}

func TestSequentialAgent_Persona(t *testing.T) {
	sa := NewSequentialAgent("seq", nil)
	p := sa.Persona()
	if p.Role == "" {
		t.Error("expected non-empty persona role")
	}
}

func TestSequentialAgent_Tools_ReturnsNil(t *testing.T) {
	sa := NewSequentialAgent("seq", nil)
	if sa.Tools() != nil {
		t.Error("expected nil tools")
	}
}

func TestSequentialAgent_Children_ReturnsChildren(t *testing.T) {
	children := []agent.Agent{
		&seqMockAgent{id: "a"},
		&seqMockAgent{id: "b"},
	}
	sa := NewSequentialAgent("seq", children)
	got := sa.Children()
	if len(got) != 2 {
		t.Fatalf("expected 2 children, got %d", len(got))
	}
}

func TestSequentialAgent_Stream_ChainsOutput(t *testing.T) {
	children := []agent.Agent{
		&seqMockAgent{
			id: "s1",
			streamFn: func(ctx context.Context, input string) iter.Seq2[agent.Event, error] {
				return func(yield func(agent.Event, error) bool) {
					yield(agent.Event{Type: agent.EventText, Text: input + "+s1", AgentID: "s1"}, nil)
				}
			},
		},
		&seqMockAgent{
			id: "s2",
			streamFn: func(ctx context.Context, input string) iter.Seq2[agent.Event, error] {
				return func(yield func(agent.Event, error) bool) {
					yield(agent.Event{Type: agent.EventText, Text: input + "+s2", AgentID: "s2"}, nil)
				}
			},
		},
	}

	sa := NewSequentialAgent("pipeline", children)
	var texts []string
	var doneCount int
	for event, err := range sa.Stream(context.Background(), "start") {
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

	// s1 gets "start", produces "start+s1"
	// s2 gets "start+s1" (from text accumulation), produces "start+s1+s2"
	if len(texts) < 2 {
		t.Fatalf("expected at least 2 text events, got %d: %v", len(texts), texts)
	}
	if doneCount != 1 {
		t.Errorf("expected 1 done event, got %d", doneCount)
	}
}

func TestSequentialAgent_Stream_Error(t *testing.T) {
	children := []agent.Agent{
		&seqMockAgent{
			id: "fail",
			streamFn: func(ctx context.Context, input string) iter.Seq2[agent.Event, error] {
				return func(yield func(agent.Event, error) bool) {
					yield(agent.Event{}, errors.New("stream failed"))
				}
			},
		},
	}

	sa := NewSequentialAgent("pipeline", children)
	var gotErr error
	for _, err := range sa.Stream(context.Background(), "start") {
		if err != nil {
			gotErr = err
			break
		}
	}
	if gotErr == nil {
		t.Fatal("expected error from stream")
	}
}
