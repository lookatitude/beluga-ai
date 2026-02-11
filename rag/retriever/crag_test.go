package retriever_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/rag/retriever"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCRAGRetriever_Defaults(t *testing.T) {
	inner := &mockRetriever{}
	model := &mockChatModel{}
	web := &mockWebSearcher{}

	r := retriever.NewCRAGRetriever(inner, model, web)
	require.NotNil(t, r)
}

func TestNewCRAGRetriever_WithThreshold(t *testing.T) {
	inner := &mockRetriever{}
	model := &mockChatModel{}
	web := &mockWebSearcher{}

	r := retriever.NewCRAGRetriever(inner, model, web, retriever.WithCRAGThreshold(0.5))
	require.NotNil(t, r)
}

func TestCRAGRetriever_Retrieve_RelevantDocs(t *testing.T) {
	innerDocs := makeDocs("doc1", "doc2")

	inner := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return innerDocs, nil
		},
	}

	// LLM returns score above threshold for each document.
	callCount := 0
	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			callCount++
			return schema.NewAIMessage("0.8"), nil
		},
	}

	web := &mockWebSearcher{}

	r := retriever.NewCRAGRetriever(inner, model, web, retriever.WithCRAGThreshold(0.5))
	docs, err := r.Retrieve(context.Background(), "relevant query")

	require.NoError(t, err)
	assert.Len(t, docs, 2, "Should return all relevant docs")
	assert.Equal(t, 2, callCount, "LLM should evaluate each document")
	// Scores should be updated.
	for _, doc := range docs {
		assert.Equal(t, 0.8, doc.Score)
	}
}

func TestCRAGRetriever_Retrieve_IrrelevantDocs_FallbackToWeb(t *testing.T) {
	innerDocs := makeDocs("doc1")

	inner := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return innerDocs, nil
		},
	}

	// LLM returns score below threshold.
	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("-0.5"), nil
		},
	}

	webDocs := makeDocs("web1", "web2")
	web := &mockWebSearcher{
		searchFn: func(ctx context.Context, query string, k int) ([]schema.Document, error) {
			return webDocs, nil
		},
	}

	r := retriever.NewCRAGRetriever(inner, model, web, retriever.WithCRAGThreshold(0.0))
	docs, err := r.Retrieve(context.Background(), "irrelevant query")

	require.NoError(t, err)
	assert.Equal(t, webDocs, docs, "Should return web search results")
}

func TestCRAGRetriever_Retrieve_EmptyInner_FallbackToWeb(t *testing.T) {
	inner := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return nil, nil
		},
	}

	model := &mockChatModel{}

	webDocs := makeDocs("web1")
	web := &mockWebSearcher{
		searchFn: func(ctx context.Context, query string, k int) ([]schema.Document, error) {
			return webDocs, nil
		},
	}

	r := retriever.NewCRAGRetriever(inner, model, web)
	docs, err := r.Retrieve(context.Background(), "query")

	require.NoError(t, err)
	assert.Equal(t, webDocs, docs, "Should fallback to web for empty inner results")
}

func TestCRAGRetriever_Retrieve_NilWebSearcher_ReturnsNil(t *testing.T) {
	inner := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return makeDocs("doc1"), nil
		},
	}

	// LLM returns score below threshold.
	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("-0.5"), nil
		},
	}

	r := retriever.NewCRAGRetriever(inner, model, nil, retriever.WithCRAGThreshold(0.0))
	docs, err := r.Retrieve(context.Background(), "query")

	require.NoError(t, err)
	assert.Nil(t, docs, "Should return nil when no web searcher available")
}

func TestCRAGRetriever_Retrieve_InnerError(t *testing.T) {
	expectedErr := errors.New("inner retriever failure")
	inner := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return nil, expectedErr
		},
	}

	model := &mockChatModel{}
	web := &mockWebSearcher{}

	r := retriever.NewCRAGRetriever(inner, model, web)
	_, err := r.Retrieve(context.Background(), "query")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "crag inner retrieve")
}

func TestCRAGRetriever_Retrieve_LLMEvaluationError(t *testing.T) {
	inner := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return makeDocs("doc1"), nil
		},
	}

	expectedErr := errors.New("llm failure")
	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return nil, expectedErr
		},
	}

	web := &mockWebSearcher{}

	r := retriever.NewCRAGRetriever(inner, model, web)
	_, err := r.Retrieve(context.Background(), "query")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "crag evaluate")
}

func TestCRAGRetriever_ScoreDocument_ValidFloat(t *testing.T) {
	inner := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return makeDocs("doc1"), nil
		},
	}

	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("0.75"), nil
		},
	}

	web := &mockWebSearcher{}

	r := retriever.NewCRAGRetriever(inner, model, web)
	docs, err := r.Retrieve(context.Background(), "query")

	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, 0.75, docs[0].Score)
}

func TestCRAGRetriever_ScoreDocument_ClampAboveOne(t *testing.T) {
	inner := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return makeDocs("doc1"), nil
		},
	}

	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("2.5"), nil
		},
	}

	web := &mockWebSearcher{}

	r := retriever.NewCRAGRetriever(inner, model, web)
	docs, err := r.Retrieve(context.Background(), "query")

	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, 1.0, docs[0].Score, "Score should be clamped to 1.0")
}

func TestCRAGRetriever_ScoreDocument_ClampBelowNegativeOne(t *testing.T) {
	inner := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return makeDocs("doc1"), nil
		},
	}

	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("-3.0"), nil
		},
	}

	webDocs := makeDocs("web1")
	web := &mockWebSearcher{
		searchFn: func(ctx context.Context, query string, k int) ([]schema.Document, error) {
			return webDocs, nil
		},
	}

	r := retriever.NewCRAGRetriever(inner, model, web, retriever.WithCRAGThreshold(0.0))
	docs, err := r.Retrieve(context.Background(), "query")

	require.NoError(t, err)
	// Score -3.0 clamped to -1.0, which is < threshold 0.0, so web fallback.
	assert.Equal(t, webDocs, docs)
}

func TestCRAGRetriever_Retrieve_WithTopK(t *testing.T) {
	innerDocs := makeDocs("doc1", "doc2", "doc3", "doc4")

	inner := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return innerDocs, nil
		},
	}

	// All docs score above threshold.
	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("0.8"), nil
		},
	}

	web := &mockWebSearcher{}

	r := retriever.NewCRAGRetriever(inner, model, web)
	docs, err := r.Retrieve(context.Background(), "query", retriever.WithTopK(2))

	require.NoError(t, err)
	assert.Len(t, docs, 2, "Should respect TopK limit")
}

func TestCRAGRetriever_Retrieve_WithHooks(t *testing.T) {
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

	innerDocs := makeDocs("doc1")
	inner := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return innerDocs, nil
		},
	}

	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("0.8"), nil
		},
	}

	web := &mockWebSearcher{}

	r := retriever.NewCRAGRetriever(inner, model, web, retriever.WithCRAGHooks(hooks))
	docs, err := r.Retrieve(context.Background(), "test query")

	require.NoError(t, err)
	assert.True(t, beforeCalled, "BeforeRetrieve should be called")
	assert.True(t, afterCalled, "AfterRetrieve should be called")
	assert.Equal(t, "test query", capturedQuery)
	assert.NotNil(t, capturedDocs)
	assert.NotNil(t, docs)
}

func TestCRAGRetriever_Retrieve_BeforeHookError(t *testing.T) {
	expectedErr := errors.New("hook error")
	hooks := retriever.Hooks{
		BeforeRetrieve: func(ctx context.Context, query string) error {
			return expectedErr
		},
	}

	inner := &mockRetriever{}
	model := &mockChatModel{}
	web := &mockWebSearcher{}

	r := retriever.NewCRAGRetriever(inner, model, web, retriever.WithCRAGHooks(hooks))
	_, err := r.Retrieve(context.Background(), "query")

	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestCRAGRetriever_Retrieve_MixedRelevance(t *testing.T) {
	// Some docs relevant, some not.
	innerDocs := []schema.Document{
		{ID: "relevant1", Content: "good content"},
		{ID: "irrelevant", Content: "bad content"},
		{ID: "relevant2", Content: "good content"},
	}

	inner := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return innerDocs, nil
		},
	}

	callCount := 0
	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			callCount++
			// First and third docs are relevant.
			if callCount == 1 || callCount == 3 {
				return schema.NewAIMessage("0.7"), nil
			}
			return schema.NewAIMessage("-0.3"), nil
		},
	}

	web := &mockWebSearcher{}

	r := retriever.NewCRAGRetriever(inner, model, web, retriever.WithCRAGThreshold(0.0))
	docs, err := r.Retrieve(context.Background(), "query")

	require.NoError(t, err)
	assert.Len(t, docs, 2, "Should return only relevant docs")
	assert.Equal(t, "relevant1", docs[0].ID)
	assert.Equal(t, "relevant2", docs[1].ID)
}

func TestCRAGRetriever_FallbackSearch_DefaultTopK(t *testing.T) {
	inner := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return nil, nil
		},
	}

	model := &mockChatModel{}

	var requestedK int
	web := &mockWebSearcher{
		searchFn: func(ctx context.Context, query string, k int) ([]schema.Document, error) {
			requestedK = k
			return makeDocs("web1"), nil
		},
	}

	r := retriever.NewCRAGRetriever(inner, model, web)
	_, err := r.Retrieve(context.Background(), "query")

	require.NoError(t, err)
	assert.Equal(t, 10, requestedK, "Should use default TopK=10 for web search")
}

func TestCRAGRetriever_FallbackSearch_CustomTopK(t *testing.T) {
	inner := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return nil, nil
		},
	}

	model := &mockChatModel{}

	var requestedK int
	web := &mockWebSearcher{
		searchFn: func(ctx context.Context, query string, k int) ([]schema.Document, error) {
			requestedK = k
			return makeDocs("web1"), nil
		},
	}

	r := retriever.NewCRAGRetriever(inner, model, web)
	_, err := r.Retrieve(context.Background(), "query", retriever.WithTopK(5))

	require.NoError(t, err)
	assert.Equal(t, 5, requestedK, "Should use custom TopK for web search")
}

func TestCRAGRetriever_ScoreDocument_InvalidFloat(t *testing.T) {
	inner := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return makeDocs("doc1"), nil
		},
	}

	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("not-a-number"), nil
		},
	}

	web := &mockWebSearcher{}

	r := retriever.NewCRAGRetriever(inner, model, web)
	_, err := r.Retrieve(context.Background(), "query")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "crag evaluate")
}

func TestCRAGRetriever_FallbackSearch_WebError(t *testing.T) {
	inner := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return nil, nil // Empty results trigger fallback
		},
	}

	model := &mockChatModel{}

	expectedErr := errors.New("web search failure")
	web := &mockWebSearcher{
		searchFn: func(ctx context.Context, query string, k int) ([]schema.Document, error) {
			return nil, expectedErr
		},
	}

	r := retriever.NewCRAGRetriever(inner, model, web)
	_, err := r.Retrieve(context.Background(), "query")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "crag web search")
}

func TestCRAGRetriever_FallbackSearch_WithAfterHook(t *testing.T) {
	// Test that AfterRetrieve hook is called in fallback path with no web searcher.
	var afterCalled bool
	var capturedDocs []schema.Document

	hooks := retriever.Hooks{
		AfterRetrieve: func(ctx context.Context, docs []schema.Document, err error) {
			afterCalled = true
			capturedDocs = docs
		},
	}

	inner := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return nil, nil
		},
	}

	model := &mockChatModel{}

	r := retriever.NewCRAGRetriever(inner, model, nil, retriever.WithCRAGHooks(hooks))
	_, err := r.Retrieve(context.Background(), "query")

	require.NoError(t, err)
	assert.True(t, afterCalled)
	assert.Nil(t, capturedDocs)
}
