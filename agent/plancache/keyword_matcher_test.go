package plancache

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeywordMatcher_Score(t *testing.T) {
	m := &KeywordMatcher{}
	ctx := context.Background()

	tests := []struct {
		name    string
		input   string
		tmpl    *Template
		wantMin float64
		wantMax float64
	}{
		{
			name:    "identical inputs score 1.0",
			input:   "search database for user records",
			tmpl:    &Template{Keywords: ExtractKeywords("search database for user records")},
			wantMin: 1.0,
			wantMax: 1.0,
		},
		{
			name:    "completely different inputs score near 0.0",
			input:   "deploy application production",
			tmpl:    &Template{Keywords: ExtractKeywords("analyze weather forecast data")},
			wantMin: 0.0,
			wantMax: 0.05,
		},
		{
			name:    "partially overlapping keywords",
			input:   "search database for user profiles",
			tmpl:    &Template{Keywords: ExtractKeywords("search database for customer records")},
			wantMin: 0.3,
			wantMax: 0.8,
		},
		{
			name:    "similar but reworded",
			input:   "find user records in database",
			tmpl:    &Template{Keywords: ExtractKeywords("search database for user records")},
			wantMin: 0.4,
			wantMax: 0.9,
		},
		{
			name:    "empty input scores 0.0",
			input:   "",
			tmpl:    &Template{Keywords: ExtractKeywords("search database")},
			wantMin: 0.0,
			wantMax: 0.0,
		},
		{
			name:    "nil template scores 0.0",
			input:   "search database",
			tmpl:    nil,
			wantMin: 0.0,
			wantMax: 0.0,
		},
		{
			name:    "empty template keywords score 0.0",
			input:   "search database",
			tmpl:    &Template{Keywords: nil},
			wantMin: 0.0,
			wantMax: 0.0,
		},
		{
			name:    "superset input has high but not perfect score",
			input:   "search database user records profiles export",
			tmpl:    &Template{Keywords: ExtractKeywords("search database user records")},
			wantMin: 0.5,
			wantMax: 0.9,
		},
		{
			name:    "single common keyword",
			input:   "deploy server production environment",
			tmpl:    &Template{Keywords: ExtractKeywords("deploy application staging cluster")},
			wantMin: 0.1,
			wantMax: 0.3,
		},
		{
			name:    "repeated keywords with different frequencies",
			input:   "search search search database",
			tmpl:    &Template{Keywords: []string{"search", "search", "search", "database"}},
			wantMin: 0.4,
			wantMax: 0.6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, err := m.Score(ctx, tt.input, tt.tmpl)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, score, tt.wantMin, "score %f below minimum %f", score, tt.wantMin)
			assert.LessOrEqual(t, score, tt.wantMax, "score %f above maximum %f", score, tt.wantMax)
		})
	}
}

func TestKeywordMatcher_Score_ContextCancellation(t *testing.T) {
	m := &KeywordMatcher{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tmpl := &Template{Keywords: []string{"search", "database"}}
	score, err := m.Score(ctx, "search database", tmpl)
	assert.Error(t, err)
	assert.Equal(t, 0.0, score)
}
