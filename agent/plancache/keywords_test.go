package plancache

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractKeywords(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "empty input",
			input: "",
			want:  nil,
		},
		{
			name:  "only stop words",
			input: "the a an is are was",
			want:  nil,
		},
		{
			name:  "single keyword",
			input: "calculate the sum",
			want:  []string{"calculate", "sum"},
		},
		{
			name:  "repeated words ranked higher",
			input: "search search the database for search results",
			want:  []string{"search", "database", "results"},
		},
		{
			name:  "mixed case normalized",
			input: "Deploy the Application to PRODUCTION server",
			want:  []string{"application", "deploy", "production", "server"},
		},
		{
			name:  "punctuation handled",
			input: "run tests, check coverage; deploy!",
			want:  []string{"check", "coverage", "deploy", "run", "tests"},
		},
		{
			name:  "numbers included",
			input: "process batch123 with 5 retries",
			want:  []string{"batch123", "process", "retries"},
		},
		{
			name:  "single character tokens filtered",
			input: "I a x run tests",
			want:  []string{"run", "tests"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractKeywords(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractKeywords_MaxLimit(t *testing.T) {
	// Build input with more than maxKeywords unique words.
	words := make([]string, 30)
	for i := range words {
		words[i] = "word" + string(rune('a'+i))
	}
	input := ""
	for _, w := range words {
		input += w + " "
	}

	got := ExtractKeywords(input)
	assert.LessOrEqual(t, len(got), maxKeywords)
}
