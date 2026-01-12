package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Configuration Tests
// ============================================================================

func TestRetrieverConfig_Validation(t *testing.T) {
	tests := []struct {
		name        string
		config      RetrieverConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "default config is valid",
			config:      DefaultRetrieverConfig(),
			expectError: false,
		},
		{
			name: "zero TopK is invalid",
			config: RetrieverConfig{
				TopK:        0,
				MinScore:    0.5,
				HybridAlpha: 0.5,
			},
			expectError: true,
			errorMsg:    "TopK must be positive",
		},
		{
			name: "negative MinScore is invalid",
			config: RetrieverConfig{
				TopK:        10,
				MinScore:    -0.1,
				HybridAlpha: 0.5,
			},
			expectError: true,
			errorMsg:    "MinScore must be between 0 and 1",
		},
		{
			name: "MinScore above 1 is invalid",
			config: RetrieverConfig{
				TopK:        10,
				MinScore:    1.5,
				HybridAlpha: 0.5,
			},
			expectError: true,
			errorMsg:    "MinScore must be between 0 and 1",
		},
		{
			name: "HybridAlpha below 0 is invalid",
			config: RetrieverConfig{
				TopK:        10,
				MinScore:    0.5,
				HybridAlpha: -0.1,
			},
			expectError: true,
			errorMsg:    "HybridAlpha must be between 0 and 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ============================================================================
// Retriever Creation Tests
// ============================================================================

func TestNewAdvancedRetriever(t *testing.T) {
	vectorStore := &mockVectorStore{}
	embedder := &mockEmbedder{}
	keywordStore := &mockKeywordStore{}

	tests := []struct {
		name        string
		opts        []RetrieverOption
		expectError bool
	}{
		{
			name:        "default options",
			opts:        nil,
			expectError: false,
		},
		{
			name: "custom TopK",
			opts: []RetrieverOption{
				WithTopK(20),
			},
			expectError: false,
		},
		{
			name: "custom strategy",
			opts: []RetrieverOption{
				WithDefaultStrategy(StrategySimilarity),
			},
			expectError: false,
		},
		{
			name: "all custom options",
			opts: []RetrieverOption{
				WithDefaultStrategy(StrategyKeyword),
				WithTopK(15),
				WithMinScore(0.8),
				WithHybridAlpha(0.7),
				WithQueryAnalysis(true),
				WithFallback(true),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retriever, err := NewAdvancedRetriever(vectorStore, embedder, keywordStore, tt.opts...)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, retriever)
			}
		})
	}
}

// ============================================================================
// Similarity Search Tests
// ============================================================================

func TestSimilaritySearch(t *testing.T) {
	ctx := context.Background()

	vectorStore := &mockVectorStore{
		docs: []VectorResult{
			{ID: "1", Content: "Error handling guide", Score: 0.95},
			{ID: "2", Content: "Go best practices", Score: 0.85},
			{ID: "3", Content: "Debugging tips", Score: 0.75},
			{ID: "4", Content: "Low relevance doc", Score: 0.5},
		},
	}

	retriever, err := NewAdvancedRetriever(
		vectorStore,
		&mockEmbedder{},
		&mockKeywordStore{},
		WithMinScore(0.7),
	)
	require.NoError(t, err)

	results, err := retriever.SimilaritySearch(ctx, "error handling")
	require.NoError(t, err)

	// Should filter out documents below MinScore
	assert.Len(t, results, 3)

	for _, r := range results {
		assert.GreaterOrEqual(t, r.Score, 0.7)
		assert.Equal(t, StrategySimilarity, r.Strategy)
	}
}

func TestSimilaritySearch_EmptyResults(t *testing.T) {
	ctx := context.Background()

	vectorStore := &mockVectorStore{
		docs: []VectorResult{
			{ID: "1", Content: "Low score doc", Score: 0.3},
		},
	}

	retriever, err := NewAdvancedRetriever(
		vectorStore,
		&mockEmbedder{},
		&mockKeywordStore{},
		WithMinScore(0.7),
	)
	require.NoError(t, err)

	results, err := retriever.SimilaritySearch(ctx, "query")
	require.NoError(t, err)
	assert.Empty(t, results)
}

// ============================================================================
// Keyword Search Tests
// ============================================================================

func TestKeywordSearch(t *testing.T) {
	ctx := context.Background()

	keywordStore := &mockKeywordStore{
		docs: []KeywordResult{
			{ID: "1", Content: "Error handling tutorial", Score: 20.0},
			{ID: "2", Content: "Go errors explained", Score: 15.0},
			{ID: "3", Content: "Exception patterns", Score: 10.0},
		},
	}

	retriever, err := NewAdvancedRetriever(
		&mockVectorStore{},
		&mockEmbedder{},
		keywordStore,
		WithMinScore(0.5),
	)
	require.NoError(t, err)

	results, err := retriever.KeywordSearch(ctx, "error handling")
	require.NoError(t, err)

	assert.NotEmpty(t, results)
	for _, r := range results {
		assert.Equal(t, StrategyKeyword, r.Strategy)
	}
}

// ============================================================================
// Hybrid Search Tests
// ============================================================================

func TestHybridSearch_CombinesResults(t *testing.T) {
	ctx := context.Background()

	vectorStore := &mockVectorStore{
		docs: []VectorResult{
			{ID: "1", Content: "Semantic match", Score: 0.9},
			{ID: "2", Content: "Only in similarity", Score: 0.85},
		},
	}

	keywordStore := &mockKeywordStore{
		docs: []KeywordResult{
			{ID: "1", Content: "Semantic match", Score: 18.0},
			{ID: "3", Content: "Only in keyword", Score: 15.0},
		},
	}

	retriever, err := NewAdvancedRetriever(
		vectorStore,
		&mockEmbedder{},
		keywordStore,
		WithMinScore(0.5),
	)
	require.NoError(t, err)

	results, err := retriever.HybridSearch(ctx, "test query")
	require.NoError(t, err)

	// Should have results from both sources
	assert.NotEmpty(t, results)

	// Doc "1" should appear with high score (appears in both)
	found := false
	for _, r := range results {
		if r.ID == "1" {
			found = true
			assert.Equal(t, StrategyHybrid, r.Strategy)
		}
	}
	assert.True(t, found, "Document appearing in both lists should be in results")
}

func TestHybridSearch_LimitsToTopK(t *testing.T) {
	ctx := context.Background()

	// Create many docs
	var vectorDocs []VectorResult
	var keywordDocs []KeywordResult
	for i := 0; i < 20; i++ {
		vectorDocs = append(vectorDocs, VectorResult{
			ID:      string(rune('A' + i)),
			Content: "Doc",
			Score:   0.9 - float64(i)*0.02,
		})
		keywordDocs = append(keywordDocs, KeywordResult{
			ID:      string(rune('A' + i)),
			Content: "Doc",
			Score:   20.0 - float64(i),
		})
	}

	retriever, err := NewAdvancedRetriever(
		&mockVectorStore{docs: vectorDocs},
		&mockEmbedder{},
		&mockKeywordStore{docs: keywordDocs},
		WithTopK(5),
		WithMinScore(0.5),
	)
	require.NoError(t, err)

	results, err := retriever.HybridSearch(ctx, "test")
	require.NoError(t, err)

	assert.LessOrEqual(t, len(results), 5)
}

// ============================================================================
// Query Analysis Tests
// ============================================================================

func TestAnalyzeQuery(t *testing.T) {
	retriever, _ := NewAdvancedRetriever(
		&mockVectorStore{},
		&mockEmbedder{},
		&mockKeywordStore{},
	)

	tests := []struct {
		query    string
		expected QueryType
	}{
		// Definition queries
		{"What is RAG?", QueryTypeDefinition},
		{"what are embeddings", QueryTypeDefinition},
		{"Define vector store", QueryTypeDefinition},

		// How-to queries
		{"How to implement caching", QueryTypeHowTo},
		{"how do I handle errors", QueryTypeHowTo},
		{"How can I optimize queries", QueryTypeHowTo},
		{"complete tutorial on RAG", QueryTypeHowTo},
		{"step by step guide", QueryTypeHowTo},

		// Comparison queries
		{"Redis vs Memcached", QueryTypeComparison},
		{"PostgreSQL versus MySQL", QueryTypeComparison},
		{"compare Qdrant and Pinecone", QueryTypeComparison},
		{"difference between RAG and fine-tuning", QueryTypeComparison},

		// Exact match queries
		{"JIRA-12345", QueryTypeExactMatch},
		{"abc12345-def4", QueryTypeExactMatch},

		// General queries
		{"best practices", QueryTypeGeneral},
		{"error handling patterns", QueryTypeGeneral},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			result := retriever.analyzeQuery(tt.query)
			assert.Equal(t, tt.expected, result, "Query: %s", tt.query)
		})
	}
}

// ============================================================================
// Multi-Strategy Search Tests
// ============================================================================

func TestMultiStrategySearch(t *testing.T) {
	ctx := context.Background()

	retriever, err := NewAdvancedRetriever(
		&mockVectorStore{
			docs: []VectorResult{{ID: "1", Content: "Test", Score: 0.9}},
		},
		&mockEmbedder{},
		&mockKeywordStore{
			docs: []KeywordResult{{ID: "1", Content: "Test", Score: 15.0}},
		},
		WithFallback(true),
	)
	require.NoError(t, err)

	results, err := retriever.MultiStrategySearch(ctx, "What is testing?")
	require.NoError(t, err)
	assert.NotEmpty(t, results)
}

// ============================================================================
// RRF Tests
// ============================================================================

func TestReciprocalRankFusion(t *testing.T) {
	retriever, _ := NewAdvancedRetriever(
		&mockVectorStore{},
		&mockEmbedder{},
		&mockKeywordStore{},
	)

	list1 := []RetrievalResult{
		{ID: "A", Score: 0.9},
		{ID: "B", Score: 0.8},
		{ID: "C", Score: 0.7},
	}

	list2 := []RetrievalResult{
		{ID: "B", Score: 0.95},
		{ID: "A", Score: 0.85},
		{ID: "D", Score: 0.75},
	}

	merged := retriever.reciprocalRankFusion(list1, list2)

	// Should contain all unique IDs
	ids := make(map[string]bool)
	for _, r := range merged {
		ids[r.ID] = true
	}
	assert.Len(t, ids, 4) // A, B, C, D

	// B should be ranked high (appears first in list2, second in list1)
	assert.True(t, merged[0].ID == "A" || merged[0].ID == "B",
		"Top result should be A or B (appear in both lists)")
}

// ============================================================================
// Helper Function Tests
// ============================================================================

func TestTruncateQuery(t *testing.T) {
	short := "Short query"
	assert.Equal(t, short, truncateQuery(short))

	long := string(make([]byte, 150))
	truncated := truncateQuery(long)
	assert.Len(t, truncated, 103) // 100 + "..."
	assert.True(t, len(truncated) <= 103)
}

func TestNormalizeScore(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{0, 0.5},           // Sigmoid of 0 is 0.5
		{100, 0.99995},     // High score → close to 1
		{-100, 0.00005},    // Low score → close to 0
	}

	for _, tt := range tests {
		result := normalizeScore(tt.input)
		assert.InDelta(t, tt.expected, result, 0.001)
	}
}

func TestContainsID(t *testing.T) {
	tests := []struct {
		query    string
		expected bool
	}{
		{"JIRA-123", true},
		{"ABC-999", true},
		{"abc12345-def4", true},
		{"normal query", false},
		{"error handling", false},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			result := containsID(tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ============================================================================
// Metrics Tests
// ============================================================================

func TestRetrieverMetrics_Creation(t *testing.T) {
	metrics, err := newRetrieverMetrics()
	require.NoError(t, err)
	require.NotNil(t, metrics)
	assert.NotNil(t, metrics.searchLat)
	assert.NotNil(t, metrics.embedLat)
	assert.NotNil(t, metrics.resultsCount)
	assert.NotNil(t, metrics.errors)
}

func TestRetrieverMetrics_Recording(t *testing.T) {
	ctx := context.Background()
	metrics, err := newRetrieverMetrics()
	require.NoError(t, err)

	// These should not panic
	metrics.recordSearchLatency(ctx, "similarity", 100*time.Millisecond)
	metrics.recordEmbeddingLatency(ctx, 50*time.Millisecond)
	metrics.recordResults(ctx, "hybrid", 10)
	metrics.recordError(ctx, "similarity", "embedding_failed")
}

// ============================================================================
// Benchmarks
// ============================================================================

func BenchmarkSimilaritySearch(b *testing.B) {
	ctx := context.Background()

	retriever, _ := NewAdvancedRetriever(
		&mockVectorStore{
			docs: []VectorResult{
				{ID: "1", Content: "Test", Score: 0.9},
			},
		},
		&mockEmbedder{},
		&mockKeywordStore{},
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = retriever.SimilaritySearch(ctx, "test query")
	}
}

func BenchmarkHybridSearch(b *testing.B) {
	ctx := context.Background()

	retriever, _ := NewAdvancedRetriever(
		&mockVectorStore{
			docs: []VectorResult{
				{ID: "1", Content: "Test", Score: 0.9},
			},
		},
		&mockEmbedder{},
		&mockKeywordStore{
			docs: []KeywordResult{
				{ID: "1", Content: "Test", Score: 15.0},
			},
		},
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = retriever.HybridSearch(ctx, "test query")
	}
}

func BenchmarkQueryAnalysis(b *testing.B) {
	retriever, _ := NewAdvancedRetriever(
		&mockVectorStore{},
		&mockEmbedder{},
		&mockKeywordStore{},
	)

	queries := []string{
		"What is RAG?",
		"How to implement caching",
		"Redis vs Memcached",
		"JIRA-12345",
		"best practices for error handling",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, q := range queries {
			_ = retriever.analyzeQuery(q)
		}
	}
}

func BenchmarkRRF(b *testing.B) {
	retriever, _ := NewAdvancedRetriever(
		&mockVectorStore{},
		&mockEmbedder{},
		&mockKeywordStore{},
	)

	list1 := make([]RetrievalResult, 100)
	list2 := make([]RetrievalResult, 100)
	for i := 0; i < 100; i++ {
		list1[i] = RetrievalResult{ID: string(rune(i)), Score: 0.9 - float64(i)*0.005}
		list2[i] = RetrievalResult{ID: string(rune(i + 50)), Score: 0.9 - float64(i)*0.005}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = retriever.reciprocalRankFusion(list1, list2)
	}
}
