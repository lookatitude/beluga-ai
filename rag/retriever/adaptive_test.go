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

func TestNewAdaptiveRetriever_WithBothRetrievers(t *testing.T) {
	model := &mockChatModel{}
	simple := &mockRetriever{}
	complex := &mockRetriever{}

	r := retriever.NewAdaptiveRetriever(model, simple, complex)
	require.NotNil(t, r)
}

func TestNewAdaptiveRetriever_NilComplexRetriever(t *testing.T) {
	model := &mockChatModel{}
	simple := &mockRetriever{}

	// If complex is nil, should use simple for both.
	r := retriever.NewAdaptiveRetriever(model, simple, nil)
	require.NotNil(t, r)
}

func TestAdaptiveRetriever_Retrieve_NoRetrieval(t *testing.T) {
	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("no_retrieval"), nil
		},
	}

	simpleDocs := makeDocs("simple")
	simple := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			t.Error("simple retriever should not be called for no_retrieval")
			return simpleDocs, nil
		},
	}

	r := retriever.NewAdaptiveRetriever(model, simple, nil)
	docs, err := r.Retrieve(context.Background(), "what is 2+2?")

	require.NoError(t, err)
	assert.Nil(t, docs, "Should return nil for no_retrieval classification")
}

func TestAdaptiveRetriever_Retrieve_Simple(t *testing.T) {
	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("simple"), nil
		},
	}

	simpleDocs := makeDocs("simple1", "simple2")
	var simpleCalled bool
	simple := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			simpleCalled = true
			return simpleDocs, nil
		},
	}

	complex := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			t.Error("complex retriever should not be called for simple classification")
			return nil, nil
		},
	}

	r := retriever.NewAdaptiveRetriever(model, simple, complex)
	docs, err := r.Retrieve(context.Background(), "what is Go?")

	require.NoError(t, err)
	assert.True(t, simpleCalled, "Simple retriever should be called")
	assert.Equal(t, simpleDocs, docs)
}

func TestAdaptiveRetriever_Retrieve_Complex(t *testing.T) {
	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("complex"), nil
		},
	}

	simple := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			t.Error("simple retriever should not be called for complex classification")
			return nil, nil
		},
	}

	complexDocs := makeDocs("complex1", "complex2")
	var complexCalled bool
	complex := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			complexCalled = true
			return complexDocs, nil
		},
	}

	r := retriever.NewAdaptiveRetriever(model, simple, complex)
	docs, err := r.Retrieve(context.Background(), "compare Go and Rust concurrency models")

	require.NoError(t, err)
	assert.True(t, complexCalled, "Complex retriever should be called")
	assert.Equal(t, complexDocs, docs)
}

func TestAdaptiveRetriever_Retrieve_UnrecognizedClassification(t *testing.T) {
	// LLM returns something unrecognized - should default to simple.
	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("unknown_classification"), nil
		},
	}

	simpleDocs := makeDocs("simple1")
	var simpleCalled bool
	simple := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			simpleCalled = true
			return simpleDocs, nil
		},
	}

	complex := &mockRetriever{}

	r := retriever.NewAdaptiveRetriever(model, simple, complex)
	docs, err := r.Retrieve(context.Background(), "query")

	require.NoError(t, err)
	assert.True(t, simpleCalled, "Should default to simple retriever for unrecognized classification")
	assert.Equal(t, simpleDocs, docs)
}

func TestAdaptiveRetriever_Retrieve_ClassificationError(t *testing.T) {
	expectedErr := errors.New("llm failure")
	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return nil, expectedErr
		},
	}

	simple := &mockRetriever{}
	complex := &mockRetriever{}

	r := retriever.NewAdaptiveRetriever(model, simple, complex)
	_, err := r.Retrieve(context.Background(), "query")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "adaptive classify")
}

func TestAdaptiveRetriever_Retrieve_SimpleRetrieverError(t *testing.T) {
	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("simple"), nil
		},
	}

	expectedErr := errors.New("simple retriever failure")
	simple := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return nil, expectedErr
		},
	}

	r := retriever.NewAdaptiveRetriever(model, simple, nil)
	_, err := r.Retrieve(context.Background(), "query")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "adaptive simple")
}

func TestAdaptiveRetriever_Retrieve_ComplexRetrieverError(t *testing.T) {
	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("complex"), nil
		},
	}

	simple := &mockRetriever{}

	expectedErr := errors.New("complex retriever failure")
	complex := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return nil, expectedErr
		},
	}

	r := retriever.NewAdaptiveRetriever(model, simple, complex)
	_, err := r.Retrieve(context.Background(), "query")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "adaptive complex")
}

func TestAdaptiveRetriever_Retrieve_WithHooks(t *testing.T) {
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

	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("simple"), nil
		},
	}

	simpleDocs := makeDocs("doc1")
	simple := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return simpleDocs, nil
		},
	}

	r := retriever.NewAdaptiveRetriever(model, simple, nil, retriever.WithAdaptiveHooks(hooks))
	docs, err := r.Retrieve(context.Background(), "test query")

	require.NoError(t, err)
	assert.True(t, beforeCalled, "BeforeRetrieve should be called")
	assert.True(t, afterCalled, "AfterRetrieve should be called")
	assert.Equal(t, "test query", capturedQuery)
	assert.Equal(t, simpleDocs, capturedDocs)
	assert.Equal(t, simpleDocs, docs)
}

func TestAdaptiveRetriever_Retrieve_BeforeHookError(t *testing.T) {
	expectedErr := errors.New("hook error")
	hooks := retriever.Hooks{
		BeforeRetrieve: func(ctx context.Context, query string) error {
			return expectedErr
		},
	}

	model := &mockChatModel{}
	simple := &mockRetriever{}

	r := retriever.NewAdaptiveRetriever(model, simple, nil, retriever.WithAdaptiveHooks(hooks))
	_, err := r.Retrieve(context.Background(), "query")

	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, 0, model.calls, "LLM should not be called if hook fails")
}

func TestAdaptiveRetriever_QueryComplexityConstants(t *testing.T) {
	// Verify the constants are defined correctly.
	assert.Equal(t, retriever.QueryComplexity("no_retrieval"), retriever.NoRetrieval)
	assert.Equal(t, retriever.QueryComplexity("simple"), retriever.SimpleRetrieval)
	assert.Equal(t, retriever.QueryComplexity("complex"), retriever.ComplexRetrieval)
}

func TestAdaptiveRetriever_ClassifyQuery_NoRetrieval_VariantResponse(t *testing.T) {
	// LLM might respond with variations like "No retrieval needed".
	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("I think no_retrieval is best"), nil
		},
	}

	simple := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			t.Error("should not call retriever for no_retrieval")
			return nil, nil
		},
	}

	r := retriever.NewAdaptiveRetriever(model, simple, nil)
	docs, err := r.Retrieve(context.Background(), "query")

	require.NoError(t, err)
	assert.Nil(t, docs, "Should detect no_retrieval in response")
}

func TestAdaptiveRetriever_ClassifyQuery_Complex_VariantResponse(t *testing.T) {
	// LLM might respond with variations like "This is complex".
	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("This is definitely complex"), nil
		},
	}

	simple := &mockRetriever{}

	complexDocs := makeDocs("complex1")
	var complexCalled bool
	complex := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			complexCalled = true
			return complexDocs, nil
		},
	}

	r := retriever.NewAdaptiveRetriever(model, simple, complex)
	docs, err := r.Retrieve(context.Background(), "query")

	require.NoError(t, err)
	assert.True(t, complexCalled, "Should detect complex in response")
	assert.Equal(t, complexDocs, docs)
}

func TestAdaptiveRetriever_ClassifyQuery_Simple_DefaultBehavior(t *testing.T) {
	// If response doesn't contain "complex" or "no_retrieval", defaults to simple.
	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("I don't know what to do"), nil
		},
	}

	simpleDocs := makeDocs("simple1")
	var simpleCalled bool
	simple := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			simpleCalled = true
			return simpleDocs, nil
		},
	}

	r := retriever.NewAdaptiveRetriever(model, simple, nil)
	docs, err := r.Retrieve(context.Background(), "query")

	require.NoError(t, err)
	assert.True(t, simpleCalled, "Should default to simple retrieval")
	assert.Equal(t, simpleDocs, docs)
}

func TestAdaptiveRetriever_OptionsPassedToInner(t *testing.T) {
	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("simple"), nil
		},
	}

	var receivedOpts []retriever.Option
	simple := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			receivedOpts = opts
			return makeDocs("doc1"), nil
		},
	}

	r := retriever.NewAdaptiveRetriever(model, simple, nil)
	_, err := r.Retrieve(context.Background(), "query", retriever.WithTopK(5), retriever.WithThreshold(0.7))

	require.NoError(t, err)
	assert.NotEmpty(t, receivedOpts, "Options should be passed to inner retriever")
}

func TestAdaptiveRetriever_ClassificationPromptFormat(t *testing.T) {
	query := "test query"
	var receivedPrompt string

	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			require.Len(t, msgs, 1)
			receivedPrompt = msgs[0].(*schema.HumanMessage).Text()
			return schema.NewAIMessage("simple"), nil
		},
	}

	simple := &mockRetriever{
		retrieveFn: func(ctx context.Context, q string, opts ...retriever.Option) ([]schema.Document, error) {
			return makeDocs("doc1"), nil
		},
	}

	r := retriever.NewAdaptiveRetriever(model, simple, nil)
	_, err := r.Retrieve(context.Background(), query)

	require.NoError(t, err)
	assert.Contains(t, receivedPrompt, query, "Prompt should contain the query")
	assert.Contains(t, receivedPrompt, "no_retrieval", "Prompt should mention no_retrieval option")
	assert.Contains(t, receivedPrompt, "simple", "Prompt should mention simple option")
	assert.Contains(t, receivedPrompt, "complex", "Prompt should mention complex option")
}
