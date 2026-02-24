---
title: "Parent Document Retrieval (PDR)"
description: "Recipe for parent document retrieval in Go — match on fine-grained chunks but return full parent documents for richer LLM context using Beluga AI."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, parent document retrieval, Go RAG pattern, chunk matching, context enrichment, PDR recipe, retrieval strategy"
---

# Parent Document Retrieval (PDR)

## Problem

You need to retrieve fine-grained chunks for initial matching but return larger parent documents for context, balancing retrieval precision with sufficient context for generation.

There is a fundamental tension in chunk sizing for RAG systems. Small chunks (100-300 tokens) produce precise embeddings that match specific queries well, but they provide too little context for the LLM to generate grounded, accurate responses. Large chunks (1000+ tokens) provide rich context but produce diffuse embeddings that match too broadly. Parent Document Retrieval resolves this by maintaining two levels: small chunks for search precision and larger parent documents for generation context.

## Solution

Implement Parent Document Retrieval (PDR) that stores documents hierarchically: small chunks are indexed in the vector store for retrieval, while a separate mapping links each chunk to its parent document. At query time, the system retrieves matching chunks, looks up their parent documents, deduplicates parents (since multiple chunks from the same parent may match), and returns the parents sorted by the best chunk score. This gives the LLM sufficient context while maintaining retrieval precision.

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
    "github.com/lookatitude/beluga-ai/rag/vectorstore"
)

var tracer = otel.Tracer("beluga.retrievers.pdr")

// ParentDocumentRetriever implements PDR
type ParentDocumentRetriever struct {
    chunkStore  vectorstore.VectorStore
    parentStore map[string]schema.Document // Map of chunk ID to parent document
    chunkK      int                        // Number of chunks to retrieve
    parentDocs  int                        // Number of parent docs to return
}

// NewParentDocumentRetriever creates a new PDR retriever
func NewParentDocumentRetriever(chunkStore vectorstore.VectorStore, chunkK, parentDocs int) *ParentDocumentRetriever {
    return &ParentDocumentRetriever{
        chunkStore:  chunkStore,
        parentStore: make(map[string]schema.Document),
        chunkK:      chunkK,
        parentDocs:  parentDocs,
    }
}

// AddDocuments adds documents with parent-child relationships
func (pdr *ParentDocumentRetriever) AddDocuments(ctx context.Context, parentDoc schema.Document, chunks []schema.Document, embedder interface{}) error {
    ctx, span := tracer.Start(ctx, "pdr_retriever.add_documents")
    defer span.End()

    // Store chunks in vector store
    chunkIDs, err := pdr.chunkStore.AddDocuments(ctx, chunks, vectorstore.WithEmbedder(embedder))
    if err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return err
    }

    // Map chunk IDs to parent document
    for _, chunkID := range chunkIDs {
        pdr.parentStore[chunkID] = parentDoc
    }

    span.SetAttributes(
        attribute.Int("chunk_count", len(chunks)),
        attribute.String("parent_id", parentDoc.ID),
    )
    span.SetStatus(trace.StatusOK, "documents added")

    return nil
}

// GetRelevantDocuments retrieves parent documents via chunk matching
func (pdr *ParentDocumentRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
    ctx, span := tracer.Start(ctx, "pdr_retriever.get_relevant")
    defer span.End()

    span.SetAttributes(attribute.String("query", query))

    // Retrieve chunks
    chunks, scores, err := pdr.chunkStore.SimilaritySearchByQuery(ctx, query, pdr.chunkK, nil)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return nil, err
    }

    span.SetAttributes(attribute.Int("chunk_count", len(chunks)))

    // Map chunks to parent documents
    parentMap := make(map[string]ParentDocWithScore)

    for i, chunk := range chunks {
        chunkID := chunk.ID
        parentDoc, exists := pdr.parentStore[chunkID]

        if !exists {
            // If no parent found, use chunk itself
            parentDoc = chunk
        }

        // Track parent with best score
        if existing, found := parentMap[parentDoc.ID]; !found || scores[i] > existing.Score {
            parentMap[parentDoc.ID] = ParentDocWithScore{
                Document: parentDoc,
                Score:    scores[i],
            }
        }
    }

    // Convert to slice and sort by score
    parents := make([]ParentDocWithScore, 0, len(parentMap))
    for _, pds := range parentMap {
        parents = append(parents, pds)
    }

    // Sort by score descending
    sort.Slice(parents, func(i, j int) bool {
        return parents[i].Score > parents[j].Score
    })

    // Limit to requested number
    if len(parents) > pdr.parentDocs {
        parents = parents[:pdr.parentDocs]
    }

    // Extract documents
    result := make([]schema.Document, len(parents))
    for i, pds := range parents {
        result[i] = pds.Document
    }

    span.SetAttributes(
        attribute.Int("parent_count", len(result)),
        attribute.Float64("top_score", parents[0].Score),
    )
    span.SetStatus(trace.StatusOK, "parent documents retrieved")

    return result, nil
}

// ParentDocWithScore represents a parent document with score
type ParentDocWithScore struct {
    Document schema.Document
    Score    float64
}

func main() {
    ctx := context.Background()

    // Create retriever
    // chunkStore := yourVectorStore
    // pdr := NewParentDocumentRetriever(chunkStore, 10, 3)

    // Add parent document with chunks
    parentDoc := schema.NewDocument("Full document content...", nil)
    chunks := []schema.Document{
        schema.NewDocument("Chunk 1 content", nil),
        schema.NewDocument("Chunk 2 content", nil),
    }

    // embedder := yourEmbedder
    // pdr.AddDocuments(ctx, parentDoc, chunks, embedder)

    // Retrieve
    // docs, err := pdr.GetRelevantDocuments(ctx, "query")
    fmt.Println("PDR retriever created")
    _ = ctx
    _ = parentDoc
    _ = chunks
}
```

## Explanation

1. **Hierarchical storage** -- Chunks are stored in the vector store for similarity search, while a separate mapping links chunk IDs to parent documents. This dual storage allows precise chunk-level retrieval with comprehensive parent-level context in the response. The vector store only indexes the small chunks, keeping the index efficient, while the parent store provides the richer context needed for generation.

2. **Score propagation** -- When multiple chunks from the same parent match a query, only the best score is kept for ranking. This prevents a parent with many low-relevance chunk matches from outranking a parent with one highly relevant chunk match. The best-score-wins approach ensures the final ranking reflects the strongest evidence for each parent document.

3. **Deduplication** -- Parent documents are deduplicated using a map keyed by parent ID, even when multiple chunks match. Without deduplication, the same parent document could appear multiple times in results, wasting context window space. The map-based approach handles deduplication efficiently in O(n) time.

> **Key insight:** Use small chunks (100-300 tokens) for precise retrieval, but return parent documents (500-2000 tokens) for generation context. This gives you both retrieval quality and generation richness. The chunk-to-parent ratio is typically 3-10x.

## Testing

```go
func TestParentDocumentRetriever_RetrievesParents(t *testing.T) {
    mockStore := &MockVectorStore{}
    pdr := NewParentDocumentRetriever(mockStore, 10, 3)

    parentDoc := schema.NewDocument("Parent", nil)
    chunks := []schema.Document{
        schema.NewDocument("Chunk 1", nil),
        schema.NewDocument("Chunk 2", nil),
    }

    err := pdr.AddDocuments(context.Background(), parentDoc, chunks, nil)
    require.NoError(t, err)

    docs, err := pdr.GetRelevantDocuments(context.Background(), "query")
    require.NoError(t, err)
    require.Contains(t, docs, parentDoc)
}
```

## Variations

### Metadata Preservation

Preserve chunk metadata in parent documents:

```go
type EnhancedParentDoc struct {
    Document schema.Document
    Chunks   []schema.Document
    Metadata map[string]interface{}
}
```

### Overlapping Chunks

Handle overlapping chunks in parent documents:

```go
func (pdr *ParentDocumentRetriever) MergeOverlappingChunks(chunks []schema.Document) schema.Document {
    // Merge overlapping chunks into parent
}
```

## Related Recipes

- [Reranking with Cohere Rerank](/cookbook/cohere-reranking) -- Improve retrieval quality with cross-encoder reranking
- [Advanced Code Splitting](/cookbook/code-splitting) -- Split documents intelligently at structural boundaries
