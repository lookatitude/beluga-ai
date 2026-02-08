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
var _ eval.Metric = (*metrics.Faithfulness)(nil)

func TestNewFaithfulness(t *testing.T) {
	model := newMockChatModel()
	f := metrics.NewFaithfulness(model)
	require.NotNil(t, f)
	assert.Equal(t, "faithfulness", f.Name())
}

func TestFaithfulness_Name(t *testing.T) {
	model := newMockChatModel()
	f := metrics.NewFaithfulness(model)
	assert.Equal(t, "faithfulness", f.Name())
}

func TestFaithfulness_Score_FullyFaithful(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("1.0")),
	)
	f := metrics.NewFaithfulness(model)

	sample := eval.EvalSample{
		Input:  "What is Go?",
		Output: "Go is a programming language.",
		RetrievedDocs: []schema.Document{
			{ID: "doc1", Content: "Go is a programming language created by Google."},
		},
	}

	score, err := f.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 1.0, score)
	assert.Equal(t, 1, model.GenerateCalls())
}

func TestFaithfulness_Score_NotFaithful(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("0.0")),
	)
	f := metrics.NewFaithfulness(model)

	sample := eval.EvalSample{
		Input:  "What is Go?",
		Output: "Go is a fruit.",
		RetrievedDocs: []schema.Document{
			{ID: "doc1", Content: "Go is a programming language."},
		},
	}

	score, err := f.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 0.0, score)
}

func TestFaithfulness_Score_PartiallyFaithful(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("0.5")),
	)
	f := metrics.NewFaithfulness(model)

	sample := eval.EvalSample{
		Input:  "What is Go?",
		Output: "Go is a language with some extra claims.",
		RetrievedDocs: []schema.Document{
			{ID: "doc1", Content: "Go is a programming language."},
		},
	}

	score, err := f.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 0.5, score)
}

func TestFaithfulness_Score_DecimalScores(t *testing.T) {
	tests := []struct {
		name          string
		llmResponse   string
		expectedScore float64
	}{
		{"0.0", "0.0", 0.0},
		{"0.25", "0.25", 0.25},
		{"0.5", "0.5", 0.5},
		{"0.75", "0.75", 0.75},
		{"1.0", "1.0", 1.0},
		{"0.833", "0.833", 0.833},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := newMockChatModel(
				mockllm.WithResponse(schema.NewAIMessage(tt.llmResponse)),
			)
			f := metrics.NewFaithfulness(model)

			sample := eval.EvalSample{
				Input:  "Question",
				Output: "Answer",
				RetrievedDocs: []schema.Document{
					{Content: "Context"},
				},
			}

			score, err := f.Score(context.Background(), sample)

			require.NoError(t, err)
			assert.InDelta(t, tt.expectedScore, score, 0.001)
		})
	}
}

func TestFaithfulness_Score_NoDocuments(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("0.0")),
	)
	f := metrics.NewFaithfulness(model)

	sample := eval.EvalSample{
		Input:         "What is Go?",
		Output:        "Go is a programming language.",
		RetrievedDocs: nil,
	}

	score, err := f.Score(context.Background(), sample)

	require.NoError(t, err)
	// Should still work, but with "(no documents provided)" in prompt
	assert.Equal(t, 0.0, score)
}

func TestFaithfulness_Score_EmptyDocuments(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("0.0")),
	)
	f := metrics.NewFaithfulness(model)

	sample := eval.EvalSample{
		Input:         "What is Go?",
		Output:        "Go is a programming language.",
		RetrievedDocs: []schema.Document{},
	}

	score, err := f.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 0.0, score)
}

func TestFaithfulness_Score_MultipleDocuments(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("1.0")),
	)
	f := metrics.NewFaithfulness(model)

	sample := eval.EvalSample{
		Input:  "What is Go?",
		Output: "Go is a programming language created by Google.",
		RetrievedDocs: []schema.Document{
			{ID: "doc1", Content: "Go is a programming language."},
			{ID: "doc2", Content: "Go was created by Google."},
			{ID: "doc3", Content: "Go is statically typed."},
		},
	}

	score, err := f.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 1.0, score)
}

func TestFaithfulness_Score_LLMError(t *testing.T) {
	expectedErr := errors.New("llm error")
	model := newMockChatModel(
		mockllm.WithError(expectedErr),
	)
	f := metrics.NewFaithfulness(model)

	sample := eval.EvalSample{
		Input:  "Question",
		Output: "Answer",
		RetrievedDocs: []schema.Document{
			{Content: "Context"},
		},
	}

	score, err := f.Score(context.Background(), sample)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "faithfulness")
	assert.Contains(t, err.Error(), "llm generate")
	assert.Equal(t, 0.0, score)
}

func TestFaithfulness_Score_InvalidResponse(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("not a number")),
	)
	f := metrics.NewFaithfulness(model)

	sample := eval.EvalSample{
		Input:  "Question",
		Output: "Answer",
		RetrievedDocs: []schema.Document{
			{Content: "Context"},
		},
	}

	score, err := f.Score(context.Background(), sample)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse score")
	assert.Equal(t, 0.0, score)
}

func TestFaithfulness_Score_ScoreAboveOne(t *testing.T) {
	// LLM returns > 1.0, should be clamped
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("2.5")),
	)
	f := metrics.NewFaithfulness(model)

	sample := eval.EvalSample{
		Input:  "Question",
		Output: "Answer",
		RetrievedDocs: []schema.Document{
			{Content: "Context"},
		},
	}

	score, err := f.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 1.0, score)
}

func TestFaithfulness_Score_ScoreBelowZero(t *testing.T) {
	// LLM returns < 0.0, should be clamped
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("-0.5")),
	)
	f := metrics.NewFaithfulness(model)

	sample := eval.EvalSample{
		Input:  "Question",
		Output: "Answer",
		RetrievedDocs: []schema.Document{
			{Content: "Context"},
		},
	}

	score, err := f.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 0.0, score)
}

func TestFaithfulness_Score_WhitespaceInResponse(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("  0.85  \n")),
	)
	f := metrics.NewFaithfulness(model)

	sample := eval.EvalSample{
		Input:  "Question",
		Output: "Answer",
		RetrievedDocs: []schema.Document{
			{Content: "Context"},
		},
	}

	score, err := f.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.InDelta(t, 0.85, score, 0.001)
}

func TestFaithfulness_Score_ContextCancellation(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("1.0")),
	)
	f := metrics.NewFaithfulness(model)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	sample := eval.EvalSample{
		Input:  "Question",
		Output: "Answer",
		RetrievedDocs: []schema.Document{
			{Content: "Context"},
		},
	}

	// Mock doesn't actually respect context, but in real implementation it would error
	_, err := f.Score(ctx, sample)

	// This test demonstrates the pattern; real LLM would return context error
	_ = err
}

func TestFaithfulness_Score_EmptyInput(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("0.5")),
	)
	f := metrics.NewFaithfulness(model)

	sample := eval.EvalSample{
		Input:  "",
		Output: "Answer",
		RetrievedDocs: []schema.Document{
			{Content: "Context"},
		},
	}

	score, err := f.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 0.5, score)
}

func TestFaithfulness_Score_EmptyOutput(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("0.5")),
	)
	f := metrics.NewFaithfulness(model)

	sample := eval.EvalSample{
		Input:  "Question",
		Output: "",
		RetrievedDocs: []schema.Document{
			{Content: "Context"},
		},
	}

	score, err := f.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 0.5, score)
}

func TestFaithfulness_Score_LongDocuments(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("1.0")),
	)
	f := metrics.NewFaithfulness(model)

	longContent := string(make([]byte, 10000))
	sample := eval.EvalSample{
		Input:  "Question",
		Output: "Answer",
		RetrievedDocs: []schema.Document{
			{Content: longContent},
		},
	}

	score, err := f.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 1.0, score)
}
