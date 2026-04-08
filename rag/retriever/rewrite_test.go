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

func TestNewQueryRewriter_Defaults(t *testing.T) {
	inner := &mockRetriever{}
	model := &mockChatModel{}

	r := retriever.NewQueryRewriter(inner, model)
	require.NotNil(t, r)
}

func TestQueryRewriter_Retrieve_HighRelevance_NoRewrite(t *testing.T) {
	// Documents with high scores should be returned without rewriting.
	highScoreDocs := []schema.Document{
		{ID: "d1", Content: "relevant", Score: 0.9},
		{ID: "d2", Content: "also relevant", Score: 0.8},
	}

	inner := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return highScoreDocs, nil
		},
	}

	model := &mockChatModel{}

	r := retriever.NewQueryRewriter(inner, model, retriever.WithRelevanceThreshold(0.7))
	docs, err := r.Retrieve(context.Background(), "test query")

	require.NoError(t, err)
	assert.Len(t, docs, 2)
	assert.Equal(t, 0, model.calls, "LLM should not be called when relevance is high")
}

func TestQueryRewriter_Retrieve_LowRelevance_Rewrites(t *testing.T) {
	callCount := 0

	inner := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			callCount++
			if callCount == 1 {
				// First attempt: low relevance docs.
				return []schema.Document{
					{ID: "d1", Content: "low", Score: 0.3},
				}, nil
			}
			// After rewrite: high relevance docs.
			return []schema.Document{
				{ID: "d2", Content: "high", Score: 0.9},
			}, nil
		},
	}

	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("improved query"), nil
		},
	}

	r := retriever.NewQueryRewriter(inner, model,
		retriever.WithRelevanceThreshold(0.7),
		retriever.WithMaxRewrites(3),
	)
	docs, err := r.Retrieve(context.Background(), "test query")

	require.NoError(t, err)
	assert.Len(t, docs, 1)
	assert.Equal(t, "d2", docs[0].ID)
	assert.Equal(t, 2, callCount, "Should have retried after rewrite")
	assert.Equal(t, 1, model.calls, "LLM should be called once for rewrite")
}

func TestQueryRewriter_Retrieve_MaxRewritesExhausted(t *testing.T) {
	inner := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			// Always return low relevance docs.
			return []schema.Document{
				{ID: "d1", Content: "low", Score: 0.2},
			}, nil
		},
	}

	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("rewritten query"), nil
		},
	}

	r := retriever.NewQueryRewriter(inner, model,
		retriever.WithRelevanceThreshold(0.7),
		retriever.WithMaxRewrites(2),
	)
	docs, err := r.Retrieve(context.Background(), "test query")

	require.NoError(t, err)
	// Should return the last attempt's docs even though score is low.
	assert.Len(t, docs, 1)
	assert.Equal(t, "d1", docs[0].ID)
}

func TestQueryRewriter_Retrieve_EmptyResults_RewritesAndRetries(t *testing.T) {
	callCount := 0

	inner := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			callCount++
			if callCount < 3 {
				return nil, nil // Empty results.
			}
			return []schema.Document{
				{ID: "d1", Content: "found", Score: 0.9},
			}, nil
		},
	}

	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("better query"), nil
		},
	}

	r := retriever.NewQueryRewriter(inner, model,
		retriever.WithMaxRewrites(3),
		retriever.WithRelevanceThreshold(0.5),
	)
	docs, err := r.Retrieve(context.Background(), "query")

	require.NoError(t, err)
	assert.Len(t, docs, 1)
	assert.Equal(t, 3, callCount)
}

func TestQueryRewriter_Retrieve_InnerError(t *testing.T) {
	expectedErr := errors.New("inner failure")
	inner := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return nil, expectedErr
		},
	}

	model := &mockChatModel{}

	r := retriever.NewQueryRewriter(inner, model)
	_, err := r.Retrieve(context.Background(), "query")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "rewrite retrieve")
}

func TestQueryRewriter_Retrieve_RewriteError(t *testing.T) {
	inner := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return []schema.Document{
				{ID: "d1", Content: "low", Score: 0.1},
			}, nil
		},
	}

	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return nil, errors.New("llm failure")
		},
	}

	r := retriever.NewQueryRewriter(inner, model,
		retriever.WithRelevanceThreshold(0.9),
	)
	_, err := r.Retrieve(context.Background(), "query")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "rewrite query")
}

func TestQueryRewriter_Retrieve_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	inner := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return []schema.Document{{ID: "d1", Score: 0.1}}, nil
		},
	}

	model := &mockChatModel{}

	r := retriever.NewQueryRewriter(inner, model, retriever.WithRelevanceThreshold(0.9))
	_, err := r.Retrieve(ctx, "query")

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestQueryRewriter_Retrieve_WithRewriteModel(t *testing.T) {
	inner := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			if query == "rewritten" {
				return []schema.Document{{ID: "d1", Score: 0.9}}, nil
			}
			return []schema.Document{{ID: "d1", Score: 0.1}}, nil
		},
	}

	mainModel := &mockChatModel{}
	rewriteModel := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("rewritten"), nil
		},
	}

	r := retriever.NewQueryRewriter(inner, mainModel,
		retriever.WithRewriteModel(rewriteModel),
		retriever.WithRelevanceThreshold(0.5),
	)
	docs, err := r.Retrieve(context.Background(), "query")

	require.NoError(t, err)
	assert.Len(t, docs, 1)
	assert.Equal(t, 0, mainModel.calls, "Main model should not be called for rewrites")
	assert.Equal(t, 1, rewriteModel.calls, "Rewrite model should be called")
}

func TestQueryRewriter_Retrieve_WithHooks(t *testing.T) {
	var beforeCalled, afterCalled bool

	hooks := retriever.Hooks{
		BeforeRetrieve: func(ctx context.Context, query string) error {
			beforeCalled = true
			return nil
		},
		AfterRetrieve: func(ctx context.Context, docs []schema.Document, err error) {
			afterCalled = true
		},
	}

	inner := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return []schema.Document{{ID: "d1", Score: 0.9}}, nil
		},
	}

	model := &mockChatModel{}

	r := retriever.NewQueryRewriter(inner, model, retriever.WithRewriteHooks(hooks))
	_, err := r.Retrieve(context.Background(), "query")

	require.NoError(t, err)
	assert.True(t, beforeCalled)
	assert.True(t, afterCalled)
}

func TestQueryRewriter_Retrieve_BeforeHookError(t *testing.T) {
	expectedErr := errors.New("hook error")
	hooks := retriever.Hooks{
		BeforeRetrieve: func(ctx context.Context, query string) error {
			return expectedErr
		},
	}

	inner := &mockRetriever{}
	model := &mockChatModel{}

	r := retriever.NewQueryRewriter(inner, model, retriever.WithRewriteHooks(hooks))
	_, err := r.Retrieve(context.Background(), "query")

	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

// Compile-time check.
var _ retriever.Retriever = (*retriever.QueryRewriter)(nil)
