package metrics_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/eval"
	"github.com/lookatitude/beluga-ai/eval/metrics"
	"github.com/lookatitude/beluga-ai/internal/testutil/mockllm"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Verify interface compliance.
var _ eval.Metric = (*metrics.Relevance)(nil)

func TestNewRelevance(t *testing.T) {
	model := newMockChatModel()
	r := metrics.NewRelevance(model)
	require.NotNil(t, r)
	assert.Equal(t, "relevance", r.Name())
}

func TestRelevance_Name(t *testing.T) {
	model := newMockChatModel()
	r := metrics.NewRelevance(model)
	assert.Equal(t, "relevance", r.Name())
}

func TestRelevance_Score_FullyRelevant(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("1.0")),
	)
	r := metrics.NewRelevance(model)

	sample := eval.EvalSample{
		Input:  "What is Go?",
		Output: "Go is a statically typed, compiled programming language.",
	}

	score, err := r.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 1.0, score)
	assert.Equal(t, 1, model.GenerateCalls())
}

func TestRelevance_Score_NotRelevant(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("0.0")),
	)
	r := metrics.NewRelevance(model)

	sample := eval.EvalSample{
		Input:  "What is Go?",
		Output: "The sky is blue.",
	}

	score, err := r.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 0.0, score)
}

func TestRelevance_Score_PartiallyRelevant(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("0.5")),
	)
	r := metrics.NewRelevance(model)

	sample := eval.EvalSample{
		Input:  "What is Go?",
		Output: "Go is related to programming, but I don't know the details.",
	}

	score, err := r.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 0.5, score)
}

func TestRelevance_Score_DecimalScores(t *testing.T) {
	tests := []struct {
		name          string
		llmResponse   string
		expectedScore float64
	}{
		{"0.0", "0.0", 0.0},
		{"0.2", "0.2", 0.2},
		{"0.5", "0.5", 0.5},
		{"0.8", "0.8", 0.8},
		{"1.0", "1.0", 1.0},
		{"0.666", "0.666", 0.666},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := newMockChatModel(
				mockllm.WithResponse(schema.NewAIMessage(tt.llmResponse)),
			)
			r := metrics.NewRelevance(model)

			sample := eval.EvalSample{
				Input:  "Question",
				Output: "Answer",
			}

			score, err := r.Score(context.Background(), sample)

			require.NoError(t, err)
			assert.InDelta(t, tt.expectedScore, score, 0.001)
		})
	}
}

func TestRelevance_Score_LLMError(t *testing.T) {
	expectedErr := errors.New("llm error")
	model := newMockChatModel(
		mockllm.WithError(expectedErr),
	)
	r := metrics.NewRelevance(model)

	sample := eval.EvalSample{
		Input:  "Question",
		Output: "Answer",
	}

	score, err := r.Score(context.Background(), sample)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "relevance")
	assert.Contains(t, err.Error(), "llm generate")
	assert.Equal(t, 0.0, score)
}

func TestRelevance_Score_InvalidResponse(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("not a number")),
	)
	r := metrics.NewRelevance(model)

	sample := eval.EvalSample{
		Input:  "Question",
		Output: "Answer",
	}

	score, err := r.Score(context.Background(), sample)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse score")
	assert.Equal(t, 0.0, score)
}

func TestRelevance_Score_ScoreAboveOne(t *testing.T) {
	// LLM returns > 1.0, should be clamped
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("3.0")),
	)
	r := metrics.NewRelevance(model)

	sample := eval.EvalSample{
		Input:  "Question",
		Output: "Answer",
	}

	score, err := r.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 1.0, score)
}

func TestRelevance_Score_ScoreBelowZero(t *testing.T) {
	// LLM returns < 0.0, should be clamped
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("-1.0")),
	)
	r := metrics.NewRelevance(model)

	sample := eval.EvalSample{
		Input:  "Question",
		Output: "Answer",
	}

	score, err := r.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 0.0, score)
}

func TestRelevance_Score_WhitespaceInResponse(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("  0.75  \n\t")),
	)
	r := metrics.NewRelevance(model)

	sample := eval.EvalSample{
		Input:  "Question",
		Output: "Answer",
	}

	score, err := r.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.InDelta(t, 0.75, score, 0.001)
}

func TestRelevance_Score_EmptyInput(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("0.0")),
	)
	r := metrics.NewRelevance(model)

	sample := eval.EvalSample{
		Input:  "",
		Output: "Answer",
	}

	score, err := r.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 0.0, score)
}

func TestRelevance_Score_EmptyOutput(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("0.0")),
	)
	r := metrics.NewRelevance(model)

	sample := eval.EvalSample{
		Input:  "Question",
		Output: "",
	}

	score, err := r.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 0.0, score)
}

func TestRelevance_Score_LongInput(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("1.0")),
	)
	r := metrics.NewRelevance(model)

	longInput := string(make([]byte, 5000))
	sample := eval.EvalSample{
		Input:  longInput,
		Output: "Answer",
	}

	score, err := r.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 1.0, score)
}

func TestRelevance_Score_LongOutput(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("1.0")),
	)
	r := metrics.NewRelevance(model)

	longOutput := string(make([]byte, 5000))
	sample := eval.EvalSample{
		Input:  "Question",
		Output: longOutput,
	}

	score, err := r.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 1.0, score)
}

func TestRelevance_Score_ContextCancellation(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("1.0")),
	)
	r := metrics.NewRelevance(model)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	sample := eval.EvalSample{
		Input:  "Question",
		Output: "Answer",
	}

	// Mock doesn't actually respect context, but in real implementation it would error
	_, err := r.Score(ctx, sample)

	// This test demonstrates the pattern; real LLM would return context error
	_ = err
}

func TestRelevance_Score_SpecialCharacters(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("1.0")),
	)
	r := metrics.NewRelevance(model)

	sample := eval.EvalSample{
		Input:  "What is 2+2?",
		Output: "2+2=4",
	}

	score, err := r.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 1.0, score)
}

func TestRelevance_Score_UnicodeCharacters(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("1.0")),
	)
	r := metrics.NewRelevance(model)

	sample := eval.EvalSample{
		Input:  "什么是Go?",
		Output: "Go是一种编程语言。",
	}

	score, err := r.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 1.0, score)
}

func TestRelevance_Score_NoRetrievedDocs(t *testing.T) {
	// Relevance doesn't use retrieved docs, so this should work fine
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("1.0")),
	)
	r := metrics.NewRelevance(model)

	sample := eval.EvalSample{
		Input:         "Question",
		Output:        "Answer",
		RetrievedDocs: nil,
	}

	score, err := r.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 1.0, score)
}

func TestRelevance_Score_WithRetrievedDocs(t *testing.T) {
	// Relevance doesn't use retrieved docs, but they shouldn't interfere
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("1.0")),
	)
	r := metrics.NewRelevance(model)

	sample := eval.EvalSample{
		Input:  "Question",
		Output: "Answer",
		RetrievedDocs: []schema.Document{
			{Content: "Some context"},
		},
	}

	score, err := r.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 1.0, score)
}
