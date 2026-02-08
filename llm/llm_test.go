package llm

import (
	"context"
	"errors"
	"iter"
	"testing"

	"github.com/lookatitude/beluga-ai/schema"
)

func TestChatModel_InterfaceCompliance(t *testing.T) {
	// Verify stubModel implements ChatModel at compile time.
	var _ ChatModel = (*stubModel)(nil)
}

func TestStubModel_Generate_Default(t *testing.T) {
	m := &stubModel{id: "test"}
	resp, err := m.Generate(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.ModelID != "test" {
		t.Errorf("ModelID = %q, want %q", resp.ModelID, "test")
	}
}

func TestStubModel_Generate_CustomFn(t *testing.T) {
	m := &stubModel{
		id: "custom",
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) {
			return &schema.AIMessage{
				Parts:   []schema.ContentPart{schema.TextPart{Text: "custom response"}},
				ModelID: "custom",
			}, nil
		},
	}

	resp, err := m.Generate(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Text() != "custom response" {
		t.Errorf("Text() = %q, want %q", resp.Text(), "custom response")
	}
}

func TestStubModel_Generate_Error(t *testing.T) {
	sentinel := errors.New("generate failed")
	m := &stubModel{
		id: "errmodel",
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) {
			return nil, sentinel
		},
	}

	_, err := m.Generate(context.Background(), nil)
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got %v", err)
	}
}

func TestStubModel_Stream_Default(t *testing.T) {
	m := &stubModel{id: "test"}

	var deltas []string
	for chunk, err := range m.Stream(context.Background(), nil) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		deltas = append(deltas, chunk.Delta)
	}

	if len(deltas) != 1 || deltas[0] != "hello" {
		t.Errorf("unexpected deltas: %v", deltas)
	}
}

func TestStubModel_Stream_CustomFn(t *testing.T) {
	m := &stubModel{
		id: "custom",
		streamFn: func(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) iter.Seq2[schema.StreamChunk, error] {
			return func(yield func(schema.StreamChunk, error) bool) {
				yield(schema.StreamChunk{Delta: "a"}, nil)
				yield(schema.StreamChunk{Delta: "b"}, nil)
				yield(schema.StreamChunk{Delta: "c"}, nil)
			}
		},
	}

	var deltas []string
	for chunk, err := range m.Stream(context.Background(), nil) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		deltas = append(deltas, chunk.Delta)
	}

	if len(deltas) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(deltas))
	}
	if deltas[0] != "a" || deltas[1] != "b" || deltas[2] != "c" {
		t.Errorf("unexpected deltas: %v", deltas)
	}
}

func TestStubModel_Stream_Error(t *testing.T) {
	sentinel := errors.New("stream failed")
	m := &stubModel{
		id: "errmodel",
		streamFn: func(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) iter.Seq2[schema.StreamChunk, error] {
			return func(yield func(schema.StreamChunk, error) bool) {
				yield(schema.StreamChunk{}, sentinel)
			}
		},
	}

	var gotErr error
	for _, err := range m.Stream(context.Background(), nil) {
		if err != nil {
			gotErr = err
			break
		}
	}
	if !errors.Is(gotErr, sentinel) {
		t.Errorf("expected sentinel error, got %v", gotErr)
	}
}

func TestStubModel_BindTools(t *testing.T) {
	m := &stubModel{id: "base"}
	bound := m.BindTools([]schema.ToolDefinition{{Name: "search"}})
	if bound == nil {
		t.Fatal("BindTools returned nil")
	}
	if bound.ModelID() != "base" {
		t.Errorf("ModelID = %q, want %q", bound.ModelID(), "base")
	}
}

func TestStubModel_ModelID(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"gpt-4o", "gpt-4o"},
		{"claude-sonnet", "claude-sonnet"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			m := &stubModel{id: tt.id}
			if m.ModelID() != tt.want {
				t.Errorf("ModelID() = %q, want %q", m.ModelID(), tt.want)
			}
		})
	}
}

func TestStubModel_Generate_ContextCancelled(t *testing.T) {
	m := &stubModel{
		id: "ctx-test",
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) {
			return nil, ctx.Err()
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := m.Generate(ctx, nil)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestStubModel_Generate_PassesMessages(t *testing.T) {
	var gotMsgs []schema.Message
	m := &stubModel{
		id: "msg-test",
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) {
			gotMsgs = msgs
			return &schema.AIMessage{}, nil
		},
	}

	msgs := []schema.Message{
		schema.NewSystemMessage("sys"),
		schema.NewHumanMessage("hello"),
	}
	_, _ = m.Generate(context.Background(), msgs)

	if len(gotMsgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(gotMsgs))
	}
	if gotMsgs[0].GetRole() != schema.RoleSystem {
		t.Errorf("first message role = %q, want %q", gotMsgs[0].GetRole(), schema.RoleSystem)
	}
	if gotMsgs[1].GetRole() != schema.RoleHuman {
		t.Errorf("second message role = %q, want %q", gotMsgs[1].GetRole(), schema.RoleHuman)
	}
}

func TestStubModel_Generate_PassesOptions(t *testing.T) {
	var gotOpts []GenerateOption
	m := &stubModel{
		id: "opt-test",
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) {
			gotOpts = opts
			return &schema.AIMessage{}, nil
		},
	}

	opts := []GenerateOption{WithMaxTokens(100), WithTemperature(0.7)}
	_, _ = m.Generate(context.Background(), nil, opts...)

	if len(gotOpts) != 2 {
		t.Fatalf("expected 2 options, got %d", len(gotOpts))
	}
}

func TestStubModel_Generate_NilMessages(t *testing.T) {
	m := &stubModel{id: "nil-test"}
	resp, err := m.Generate(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response even with nil messages")
	}
}
