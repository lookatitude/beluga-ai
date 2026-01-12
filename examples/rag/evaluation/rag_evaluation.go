// Package evaluation demonstrates how to measure and benchmark
// RAG pipeline quality using standard IR metrics.
//
// This example shows you how to evaluate retrieval with precision,
// recall, MRR, and NDCG, helping you understand and improve your
// RAG system's performance.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// ============================================================================
// Types
// ============================================================================

// Retriever interface for the system under test.
type Retriever interface {
	Retrieve(ctx context.Context, query string, topK int) ([]RetrievalResult, error)
}

// RetrievalResult from the retriever.
type RetrievalResult struct {
	ID      string
	Content string
	Score   float64
}

// Document represents a document in the corpus.
type Document struct {
	ID      string
	Content string
}

// EvaluationQuery represents a query with ground truth.
type EvaluationQuery struct {
	ID              string
	Query           string
	RelevantDocIDs  []string
	GradedRelevance map[string]int // Optional: doc_id → relevance score (0-3)
}

// EvaluationResult contains metrics for a single query.
type EvaluationResult struct {
	QueryID          string
	Query            string
	PrecisionAt1     float64
	PrecisionAt5     float64
	PrecisionAt10    float64
	RecallAt5        float64
	RecallAt10       float64
	MRR              float64
	NDCG             float64
	RetrievedCount   int
	RelevantFound    int
	FirstRelevantPos int
	Latency          time.Duration
}

// AggregateResults contains summary metrics across all queries.
type AggregateResults struct {
	TotalQueries       int
	MeanPrecisionAt1   float64
	MeanPrecisionAt5   float64
	MeanPrecisionAt10  float64
	MeanRecallAt5      float64
	MeanRecallAt10     float64
	MeanMRR            float64
	MeanNDCG           float64
	MeanLatency        time.Duration
	QueriesWithNoHits  int
	QueriesWithPerfect int // P@1 = 1.0
}

// ============================================================================
// RAG Evaluator
// ============================================================================

// RAGEvaluator evaluates retrieval quality.
type RAGEvaluator struct {
	retriever Retriever
	metrics   *EvaluationMetrics
	tracer    trace.Tracer
}

// NewRAGEvaluator creates a new evaluator.
func NewRAGEvaluator(retriever Retriever) (*RAGEvaluator, error) {
	metrics, err := newEvaluationMetrics()
	if err != nil {
		return nil, err
	}

	return &RAGEvaluator{
		retriever: retriever,
		metrics:   metrics,
		tracer:    otel.Tracer("beluga.rag.evaluation"),
	}, nil
}

// EvaluateQuery evaluates a single query.
func (e *RAGEvaluator) EvaluateQuery(
	ctx context.Context,
	query EvaluationQuery,
	topK int,
) (*EvaluationResult, error) {
	ctx, span := e.tracer.Start(ctx, "evaluator.EvaluateQuery",
		trace.WithAttributes(
			attribute.String("query_id", query.ID),
			attribute.Int("relevant_count", len(query.RelevantDocIDs)),
		),
	)
	defer span.End()

	start := time.Now()

	// Retrieve results
	results, err := e.retriever.Retrieve(ctx, query.Query, topK)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("retrieval failed: %w", err)
	}

	latency := time.Since(start)

	// Build relevance set for O(1) lookup
	relevantSet := make(map[string]bool)
	for _, id := range query.RelevantDocIDs {
		relevantSet[id] = true
	}

	// Calculate metrics
	result := &EvaluationResult{
		QueryID:          query.ID,
		Query:            query.Query,
		RetrievedCount:   len(results),
		FirstRelevantPos: -1,
		Latency:          latency,
	}

	// Count relevant documents and find positions
	for i, r := range results {
		if relevantSet[r.ID] {
			result.RelevantFound++
			if result.FirstRelevantPos == -1 {
				result.FirstRelevantPos = i + 1
			}
		}
	}

	// Calculate Precision@K
	result.PrecisionAt1 = precisionAtK(results, relevantSet, 1)
	result.PrecisionAt5 = precisionAtK(results, relevantSet, 5)
	result.PrecisionAt10 = precisionAtK(results, relevantSet, 10)

	// Calculate Recall@K
	result.RecallAt5 = recallAtK(results, relevantSet, len(query.RelevantDocIDs), 5)
	result.RecallAt10 = recallAtK(results, relevantSet, len(query.RelevantDocIDs), 10)

	// Calculate MRR
	if result.FirstRelevantPos > 0 {
		result.MRR = 1.0 / float64(result.FirstRelevantPos)
	}

	// Calculate NDCG
	if query.GradedRelevance != nil {
		result.NDCG = calculateNDCG(results, query.GradedRelevance, topK)
	} else {
		// Binary relevance: treat all relevant as score 1
		binaryGraded := make(map[string]int)
		for _, id := range query.RelevantDocIDs {
			binaryGraded[id] = 1
		}
		result.NDCG = calculateNDCG(results, binaryGraded, topK)
	}

	// Record metrics
	e.metrics.recordQueryEvaluation(ctx, result)

	span.SetAttributes(
		attribute.Float64("precision_at_5", result.PrecisionAt5),
		attribute.Float64("recall_at_5", result.RecallAt5),
		attribute.Float64("mrr", result.MRR),
		attribute.Float64("ndcg", result.NDCG),
	)

	return result, nil
}

// EvaluateDataset runs evaluation on a complete dataset.
func (e *RAGEvaluator) EvaluateDataset(
	ctx context.Context,
	queries []EvaluationQuery,
	topK int,
) (*AggregateResults, []EvaluationResult, error) {
	ctx, span := e.tracer.Start(ctx, "evaluator.EvaluateDataset",
		trace.WithAttributes(
			attribute.Int("query_count", len(queries)),
			attribute.Int("top_k", topK),
		),
	)
	defer span.End()

	results := make([]EvaluationResult, 0, len(queries))

	for _, query := range queries {
		result, err := e.EvaluateQuery(ctx, query, topK)
		if err != nil {
			log.Printf("Warning: query %s failed: %v", query.ID, err)
			continue
		}
		results = append(results, *result)
	}

	// Calculate aggregate metrics
	agg := aggregateResults(results)

	e.metrics.recordDatasetEvaluation(ctx, agg)

	span.SetAttributes(
		attribute.Float64("mean_precision_at_5", agg.MeanPrecisionAt5),
		attribute.Float64("mean_recall_at_5", agg.MeanRecallAt5),
		attribute.Float64("mean_mrr", agg.MeanMRR),
		attribute.Int("queries_evaluated", agg.TotalQueries),
	)

	return agg, results, nil
}

// ============================================================================
// Metric Calculations
// ============================================================================

func precisionAtK(results []RetrievalResult, relevantSet map[string]bool, k int) float64 {
	if k > len(results) {
		k = len(results)
	}
	if k == 0 {
		return 0
	}

	relevant := 0
	for i := 0; i < k; i++ {
		if relevantSet[results[i].ID] {
			relevant++
		}
	}
	return float64(relevant) / float64(k)
}

func recallAtK(results []RetrievalResult, relevantSet map[string]bool, totalRelevant, k int) float64 {
	if totalRelevant == 0 {
		return 0
	}
	if k > len(results) {
		k = len(results)
	}

	found := 0
	for i := 0; i < k; i++ {
		if relevantSet[results[i].ID] {
			found++
		}
	}
	return float64(found) / float64(totalRelevant)
}

func calculateNDCG(results []RetrievalResult, relevance map[string]int, k int) float64 {
	if k > len(results) {
		k = len(results)
	}
	if k == 0 {
		return 0
	}

	// Calculate DCG
	dcg := 0.0
	for i := 0; i < k; i++ {
		rel := float64(relevance[results[i].ID])
		dcg += rel / math.Log2(float64(i+2)) // +2 because log2(1) = 0
	}

	// Calculate ideal DCG
	scores := make([]int, 0, len(relevance))
	for _, score := range relevance {
		scores = append(scores, score)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(scores)))

	idcg := 0.0
	for i := 0; i < k && i < len(scores); i++ {
		idcg += float64(scores[i]) / math.Log2(float64(i+2))
	}

	if idcg == 0 {
		return 0
	}
	return dcg / idcg
}

func aggregateResults(results []EvaluationResult) *AggregateResults {
	if len(results) == 0 {
		return &AggregateResults{}
	}

	agg := &AggregateResults{
		TotalQueries: len(results),
	}

	var totalLatency time.Duration

	for _, r := range results {
		agg.MeanPrecisionAt1 += r.PrecisionAt1
		agg.MeanPrecisionAt5 += r.PrecisionAt5
		agg.MeanPrecisionAt10 += r.PrecisionAt10
		agg.MeanRecallAt5 += r.RecallAt5
		agg.MeanRecallAt10 += r.RecallAt10
		agg.MeanMRR += r.MRR
		agg.MeanNDCG += r.NDCG
		totalLatency += r.Latency

		if r.RelevantFound == 0 {
			agg.QueriesWithNoHits++
		}
		if r.PrecisionAt1 == 1.0 {
			agg.QueriesWithPerfect++
		}
	}

	n := float64(len(results))
	agg.MeanPrecisionAt1 /= n
	agg.MeanPrecisionAt5 /= n
	agg.MeanPrecisionAt10 /= n
	agg.MeanRecallAt5 /= n
	agg.MeanRecallAt10 /= n
	agg.MeanMRR /= n
	agg.MeanNDCG /= n
	agg.MeanLatency = totalLatency / time.Duration(len(results))

	return agg
}

// ============================================================================
// Dataset Loading
// ============================================================================

// LoadDatasetFromJSON loads an evaluation dataset from a JSON file.
func LoadDatasetFromJSON(path string) ([]EvaluationQuery, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var dataset struct {
		Queries []struct {
			ID              string         `json:"id"`
			Query           string         `json:"query"`
			RelevantDocIDs  []string       `json:"relevant_doc_ids"`
			GradedRelevance map[string]int `json:"graded_relevance"`
		} `json:"queries"`
	}

	if err := json.Unmarshal(data, &dataset); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	queries := make([]EvaluationQuery, len(dataset.Queries))
	for i, q := range dataset.Queries {
		queries[i] = EvaluationQuery{
			ID:              q.ID,
			Query:           q.Query,
			RelevantDocIDs:  q.RelevantDocIDs,
			GradedRelevance: q.GradedRelevance,
		}
	}

	return queries, nil
}

// GenerateSyntheticDataset creates a test dataset from documents.
func GenerateSyntheticDataset(documents []Document, queriesPerDoc int) []EvaluationQuery {
	queries := make([]EvaluationQuery, 0, len(documents)*queriesPerDoc)

	for _, doc := range documents {
		syntheticQueries := generateQueriesFromDoc(doc, queriesPerDoc)

		for i, q := range syntheticQueries {
			queries = append(queries, EvaluationQuery{
				ID:             fmt.Sprintf("%s_q%d", doc.ID, i),
				Query:          q,
				RelevantDocIDs: []string{doc.ID},
			})
		}
	}

	return queries
}

func generateQueriesFromDoc(doc Document, count int) []string {
	sentences := splitSentences(doc.Content)

	queries := make([]string, 0, count)
	for i := 0; i < count && i < len(sentences); i++ {
		queries = append(queries, sentences[i])
	}

	return queries
}

func splitSentences(text string) []string {
	// Simple sentence splitting
	text = strings.ReplaceAll(text, ".", ".\n")
	text = strings.ReplaceAll(text, "!", "!\n")
	text = strings.ReplaceAll(text, "?", "?\n")

	lines := strings.Split(text, "\n")
	sentences := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 10 { // Skip very short lines
			sentences = append(sentences, line)
		}
	}
	return sentences
}

// ============================================================================
// OTEL Metrics
// ============================================================================

// EvaluationMetrics provides OTEL instrumentation for evaluation.
type EvaluationMetrics struct {
	tracer           trace.Tracer
	meter            metric.Meter
	queryLatency     metric.Float64Histogram
	evaluationsTotal metric.Int64Counter
	precisionGauge   metric.Float64ObservableGauge
	recallGauge      metric.Float64ObservableGauge
	mrrGauge         metric.Float64ObservableGauge

	// Latest values for observable gauges
	latestPrecision float64
	latestRecall    float64
	latestMRR       float64
}

func newEvaluationMetrics() (*EvaluationMetrics, error) {
	meter := otel.Meter("beluga.rag.evaluation")

	queryLatency, err := meter.Float64Histogram(
		"beluga.rag.evaluation_latency_seconds",
		metric.WithDescription("Latency of individual query evaluation"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.0),
	)
	if err != nil {
		return nil, err
	}

	evaluationsTotal, err := meter.Int64Counter(
		"beluga.rag.evaluations_total",
		metric.WithDescription("Total evaluation runs"),
	)
	if err != nil {
		return nil, err
	}

	m := &EvaluationMetrics{
		tracer:           otel.Tracer("beluga.rag.evaluation"),
		meter:            meter,
		queryLatency:     queryLatency,
		evaluationsTotal: evaluationsTotal,
	}

	// Register observable gauges
	m.precisionGauge, _ = meter.Float64ObservableGauge(
		"beluga.rag.precision_at_5",
		metric.WithDescription("Latest Precision@5 score"),
		metric.WithFloat64Callback(func(_ context.Context, o metric.Float64Observer) error {
			o.Observe(m.latestPrecision)
			return nil
		}),
	)

	m.recallGauge, _ = meter.Float64ObservableGauge(
		"beluga.rag.recall_at_5",
		metric.WithDescription("Latest Recall@5 score"),
		metric.WithFloat64Callback(func(_ context.Context, o metric.Float64Observer) error {
			o.Observe(m.latestRecall)
			return nil
		}),
	)

	m.mrrGauge, _ = meter.Float64ObservableGauge(
		"beluga.rag.mrr",
		metric.WithDescription("Latest MRR score"),
		metric.WithFloat64Callback(func(_ context.Context, o metric.Float64Observer) error {
			o.Observe(m.latestMRR)
			return nil
		}),
	)

	return m, nil
}

func (m *EvaluationMetrics) recordQueryEvaluation(ctx context.Context, result *EvaluationResult) {
	m.queryLatency.Record(ctx, result.Latency.Seconds())
}

func (m *EvaluationMetrics) recordDatasetEvaluation(ctx context.Context, agg *AggregateResults) {
	m.evaluationsTotal.Add(ctx, 1)
	m.latestPrecision = agg.MeanPrecisionAt5
	m.latestRecall = agg.MeanRecallAt5
	m.latestMRR = agg.MeanMRR
}

// ============================================================================
// Mock Retriever (for demonstration)
// ============================================================================

type mockRetriever struct {
	docs []RetrievalResult
}

func (m *mockRetriever) Retrieve(ctx context.Context, query string, topK int) ([]RetrievalResult, error) {
	results := m.docs
	if len(results) > topK {
		results = results[:topK]
	}
	return results, nil
}

// ============================================================================
// Example Usage
// ============================================================================

func main() {
	ctx := context.Background()

	// Create a mock retriever with some test documents
	retriever := &mockRetriever{
		docs: []RetrievalResult{
			{ID: "1", Content: "Error handling in Go", Score: 0.95},
			{ID: "2", Content: "Best practices for APIs", Score: 0.88},
			{ID: "3", Content: "Debugging techniques", Score: 0.82},
			{ID: "4", Content: "Testing strategies", Score: 0.78},
			{ID: "5", Content: "Performance optimization", Score: 0.72},
		},
	}

	// Create evaluator
	evaluator, err := NewRAGEvaluator(retriever)
	if err != nil {
		log.Fatalf("Failed to create evaluator: %v", err)
	}

	// Create evaluation dataset
	queries := []EvaluationQuery{
		{
			ID:             "q1",
			Query:          "How do I handle errors?",
			RelevantDocIDs: []string{"1", "3"},
		},
		{
			ID:             "q2",
			Query:          "API design patterns",
			RelevantDocIDs: []string{"2"},
		},
		{
			ID:             "q3",
			Query:          "How to write tests?",
			RelevantDocIDs: []string{"4"},
			GradedRelevance: map[string]int{
				"4": 3, // Highly relevant
				"3": 1, // Marginally relevant
			},
		},
	}

	// Run evaluation
	log.Println("=== Single Query Evaluation ===")
	result, err := evaluator.EvaluateQuery(ctx, queries[0], 5)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		log.Printf("Query: %s", result.Query)
		log.Printf("  Precision@5: %.2f", result.PrecisionAt5)
		log.Printf("  Recall@5: %.2f", result.RecallAt5)
		log.Printf("  MRR: %.2f", result.MRR)
		log.Printf("  NDCG: %.2f", result.NDCG)
		log.Printf("  Latency: %v", result.Latency)
	}

	// Run dataset evaluation
	log.Println("\n=== Dataset Evaluation ===")
	agg, results, err := evaluator.EvaluateDataset(ctx, queries, 5)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		log.Printf("Total Queries: %d", agg.TotalQueries)
		log.Printf("Mean Precision@5: %.2f", agg.MeanPrecisionAt5)
		log.Printf("Mean Recall@5: %.2f", agg.MeanRecallAt5)
		log.Printf("Mean MRR: %.2f", agg.MeanMRR)
		log.Printf("Mean NDCG: %.2f", agg.MeanNDCG)
		log.Printf("Mean Latency: %v", agg.MeanLatency)
		log.Printf("Queries with no hits: %d", agg.QueriesWithNoHits)
		log.Printf("Queries with perfect P@1: %d", agg.QueriesWithPerfect)

		log.Println("\nPer-Query Results:")
		for _, r := range results {
			log.Printf("  %s: P@5=%.2f, R@5=%.2f, MRR=%.2f",
				r.QueryID, r.PrecisionAt5, r.RecallAt5, r.MRR)
		}
	}

	log.Println("\nRAG Evaluation demo complete!")
}
