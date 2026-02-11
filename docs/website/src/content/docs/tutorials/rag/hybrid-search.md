---
title: Hybrid Search Implementation
description: Combine vector search and keyword search with Reciprocal Rank Fusion (RRF) for production-grade retrieval.
---

Vector search excels at semantic similarity ("canine" matches "dog") but fails at exact matches like error codes or product IDs. Keyword search (BM25) excels at exact matches but fails at synonyms. Neither method alone provides reliable retrieval across all query types. Hybrid search combines both methods using **Reciprocal Rank Fusion (RRF)** to produce a single, highly relevant result list that captures both semantic understanding and exact matching.

Beluga AI v2 uses hybrid search as the default retrieval strategy because it consistently outperforms either approach alone across diverse query types.

## What You Will Build

A hybrid retriever that queries both vector and keyword backends in parallel, fuses the results with RRF, and returns a unified ranked list.

## Prerequisites

- Understanding of [In-memory Vector Store](/tutorials/providers/inmemory-vectorstore)
- A vector store and a keyword search backend (or a mock)

## Why Hybrid Search

| Search Type | Strengths | Weaknesses |
|:---|:---|:---|
| Vector | Concepts, synonyms, paraphrases | Exact IDs, codes, proper nouns |
| Keyword (BM25) | Exact matches, filtering | Synonyms, paraphrases |
| Hybrid | Both | Slightly more complex |

## Step 1: Define the Retriever Interface

The `Retriever` interface is intentionally minimal -- a single method that takes a query string and returns ranked documents. Both vector search and keyword search backends implement this interface, allowing the hybrid retriever to treat them uniformly. This follows the same interface-first design pattern used across Beluga AI.

```go
package main

import (
    "context"
    "sort"

    "github.com/lookatitude/beluga-ai/schema"
)

// Retriever searches for relevant documents.
type Retriever interface {
    Retrieve(ctx context.Context, query string, topK int) ([]schema.Document, error)
}
```

## Step 2: Build the Hybrid Retriever

The hybrid retriever queries both backends and fuses the results. Each backend is asked for `topK * 2` results (over-fetching) because RRF may re-rank documents from one list above documents from the other -- having a larger candidate pool gives the fusion algorithm more material to work with, producing a better final ranking.

```go
// HybridRetriever combines vector and keyword search.
type HybridRetriever struct {
    vectorRetriever  Retriever
    keywordRetriever Retriever
}

func NewHybridRetriever(vector, keyword Retriever) *HybridRetriever {
    return &HybridRetriever{
        vectorRetriever:  vector,
        keywordRetriever: keyword,
    }
}

func (h *HybridRetriever) Retrieve(ctx context.Context, query string, topK int) ([]schema.Document, error) {
    // Query both backends (could be parallelized with goroutines)
    vecDocs, err := h.vectorRetriever.Retrieve(ctx, query, topK*2)
    if err != nil {
        return nil, fmt.Errorf("vector search: %w", err)
    }

    keyDocs, err := h.keywordRetriever.Retrieve(ctx, query, topK*2)
    if err != nil {
        return nil, fmt.Errorf("keyword search: %w", err)
    }

    // Fuse results
    fused := reciprocalRankFusion(vecDocs, keyDocs)

    // Return top K
    if len(fused) > topK {
        fused = fused[:topK]
    }

    return fused, nil
}
```

## Step 3: Implement Reciprocal Rank Fusion

RRF is the key to making hybrid search work. Unlike simple score averaging, RRF does not require calibrated similarity scores -- it works purely on rank positions. This is important because vector similarity scores (0.0 to 1.0) and BM25 scores (unbounded positive values) are not directly comparable. The RRF formula `1 / (k + rank + 1)` assigns each document a score based on its position in each list, then sums the scores across all lists. Documents that rank highly in both lists receive the highest combined scores.

The constant `k = 60` is the standard value from the original RRF paper. It controls how quickly scores decay with rank position -- higher values make the ranking more uniform (less difference between rank 1 and rank 10), while lower values amplify the advantage of top-ranked results.

```go
import "fmt"

const rrfK = 60.0 // Standard RRF constant

type scoredDoc struct {
    doc   schema.Document
    score float64
}

func reciprocalRankFusion(lists ...[]schema.Document) []schema.Document {
    scores := make(map[string]*scoredDoc)

    for _, list := range lists {
        for rank, doc := range list {
            key := docKey(doc)
            if _, ok := scores[key]; !ok {
                scores[key] = &scoredDoc{doc: doc}
            }
            // RRF formula: 1 / (k + rank + 1)
            scores[key].score += 1.0 / (rrfK + float64(rank) + 1.0)
        }
    }

    // Sort by descending score
    result := make([]scoredDoc, 0, len(scores))
    for _, sd := range scores {
        result = append(result, *sd)
    }
    sort.Slice(result, func(i, j int) bool {
        return result[i].score > result[j].score
    })

    // Extract documents with scores
    docs := make([]schema.Document, len(result))
    for i, sd := range result {
        sd.doc.Score = sd.score
        docs[i] = sd.doc
    }

    return docs
}

func docKey(doc schema.Document) string {
    if doc.ID != "" {
        return doc.ID
    }
    return doc.Content // Fallback to content hash
}
```

## Step 4: Parallel Retrieval

Since vector search and keyword search are independent operations targeting different backends, running them concurrently cuts latency roughly in half. Each search runs in its own goroutine, and a `sync.WaitGroup` synchronizes completion. The error handling follows a fail-fast pattern: if either backend fails, the entire operation returns an error rather than returning partial results that could mislead the downstream model.

```go
import "sync"

func (h *HybridRetriever) RetrieveParallel(ctx context.Context, query string, topK int) ([]schema.Document, error) {
    var vecDocs, keyDocs []schema.Document
    var vecErr, keyErr error
    var wg sync.WaitGroup

    wg.Add(2)
    go func() {
        defer wg.Done()
        vecDocs, vecErr = h.vectorRetriever.Retrieve(ctx, query, topK*2)
    }()
    go func() {
        defer wg.Done()
        keyDocs, keyErr = h.keywordRetriever.Retrieve(ctx, query, topK*2)
    }()
    wg.Wait()

    if vecErr != nil {
        return nil, fmt.Errorf("vector search: %w", vecErr)
    }
    if keyErr != nil {
        return nil, fmt.Errorf("keyword search: %w", keyErr)
    }

    fused := reciprocalRankFusion(vecDocs, keyDocs)
    if len(fused) > topK {
        fused = fused[:topK]
    }

    return fused, nil
}
```

## Step 5: Use in a RAG Pipeline

The hybrid retriever integrates into a standard RAG pipeline as a drop-in replacement for any single retriever. The query "error code ERR_CONNECTION_REFUSED" demonstrates hybrid search's strength: vector search finds semantically related troubleshooting docs, while keyword search finds exact matches for the error code string. RRF combines both, giving the model comprehensive context for generating an accurate answer.

```go
func main() {
    ctx := context.Background()

    hybrid := NewHybridRetriever(vectorRetriever, keywordRetriever)

    // Retrieve relevant documents
    docs, err := hybrid.Retrieve(ctx, "error code ERR_CONNECTION_REFUSED", 5)
    if err != nil {
        fmt.Printf("Search error: %v\n", err)
        return
    }

    // Build context for the LLM
    var context string
    for _, doc := range docs {
        context += fmt.Sprintf("---\n%s\n", doc.Content)
    }

    // Generate answer using retrieved context
    msgs := []schema.Message{
        schema.NewSystemMessage("Answer based on the following context:\n" + context),
        schema.NewHumanMessage("How do I fix ERR_CONNECTION_REFUSED?"),
    }

    resp, err := model.Generate(ctx, msgs)
    if err != nil {
        fmt.Printf("Generate error: %v\n", err)
        return
    }
    fmt.Println(resp.Text())
}
```

## Verification

1. Search for a generic concept ("how to fix printer") -- vector search should contribute more.
2. Search for an exact error code ("Error X99-Z") -- keyword search should contribute more.
3. Hybrid results should rank relevant documents highly for both query types.

## Next Steps

- [Multi-query Retrieval](/tutorials/rag/multiquery-chains) -- Expand queries for better recall
- [pgvector Sharding](/tutorials/providers/pgvector-sharding) -- Scale the vector backend
