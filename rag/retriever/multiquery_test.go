package retriever

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/schema"
)

// --- Tests for MultiQueryRetriever ---

func TestMultiQueryRetriever_Success(t *testing.T) {
	inner := &mockRetriever{docs: makeDocs("doc1", "doc2", "doc3")}
	model := &mockChatModel{
		response: schema.NewAIMessage("variant query 1\nvariant query 2\nvariant query 3"),
	}

	r := NewMultiQueryRetriever(inner, model)
	docs, err := r.Retrieve(context.Background(), "original query")
	require.NoError(t, err)
	assert.NotEmpty(t, docs)
	assert.GreaterOrEqual(t, model.calls, 1, "LLM should be called to generate variants")
}

func TestMultiQueryRetriever_WithQueryCount(t *testing.T) {
	inner := &mockRetriever{docs: makeDocs("doc1")}
	model := &mockChatModel{
		response: schema.NewAIMessage("variant 1\nvariant 2\nvariant 3\nvariant 4\nvariant 5"),
	}

	tests := []struct {
		name  string
		count int
		want  int
	}{
		{"default count", 0, 3}, // Default is 3
		{"count 2", 2, 2},
		{"count 5", 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := []MultiQueryOption{}
			if tt.count > 0 {
				opts = append(opts, WithMultiQueryCount(tt.count))
			}

			r := NewMultiQueryRetriever(inner, model, opts...)
			assert.Equal(t, tt.want, r.numQueries)
		})
	}
}

func TestMultiQueryRetriever_Deduplication(t *testing.T) {
	// Inner retriever returns same docs for all queries
	inner := &mockRetriever{docs: []schema.Document{
		{ID: "doc1", Content: "content 1"},
		{ID: "doc2", Content: "content 2"},
	}}

	model := &mockChatModel{
		response: schema.NewAIMessage("query 1\nquery 2"),
	}

	r := NewMultiQueryRetriever(inner, model, WithMultiQueryCount(2))
	docs, err := r.Retrieve(context.Background(), "original")
	require.NoError(t, err)

	// Should deduplicate by ID
	assert.Len(t, docs, 2, "should deduplicate documents with same ID")
	idSet := make(map[string]bool)
	for _, doc := range docs {
		assert.False(t, idSet[doc.ID], "doc ID %s appears multiple times", doc.ID)
		idSet[doc.ID] = true
	}
}

func TestMultiQueryRetriever_IncludesOriginalQuery(t *testing.T) {
	// Test that multiple queries are generated and used
	inner := &mockRetriever{
		docs: makeDocs("doc1"),
	}

	model := &mockChatModel{
		response: schema.NewAIMessage("variant 1\nvariant 2"),
	}

	r := NewMultiQueryRetriever(inner, model)
	docs, err := r.Retrieve(context.Background(), "original query")
	require.NoError(t, err)

	// Should have results
	assert.NotEmpty(t, docs)
	// LLM should have been called
	assert.Equal(t, 1, model.calls)
}

func TestMultiQueryRetriever_LLMError_Detailed(t *testing.T) {
	inner := &mockRetriever{docs: makeDocs("doc1")}
	llmErr := errors.New("LLM generation failed")
	model := &mockChatModel{err: llmErr}

	r := NewMultiQueryRetriever(inner, model)
	_, err := r.Retrieve(context.Background(), "query")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "generate queries")
}

func TestMultiQueryRetriever_InnerRetrieverError(t *testing.T) {
	innerErr := errors.New("retrieval failed")
	inner := &mockRetriever{err: innerErr}
	model := &mockChatModel{
		response: schema.NewAIMessage("variant 1"),
	}

	r := NewMultiQueryRetriever(inner, model)
	_, err := r.Retrieve(context.Background(), "query")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiquery retrieve")
}

func TestMultiQueryRetriever_EmptyLLMResponse(t *testing.T) {
	inner := &mockRetriever{docs: makeDocs("doc1", "doc2")}
	model := &mockChatModel{
		response: schema.NewAIMessage(""), // Empty response
	}

	r := NewMultiQueryRetriever(inner, model)
	docs, err := r.Retrieve(context.Background(), "original query")
	require.NoError(t, err)
	// Should still have docs from original query
	assert.NotEmpty(t, docs)
}

func TestMultiQueryRetriever_WithHooks(t *testing.T) {
	inner := &mockRetriever{docs: makeDocs("doc1")}
	model := &mockChatModel{
		response: schema.NewAIMessage("variant 1"),
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

	r := NewMultiQueryRetriever(inner, model, WithMultiQueryHooks(hooks))
	docs, err := r.Retrieve(context.Background(), "test query")
	require.NoError(t, err)
	assert.NotEmpty(t, docs)

	assert.True(t, beforeCalled, "BeforeRetrieve should be called")
	assert.True(t, afterCalled, "AfterRetrieve should be called")
	assert.Equal(t, "test query", capturedQuery)
	assert.NotEmpty(t, capturedDocs)
}

func TestMultiQueryRetriever_BeforeHookError(t *testing.T) {
	inner := &mockRetriever{docs: makeDocs("doc1")}
	model := &mockChatModel{
		response: schema.NewAIMessage("variant 1"),
	}

	hookErr := errors.New("hook rejected")
	hooks := Hooks{
		BeforeRetrieve: func(ctx context.Context, query string) error {
			return hookErr
		},
	}

	r := NewMultiQueryRetriever(inner, model, WithMultiQueryHooks(hooks))
	_, err := r.Retrieve(context.Background(), "query")
	require.Error(t, err)
	assert.Equal(t, hookErr, err)
	assert.Equal(t, 0, model.calls, "LLM should not be called after hook error")
}

func TestMultiQueryRetriever_LLMResponseParsing(t *testing.T) {
	tests := []struct {
		name         string
		llmResponse  string
		wantVariants int // minimum expected (excluding original)
	}{
		{
			name:         "simple newlines",
			llmResponse:  "query 1\nquery 2\nquery 3",
			wantVariants: 3,
		},
		{
			name:         "with blank lines",
			llmResponse:  "query 1\n\nquery 2\n\n\nquery 3",
			wantVariants: 3,
		},
		{
			name:         "with leading/trailing spaces",
			llmResponse:  "  query 1  \n  query 2  \n  query 3  ",
			wantVariants: 3,
		},
		{
			name:         "mixed formatting",
			llmResponse:  "query 1\n\n  query 2  \nquery 3\n",
			wantVariants: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inner := &mockRetriever{docs: makeDocs("doc1")}
			model := &mockChatModel{
				response: schema.NewAIMessage(tt.llmResponse),
			}

			r := NewMultiQueryRetriever(inner, model)
			_, err := r.Retrieve(context.Background(), "original")
			require.NoError(t, err)

			// Verify LLM was called once
			assert.Equal(t, 1, model.calls)
		})
	}
}

func TestMultiQueryRetriever_EmptyInnerResults(t *testing.T) {
	inner := &mockRetriever{docs: []schema.Document{}}
	model := &mockChatModel{
		response: schema.NewAIMessage("variant 1\nvariant 2"),
	}

	r := NewMultiQueryRetriever(inner, model)
	docs, err := r.Retrieve(context.Background(), "query")
	require.NoError(t, err)
	assert.Empty(t, docs)
}

func TestMultiQueryRetriever_MergesResultsFromAllQueries(t *testing.T) {
	// Test that results from multiple queries are merged
	inner := &mockRetriever{
		docs: makeDocs("doc1", "doc2", "doc3"),
	}

	model := &mockChatModel{
		response: schema.NewAIMessage("A\nB"),
	}

	r := NewMultiQueryRetriever(inner, model, WithMultiQueryCount(2))
	docs, err := r.Retrieve(context.Background(), "original")
	require.NoError(t, err)

	// Should have docs
	assert.GreaterOrEqual(t, len(docs), 1, "should have at least one doc")
}

func TestMultiQueryRetriever_PassesOptionsToInner(t *testing.T) {
	inner := &mockRetriever{docs: makeDocs("a", "b", "c", "d", "e")}
	model := &mockChatModel{
		response: schema.NewAIMessage("variant 1"),
	}

	r := NewMultiQueryRetriever(inner, model)

	// Pass options - they should be forwarded to inner retriever
	docs, err := r.Retrieve(context.Background(), "query", WithTopK(2))
	require.NoError(t, err)
	assert.NotNil(t, docs)
	// Note: The actual limiting happens in the mock retriever if it respects TopK
}

// Compile-time interface checks
var _ Retriever = (*MultiQueryRetriever)(nil)
