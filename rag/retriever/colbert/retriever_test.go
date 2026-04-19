package colbert

import (
	"context"
	"fmt"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/rag/embedding"
	"github.com/lookatitude/beluga-ai/v2/rag/retriever"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

// mockMultiVectorEmbedder is a test double for embedding.MultiVectorEmbedder.
type mockMultiVectorEmbedder struct {
	embedMultiFn    func(ctx context.Context, texts []string) ([][][]float32, error)
	tokenDimensions int
}

var _ embedding.MultiVectorEmbedder = (*mockMultiVectorEmbedder)(nil)

func (m *mockMultiVectorEmbedder) EmbedMulti(ctx context.Context, texts []string) ([][][]float32, error) {
	if m.embedMultiFn != nil {
		return m.embedMultiFn(ctx, texts)
	}
	// Default: return single token per text, identity-like vectors.
	result := make([][][]float32, len(texts))
	for i := range texts {
		result[i] = [][]float32{{1, 0, 0}}
	}
	return result, nil
}

func (m *mockMultiVectorEmbedder) TokenDimensions() int {
	return m.tokenDimensions
}

func TestNewColBERTRetriever(t *testing.T) {
	emb := &mockMultiVectorEmbedder{tokenDimensions: 3}
	idx := NewInMemoryIndex()

	tests := []struct {
		name    string
		opts    []ColBERTOption
		wantErr bool
	}{
		{
			name:    "missing embedder",
			opts:    []ColBERTOption{WithIndex(idx)},
			wantErr: true,
		},
		{
			name:    "missing index",
			opts:    []ColBERTOption{WithEmbedder(emb)},
			wantErr: true,
		},
		{
			name:    "missing both",
			opts:    nil,
			wantErr: true,
		},
		{
			name: "valid config",
			opts: []ColBERTOption{WithEmbedder(emb), WithIndex(idx), WithTopK(5)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewColBERTRetriever(tt.opts...)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if r == nil {
				t.Fatal("expected non-nil retriever")
			}
		})
	}
}

func TestColBERTRetriever_Retrieve(t *testing.T) {
	ctx := context.Background()

	// Set up index with known documents.
	idx := NewInMemoryIndex()
	_ = idx.Add(ctx, "doc-go", [][]float32{{1, 0, 0}, {0.5, 0.5, 0}})
	_ = idx.Add(ctx, "doc-rust", [][]float32{{0, 1, 0}, {0, 0.5, 0.5}})
	_ = idx.Add(ctx, "doc-python", [][]float32{{0, 0, 1}, {0.5, 0, 0.5}})

	emb := &mockMultiVectorEmbedder{
		tokenDimensions: 3,
		embedMultiFn: func(_ context.Context, texts []string) ([][][]float32, error) {
			// Query embedding: token vectors pointing toward doc-go.
			return [][][]float32{{{1, 0, 0}}}, nil
		},
	}

	r, err := NewColBERTRetriever(
		WithEmbedder(emb),
		WithIndex(idx),
		WithTopK(2),
	)
	if err != nil {
		t.Fatal(err)
	}

	docs, err := r.Retrieve(ctx, "what is Go?", retriever.WithTopK(2))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("got %d docs, want 2", len(docs))
	}
	if docs[0].ID != "doc-go" {
		t.Errorf("top doc = %q, want %q", docs[0].ID, "doc-go")
	}
	if docs[0].Score <= docs[1].Score {
		t.Errorf("expected descending scores: %f <= %f", docs[0].Score, docs[1].Score)
	}
}

func TestColBERTRetriever_RetrieveWithThreshold(t *testing.T) {
	ctx := context.Background()

	idx := NewInMemoryIndex()
	_ = idx.Add(ctx, "good", [][]float32{{1, 0, 0}})
	_ = idx.Add(ctx, "bad", [][]float32{{0, 1, 0}}) // orthogonal -> score = 0

	emb := &mockMultiVectorEmbedder{
		tokenDimensions: 3,
		embedMultiFn: func(_ context.Context, _ []string) ([][][]float32, error) {
			return [][][]float32{{{1, 0, 0}}}, nil
		},
	}

	r, err := NewColBERTRetriever(
		WithEmbedder(emb),
		WithIndex(idx),
		WithTopK(10),
	)
	if err != nil {
		t.Fatal(err)
	}

	docs, err := r.Retrieve(ctx, "query", retriever.WithThreshold(0.5))
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("got %d docs, want 1 (threshold should filter orthogonal)", len(docs))
	}
	if docs[0].ID != "good" {
		t.Errorf("got doc %q, want %q", docs[0].ID, "good")
	}
}

func TestColBERTRetriever_EmbedError(t *testing.T) {
	ctx := context.Background()
	idx := NewInMemoryIndex()

	emb := &mockMultiVectorEmbedder{
		tokenDimensions: 3,
		embedMultiFn: func(_ context.Context, _ []string) ([][][]float32, error) {
			return nil, fmt.Errorf("embed failed")
		},
	}

	r, err := NewColBERTRetriever(
		WithEmbedder(emb),
		WithIndex(idx),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = r.Retrieve(ctx, "query")
	if err == nil {
		t.Fatal("expected error from embedder failure")
	}
}

func TestColBERTRetriever_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	idx := NewInMemoryIndex()
	emb := &mockMultiVectorEmbedder{
		tokenDimensions: 3,
		embedMultiFn: func(ctx context.Context, _ []string) ([][][]float32, error) {
			return nil, ctx.Err()
		},
	}

	r, err := NewColBERTRetriever(
		WithEmbedder(emb),
		WithIndex(idx),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = r.Retrieve(ctx, "query")
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestColBERTRetriever_Hooks(t *testing.T) {
	ctx := context.Background()
	idx := NewInMemoryIndex()
	_ = idx.Add(ctx, "doc1", [][]float32{{1, 0, 0}})

	emb := &mockMultiVectorEmbedder{tokenDimensions: 3}

	var beforeCalled, afterCalled bool
	r, err := NewColBERTRetriever(
		WithEmbedder(emb),
		WithIndex(idx),
		WithHooks(retriever.Hooks{
			BeforeRetrieve: func(_ context.Context, query string) error {
				beforeCalled = true
				if query != "test" {
					return fmt.Errorf("unexpected query: %s", query)
				}
				return nil
			},
			AfterRetrieve: func(_ context.Context, _ []schema.Document, _ error) {
				afterCalled = true
			},
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = r.Retrieve(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	if !beforeCalled {
		t.Error("BeforeRetrieve hook was not called")
	}
	if !afterCalled {
		t.Error("AfterRetrieve hook was not called")
	}
}

func TestColBERTRetriever_PerCallTopKOverride(t *testing.T) {
	ctx := context.Background()

	idx := NewInMemoryIndex()
	for i, id := range []string{"a", "b", "c", "d", "e"} {
		_ = idx.Add(ctx, id, [][]float32{{float32(i + 1), 0, 0}})
	}

	emb := &mockMultiVectorEmbedder{
		tokenDimensions: 3,
		embedMultiFn: func(_ context.Context, _ []string) ([][][]float32, error) {
			return [][][]float32{{{1, 0, 0}}}, nil
		},
	}

	// Retriever configured with TopK=2 but caller requests 4.
	r, err := NewColBERTRetriever(
		WithEmbedder(emb),
		WithIndex(idx),
		WithTopK(2),
	)
	if err != nil {
		t.Fatal(err)
	}

	docs, err := r.Retrieve(ctx, "query", retriever.WithTopK(4))
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 4 {
		t.Fatalf("per-call topK override: got %d docs, want 4", len(docs))
	}
}

func TestColBERTRetriever_AfterHookOnErrorPaths(t *testing.T) {
	ctx := context.Background()
	idx := NewInMemoryIndex()

	embErr := &mockMultiVectorEmbedder{
		tokenDimensions: 3,
		embedMultiFn: func(_ context.Context, _ []string) ([][][]float32, error) {
			return nil, fmt.Errorf("embed boom")
		},
	}

	var afterErr error
	var afterCalled bool
	r, err := NewColBERTRetriever(
		WithEmbedder(embErr),
		WithIndex(idx),
		WithHooks(retriever.Hooks{
			AfterRetrieve: func(_ context.Context, _ []schema.Document, e error) {
				afterCalled = true
				afterErr = e
			},
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = r.Retrieve(ctx, "q")
	if err == nil {
		t.Fatal("expected error from embedder failure")
	}
	if !afterCalled {
		t.Error("AfterRetrieve must be called on embed error path")
	}
	if afterErr == nil {
		t.Error("AfterRetrieve must receive the error")
	}
}

func TestColBERTRetriever_EmptyEmbeddings(t *testing.T) {
	ctx := context.Background()
	idx := NewInMemoryIndex()

	emb := &mockMultiVectorEmbedder{
		tokenDimensions: 3,
		embedMultiFn: func(_ context.Context, _ []string) ([][][]float32, error) {
			return [][][]float32{}, nil
		},
	}

	r, err := NewColBERTRetriever(
		WithEmbedder(emb),
		WithIndex(idx),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = r.Retrieve(ctx, "query")
	if err == nil {
		t.Fatal("expected error when embedder returns empty embeddings")
	}
}
