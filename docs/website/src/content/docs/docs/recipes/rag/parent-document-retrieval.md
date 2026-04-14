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

Implement Parent Document Retrieval (PDR) that stores documents hierarchically: small chunks are embedded and indexed in the vector store for retrieval, while a separate mapping links each chunk ID to its parent document. At query time, the system embeds the query, retrieves matching chunks, looks up their parent documents, deduplicates parents (since multiple chunks from the same parent may match), and returns the parents sorted by the best chunk score.

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

    "github.com/lookatitude/beluga-ai/rag/embedding"
    "github.com/lookatitude/beluga-ai/rag/vectorstore"
    "github.com/lookatitude/beluga-ai/schema"
)

var tracer = otel.Tracer("beluga.retrievers.pdr")

// ParentDocumentRetriever implements PDR: retrieves chunks, maps to parents.
type ParentDocumentRetriever struct {
    chunkStore  vectorstore.VectorStore
    embedder    embedding.Embedder
    parentStore map[string]schema.Document // chunk ID -> parent document
    chunkK      int                        // number of chunks to retrieve
    parentDocs  int                        // number of parent docs to return
}

// NewParentDocumentRetriever creates a new PDR retriever.
func NewParentDocumentRetriever(chunkStore vectorstore.VectorStore, embedder embedding.Embedder, chunkK, parentDocs int) *ParentDocumentRetriever {
    return &ParentDocumentRetriever{
        chunkStore:  chunkStore,
        embedder:    embedder,
        parentStore: make(map[string]schema.Document),
        chunkK:      chunkK,
        parentDocs:  parentDocs,
    }
}

// IndexChunks embeds chunks and adds them to the vector store, recording the
// parent document for each chunk ID.
func (pdr *ParentDocumentRetriever) IndexChunks(ctx context.Context, parentDoc schema.Document, chunks []schema.Document) error {
    ctx, span := tracer.Start(ctx, "pdr_retriever.index_chunks")
    defer span.End()

    texts := make([]string, len(chunks))
    for i, c := range chunks {
        texts[i] = c.Content
    }

    embeddings, err := pdr.embedder.Embed(ctx, texts)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return fmt.Errorf("pdr: embed chunks: %w", err)
    }

    if err := pdr.chunkStore.Add(ctx, chunks, embeddings); err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return fmt.Errorf("pdr: add chunks: %w", err)
    }

    // Record chunk ID -> parent mapping.
    for _, c := range chunks {
        pdr.parentStore[c.ID] = parentDoc
    }

    span.SetAttributes(
        attribute.Int("chunk_count", len(chunks)),
        attribute.String("parent_id", parentDoc.ID),
    )
    span.SetStatus(trace.StatusOK, "chunks indexed")

    return nil
}

// ParentDocWithScore tracks the best chunk score for a parent document.
type ParentDocWithScore struct {
    Document schema.Document
    Score    float64
}

// Retrieve embeds the query, finds matching chunks, and returns deduplicated parent documents.
func (pdr *ParentDocumentRetriever) Retrieve(ctx context.Context, query string) ([]schema.Document, error) {
    ctx, span := tracer.Start(ctx, "pdr_retriever.retrieve")
    defer span.End()

    span.SetAttributes(attribute.String("query", query))

    queryVec, err := pdr.embedder.EmbedSingle(ctx, query)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return nil, fmt.Errorf("pdr: embed query: %w", err)
    }

    chunks, err := pdr.chunkStore.Search(ctx, queryVec, pdr.chunkK)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return nil, fmt.Errorf("pdr: search chunks: %w", err)
    }

    span.SetAttributes(attribute.Int("chunk_count", len(chunks)))

    // Map chunks to parent documents, keeping the best score per parent.
    parentMap := make(map[string]ParentDocWithScore)
    for _, chunk := range chunks {
        parentDoc, exists := pdr.parentStore[chunk.ID]
        if !exists {
            parentDoc = chunk // fall back to chunk itself if no parent recorded
        }

        existing, found := parentMap[parentDoc.ID]
        if !found || chunk.Score > existing.Score {
            parentMap[parentDoc.ID] = ParentDocWithScore{
                Document: parentDoc,
                Score:    chunk.Score,
            }
        }
    }

    // Sort by descending score.
    parents := make([]ParentDocWithScore, 0, len(parentMap))
    for _, pds := range parentMap {
        parents = append(parents, pds)
    }
    sort.Slice(parents, func(i, j int) bool {
        return parents[i].Score > parents[j].Score
    })

    if len(parents) > pdr.parentDocs {
        parents = parents[:pdr.parentDocs]
    }

    result := make([]schema.Document, len(parents))
    for i, pds := range parents {
        result[i] = pds.Document
    }

    if len(parents) > 0 {
        span.SetAttributes(
            attribute.Int("parent_count", len(result)),
            attribute.Float64("top_score", parents[0].Score),
        )
    }
    span.SetStatus(trace.StatusOK, "parent documents retrieved")

    return result, nil
}

func main() {
    ctx := context.Background()

    // chunkStore and embedder are initialized via vectorstore.New / embedding.New.
    var chunkStore vectorstore.VectorStore
    var emb embedding.Embedder

    pdr := NewParentDocumentRetriever(chunkStore, emb, 10, 3)

    parentDoc := schema.Document{ID: "doc-1", Content: "Full document content..."}
    chunks := []schema.Document{
        {ID: "chunk-1-1", Content: "Chunk 1 content", Metadata: map[string]any{"parent_id": "doc-1"}},
        {ID: "chunk-1-2", Content: "Chunk 2 content", Metadata: map[string]any{"parent_id": "doc-1"}},
    }

    if err := pdr.IndexChunks(ctx, parentDoc, chunks); err != nil {
        fmt.Printf("index error: %v\n", err)
        return
    }

    docs, err := pdr.Retrieve(ctx, "query about document content")
    if err != nil {
        fmt.Printf("retrieve error: %v\n", err)
        return
    }
    fmt.Printf("Retrieved %d parent documents\n", len(docs))
}
```

## Explanation

1. **Hierarchical storage** -- Chunks are embedded and stored in the vector store for similarity search, while `parentStore` maps chunk IDs to parent documents. This dual storage allows precise chunk-level retrieval with comprehensive parent-level context. The vector store indexes only the small chunks, keeping the index efficient, while the parent map provides richer context for generation.

2. **Score propagation** -- When multiple chunks from the same parent match a query, only the best `Score` (populated by `vectorstore.Search`) is kept for ranking. This prevents a parent with many low-relevance chunk matches from outranking one with a single highly relevant match.

3. **Deduplication** -- Parent documents are deduplicated using a map keyed by parent ID. Without deduplication, the same parent could appear multiple times in results, wasting the LLM's context window.

> **Key insight:** Use small chunks (100-300 tokens) for precise retrieval, but return parent documents (500-2000 tokens) for generation context. The chunk-to-parent ratio is typically 3-10x.

## Variations

### Metadata Preservation

Preserve chunk metadata in parent documents:

```go
type EnhancedParentDoc struct {
    Document schema.Document
    Chunks   []schema.Document
    Metadata map[string]any
}
```

### Overlapping Chunks

Handle overlapping chunks in parent documents:

```go
func (pdr *ParentDocumentRetriever) MergeOverlappingChunks(chunks []schema.Document) schema.Document {
    // Merge overlapping chunks into parent
    return schema.Document{}
}
```

## Related Recipes

- [Reranking with Cohere Rerank](/docs/recipes/cohere-reranking) -- Improve retrieval quality with cross-encoder reranking
- [Advanced Code Splitting](/docs/recipes/code-splitting) -- Split documents intelligently at structural boundaries
