package llm

import (
	"context"
	"errors"
	"iter"
	"log/slog"
	"testing"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/schema"
)

// stubModel is a minimal ChatModel for testing.
type stubModel struct {
	id         string
	generateFn func(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error)
	streamFn   func(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) iter.Seq2[schema.StreamChunk, error]
}

func (m *stubModel) Generate(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) {
	if m.generateFn != nil {
		return m.generateFn(ctx, msgs, opts...)
	}
	return &schema.AIMessage{
		Parts:   []schema.ContentPart{schema.TextPart{Text: "stub response"}},
		ModelID: m.id,
	}, nil
}

func (m *stubModel) Stream(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	if m.streamFn != nil {
		return m.streamFn(ctx, msgs, opts...)
	}
	return func(yield func(schema.StreamChunk, error) bool) {
		yield(schema.StreamChunk{Delta: "hello"}, nil)
	}
}

func (m *stubModel) BindTools(tools []schema.ToolDefinition) ChatModel {
	return m
}

func (m *stubModel) ModelID() string { return m.id }

func TestApplyMiddleware_Order(t *testing.T) {
	var order []string

	mw1 := func(next ChatModel) ChatModel {
		return &stubModel{
			id: next.ModelID(),
			generateFn: func(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) {
				order = append(order, "mw1-before")
				resp, err := next.Generate(ctx, msgs, opts...)
				order = append(order, "mw1-after")
				return resp, err
			},
		}
	}

	mw2 := func(next ChatModel) ChatModel {
		return &stubModel{
			id: next.ModelID(),
			generateFn: func(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) {
				order = append(order, "mw2-before")
				resp, err := next.Generate(ctx, msgs, opts...)
				order = append(order, "mw2-after")
				return resp, err
			},
		}
	}

	base := &stubModel{id: "base"}
	wrapped := ApplyMiddleware(base, mw1, mw2)

	_, _ = wrapped.Generate(context.Background(), nil)

	// mw1 is outermost (first), mw2 is inner.
	want := []string{"mw1-before", "mw2-before", "mw2-after", "mw1-after"}
	if len(order) != len(want) {
		t.Fatalf("got %d calls, want %d: %v", len(order), len(want), order)
	}
	for i, v := range order {
		if v != want[i] {
			t.Errorf("order[%d] = %q, want %q", i, v, want[i])
		}
	}
}

func TestApplyMiddleware_NoMiddleware(t *testing.T) {
	base := &stubModel{id: "base"}
	result := ApplyMiddleware(base)
	if result.ModelID() != "base" {
		t.Errorf("expected base model, got %q", result.ModelID())
	}
}

func TestWithLogging(t *testing.T) {
	// Use a discard handler so logs don't pollute test output.
	logger := slog.New(slog.NewTextHandler(discardWriter{}, nil))
	base := &stubModel{id: "test-model"}

	wrapped := ApplyMiddleware(base, WithLogging(logger))

	// Generate should succeed and pass through.
	resp, err := wrapped.Generate(context.Background(), []schema.Message{schema.NewHumanMessage("hi")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	// Stream should succeed and pass through.
	var chunks []string
	for chunk, err := range wrapped.Stream(context.Background(), []schema.Message{schema.NewHumanMessage("hi")}) {
		if err != nil {
			t.Fatalf("unexpected stream error: %v", err)
		}
		chunks = append(chunks, chunk.Delta)
	}
	if len(chunks) != 1 || chunks[0] != "hello" {
		t.Errorf("unexpected chunks: %v", chunks)
	}
}

func TestWithLogging_Error(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(discardWriter{}, nil))
	base := &stubModel{
		id: "err-model",
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) {
			return nil, errors.New("model error")
		},
	}

	wrapped := ApplyMiddleware(base, WithLogging(logger))
	_, err := wrapped.Generate(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWithFallback_PrimarySucceeds(t *testing.T) {
	primary := &stubModel{id: "primary"}
	fallback := &stubModel{id: "fallback"}

	wrapped := ApplyMiddleware(primary, WithFallback(fallback))

	resp, err := wrapped.Generate(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ModelID != "primary" {
		t.Errorf("expected primary model response, got %q", resp.ModelID)
	}
}

func TestWithFallback_FallsBackOnRetryable(t *testing.T) {
	retryableErr := core.NewError("test", core.ErrProviderDown, "down", nil)
	primary := &stubModel{
		id: "primary",
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) {
			return nil, retryableErr
		},
	}
	fallback := &stubModel{id: "fallback"}

	wrapped := ApplyMiddleware(primary, WithFallback(fallback))

	resp, err := wrapped.Generate(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ModelID != "fallback" {
		t.Errorf("expected fallback model response, got %q", resp.ModelID)
	}
}

func TestWithFallback_NoFallbackOnNonRetryable(t *testing.T) {
	nonRetryableErr := core.NewError("test", core.ErrAuth, "auth failed", nil)
	primary := &stubModel{
		id: "primary",
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) {
			return nil, nonRetryableErr
		},
	}
	fallback := &stubModel{id: "fallback"}

	wrapped := ApplyMiddleware(primary, WithFallback(fallback))

	_, err := wrapped.Generate(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, nonRetryableErr) {
		t.Errorf("expected non-retryable error to pass through, got: %v", err)
	}
}

func TestWithFallback_StreamFallback(t *testing.T) {
	retryableErr := core.NewError("test", core.ErrRateLimit, "rate limited", nil)
	primary := &stubModel{
		id: "primary",
		streamFn: func(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) iter.Seq2[schema.StreamChunk, error] {
			return func(yield func(schema.StreamChunk, error) bool) {
				yield(schema.StreamChunk{}, retryableErr)
			}
		},
	}
	fallback := &stubModel{
		id: "fallback",
		streamFn: func(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) iter.Seq2[schema.StreamChunk, error] {
			return func(yield func(schema.StreamChunk, error) bool) {
				yield(schema.StreamChunk{Delta: "fallback-chunk"}, nil)
			}
		},
	}

	wrapped := ApplyMiddleware(primary, WithFallback(fallback))

	var deltas []string
	for chunk, err := range wrapped.Stream(context.Background(), nil) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		deltas = append(deltas, chunk.Delta)
	}
	if len(deltas) != 1 || deltas[0] != "fallback-chunk" {
		t.Errorf("expected fallback chunks, got: %v", deltas)
	}
}

func TestWithHooks_BeforeGenerateAborts(t *testing.T) {
	hooks := Hooks{
		BeforeGenerate: func(ctx context.Context, msgs []schema.Message) error {
			return errors.New("blocked by hook")
		},
	}
	base := &stubModel{id: "base"}
	wrapped := ApplyMiddleware(base, WithHooks(hooks))

	_, err := wrapped.Generate(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error from BeforeGenerate hook")
	}
	if err.Error() != "blocked by hook" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWithHooks_OnToolCallTriggered(t *testing.T) {
	var calls []string
	hooks := Hooks{
		OnToolCall: func(ctx context.Context, call schema.ToolCall) {
			calls = append(calls, call.Name)
		},
	}
	base := &stubModel{
		id: "base",
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) {
			return &schema.AIMessage{
				ToolCalls: []schema.ToolCall{
					{ID: "1", Name: "search"},
					{ID: "2", Name: "calculate"},
				},
			}, nil
		},
	}
	wrapped := ApplyMiddleware(base, WithHooks(hooks))

	_, _ = wrapped.Generate(context.Background(), nil)
	if len(calls) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(calls))
	}
	if calls[0] != "search" || calls[1] != "calculate" {
		t.Errorf("unexpected calls: %v", calls)
	}
}

func TestWithHooks_StreamBeforeGenerateAborts(t *testing.T) {
	hooks := Hooks{
		BeforeGenerate: func(ctx context.Context, msgs []schema.Message) error {
			return errors.New("stream blocked")
		},
	}
	base := &stubModel{id: "base"}
	wrapped := ApplyMiddleware(base, WithHooks(hooks))

	var gotErr error
	for _, err := range wrapped.Stream(context.Background(), nil) {
		if err != nil {
			gotErr = err
			break
		}
	}
	if gotErr == nil {
		t.Fatal("expected error from stream BeforeGenerate hook")
	}
}

func TestBindTools_MiddlewarePreserved(t *testing.T) {
	base := &stubModel{id: "base"}
	logged := ApplyMiddleware(base, WithLogging(slog.New(slog.NewTextHandler(discardWriter{}, nil))))

	bound := logged.BindTools([]schema.ToolDefinition{{Name: "test"}})
	if bound == nil {
		t.Fatal("BindTools returned nil")
	}
	if bound.ModelID() != "base" {
		t.Errorf("expected ModelID %q, got %q", "base", bound.ModelID())
	}
}

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }
