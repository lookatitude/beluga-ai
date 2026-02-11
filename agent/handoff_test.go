package agent

import (
	"context"
	"errors"
	"iter"
	"strings"
	"testing"

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

func TestHandoffsToTools_GeneratesCorrectNames(t *testing.T) {
	agents := []*mockAgent{
		{id: "researcher"},
		{id: "coder"},
		{id: "reviewer"},
	}

	handoffs := make([]Handoff, len(agents))
	for i, a := range agents {
		handoffs[i] = HandoffTo(a, "Transfer to "+a.id)
	}

	tools := HandoffsToTools(handoffs)
	if len(tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(tools))
	}

	expected := []string{
		"transfer_to_researcher",
		"transfer_to_coder",
		"transfer_to_reviewer",
	}
	for i, tool := range tools {
		if tool.Name() != expected[i] {
			t.Errorf("tool[%d].Name() = %q, want %q", i, tool.Name(), expected[i])
		}
	}
}

func TestHandoffsToTools_Description(t *testing.T) {
	tests := []struct {
		name    string
		handoff Handoff
		wantIn  string
	}{
		{
			name:    "custom description",
			handoff: HandoffTo(&mockAgent{id: "test"}, "Custom desc"),
			wantIn:  "Custom desc",
		},
		{
			name: "default description",
			handoff: Handoff{
				TargetAgent: &mockAgent{id: "myagent"},
			},
			wantIn: "Transfer the conversation to myagent.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tools := HandoffsToTools([]Handoff{tt.handoff})
			if len(tools) != 1 {
				t.Fatalf("expected 1 tool, got %d", len(tools))
			}
			if !strings.Contains(tools[0].Description(), tt.wantIn) {
				t.Errorf("Description() = %q, want to contain %q", tools[0].Description(), tt.wantIn)
			}
		})
	}
}

func TestHandoffsToTools_InputSchema(t *testing.T) {
	handoff := HandoffTo(&mockAgent{id: "test"}, "test")
	tools := HandoffsToTools([]Handoff{handoff})

	s := tools[0].InputSchema()
	if s == nil {
		t.Fatal("InputSchema() returned nil")
	}
	if s["type"] != "object" {
		t.Errorf("schema type = %v, want %q", s["type"], "object")
	}
	props, ok := s["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties is not map[string]any: %T", s["properties"])
	}
	if _, ok := props["message"]; !ok {
		t.Error("schema missing 'message' property")
	}
}

func TestHandoffsToTools_Execute(t *testing.T) {
	target := &mockAgent{id: "helper"}
	handoff := HandoffTo(target, "Get help")

	tools := HandoffsToTools([]Handoff{handoff})
	result, err := tools[0].Execute(context.Background(), map[string]any{
		"message": "please help",
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.IsError {
		t.Error("result should not be error")
	}
}

func TestHandoffsToTools_Execute_Disabled(t *testing.T) {
	target := &mockAgent{id: "helper"}
	handoff := Handoff{
		TargetAgent: target,
		Description: "disabled handoff",
		IsEnabled:   func(ctx context.Context) bool { return false },
	}

	tools := HandoffsToTools([]Handoff{handoff})
	result, err := tools[0].Execute(context.Background(), map[string]any{
		"message": "help",
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result when handoff is disabled")
	}
}

func TestHandoffsToTools_Execute_OnHandoffCallback(t *testing.T) {
	callbackCalled := false
	target := &mockAgent{id: "helper"}
	handoff := Handoff{
		TargetAgent: target,
		OnHandoff: func(ctx context.Context) error {
			callbackCalled = true
			return nil
		},
	}

	tools := HandoffsToTools([]Handoff{handoff})
	_, err := tools[0].Execute(context.Background(), map[string]any{"message": "hi"})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !callbackCalled {
		t.Error("OnHandoff callback was not called")
	}
}

func TestHandoffsToTools_Execute_InputFilter(t *testing.T) {
	target := &streamMockAgent{id: "filtered", result: "filtered-result"}
	handoff := Handoff{
		TargetAgent: target,
		InputFilter: func(input HandoffInput) HandoffInput {
			input.Message = "FILTERED: " + input.Message
			return input
		},
	}

	tools := HandoffsToTools([]Handoff{handoff})
	_, err := tools[0].Execute(context.Background(), map[string]any{"message": "hello"})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	// The filtered agent will receive "FILTERED: hello" as input.
}

func TestHandoffTo(t *testing.T) {
	target := &mockAgent{id: "test"}
	h := HandoffTo(target, "my description")

	if h.TargetAgent.ID() != "test" {
		t.Errorf("TargetAgent.ID() = %q, want %q", h.TargetAgent.ID(), "test")
	}
	if h.Description != "my description" {
		t.Errorf("Description = %q, want %q", h.Description, "my description")
	}
	if h.InputFilter != nil {
		t.Error("InputFilter should be nil for simple HandoffTo")
	}
	if h.OnHandoff != nil {
		t.Error("OnHandoff should be nil for simple HandoffTo")
	}
}

func TestIsHandoffTool(t *testing.T) {
	tests := []struct {
		name string
		call schema.ToolCall
		want bool
	}{
		{name: "handoff tool", call: schema.ToolCall{Name: "transfer_to_agent"}, want: true},
		{name: "not handoff", call: schema.ToolCall{Name: "search"}, want: false},
		{name: "empty name", call: schema.ToolCall{Name: ""}, want: false},
		{name: "just prefix", call: schema.ToolCall{Name: "transfer_to_"}, want: false},
		{name: "with id", call: schema.ToolCall{Name: "transfer_to_helper"}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsHandoffTool(tt.call); got != tt.want {
				t.Errorf("IsHandoffTool(%q) = %v, want %v", tt.call.Name, got, tt.want)
			}
		})
	}
}

func TestHandoffTargetID(t *testing.T) {
	tests := []struct {
		name string
		call schema.ToolCall
		want string
	}{
		{name: "valid handoff", call: schema.ToolCall{Name: "transfer_to_researcher"}, want: "researcher"},
		{name: "not a handoff", call: schema.ToolCall{Name: "search"}, want: ""},
		{name: "empty", call: schema.ToolCall{Name: ""}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HandoffTargetID(tt.call); got != tt.want {
				t.Errorf("HandoffTargetID(%q) = %q, want %q", tt.call.Name, got, tt.want)
			}
		})
	}
}

func TestParseHandoffInput(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		want    string
		wantErr bool
	}{
		{name: "valid", args: `{"message":"hello"}`, want: "hello"},
		{name: "empty message", args: `{"message":""}`, want: ""},
		{name: "missing message", args: `{}`, want: ""},
		{name: "invalid json", args: `not json`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseHandoffInput(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ParseHandoffInput(%q) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}

func TestWithHandoffContext(t *testing.T) {
	ctx := context.Background()
	data := map[string]any{"key": "value", "count": 42}

	ctx = WithHandoffContext(ctx, data)

	got := ctx.Value(handoffContextKey{})
	if got == nil {
		t.Fatal("expected non-nil context value")
	}
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", got)
	}
	if m["key"] != "value" {
		t.Errorf("expected key=value, got %v", m["key"])
	}
	if m["count"] != 42 {
		t.Errorf("expected count=42, got %v", m["count"])
	}
}

func TestHandoffsToTools_Empty(t *testing.T) {
	tools := HandoffsToTools(nil)
	if len(tools) != 0 {
		t.Errorf("expected empty tools, got %d", len(tools))
	}

	tools = HandoffsToTools([]Handoff{})
	if len(tools) != 0 {
		t.Errorf("expected empty tools, got %d", len(tools))
	}
}

// TestHandoffsToTools_Execute_OnHandoffError tests OnHandoff error path.
func TestHandoffsToTools_Execute_OnHandoffError(t *testing.T) {
	target := &mockAgent{id: "helper"}
	handoff := Handoff{
		TargetAgent: target,
		OnHandoff: func(ctx context.Context) error {
			return errors.New("handoff callback failed")
		},
	}

	tools := HandoffsToTools([]Handoff{handoff})
	result, err := tools[0].Execute(context.Background(), map[string]any{"message": "test"})

	if err == nil {
		t.Fatal("expected error from OnHandoff callback failure")
	}
	if err.Error() != "handoff callback failed" {
		t.Errorf("error = %q, want %q", err.Error(), "handoff callback failed")
	}
	if result != nil {
		t.Error("expected nil result on callback error")
	}
}

// TestHandoffsToTools_Execute_TargetInvokeError tests target agent invoke error.
func TestHandoffsToTools_Execute_TargetInvokeError(t *testing.T) {
	target := &errorAgent{id: "failing", err: errors.New("invoke failed")}
	handoff := HandoffTo(target, "test")

	tools := HandoffsToTools([]Handoff{handoff})
	result, err := tools[0].Execute(context.Background(), map[string]any{"message": "test"})

	if err == nil {
		t.Fatal("expected error from target invoke failure")
	}
	if result != nil {
		t.Error("expected nil result on invoke error")
	}
}

// TestHandoffsToTools_Execute_WithHandoffContext tests handoff context usage.
func TestHandoffsToTools_Execute_WithHandoffContext(t *testing.T) {
	target := &streamMockAgent{id: "target", result: "ok"}

	var capturedInput HandoffInput
	handoff := Handoff{
		TargetAgent: target,
		InputFilter: func(input HandoffInput) HandoffInput {
			capturedInput = input
			return input
		},
	}

	tools := HandoffsToTools([]Handoff{handoff})

	ctx := WithHandoffContext(context.Background(), map[string]any{
		"source": "agent-1",
		"priority": "high",
	})

	_, err := tools[0].Execute(ctx, map[string]any{"message": "hello"})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if capturedInput.Context == nil {
		t.Fatal("expected context to be populated")
	}
	if capturedInput.Context["source"] != "agent-1" {
		t.Errorf("context source = %v, want %q", capturedInput.Context["source"], "agent-1")
	}
}

// errorAgent is a test agent that returns an error on Invoke.
type errorAgent struct {
	id  string
	err error
}

func (a *errorAgent) ID() string {
	return a.id
}

func (a *errorAgent) Invoke(ctx context.Context, input string, opts ...Option) (string, error) {
	return "", a.err
}

func (a *errorAgent) Stream(ctx context.Context, input string, opts ...Option) iter.Seq2[Event, error] {
	return func(yield func(Event, error) bool) {
		yield(Event{Type: EventError, AgentID: a.id}, a.err)
	}
}

func (a *errorAgent) Persona() Persona {
	return Persona{}
}

func (a *errorAgent) Tools() []tool.Tool {
	return nil
}

func (a *errorAgent) Children() []Agent {
	return nil
}

var _ Agent = (*errorAgent)(nil)
