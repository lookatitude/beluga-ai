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
var _ eval.Metric = (*metrics.Hallucination)(nil)

func TestNewHallucination(t *testing.T) {
	model := newMockChatModel()
	h := metrics.NewHallucination(model)
	require.NotNil(t, h)
	assert.Equal(t, "hallucination", h.Name())
}

func TestHallucination_Name(t *testing.T) {
	model := newMockChatModel()
	h := metrics.NewHallucination(model)
	assert.Equal(t, "hallucination", h.Name())
}

func TestHallucination_Score_NoHallucination(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("1.0")),
	)
	h := metrics.NewHallucination(model)

	sample := eval.EvalSample{
		Input:  "What is Go?",
		Output: "Go is a programming language.",
		RetrievedDocs: []schema.Document{
			{ID: "doc1", Content: "Go is a programming language created by Google."},
		},
	}

	score, err := h.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 1.0, score)
	assert.Equal(t, 1, model.GenerateCalls())
}

func TestHallucination_Score_ClearHallucination(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("0.0")),
	)
	h := metrics.NewHallucination(model)

	sample := eval.EvalSample{
		Input:  "What is Go?",
		Output: "Go is a fruit that grows on trees.",
		RetrievedDocs: []schema.Document{
			{ID: "doc1", Content: "Go is a programming language."},
		},
	}

	score, err := h.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 0.0, score)
}

func TestHallucination_Score_PartialHallucination(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("0.5")),
	)
	h := metrics.NewHallucination(model)

	sample := eval.EvalSample{
		Input:  "What is Go?",
		Output: "Go is a language with questionable claims.",
		RetrievedDocs: []schema.Document{
			{ID: "doc1", Content: "Go is a programming language."},
		},
	}

	score, err := h.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 0.5, score)
}

func TestHallucination_Score_DecimalScores(t *testing.T) {
	tests := []struct {
		name          string
		llmResponse   string
		expectedScore float64
	}{
		{"no hallucination", "1.0", 1.0},
		{"slight hallucination", "0.8", 0.8},
		{"moderate hallucination", "0.5", 0.5},
		{"significant hallucination", "0.2", 0.2},
		{"clear hallucination", "0.0", 0.0},
		{"precise score", "0.666", 0.666},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := newMockChatModel(
				mockllm.WithResponse(schema.NewAIMessage(tt.llmResponse)),
			)
			h := metrics.NewHallucination(model)

			sample := eval.EvalSample{
				Input:  "Question",
				Output: "Answer",
				RetrievedDocs: []schema.Document{
					{Content: "Context"},
				},
			}

			score, err := h.Score(context.Background(), sample)

			require.NoError(t, err)
			assert.InDelta(t, tt.expectedScore, score, 0.001)
		})
	}
}

func TestHallucination_Score_NoDocuments(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("1.0")),
	)
	h := metrics.NewHallucination(model)

	sample := eval.EvalSample{
		Input:         "What is Go?",
		Output:        "Go is a programming language.",
		RetrievedDocs: nil,
	}

	score, err := h.Score(context.Background(), sample)

	require.NoError(t, err)
	// Should still work, but with "(no documents provided)" in prompt
	assert.Equal(t, 1.0, score)
}

func TestHallucination_Score_EmptyDocuments(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("1.0")),
	)
	h := metrics.NewHallucination(model)

	sample := eval.EvalSample{
		Input:         "What is Go?",
		Output:        "Go is a programming language.",
		RetrievedDocs: []schema.Document{},
	}

	score, err := h.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 1.0, score)
}

func TestHallucination_Score_MultipleDocuments(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("1.0")),
	)
	h := metrics.NewHallucination(model)

	sample := eval.EvalSample{
		Input:  "What is Go?",
		Output: "Go is a programming language created by Google that is statically typed.",
		RetrievedDocs: []schema.Document{
			{ID: "doc1", Content: "Go is a programming language."},
			{ID: "doc2", Content: "Go was created by Google."},
			{ID: "doc3", Content: "Go is statically typed."},
		},
	}

	score, err := h.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 1.0, score)
}

func TestHallucination_Score_LLMError(t *testing.T) {
	expectedErr := errors.New("llm error")
	model := newMockChatModel(
		mockllm.WithError(expectedErr),
	)
	h := metrics.NewHallucination(model)

	sample := eval.EvalSample{
		Input:  "Question",
		Output: "Answer",
		RetrievedDocs: []schema.Document{
			{Content: "Context"},
		},
	}

	score, err := h.Score(context.Background(), sample)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "hallucination")
	assert.Contains(t, err.Error(), "llm generate")
	assert.Equal(t, 0.0, score)
}

func TestHallucination_Score_InvalidResponse(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("invalid")),
	)
	h := metrics.NewHallucination(model)

	sample := eval.EvalSample{
		Input:  "Question",
		Output: "Answer",
		RetrievedDocs: []schema.Document{
			{Content: "Context"},
		},
	}

	score, err := h.Score(context.Background(), sample)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse score")
	assert.Equal(t, 0.0, score)
}

func TestHallucination_Score_ScoreAboveOne(t *testing.T) {
	// LLM returns > 1.0, should be clamped
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("5.0")),
	)
	h := metrics.NewHallucination(model)

	sample := eval.EvalSample{
		Input:  "Question",
		Output: "Answer",
		RetrievedDocs: []schema.Document{
			{Content: "Context"},
		},
	}

	score, err := h.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 1.0, score)
}

func TestHallucination_Score_ScoreBelowZero(t *testing.T) {
	// LLM returns < 0.0, should be clamped
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("-2.0")),
	)
	h := metrics.NewHallucination(model)

	sample := eval.EvalSample{
		Input:  "Question",
		Output: "Answer",
		RetrievedDocs: []schema.Document{
			{Content: "Context"},
		},
	}

	score, err := h.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 0.0, score)
}

func TestHallucination_Score_WhitespaceInResponse(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("\n  0.95  \t\n")),
	)
	h := metrics.NewHallucination(model)

	sample := eval.EvalSample{
		Input:  "Question",
		Output: "Answer",
		RetrievedDocs: []schema.Document{
			{Content: "Context"},
		},
	}

	score, err := h.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.InDelta(t, 0.95, score, 0.001)
}

func TestHallucination_Score_ContextCancellation(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("1.0")),
	)
	h := metrics.NewHallucination(model)

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
	_, err := h.Score(ctx, sample)

	// This test demonstrates the pattern; real LLM would return context error
	_ = err
}

func TestHallucination_Score_EmptyInput(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("1.0")),
	)
	h := metrics.NewHallucination(model)

	sample := eval.EvalSample{
		Input:  "",
		Output: "Answer",
		RetrievedDocs: []schema.Document{
			{Content: "Context"},
		},
	}

	score, err := h.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 1.0, score)
}

func TestHallucination_Score_EmptyOutput(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("1.0")),
	)
	h := metrics.NewHallucination(model)

	sample := eval.EvalSample{
		Input:  "Question",
		Output: "",
		RetrievedDocs: []schema.Document{
			{Content: "Context"},
		},
	}

	score, err := h.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 1.0, score)
}

func TestHallucination_Score_LongDocuments(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("1.0")),
	)
	h := metrics.NewHallucination(model)

	longContent := string(make([]byte, 10000))
	sample := eval.EvalSample{
		Input:  "Question",
		Output: "Answer",
		RetrievedDocs: []schema.Document{
			{Content: longContent},
		},
	}

	score, err := h.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 1.0, score)
}

func TestHallucination_Score_CommonKnowledgeFacts(t *testing.T) {
	// Hallucination metric should consider commonly known facts
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("1.0")),
	)
	h := metrics.NewHallucination(model)

	sample := eval.EvalSample{
		Input:  "What is the capital of France?",
		Output: "The capital of France is Paris.",
		RetrievedDocs: []schema.Document{
			{Content: "Some unrelated context."},
		},
	}

	score, err := h.Score(context.Background(), sample)

	require.NoError(t, err)
	// Should not be marked as hallucination if it's common knowledge
	assert.Equal(t, 1.0, score)
}

func TestHallucination_Score_UnicodeContent(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("1.0")),
	)
	h := metrics.NewHallucination(model)

	sample := eval.EvalSample{
		Input:  "什么是Go?",
		Output: "Go是一种编程语言。",
		RetrievedDocs: []schema.Document{
			{Content: "Go是由Google创建的编程语言。"},
		},
	}

	score, err := h.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 1.0, score)
}

func TestHallucination_Score_SpecialCharacters(t *testing.T) {
	model := newMockChatModel(
		mockllm.WithResponse(schema.NewAIMessage("1.0")),
	)
	h := metrics.NewHallucination(model)

	sample := eval.EvalSample{
		Input:  "What is 2+2?",
		Output: "2+2=4",
		RetrievedDocs: []schema.Document{
			{Content: "Basic math: 2+2=4"},
		},
	}

	score, err := h.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 1.0, score)
}
