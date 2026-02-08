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
var _ eval.Metric = (*metrics.Latency)(nil)

func TestNewLatency(t *testing.T) {
	l := metrics.NewLatency()
	require.NotNil(t, l)
	assert.Equal(t, "latency", l.Name())
}

func TestNewLatency_WithCustomThreshold(t *testing.T) {
	l := metrics.NewLatency(metrics.WithMaxLatencyMs(5000.0))
	require.NotNil(t, l)
	assert.Equal(t, "latency", l.Name())
}

func TestLatency_Name(t *testing.T) {
	l := metrics.NewLatency()
	assert.Equal(t, "latency", l.Name())
}

func TestLatency_Score_ValidMetadata(t *testing.T) {
	tests := []struct {
		name          string
		latencyMs     any
		maxLatencyMs  float64
		expectedScore float64
	}{
		{
			name:          "zero latency",
			latencyMs:     0.0,
			maxLatencyMs:  10000.0,
			expectedScore: 1.0,
		},
		{
			name:          "low latency",
			latencyMs:     1000.0,
			maxLatencyMs:  10000.0,
			expectedScore: 0.9,
		},
		{
			name:          "medium latency",
			latencyMs:     5000.0,
			maxLatencyMs:  10000.0,
			expectedScore: 0.5,
		},
		{
			name:          "high latency at threshold",
			latencyMs:     10000.0,
			maxLatencyMs:  10000.0,
			expectedScore: 0.0,
		},
		{
			name:          "latency above threshold",
			latencyMs:     15000.0,
			maxLatencyMs:  10000.0,
			expectedScore: 0.0,
		},
		{
			name:          "negative latency treated as zero",
			latencyMs:     -100.0,
			maxLatencyMs:  10000.0,
			expectedScore: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := metrics.NewLatency(metrics.WithMaxLatencyMs(tt.maxLatencyMs))
			sample := eval.EvalSample{
				Metadata: map[string]any{
					"latency_ms": tt.latencyMs,
				},
			}

			score, err := l.Score(context.Background(), sample)

			require.NoError(t, err)
			assert.InDelta(t, tt.expectedScore, score, 0.01)
		})
	}
}

func TestLatency_Score_DifferentNumericTypes(t *testing.T) {
	tests := []struct {
		name      string
		latencyMs any
		wantScore float64
	}{
		{
			name:      "float64",
			latencyMs: float64(1000.0),
			wantScore: 0.9,
		},
		{
			name:      "float32",
			latencyMs: float32(1000.0),
			wantScore: 0.9,
		},
		{
			name:      "int",
			latencyMs: int(1000),
			wantScore: 0.9,
		},
		{
			name:      "int64",
			latencyMs: int64(1000),
			wantScore: 0.9,
		},
		{
			name:      "int32",
			latencyMs: int32(1000),
			wantScore: 0.9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := metrics.NewLatency(metrics.WithMaxLatencyMs(10000.0))
			sample := eval.EvalSample{
				Metadata: map[string]any{
					"latency_ms": tt.latencyMs,
				},
			}

			score, err := l.Score(context.Background(), sample)

			require.NoError(t, err)
			assert.InDelta(t, tt.wantScore, score, 0.01)
		})
	}
}

func TestLatency_Score_MissingMetadata(t *testing.T) {
	l := metrics.NewLatency()
	sample := eval.EvalSample{
		Metadata: map[string]any{},
	}

	score, err := l.Score(context.Background(), sample)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing metadata key")
	assert.Equal(t, 0.0, score)
}

func TestLatency_Score_NilMetadata(t *testing.T) {
	l := metrics.NewLatency()
	sample := eval.EvalSample{
		Metadata: nil,
	}

	score, err := l.Score(context.Background(), sample)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing metadata key")
	assert.Equal(t, 0.0, score)
}

func TestLatency_Score_InvalidMetadataType(t *testing.T) {
	l := metrics.NewLatency()
	sample := eval.EvalSample{
		Metadata: map[string]any{
			"latency_ms": "not a number",
		},
	}

	score, err := l.Score(context.Background(), sample)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported numeric type")
	assert.Equal(t, 0.0, score)
}

func TestLatency_Score_CustomMaxLatency(t *testing.T) {
	l := metrics.NewLatency(metrics.WithMaxLatencyMs(5000.0))
	sample := eval.EvalSample{
		Metadata: map[string]any{
			"latency_ms": 2500.0,
		},
	}

	score, err := l.Score(context.Background(), sample)

	require.NoError(t, err)
	// 1.0 - (2500 / 5000) = 0.5
	assert.InDelta(t, 0.5, score, 0.01)
}

func TestLatency_Score_DefaultMaxLatency(t *testing.T) {
	l := metrics.NewLatency()
	sample := eval.EvalSample{
		Metadata: map[string]any{
			"latency_ms": 5000.0,
		},
	}

	score, err := l.Score(context.Background(), sample)

	require.NoError(t, err)
	// Default max is 10000, so: 1.0 - (5000 / 10000) = 0.5
	assert.InDelta(t, 0.5, score, 0.01)
}

func TestLatency_Score_ZeroMaxLatency(t *testing.T) {
	// WithMaxLatencyMs with 0 or negative should be ignored
	l := metrics.NewLatency(metrics.WithMaxLatencyMs(0))
	sample := eval.EvalSample{
		Metadata: map[string]any{
			"latency_ms": 5000.0,
		},
	}

	score, err := l.Score(context.Background(), sample)

	require.NoError(t, err)
	// Should use default max latency
	assert.InDelta(t, 0.5, score, 0.01)
}

func TestLatency_Score_ContextIgnored(t *testing.T) {
	// Latency metric doesn't use context, but it should accept it
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel context

	l := metrics.NewLatency()
	sample := eval.EvalSample{
		Metadata: map[string]any{
			"latency_ms": 1000.0,
		},
	}

	score, err := l.Score(ctx, sample)

	// Should succeed despite cancelled context
	require.NoError(t, err)
	assert.InDelta(t, 0.9, score, 0.01)
}

func TestLatency_DefaultConstant(t *testing.T) {
	// Verify the default constant is as expected
	assert.Equal(t, 10000.0, metrics.DefaultMaxLatencyMs)
}
