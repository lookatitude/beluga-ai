---
title: "Parent Document Retrieval (PDR)"
package: "retrievers"
category: "retrieval"
complexity: "advanced"
---

# Parent Document Retrieval (PDR)

## Problem

You need to retrieve fine-grained chunks for initial matching but return larger parent documents for context, balancing retrieval precision with sufficient context for generation.

## Solution

Implement Parent Document Retrieval (PDR) that stores documents hierarchically (small chunks for retrieval, parent documents for context), retrieves chunks first, then expands to parent documents. This works because small chunks improve retrieval precision, while parent documents provide necessary context.

## Code Example
```go
package main

import (
    "context"
    "fmt"
    "log"
    "sort"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
    
    "github.com/lookatitude/beluga-ai/pkg/retrievers/iface"
    "github.com/lookatitude/beluga-ai/pkg/schema"
    "github.com/lookatitude/beluga-ai/pkg/vectorstores"
)

var tracer = otel.Tracer("beluga.retrievers.pdr")

// ParentDocumentRetriever implements PDR
type ParentDocumentRetriever struct {
    chunkStore    vectorstores.VectorStore
    parentStore   map[string]schema.Document // Map of chunk ID to parent document
    chunkK        int                         // Number of chunks to retrieve
    parentDocs    int                         // Number of parent docs to return
}

// NewParentDocumentRetriever creates a new PDR retriever
func NewParentDocumentRetriever(chunkStore vectorstores.VectorStore, chunkK, parentDocs int) *ParentDocumentRetriever {
    return &ParentDocumentRetriever{
        chunkStore: chunkStore,
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
    chunkIDs, err := pdr.chunkStore.AddDocuments(ctx, chunks, vectorstores.WithEmbedder(embedder))
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
    pdr := NewParentDocumentRetriever(chunkStore, 10, 3)
    
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
}
```

## Explanation

Let's break down what's happening:

1. **Hierarchical storage** - Notice how we store chunks in the vector store for retrieval, but maintain a mapping to parent documents. This allows us to retrieve precise chunks but return comprehensive context.

2. **Score propagation** - When multiple chunks from the same parent match, we keep the best score. This ensures parent documents with highly relevant chunks rank higher.

3. **Deduplication** - We deduplicate parent documents even when multiple chunks match. This prevents returning the same parent document multiple times.

```go
**Key insight:** Use small chunks for precise retrieval, but return parent documents for context. This gives you both retrieval quality and generation context.

## Testing

```
Here's how to test this solution:
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

## Variations

### Metadata Preservation

Preserve chunk metadata in parent documents:
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

- **[Retrievers Reranking with Cohere Rerank](./retrievers-reranking-cohere-rerank.md)** - Improve retrieval quality
- **[Textsplitters Advanced Code Splitting](./textsplitters-advanced-code-splitting-tree-sitter.md)** - Split documents intelligently
- **[Retrievers Package Guide](../package_design_patterns.md)** - For a deeper understanding of retrievers
