package agent

import (
	"context"
	"errors"
	"iter"
	"testing"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

// testLLM is a mock ChatModel that satisfies llm.ChatModel for agent tests.
// BindTools returns llm.ChatModel (not a concrete type) so the interface is satisfied.
type testLLM struct {
	generateFn func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error)
	boundTools []schema.ToolDefinition
}

func (m *testLLM) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
	if m.generateFn != nil {
		return m.generateFn(ctx, msgs)
	}
	return schema.NewAIMessage("default response"), nil
}

func (m *testLLM) Stream(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	return func(yield func(schema.StreamChunk, error) bool) {
		resp, err := m.Generate(ctx, msgs, opts...)
		if err != nil {
			yield(schema.StreamChunk{}, err)
			return
		}
		yield(schema.StreamChunk{Delta: resp.Text()}, nil)
	}
}

func (m *testLLM) BindTools(tools []schema.ToolDefinition) llm.ChatModel {
	return &testLLM{
		generateFn: m.generateFn,
		boundTools: tools,
	}
}

func (m *testLLM) ModelID() string { return "test-model" }

var _ llm.ChatModel = (*testLLM)(nil)

func TestNew_BasicAgent(t *testing.T) {
	a := New("test-agent")
	if a == nil {
		t.Fatal("expected non-nil agent")
	}
	if a.ID() != "test-agent" {
		t.Errorf("ID() = %q, want %q", a.ID(), "test-agent")
	}
}

func TestNew_WithOptions(t *testing.T) {
	model := &testLLM{}
	tools := []tool.Tool{&simpleTool{toolName: "calc"}}
	persona := Persona{Role: "helper", Goal: "assist users"}

	a := New("configured",
		WithLLM(model),
		WithTools(tools),
		WithPersona(persona),
		WithMaxIterations(5),
	)

	if a.ID() != "configured" {
		t.Errorf("ID() = %q, want %q", a.ID(), "configured")
	}
	if a.Persona().Role != "helper" {
		t.Errorf("Persona().Role = %q, want %q", a.Persona().Role, "helper")
	}
	if a.Persona().Goal != "assist users" {
		t.Errorf("Persona().Goal = %q, want %q", a.Persona().Goal, "assist users")
	}
	if len(a.Tools()) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(a.Tools()))
	}
	if a.Tools()[0].Name() != "calc" {
		t.Errorf("tool name = %q, want %q", a.Tools()[0].Name(), "calc")
	}
}

func TestBaseAgent_Tools_IncludesHandoffs(t *testing.T) {
	target := &mockAgent{id: "helper"}
	a := New("main",
		WithTools([]tool.Tool{&simpleTool{toolName: "search"}}),
		WithHandoffs([]Handoff{HandoffTo(target, "help")}),
	)

	tools := a.Tools()
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools (1 tool + 1 handoff), got %d", len(tools))
	}
	if tools[0].Name() != "search" {
		t.Errorf("tools[0] = %q, want %q", tools[0].Name(), "search")
	}
	if tools[1].Name() != "transfer_to_helper" {
		t.Errorf("tools[1] = %q, want %q", tools[1].Name(), "transfer_to_helper")
	}
}

func TestBaseAgent_Tools_NoHandoffs(t *testing.T) {
	a := New("simple",
		WithTools([]tool.Tool{&simpleTool{toolName: "calc"}}),
	)
	tools := a.Tools()
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
}

func TestBaseAgent_Tools_Empty(t *testing.T) {
	a := New("empty")
	tools := a.Tools()
	if len(tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(tools))
	}
}

func TestBaseAgent_Children(t *testing.T) {
	children := []Agent{&mockAgent{id: "c1"}, &mockAgent{id: "c2"}}
	a := New("parent", WithChildren(children))

	got := a.Children()
	if len(got) != 2 {
		t.Fatalf("expected 2 children, got %d", len(got))
	}
	if got[0].ID() != "c1" {
		t.Errorf("children[0].ID() = %q, want %q", got[0].ID(), "c1")
	}
}

func TestBaseAgent_Children_Empty(t *testing.T) {
	a := New("no-children")
	if a.Children() != nil {
		t.Errorf("expected nil children, got %v", a.Children())
	}
}

func TestBaseAgent_Config(t *testing.T) {
	a := New("test", WithMaxIterations(7))
	cfg := a.Config()
	if cfg.maxIterations != 7 {
		t.Errorf("maxIterations = %d, want 7", cfg.maxIterations)
	}
}

func TestBaseAgent_Invoke_SimpleResponse(t *testing.T) {
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return schema.NewAIMessage("Hello, world!"), nil
		},
	}

	a := New("greeter",
		WithLLM(model),
		WithPlanner(NewReActPlanner(model)),
	)

	result, err := a.Invoke(context.Background(), "Hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Hello, world!" {
		t.Errorf("result = %q, want %q", result, "Hello, world!")
	}
}

func TestBaseAgent_Invoke_NoLLM_Error(t *testing.T) {
	a := New("no-llm")
	_, err := a.Invoke(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error when no LLM configured")
	}
}

func TestBaseAgent_Invoke_LLMError(t *testing.T) {
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return nil, errors.New("LLM failed")
		},
	}

	a := New("error-agent",
		WithLLM(model),
		WithPlanner(NewReActPlanner(model)),
	)

	_, err := a.Invoke(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error from LLM")
	}
}

func TestBaseAgent_Invoke_WithPersona(t *testing.T) {
	var receivedMsgs []schema.Message
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			receivedMsgs = msgs
			return schema.NewAIMessage("ok"), nil
		},
	}

	a := New("persona-agent",
		WithLLM(model),
		WithPlanner(NewReActPlanner(model)),
		WithPersona(Persona{Role: "tester"}),
	)

	_, _ = a.Invoke(context.Background(), "hello")

	if len(receivedMsgs) < 2 {
		t.Fatalf("expected at least 2 messages (system + human), got %d", len(receivedMsgs))
	}
	if receivedMsgs[0].GetRole() != schema.RoleSystem {
		t.Errorf("first message role = %q, want %q", receivedMsgs[0].GetRole(), schema.RoleSystem)
	}
	if receivedMsgs[1].GetRole() != schema.RoleHuman {
		t.Errorf("second message role = %q, want %q", receivedMsgs[1].GetRole(), schema.RoleHuman)
	}
}

func TestBaseAgent_Stream_Events(t *testing.T) {
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return schema.NewAIMessage("streamed response"), nil
		},
	}

	a := New("stream-agent",
		WithLLM(model),
		WithPlanner(NewReActPlanner(model)),
	)

	var events []Event
	for event, err := range a.Stream(context.Background(), "test") {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		events = append(events, event)
	}

	// Should have at least a text event and a done event.
	hasText := false
	hasDone := false
	for _, e := range events {
		if e.Type == EventText {
			hasText = true
		}
		if e.Type == EventDone {
			hasDone = true
		}
	}
	if !hasText {
		t.Error("expected at least one EventText event")
	}
	if !hasDone {
		t.Error("expected EventDone event")
	}
}

func TestBaseAgent_Stream_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return schema.NewAIMessage("should not reach"), nil
		},
	}

	a := New("cancel-agent",
		WithLLM(model),
		WithPlanner(NewReActPlanner(model)),
	)

	var gotErr error
	for _, err := range a.Stream(ctx, "test") {
		if err != nil {
			gotErr = err
			break
		}
	}
	if gotErr == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestBaseAgent_Invoke_RuntimeOptions(t *testing.T) {
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return schema.NewAIMessage("done"), nil
		},
	}

	a := New("base",
		WithLLM(model),
		WithPlanner(NewReActPlanner(model)),
	)

	// Runtime option overrides max iterations (shouldn't affect simple response).
	result, err := a.Invoke(context.Background(), "hello", WithMaxIterations(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "done" {
		t.Errorf("result = %q, want %q", result, "done")
	}
}

func TestBaseAgent_ImplementsAgent(t *testing.T) {
	var _ Agent = (*BaseAgent)(nil)
}

// TestBaseAgent_Stream_WithRuntimeOptions tests Stream with runtime options.
func TestBaseAgent_Stream_WithRuntimeOptions(t *testing.T) {
	baseModel := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return schema.NewAIMessage("base response"), nil
		},
	}

	a := New("test-agent", WithLLM(baseModel), WithPlanner(NewReActPlanner(baseModel)))

	runtimeModel := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return schema.NewAIMessage("runtime response"), nil
		},
	}

	var finalText string
	// Runtime options need to include both LLM and planner
	for event, err := range a.Stream(context.Background(), "test", WithLLM(runtimeModel), WithPlanner(NewReActPlanner(runtimeModel))) {
		if err != nil {
			t.Fatalf("Stream error: %v", err)
		}
		if event.Type == EventText {
			finalText = event.Text
		}
	}

	if finalText != "runtime response" {
		t.Errorf("final text = %q, want %q", finalText, "runtime response")
	}
}

// TestBaseAgent_resolvePlanner_ByNameError tests resolvePlanner with invalid name.
func TestBaseAgent_resolvePlanner_ByNameError(t *testing.T) {
	model := &testLLM{}
	a := New("test-agent", WithLLM(model), WithPlannerName("nonexistent"))

	_, err := a.resolvePlanner(a.config)
	if err == nil {
		t.Fatal("expected error for nonexistent planner name")
	}
}

// TestBaseAgent_stream_ExecutorError tests stream with executor error.
func TestBaseAgent_stream_ExecutorError(t *testing.T) {
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return nil, errors.New("LLM failed")
		},
	}

	a := New("test-agent", WithLLM(model), WithPlanner(NewReActPlanner(model)))

	var gotErr error
	for _, err := range a.stream(context.Background(), "test", a.config) {
		if err != nil {
			gotErr = err
			break
		}
	}

	if gotErr == nil {
		t.Fatal("expected error from LLM failure in stream")
	}
}

// TestBaseAgent_stream_WithHandoffs tests stream with handoffs.
func TestBaseAgent_stream_WithHandoffs(t *testing.T) {
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return schema.NewAIMessage("response"), nil
		},
	}

	targetAgent := &mockAgent{id: "target"}
	a := New("test-agent",
		WithLLM(model),
		WithPlanner(NewReActPlanner(model)),
		WithHandoffs([]Handoff{HandoffTo(targetAgent, "transfer to target")}),
	)

	toolCount := len(a.Tools())
	if toolCount != 1 {
		t.Errorf("expected 1 tool (handoff), got %d", toolCount)
	}

	var gotEvent bool
	for event, err := range a.stream(context.Background(), "test", a.config) {
		if err != nil {
			t.Fatalf("Stream error: %v", err)
		}
		gotEvent = true
		if event.Type == EventDone {
			break
		}
	}

	if !gotEvent {
		t.Error("expected at least one event")
	}
}
