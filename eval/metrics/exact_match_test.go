package metrics_test

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/eval"
	"github.com/lookatitude/beluga-ai/v2/eval/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ eval.Metric = (*metrics.ExactMatch)(nil)

func TestExactMatch_Name(t *testing.T) {
	e := metrics.NewExactMatch()
	assert.Equal(t, "exact_match", e.Name())
}

func TestExactMatch_Score_Table(t *testing.T) {
	tests := []struct {
		name          string
		output        string
		expected      string
		caseSensitive bool
		want          float64
	}{
		{name: "exact match", output: "paris", expected: "paris", want: 1.0},
		{name: "case-folded match", output: "Paris", expected: "paris", want: 1.0},
		{name: "whitespace-trim match", output: "  Paris  ", expected: "paris", want: 1.0},
		{name: "mismatch", output: "lisbon", expected: "paris", want: 0.0},
		{name: "empty expected → zero", output: "anything", expected: "", want: 0.0},
		{name: "empty both → zero", output: "", expected: "", want: 0.0},
		{name: "case-sensitive mismatch", output: "Paris", expected: "paris", caseSensitive: true, want: 0.0},
		{name: "case-sensitive exact", output: "paris", expected: "paris", caseSensitive: true, want: 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e *metrics.ExactMatch
			if tt.caseSensitive {
				e = metrics.NewExactMatch(metrics.WithCaseSensitive())
			} else {
				e = metrics.NewExactMatch()
			}
			got, err := e.Score(context.Background(), eval.EvalSample{
				Output:         tt.output,
				ExpectedOutput: tt.expected,
			})
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
