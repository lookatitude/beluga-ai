---
title: "Reranking with Cohere Rerank"
description: "Improve retrieval quality by reranking initial search results using Cohere's cross-encoder rerank API for better relevance scoring."
---

# Reranking with Cohere Rerank

## Problem

You need to improve retrieval quality by reranking initial search results using Cohere's rerank API, which can significantly improve relevance when your initial vector search returns many candidates.

Standard vector search uses bi-encoders that embed queries and documents independently, then compare via cosine similarity. This is fast (sub-millisecond per comparison) but fundamentally limited: the query and document never "see" each other during encoding. Cross-encoder rerankers like Cohere's Rerank API process query-document pairs together, allowing the model to attend to fine-grained interactions between query terms and document content. Research consistently shows that cross-encoder reranking improves retrieval quality by 10-30% on standard benchmarks compared to bi-encoder search alone.

## Solution

Implement a two-stage retrieval pipeline: first retrieve a broad candidate set using fast vector search, then rerank those candidates with a cross-encoder for better relevance ordering. The reranker wraps the base retriever, making reranking transparent to callers. If the rerank service is unavailable, the pipeline falls back to the original results, ensuring availability is not sacrificed for quality.

## Code Example

```go
package main

import (
    "context"
    "fmt"
    "sort"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"

    "github.com/lookatitude/beluga-ai/schema"
)

var tracer = otel.Tracer("beluga.retrievers.rerank")

// CohereReranker reranks documents using Cohere API
type CohereReranker struct {
    apiKey string
    model  string
    topN   int
    client CohereClient
}

// CohereClient interface for Cohere API calls
type CohereClient interface {
    Rerank(ctx context.Context, query string, documents []string, topN int) ([]RerankResult, error)
}

// RerankResult represents a reranked result
type RerankResult struct {
    Index    int
    Document string
    Score    float64
}

// NewCohereReranker creates a new reranker
func NewCohereReranker(apiKey string, model string, topN int) *CohereReranker {
    return &CohereReranker{
        apiKey: apiKey,
        model:  model,
        topN:   topN,
        client: NewCohereAPIClient(apiKey), // Implementation would create actual client
    }
}

// RerankDocuments reranks documents for a query
func (cr *CohereReranker) RerankDocuments(ctx context.Context, query string, documents []schema.Document) ([]schema.Document, []float64, error) {
    ctx, span := tracer.Start(ctx, "cohere_reranker.rerank")
    defer span.End()

    span.SetAttributes(
        attribute.String("query", query),
        attribute.Int("document_count", len(documents)),
        attribute.Int("top_n", cr.topN),
    )

    // Extract document texts
    texts := make([]string, len(documents))
    for i, doc := range documents {
        texts[i] = doc.GetContent()
    }

    // Call Cohere rerank API
    results, err := cr.client.Rerank(ctx, query, texts, cr.topN)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return nil, nil, fmt.Errorf("rerank failed: %w", err)
    }

    // Build reranked documents and scores
    rerankedDocs := make([]schema.Document, len(results))
    scores := make([]float64, len(results))

    for i, result := range results {
        rerankedDocs[i] = documents[result.Index]
        scores[i] = result.Score
    }

    span.SetAttributes(
        attribute.Int("reranked_count", len(rerankedDocs)),
        attribute.Float64("top_score", scores[0]),
    )
    span.SetStatus(trace.StatusOK, "reranking completed")

    return rerankedDocs, scores, nil
}

// RerankedRetriever wraps a retriever with reranking
type RerankedRetriever struct {
    retriever Retriever
    reranker  *CohereReranker
    initialK  int // Retrieve more initially, then rerank to topN
}

// Retriever interface for document retrieval
type Retriever interface {
    GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error)
}

// NewRerankedRetriever creates a new reranked retriever
func NewRerankedRetriever(retriever Retriever, reranker *CohereReranker, initialK int) *RerankedRetriever {
    return &RerankedRetriever{
        retriever: retriever,
        reranker:  reranker,
        initialK:  initialK,
    }
}

// GetRelevantDocuments retrieves and reranks documents
func (rr *RerankedRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
    ctx, span := tracer.Start(ctx, "reranked_retriever.get_relevant")
    defer span.End()

    // Retrieve more documents initially
    docs, err := rr.retriever.GetRelevantDocuments(ctx, query)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return nil, err
    }

    // Limit to initialK
    if len(docs) > rr.initialK {
        docs = docs[:rr.initialK]
    }

    span.SetAttributes(attribute.Int("initial_retrieval_count", len(docs)))

    // Rerank
    reranked, _, err := rr.reranker.RerankDocuments(ctx, query, docs)
    if err != nil {
        span.RecordError(err)
        // Fallback to original results
        return docs, nil
    }

    span.SetAttributes(attribute.Int("final_count", len(reranked)))
    span.SetStatus(trace.StatusOK, "retrieved and reranked")

    return reranked, nil
}

// NewCohereAPIClient creates a Cohere API client (simplified)
func NewCohereAPIClient(apiKey string) CohereClient {
    // In practice, this would create an actual Cohere API client
    return &MockCohereClient{}
}

type MockCohereClient struct{}

func (m *MockCohereClient) Rerank(ctx context.Context, query string, documents []string, topN int) ([]RerankResult, error) {
    // Mock implementation - in practice, call Cohere API
    results := make([]RerankResult, len(documents))
    for i, doc := range documents {
        results[i] = RerankResult{
            Index:    i,
            Document: doc,
            Score:    0.9 - float64(i)*0.1, // Mock scores
        }
    }

    // Sort by score descending
    sort.Slice(results, func(i, j int) bool {
        return results[i].Score > results[j].Score
    })

    // Return topN
    if len(results) > topN {
        results = results[:topN]
    }

    return results, nil
}

func main() {
    ctx := context.Background()

    // Create reranker
    reranker := NewCohereReranker("api-key", "rerank-multilingual-v3.0", 5)

    // Create reranked retriever
    // retriever := yourRetriever
    // rerankedRetriever := NewRerankedRetriever(retriever, reranker, 20)

    // Retrieve with reranking
    // docs, err := rerankedRetriever.GetRelevantDocuments(ctx, "query")
    fmt.Println("Reranker created")
    _ = ctx
    _ = reranker
}
```

## Explanation

1. **Two-stage retrieval** -- More documents are retrieved initially (initialK) than needed, then reranked to get the top results. This gives the cross-encoder a diverse candidate set to work with. A good rule of thumb is to retrieve 3-5x more candidates than the final count (e.g., retrieve 50, rerank to 10). The initial retrieval is fast (vector search), and the cross-encoder adds 100-500ms for the reranking step.

2. **Reranking as a wrapper** -- The base retriever is wrapped with reranking functionality using the decorator pattern. This means reranking can be added to any retriever without modifying the base implementation. The `RerankedRetriever` satisfies the same `Retriever` interface, so callers don't need to know reranking is happening -- it's transparent.

3. **Fallback on errors** -- If the Cohere rerank API is unavailable (network issues, rate limits, service outage), the original vector search results are returned. This ensures retrieval still works even without the rerank service, trading quality for availability. In production, you should monitor fallback rates to detect persistent reranker issues.

> **Key insight:** Retrieve more candidates than you need, then rerank to the final count. This improves quality because reranking works best with a diverse candidate set that includes both highly relevant and borderline documents.

## Testing

```go
func TestRerankedRetriever_ReranksResults(t *testing.T) {
    mockRetriever := &MockRetriever{}
    mockReranker := &CohereReranker{client: &MockCohereClient{}}

    retriever := NewRerankedRetriever(mockRetriever, mockReranker, 10)

    docs, err := retriever.GetRelevantDocuments(context.Background(), "test query")
    require.NoError(t, err)
    require.LessOrEqual(t, len(docs), 5) // Should be reranked to topN
}
```

## Variations

### Batch Reranking

Rerank multiple queries at once:

```go
func (cr *CohereReranker) BatchRerank(ctx context.Context, queries []string, documents [][]schema.Document) ([][]schema.Document, error) {
    // Batch rerank
}
```

### Hybrid Scoring

Combine vector similarity with rerank scores:

```go
type HybridScorer struct {
    vectorWeight float64
    rerankWeight float64
}
```

## Related Recipes

- [Parent Document Retrieval (PDR)](/cookbook/parent-document-retrieval) -- Retrieve parent documents
- [Vectorstores Advanced Meta-filtering](/cookbook/meta-filtering) -- Advanced filtering
