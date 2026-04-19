package retriever_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/rag/retriever"
	"github.com/lookatitude/beluga-ai/v2/schema"
	"github.com/lookatitude/beluga-ai/v2/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAsTool_Name(t *testing.T) {
	inner := &mockRetriever{}
	tl := retriever.AsTool(inner, "search_docs", "Search documents")

	assert.Equal(t, "search_docs", tl.Name())
	assert.Equal(t, "Search documents", tl.Description())
}

func TestAsTool_InputSchema(t *testing.T) {
	inner := &mockRetriever{}
	tl := retriever.AsTool(inner, "search", "Search")

	schema := tl.InputSchema()
	require.NotNil(t, schema)
	assert.Equal(t, "object", schema["type"])

	props, ok := schema["properties"].(map[string]any)
	require.True(t, ok)
	queryProp, ok := props["query"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", queryProp["type"])

	required, ok := schema["required"].([]string)
	require.True(t, ok)
	assert.Contains(t, required, "query")
}

func TestAsTool_Execute_HappyPath(t *testing.T) {
	docs := []schema.Document{
		{ID: "d1", Content: "First document", Score: 0.9},
		{ID: "d2", Content: "Second document", Score: 0.7},
	}

	inner := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			assert.Equal(t, "test query", query)
			return docs, nil
		},
	}

	tl := retriever.AsTool(inner, "search", "Search")
	result, err := tl.Execute(context.Background(), map[string]any{
		"query": "test query",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)
	assert.Len(t, result.Content, 1)

	text := result.Content[0].(schema.TextPart).Text
	assert.Contains(t, text, "First document")
	assert.Contains(t, text, "Second document")
	assert.Contains(t, text, "0.90")
}

func TestAsTool_Execute_EmptyResults(t *testing.T) {
	inner := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return nil, nil
		},
	}

	tl := retriever.AsTool(inner, "search", "Search")
	result, err := tl.Execute(context.Background(), map[string]any{
		"query": "test",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)
	text := result.Content[0].(schema.TextPart).Text
	assert.Equal(t, "No documents found.", text)
}

func TestAsTool_Execute_RetrieverError(t *testing.T) {
	inner := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return nil, errors.New("search failed")
		},
	}

	tl := retriever.AsTool(inner, "search", "Search")
	result, err := tl.Execute(context.Background(), map[string]any{
		"query": "test",
	})

	// Error is returned as an error result, not a Go error.
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestAsTool_Execute_MissingQuery(t *testing.T) {
	inner := &mockRetriever{}
	tl := retriever.AsTool(inner, "search", "Search")

	_, err := tl.Execute(context.Background(), map[string]any{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required field")
}

func TestAsTool_Execute_InvalidQueryType(t *testing.T) {
	inner := &mockRetriever{}
	tl := retriever.AsTool(inner, "search", "Search")

	_, err := tl.Execute(context.Background(), map[string]any{
		"query": 42,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be a string")
}

func TestAsTool_Execute_EmptyQuery(t *testing.T) {
	inner := &mockRetriever{}
	tl := retriever.AsTool(inner, "search", "Search")

	_, err := tl.Execute(context.Background(), map[string]any{
		"query": "",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "must not be empty")
}

func TestAsTool_WithToolTopK(t *testing.T) {
	var capturedOpts []retriever.Option

	inner := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			capturedOpts = opts
			return nil, nil
		},
	}

	tl := retriever.AsTool(inner, "search", "Search", retriever.WithToolTopK(5))
	_, err := tl.Execute(context.Background(), map[string]any{
		"query": "test",
	})

	require.NoError(t, err)
	// Verify TopK was passed through options.
	cfg := retriever.ApplyOptions(capturedOpts...)
	assert.Equal(t, 5, cfg.TopK)
}

// Compile-time check.
var _ tool.Tool = retriever.AsTool(&mockRetriever{}, "test", "test")
