package vectorstore_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	_ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/inmemory"
	"github.com/lookatitude/beluga-ai/schema"
)

func TestRegistry(t *testing.T) {
	t.Run("list includes inmemory", func(t *testing.T) {
		names := vectorstore.List()
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

	t.Run("new creates store", func(t *testing.T) {
		store, err := vectorstore.New("inmemory", config.ProviderConfig{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if store == nil {
			t.Fatal("expected non-nil store")
		}
	})

	t.Run("new unknown provider", func(t *testing.T) {
		_, err := vectorstore.New("nonexistent", config.ProviderConfig{})
		if err == nil {
			t.Fatal("expected error for unknown provider")
		}
	})
}

func TestSearchStrategy_String(t *testing.T) {
	tests := []struct {
		strategy vectorstore.SearchStrategy
		want     string
	}{
		{vectorstore.Cosine, "cosine"},
		{vectorstore.DotProduct, "dot_product"},
		{vectorstore.Euclidean, "euclidean"},
		{vectorstore.SearchStrategy(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.strategy.String(); got != tt.want {
			t.Errorf("SearchStrategy(%d).String() = %q, want %q", tt.strategy, got, tt.want)
		}
	}
}

func newTestStore(t *testing.T) vectorstore.VectorStore {
	t.Helper()
	store, err := vectorstore.New("inmemory", config.ProviderConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return store
}

func TestAdd_And_Search(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	docs := []schema.Document{
		{ID: "1", Content: "hello world", Metadata: map[string]any{"topic": "greeting"}},
		{ID: "2", Content: "goodbye world", Metadata: map[string]any{"topic": "farewell"}},
		{ID: "3", Content: "hello again", Metadata: map[string]any{"topic": "greeting"}},
	}
	embeddings := [][]float32{
		{1.0, 0.0, 0.0},
		{0.0, 1.0, 0.0},
		{0.9, 0.1, 0.0},
	}

	err := store.Add(ctx, docs, embeddings)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("search by similarity", func(t *testing.T) {
		query := []float32{1.0, 0.0, 0.0}
		results, err := store.Search(ctx, query, 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
		// Doc "1" should be most similar to query.
		if results[0].ID != "1" {
			t.Fatalf("expected first result to be doc '1', got %q", results[0].ID)
		}
		if results[0].Score <= 0 {
			t.Fatalf("expected positive score, got %f", results[0].Score)
		}
	})

	t.Run("search with filter", func(t *testing.T) {
		query := []float32{1.0, 0.0, 0.0}
		results, err := store.Search(ctx, query, 10, vectorstore.WithFilter(map[string]any{"topic": "farewell"}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].ID != "2" {
			t.Fatalf("expected result to be doc '2', got %q", results[0].ID)
		}
	})

	t.Run("search with threshold", func(t *testing.T) {
		// Query perpendicular to doc "2", moderately close to "1" and "3".
		// Use a very high threshold so only perfect match (doc "1") passes.
		query := []float32{1.0, 0.0, 0.0}
		results, err := store.Search(ctx, query, 10, vectorstore.WithThreshold(0.999))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Only doc "1" has cosine similarity of exactly 1.0 to the query.
		if len(results) != 1 {
			t.Fatalf("expected 1 result with threshold 0.999, got %d", len(results))
		}
		if results[0].ID != "1" {
			t.Fatalf("expected result to be doc '1', got %q", results[0].ID)
		}
	})

	t.Run("search k larger than store", func(t *testing.T) {
		query := []float32{1.0, 0.0, 0.0}
		results, err := store.Search(ctx, query, 100)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(results))
		}
	})
}

func TestAdd_MismatchedLengths(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	docs := []schema.Document{{ID: "1"}}
	embeddings := [][]float32{{1.0}, {2.0}}

	err := store.Add(ctx, docs, embeddings)
	if err == nil {
		t.Fatal("expected error for mismatched lengths")
	}
}

func TestDelete(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	docs := []schema.Document{
		{ID: "1", Content: "hello"},
		{ID: "2", Content: "world"},
	}
	embeddings := [][]float32{
		{1.0, 0.0},
		{0.0, 1.0},
	}

	_ = store.Add(ctx, docs, embeddings)

	err := store.Delete(ctx, []string{"1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	results, err := store.Search(ctx, []float32{1.0, 0.0}, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result after delete, got %d", len(results))
	}
	if results[0].ID != "2" {
		t.Fatalf("expected remaining doc to be '2', got %q", results[0].ID)
	}
}

func TestDelete_NonExistent(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Deleting non-existent IDs should not error.
	err := store.Delete(ctx, []string{"nonexistent"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearchStrategies(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	docs := []schema.Document{
		{ID: "1", Content: "close"},
		{ID: "2", Content: "far"},
	}
	embeddings := [][]float32{
		{0.9, 0.1},
		{0.1, 0.9},
	}
	_ = store.Add(ctx, docs, embeddings)

	query := []float32{1.0, 0.0}

	t.Run("cosine", func(t *testing.T) {
		results, err := store.Search(ctx, query, 2, vectorstore.WithStrategy(vectorstore.Cosine))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if results[0].ID != "1" {
			t.Fatalf("expected doc '1' first with cosine, got %q", results[0].ID)
		}
	})

	t.Run("dot product", func(t *testing.T) {
		results, err := store.Search(ctx, query, 2, vectorstore.WithStrategy(vectorstore.DotProduct))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if results[0].ID != "1" {
			t.Fatalf("expected doc '1' first with dot product, got %q", results[0].ID)
		}
	})

	t.Run("euclidean", func(t *testing.T) {
		results, err := store.Search(ctx, query, 2, vectorstore.WithStrategy(vectorstore.Euclidean))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if results[0].ID != "1" {
			t.Fatalf("expected doc '1' first with euclidean, got %q", results[0].ID)
		}
	})
}

func TestComposeHooks(t *testing.T) {
	var order []string
	h1 := vectorstore.Hooks{
		BeforeAdd: func(_ context.Context, _ []schema.Document) error {
			order = append(order, "h1-before")
			return nil
		},
		AfterSearch: func(_ context.Context, _ []schema.Document, _ error) {
			order = append(order, "h1-after")
		},
	}
	h2 := vectorstore.Hooks{
		BeforeAdd: func(_ context.Context, _ []schema.Document) error {
			order = append(order, "h2-before")
			return nil
		},
		AfterSearch: func(_ context.Context, _ []schema.Document, _ error) {
			order = append(order, "h2-after")
		},
	}

	composed := vectorstore.ComposeHooks(h1, h2)

	ctx := context.Background()
	err := composed.BeforeAdd(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	composed.AfterSearch(ctx, nil, nil)

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
	h1 := vectorstore.Hooks{
		BeforeAdd: func(_ context.Context, _ []schema.Document) error {
			return errAbort
		},
	}
	h2 := vectorstore.Hooks{
		BeforeAdd: func(_ context.Context, _ []schema.Document) error {
			t.Fatal("h2 should not be called")
			return nil
		},
	}

	composed := vectorstore.ComposeHooks(h1, h2)
	err := composed.BeforeAdd(context.Background(), nil)
	if !errors.Is(err, errAbort) {
		t.Fatalf("expected errAbort, got %v", err)
	}
}

func TestMiddleware_WithHooks(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	var addCalled, searchCalled bool
	hooks := vectorstore.Hooks{
		BeforeAdd: func(_ context.Context, docs []schema.Document) error {
			addCalled = true
			return nil
		},
		AfterSearch: func(_ context.Context, results []schema.Document, _ error) {
			searchCalled = true
		},
	}

	wrapped := vectorstore.ApplyMiddleware(store, vectorstore.WithHooks(hooks))

	docs := []schema.Document{{ID: "1", Content: "test"}}
	embeddings := [][]float32{{1.0, 0.0}}
	err := wrapped.Add(ctx, docs, embeddings)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !addCalled {
		t.Fatal("BeforeAdd hook not called")
	}

	_, err = wrapped.Search(ctx, []float32{1.0, 0.0}, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !searchCalled {
		t.Fatal("AfterSearch hook not called")
	}
}

func TestMiddleware_HooksAbort(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	errAbort := errors.New("abort")
	hooks := vectorstore.Hooks{
		BeforeAdd: func(_ context.Context, _ []schema.Document) error {
			return errAbort
		},
	}

	wrapped := vectorstore.ApplyMiddleware(store, vectorstore.WithHooks(hooks))
	err := wrapped.Add(ctx, []schema.Document{{ID: "1"}}, [][]float32{{1.0}})
	if !errors.Is(err, errAbort) {
		t.Fatalf("expected errAbort, got %v", err)
	}
}

func TestMiddleware_HooksDelete(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Add a doc first.
	err := store.Add(ctx, []schema.Document{{ID: "1", Content: "test"}}, [][]float32{{1.0, 0.0}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hooks := vectorstore.Hooks{}
	wrapped := vectorstore.ApplyMiddleware(store, vectorstore.WithHooks(hooks))

	// Delete through the hooked store.
	err = wrapped.Delete(ctx, []string{"1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the doc was deleted.
	results, err := wrapped.Search(ctx, []float32{1.0, 0.0}, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results after delete, got %d", len(results))
	}
}

func TestSearchOptions(t *testing.T) {
	cfg := &vectorstore.SearchConfig{}

	vectorstore.WithFilter(map[string]any{"key": "val"})(cfg)
	if cfg.Filter["key"] != "val" {
		t.Fatalf("expected filter key=val, got %v", cfg.Filter)
	}

	vectorstore.WithThreshold(0.5)(cfg)
	if cfg.Threshold != 0.5 {
		t.Fatalf("expected threshold 0.5, got %f", cfg.Threshold)
	}

	vectorstore.WithStrategy(vectorstore.DotProduct)(cfg)
	if cfg.Strategy != vectorstore.DotProduct {
		t.Fatalf("expected DotProduct strategy, got %v", cfg.Strategy)
	}
}
