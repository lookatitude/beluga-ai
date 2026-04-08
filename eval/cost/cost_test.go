package cost

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/eval"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// staticMetric is a test metric that always returns a fixed score.
type staticMetric struct {
	name  string
	score float64
	err   error
}

func (m *staticMetric) Name() string { return m.name }
func (m *staticMetric) Score(_ context.Context, _ eval.EvalSample) (float64, error) {
	return m.score, m.err
}

func sampleWithMeta(model string, inputTokens, outputTokens int) eval.EvalSample {
	return eval.EvalSample{
		Input:  "test",
		Output: "test output",
		Metadata: map[string]any{
			"model":         model,
			"input_tokens":  inputTokens,
			"output_tokens": outputTokens,
		},
	}
}

func TestCostMetric_Name(t *testing.T) {
	cm := NewCostMetric(WithMetricName("my-cost"))
	assert.Equal(t, "my-cost", cm.Name())
}

func TestCostMetric_Score(t *testing.T) {
	pricing := map[string]ModelPricing{
		"gpt-4o": {InputTokenPrice: 5.0, OutputTokenPrice: 15.0},
	}

	tests := []struct {
		name      string
		sample    eval.EvalSample
		quality   eval.Metric
		wantScore float64
		wantErr   bool
		tolerance float64
	}{
		{
			name:      "raw cost no quality metric",
			sample:    sampleWithMeta("gpt-4o", 1000, 500),
			wantScore: (1000*5.0 + 500*15.0) / 1_000_000, // 0.0125
			tolerance: 0.0001,
		},
		{
			name:      "quality per dollar",
			sample:    sampleWithMeta("gpt-4o", 1000, 500),
			quality:   &staticMetric{name: "q", score: 0.8},
			wantScore: (0.8 / 0.0125) / 1000.0, // 0.064
			tolerance: 0.001,
		},
		{
			name:    "missing model",
			sample:  eval.EvalSample{Metadata: map[string]any{"input_tokens": 100, "output_tokens": 50}},
			wantErr: true,
		},
		{
			name:    "unknown model",
			sample:  sampleWithMeta("unknown", 100, 50),
			wantErr: true,
		},
		{
			name:    "quality metric error",
			sample:  sampleWithMeta("gpt-4o", 100, 50),
			quality: &staticMetric{name: "q", err: errors.New("fail")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := []CostOption{WithPricing(pricing)}
			if tt.quality != nil {
				opts = append(opts, WithQualityMetric(tt.quality))
			}
			cm := NewCostMetric(opts...)

			score, err := cm.Score(context.Background(), tt.sample)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.InDelta(t, tt.wantScore, score, tt.tolerance)
		})
	}
}

func TestCostMetric_ComputeRawCost(t *testing.T) {
	pricing := map[string]ModelPricing{
		"gpt-4o": {InputTokenPrice: 5.0, OutputTokenPrice: 15.0},
	}
	cm := NewCostMetric(WithPricing(pricing))

	cost, err := cm.ComputeRawCost(sampleWithMeta("gpt-4o", 1_000_000, 500_000))
	require.NoError(t, err)
	assert.InDelta(t, 12.5, cost, 0.001) // 5.0 + 7.5
}

func TestParetoAnalyzer_Analyze(t *testing.T) {
	tests := []struct {
		name        string
		configs     []ConfigResult
		wantOptimal int
		wantDom     int
	}{
		{
			name:        "empty",
			configs:     nil,
			wantOptimal: 0,
			wantDom:     0,
		},
		{
			name: "single config",
			configs: []ConfigResult{
				{Name: "a", Quality: 0.8, Cost: 1.0},
			},
			wantOptimal: 1,
			wantDom:     0,
		},
		{
			name: "clear domination",
			configs: []ConfigResult{
				{Name: "a", Quality: 0.9, Cost: 1.0}, // dominates b
				{Name: "b", Quality: 0.5, Cost: 2.0}, // dominated
			},
			wantOptimal: 1,
			wantDom:     1,
		},
		{
			name: "pareto front",
			configs: []ConfigResult{
				{Name: "cheap", Quality: 0.6, Cost: 0.5},
				{Name: "balanced", Quality: 0.8, Cost: 1.0},
				{Name: "premium", Quality: 0.95, Cost: 3.0},
				{Name: "waste", Quality: 0.5, Cost: 2.0}, // dominated by balanced
			},
			wantOptimal: 3,
			wantDom:     1,
		},
		{
			name: "identical configs",
			configs: []ConfigResult{
				{Name: "a", Quality: 0.8, Cost: 1.0},
				{Name: "b", Quality: 0.8, Cost: 1.0},
			},
			wantOptimal: 2,
			wantDom:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewParetoAnalyzer()
			result := analyzer.Analyze(tt.configs)
			assert.Len(t, result.Optimal, tt.wantOptimal)
			assert.Len(t, result.Dominated, tt.wantDom)
		})
	}
}

func TestBudgetAlert(t *testing.T) {
	t.Run("fires on threshold", func(t *testing.T) {
		var alertTotal, alertThreshold float64
		alert, err := NewBudgetAlert(
			WithThreshold(1.0),
			WithOnAlert(func(total, threshold float64) {
				alertTotal = total
				alertThreshold = threshold
			}),
		)
		require.NoError(t, err)

		assert.False(t, alert.Add(0.5))
		assert.False(t, alert.Exceeded())

		assert.True(t, alert.Add(0.6))
		assert.True(t, alert.Exceeded())
		assert.InDelta(t, 1.1, alertTotal, 0.001)
		assert.InDelta(t, 1.0, alertThreshold, 0.001)
	})

	t.Run("fires only once", func(t *testing.T) {
		fireCount := 0
		alert, err := NewBudgetAlert(
			WithThreshold(1.0),
			WithOnAlert(func(_, _ float64) { fireCount++ }),
		)
		require.NoError(t, err)

		alert.Add(2.0)
		alert.Add(3.0)
		assert.Equal(t, 1, fireCount)
	})

	t.Run("reset", func(t *testing.T) {
		alert, err := NewBudgetAlert(WithThreshold(1.0))
		require.NoError(t, err)

		alert.Add(2.0)
		assert.True(t, alert.Exceeded())

		alert.Reset()
		assert.False(t, alert.Exceeded())
		assert.InDelta(t, 0.0, alert.Total(), 0.001)
	})

	t.Run("invalid threshold", func(t *testing.T) {
		_, err := NewBudgetAlert(WithThreshold(0))
		require.Error(t, err)
	})
}

func TestGenerateReport(t *testing.T) {
	configs := []ConfigResult{
		{Name: "cheap", Quality: 0.75, Cost: 0.01},
		{Name: "premium", Quality: 0.95, Cost: 0.10},
		{Name: "waste", Quality: 0.5, Cost: 0.20},
	}

	report := GenerateReport(configs)
	assert.InDelta(t, 0.31, report.TotalCost, 0.001)
	assert.True(t, report.AverageQuality > 0)
	assert.True(t, report.QualityPerDollar > 0)
	assert.True(t, len(report.ParetoOptimal) >= 1)
	assert.True(t, len(report.Recommendations) >= 1)

	str := report.String()
	assert.Contains(t, str, "Cost Report")
	assert.Contains(t, str, "Pareto-Optimal")
}

func TestGenerateReport_Empty(t *testing.T) {
	report := GenerateReport(nil)
	assert.Equal(t, 0.0, report.TotalCost)
	assert.Empty(t, report.ParetoOptimal)
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    float64
		wantErr bool
	}{
		{"float64", float64(1.5), 1.5, false},
		{"float32", float32(1.5), 1.5, false},
		{"int", 42, 42.0, false},
		{"int64", int64(42), 42.0, false},
		{"int32", int32(42), 42.0, false},
		{"string", "42", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toFloat64(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.InDelta(t, tt.want, got, 0.01)
		})
	}
}
