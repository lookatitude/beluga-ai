// Package advancedretrieval demonstrates similarity, keyword, and hybrid search
// for production-ready RAG systems with comprehensive OTEL instrumentation.
//
// This example shows you how to build a multi-strategy retriever that
// intelligently selects the best approach based on query characteristics.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// ============================================================================
// Types and Configuration
// ============================================================================

// RetrievalStrategy defines how to search for documents.
type RetrievalStrategy string

const (
	StrategySimilarity RetrievalStrategy = "similarity"
	StrategyKeyword    RetrievalStrategy = "keyword"
	StrategyHybrid     RetrievalStrategy = "hybrid"
	StrategyMulti      RetrievalStrategy = "multi"
)

// QueryType represents the detected type of query.
type QueryType string

const (
	QueryTypeDefinition QueryType = "definition"
	QueryTypeHowTo      QueryType = "how_to"
	QueryTypeComparison QueryType = "comparison"
	QueryTypeExactMatch QueryType = "exact_match"
	QueryTypeGeneral    QueryType = "general"
)

// RetrieverConfig configures the advanced retriever.
type RetrieverConfig struct {
	DefaultStrategy     RetrievalStrategy
	TopK                int
	MinScore            float64
	HybridAlpha         float64
	EnableQueryAnalysis bool
	FallbackEnabled     bool
}

// DefaultRetrieverConfig returns sensible defaults.
func DefaultRetrieverConfig() RetrieverConfig {
	return RetrieverConfig{
		DefaultStrategy:     StrategyHybrid,
		TopK:                10,
		MinScore:            0.7,
		HybridAlpha:         0.5,
		EnableQueryAnalysis: true,
		FallbackEnabled:     true,
	}
}

// RetrieverOption is a functional option for configuring the retriever.
type RetrieverOption func(*RetrieverConfig)

// WithDefaultStrategy sets the default retrieval strategy.
func WithDefaultStrategy(s RetrievalStrategy) RetrieverOption {
	return func(c *RetrieverConfig) { c.DefaultStrategy = s }
}

// WithTopK sets the number of results to return.
func WithTopK(k int) RetrieverOption {
	return func(c *RetrieverConfig) { c.TopK = k }
}

// WithMinScore sets the minimum score threshold.
func WithMinScore(score float64) RetrieverOption {
	return func(c *RetrieverConfig) { c.MinScore = score }
}

// WithHybridAlpha sets the weight for similarity in hybrid search.
func WithHybridAlpha(alpha float64) RetrieverOption {
	return func(c *RetrieverConfig) { c.HybridAlpha = alpha }
}

// WithQueryAnalysis enables/disables automatic query analysis.
func WithQueryAnalysis(enabled bool) RetrieverOption {
	return func(c *RetrieverConfig) { c.EnableQueryAnalysis = enabled }
}

// WithFallback enables/disables fallback to hybrid on poor results.
func WithFallback(enabled bool) RetrieverOption {
	return func(c *RetrieverConfig) { c.FallbackEnabled = enabled }
}

// RetrievalResult represents a single retrieval result.
type RetrievalResult struct {
	ID       string
	Content  string
	Score    float64
	Metadata map[string]interface{}
	Strategy RetrievalStrategy
}

// Document represents a document to be indexed.
type Document struct {
	ID       string
	Content  string
	Metadata map[string]interface{}
}

// ============================================================================
// Interfaces
// ============================================================================

// VectorStore interface for vector similarity search.
type VectorStore interface {
	SimilaritySearch(ctx context.Context, embedding []float64, topK int) ([]VectorResult, error)
	Add(ctx context.Context, id string, embedding []float64, content string, metadata map[string]interface{}) error
}

// VectorResult represents a vector search result.
type VectorResult struct {
	ID       string
	Content  string
	Score    float64
	Metadata map[string]interface{}
}

// Embedder interface for generating embeddings.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float64, error)
}

// KeywordStore interface for keyword-based search.
type KeywordStore interface {
	Search(ctx context.Context, query string, topK int) ([]KeywordResult, error)
	Index(ctx context.Context, id string, content string) error
}

// KeywordResult represents a keyword search result.
type KeywordResult struct {
	ID      string
	Content string
	Score   float64
}

// ============================================================================
// Advanced Retriever
// ============================================================================

// AdvancedRetriever implements multiple retrieval strategies.
type AdvancedRetriever struct {
	config       RetrieverConfig
	vectorStore  VectorStore
	embedder     Embedder
	keywordStore KeywordStore
	metrics      *RetrieverMetrics
	tracer       trace.Tracer
}

// NewAdvancedRetriever creates a retriever with multiple strategies.
func NewAdvancedRetriever(
	vectorStore VectorStore,
	embedder Embedder,
	keywordStore KeywordStore,
	opts ...RetrieverOption,
) (*AdvancedRetriever, error) {
	config := DefaultRetrieverConfig()

	for _, opt := range opts {
		opt(&config)
	}

	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	metrics, err := newRetrieverMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics: %w", err)
	}

	return &AdvancedRetriever{
		config:       config,
		vectorStore:  vectorStore,
		embedder:     embedder,
		keywordStore: keywordStore,
		metrics:      metrics,
		tracer:       otel.Tracer("beluga.retrieval"),
	}, nil
}

func validateConfig(config RetrieverConfig) error {
	if config.TopK <= 0 {
		return errors.New("TopK must be positive")
	}
	if config.MinScore < 0 || config.MinScore > 1 {
		return errors.New("MinScore must be between 0 and 1")
	}
	if config.HybridAlpha < 0 || config.HybridAlpha > 1 {
		return errors.New("HybridAlpha must be between 0 and 1")
	}
	return nil
}

// Retrieve searches using the specified strategy.
func (r *AdvancedRetriever) Retrieve(
	ctx context.Context,
	query string,
	strategy RetrievalStrategy,
) ([]RetrievalResult, error) {
	switch strategy {
	case StrategySimilarity:
		return r.SimilaritySearch(ctx, query)
	case StrategyKeyword:
		return r.KeywordSearch(ctx, query)
	case StrategyHybrid:
		return r.HybridSearch(ctx, query)
	case StrategyMulti:
		return r.MultiStrategySearch(ctx, query)
	default:
		return r.HybridSearch(ctx, query)
	}
}

// SimilaritySearch performs vector similarity search.
func (r *AdvancedRetriever) SimilaritySearch(
	ctx context.Context,
	query string,
) ([]RetrievalResult, error) {
	ctx, span := r.tracer.Start(ctx, "retriever.SimilaritySearch",
		trace.WithAttributes(
			attribute.String("query", truncateQuery(query)),
			attribute.Int("top_k", r.config.TopK),
		),
	)
	defer span.End()

	start := time.Now()

	// Step 1: Generate embedding for the query
	embedding, err := r.embedder.Embed(ctx, query)
	if err != nil {
		span.RecordError(err)
		r.metrics.recordError(ctx, "similarity", "embedding_failed")
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	embedTime := time.Since(start)
	r.metrics.recordEmbeddingLatency(ctx, embedTime)

	// Step 2: Search the vector store
	searchStart := time.Now()
	docs, err := r.vectorStore.SimilaritySearch(ctx, embedding, r.config.TopK)
	if err != nil {
		span.RecordError(err)
		r.metrics.recordError(ctx, "similarity", "search_failed")
		return nil, fmt.Errorf("similarity search failed: %w", err)
	}

	searchTime := time.Since(searchStart)
	r.metrics.recordSearchLatency(ctx, "similarity", searchTime)

	// Step 3: Filter by minimum score and convert results
	results := make([]RetrievalResult, 0, len(docs))
	for _, doc := range docs {
		if doc.Score >= r.config.MinScore {
			results = append(results, RetrievalResult{
				ID:       doc.ID,
				Content:  doc.Content,
				Score:    doc.Score,
				Metadata: doc.Metadata,
				Strategy: StrategySimilarity,
			})
		}
	}

	span.SetAttributes(
		attribute.Int("results_count", len(results)),
		attribute.Float64("embed_latency_ms", float64(embedTime.Milliseconds())),
		attribute.Float64("search_latency_ms", float64(searchTime.Milliseconds())),
	)

	r.metrics.recordResults(ctx, "similarity", len(results))

	return results, nil
}

// KeywordSearch performs traditional keyword-based search.
func (r *AdvancedRetriever) KeywordSearch(
	ctx context.Context,
	query string,
) ([]RetrievalResult, error) {
	ctx, span := r.tracer.Start(ctx, "retriever.KeywordSearch",
		trace.WithAttributes(
			attribute.String("query", truncateQuery(query)),
			attribute.Int("top_k", r.config.TopK),
		),
	)
	defer span.End()

	start := time.Now()

	// Search using keyword store (BM25, Elasticsearch, etc.)
	keywordResults, err := r.keywordStore.Search(ctx, query, r.config.TopK)
	if err != nil {
		span.RecordError(err)
		r.metrics.recordError(ctx, "keyword", "search_failed")
		return nil, fmt.Errorf("keyword search failed: %w", err)
	}

	searchTime := time.Since(start)
	r.metrics.recordSearchLatency(ctx, "keyword", searchTime)

	// Convert to common result format
	results := make([]RetrievalResult, 0, len(keywordResults))
	for _, kr := range keywordResults {
		// Normalize BM25 scores to 0-1 range
		normalizedScore := normalizeScore(kr.Score)

		if normalizedScore >= r.config.MinScore {
			results = append(results, RetrievalResult{
				ID:       kr.ID,
				Content:  kr.Content,
				Score:    normalizedScore,
				Strategy: StrategyKeyword,
			})
		}
	}

	span.SetAttributes(
		attribute.Int("results_count", len(results)),
		attribute.Float64("search_latency_ms", float64(searchTime.Milliseconds())),
	)

	r.metrics.recordResults(ctx, "keyword", len(results))

	return results, nil
}

// HybridSearch combines similarity and keyword search using RRF.
func (r *AdvancedRetriever) HybridSearch(
	ctx context.Context,
	query string,
) ([]RetrievalResult, error) {
	ctx, span := r.tracer.Start(ctx, "retriever.HybridSearch",
		trace.WithAttributes(
			attribute.String("query", truncateQuery(query)),
			attribute.Float64("alpha", r.config.HybridAlpha),
		),
	)
	defer span.End()

	start := time.Now()

	// Run both searches in parallel
	var wg sync.WaitGroup
	var similarityResults, keywordResults []RetrievalResult
	var simErr, kwErr error

	wg.Add(2)

	go func() {
		defer wg.Done()
		similarityResults, simErr = r.SimilaritySearch(ctx, query)
	}()

	go func() {
		defer wg.Done()
		keywordResults, kwErr = r.KeywordSearch(ctx, query)
	}()

	wg.Wait()

	// Handle errors - if one fails, use the other
	if simErr != nil && kwErr != nil {
		span.RecordError(simErr)
		return nil, fmt.Errorf("both searches failed: similarity=%v, keyword=%v", simErr, kwErr)
	}

	if simErr != nil {
		// Fall back to keyword only
		span.SetAttributes(attribute.Bool("similarity_fallback", false))
		return keywordResults, nil
	}

	if kwErr != nil {
		// Fall back to similarity only
		span.SetAttributes(attribute.Bool("keyword_fallback", false))
		return similarityResults, nil
	}

	// Merge results using Reciprocal Rank Fusion (RRF)
	merged := r.reciprocalRankFusion(similarityResults, keywordResults)

	// Limit to TopK
	if len(merged) > r.config.TopK {
		merged = merged[:r.config.TopK]
	}

	totalTime := time.Since(start)
	span.SetAttributes(
		attribute.Int("similarity_count", len(similarityResults)),
		attribute.Int("keyword_count", len(keywordResults)),
		attribute.Int("merged_count", len(merged)),
		attribute.Float64("total_latency_ms", float64(totalTime.Milliseconds())),
	)

	r.metrics.recordSearchLatency(ctx, "hybrid", totalTime)
	r.metrics.recordResults(ctx, "hybrid", len(merged))

	return merged, nil
}

// MultiStrategySearch analyzes the query and uses the best strategy.
func (r *AdvancedRetriever) MultiStrategySearch(
	ctx context.Context,
	query string,
) ([]RetrievalResult, error) {
	ctx, span := r.tracer.Start(ctx, "retriever.MultiStrategySearch")
	defer span.End()

	// Step 1: Analyze query type
	queryType := r.analyzeQuery(query)
	span.SetAttributes(attribute.String("query_type", string(queryType)))

	// Step 2: Select strategy based on query type
	var results []RetrievalResult
	var err error

	switch queryType {
	case QueryTypeDefinition:
		results, err = r.SimilaritySearch(ctx, query)
	case QueryTypeExactMatch:
		results, err = r.KeywordSearch(ctx, query)
	case QueryTypeHowTo, QueryTypeComparison:
		results, err = r.HybridSearch(ctx, query)
	default:
		results, err = r.Retrieve(ctx, query, r.config.DefaultStrategy)
	}

	// Step 3: Apply fallback if enabled and results are poor
	if r.config.FallbackEnabled && (err != nil || len(results) < 3) {
		span.SetAttributes(attribute.Bool("fallback_triggered", true))

		fallbackResults, fallbackErr := r.HybridSearch(ctx, query)
		if fallbackErr == nil && len(fallbackResults) > len(results) {
			results = fallbackResults
			err = nil
		}
	}

	span.SetAttributes(attribute.Int("results_count", len(results)))

	return results, err
}

// analyzeQuery determines the type of query for strategy selection.
func (r *AdvancedRetriever) analyzeQuery(query string) QueryType {
	q := strings.ToLower(query)

	// Check for definition patterns
	if strings.HasPrefix(q, "what is ") ||
		strings.HasPrefix(q, "what are ") ||
		strings.HasPrefix(q, "define ") {
		return QueryTypeDefinition
	}

	// Check for how-to patterns
	if strings.HasPrefix(q, "how to ") ||
		strings.HasPrefix(q, "how do ") ||
		strings.HasPrefix(q, "how can ") ||
		strings.Contains(q, "tutorial") ||
		strings.Contains(q, "guide") {
		return QueryTypeHowTo
	}

	// Check for comparison patterns
	if strings.Contains(q, " vs ") ||
		strings.Contains(q, " versus ") ||
		strings.Contains(q, "compare") ||
		strings.Contains(q, "difference between") {
		return QueryTypeComparison
	}

	// Check for exact match patterns
	if containsID(q) {
		return QueryTypeExactMatch
	}

	return QueryTypeGeneral
}

// reciprocalRankFusion merges ranked lists using RRF algorithm.
func (r *AdvancedRetriever) reciprocalRankFusion(
	lists ...[]RetrievalResult,
) []RetrievalResult {
	const k = 60 // RRF constant

	scores := make(map[string]float64)
	results := make(map[string]RetrievalResult)

	for _, list := range lists {
		for rank, result := range list {
			rrf := 1.0 / float64(k+rank+1)
			scores[result.ID] += rrf

			if existing, ok := results[result.ID]; !ok || result.Score > existing.Score {
				results[result.ID] = result
			}
		}
	}

	merged := make([]RetrievalResult, 0, len(results))
	for id, result := range results {
		result.Score = scores[id]
		result.Strategy = StrategyHybrid
		merged = append(merged, result)
	}

	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Score > merged[j].Score
	})

	return merged
}

// ============================================================================
// Helper Functions
// ============================================================================

func truncateQuery(query string) string {
	if len(query) > 100 {
		return query[:100] + "..."
	}
	return query
}

func normalizeScore(score float64) float64 {
	// Sigmoid normalization for BM25 scores
	return 1.0 / (1.0 + math.Exp(-score/10.0))
}

func containsID(query string) bool {
	patterns := []string{
		`[A-Z]+-\d+`,              // JIRA-style
		`[0-9a-f]{8}-[0-9a-f]{4}`, // UUID prefix
	}
	for _, p := range patterns {
		if matched, _ := regexp.MatchString(p, query); matched {
			return true
		}
	}
	return false
}

// ============================================================================
// Metrics
// ============================================================================

// RetrieverMetrics provides OTEL instrumentation.
type RetrieverMetrics struct {
	tracer       trace.Tracer
	meter        metric.Meter
	searchLat    metric.Float64Histogram
	embedLat     metric.Float64Histogram
	resultsCount metric.Int64Counter
	errors       metric.Int64Counter
}

func newRetrieverMetrics() (*RetrieverMetrics, error) {
	meter := otel.Meter("beluga.retrieval")

	searchLat, err := meter.Float64Histogram(
		"beluga.retrieval.search_latency_seconds",
		metric.WithDescription("Search latency by strategy"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.0),
	)
	if err != nil {
		return nil, err
	}

	embedLat, err := meter.Float64Histogram(
		"beluga.retrieval.embedding_latency_seconds",
		metric.WithDescription("Embedding generation latency"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.01, 0.05, 0.1, 0.25, 0.5, 1.0),
	)
	if err != nil {
		return nil, err
	}

	resultsCount, err := meter.Int64Counter(
		"beluga.retrieval.results_total",
		metric.WithDescription("Total results returned"),
	)
	if err != nil {
		return nil, err
	}

	errors, err := meter.Int64Counter(
		"beluga.retrieval.errors_total",
		metric.WithDescription("Total retrieval errors"),
	)
	if err != nil {
		return nil, err
	}

	return &RetrieverMetrics{
		tracer:       otel.Tracer("beluga.retrieval"),
		meter:        meter,
		searchLat:    searchLat,
		embedLat:     embedLat,
		resultsCount: resultsCount,
		errors:       errors,
	}, nil
}

func (m *RetrieverMetrics) recordSearchLatency(ctx context.Context, strategy string, d time.Duration) {
	m.searchLat.Record(ctx, d.Seconds(),
		metric.WithAttributes(attribute.String("strategy", strategy)),
	)
}

func (m *RetrieverMetrics) recordEmbeddingLatency(ctx context.Context, d time.Duration) {
	m.embedLat.Record(ctx, d.Seconds())
}

func (m *RetrieverMetrics) recordResults(ctx context.Context, strategy string, count int) {
	m.resultsCount.Add(ctx, int64(count),
		metric.WithAttributes(attribute.String("strategy", strategy)),
	)
}

func (m *RetrieverMetrics) recordError(ctx context.Context, strategy, errorType string) {
	m.errors.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("strategy", strategy),
			attribute.String("error_type", errorType),
		),
	)
}

// ============================================================================
// Mock Implementations (for demonstration)
// ============================================================================

type mockVectorStore struct {
	docs []VectorResult
}

func (m *mockVectorStore) SimilaritySearch(ctx context.Context, embedding []float64, topK int) ([]VectorResult, error) {
	// Return mock results
	results := m.docs
	if len(results) > topK {
		results = results[:topK]
	}
	return results, nil
}

func (m *mockVectorStore) Add(ctx context.Context, id string, embedding []float64, content string, metadata map[string]interface{}) error {
	m.docs = append(m.docs, VectorResult{
		ID:       id,
		Content:  content,
		Score:    0.9,
		Metadata: metadata,
	})
	return nil
}

type mockEmbedder struct{}

func (m *mockEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	// Return a mock embedding
	return make([]float64, 1536), nil
}

type mockKeywordStore struct {
	docs []KeywordResult
}

func (m *mockKeywordStore) Search(ctx context.Context, query string, topK int) ([]KeywordResult, error) {
	results := m.docs
	if len(results) > topK {
		results = results[:topK]
	}
	return results, nil
}

func (m *mockKeywordStore) Index(ctx context.Context, id string, content string) error {
	m.docs = append(m.docs, KeywordResult{
		ID:      id,
		Content: content,
		Score:   15.0, // BM25-like score
	})
	return nil
}

// ============================================================================
// Example Usage
// ============================================================================

func main() {
	ctx := context.Background()

	// Create mock stores for demonstration
	vectorStore := &mockVectorStore{
		docs: []VectorResult{
			{ID: "1", Content: "Error handling in Go", Score: 0.92},
			{ID: "2", Content: "Best practices for API design", Score: 0.85},
			{ID: "3", Content: "Debugging techniques", Score: 0.78},
		},
	}

	keywordStore := &mockKeywordStore{
		docs: []KeywordResult{
			{ID: "4", Content: "Go error handling tutorial", Score: 18.5},
			{ID: "1", Content: "Error handling in Go", Score: 15.2},
			{ID: "5", Content: "Exception patterns", Score: 12.0},
		},
	}

	embedder := &mockEmbedder{}

	// Create the advanced retriever
	retriever, err := NewAdvancedRetriever(
		vectorStore,
		embedder,
		keywordStore,
		WithDefaultStrategy(StrategyHybrid),
		WithTopK(5),
		WithMinScore(0.7),
		WithHybridAlpha(0.5),
	)
	if err != nil {
		log.Fatalf("Failed to create retriever: %v", err)
	}

	// Example 1: Similarity search
	log.Println("=== Similarity Search ===")
	results, err := retriever.SimilaritySearch(ctx, "How do I handle errors?")
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		for _, r := range results {
			log.Printf("  [%.2f] %s: %s", r.Score, r.ID, r.Content)
		}
	}

	// Example 2: Keyword search
	log.Println("\n=== Keyword Search ===")
	results, err = retriever.KeywordSearch(ctx, "error handling tutorial")
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		for _, r := range results {
			log.Printf("  [%.2f] %s: %s", r.Score, r.ID, r.Content)
		}
	}

	// Example 3: Hybrid search
	log.Println("\n=== Hybrid Search ===")
	results, err = retriever.HybridSearch(ctx, "error handling best practices")
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		for _, r := range results {
			log.Printf("  [%.2f] %s: %s", r.Score, r.ID, r.Content)
		}
	}

	// Example 4: Multi-strategy search
	log.Println("\n=== Multi-Strategy Search ===")
	queries := []string{
		"What is RAG?",              // Definition → Similarity
		"How to implement caching?", // How-to → Hybrid
		"Redis vs Memcached",        // Comparison → Hybrid
		"JIRA-12345",                // Exact match → Keyword
	}

	for _, q := range queries {
		queryType := retriever.analyzeQuery(q)
		log.Printf("  Query: %q → Type: %s", q, queryType)
	}

	log.Println("\nAdvanced retrieval demo complete!")
}
