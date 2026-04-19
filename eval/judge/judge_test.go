package judge

import (
	"context"
	"errors"
	"iter"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/eval"
	"github.com/lookatitude/beluga-ai/v2/internal/testutil/mockllm"
	"github.com/lookatitude/beluga-ai/v2/llm"
	"github.com/lookatitude/beluga-ai/v2/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockChatModel adapts mockllm.MockChatModel to llm.ChatModel.
type mockChatModel struct {
	*mockllm.MockChatModel
}

func (m *mockChatModel) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
	return m.MockChatModel.Generate(ctx, msgs)
}

func (m *mockChatModel) Stream(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	return m.MockChatModel.Stream(ctx, msgs)
}

func (m *mockChatModel) BindTools(tools []schema.ToolDefinition) llm.ChatModel {
	return &mockChatModel{m.MockChatModel.BindTools(tools)}
}

func newMock(opts ...mockllm.Option) *mockChatModel {
	return &mockChatModel{mockllm.New(opts...)}
}

func testRubric() *Rubric {
	return &Rubric{
		Name:        "test-rubric",
		Description: "A test rubric",
		Criteria: []Criterion{
			{
				Name:        "accuracy",
				Description: "How accurate is the response",
				Weight:      2.0,
				Levels: []ScoreLevel{
					{Label: "poor", Score: 0.0, Description: "Inaccurate"},
					{Label: "good", Score: 0.5, Description: "Partially accurate"},
					{Label: "excellent", Score: 1.0, Description: "Fully accurate"},
				},
			},
			{
				Name:        "clarity",
				Description: "How clear is the response",
				Weight:      1.0,
				Levels: []ScoreLevel{
					{Label: "poor", Score: 0.0, Description: "Unclear"},
					{Label: "excellent", Score: 1.0, Description: "Clear"},
				},
			},
		},
	}
}

func TestJudgeMetric_Name(t *testing.T) {
	model := newMock(mockllm.WithResponse(schema.NewAIMessage("accuracy: 0.8\nclarity: 0.9")))
	jm, err := NewJudgeMetric(WithModel(model), WithRubric(testRubric()), WithMetricName("my-judge"))
	require.NoError(t, err)
	assert.Equal(t, "my-judge", jm.Name())
}

func TestJudgeMetric_Score(t *testing.T) {
	tests := []struct {
		name      string
		response  string
		wantScore float64
		wantErr   bool
		modelErr  error
		tolerance float64
	}{
		{
			name:      "perfect scores",
			response:  "accuracy: 1.0\nclarity: 1.0",
			wantScore: 1.0,
			tolerance: 0.001,
		},
		{
			name:      "weighted average",
			response:  "accuracy: 0.8\nclarity: 0.5",
			wantScore: (0.8*2.0 + 0.5*1.0) / 3.0, // ~0.7
			tolerance: 0.001,
		},
		{
			name:      "zero scores",
			response:  "accuracy: 0.0\nclarity: 0.0",
			wantScore: 0.0,
			tolerance: 0.001,
		},
		{
			name:     "missing criterion",
			response: "accuracy: 0.8",
			wantErr:  true,
		},
		{
			name:     "llm error",
			modelErr: errors.New("api error"),
			wantErr:  true,
		},
		{
			name:      "clamp scores above 1",
			response:  "accuracy: 1.5\nclarity: 0.5",
			wantScore: (1.0*2.0 + 0.5*1.0) / 3.0,
			tolerance: 0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts []mockllm.Option
			if tt.modelErr != nil {
				opts = append(opts, mockllm.WithError(tt.modelErr))
			} else {
				opts = append(opts, mockllm.WithResponse(schema.NewAIMessage(tt.response)))
			}
			model := newMock(opts...)
			jm, err := NewJudgeMetric(WithModel(model), WithRubric(testRubric()))
			require.NoError(t, err)

			score, err := jm.Score(context.Background(), eval.EvalSample{
				Input:  "What is Go?",
				Output: "Go is a programming language.",
			})
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.InDelta(t, tt.wantScore, score, tt.tolerance)
		})
	}
}

func TestNewJudgeMetric_Validation(t *testing.T) {
	model := newMock()

	t.Run("no model", func(t *testing.T) {
		_, err := NewJudgeMetric(WithRubric(testRubric()))
		require.Error(t, err)
	})

	t.Run("no rubric", func(t *testing.T) {
		_, err := NewJudgeMetric(WithModel(model))
		require.Error(t, err)
	})

	t.Run("invalid rubric", func(t *testing.T) {
		_, err := NewJudgeMetric(WithModel(model), WithRubric(&Rubric{Name: ""}))
		require.Error(t, err)
	})
}

func TestRubric_Validate(t *testing.T) {
	tests := []struct {
		name    string
		rubric  *Rubric
		wantErr bool
	}{
		{
			name:   "valid rubric",
			rubric: testRubric(),
		},
		{
			name:    "empty name",
			rubric:  &Rubric{Name: "", Criteria: []Criterion{{Name: "a", Weight: 1, Levels: []ScoreLevel{{Label: "x", Score: 1}}}}},
			wantErr: true,
		},
		{
			name:    "no criteria",
			rubric:  &Rubric{Name: "test"},
			wantErr: true,
		},
		{
			name: "zero weight",
			rubric: &Rubric{Name: "test", Criteria: []Criterion{
				{Name: "a", Weight: 0, Levels: []ScoreLevel{{Label: "x", Score: 1}}},
			}},
			wantErr: true,
		},
		{
			name: "no levels",
			rubric: &Rubric{Name: "test", Criteria: []Criterion{
				{Name: "a", Weight: 1},
			}},
			wantErr: true,
		},
		{
			name: "duplicate criterion name",
			rubric: &Rubric{Name: "test", Criteria: []Criterion{
				{Name: "a", Weight: 1, Levels: []ScoreLevel{{Label: "x", Score: 0.5}}},
				{Name: "a", Weight: 1, Levels: []ScoreLevel{{Label: "y", Score: 0.5}}},
			}},
			wantErr: true,
		},
		{
			name: "score level above 1",
			rubric: &Rubric{Name: "test", Criteria: []Criterion{
				{Name: "a", Weight: 1, Levels: []ScoreLevel{{Label: "x", Score: 1.5}}},
			}},
			wantErr: true,
		},
		{
			name: "score level below 0",
			rubric: &Rubric{Name: "test", Criteria: []Criterion{
				{Name: "a", Weight: 1, Levels: []ScoreLevel{{Label: "x", Score: -0.1}}},
			}},
			wantErr: true,
		},
		{
			name: "empty criterion name",
			rubric: &Rubric{Name: "test", Criteria: []Criterion{
				{Name: "", Weight: 1, Levels: []ScoreLevel{{Label: "x", Score: 0.5}}},
			}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rubric.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestRubric_ToPrompt(t *testing.T) {
	rubric := testRubric()
	prompt := rubric.ToPrompt()
	assert.Contains(t, prompt, "test-rubric")
	assert.Contains(t, prompt, "accuracy")
	assert.Contains(t, prompt, "clarity")
	assert.Contains(t, prompt, "excellent")
}

func TestBatchJudge_Evaluate(t *testing.T) {
	model := newMock(mockllm.WithResponse(schema.NewAIMessage("accuracy: 0.9\nclarity: 0.8")))
	jm, err := NewJudgeMetric(WithModel(model), WithRubric(testRubric()))
	require.NoError(t, err)

	batch, err := NewBatchJudge(WithJudgeMetric(jm), WithParallel(2))
	require.NoError(t, err)

	samples := []eval.EvalSample{
		{Input: "q1", Output: "a1"},
		{Input: "q2", Output: "a2"},
		{Input: "q3", Output: "a3"},
	}

	result, err := batch.Evaluate(context.Background(), samples)
	require.NoError(t, err)
	assert.Len(t, result.Scores, 3)
	assert.Empty(t, result.Errors)
}

func TestBatchJudge_NoMetric(t *testing.T) {
	_, err := NewBatchJudge()
	require.Error(t, err)
}

func TestBatchJudge_OnResultCallback(t *testing.T) {
	model := newMock(mockllm.WithResponse(schema.NewAIMessage("accuracy: 0.5\nclarity: 0.5")))
	jm, err := NewJudgeMetric(WithModel(model), WithRubric(testRubric()))
	require.NoError(t, err)

	var callbackCount int
	batch, err := NewBatchJudge(
		WithJudgeMetric(jm),
		WithOnResult(func(index int, score float64, err error) {
			callbackCount++
		}),
	)
	require.NoError(t, err)

	_, err = batch.Evaluate(context.Background(), []eval.EvalSample{{Input: "q", Output: "a"}})
	require.NoError(t, err)
	assert.Equal(t, 1, callbackCount)
}

func TestBatchJudge_ContextCancellation(t *testing.T) {
	model := newMock(mockllm.WithResponse(schema.NewAIMessage("accuracy: 0.5\nclarity: 0.5")))
	jm, err := NewJudgeMetric(WithModel(model), WithRubric(testRubric()))
	require.NoError(t, err)

	// Use parallelism 1 so the semaphore is contended, exercising the
	// select-on-ctx.Done() path in the dispatch loop.
	batch, err := NewBatchJudge(WithJudgeMetric(jm), WithParallel(1))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately so dispatch sees it.

	samples := []eval.EvalSample{
		{Input: "q1", Output: "a1"},
		{Input: "q2", Output: "a2"},
	}

	// Should complete without blocking or panicking even when context is cancelled.
	result, err := batch.Evaluate(ctx, samples)
	require.NoError(t, err) // Evaluate always returns nil error.
	// At least no samples should fail due to panics; partial results are fine.
	assert.NotNil(t, result)
}

func TestConsistencyChecker_Check(t *testing.T) {
	model1 := newMock(
		mockllm.WithResponse(schema.NewAIMessage("accuracy: 0.8\nclarity: 0.9")),
		mockllm.WithModelID("model-1"),
	)
	model2 := newMock(
		mockllm.WithResponse(schema.NewAIMessage("accuracy: 0.8\nclarity: 0.9")),
		mockllm.WithModelID("model-2"),
	)

	checker, err := NewConsistencyChecker(
		WithConsistencyRubric(testRubric()),
		WithModels(model1, model2),
		WithRepeats(2),
		WithConsistencyParallel(2),
	)
	require.NoError(t, err)

	result, err := checker.Check(context.Background(), eval.EvalSample{
		Input:  "What is Go?",
		Output: "Go is a programming language.",
	})
	require.NoError(t, err)
	assert.True(t, result.MeanScore > 0)
	assert.True(t, result.Agreement > 0)
	assert.Len(t, result.Scores, 2)
}

func TestConsistencyChecker_Validation(t *testing.T) {
	model := newMock()

	t.Run("no models", func(t *testing.T) {
		_, err := NewConsistencyChecker(WithConsistencyRubric(testRubric()))
		require.Error(t, err)
	})

	t.Run("no rubric", func(t *testing.T) {
		_, err := NewConsistencyChecker(WithModels(model))
		require.Error(t, err)
	})
}

func TestComputeAgreement(t *testing.T) {
	tests := []struct {
		name   string
		scores []float64
		want   float64
	}{
		{"single score", []float64{0.5}, 1.0},
		{"identical", []float64{0.5, 0.5, 0.5}, 1.0},
		{"within threshold", []float64{0.5, 0.55, 0.6}, 1.0},
		{"mixed agreement", []float64{0.0, 0.5, 1.0}, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeAgreement(tt.scores)
			assert.InDelta(t, tt.want, got, 0.01)
		})
	}
}

func TestParseJudgeResponse(t *testing.T) {
	rubric := testRubric()

	tests := []struct {
		name    string
		text    string
		want    map[string]float64
		wantErr bool
	}{
		{
			name: "valid response",
			text: "accuracy: 0.8\nclarity: 0.9",
			want: map[string]float64{"accuracy": 0.8, "clarity": 0.9},
		},
		{
			name: "with extra whitespace",
			text: "  accuracy : 0.8 \n  clarity : 0.9  \n",
			want: map[string]float64{"accuracy": 0.8, "clarity": 0.9},
		},
		{
			name:    "missing criterion",
			text:    "accuracy: 0.8",
			wantErr: true,
		},
		{
			name: "ignores unparseable lines",
			text: "Some preamble\naccuracy: 0.8\nclarity: 0.9\nDone!",
			want: map[string]float64{"accuracy": 0.8, "clarity": 0.9},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseJudgeResponse(tt.text, rubric)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
