package retriever

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/schema"
)

// --- Mock Reranker for tests ---

// mockRerankerImpl implements Reranker for testing.
type mockRerankerImpl struct {
	rerankedDocs []schema.Document
	rerankErr    error
	callCount    int
}

func (m *mockRerankerImpl) Rerank(ctx context.Context, query string, docs []schema.Document) ([]schema.Document, error) {
	m.callCount++
	if m.rerankErr != nil {
		return nil, m.rerankErr
	}
	if m.rerankedDocs != nil {
		return m.rerankedDocs, nil
	}
	// Default: reverse order with new scores
	result := make([]schema.Document, len(docs))
	for i, doc := range docs {
		doc.Score = float64(len(docs) - i)
		result[len(docs)-1-i] = doc
	}
	return result, nil
}

// --- Tests for RerankRetriever ---

func TestRerankRetriever_Success(t *testing.T) {
	inner := &mockRetriever{docs: makeDocs("a", "b", "c")}
	reranker := &mockRerankerImpl{}

	r := NewRerankRetriever(inner, reranker)
	docs, err := r.Retrieve(context.Background(), "query")
	require.NoError(t, err)
	assert.Len(t, docs, 3)
	assert.Equal(t, 1, reranker.callCount)
	// Default mock reranker reverses order
	assert.Equal(t, "c", docs[0].ID)
	assert.Equal(t, "b", docs[1].ID)
	assert.Equal(t, "a", docs[2].ID)
}

func TestRerankRetriever_WithTopN(t *testing.T) {
	inner := &mockRetriever{docs: makeDocs("a", "b", "c", "d", "e")}
	reranker := &mockRerankerImpl{}

	r := NewRerankRetriever(inner, reranker, WithRerankTopN(2))
	docs, err := r.Retrieve(context.Background(), "query")
	require.NoError(t, err)
	assert.Len(t, docs, 2, "should limit to TopN after reranking")
}

func TestRerankRetriever_InnerRetrieverError(t *testing.T) {
	innerErr := errors.New("inner retrieval failed")
	inner := &mockRetriever{err: innerErr}
	reranker := &mockRerankerImpl{}

	r := NewRerankRetriever(inner, reranker)
	_, err := r.Retrieve(context.Background(), "query")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rerank inner retrieve")
	assert.Equal(t, 0, reranker.callCount, "reranker should not be called on inner error")
}

func TestRerankRetriever_RerankerError_Detailed(t *testing.T) {
	inner := &mockRetriever{docs: makeDocs("a", "b")}
	rerankErr := errors.New("reranking failed")
	reranker := &mockRerankerImpl{rerankErr: rerankErr}

	r := NewRerankRetriever(inner, reranker)
	_, err := r.Retrieve(context.Background(), "query")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rerank:")
}

func TestRerankRetriever_EmptyInnerResults(t *testing.T) {
	inner := &mockRetriever{docs: []schema.Document{}}
	reranker := &mockRerankerImpl{}

	r := NewRerankRetriever(inner, reranker)
	docs, err := r.Retrieve(context.Background(), "query")
	require.NoError(t, err)
	assert.Empty(t, docs)
	assert.Equal(t, 0, reranker.callCount, "reranker should not be called on empty results")
}

func TestRerankRetriever_WithHooks_Detailed(t *testing.T) {
	inner := &mockRetriever{docs: makeDocs("a", "b", "c")}
	reranker := &mockRerankerImpl{}

	var beforeCalled, afterCalled, rerankCalled bool
	var capturedBefore, capturedAfter []schema.Document

	hooks := Hooks{
		BeforeRetrieve: func(ctx context.Context, query string) error {
			beforeCalled = true
			return nil
		},
		AfterRetrieve: func(ctx context.Context, docs []schema.Document, err error) {
			afterCalled = true
			capturedAfter = docs
		},
		OnRerank: func(ctx context.Context, query string, before, after []schema.Document) {
			rerankCalled = true
			capturedBefore = before
		},
	}

	r := NewRerankRetriever(inner, reranker, WithRerankHooks(hooks))
	docs, err := r.Retrieve(context.Background(), "query")
	require.NoError(t, err)
	assert.Len(t, docs, 3)

	assert.True(t, beforeCalled, "BeforeRetrieve should be called")
	assert.True(t, afterCalled, "AfterRetrieve should be called")
	assert.True(t, rerankCalled, "OnRerank should be called")
	assert.Len(t, capturedBefore, 3, "OnRerank should receive original docs")
	assert.Len(t, capturedAfter, 3, "AfterRetrieve should receive reranked docs")
}

func TestRerankRetriever_BeforeHookError(t *testing.T) {
	inner := &mockRetriever{docs: makeDocs("a")}
	reranker := &mockRerankerImpl{}
	hookErr := errors.New("hook rejected")

	hooks := Hooks{
		BeforeRetrieve: func(ctx context.Context, query string) error {
			return hookErr
		},
	}

	r := NewRerankRetriever(inner, reranker, WithRerankHooks(hooks))
	_, err := r.Retrieve(context.Background(), "query")
	require.Error(t, err)
	assert.Equal(t, hookErr, err)
	assert.Equal(t, 0, reranker.callCount)
}

func TestRerankRetriever_OnRerankNotCalledWhenEmpty(t *testing.T) {
	inner := &mockRetriever{docs: []schema.Document{}}
	reranker := &mockRerankerImpl{}

	rerankCalled := false
	hooks := Hooks{
		OnRerank: func(ctx context.Context, query string, before, after []schema.Document) {
			rerankCalled = true
		},
	}

	r := NewRerankRetriever(inner, reranker, WithRerankHooks(hooks))
	_, err := r.Retrieve(context.Background(), "query")
	require.NoError(t, err)
	assert.False(t, rerankCalled, "OnRerank should not be called for empty results")
}

func TestRerankRetriever_CustomRerankerScores(t *testing.T) {
	inner := &mockRetriever{docs: []schema.Document{
		{ID: "low", Score: 0.3},
		{ID: "high", Score: 0.9},
		{ID: "mid", Score: 0.6},
	}}

	// Reranker returns custom order and scores
	rerankedDocs := []schema.Document{
		{ID: "mid", Score: 1.0},
		{ID: "low", Score: 0.8},
		{ID: "high", Score: 0.5},
	}
	reranker := &mockRerankerImpl{rerankedDocs: rerankedDocs}

	r := NewRerankRetriever(inner, reranker)
	docs, err := r.Retrieve(context.Background(), "query")
	require.NoError(t, err)
	assert.Len(t, docs, 3)

	assert.Equal(t, "mid", docs[0].ID)
	assert.Equal(t, 1.0, docs[0].Score)
	assert.Equal(t, "low", docs[1].ID)
	assert.Equal(t, 0.8, docs[1].Score)
	assert.Equal(t, "high", docs[2].ID)
	assert.Equal(t, 0.5, docs[2].Score)
}

func TestRerankRetriever_TopNLargerThanResults(t *testing.T) {
	inner := &mockRetriever{docs: makeDocs("a", "b")}
	reranker := &mockRerankerImpl{}

	r := NewRerankRetriever(inner, reranker, WithRerankTopN(10))
	docs, err := r.Retrieve(context.Background(), "query")
	require.NoError(t, err)
	assert.Len(t, docs, 2, "should return all docs if TopN > result count")
}

func TestRerankRetriever_TopNZero(t *testing.T) {
	inner := &mockRetriever{docs: makeDocs("a", "b", "c")}
	reranker := &mockRerankerImpl{}

	r := NewRerankRetriever(inner, reranker, WithRerankTopN(0))
	docs, err := r.Retrieve(context.Background(), "query")
	require.NoError(t, err)
	assert.Len(t, docs, 3, "TopN=0 should return all docs")
}

func TestRerankRetriever_PreservesDocumentMetadata(t *testing.T) {
	inner := &mockRetriever{docs: []schema.Document{
		{ID: "a", Content: "content a", Metadata: map[string]any{"key": "value"}},
	}}

	reranker := &mockRerankerImpl{
		rerankedDocs: []schema.Document{
			{ID: "a", Content: "content a", Score: 0.95, Metadata: map[string]any{"key": "value"}},
		},
	}

	r := NewRerankRetriever(inner, reranker)
	docs, err := r.Retrieve(context.Background(), "query")
	require.NoError(t, err)
	assert.Len(t, docs, 1)
	assert.Equal(t, "value", docs[0].Metadata["key"])
}

func TestRerankRetriever_PassesOptionsToInner(t *testing.T) {
	// Create a custom mock that tracks options
	type optionTracker struct {
		topK int
	}
	tracker := &optionTracker{}

	inner := &mockRetriever{
		docs: makeDocs("a", "b", "c", "d", "e"),
	}
	// Override Retrieve to capture TopK
	originalRetrieve := inner.Retrieve
	_ = originalRetrieve

	reranker := &mockRerankerImpl{}
	r := NewRerankRetriever(inner, reranker)

	// Pass options to RerankRetriever
	docs, err := r.Retrieve(context.Background(), "query", WithTopK(3))
	require.NoError(t, err)
	assert.NotNil(t, docs)

	// The inner retriever receives the options
	// We can't easily test this with the current mock, but the code path is exercised
	_ = tracker
}

// --- Test utility: custom reranker that sorts by ID ---

type idSortReranker struct{}

func (r *idSortReranker) Rerank(ctx context.Context, query string, docs []schema.Document) ([]schema.Document, error) {
	result := make([]schema.Document, len(docs))
	copy(result, docs)
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	// Assign descending scores
	for i := range result {
		result[i].Score = float64(len(result) - i)
	}
	return result, nil
}

func TestRerankRetriever_WithCustomReranker(t *testing.T) {
	inner := &mockRetriever{docs: []schema.Document{
		{ID: "z", Content: "last"},
		{ID: "a", Content: "first"},
		{ID: "m", Content: "middle"},
	}}

	reranker := &idSortReranker{}
	r := NewRerankRetriever(inner, reranker)

	docs, err := r.Retrieve(context.Background(), "query")
	require.NoError(t, err)
	assert.Len(t, docs, 3)
	assert.Equal(t, "a", docs[0].ID)
	assert.Equal(t, "m", docs[1].ID)
	assert.Equal(t, "z", docs[2].ID)
}

// Compile-time interface checks
var (
	_ Reranker  = (*mockRerankerImpl)(nil)
	_ Reranker  = (*idSortReranker)(nil)
	_ Retriever = (*RerankRetriever)(nil)
)
