package retriever_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/rag/retriever"
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHybridRetriever_Defaults(t *testing.T) {
	store := &mockVectorStore{}
	embedder := &mockEmbedder{}
	bm25 := &mockBM25Searcher{}

	r := retriever.NewHybridRetriever(store, embedder, bm25)
	require.NotNil(t, r)
}

func TestNewHybridRetriever_WithRRFK(t *testing.T) {
	store := &mockVectorStore{}
	embedder := &mockEmbedder{}
	bm25 := &mockBM25Searcher{}

	r := retriever.NewHybridRetriever(store, embedder, bm25, retriever.WithHybridRRFK(100))
	require.NotNil(t, r)
}

func TestNewHybridRetriever_WithRRFK_IgnoresZero(t *testing.T) {
	store := &mockVectorStore{}
	embedder := &mockEmbedder{}
	bm25 := &mockBM25Searcher{}

	// Should ignore zero and use default (60).
	r := retriever.NewHybridRetriever(store, embedder, bm25, retriever.WithHybridRRFK(0))
	require.NotNil(t, r)
}

func TestNewHybridRetriever_WithRRFK_IgnoresNegative(t *testing.T) {
	store := &mockVectorStore{}
	embedder := &mockEmbedder{}
	bm25 := &mockBM25Searcher{}

	// Should ignore negative and use default (60).
	r := retriever.NewHybridRetriever(store, embedder, bm25, retriever.WithHybridRRFK(-10))
	require.NotNil(t, r)
}

func TestHybridRetriever_Retrieve_HappyPath(t *testing.T) {
	vectorDocs := makeDocs("v1", "v2", "v3")
	bm25Docs := makeDocs("b1", "v1", "b2")

	embedder := &mockEmbedder{
		embedSingleFn: func(ctx context.Context, text string) ([]float32, error) {
			return []float32{1.0, 0.0, 0.0}, nil
		},
	}

	store := &mockVectorStore{
		searchFn: func(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
			return vectorDocs, nil
		},
	}

	bm25 := &mockBM25Searcher{
		searchFn: func(ctx context.Context, query string, k int) ([]schema.Document, error) {
			return bm25Docs, nil
		},
	}

	r := retriever.NewHybridRetriever(store, embedder, bm25)
	docs, err := r.Retrieve(context.Background(), "test query")

	require.NoError(t, err)
	assert.NotEmpty(t, docs)
	// v1 should appear (exists in both result sets).
	foundV1 := false
	for _, doc := range docs {
		if doc.ID == "v1" {
			foundV1 = true
			break
		}
	}
	assert.True(t, foundV1, "v1 should appear in fused results")
}

func TestHybridRetriever_Retrieve_WithTopK(t *testing.T) {
	vectorDocs := makeDocs("v1", "v2", "v3", "v4", "v5")
	bm25Docs := makeDocs("b1", "b2", "b3", "b4", "b5")

	embedder := &mockEmbedder{}
	store := &mockVectorStore{
		searchFn: func(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
			// HybridRetriever fetches 2x TopK for better fusion.
			return vectorDocs, nil
		},
	}

	bm25 := &mockBM25Searcher{
		searchFn: func(ctx context.Context, query string, k int) ([]schema.Document, error) {
			return bm25Docs, nil
		},
	}

	r := retriever.NewHybridRetriever(store, embedder, bm25)
	docs, err := r.Retrieve(context.Background(), "query", retriever.WithTopK(3))

	require.NoError(t, err)
	assert.LessOrEqual(t, len(docs), 3, "Should respect TopK limit")
}

func TestHybridRetriever_Retrieve_EmbedError(t *testing.T) {
	expectedErr := errors.New("embed failure")
	embedder := &mockEmbedder{
		embedSingleFn: func(ctx context.Context, text string) ([]float32, error) {
			return nil, expectedErr
		},
	}

	store := &mockVectorStore{}
	bm25 := &mockBM25Searcher{}

	r := retriever.NewHybridRetriever(store, embedder, bm25)
	_, err := r.Retrieve(context.Background(), "query")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "hybrid embed")
}

func TestHybridRetriever_Retrieve_VectorSearchError(t *testing.T) {
	expectedErr := errors.New("vector search failure")
	embedder := &mockEmbedder{}

	store := &mockVectorStore{
		searchFn: func(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
			return nil, expectedErr
		},
	}

	bm25 := &mockBM25Searcher{}

	r := retriever.NewHybridRetriever(store, embedder, bm25)
	_, err := r.Retrieve(context.Background(), "query")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "hybrid vector search")
}

func TestHybridRetriever_Retrieve_BM25Error(t *testing.T) {
	embedder := &mockEmbedder{}
	store := &mockVectorStore{
		searchFn: func(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
			return makeDocs("v1"), nil
		},
	}

	expectedErr := errors.New("bm25 failure")
	bm25 := &mockBM25Searcher{
		searchFn: func(ctx context.Context, query string, k int) ([]schema.Document, error) {
			return nil, expectedErr
		},
	}

	r := retriever.NewHybridRetriever(store, embedder, bm25)
	_, err := r.Retrieve(context.Background(), "query")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "hybrid bm25 search")
}

func TestHybridRetriever_Retrieve_WithMetadata(t *testing.T) {
	embedder := &mockEmbedder{}

	metadata := map[string]any{"category": "tech"}
	var receivedOpts []vectorstore.SearchOption

	store := &mockVectorStore{
		searchFn: func(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
			receivedOpts = opts
			return makeDocs("v1"), nil
		},
	}

	bm25 := &mockBM25Searcher{
		searchFn: func(ctx context.Context, query string, k int) ([]schema.Document, error) {
			return makeDocs("b1"), nil
		},
	}

	r := retriever.NewHybridRetriever(store, embedder, bm25)
	_, err := r.Retrieve(context.Background(), "query", retriever.WithMetadata(metadata))

	require.NoError(t, err)
	assert.NotEmpty(t, receivedOpts, "Metadata should be passed to vector store")
}

func TestHybridRetriever_Retrieve_WithHooks(t *testing.T) {
	var beforeCalled, afterCalled bool
	var capturedQuery string
	var capturedDocs []schema.Document

	hooks := retriever.Hooks{
		BeforeRetrieve: func(ctx context.Context, query string) error {
			beforeCalled = true
			capturedQuery = query
			return nil
		},
		AfterRetrieve: func(ctx context.Context, docs []schema.Document, err error) {
			afterCalled = true
			capturedDocs = docs
		},
	}

	embedder := &mockEmbedder{}
	store := &mockVectorStore{
		searchFn: func(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
			return makeDocs("v1"), nil
		},
	}

	bm25 := &mockBM25Searcher{
		searchFn: func(ctx context.Context, query string, k int) ([]schema.Document, error) {
			return makeDocs("b1"), nil
		},
	}

	r := retriever.NewHybridRetriever(store, embedder, bm25, retriever.WithHybridHooks(hooks))
	docs, err := r.Retrieve(context.Background(), "test query")

	require.NoError(t, err)
	assert.True(t, beforeCalled, "BeforeRetrieve should be called")
	assert.True(t, afterCalled, "AfterRetrieve should be called")
	assert.Equal(t, "test query", capturedQuery)
	assert.NotNil(t, capturedDocs)
	assert.NotNil(t, docs)
}

func TestHybridRetriever_Retrieve_BeforeHookError(t *testing.T) {
	expectedErr := errors.New("hook error")
	hooks := retriever.Hooks{
		BeforeRetrieve: func(ctx context.Context, query string) error {
			return expectedErr
		},
	}

	embedder := &mockEmbedder{}
	store := &mockVectorStore{}
	bm25 := &mockBM25Searcher{}

	r := retriever.NewHybridRetriever(store, embedder, bm25, retriever.WithHybridHooks(hooks))
	_, err := r.Retrieve(context.Background(), "query")

	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestHybridRetriever_Retrieve_RRFFusion(t *testing.T) {
	// Both result sets contain doc "common" but at different ranks.
	// RRF should give it a higher score.
	vectorDocs := []schema.Document{
		{ID: "common", Content: "shared", Score: 1.0},
		{ID: "v-only", Content: "vector", Score: 0.8},
	}
	bm25Docs := []schema.Document{
		{ID: "b-only", Content: "bm25", Score: 1.0},
		{ID: "common", Content: "shared", Score: 0.9},
	}

	embedder := &mockEmbedder{}
	store := &mockVectorStore{
		searchFn: func(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
			return vectorDocs, nil
		},
	}

	bm25 := &mockBM25Searcher{
		searchFn: func(ctx context.Context, query string, k int) ([]schema.Document, error) {
			return bm25Docs, nil
		},
	}

	r := retriever.NewHybridRetriever(store, embedder, bm25)
	docs, err := r.Retrieve(context.Background(), "query")

	require.NoError(t, err)
	assert.NotEmpty(t, docs)

	// "common" should be in results.
	foundCommon := false
	for _, doc := range docs {
		if doc.ID == "common" {
			foundCommon = true
			// After RRF fusion, score should be updated.
			assert.Greater(t, doc.Score, 0.0)
			break
		}
	}
	assert.True(t, foundCommon, "common document should appear in fused results")
}

func TestHybridRetriever_Retrieve_MinimumCandidates(t *testing.T) {
	// When TopK is very small, should fetch at least 20 candidates.
	embedder := &mockEmbedder{}

	var vectorK, bm25K int
	store := &mockVectorStore{
		searchFn: func(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
			vectorK = k
			return makeDocs("v1"), nil
		},
	}

	bm25 := &mockBM25Searcher{
		searchFn: func(ctx context.Context, query string, k int) ([]schema.Document, error) {
			bm25K = k
			return makeDocs("b1"), nil
		},
	}

	r := retriever.NewHybridRetriever(store, embedder, bm25)
	_, err := r.Retrieve(context.Background(), "query", retriever.WithTopK(3))

	require.NoError(t, err)
	assert.GreaterOrEqual(t, vectorK, 20, "Should fetch at least 20 vector candidates")
	assert.GreaterOrEqual(t, bm25K, 20, "Should fetch at least 20 BM25 candidates")
}
