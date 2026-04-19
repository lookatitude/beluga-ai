package retriever_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/llm"
	"github.com/lookatitude/beluga-ai/v2/rag/retriever"
	"github.com/lookatitude/beluga-ai/v2/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSubQuestionRetriever_Defaults(t *testing.T) {
	r := retriever.NewSubQuestionRetriever()
	require.NotNil(t, r)
}

func TestSubQuestionRetriever_Retrieve_DecomposesAndRoutes(t *testing.T) {
	docsA := []schema.Document{{ID: "a1", Content: "doc A"}}
	docsB := []schema.Document{{ID: "b1", Content: "doc B"}}

	retA := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return docsA, nil
		},
	}
	retB := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return docsB, nil
		},
	}

	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("alpha|What is A?\nbeta|What is B?"), nil
		},
	}

	decomposer := retriever.NewLLMDecomposer(model)

	r := retriever.NewSubQuestionRetriever(
		retriever.WithDecomposer(decomposer),
		retriever.WithRetrievers(map[string]retriever.Retriever{
			"alpha": retA,
			"beta":  retB,
		}),
	)

	docs, err := r.Retrieve(context.Background(), "complex query")

	require.NoError(t, err)
	assert.Len(t, docs, 2)

	ids := make(map[string]bool)
	for _, d := range docs {
		ids[d.ID] = true
	}
	assert.True(t, ids["a1"])
	assert.True(t, ids["b1"])
}

func TestSubQuestionRetriever_Retrieve_Deduplicates(t *testing.T) {
	sharedDoc := schema.Document{ID: "shared", Content: "shared doc"}

	ret := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return []schema.Document{sharedDoc}, nil
		},
	}

	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("alpha|Q1\nalpha|Q2"), nil
		},
	}

	r := retriever.NewSubQuestionRetriever(
		retriever.WithDecomposer(retriever.NewLLMDecomposer(model)),
		retriever.WithRetrievers(map[string]retriever.Retriever{"alpha": ret}),
	)

	docs, err := r.Retrieve(context.Background(), "query")

	require.NoError(t, err)
	assert.Len(t, docs, 1, "Duplicate docs should be deduplicated")
}

func TestSubQuestionRetriever_Retrieve_MaxSubQuestions(t *testing.T) {
	var queriesSeen []string

	ret := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			queriesSeen = append(queriesSeen, query)
			return []schema.Document{{ID: query, Content: query}}, nil
		},
	}

	// Generate many sub-questions.
	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			lines := make([]string, 10)
			for i := range lines {
				lines[i] = fmt.Sprintf("alpha|Question %d", i+1)
			}
			return schema.NewAIMessage(strings.Join(lines, "\n")), nil
		},
	}

	r := retriever.NewSubQuestionRetriever(
		retriever.WithDecomposer(retriever.NewLLMDecomposer(model)),
		retriever.WithRetrievers(map[string]retriever.Retriever{"alpha": ret}),
		retriever.WithMaxSubQuestions(3),
	)

	docs, err := r.Retrieve(context.Background(), "query")

	require.NoError(t, err)
	assert.Len(t, queriesSeen, 3, "Should limit to 3 sub-questions")
	assert.Len(t, docs, 3)
}

func TestSubQuestionRetriever_Retrieve_NoDecomposer(t *testing.T) {
	r := retriever.NewSubQuestionRetriever(
		retriever.WithRetrievers(map[string]retriever.Retriever{
			"alpha": &mockRetriever{},
		}),
	)

	_, err := r.Retrieve(context.Background(), "query")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "decomposer not configured")
}

func TestSubQuestionRetriever_Retrieve_NoRetrievers(t *testing.T) {
	model := &mockChatModel{}

	r := retriever.NewSubQuestionRetriever(
		retriever.WithDecomposer(retriever.NewLLMDecomposer(model)),
	)

	_, err := r.Retrieve(context.Background(), "query")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no retrievers configured")
}

func TestSubQuestionRetriever_Retrieve_DecomposeError(t *testing.T) {
	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return nil, errors.New("llm failure")
		},
	}

	r := retriever.NewSubQuestionRetriever(
		retriever.WithDecomposer(retriever.NewLLMDecomposer(model)),
		retriever.WithRetrievers(map[string]retriever.Retriever{
			"alpha": &mockRetriever{},
		}),
	)

	_, err := r.Retrieve(context.Background(), "query")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "subquestion decompose")
}

func TestSubQuestionRetriever_Retrieve_InnerRetrieverError(t *testing.T) {
	expectedErr := errors.New("retriever failure")
	ret := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return nil, expectedErr
		},
	}

	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("alpha|What is it?"), nil
		},
	}

	r := retriever.NewSubQuestionRetriever(
		retriever.WithDecomposer(retriever.NewLLMDecomposer(model)),
		retriever.WithRetrievers(map[string]retriever.Retriever{"alpha": ret}),
	)

	_, err := r.Retrieve(context.Background(), "query")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "subquestion retrieve")
}

func TestSubQuestionRetriever_Retrieve_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	ret := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			callCount++
			if callCount == 1 {
				cancel() // Cancel after first sub-question.
			}
			return []schema.Document{{ID: "d1"}}, nil
		},
	}

	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("alpha|Q1\nalpha|Q2\nalpha|Q3"), nil
		},
	}

	r := retriever.NewSubQuestionRetriever(
		retriever.WithDecomposer(retriever.NewLLMDecomposer(model)),
		retriever.WithRetrievers(map[string]retriever.Retriever{"alpha": ret}),
	)

	_, err := r.Retrieve(ctx, "query")

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestSubQuestionRetriever_Retrieve_UnknownRetrieverSkipped(t *testing.T) {
	ret := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return []schema.Document{{ID: "d1", Content: "found"}}, nil
		},
	}

	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("unknown_retriever|Q1\nalpha|Q2"), nil
		},
	}

	r := retriever.NewSubQuestionRetriever(
		retriever.WithDecomposer(retriever.NewLLMDecomposer(model)),
		retriever.WithRetrievers(map[string]retriever.Retriever{"alpha": ret}),
	)

	docs, err := r.Retrieve(context.Background(), "query")

	require.NoError(t, err)
	// The "unknown_retriever" sub-question gets routed to fallback "alpha" by the decomposer,
	// so both sub-questions hit "alpha". Dedup means we get 1 doc.
	assert.Len(t, docs, 1)
}

func TestSubQuestionRetriever_Retrieve_WithHooks(t *testing.T) {
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

	ret := &mockRetriever{
		retrieveFn: func(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
			return []schema.Document{{ID: "d1"}}, nil
		},
	}

	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("alpha|Q1"), nil
		},
	}

	r := retriever.NewSubQuestionRetriever(
		retriever.WithDecomposer(retriever.NewLLMDecomposer(model)),
		retriever.WithRetrievers(map[string]retriever.Retriever{"alpha": ret}),
		retriever.WithSubQuestionHooks(hooks),
	)

	_, err := r.Retrieve(context.Background(), "query")

	require.NoError(t, err)
	assert.True(t, beforeCalled)
	assert.True(t, afterCalled)
}

func TestSubQuestionRetriever_Retrieve_BeforeHookError(t *testing.T) {
	expectedErr := errors.New("hook error")
	hooks := retriever.Hooks{
		BeforeRetrieve: func(ctx context.Context, query string) error {
			return expectedErr
		},
	}

	r := retriever.NewSubQuestionRetriever(
		retriever.WithDecomposer(retriever.NewLLMDecomposer(&mockChatModel{})),
		retriever.WithRetrievers(map[string]retriever.Retriever{"alpha": &mockRetriever{}}),
		retriever.WithSubQuestionHooks(hooks),
	)

	_, err := r.Retrieve(context.Background(), "query")

	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestLLMDecomposer_FallbackRetriever(t *testing.T) {
	// Lines without pipes should use the first available retriever.
	model := &mockChatModel{
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("Just a plain question"), nil
		},
	}

	decomposer := retriever.NewLLMDecomposer(model)
	subs, err := decomposer.Decompose(context.Background(), "query", []string{"default", "other"})

	require.NoError(t, err)
	require.Len(t, subs, 1)
	assert.Equal(t, "default", subs[0].Retriever)
	assert.Equal(t, "Just a plain question", subs[0].Question)
}

// Compile-time checks.
var (
	_ retriever.Retriever  = (*retriever.SubQuestionRetriever)(nil)
	_ retriever.Decomposer = (*retriever.LLMDecomposer)(nil)
)
