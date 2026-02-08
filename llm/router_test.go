package llm

import (
	"context"
	"iter"
	"testing"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/schema"
)

func TestNewRouter_DefaultStrategy(t *testing.T) {
	r := NewRouter()
	if r.strategy == nil {
		t.Fatal("expected default strategy to be set")
	}
	if r.ModelID() != "router" {
		t.Errorf("ModelID() = %q, want %q", r.ModelID(), "router")
	}
}

func TestRouter_NoModels(t *testing.T) {
	r := NewRouter()

	_, err := r.Generate(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error when no models configured")
	}
}

func TestRoundRobin_CyclesThroughModels(t *testing.T) {
	models := []ChatModel{
		&stubModel{id: "a"},
		&stubModel{id: "b"},
		&stubModel{id: "c"},
	}

	r := NewRouter(
		WithModels(models...),
		WithStrategy(&RoundRobin{}),
	)

	// Call Generate multiple times and verify round-robin.
	expected := []string{"a", "b", "c", "a", "b"}
	for i, want := range expected {
		resp, err := r.Generate(context.Background(), nil)
		if err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
		got := resp.ModelID
		if got != want {
			t.Errorf("call %d: ModelID = %q, want %q", i, got, want)
		}
	}
}

func TestRoundRobin_EmptyModels(t *testing.T) {
	rr := &RoundRobin{}
	_, err := rr.Select(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error for empty models")
	}
}

func TestFailoverChain_ReturnsFirstModel(t *testing.T) {
	models := []ChatModel{
		&stubModel{id: "primary"},
		&stubModel{id: "secondary"},
	}
	fc := &FailoverChain{}
	model, err := fc.Select(context.Background(), models, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model.ModelID() != "primary" {
		t.Errorf("expected primary, got %q", model.ModelID())
	}
}

func TestFailoverChain_EmptyModels(t *testing.T) {
	fc := &FailoverChain{}
	_, err := fc.Select(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error for empty models")
	}
}

func TestFailoverRouter_Generate_FailsOverOnRetryable(t *testing.T) {
	retryableErr := core.NewError("test", core.ErrProviderDown, "down", nil)

	models := []ChatModel{
		&stubModel{
			id: "failing",
			generateFn: func(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) {
				return nil, retryableErr
			},
		},
		&stubModel{id: "backup"},
	}

	fr := NewFailoverRouter(models...)
	resp, err := fr.Generate(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ModelID != "backup" {
		t.Errorf("expected backup model, got %q", resp.ModelID)
	}
}

func TestFailoverRouter_Generate_StopsOnNonRetryable(t *testing.T) {
	nonRetryable := core.NewError("test", core.ErrAuth, "auth", nil)

	models := []ChatModel{
		&stubModel{
			id: "failing",
			generateFn: func(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) {
				return nil, nonRetryable
			},
		},
		&stubModel{id: "backup"},
	}

	fr := NewFailoverRouter(models...)
	_, err := fr.Generate(context.Background(), nil)
	if err == nil {
		t.Fatal("expected non-retryable error to stop failover")
	}
}

func TestFailoverRouter_Generate_AllFail(t *testing.T) {
	retryableErr := core.NewError("test", core.ErrTimeout, "timeout", nil)

	models := []ChatModel{
		&stubModel{
			id: "a",
			generateFn: func(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) {
				return nil, retryableErr
			},
		},
		&stubModel{
			id: "b",
			generateFn: func(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) {
				return nil, retryableErr
			},
		},
	}

	fr := NewFailoverRouter(models...)
	_, err := fr.Generate(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error when all models fail")
	}
}

func TestFailoverRouter_Stream_FailsOver(t *testing.T) {
	retryableErr := core.NewError("test", core.ErrRateLimit, "rate limited", nil)

	models := []ChatModel{
		&stubModel{
			id: "failing",
			streamFn: func(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) iter.Seq2[schema.StreamChunk, error] {
				return func(yield func(schema.StreamChunk, error) bool) {
					yield(schema.StreamChunk{}, retryableErr)
				}
			},
		},
		&stubModel{
			id: "backup",
			streamFn: func(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) iter.Seq2[schema.StreamChunk, error] {
				return func(yield func(schema.StreamChunk, error) bool) {
					yield(schema.StreamChunk{Delta: "ok"}, nil)
				}
			},
		},
	}

	fr := NewFailoverRouter(models...)

	var deltas []string
	for chunk, err := range fr.Stream(context.Background(), nil) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		deltas = append(deltas, chunk.Delta)
	}
	if len(deltas) != 1 || deltas[0] != "ok" {
		t.Errorf("expected backup chunks, got: %v", deltas)
	}
}

func TestFailoverRouter_ModelID(t *testing.T) {
	fr := NewFailoverRouter(&stubModel{id: "a"})
	if fr.ModelID() != "failover-router" {
		t.Errorf("ModelID() = %q, want %q", fr.ModelID(), "failover-router")
	}
}

func TestFailoverRouter_BindTools(t *testing.T) {
	fr := NewFailoverRouter(&stubModel{id: "a"})
	tools := []schema.ToolDefinition{{Name: "test"}}
	bound := fr.BindTools(tools)
	if bound.ModelID() != "failover-router" {
		t.Errorf("expected failover-router, got %q", bound.ModelID())
	}
}

func TestRouter_Stream_NoModelsError(t *testing.T) {
	r := NewRouter()

	var gotErr error
	for _, err := range r.Stream(context.Background(), nil) {
		if err != nil {
			gotErr = err
			break
		}
	}
	if gotErr == nil {
		t.Fatal("expected error from stream with no models")
	}
}

func TestRouter_BindTools(t *testing.T) {
	models := []ChatModel{&stubModel{id: "a"}}
	r := NewRouter(WithModels(models...))
	tools := []schema.ToolDefinition{{Name: "test"}}
	bound := r.BindTools(tools)
	if bound.ModelID() != "router" {
		t.Errorf("expected router, got %q", bound.ModelID())
	}
}
