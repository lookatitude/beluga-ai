# Advanced Retrieval Strategies Guide

> **Learn how to implement similarity search, keyword search, hybrid search, and multi-strategy retrieval for optimal RAG performance.**

## Introduction

Retrieval is the foundation of RAG (Retrieval-Augmented Generation). The quality of your retrieval directly impacts the quality of your AI's responses. While basic vector similarity search works for many use cases, production systems often need more sophisticated approaches.

In this guide, you'll learn:

- How similarity search uses embeddings to find semantically related content
- How keyword search uses traditional text matching for exact terms
- How hybrid search combines both for the best of both worlds
- How to implement multi-strategy retrieval with fallbacks
- How to integrate text splitters for optimal chunking
- How to instrument retrieval with OTEL for optimization

## Prerequisites

| Requirement | Why You Need It |
|-------------|-----------------|
| **Go 1.24+** | Required for Beluga AI framework |
| **Vector store configured** | Qdrant, Pinecone, or PostgreSQL with pgvector |
| **Embeddings provider** | OpenAI, Voyage, or custom embeddings |
| **Understanding of embeddings** | How text becomes vectors |

## Concepts

### Similarity Search

Similarity search finds documents that are semantically similar to your query. It works by:

1. Converting your query text into an embedding (a vector of numbers)
2. Comparing this vector against all stored document vectors
3. Returning documents with the closest vectors (highest cosine similarity)

```
Query: "How do I handle errors in my API?"
       ↓ Embedding
Query Vector: [0.12, -0.45, 0.78, ...]
       ↓ Compare to stored vectors
Results:
  1. "Error handling best practices for REST APIs" (similarity: 0.89)
  2. "Building robust exception handling" (similarity: 0.85)
  3. "API design patterns" (similarity: 0.72)
```

**Strengths**: Great for semantic understanding, handles synonyms, works across languages
**Weaknesses**: May miss exact keyword matches, expensive (requires embedding)

### Keyword Search

Keyword search uses traditional text matching techniques like BM25 or TF-IDF to find documents containing specific terms.

```
Query: "OpenTelemetry OTEL tracing"
       ↓ Tokenize and match
Results:
  1. "OpenTelemetry Tracing Setup Guide" (contains: OpenTelemetry, OTEL, tracing)
  2. "Distributed Tracing with OTEL" (contains: OTEL, tracing)
  3. "Monitoring Best Practices" (contains: tracing)
```

**Strengths**: Fast, exact term matching, good for technical jargon and proper nouns
**Weaknesses**: No semantic understanding, misses synonyms

### Hybrid Search

Hybrid search combines similarity and keyword search to get the benefits of both:

```
Query: "OTEL distributed tracing setup"
       ↓ Run both searches
       
Similarity Results:           Keyword Results:
1. "Tracing fundamentals"     1. "OTEL tracing guide"
2. "Distributed systems"      2. "OTEL setup tutorial"
3. "Observability basics"     3. "Distributed tracing"

       ↓ Merge and re-rank (using RRF or other fusion)

Hybrid Results:
1. "OTEL tracing guide" (appeared in both, high combined score)
2. "OTEL setup tutorial" (keyword match + semantic relevance)
3. "Distributed tracing" (strong in both)
```

### Multi-Strategy Retrieval

Multi-strategy retrieval goes beyond hybrid by using different strategies based on query type:

```
┌─────────────────────────────────────────────────────────────┐
│                   Query Analyzer                            │
├─────────────────────────────────────────────────────────────┤
│  "What is X?" → Definition search (similarity)              │
│  "ACME-123"   → Exact match (keyword)                       │
│  "How to X?"  → Tutorial search (hybrid + filter)           │
│  Complex      → Multi-index search (parallel)               │
└─────────────────────────────────────────────────────────────┘
```

## Step-by-Step Tutorial

### Step 1: Set Up the Retriever

First, let's create a retriever that supports multiple strategies:

```go
package retrieval

import (
    "context"
    "fmt"
    "sort"
    "sync"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/vectorstores"
    "github.com/lookatitude/beluga-ai/pkg/embeddings"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

// RetrievalStrategy defines how to search for documents.
type RetrievalStrategy string

const (
    StrategySimilarity RetrievalStrategy = "similarity"
    StrategyKeyword    RetrievalStrategy = "keyword"
    StrategyHybrid     RetrievalStrategy = "hybrid"
    StrategyMulti      RetrievalStrategy = "multi"
)

// RetrieverConfig configures the advanced retriever.
type RetrieverConfig struct {
    // Default strategy to use
    DefaultStrategy RetrievalStrategy
    
    // Number of results to return
    TopK int
    
    // Score threshold (0.0 to 1.0)
    MinScore float64
    
    // Hybrid search settings
    HybridAlpha float64 // Weight for similarity (0.0 to 1.0)
    
    // Multi-strategy settings
    EnableQueryAnalysis bool
    FallbackEnabled     bool
}

// AdvancedRetriever implements multiple retrieval strategies.
type AdvancedRetriever struct {
    config      RetrieverConfig
    vectorStore vectorstores.VectorStore
    embedder    embeddings.Embedder
    keywordStore KeywordStore // BM25 or similar
    metrics     *RetrieverMetrics
    tracer      trace.Tracer
}

// KeywordStore interface for keyword-based search.
type KeywordStore interface {
    Search(ctx context.Context, query string, topK int) ([]KeywordResult, error)
}

type KeywordResult struct {
    ID      string
    Content string
    Score   float64
}

// NewAdvancedRetriever creates a retriever with multiple strategies.
func NewAdvancedRetriever(
    vectorStore vectorstores.VectorStore,
    embedder embeddings.Embedder,
    keywordStore KeywordStore,
    opts ...RetrieverOption,
) (*AdvancedRetriever, error) {
    config := RetrieverConfig{
        DefaultStrategy:     StrategyHybrid,
        TopK:                10,
        MinScore:            0.7,
        HybridAlpha:         0.5,
        EnableQueryAnalysis: true,
        FallbackEnabled:     true,
    }
    
    for _, opt := range opts {
        opt(&config)
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

// RetrieverOption configures the retriever.
type RetrieverOption func(*RetrieverConfig)

func WithDefaultStrategy(s RetrievalStrategy) RetrieverOption {
    return func(c *RetrieverConfig) { c.DefaultStrategy = s }
}

func WithTopK(k int) RetrieverOption {
    return func(c *RetrieverConfig) { c.TopK = k }
}

func WithMinScore(score float64) RetrieverOption {
    return func(c *RetrieverConfig) { c.MinScore = score }
}

func WithHybridAlpha(alpha float64) RetrieverOption {
    return func(c *RetrieverConfig) { c.HybridAlpha = alpha }
}
```

### Step 2: Implement Similarity Search

```go
// RetrievalResult represents a single retrieval result.
type RetrievalResult struct {
    ID       string
    Content  string
    Score    float64
    Metadata map[string]interface{}
    Strategy RetrievalStrategy
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
```

### Step 3: Implement Keyword Search

```go
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

// normalizeScore converts raw BM25 scores to 0-1 range.
func normalizeScore(score float64) float64 {
    // BM25 scores are unbounded; use sigmoid normalization
    return 1.0 / (1.0 + math.Exp(-score/10.0))
}
```

### Step 4: Implement Hybrid Search

```go
// HybridSearch combines similarity and keyword search.
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

// reciprocalRankFusion merges ranked lists using RRF algorithm.
// RRF is robust to outliers and doesn't require score normalization.
func (r *AdvancedRetriever) reciprocalRankFusion(
    lists ...[]RetrievalResult,
) []RetrievalResult {
    const k = 60 // RRF constant, typically 60
    
    // Calculate RRF scores
    scores := make(map[string]float64)
    results := make(map[string]RetrievalResult)
    
    for _, list := range lists {
        for rank, result := range list {
            rrf := 1.0 / float64(k+rank+1)
            scores[result.ID] += rrf
            
            // Keep the result with highest original score
            if existing, ok := results[result.ID]; !ok || result.Score > existing.Score {
                results[result.ID] = result
            }
        }
    }
    
    // Convert to slice and sort by RRF score
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
```

### Step 5: Implement Multi-Strategy Retrieval

```go
// QueryType represents the detected type of query.
type QueryType string

const (
    QueryTypeDefinition  QueryType = "definition"
    QueryTypeHowTo       QueryType = "how_to"
    QueryTypeComparison  QueryType = "comparison"
    QueryTypeExactMatch  QueryType = "exact_match"
    QueryTypeGeneral     QueryType = "general"
)

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
        // Definitions benefit from semantic search
        results, err = r.SimilaritySearch(ctx, query)
        
    case QueryTypeExactMatch:
        // Exact terms need keyword search
        results, err = r.KeywordSearch(ctx, query)
        
    case QueryTypeHowTo, QueryTypeComparison:
        // Complex queries benefit from hybrid
        results, err = r.HybridSearch(ctx, query)
        
    default:
        // Use configured default
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
    query = strings.ToLower(query)
    
    // Check for definition patterns
    if strings.HasPrefix(query, "what is ") ||
       strings.HasPrefix(query, "what are ") ||
       strings.HasPrefix(query, "define ") {
        return QueryTypeDefinition
    }
    
    // Check for how-to patterns
    if strings.HasPrefix(query, "how to ") ||
       strings.HasPrefix(query, "how do ") ||
       strings.HasPrefix(query, "how can ") ||
       strings.Contains(query, "tutorial") ||
       strings.Contains(query, "guide") {
        return QueryTypeHowTo
    }
    
    // Check for comparison patterns
    if strings.Contains(query, " vs ") ||
       strings.Contains(query, " versus ") ||
       strings.Contains(query, "compare") ||
       strings.Contains(query, "difference between") {
        return QueryTypeComparison
    }
    
    // Check for exact match patterns (IDs, codes, specific terms)
    if containsID(query) || containsTechnicalCode(query) {
        return QueryTypeExactMatch
    }
    
    return QueryTypeGeneral
}

func containsID(query string) bool {
    // Check for patterns like JIRA-123, UUID, etc.
    patterns := []string{
        `[A-Z]+-\d+`,           // JIRA-style
        `[0-9a-f]{8}-[0-9a-f]{4}`, // UUID prefix
    }
    for _, p := range patterns {
        if matched, _ := regexp.MatchString(p, query); matched {
            return true
        }
    }
    return false
}
```

### Step 6: Integrate Text Splitters

```go
// TextSplitterConfig configures how documents are chunked.
type TextSplitterConfig struct {
    ChunkSize      int
    ChunkOverlap   int
    SplitterType   string // "recursive", "semantic", "sentence"
    KeepSeparator  bool
}

// PrepareDocuments splits and indexes documents for retrieval.
func (r *AdvancedRetriever) PrepareDocuments(
    ctx context.Context,
    documents []Document,
    splitterConfig TextSplitterConfig,
) error {
    ctx, span := r.tracer.Start(ctx, "retriever.PrepareDocuments",
        trace.WithAttributes(
            attribute.Int("doc_count", len(documents)),
            attribute.Int("chunk_size", splitterConfig.ChunkSize),
        ),
    )
    defer span.End()
    
    // Create appropriate splitter
    splitter := createSplitter(splitterConfig)
    
    var allChunks []Document
    
    for _, doc := range documents {
        // Split document into chunks
        chunks := splitter.Split(doc.Content)
        
        for i, chunk := range chunks {
            allChunks = append(allChunks, Document{
                ID:      fmt.Sprintf("%s_chunk_%d", doc.ID, i),
                Content: chunk,
                Metadata: map[string]interface{}{
                    "source_id":    doc.ID,
                    "chunk_index":  i,
                    "total_chunks": len(chunks),
                },
            })
        }
    }
    
    span.SetAttributes(attribute.Int("chunk_count", len(allChunks)))
    
    // Generate embeddings and store
    for _, chunk := range allChunks {
        embedding, err := r.embedder.Embed(ctx, chunk.Content)
        if err != nil {
            return fmt.Errorf("failed to embed chunk %s: %w", chunk.ID, err)
        }
        
        if err := r.vectorStore.Add(ctx, chunk.ID, embedding, chunk.Content, chunk.Metadata); err != nil {
            return fmt.Errorf("failed to store chunk %s: %w", chunk.ID, err)
        }
        
        // Also index for keyword search
        if err := r.keywordStore.Index(ctx, chunk.ID, chunk.Content); err != nil {
            return fmt.Errorf("failed to index chunk %s: %w", chunk.ID, err)
        }
    }
    
    r.metrics.recordDocumentsIndexed(ctx, len(allChunks))
    
    return nil
}

func createSplitter(config TextSplitterConfig) TextSplitter {
    switch config.SplitterType {
    case "recursive":
        return NewRecursiveSplitter(config.ChunkSize, config.ChunkOverlap)
    case "semantic":
        return NewSemanticSplitter(config.ChunkSize)
    case "sentence":
        return NewSentenceSplitter(config.ChunkSize)
    default:
        return NewRecursiveSplitter(config.ChunkSize, config.ChunkOverlap)
    }
}
```

## Code Example

See the complete implementation:

- [advanced_retrieval.go](./advanced_retrieval.go) - Full implementation
- [advanced_retrieval_test.go](./advanced_retrieval_test.go) - Test suite

## Testing

### Unit Testing Retrieval Strategies

```go
func TestSimilaritySearch(t *testing.T) {
    ctx := context.Background()
    retriever := setupTestRetriever(t)
    
    results, err := retriever.SimilaritySearch(ctx, "error handling in Go")
    require.NoError(t, err)
    require.NotEmpty(t, results)
    
    // Verify all results are above threshold
    for _, r := range results {
        assert.GreaterOrEqual(t, r.Score, retriever.config.MinScore)
        assert.Equal(t, StrategySimilarity, r.Strategy)
    }
}

func TestHybridSearch_CombinesResults(t *testing.T) {
    ctx := context.Background()
    retriever := setupTestRetriever(t)
    
    results, err := retriever.HybridSearch(ctx, "OpenTelemetry tracing setup")
    require.NoError(t, err)
    
    // Should have results from both strategies
    assert.NotEmpty(t, results)
    assert.Equal(t, StrategyHybrid, results[0].Strategy)
}

func TestMultiStrategy_SelectsCorrectStrategy(t *testing.T) {
    tests := []struct {
        query        string
        expectedType QueryType
    }{
        {"What is RAG?", QueryTypeDefinition},
        {"How to implement caching", QueryTypeHowTo},
        {"Redis vs Memcached", QueryTypeComparison},
        {"JIRA-12345", QueryTypeExactMatch},
        {"best practices", QueryTypeGeneral},
    }
    
    retriever := setupTestRetriever(t)
    
    for _, tt := range tests {
        t.Run(tt.query, func(t *testing.T) {
            queryType := retriever.analyzeQuery(tt.query)
            assert.Equal(t, tt.expectedType, queryType)
        })
    }
}
```

### Benchmarking Retrieval

```go
func BenchmarkSimilaritySearch(b *testing.B) {
    ctx := context.Background()
    retriever := setupBenchRetriever(b)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = retriever.SimilaritySearch(ctx, "benchmark query")
    }
}

func BenchmarkHybridSearch(b *testing.B) {
    ctx := context.Background()
    retriever := setupBenchRetriever(b)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = retriever.HybridSearch(ctx, "benchmark query")
    }
}
```

## Best Practices

### 1. Choose the Right Default Strategy

```go
// For most RAG applications, hybrid is a good default
WithDefaultStrategy(StrategyHybrid)

// For semantic search apps (Q&A, recommendations)
WithDefaultStrategy(StrategySimilarity)

// For code search or technical docs with exact terms
WithDefaultStrategy(StrategyKeyword)
```

### 2. Tune Alpha for Your Domain

```go
// More weight to semantic similarity (general knowledge)
WithHybridAlpha(0.7)

// More weight to keywords (technical documentation)
WithHybridAlpha(0.3)

// Balanced (default)
WithHybridAlpha(0.5)
```

### 3. Use Appropriate Chunk Sizes

```go
// For detailed technical docs
TextSplitterConfig{ChunkSize: 500, ChunkOverlap: 50}

// For conversational content
TextSplitterConfig{ChunkSize: 1000, ChunkOverlap: 100}

// For code
TextSplitterConfig{ChunkSize: 1500, ChunkOverlap: 200}
```

### 4. Implement Caching

```go
// Cache embeddings for repeated queries
cache := NewEmbeddingCache(
    WithTTL(1 * time.Hour),
    WithMaxSize(10000),
)

// Cache search results for hot queries
resultCache := NewResultCache(
    WithTTL(5 * time.Minute),
    WithMaxSize(1000),
)
```

## Troubleshooting

### Q: Similarity search returns irrelevant results
**A:** Check your embedding model quality, increase MinScore threshold, or try a different embedding provider. Also verify your documents are well-chunked.

### Q: Keyword search misses relevant documents
**A:** Ensure documents are properly indexed. Consider stemming/lemmatization. Check if the keyword store supports partial matching.

### Q: Hybrid search is too slow
**A:** The parallel execution should be fast. Check if one strategy is slower than the other. Consider caching embeddings for repeated queries.

### Q: Query analysis picks the wrong strategy
**A:** Tune the pattern matching or consider using an LLM-based query classifier for more accuracy.

## Related Resources

- **[Multimodal RAG Guide](../../docs/guides/rag-multimodal.md)**: RAG with images and video
- **[RAG Evaluation](./rag_evaluation_guide.md)**: Measuring retrieval quality
- **[RAG Strategies Use Case](../../docs/use-cases/rag-strategies.md)**: When to use which strategy
- **[Vector Stores Guide](../../docs/guides/extensibility.md#vector-stores)**: Vector store setup
- **[Observability Tracing](../../docs/guides/observability-tracing.md)**: Distributed tracing
