package embedding_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/embedding"
	_ "github.com/lookatitude/beluga-ai/rag/embedding/providers/inmemory"
)

func TestRegistry(t *testing.T) {
	t.Run("list includes inmemory", func(t *testing.T) {
		names := embedding.List()
		found := false
		for _, n := range names {
			if n == "inmemory" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected 'inmemory' in List(), got %v", names)
		}
	})

	t.Run("new creates embedder", func(t *testing.T) {
		emb, err := embedding.New("inmemory", config.ProviderConfig{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if emb == nil {
			t.Fatal("expected non-nil embedder")
		}
		if emb.Dimensions() != 128 {
			t.Fatalf("expected 128 dimensions, got %d", emb.Dimensions())
		}
	})

	t.Run("new with custom dimensions", func(t *testing.T) {
		emb, err := embedding.New("inmemory", config.ProviderConfig{
			Options: map[string]any{"dimensions": float64(64)},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if emb.Dimensions() != 64 {
			t.Fatalf("expected 64 dimensions, got %d", emb.Dimensions())
		}
	})

	t.Run("new unknown provider", func(t *testing.T) {
		_, err := embedding.New("nonexistent", config.ProviderConfig{})
		if err == nil {
			t.Fatal("expected error for unknown provider")
		}
	})
}

func TestEmbed(t *testing.T) {
	emb, err := embedding.New("inmemory", config.ProviderConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()

	t.Run("batch embed", func(t *testing.T) {
		texts := []string{"hello", "world", "test"}
		vectors, err := emb.Embed(ctx, texts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(vectors) != 3 {
			t.Fatalf("expected 3 vectors, got %d", len(vectors))
		}
		for i, vec := range vectors {
			if len(vec) != 128 {
				t.Fatalf("vector %d: expected 128 dimensions, got %d", i, len(vec))
			}
		}
	})

	t.Run("embed single", func(t *testing.T) {
		vec, err := emb.EmbedSingle(ctx, "hello")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(vec) != 128 {
			t.Fatalf("expected 128 dimensions, got %d", len(vec))
		}
	})

	t.Run("deterministic", func(t *testing.T) {
		v1, _ := emb.EmbedSingle(ctx, "deterministic test")
		v2, _ := emb.EmbedSingle(ctx, "deterministic test")
		for i := range v1 {
			if v1[i] != v2[i] {
				t.Fatalf("embedding not deterministic at index %d: %f != %f", i, v1[i], v2[i])
			}
		}
	})

	t.Run("different texts produce different embeddings", func(t *testing.T) {
		v1, _ := emb.EmbedSingle(ctx, "hello")
		v2, _ := emb.EmbedSingle(ctx, "goodbye")
		same := true
		for i := range v1 {
			if v1[i] != v2[i] {
				same = false
				break
			}
		}
		if same {
			t.Fatal("expected different embeddings for different texts")
		}
	})

	t.Run("empty batch", func(t *testing.T) {
		vectors, err := emb.Embed(ctx, []string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(vectors) != 0 {
			t.Fatalf("expected 0 vectors, got %d", len(vectors))
		}
	})
}

func TestComposeHooks(t *testing.T) {
	var order []string
	h1 := embedding.Hooks{
		BeforeEmbed: func(_ context.Context, _ []string) error {
			order = append(order, "h1-before")
			return nil
		},
		AfterEmbed: func(_ context.Context, _ [][]float32, _ error) {
			order = append(order, "h1-after")
		},
	}
	h2 := embedding.Hooks{
		BeforeEmbed: func(_ context.Context, _ []string) error {
			order = append(order, "h2-before")
			return nil
		},
		AfterEmbed: func(_ context.Context, _ [][]float32, _ error) {
			order = append(order, "h2-after")
		},
	}

	composed := embedding.ComposeHooks(h1, h2)

	ctx := context.Background()
	err := composed.BeforeEmbed(ctx, []string{"test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	composed.AfterEmbed(ctx, nil, nil)

	expected := []string{"h1-before", "h2-before", "h1-after", "h2-after"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d calls, got %d: %v", len(expected), len(order), order)
	}
	for i, exp := range expected {
		if order[i] != exp {
			t.Fatalf("call %d: expected %q, got %q", i, exp, order[i])
		}
	}
}

func TestComposeHooks_BeforeError(t *testing.T) {
	errAbort := errors.New("abort")
	h1 := embedding.Hooks{
		BeforeEmbed: func(_ context.Context, _ []string) error {
			return errAbort
		},
	}
	h2 := embedding.Hooks{
		BeforeEmbed: func(_ context.Context, _ []string) error {
			t.Fatal("h2 should not be called")
			return nil
		},
	}

	composed := embedding.ComposeHooks(h1, h2)
	err := composed.BeforeEmbed(context.Background(), []string{"test"})
	if !errors.Is(err, errAbort) {
		t.Fatalf("expected errAbort, got %v", err)
	}
}

func TestMiddleware(t *testing.T) {
	emb, _ := embedding.New("inmemory", config.ProviderConfig{})
	ctx := context.Background()

	var called bool
	hooks := embedding.Hooks{
		BeforeEmbed: func(_ context.Context, texts []string) error {
			called = true
			if len(texts) != 1 || texts[0] != "hello" {
				t.Fatalf("unexpected texts: %v", texts)
			}
			return nil
		},
	}

	wrapped := embedding.ApplyMiddleware(emb, embedding.WithHooks(hooks))
	_, err := wrapped.Embed(ctx, []string{"hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("hooks not called")
	}

	// Verify dimensions passthrough.
	if wrapped.Dimensions() != emb.Dimensions() {
		t.Fatalf("dimensions mismatch: %d != %d", wrapped.Dimensions(), emb.Dimensions())
	}
}

func TestMiddleware_HooksAbort(t *testing.T) {
	emb, _ := embedding.New("inmemory", config.ProviderConfig{})
	ctx := context.Background()

	errAbort := errors.New("abort")
	hooks := embedding.Hooks{
		BeforeEmbed: func(_ context.Context, _ []string) error {
			return errAbort
		},
	}

	wrapped := embedding.ApplyMiddleware(emb, embedding.WithHooks(hooks))

	_, err := wrapped.Embed(ctx, []string{"hello"})
	if !errors.Is(err, errAbort) {
		t.Fatalf("expected errAbort, got %v", err)
	}

	_, err = wrapped.EmbedSingle(ctx, "hello")
	if !errors.Is(err, errAbort) {
		t.Fatalf("expected errAbort for EmbedSingle, got %v", err)
	}
}

func TestMiddleware_AfterEmbedHook(t *testing.T) {
	emb, _ := embedding.New("inmemory", config.ProviderConfig{})
	ctx := context.Background()

	var afterCalled bool
	var capturedEmbeddings [][]float32
	var capturedErr error

	hooks := embedding.Hooks{
		AfterEmbed: func(_ context.Context, embeddings [][]float32, err error) {
			afterCalled = true
			capturedEmbeddings = embeddings
			capturedErr = err
		},
	}

	wrapped := embedding.ApplyMiddleware(emb, embedding.WithHooks(hooks))

	t.Run("Embed calls AfterEmbed hook", func(t *testing.T) {
		afterCalled = false
		capturedEmbeddings = nil
		capturedErr = nil

		vecs, err := wrapped.Embed(ctx, []string{"test"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !afterCalled {
			t.Fatal("AfterEmbed hook not called")
		}
		if len(capturedEmbeddings) != 1 {
			t.Fatalf("expected 1 embedding in hook, got %d", len(capturedEmbeddings))
		}
		if len(capturedEmbeddings[0]) != len(vecs[0]) {
			t.Fatalf("embedding dimension mismatch in hook: %d != %d", len(capturedEmbeddings[0]), len(vecs[0]))
		}
		if capturedErr != nil {
			t.Fatalf("expected nil error in hook, got %v", capturedErr)
		}
	})

	t.Run("EmbedSingle calls AfterEmbed with non-nil vec", func(t *testing.T) {
		afterCalled = false
		capturedEmbeddings = nil
		capturedErr = nil

		vec, err := wrapped.EmbedSingle(ctx, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !afterCalled {
			t.Fatal("AfterEmbed hook not called")
		}
		if len(capturedEmbeddings) != 1 {
			t.Fatalf("expected 1 embedding in hook, got %d", len(capturedEmbeddings))
		}
		if len(capturedEmbeddings[0]) != len(vec) {
			t.Fatalf("embedding dimension mismatch in hook: %d != %d", len(capturedEmbeddings[0]), len(vec))
		}
		if capturedErr != nil {
			t.Fatalf("expected nil error in hook, got %v", capturedErr)
		}
	})
}

func TestMiddleware_EmbedSingleAfterEmbedWithError(t *testing.T) {
	// Create a mock embedder that returns an error
	mockEmb := &errorEmbedder{err: errors.New("embed error"), dims: 128}
	ctx := context.Background()

	var afterCalled bool
	var capturedEmbeddings [][]float32
	var capturedErr error

	hooks := embedding.Hooks{
		AfterEmbed: func(_ context.Context, embeddings [][]float32, err error) {
			afterCalled = true
			capturedEmbeddings = embeddings
			capturedErr = err
		},
	}

	wrapped := embedding.ApplyMiddleware(mockEmb, embedding.WithHooks(hooks))

	t.Run("EmbedSingle calls AfterEmbed with nil vec on error", func(t *testing.T) {
		afterCalled = false
		capturedEmbeddings = nil
		capturedErr = nil

		vec, err := wrapped.EmbedSingle(ctx, "test")
		if err == nil {
			t.Fatal("expected error")
		}
		if vec != nil {
			t.Fatalf("expected nil vec, got %v", vec)
		}
		if !afterCalled {
			t.Fatal("AfterEmbed hook not called")
		}
		if capturedEmbeddings != nil {
			t.Fatalf("expected nil embeddings in hook, got %v", capturedEmbeddings)
		}
		if capturedErr == nil {
			t.Fatal("expected non-nil error in hook")
		}
	})
}

// errorEmbedder is a mock embedder that always returns an error.
type errorEmbedder struct {
	err  error
	dims int
}

func (e *errorEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return nil, e.err
}

func (e *errorEmbedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	return nil, e.err
}

func (e *errorEmbedder) Dimensions() int {
	return e.dims
}

func TestApplyMiddleware_Order(t *testing.T) {
	emb, _ := embedding.New("inmemory", config.ProviderConfig{})
	ctx := context.Background()

	var order []string
	mw1 := func(next embedding.Embedder) embedding.Embedder {
		return &orderEmbedder{next: next, name: "mw1", order: &order}
	}
	mw2 := func(next embedding.Embedder) embedding.Embedder {
		return &orderEmbedder{next: next, name: "mw2", order: &order}
	}

	wrapped := embedding.ApplyMiddleware(emb, mw1, mw2)
	_, _ = wrapped.Embed(ctx, []string{"test"})

	// mw1 is outermost, so it runs first.
	expected := []string{"mw1", "mw2"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d calls, got %d: %v", len(expected), len(order), order)
	}
	for i, exp := range expected {
		if order[i] != exp {
			t.Fatalf("call %d: expected %q, got %q", i, exp, order[i])
		}
	}
}

type orderEmbedder struct {
	next  embedding.Embedder
	name  string
	order *[]string
}

func (e *orderEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	*e.order = append(*e.order, e.name)
	return e.next.Embed(ctx, texts)
}

func (e *orderEmbedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	*e.order = append(*e.order, e.name)
	return e.next.EmbedSingle(ctx, text)
}

func (e *orderEmbedder) Dimensions() int {
	return e.next.Dimensions()
}
