package retriever_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/rag/retriever"
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHyDERetriever_Defaults(t *testing.T) {
	model := &mockChatModel{}
	embedder := &mockEmbedder{}
	store := &mockVectorStore{}

	r := retriever.NewHyDERetriever(model, embedder, store)
	require.NotNil(t, r)
}

func TestNewHyDERetriever_WithCustomPrompt(t *testing.T) {
	model := &mockChatModel{}
	embedder := &mockEmbedder{}
	store := &mockVectorStore{}

	customPrompt := "Custom prompt: %s"
	r := retriever.NewHyDERetriever(model, embedder, store, retriever.WithHyDEPrompt(customPrompt))
	require.NotNil(t, r)
	// Verify custom prompt is used by checking generated message.
}

func TestHyDERetriever_Retrieve_HappyPath(t *testing.T) {
	hypotheticalDoc := "This is a hypothetical answer about Go programming."
	expectedDocs := makeDocs("doc1", "doc2", "doc3")

	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage(hypotheticalDoc), nil
		},
	}

	embedder := &mockEmbedder{
		embedSingleFn: func(ctx context.Context, text string) ([]float32, error) {
			assert.Equal(t, hypotheticalDoc, text)
			return []float32{1.0, 0.5, 0.3}, nil
		},
	}

	store := &mockVectorStore{
		searchFn: func(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
			assert.Equal(t, []float32{1.0, 0.5, 0.3}, query)
			assert.Equal(t, 10, k) // Default TopK
			return expectedDocs, nil
		},
	}

	r := retriever.NewHyDERetriever(model, embedder, store)
	docs, err := r.Retrieve(context.Background(), "what is Go?")

	require.NoError(t, err)
	assert.Equal(t, expectedDocs, docs)
	assert.Equal(t, 1, model.calls, "LLM should be called once")
}

func TestHyDERetriever_Retrieve_WithTopK(t *testing.T) {
	expectedDocs := makeDocs("doc1", "doc2")

	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("hypothetical answer"), nil
		},
	}

	embedder := &mockEmbedder{}

	store := &mockVectorStore{
		searchFn: func(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
			assert.Equal(t, 5, k)
			return expectedDocs, nil
		},
	}

	r := retriever.NewHyDERetriever(model, embedder, store)
	docs, err := r.Retrieve(context.Background(), "query", retriever.WithTopK(5))

	require.NoError(t, err)
	assert.Equal(t, expectedDocs, docs)
}

func TestHyDERetriever_Retrieve_WithThreshold(t *testing.T) {
	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("hypothetical answer"), nil
		},
	}

	embedder := &mockEmbedder{}

	var receivedOpts []vectorstore.SearchOption
	store := &mockVectorStore{
		searchFn: func(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
			receivedOpts = opts
			return makeDocs("doc1"), nil
		},
	}

	r := retriever.NewHyDERetriever(model, embedder, store)
	_, err := r.Retrieve(context.Background(), "query", retriever.WithThreshold(0.7))

	require.NoError(t, err)
	assert.NotEmpty(t, receivedOpts, "Threshold should be passed to vector store")
}

func TestHyDERetriever_Retrieve_WithMetadata(t *testing.T) {
	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("hypothetical answer"), nil
		},
	}

	embedder := &mockEmbedder{}

	metadata := map[string]any{"author": "test", "year": 2024}
	var receivedOpts []vectorstore.SearchOption
	store := &mockVectorStore{
		searchFn: func(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
			receivedOpts = opts
			return makeDocs("doc1"), nil
		},
	}

	r := retriever.NewHyDERetriever(model, embedder, store)
	_, err := r.Retrieve(context.Background(), "query", retriever.WithMetadata(metadata))

	require.NoError(t, err)
	assert.NotEmpty(t, receivedOpts, "Metadata should be passed to vector store")
}

func TestHyDERetriever_Retrieve_LLMError(t *testing.T) {
	expectedErr := errors.New("llm failure")
	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return nil, expectedErr
		},
	}

	embedder := &mockEmbedder{}
	store := &mockVectorStore{}

	r := retriever.NewHyDERetriever(model, embedder, store)
	_, err := r.Retrieve(context.Background(), "query")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "hyde generate")
}

func TestHyDERetriever_Retrieve_EmbedError(t *testing.T) {
	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("hypothetical answer"), nil
		},
	}

	expectedErr := errors.New("embed failure")
	embedder := &mockEmbedder{
		embedSingleFn: func(ctx context.Context, text string) ([]float32, error) {
			return nil, expectedErr
		},
	}

	store := &mockVectorStore{}

	r := retriever.NewHyDERetriever(model, embedder, store)
	_, err := r.Retrieve(context.Background(), "query")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "hyde embed")
}

func TestHyDERetriever_Retrieve_VectorStoreError(t *testing.T) {
	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("hypothetical answer"), nil
		},
	}

	embedder := &mockEmbedder{}

	expectedErr := errors.New("vector store failure")
	store := &mockVectorStore{
		searchFn: func(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
			return nil, expectedErr
		},
	}

	r := retriever.NewHyDERetriever(model, embedder, store)
	_, err := r.Retrieve(context.Background(), "query")

	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestHyDERetriever_Retrieve_WithHooks(t *testing.T) {
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

	expectedDocs := makeDocs("doc1")
	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("hypothetical answer"), nil
		},
	}

	embedder := &mockEmbedder{}
	store := &mockVectorStore{
		searchFn: func(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
			return expectedDocs, nil
		},
	}

	r := retriever.NewHyDERetriever(model, embedder, store, retriever.WithHyDEHooks(hooks))
	docs, err := r.Retrieve(context.Background(), "test query")

	require.NoError(t, err)
	assert.True(t, beforeCalled, "BeforeRetrieve should be called")
	assert.True(t, afterCalled, "AfterRetrieve should be called")
	assert.Equal(t, "test query", capturedQuery)
	assert.Equal(t, expectedDocs, capturedDocs)
	assert.Equal(t, expectedDocs, docs)
}

func TestHyDERetriever_Retrieve_BeforeHookError(t *testing.T) {
	expectedErr := errors.New("hook error")
	hooks := retriever.Hooks{
		BeforeRetrieve: func(ctx context.Context, query string) error {
			return expectedErr
		},
	}

	model := &mockChatModel{}
	embedder := &mockEmbedder{}
	store := &mockVectorStore{}

	r := retriever.NewHyDERetriever(model, embedder, store, retriever.WithHyDEHooks(hooks))
	_, err := r.Retrieve(context.Background(), "query")

	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, 0, model.calls, "LLM should not be called if hook fails")
}

func TestHyDERetriever_CustomPromptFormat(t *testing.T) {
	customPrompt := "Generate answer for: %s"
	query := "what is Go?"
	var receivedPrompt string

	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			require.Len(t, msgs, 1)
			receivedPrompt = msgs[0].(*schema.HumanMessage).Text()
			return schema.NewAIMessage("answer"), nil
		},
	}

	embedder := &mockEmbedder{}
	store := &mockVectorStore{
		searchFn: func(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
			return makeDocs("doc1"), nil
		},
	}

	r := retriever.NewHyDERetriever(model, embedder, store, retriever.WithHyDEPrompt(customPrompt))
	_, err := r.Retrieve(context.Background(), query)

	require.NoError(t, err)
	assert.Contains(t, receivedPrompt, query)
	assert.Contains(t, receivedPrompt, "Generate answer for:")
}
