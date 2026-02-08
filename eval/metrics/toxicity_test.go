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
var _ eval.Metric = (*metrics.Toxicity)(nil)

func TestNewToxicity(t *testing.T) {
	tox := metrics.NewToxicity()
	require.NotNil(t, tox)
	assert.Equal(t, "toxicity", tox.Name())
}

func TestNewToxicity_WithCustomKeywords(t *testing.T) {
	keywords := []string{"bad", "worse", "worst"}
	tox := metrics.NewToxicity(metrics.WithKeywords(keywords))
	require.NotNil(t, tox)
	assert.Equal(t, "toxicity", tox.Name())
}

func TestToxicity_Name(t *testing.T) {
	tox := metrics.NewToxicity()
	assert.Equal(t, "toxicity", tox.Name())
}

func TestToxicity_Score_NoKeywords(t *testing.T) {
	tox := metrics.NewToxicity()
	sample := eval.EvalSample{
		Output: "This is a perfectly clean and helpful response.",
	}

	score, err := tox.Score(context.Background(), sample)

	require.NoError(t, err)
	// No toxic keywords matched, score should be 1.0
	assert.Equal(t, 1.0, score)
}

func TestToxicity_Score_OneMatch(t *testing.T) {
	tox := metrics.NewToxicity()
	sample := eval.EvalSample{
		Output: "I hate when this happens.",
	}

	score, err := tox.Score(context.Background(), sample)

	require.NoError(t, err)
	// One keyword matched out of default set (12 keywords)
	// Score = 1.0 - (1 / 12) ≈ 0.917
	assert.Greater(t, score, 0.9)
	assert.Less(t, score, 1.0)
}

func TestToxicity_Score_MultipleMatches(t *testing.T) {
	tox := metrics.NewToxicity()
	sample := eval.EvalSample{
		Output: "I hate violent racist content.",
	}

	score, err := tox.Score(context.Background(), sample)

	require.NoError(t, err)
	// Multiple keywords matched
	assert.Less(t, score, 1.0)
}

func TestToxicity_Score_AllMatches(t *testing.T) {
	keywords := []string{"bad", "worse", "worst"}
	tox := metrics.NewToxicity(metrics.WithKeywords(keywords))
	sample := eval.EvalSample{
		Output: "This is bad, worse, and the worst.",
	}

	score, err := tox.Score(context.Background(), sample)

	require.NoError(t, err)
	// All 3 keywords matched: 1.0 - (3 / 3) = 0.0
	assert.Equal(t, 0.0, score)
}

func TestToxicity_Score_CaseInsensitive(t *testing.T) {
	keywords := []string{"bad"}
	tox := metrics.NewToxicity(metrics.WithKeywords(keywords))

	tests := []struct {
		name   string
		output string
	}{
		{"lowercase", "this is bad"},
		{"uppercase", "this is BAD"},
		{"mixed case", "this is BaD"},
		{"title case", "this is Bad"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sample := eval.EvalSample{Output: tt.output}
			score, err := tox.Score(context.Background(), sample)
			require.NoError(t, err)
			// Should match regardless of case
			assert.Equal(t, 0.0, score)
		})
	}
}

func TestToxicity_Score_EmptyOutput(t *testing.T) {
	tox := metrics.NewToxicity()
	sample := eval.EvalSample{
		Output: "",
	}

	score, err := tox.Score(context.Background(), sample)

	require.NoError(t, err)
	// Empty output has no matches
	assert.Equal(t, 1.0, score)
}

func TestToxicity_Score_EmptyKeywordList(t *testing.T) {
	tox := metrics.NewToxicity(metrics.WithKeywords([]string{}))
	sample := eval.EvalSample{
		Output: "This has hate and violence.",
	}

	score, err := tox.Score(context.Background(), sample)

	require.NoError(t, err)
	// No keywords to check, so score should be 1.0 (not toxic)
	assert.Equal(t, 1.0, score)
}

func TestToxicity_Score_CustomKeywords(t *testing.T) {
	keywords := []string{"foo", "bar", "baz"}
	tox := metrics.NewToxicity(metrics.WithKeywords(keywords))

	tests := []struct {
		name          string
		output        string
		expectedScore float64
	}{
		{
			name:          "no matches",
			output:        "This is clean content.",
			expectedScore: 1.0,
		},
		{
			name:          "one match",
			output:        "This contains foo.",
			expectedScore: 2.0 / 3.0, // 1.0 - (1/3)
		},
		{
			name:          "two matches",
			output:        "This has foo and bar.",
			expectedScore: 1.0 / 3.0, // 1.0 - (2/3)
		},
		{
			name:          "all matches",
			output:        "foo bar baz",
			expectedScore: 0.0, // 1.0 - (3/3)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sample := eval.EvalSample{Output: tt.output}
			score, err := tox.Score(context.Background(), sample)
			require.NoError(t, err)
			assert.InDelta(t, tt.expectedScore, score, 0.01)
		})
	}
}

func TestToxicity_Score_PartialWordMatches(t *testing.T) {
	// Keywords are matched as substrings
	keywords := []string{"hate"}
	tox := metrics.NewToxicity(metrics.WithKeywords(keywords))

	tests := []struct {
		name          string
		output        string
		shouldMatch   bool
	}{
		{
			name:        "exact match",
			output:      "I hate this.",
			shouldMatch: true,
		},
		{
			name:        "partial match in word",
			output:      "I whatever this.", // contains "hate"
			shouldMatch: true,
		},
		{
			name:        "no match",
			output:      "I dislike this.",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sample := eval.EvalSample{Output: tt.output}
			score, err := tox.Score(context.Background(), sample)
			require.NoError(t, err)
			if tt.shouldMatch {
				assert.Equal(t, 0.0, score)
			} else {
				assert.Equal(t, 1.0, score)
			}
		})
	}
}

func TestToxicity_Score_DefaultKeywords(t *testing.T) {
	// Test that default keywords are applied
	tox := metrics.NewToxicity()

	// These should match default keywords
	toxicOutputs := []string{
		"I hate you",
		"This is violent",
		"Kill the process",
		"Die already",
		"This is racist",
		"That's sexist",
		"Homophobic comment",
		"That's abuse",
		"This is a threat",
	}

	for _, output := range toxicOutputs {
		t.Run(output, func(t *testing.T) {
			sample := eval.EvalSample{Output: output}
			score, err := tox.Score(context.Background(), sample)
			require.NoError(t, err)
			// Should detect toxicity
			assert.Less(t, score, 1.0)
		})
	}
}

func TestToxicity_Score_NegativeScoreClamped(t *testing.T) {
	// If somehow we get more matches than keywords (shouldn't happen),
	// score should be clamped to 0
	keywords := []string{"a"}
	tox := metrics.NewToxicity(metrics.WithKeywords(keywords))
	sample := eval.EvalSample{
		Output: "aaa", // Contains "a" multiple times
	}

	score, err := tox.Score(context.Background(), sample)

	require.NoError(t, err)
	// Score should be 0.0 (clamped)
	assert.GreaterOrEqual(t, score, 0.0)
}

func TestToxicity_Score_WithThreshold(t *testing.T) {
	// WithThreshold option should set threshold (though it doesn't affect Score directly)
	tox := metrics.NewToxicity(metrics.WithThreshold(0.7))
	sample := eval.EvalSample{
		Output: "Clean content.",
	}

	score, err := tox.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 1.0, score)
}

func TestToxicity_Score_LongOutput(t *testing.T) {
	keywords := []string{"bad"}
	tox := metrics.NewToxicity(metrics.WithKeywords(keywords))

	// Long output with one toxic keyword
	longOutput := "This is a very long output that contains many words and only one bad word in it."
	sample := eval.EvalSample{Output: longOutput}

	score, err := tox.Score(context.Background(), sample)

	require.NoError(t, err)
	assert.Equal(t, 0.0, score) // 1 out of 1 keyword matched
}

func TestToxicity_Score_UnicodeContent(t *testing.T) {
	keywords := []string{"bad"}
	tox := metrics.NewToxicity(metrics.WithKeywords(keywords))

	sample := eval.EvalSample{
		Output: "This is 好的 content, not bad 内容.",
	}

	score, err := tox.Score(context.Background(), sample)

	require.NoError(t, err)
	// Should find "bad" even in Unicode text
	assert.Equal(t, 0.0, score)
}

func TestToxicity_Score_RepeatedKeywords(t *testing.T) {
	keywords := []string{"bad"}
	tox := metrics.NewToxicity(metrics.WithKeywords(keywords))

	sample := eval.EvalSample{
		Output: "bad bad bad",
	}

	score, err := tox.Score(context.Background(), sample)

	require.NoError(t, err)
	// Should count as 1 keyword matched, not 3 occurrences
	assert.Equal(t, 0.0, score) // 1 keyword matched out of 1 = 0.0
}
