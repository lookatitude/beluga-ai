package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Precision@K Tests
// ============================================================================

func TestPrecisionAtK(t *testing.T) {
	relevantSet := map[string]bool{"a": true, "b": true, "c": true}

	tests := []struct {
		name     string
		results  []RetrievalResult
		k        int
		expected float64
	}{
		{
			name: "all relevant in top K",
			results: []RetrievalResult{
				{ID: "a"}, {ID: "b"}, {ID: "c"},
			},
			k:        3,
			expected: 1.0,
		},
		{
			name: "none relevant in top K",
			results: []RetrievalResult{
				{ID: "x"}, {ID: "y"}, {ID: "z"},
			},
			k:        3,
			expected: 0.0,
		},
		{
			name: "partial relevance",
			results: []RetrievalResult{
				{ID: "a"}, {ID: "x"}, {ID: "b"},
			},
			k:        3,
			expected: 2.0 / 3.0,
		},
		{
			name: "K larger than results",
			results: []RetrievalResult{
				{ID: "a"}, {ID: "b"},
			},
			k:        5,
			expected: 1.0,
		},
		{
			name:     "empty results",
			results:  []RetrievalResult{},
			k:        5,
			expected: 0.0,
		},
		{
			name: "K is zero",
			results: []RetrievalResult{
				{ID: "a"}, {ID: "b"},
			},
			k:        0,
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := precisionAtK(tt.results, relevantSet, tt.k)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

// ============================================================================
// Recall@K Tests
// ============================================================================

func TestRecallAtK(t *testing.T) {
	relevantSet := map[string]bool{"a": true, "b": true, "c": true, "d": true}

	tests := []struct {
		name          string
		results       []RetrievalResult
		totalRelevant int
		k             int
		expected      float64
	}{
		{
			name: "all found in top K",
			results: []RetrievalResult{
				{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"},
			},
			totalRelevant: 4,
			k:             4,
			expected:      1.0,
		},
		{
			name: "partial recall",
			results: []RetrievalResult{
				{ID: "a"}, {ID: "x"}, {ID: "b"},
			},
			totalRelevant: 4,
			k:             3,
			expected:      0.5,
		},
		{
			name: "none found",
			results: []RetrievalResult{
				{ID: "x"}, {ID: "y"}, {ID: "z"},
			},
			totalRelevant: 4,
			k:             3,
			expected:      0.0,
		},
		{
			name:          "no relevant documents exist",
			results:       []RetrievalResult{{ID: "a"}},
			totalRelevant: 0,
			k:             5,
			expected:      0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := recallAtK(tt.results, relevantSet, tt.totalRelevant, tt.k)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

// ============================================================================
// NDCG Tests
// ============================================================================

func TestCalculateNDCG(t *testing.T) {
	tests := []struct {
		name      string
		results   []RetrievalResult
		relevance map[string]int
		k         int
		expected  float64
	}{
		{
			name: "perfect ranking",
			results: []RetrievalResult{
				{ID: "a"}, {ID: "b"}, {ID: "c"},
			},
			relevance: map[string]int{"a": 3, "b": 2, "c": 1},
			k:         3,
			expected:  1.0,
		},
		{
			name: "worst ranking",
			results: []RetrievalResult{
				{ID: "c"}, {ID: "b"}, {ID: "a"},
			},
			relevance: map[string]int{"a": 3, "b": 2, "c": 1},
			k:         3,
			expected:  0.786, // Approximate
		},
		{
			name:      "no relevant documents",
			results:   []RetrievalResult{{ID: "x"}, {ID: "y"}},
			relevance: map[string]int{"a": 3},
			k:         2,
			expected:  0.0,
		},
		{
			name:      "empty results",
			results:   []RetrievalResult{},
			relevance: map[string]int{"a": 3},
			k:         5,
			expected:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateNDCG(tt.results, tt.relevance, tt.k)
			assert.InDelta(t, tt.expected, result, 0.01)
		})
	}
}

// ============================================================================
// Aggregate Results Tests
// ============================================================================

func TestAggregateResults(t *testing.T) {
	tests := []struct {
		name    string
		results []EvaluationResult
		check   func(*testing.T, *AggregateResults)
	}{
		{
			name:    "empty results",
			results: []EvaluationResult{},
			check: func(t *testing.T, agg *AggregateResults) {
				assert.Equal(t, 0, agg.TotalQueries)
			},
		},
		{
			name: "single perfect result",
			results: []EvaluationResult{
				{
					PrecisionAt1: 1.0,
					PrecisionAt5: 1.0,
					RecallAt5:    1.0,
					MRR:          1.0,
					NDCG:         1.0,
					Latency:      100 * time.Millisecond,
				},
			},
			check: func(t *testing.T, agg *AggregateResults) {
				assert.Equal(t, 1, agg.TotalQueries)
				assert.Equal(t, 1.0, agg.MeanPrecisionAt1)
				assert.Equal(t, 1.0, agg.MeanMRR)
				assert.Equal(t, 1, agg.QueriesWithPerfect)
			},
		},
		{
			name: "mixed results",
			results: []EvaluationResult{
				{PrecisionAt5: 1.0, MRR: 1.0, RelevantFound: 3},
				{PrecisionAt5: 0.5, MRR: 0.5, RelevantFound: 1},
				{PrecisionAt5: 0.0, MRR: 0.0, RelevantFound: 0},
			},
			check: func(t *testing.T, agg *AggregateResults) {
				assert.Equal(t, 3, agg.TotalQueries)
				assert.InDelta(t, 0.5, agg.MeanPrecisionAt5, 0.001)
				assert.InDelta(t, 0.5, agg.MeanMRR, 0.001)
				assert.Equal(t, 1, agg.QueriesWithNoHits)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := aggregateResults(tt.results)
			tt.check(t, agg)
		})
	}
}

// ============================================================================
// Evaluator Tests
// ============================================================================

func TestRAGEvaluator_EvaluateQuery(t *testing.T) {
	ctx := context.Background()

	retriever := &mockRetriever{
		docs: []RetrievalResult{
			{ID: "1", Score: 0.95},
			{ID: "2", Score: 0.88},
			{ID: "3", Score: 0.82},
		},
	}

	evaluator, err := NewRAGEvaluator(retriever)
	require.NoError(t, err)

	query := EvaluationQuery{
		ID:             "test",
		Query:          "test query",
		RelevantDocIDs: []string{"1", "3"},
	}

	result, err := evaluator.EvaluateQuery(ctx, query, 5)
	require.NoError(t, err)

	assert.Equal(t, "test", result.QueryID)
	assert.Equal(t, 3, result.RetrievedCount)
	assert.Equal(t, 2, result.RelevantFound)
	assert.Equal(t, 1, result.FirstRelevantPos)
	assert.Equal(t, 1.0, result.MRR)
	assert.Greater(t, result.Latency, time.Duration(0))
}

func TestRAGEvaluator_EvaluateDataset(t *testing.T) {
	ctx := context.Background()

	retriever := &mockRetriever{
		docs: []RetrievalResult{
			{ID: "1", Score: 0.95},
			{ID: "2", Score: 0.88},
		},
	}

	evaluator, err := NewRAGEvaluator(retriever)
	require.NoError(t, err)

	queries := []EvaluationQuery{
		{ID: "q1", Query: "query 1", RelevantDocIDs: []string{"1"}},
		{ID: "q2", Query: "query 2", RelevantDocIDs: []string{"2"}},
		{ID: "q3", Query: "query 3", RelevantDocIDs: []string{"3"}}, // Not in results
	}

	agg, results, err := evaluator.EvaluateDataset(ctx, queries, 5)
	require.NoError(t, err)

	assert.Equal(t, 3, agg.TotalQueries)
	assert.Len(t, results, 3)
	assert.Equal(t, 1, agg.QueriesWithNoHits) // Query 3 has no hits
}

// ============================================================================
// Dataset Generation Tests
// ============================================================================

func TestGenerateSyntheticDataset(t *testing.T) {
	docs := []Document{
		{ID: "1", Content: "This is the first sentence. This is the second sentence."},
		{ID: "2", Content: "Another document here. With multiple sentences."},
	}

	queries := GenerateSyntheticDataset(docs, 2)

	assert.Len(t, queries, 4) // 2 docs * 2 queries each

	// Check that each query references its source document
	for _, q := range queries {
		assert.NotEmpty(t, q.ID)
		assert.NotEmpty(t, q.Query)
		assert.Len(t, q.RelevantDocIDs, 1)
	}
}

func TestSplitSentences(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected int // Minimum sentences expected
	}{
		{
			name:     "simple sentences",
			text:     "First sentence. Second sentence. Third sentence.",
			expected: 3,
		},
		{
			name:     "with question",
			text:     "Statement here. Question here? Another statement.",
			expected: 3,
		},
		{
			name:     "with exclamation",
			text:     "Wow! That is amazing. Really great!",
			expected: 3,
		},
		{
			name:     "short lines filtered",
			text:     "A. B. This is a longer sentence.",
			expected: 1, // Only the long sentence
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentences := splitSentences(tt.text)
			assert.GreaterOrEqual(t, len(sentences), tt.expected)
		})
	}
}

// ============================================================================
// Metrics Tests
// ============================================================================

func TestEvaluationMetrics_Creation(t *testing.T) {
	metrics, err := newEvaluationMetrics()
	require.NoError(t, err)
	require.NotNil(t, metrics)
	assert.NotNil(t, metrics.queryLatency)
	assert.NotNil(t, metrics.evaluationsTotal)
}

func TestEvaluationMetrics_Recording(t *testing.T) {
	ctx := context.Background()
	metrics, err := newEvaluationMetrics()
	require.NoError(t, err)

	// These should not panic
	result := &EvaluationResult{
		Latency: 100 * time.Millisecond,
	}
	metrics.recordQueryEvaluation(ctx, result)

	agg := &AggregateResults{
		MeanPrecisionAt5: 0.8,
		MeanRecallAt5:    0.7,
		MeanMRR:          0.9,
	}
	metrics.recordDatasetEvaluation(ctx, agg)

	// Verify values were stored
	assert.Equal(t, 0.8, metrics.latestPrecision)
	assert.Equal(t, 0.7, metrics.latestRecall)
	assert.Equal(t, 0.9, metrics.latestMRR)
}

// ============================================================================
// Benchmarks
// ============================================================================

func BenchmarkPrecisionAtK(b *testing.B) {
	relevantSet := map[string]bool{"a": true, "b": true, "c": true}
	results := make([]RetrievalResult, 100)
	for i := range results {
		results[i] = RetrievalResult{ID: string(rune('a' + i%26))}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = precisionAtK(results, relevantSet, 10)
	}
}

func BenchmarkCalculateNDCG(b *testing.B) {
	relevance := map[string]int{"a": 3, "b": 2, "c": 1, "d": 1}
	results := make([]RetrievalResult, 100)
	for i := range results {
		results[i] = RetrievalResult{ID: string(rune('a' + i%26))}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = calculateNDCG(results, relevance, 10)
	}
}

func BenchmarkEvaluateQuery(b *testing.B) {
	ctx := context.Background()
	retriever := &mockRetriever{
		docs: make([]RetrievalResult, 10),
	}

	evaluator, _ := NewRAGEvaluator(retriever)
	query := EvaluationQuery{
		ID:             "bench",
		Query:          "test",
		RelevantDocIDs: []string{"1", "2", "3"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = evaluator.EvaluateQuery(ctx, query, 10)
	}
}
