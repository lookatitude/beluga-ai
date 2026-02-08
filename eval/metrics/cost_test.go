package metrics_test

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/eval"
	"github.com/lookatitude/beluga-ai/eval/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Verify interface compliance.
var _ eval.Metric = (*metrics.Cost)(nil)

func TestNewCost(t *testing.T) {
	c := metrics.NewCost()
	require.NotNil(t, c)
	assert.Equal(t, "cost", c.Name())
}

func TestNewCost_WithPricing(t *testing.T) {
	pricing := map[string]metrics.ModelPricing{
		"gpt-4": {
			InputTokenPrice:  30.0,
			OutputTokenPrice: 60.0,
		},
	}

	c := metrics.NewCost(metrics.WithPricing(pricing))
	require.NotNil(t, c)
	assert.Equal(t, "cost", c.Name())
}

func TestCost_Name(t *testing.T) {
	c := metrics.NewCost()
	assert.Equal(t, "cost", c.Name())
}

func TestCost_Score_ValidMetadata(t *testing.T) {
	pricing := map[string]metrics.ModelPricing{
		"gpt-4": {
			InputTokenPrice:  30.0,  // $30 per 1M input tokens
			OutputTokenPrice: 60.0,  // $60 per 1M output tokens
		},
	}

	c := metrics.NewCost(metrics.WithPricing(pricing))
	sample := eval.EvalSample{
		Metadata: map[string]any{
			"model":         "gpt-4",
			"input_tokens":  1000,
			"output_tokens": 500,
		},
	}

	score, err := c.Score(context.Background(), sample)

	require.NoError(t, err)
	// Cost = (1000 * 30 / 1,000,000) + (500 * 60 / 1,000,000)
	//      = 0.03 + 0.03 = 0.06
	assert.InDelta(t, 0.06, score, 0.001)
}

func TestCost_Score_DifferentModels(t *testing.T) {
	pricing := map[string]metrics.ModelPricing{
		"gpt-4": {
			InputTokenPrice:  30.0,
			OutputTokenPrice: 60.0,
		},
		"gpt-3.5-turbo": {
			InputTokenPrice:  0.5,
			OutputTokenPrice: 1.5,
		},
	}

	tests := []struct {
		name         string
		model        string
		inputTokens  int
		outputTokens int
		expectedCost float64
	}{
		{
			name:         "gpt-4",
			model:        "gpt-4",
			inputTokens:  1000,
			outputTokens: 1000,
			expectedCost: 0.09, // (1000*30 + 1000*60) / 1M = 0.09
		},
		{
			name:         "gpt-3.5-turbo",
			model:        "gpt-3.5-turbo",
			inputTokens:  1000,
			outputTokens: 1000,
			expectedCost: 0.002, // (1000*0.5 + 1000*1.5) / 1M = 0.002
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := metrics.NewCost(metrics.WithPricing(pricing))
			sample := eval.EvalSample{
				Metadata: map[string]any{
					"model":         tt.model,
					"input_tokens":  tt.inputTokens,
					"output_tokens": tt.outputTokens,
				},
			}

			score, err := c.Score(context.Background(), sample)

			require.NoError(t, err)
			assert.InDelta(t, tt.expectedCost, score, 0.0001)
		})
	}
}

func TestCost_Score_MissingModel(t *testing.T) {
	c := metrics.NewCost()
	sample := eval.EvalSample{
		Metadata: map[string]any{
			"input_tokens":  1000,
			"output_tokens": 500,
		},
	}

	score, err := c.Score(context.Background(), sample)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing metadata key")
	assert.Contains(t, err.Error(), "model")
	assert.Equal(t, 0.0, score)
}

func TestCost_Score_MissingInputTokens(t *testing.T) {
	pricing := map[string]metrics.ModelPricing{
		"gpt-4": {InputTokenPrice: 30.0, OutputTokenPrice: 60.0},
	}

	c := metrics.NewCost(metrics.WithPricing(pricing))
	sample := eval.EvalSample{
		Metadata: map[string]any{
			"model":         "gpt-4",
			"output_tokens": 500,
		},
	}

	score, err := c.Score(context.Background(), sample)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing metadata key")
	assert.Contains(t, err.Error(), "input_tokens")
	assert.Equal(t, 0.0, score)
}

func TestCost_Score_MissingOutputTokens(t *testing.T) {
	pricing := map[string]metrics.ModelPricing{
		"gpt-4": {InputTokenPrice: 30.0, OutputTokenPrice: 60.0},
	}

	c := metrics.NewCost(metrics.WithPricing(pricing))
	sample := eval.EvalSample{
		Metadata: map[string]any{
			"model":        "gpt-4",
			"input_tokens": 1000,
		},
	}

	score, err := c.Score(context.Background(), sample)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing metadata key")
	assert.Contains(t, err.Error(), "output_tokens")
	assert.Equal(t, 0.0, score)
}

func TestCost_Score_UnknownModel(t *testing.T) {
	pricing := map[string]metrics.ModelPricing{
		"gpt-4": {InputTokenPrice: 30.0, OutputTokenPrice: 60.0},
	}

	c := metrics.NewCost(metrics.WithPricing(pricing))
	sample := eval.EvalSample{
		Metadata: map[string]any{
			"model":         "unknown-model",
			"input_tokens":  1000,
			"output_tokens": 500,
		},
	}

	score, err := c.Score(context.Background(), sample)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no pricing for model")
	assert.Contains(t, err.Error(), "unknown-model")
	assert.Equal(t, 0.0, score)
}

func TestCost_Score_InvalidModelType(t *testing.T) {
	pricing := map[string]metrics.ModelPricing{
		"gpt-4": {InputTokenPrice: 30.0, OutputTokenPrice: 60.0},
	}

	c := metrics.NewCost(metrics.WithPricing(pricing))
	sample := eval.EvalSample{
		Metadata: map[string]any{
			"model":         123, // Not a string
			"input_tokens":  1000,
			"output_tokens": 500,
		},
	}

	score, err := c.Score(context.Background(), sample)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be a string")
	assert.Equal(t, 0.0, score)
}

func TestCost_Score_InvalidInputTokensType(t *testing.T) {
	pricing := map[string]metrics.ModelPricing{
		"gpt-4": {InputTokenPrice: 30.0, OutputTokenPrice: 60.0},
	}

	c := metrics.NewCost(metrics.WithPricing(pricing))
	sample := eval.EvalSample{
		Metadata: map[string]any{
			"model":         "gpt-4",
			"input_tokens":  "not a number",
			"output_tokens": 500,
		},
	}

	score, err := c.Score(context.Background(), sample)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "input_tokens")
	assert.Equal(t, 0.0, score)
}

func TestCost_Score_InvalidOutputTokensType(t *testing.T) {
	pricing := map[string]metrics.ModelPricing{
		"gpt-4": {InputTokenPrice: 30.0, OutputTokenPrice: 60.0},
	}

	c := metrics.NewCost(metrics.WithPricing(pricing))
	sample := eval.EvalSample{
		Metadata: map[string]any{
			"model":         "gpt-4",
			"input_tokens":  1000,
			"output_tokens": "not a number",
		},
	}

	score, err := c.Score(context.Background(), sample)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "output_tokens")
	assert.Equal(t, 0.0, score)
}

func TestCost_Score_ZeroTokens(t *testing.T) {
	pricing := map[string]metrics.ModelPricing{
		"gpt-4": {InputTokenPrice: 30.0, OutputTokenPrice: 60.0},
	}

	c := metrics.NewCost(metrics.WithPricing(pricing))
	sample := eval.EvalSample{
		Metadata: map[string]any{
			"model":         "gpt-4",
			"input_tokens":  0,
			"output_tokens": 0,
		},
	}

	score, err := c.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 0.0, score)
}

func TestCost_Score_DifferentNumericTypes(t *testing.T) {
	pricing := map[string]metrics.ModelPricing{
		"gpt-4": {InputTokenPrice: 30.0, OutputTokenPrice: 60.0},
	}

	tests := []struct {
		name         string
		inputTokens  any
		outputTokens any
	}{
		{"int", int(1000), int(500)},
		{"int64", int64(1000), int64(500)},
		{"int32", int32(1000), int32(500)},
		{"float64", float64(1000), float64(500)},
		{"float32", float32(1000), float32(500)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := metrics.NewCost(metrics.WithPricing(pricing))
			sample := eval.EvalSample{
				Metadata: map[string]any{
					"model":         "gpt-4",
					"input_tokens":  tt.inputTokens,
					"output_tokens": tt.outputTokens,
				},
			}

			score, err := c.Score(context.Background(), sample)

			require.NoError(t, err)
			assert.InDelta(t, 0.06, score, 0.001)
		})
	}
}

func TestCost_Score_LargeCost(t *testing.T) {
	pricing := map[string]metrics.ModelPricing{
		"expensive-model": {
			InputTokenPrice:  1000.0,
			OutputTokenPrice: 2000.0,
		},
	}

	c := metrics.NewCost(metrics.WithPricing(pricing))
	sample := eval.EvalSample{
		Metadata: map[string]any{
			"model":         "expensive-model",
			"input_tokens":  100000,
			"output_tokens": 50000,
		},
	}

	score, err := c.Score(context.Background(), sample)

	require.NoError(t, err)
	// (100000 * 1000 + 50000 * 2000) / 1M = 200
	assert.InDelta(t, 200.0, score, 0.1)
}

func TestCost_Score_EmptyPricing(t *testing.T) {
	c := metrics.NewCost()
	sample := eval.EvalSample{
		Metadata: map[string]any{
			"model":         "gpt-4",
			"input_tokens":  1000,
			"output_tokens": 500,
		},
	}

	score, err := c.Score(context.Background(), sample)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no pricing for model")
	assert.Equal(t, 0.0, score)
}
