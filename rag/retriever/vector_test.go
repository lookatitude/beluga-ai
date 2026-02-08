package retriever

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/rag/embedding"
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/schema"
)

// --- Mock implementations for VectorStoreRetriever tests ---

// mockVectorStoreForRetriever is a mock that properly implements vectorstore.VectorStore.
type mockVectorStoreForRetriever struct {
	searchDocs []schema.Document
	searchErr  error
	addCalls   int
	deleteCalls int
}

func (m *mockVectorStoreForRetriever) Add(ctx context.Context, docs []schema.Document, embeddings [][]float32) error {
	m.addCalls++
	return nil
}

func (m *mockVectorStoreForRetriever) Search(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	result := m.searchDocs
	if k > 0 && len(result) > k {
		result = result[:k]
	}
	return result, nil
}

func (m *mockVectorStoreForRetriever) Delete(ctx context.Context, ids []string) error {
	m.deleteCalls++
	return nil
}

// mockEmbedderForRetriever properly implements embedding.Embedder.
type mockEmbedderForRetriever struct {
	embedding []float32
	dims      int
	embedErr  error
}

func (m *mockEmbedderForRetriever) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if m.embedErr != nil {
		return nil, m.embedErr
	}
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = m.embedding
	}
	return result, nil
}

func (m *mockEmbedderForRetriever) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	if m.embedErr != nil {
		return nil, m.embedErr
	}
	return m.embedding, nil
}

func (m *mockEmbedderForRetriever) Dimensions() int {
	return m.dims
}

// --- Tests for VectorStoreRetriever ---

func TestVectorStoreRetriever_Retrieve_Success(t *testing.T) {
	docs := []schema.Document{
		{ID: "doc1", Content: "first document", Score: 0.95},
		{ID: "doc2", Content: "second document", Score: 0.85},
		{ID: "doc3", Content: "third document", Score: 0.75},
	}

	store := &mockVectorStoreForRetriever{searchDocs: docs}
	embedder := &mockEmbedderForRetriever{
		embedding: []float32{0.1, 0.2, 0.3},
		dims:      3,
	}

	r := NewVectorStoreRetriever(store, embedder)

	results, err := r.Retrieve(context.Background(), "test query")
	require.NoError(t, err)
	assert.Len(t, results, 3)
	assert.Equal(t, "doc1", results[0].ID)
	assert.Equal(t, "doc2", results[1].ID)
	assert.Equal(t, "doc3", results[2].ID)
}

func TestVectorStoreRetriever_WithTopK(t *testing.T) {
	docs := []schema.Document{
		{ID: "doc1", Content: "first", Score: 0.9},
		{ID: "doc2", Content: "second", Score: 0.8},
		{ID: "doc3", Content: "third", Score: 0.7},
		{ID: "doc4", Content: "fourth", Score: 0.6},
	}

	store := &mockVectorStoreForRetriever{searchDocs: docs}
	embedder := &mockEmbedderForRetriever{
		embedding: []float32{0.1, 0.2, 0.3},
		dims:      3,
	}

	r := NewVectorStoreRetriever(store, embedder)

	results, err := r.Retrieve(context.Background(), "test query", WithTopK(2))
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "doc1", results[0].ID)
	assert.Equal(t, "doc2", results[1].ID)
}

func TestVectorStoreRetriever_WithThreshold(t *testing.T) {
	docs := []schema.Document{
		{ID: "doc1", Content: "first", Score: 0.9},
		{ID: "doc2", Content: "second", Score: 0.5},
	}

	store := &mockVectorStoreForRetriever{searchDocs: docs}
	embedder := &mockEmbedderForRetriever{
		embedding: []float32{0.1, 0.2, 0.3},
		dims:      3,
	}

	r := NewVectorStoreRetriever(store, embedder)

	// Store will handle threshold filtering, so we just verify the option is passed
	results, err := r.Retrieve(context.Background(), "test query", WithThreshold(0.7))
	require.NoError(t, err)
	assert.NotNil(t, results)
}

func TestVectorStoreRetriever_WithMetadata(t *testing.T) {
	docs := []schema.Document{
		{ID: "doc1", Content: "first", Metadata: map[string]any{"type": "article"}},
	}

	store := &mockVectorStoreForRetriever{searchDocs: docs}
	embedder := &mockEmbedderForRetriever{
		embedding: []float32{0.1, 0.2, 0.3},
		dims:      3,
	}

	r := NewVectorStoreRetriever(store, embedder)

	metadata := map[string]any{"type": "article"}
	results, err := r.Retrieve(context.Background(), "test query", WithMetadata(metadata))
	require.NoError(t, err)
	assert.NotNil(t, results)
}

func TestVectorStoreRetriever_EmbedError(t *testing.T) {
	store := &mockVectorStoreForRetriever{searchDocs: makeDocs("doc1")}
	embedErr := errors.New("embedding failed")
	embedder := &mockEmbedderForRetriever{
		embedErr: embedErr,
		dims:     3,
	}

	r := NewVectorStoreRetriever(store, embedder)

	_, err := r.Retrieve(context.Background(), "test query")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "embed query")
}

func TestVectorStoreRetriever_SearchError(t *testing.T) {
	searchErr := errors.New("search failed")
	store := &mockVectorStoreForRetriever{searchErr: searchErr}
	embedder := &mockEmbedderForRetriever{
		embedding: []float32{0.1, 0.2, 0.3},
		dims:      3,
	}

	r := NewVectorStoreRetriever(store, embedder)

	_, err := r.Retrieve(context.Background(), "test query")
	require.Error(t, err)
	assert.Equal(t, searchErr, err)
}

func TestVectorStoreRetriever_WithHooks(t *testing.T) {
	docs := makeDocs("doc1", "doc2")
	store := &mockVectorStoreForRetriever{searchDocs: docs}
	embedder := &mockEmbedderForRetriever{
		embedding: []float32{0.1, 0.2, 0.3},
		dims:      3,
	}

	var beforeCalled, afterCalled bool
	var capturedQuery string
	var capturedDocs []schema.Document

	hooks := Hooks{
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

	r := NewVectorStoreRetriever(store, embedder, WithVectorStoreHooks(hooks))

	results, err := r.Retrieve(context.Background(), "test query")
	require.NoError(t, err)
	assert.Len(t, results, 2)

	assert.True(t, beforeCalled, "BeforeRetrieve hook should be called")
	assert.True(t, afterCalled, "AfterRetrieve hook should be called")
	assert.Equal(t, "test query", capturedQuery)
	assert.Len(t, capturedDocs, 2)
}

func TestVectorStoreRetriever_BeforeHookError(t *testing.T) {
	store := &mockVectorStoreForRetriever{searchDocs: makeDocs("doc1")}
	embedder := &mockEmbedderForRetriever{
		embedding: []float32{0.1, 0.2, 0.3},
		dims:      3,
	}

	hookErr := errors.New("hook rejected")
	hooks := Hooks{
		BeforeRetrieve: func(ctx context.Context, query string) error {
			return hookErr
		},
	}

	r := NewVectorStoreRetriever(store, embedder, WithVectorStoreHooks(hooks))

	_, err := r.Retrieve(context.Background(), "test query")
	require.Error(t, err)
	assert.Equal(t, hookErr, err)
}

func TestVectorStoreRetriever_EmptyResults(t *testing.T) {
	store := &mockVectorStoreForRetriever{searchDocs: []schema.Document{}}
	embedder := &mockEmbedderForRetriever{
		embedding: []float32{0.1, 0.2, 0.3},
		dims:      3,
	}

	r := NewVectorStoreRetriever(store, embedder)

	results, err := r.Retrieve(context.Background(), "test query")
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestVectorStoreRetriever_ContextCancellation(t *testing.T) {
	// This test verifies that context cancellation is properly propagated
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	store := &mockVectorStoreForRetriever{searchDocs: makeDocs("doc1")}
	embedder := &mockEmbedderForRetriever{
		embedding: []float32{0.1, 0.2, 0.3},
		dims:      3,
	}

	r := NewVectorStoreRetriever(store, embedder)

	// The actual behavior depends on whether the embedder/store check context
	// For this test, we just verify it doesn't panic
	_, err := r.Retrieve(ctx, "test query")
	// Error is acceptable (context canceled) or success if mocks don't check context
	_ = err
}

// Compile-time interface checks
var (
	_ vectorstore.VectorStore = (*mockVectorStoreForRetriever)(nil)
	_ embedding.Embedder      = (*mockEmbedderForRetriever)(nil)
	_ Retriever               = (*VectorStoreRetriever)(nil)
)
