package tool

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/schema"
)

func TestTextResult(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{name: "simple text", text: "hello"},
		{name: "empty text", text: ""},
		{name: "multiline", text: "line1\nline2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := TextResult(tt.text)
			if r.IsError {
				t.Error("TextResult should not set IsError")
			}
			if len(r.Content) != 1 {
				t.Fatalf("expected 1 content part, got %d", len(r.Content))
			}
			tp, ok := r.Content[0].(schema.TextPart)
			if !ok {
				t.Fatalf("expected TextPart, got %T", r.Content[0])
			}
			if tp.Text != tt.text {
				t.Errorf("text = %q, want %q", tp.Text, tt.text)
			}
		})
	}
}

func TestErrorResult(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantMsg string
	}{
		{name: "simple error", err: errors.New("something went wrong"), wantMsg: "something went wrong"},
		{name: "wrapped error", err: errors.New("wrapped: inner"), wantMsg: "wrapped: inner"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := ErrorResult(tt.err)
			if !r.IsError {
				t.Error("ErrorResult should set IsError to true")
			}
			if len(r.Content) != 1 {
				t.Fatalf("expected 1 content part, got %d", len(r.Content))
			}
			tp, ok := r.Content[0].(schema.TextPart)
			if !ok {
				t.Fatalf("expected TextPart, got %T", r.Content[0])
			}
			if tp.Text != tt.wantMsg {
				t.Errorf("text = %q, want %q", tp.Text, tt.wantMsg)
			}
		})
	}
}

func TestToDefinition(t *testing.T) {
	tool := &mockTool{
		name:        "search",
		description: "Search the web",
		inputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string"},
			},
		},
	}

	def := ToDefinition(tool)
	if def.Name != "search" {
		t.Errorf("Name = %q, want %q", def.Name, "search")
	}
	if def.Description != "Search the web" {
		t.Errorf("Description = %q, want %q", def.Description, "Search the web")
	}
	if def.InputSchema == nil {
		t.Error("InputSchema should not be nil")
	}
}

// mockTool implements Tool for testing.
type mockTool struct {
	name        string
	description string
	inputSchema map[string]any
	executeFn   func(input map[string]any) (*Result, error)
	executeCtxFn func(ctx context.Context, input map[string]any) (*Result, error)
}

func (m *mockTool) Name() string              { return m.name }
func (m *mockTool) Description() string        { return m.description }
func (m *mockTool) InputSchema() map[string]any { return m.inputSchema }
func (m *mockTool) Execute(ctx context.Context, input map[string]any) (*Result, error) {
	if m.executeCtxFn != nil {
		return m.executeCtxFn(ctx, input)
	}
	if m.executeFn != nil {
		return m.executeFn(input)
	}
	return TextResult("mock result"), nil
}
