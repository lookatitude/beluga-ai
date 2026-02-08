package llm

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/schema"
)

func TestComposeHooks_BeforeGenerate_Order(t *testing.T) {
	var order []int
	h1 := Hooks{
		BeforeGenerate: func(ctx context.Context, msgs []schema.Message) error {
			order = append(order, 1)
			return nil
		},
	}
	h2 := Hooks{
		BeforeGenerate: func(ctx context.Context, msgs []schema.Message) error {
			order = append(order, 2)
			return nil
		},
	}

	composed := ComposeHooks(h1, h2)
	err := composed.BeforeGenerate(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Errorf("expected order [1, 2], got %v", order)
	}
}

func TestComposeHooks_BeforeGenerate_ShortCircuits(t *testing.T) {
	sentinel := errors.New("blocked")
	var called bool

	h1 := Hooks{
		BeforeGenerate: func(ctx context.Context, msgs []schema.Message) error {
			return sentinel
		},
	}
	h2 := Hooks{
		BeforeGenerate: func(ctx context.Context, msgs []schema.Message) error {
			called = true
			return nil
		},
	}

	composed := ComposeHooks(h1, h2)
	err := composed.BeforeGenerate(context.Background(), nil)
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got %v", err)
	}
	if called {
		t.Error("h2 should not have been called after h1 returned an error")
	}
}

func TestComposeHooks_AfterGenerate_Order(t *testing.T) {
	var order []int
	h1 := Hooks{
		AfterGenerate: func(ctx context.Context, resp *schema.AIMessage, err error) {
			order = append(order, 1)
		},
	}
	h2 := Hooks{
		AfterGenerate: func(ctx context.Context, resp *schema.AIMessage, err error) {
			order = append(order, 2)
		},
	}

	composed := ComposeHooks(h1, h2)
	resp := &schema.AIMessage{}
	composed.AfterGenerate(context.Background(), resp, nil)

	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Errorf("expected order [1, 2], got %v", order)
	}
}

func TestComposeHooks_AfterGenerate_WithError(t *testing.T) {
	var gotResp *schema.AIMessage
	var gotErr error
	h := Hooks{
		AfterGenerate: func(ctx context.Context, resp *schema.AIMessage, err error) {
			gotResp = resp
			gotErr = err
		},
	}

	composed := ComposeHooks(h)
	testErr := errors.New("test error")
	composed.AfterGenerate(context.Background(), nil, testErr)

	if gotResp != nil {
		t.Error("expected nil response")
	}
	if !errors.Is(gotErr, testErr) {
		t.Errorf("expected test error, got %v", gotErr)
	}
}

func TestComposeHooks_OnStream_Order(t *testing.T) {
	var order []int
	h1 := Hooks{
		OnStream: func(ctx context.Context, chunk schema.StreamChunk) {
			order = append(order, 1)
		},
	}
	h2 := Hooks{
		OnStream: func(ctx context.Context, chunk schema.StreamChunk) {
			order = append(order, 2)
		},
	}

	composed := ComposeHooks(h1, h2)
	composed.OnStream(context.Background(), schema.StreamChunk{Delta: "hello"})

	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Errorf("expected order [1, 2], got %v", order)
	}
}

func TestComposeHooks_OnStream_ReceivesChunk(t *testing.T) {
	var gotDelta string
	h := Hooks{
		OnStream: func(ctx context.Context, chunk schema.StreamChunk) {
			gotDelta = chunk.Delta
		},
	}

	composed := ComposeHooks(h)
	composed.OnStream(context.Background(), schema.StreamChunk{Delta: "test-delta"})

	if gotDelta != "test-delta" {
		t.Errorf("expected delta %q, got %q", "test-delta", gotDelta)
	}
}

func TestComposeHooks_OnToolCall_Order(t *testing.T) {
	var calls []string
	h1 := Hooks{
		OnToolCall: func(ctx context.Context, call schema.ToolCall) {
			calls = append(calls, "h1:"+call.Name)
		},
	}
	h2 := Hooks{
		OnToolCall: func(ctx context.Context, call schema.ToolCall) {
			calls = append(calls, "h2:"+call.Name)
		},
	}

	composed := ComposeHooks(h1, h2)
	composed.OnToolCall(context.Background(), schema.ToolCall{Name: "search"})

	if len(calls) != 2 || calls[0] != "h1:search" || calls[1] != "h2:search" {
		t.Errorf("expected [h1:search, h2:search], got %v", calls)
	}
}

func TestComposeHooks_OnError_ShortCircuits(t *testing.T) {
	replacement := errors.New("replaced")
	var called bool

	h1 := Hooks{
		OnError: func(ctx context.Context, err error) error {
			return replacement
		},
	}
	h2 := Hooks{
		OnError: func(ctx context.Context, err error) error {
			called = true
			return err
		},
	}

	composed := ComposeHooks(h1, h2)
	result := composed.OnError(context.Background(), errors.New("original"))

	if !errors.Is(result, replacement) {
		t.Errorf("expected replacement error, got %v", result)
	}
	if called {
		t.Error("h2 should not have been called after h1 returned non-nil")
	}
}

func TestComposeHooks_OnError_PassesThrough(t *testing.T) {
	original := errors.New("original")

	h1 := Hooks{
		OnError: func(ctx context.Context, err error) error {
			return nil // suppresses
		},
	}
	h2 := Hooks{
		OnError: func(ctx context.Context, err error) error {
			return nil // suppresses
		},
	}

	composed := ComposeHooks(h1, h2)
	result := composed.OnError(context.Background(), original)

	// When all hooks return nil, the composed hook returns the original error.
	if !errors.Is(result, original) {
		t.Errorf("expected original error when all hooks return nil, got %v", result)
	}
}

func TestComposeHooks_NilHooksSkipped(t *testing.T) {
	var called bool
	h1 := Hooks{} // all nil hooks
	h2 := Hooks{
		BeforeGenerate: func(ctx context.Context, msgs []schema.Message) error {
			called = true
			return nil
		},
	}

	composed := ComposeHooks(h1, h2)
	err := composed.BeforeGenerate(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("h2.BeforeGenerate should have been called")
	}
}

func TestComposeHooks_Empty(t *testing.T) {
	composed := ComposeHooks()

	// All callbacks should be non-nil and function safely.
	if err := composed.BeforeGenerate(context.Background(), nil); err != nil {
		t.Errorf("BeforeGenerate on empty compose returned error: %v", err)
	}
	// AfterGenerate, OnStream, OnToolCall should not panic.
	composed.AfterGenerate(context.Background(), nil, nil)
	composed.OnStream(context.Background(), schema.StreamChunk{})
	composed.OnToolCall(context.Background(), schema.ToolCall{})

	// OnError should return the original error.
	orig := errors.New("test")
	if result := composed.OnError(context.Background(), orig); !errors.Is(result, orig) {
		t.Errorf("OnError should return original error, got %v", result)
	}
}

func TestComposeHooks_ContextPropagated(t *testing.T) {
	type ctxKey string
	key := ctxKey("test-key")
	ctx := context.WithValue(context.Background(), key, "test-value")

	var gotValue string
	h := Hooks{
		BeforeGenerate: func(ctx context.Context, msgs []schema.Message) error {
			if v, ok := ctx.Value(key).(string); ok {
				gotValue = v
			}
			return nil
		},
	}

	composed := ComposeHooks(h)
	_ = composed.BeforeGenerate(ctx, nil)

	if gotValue != "test-value" {
		t.Errorf("expected context value %q, got %q", "test-value", gotValue)
	}
}

func TestComposeHooks_MessagesPropagated(t *testing.T) {
	var gotMsgs []schema.Message
	h := Hooks{
		BeforeGenerate: func(ctx context.Context, msgs []schema.Message) error {
			gotMsgs = msgs
			return nil
		},
	}

	msgs := []schema.Message{
		schema.NewHumanMessage("hello"),
		schema.NewSystemMessage("system"),
	}

	composed := ComposeHooks(h)
	_ = composed.BeforeGenerate(context.Background(), msgs)

	if len(gotMsgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(gotMsgs))
	}
}
