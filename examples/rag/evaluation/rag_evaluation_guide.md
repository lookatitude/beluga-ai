# RAG Evaluation Guide

> **Learn how to measure and improve your RAG pipeline's retrieval quality with precision, recall, and MRR metrics.**

## Introduction

Building a RAG system is straightforward. Building a *good* RAG system requires measurement. Without proper evaluation, you're flying blind—you don't know if your retrieval is actually returning relevant documents, or if your changes are making things better or worse.

In this guide, you'll learn:

- How to measure retrieval quality with standard IR metrics
- How to create evaluation datasets from your domain
- How to run automated benchmarks
- How to interpret results and identify improvements
- How to track retrieval quality over time with OTEL

## Prerequisites

| Requirement | Why You Need It |
|-------------|-----------------|
| **Go 1.24+** | Required for Beluga AI framework |
| **Working RAG pipeline** | Something to evaluate |
| **Evaluation dataset** | Ground truth for measurement |
| **Understanding of retrieval** | Basic concepts covered in retrieval guide |

## Concepts

### Why Evaluation Matters

RAG systems can fail silently. The LLM will still generate fluent responses even if the retrieved context is wrong. Only by measuring do you know if:

1. **Retrieval is working**: Are you finding the right documents?
2. **Ranking is correct**: Are the best documents at the top?
3. **Coverage is adequate**: Are you missing important documents?
4. **Changes help or hurt**: Did that new embedding model actually improve things?

### Core Metrics

#### Precision@K

Precision@K measures what fraction of your top K results are relevant:

```
Precision@K = (Relevant documents in top K) / K

Example:
  Query: "How do I handle errors in Go?"
  Top 3 results: [Relevant, Not Relevant, Relevant]
  Precision@3 = 2/3 = 0.67
```

**Use when**: You care about the quality of results shown to users.

#### Recall@K

Recall@K measures what fraction of all relevant documents you retrieved:

```
Recall@K = (Relevant documents in top K) / (Total relevant documents)

Example:
  Query: "How do I handle errors in Go?"
  Total relevant documents: 5
  Relevant in top 3: 2
  Recall@3 = 2/5 = 0.40
```

**Use when**: You need comprehensive coverage (e.g., legal or medical search).

#### Mean Reciprocal Rank (MRR)

MRR measures how early the first relevant result appears:

```
MRR = 1 / (rank of first relevant result)

Example:
  Query 1: First relevant at position 1 → RR = 1.0
  Query 2: First relevant at position 3 → RR = 0.33
  Query 3: First relevant at position 2 → RR = 0.5
  MRR = (1.0 + 0.33 + 0.5) / 3 = 0.61
```

**Use when**: Users typically only look at the first result.

#### Normalized Discounted Cumulative Gain (NDCG)

NDCG accounts for both relevance and position, penalizing relevant results that appear lower:

```
DCG@K = Σ (relevance_i / log2(i + 1)) for i = 1 to K
NDCG@K = DCG@K / Ideal DCG@K

Example with graded relevance (0-3):
  Results: [3, 2, 1, 0, 2]
  DCG@5 = 3/1 + 2/1.58 + 1/2 + 0/2.32 + 2/2.58 = 5.04
  Ideal = [3, 2, 2, 1, 0] → IDCG@5 = 6.15
  NDCG@5 = 5.04 / 6.15 = 0.82
```

**Use when**: You have graded relevance judgments (not just binary).

### Evaluation Dataset Structure

An evaluation dataset consists of queries with their expected relevant documents:

```go
type EvaluationDataset struct {
    Queries []EvaluationQuery `json:"queries"`
}

type EvaluationQuery struct {
    ID              string   `json:"id"`
    Query           string   `json:"query"`
    RelevantDocIDs  []string `json:"relevant_doc_ids"`
    GradedRelevance map[string]int `json:"graded_relevance,omitempty"` // doc_id → score
}
```

## Step-by-Step Tutorial

### Step 1: Define the Evaluation Framework

```go
package evaluation

import (
    "context"
    "encoding/json"
    "fmt"
    "math"
    "os"
    "sort"
    "time"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
    "go.opentelemetry.io/otel/trace"
)

// RAGEvaluator evaluates retrieval quality.
type RAGEvaluator struct {
    retriever Retriever
    metrics   *EvaluationMetrics
    tracer    trace.Tracer
}

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

// EvaluationQuery represents a query with ground truth.
type EvaluationQuery struct {
    ID              string
    Query           string
    RelevantDocIDs  []string
    GradedRelevance map[string]int // Optional: doc_id → relevance score
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
    QueriesWithPerfect int
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
```

### Step 2: Implement Metric Calculations

```go
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
    relevantPositions := []int{}
    for i, r := range results {
        if relevantSet[r.ID] {
            result.RelevantFound++
            relevantPositions = append(relevantPositions, i+1) // 1-indexed

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
```

### Step 3: Run Batch Evaluation

```go
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
            // Log error but continue
            continue
        }
        results = append(results, *result)
    }

    // Calculate aggregate metrics
    agg := e.aggregateResults(results)

    e.metrics.recordDatasetEvaluation(ctx, agg)

    span.SetAttributes(
        attribute.Float64("mean_precision_at_5", agg.MeanPrecisionAt5),
        attribute.Float64("mean_recall_at_5", agg.MeanRecallAt5),
        attribute.Float64("mean_mrr", agg.MeanMRR),
        attribute.Int("queries_evaluated", agg.TotalQueries),
    )

    return agg, results, nil
}

func (e *RAGEvaluator) aggregateResults(results []EvaluationResult) *AggregateResults {
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
```

### Step 4: Create Evaluation Datasets

```go
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

// GenerateSyntheticDataset creates a test dataset from your documents.
// This is useful for initial testing before you have real evaluation data.
func GenerateSyntheticDataset(documents []Document, queriesPerDoc int) []EvaluationQuery {
    queries := make([]EvaluationQuery, 0, len(documents)*queriesPerDoc)

    for _, doc := range documents {
        // Generate queries from document content
        // In practice, you'd use an LLM to generate realistic queries
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
    // Simple heuristic: use first N sentences as queries
    // In production, use an LLM to generate realistic queries
    sentences := splitSentences(doc.Content)
    
    queries := make([]string, 0, count)
    for i := 0; i < count && i < len(sentences); i++ {
        queries = append(queries, sentences[i])
    }
    
    return queries
}
```

### Step 5: Add OTEL Metrics

```go
// EvaluationMetrics provides OTEL instrumentation for evaluation.
type EvaluationMetrics struct {
    tracer           trace.Tracer
    meter            metric.Meter
    precisionGauge   metric.Float64ObservableGauge
    recallGauge      metric.Float64ObservableGauge
    mrrGauge         metric.Float64ObservableGauge
    queryLatency     metric.Float64Histogram
    evaluationsTotal metric.Int64Counter

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

    // Register observable gauges for tracking latest metrics
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
```

## Code Example

See the complete implementation:

- [rag_evaluation.go](./rag_evaluation.go) - Full implementation
- [rag_evaluation_test.go](./rag_evaluation_test.go) - Test suite

## Testing

### Table-Driven Tests for Metrics

```go
func TestPrecisionAtK(t *testing.T) {
    relevantSet := map[string]bool{"a": true, "b": true, "c": true}

    tests := []struct {
        name     string
        results  []RetrievalResult
        k        int
        expected float64
    }{
        {
            name: "all relevant",
            results: []RetrievalResult{
                {ID: "a"}, {ID: "b"}, {ID: "c"},
            },
            k:        3,
            expected: 1.0,
        },
        {
            name: "none relevant",
            results: []RetrievalResult{
                {ID: "x"}, {ID: "y"}, {ID: "z"},
            },
            k:        3,
            expected: 0.0,
        },
        {
            name: "mixed",
            results: []RetrievalResult{
                {ID: "a"}, {ID: "x"}, {ID: "b"},
            },
            k:        3,
            expected: 2.0 / 3.0,
        },
        {
            name: "k larger than results",
            results: []RetrievalResult{
                {ID: "a"}, {ID: "b"},
            },
            k:        5,
            expected: 1.0,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := precisionAtK(tt.results, relevantSet, tt.k)
            assert.InDelta(t, tt.expected, result, 0.001)
        })
    }
}
```

### Benchmark Evaluation

```go
func BenchmarkEvaluateQuery(b *testing.B) {
    evaluator := setupBenchEvaluator(b)
    query := EvaluationQuery{
        ID:             "bench",
        Query:          "test query",
        RelevantDocIDs: []string{"1", "2", "3"},
    }
    ctx := context.Background()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = evaluator.EvaluateQuery(ctx, query, 10)
    }
}
```

## Best Practices

### 1. Create Representative Evaluation Sets

```go
// Good: Diverse queries covering different types
queries := []EvaluationQuery{
    {Query: "What is X?", ...},           // Definition
    {Query: "How do I Y?", ...},          // How-to
    {Query: "Error when Z", ...},         // Troubleshooting
    {Query: "Compare A and B", ...},      // Comparison
}

// Bad: Repetitive or biased queries
queries := []EvaluationQuery{
    {Query: "What is X?", ...},
    {Query: "What is Y?", ...},
    {Query: "What is Z?", ...},  // All same type!
}
```

### 2. Use Graded Relevance When Possible

```go
// Binary relevance: less informative
RelevantDocIDs: []string{"1", "2", "3"}

// Graded relevance: more nuanced
GradedRelevance: map[string]int{
    "1": 3,  // Highly relevant
    "2": 2,  // Relevant
    "3": 1,  // Marginally relevant
}
```

### 3. Track Metrics Over Time

```go
// Run evaluation as part of CI/CD
func RunCIEvaluation() {
    evaluator := setupEvaluator()
    results, _ := evaluator.EvaluateDataset(ctx, testQueries, 10)
    
    // Fail if metrics drop below threshold
    if results.MeanMRR < 0.7 {
        log.Fatal("MRR dropped below threshold!")
    }
}
```

### 4. Analyze Failure Cases

```go
// Find queries with poor performance
for _, r := range results {
    if r.MRR < 0.3 {
        log.Printf("Low MRR query: %s (MRR=%.2f)", r.Query, r.MRR)
        // Investigate why this query fails
    }
}
```

## Troubleshooting

### Q: All my metrics are 0
**A:** Check if your evaluation dataset has the correct document IDs. The IDs must match exactly with what your retriever returns.

### Q: Precision is high but recall is low
**A:** Your retriever is accurate but not finding all relevant documents. Consider increasing TopK or improving your embedding model's coverage.

### Q: MRR is low despite good precision
**A:** Your relevant documents are being ranked low. Check your ranking algorithm or consider adding re-ranking.

### Q: Evaluation is too slow
**A:** Consider sampling your evaluation dataset, or running evaluation in parallel. Cache embeddings if possible.

## Related Resources

- **[Advanced Retrieval Guide](../../vectorstores/advanced_retrieval/advanced_retrieval_guide.md)**: Retrieval strategies
- **[RAG Strategies Use Case](../../docs/use-cases/rag-strategies.md)**: When to use which approach
- **[Multimodal RAG Guide](../../docs/guides/rag-multimodal.md)**: RAG with images
- **[Observability Tracing](../../docs/guides/observability-tracing.md)**: Monitoring your RAG
