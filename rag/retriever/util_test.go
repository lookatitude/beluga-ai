package retriever

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lookatitude/beluga-ai/schema"
)

// --- Tests for utility functions ---

func TestSortByScore_Detailed(t *testing.T) {
	tests := []struct {
		name     string
		input    []schema.Document
		expected []string // Expected order of IDs after sorting
	}{
		{
			name: "already sorted",
			input: []schema.Document{
				{ID: "a", Score: 0.9},
				{ID: "b", Score: 0.7},
				{ID: "c", Score: 0.5},
			},
			expected: []string{"a", "b", "c"},
		},
		{
			name: "reverse order",
			input: []schema.Document{
				{ID: "a", Score: 0.3},
				{ID: "b", Score: 0.7},
				{ID: "c", Score: 0.9},
			},
			expected: []string{"c", "b", "a"},
		},
		{
			name: "mixed order",
			input: []schema.Document{
				{ID: "a", Score: 0.5},
				{ID: "b", Score: 0.9},
				{ID: "c", Score: 0.3},
				{ID: "d", Score: 0.7},
			},
			expected: []string{"b", "d", "a", "c"},
		},
		{
			name: "equal scores",
			input: []schema.Document{
				{ID: "a", Score: 0.5},
				{ID: "b", Score: 0.5},
				{ID: "c", Score: 0.5},
			},
			expected: []string{"a", "b", "c"}, // Order preserved when equal
		},
		{
			name:     "empty",
			input:    []schema.Document{},
			expected: []string{},
		},
		{
			name: "single document",
			input: []schema.Document{
				{ID: "a", Score: 0.5},
			},
			expected: []string{"a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docs := make([]schema.Document, len(tt.input))
			copy(docs, tt.input)

			sortByScore(docs)

			assert.Len(t, docs, len(tt.expected))
			for i, expectedID := range tt.expected {
				assert.Equal(t, expectedID, docs[i].ID, "position %d", i)
			}

			// Verify descending order
			for i := 1; i < len(docs); i++ {
				assert.GreaterOrEqual(t, docs[i-1].Score, docs[i].Score,
					"scores should be in descending order")
			}
		})
	}
}

func TestDedup_Detailed(t *testing.T) {
	tests := []struct {
		name     string
		input    []schema.Document
		expected map[string]float64 // ID -> expected score (highest)
	}{
		{
			name: "no duplicates",
			input: []schema.Document{
				{ID: "a", Score: 0.9},
				{ID: "b", Score: 0.8},
				{ID: "c", Score: 0.7},
			},
			expected: map[string]float64{
				"a": 0.9,
				"b": 0.8,
				"c": 0.7,
			},
		},
		{
			name: "duplicates with different scores",
			input: []schema.Document{
				{ID: "a", Score: 0.5},
				{ID: "b", Score: 0.8},
				{ID: "a", Score: 0.9}, // Higher score
				{ID: "c", Score: 0.3},
				{ID: "b", Score: 0.2}, // Lower score
			},
			expected: map[string]float64{
				"a": 0.9, // Keeps highest
				"b": 0.8, // Keeps highest
				"c": 0.3,
			},
		},
		{
			name: "all duplicates",
			input: []schema.Document{
				{ID: "a", Score: 0.5},
				{ID: "a", Score: 0.9},
				{ID: "a", Score: 0.3},
			},
			expected: map[string]float64{
				"a": 0.9,
			},
		},
		{
			name:     "empty",
			input:    []schema.Document{},
			expected: map[string]float64{},
		},
		{
			name: "single document",
			input: []schema.Document{
				{ID: "a", Score: 0.5},
			},
			expected: map[string]float64{
				"a": 0.5,
			},
		},
		{
			name: "preserves first occurrence position",
			input: []schema.Document{
				{ID: "a", Score: 0.5},
				{ID: "b", Score: 0.8},
				{ID: "c", Score: 0.3},
				{ID: "a", Score: 0.9}, // Higher score, but a already seen
			},
			expected: map[string]float64{
				"a": 0.9, // Score updated to highest
				"b": 0.8,
				"c": 0.3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dedup(tt.input)

			assert.Len(t, result, len(tt.expected), "result length should match unique IDs")

			// Check each document
			resultMap := make(map[string]float64)
			for _, doc := range result {
				resultMap[doc.ID] = doc.Score
			}

			for id, expectedScore := range tt.expected {
				score, ok := resultMap[id]
				assert.True(t, ok, "ID %s should be in result", id)
				assert.Equal(t, expectedScore, score, "ID %s should have score %f", id, expectedScore)
			}

			// Verify no duplicate IDs in result
			seenIDs := make(map[string]bool)
			for _, doc := range result {
				assert.False(t, seenIDs[doc.ID], "ID %s appears multiple times in result", doc.ID)
				seenIDs[doc.ID] = true
			}
		})
	}
}

func TestDedupPreservesMetadata(t *testing.T) {
	input := []schema.Document{
		{ID: "a", Score: 0.5, Content: "first", Metadata: map[string]any{"version": 1}},
		{ID: "a", Score: 0.9, Content: "second", Metadata: map[string]any{"version": 2}},
	}

	result := dedup(input)
	assert.Len(t, result, 1)
	assert.Equal(t, "a", result[0].ID)
	assert.Equal(t, 0.9, result[0].Score)
	// Should preserve metadata from highest-scoring version
	assert.Equal(t, "second", result[0].Content)
	assert.Equal(t, 2, result[0].Metadata["version"])
}

func TestDedupWithZeroScores(t *testing.T) {
	input := []schema.Document{
		{ID: "a", Score: 0.0},
		{ID: "b", Score: 0.0},
		{ID: "a", Score: 0.0},
	}

	result := dedup(input)
	assert.Len(t, result, 2, "should have 2 unique IDs")

	idSet := make(map[string]bool)
	for _, doc := range result {
		idSet[doc.ID] = true
	}
	assert.True(t, idSet["a"])
	assert.True(t, idSet["b"])
}

func TestDedupOrderStability(t *testing.T) {
	// When duplicates have equal scores, dedup should preserve the first occurrence
	input := []schema.Document{
		{ID: "a", Score: 0.5, Content: "first"},
		{ID: "b", Score: 0.8},
		{ID: "a", Score: 0.5, Content: "second"}, // Same score as first "a"
		{ID: "c", Score: 0.3},
	}

	result := dedup(input)
	assert.Len(t, result, 3)

	// Find the "a" document
	var aDoc schema.Document
	for _, doc := range result {
		if doc.ID == "a" {
			aDoc = doc
			break
		}
	}

	// Should keep first "a" since scores are equal
	assert.Equal(t, "first", aDoc.Content)
}

func TestSortByScore_NegativeScores(t *testing.T) {
	docs := []schema.Document{
		{ID: "a", Score: -0.5},
		{ID: "b", Score: 0.5},
		{ID: "c", Score: -0.3},
		{ID: "d", Score: 0.0},
	}

	sortByScore(docs)

	assert.Equal(t, "b", docs[0].ID, "highest positive")
	assert.Equal(t, "d", docs[1].ID, "zero")
	assert.Equal(t, "c", docs[2].ID, "less negative")
	assert.Equal(t, "a", docs[3].ID, "most negative")
}

func TestDedup_EmptyID(t *testing.T) {
	input := []schema.Document{
		{ID: "", Score: 0.5},
		{ID: "", Score: 0.9},
		{ID: "a", Score: 0.8},
	}

	result := dedup(input)
	// Empty IDs are treated as the same document
	assert.Len(t, result, 2)

	// Should keep highest score for empty ID
	for _, doc := range result {
		if doc.ID == "" {
			assert.Equal(t, 0.9, doc.Score)
		}
	}
}

func TestSortByScore_LargeDataset(t *testing.T) {
	// Test with many documents
	docs := make([]schema.Document, 1000)
	for i := range docs {
		docs[i] = schema.Document{
			ID:    string(rune('a' + i%26)),
			Score: float64(i % 100),
		}
	}

	sortByScore(docs)

	// Verify sorted in descending order
	for i := 1; i < len(docs); i++ {
		assert.GreaterOrEqual(t, docs[i-1].Score, docs[i].Score)
	}
}

func TestDedup_LargeDataset(t *testing.T) {
	// Create dataset with many duplicates
	docs := make([]schema.Document, 1000)
	for i := range docs {
		docs[i] = schema.Document{
			ID:    string(rune('a' + i%10)), // Only 10 unique IDs
			Score: float64(i),
		}
	}

	result := dedup(docs)
	assert.Len(t, result, 10, "should have 10 unique IDs")

	// Each ID should have the highest score from its duplicates
	scoreMap := make(map[string]float64)
	for _, doc := range result {
		scoreMap[doc.ID] = doc.Score
	}

	// Verify we got the highest scores
	for i := 0; i < 10; i++ {
		id := string(rune('a' + i))
		// Highest occurrence of this ID is at index 990+i
		expectedScore := float64(990 + i)
		assert.Equal(t, expectedScore, scoreMap[id])
	}
}

func TestSortByScore_MutatesInPlace(t *testing.T) {
	docs := []schema.Document{
		{ID: "a", Score: 0.3},
		{ID: "b", Score: 0.9},
	}

	// Save pointer to first element before sort
	docsBefore := make([]schema.Document, len(docs))
	copy(docsBefore, docs)

	sortByScore(docs)

	// Should mutate the slice in place (same capacity and backing array)
	// After sort: docs[0] should be "b" (highest score)
	assert.Equal(t, "b", docs[0].ID)
	assert.Equal(t, "a", docs[1].ID)

	// The slice itself is mutated, which is what we want to verify
	assert.Len(t, docs, 2)
}

func TestDedup_DoesNotMutateInput(t *testing.T) {
	input := []schema.Document{
		{ID: "a", Score: 0.5},
		{ID: "a", Score: 0.9},
	}

	originalFirstScore := input[0].Score
	_ = dedup(input)

	// Input should not be modified
	assert.Equal(t, originalFirstScore, input[0].Score)
}
