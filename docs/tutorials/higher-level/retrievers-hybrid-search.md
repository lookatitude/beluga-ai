# Hybrid Search Implementation

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this tutorial, we'll implement Hybrid Search by combining the precision of keyword search (BM25) with the semantic understanding of vector search. We'll use Reciprocal Rank Fusion (RRF) to merge results into a single, highly relevant list.

## Learning Objectives
- ✅ Understand Semantic vs. Keyword search pros/cons
- ✅ Implement Reciprocal Rank Fusion (RRF)
- ✅ Build a Hybrid Retriever using Beluga AI

## Introduction
Welcome, colleague! Vector search is amazing for concepts, but it's terrible at exact matches like error codes or product IDs. To build a production-grade RAG system, you need both. Let's build a hybrid retriever that gives us the best of both worlds.

## Prerequisites

- [Simple RAG](../../getting-started/02-simple-rag.md)
- Vector Store setup

## Why Hybrid?

- **Vector Search**: Finds concepts ("canine" matches "dog"). Fails at exact IDs ("Error 504") or acronyms.
- **Keyword Search**: Finds exact matches. Fails at synonyms.
- **Hybrid**: Best of both worlds.

## Step 1: Define the Retriever Interface

We need a retriever that queries both sources.
```go
type HybridRetriever struct {
    vectorRetriever core.Retriever
    keywordRetriever core.Retriever
    weightVector float64
    weightKeyword float64
}

## Step 2: Implementing Search
func (h *HybridRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
    // 1. Parallel execution
    vecDocs, _ := h.vectorRetriever.GetRelevantDocuments(ctx, query)
    keyDocs, _ := h.keywordRetriever.GetRelevantDocuments(ctx, query)
    
    // 2. Fusion (Simple weighted) or RRF
    return fuseResults(vecDocs, keyDocs), nil
}
```

## Step 3: Reciprocal Rank Fusion (RRF)

RRF is a standard algorithm for merging ranked lists without needing calibrated scores.
```go
func fuseResults(listA, listB []schema.Document) []schema.Document {
    scores := make(map[string]float64)
    k := 60.0 // RRF constant
    
    // Score list A
    for rank, doc := range listA {
        scores[doc.ID] += 1.0 / (k + float64(rank) + 1.0)
    }
    
    // Score list B
    for rank, doc := range listB {
        scores[doc.ID] += 1.0 / (k + float64(rank) + 1.0)
    }
    
    // Sort by score and return top docs...
    return sortDocs(scores)
}

## Step 4: Using the Retriever
func main() {
    // Create underlying retrievers
    vecRetriever := vectorstores.NewRetriever(vStore)
    keyRetriever := elasticsearch.NewRetriever(esClient) // Hypothetical
    
    // Create Hybrid
    hybrid := NewHybridRetriever(vecRetriever, keyRetriever)
    
    // Use in RAG
    ragChain := orchestration.NewChain([]core.Runnable{
        hybrid,
        generationStep,
    })
}
```

## Verification

1. Search for a generic concept ("how to fix printer"). Vector should win.
2. Search for an exact error code ("Error X99-Z"). Keyword should win.
3. Hybrid should return relevant results for both.

## Next Steps

- **[Multi-query Retrieval Chains](./retrievers-multiquery-chains.md)** - Enhance the query side
- **[Production pgvector Sharding](../providers/vectorstores-pgvector-sharding.md)** - Scale the vector side
