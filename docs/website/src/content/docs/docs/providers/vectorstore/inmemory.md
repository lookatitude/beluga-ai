---
title: "In-Memory Vector Store Provider"
description: "Thread-safe in-memory vector store for testing in Beluga AI. Zero external dependencies with metadata filtering for development and CI in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "in-memory vector store, test vector store, development, thread-safe, Go, RAG testing, Beluga AI"
---

The in-memory vector store provider implements the `vectorstore.VectorStore` interface using a thread-safe map with linear-scan search. It supports all three distance strategies (cosine, dot-product, Euclidean) and requires no external dependencies.

Use the in-memory store for unit tests and development. It requires no setup, no external services, and provides deterministic behavior for test assertions. Pair it with the [in-memory embedder](/providers/embedding/inmemory) for a fully self-contained test setup with no network dependencies.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/rag/vectorstore/providers/inmemory
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/rag/vectorstore"
    "github.com/lookatitude/beluga-ai/schema"
    _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/inmemory"
)

func main() {
    store, err := vectorstore.New("inmemory", config.ProviderConfig{})
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    docs := []schema.Document{
        {ID: "doc1", Content: "Go is a statically typed language", Metadata: map[string]any{"lang": "en"}},
        {ID: "doc2", Content: "Python is a dynamically typed language", Metadata: map[string]any{"lang": "en"}},
    }
    embeddings := [][]float32{
        {0.1, 0.2, 0.3, 0.4},
        {0.5, 0.6, 0.7, 0.8},
    }

    err = store.Add(ctx, docs, embeddings)
    if err != nil {
        log.Fatal(err)
    }

    results, err := store.Search(ctx, []float32{0.1, 0.2, 0.3, 0.4}, 2)
    if err != nil {
        log.Fatal(err)
    }

    for _, doc := range results {
        fmt.Printf("ID: %s, Score: %.4f, Content: %s\n", doc.ID, doc.Score, doc.Content)
    }
}
```

## Configuration

No configuration parameters are required. The provider is instantiated with no options:

```go
store, err := vectorstore.New("inmemory", config.ProviderConfig{})
```

## Direct Construction

```go
import (
    memstore "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/inmemory"
)

store := memstore.New()
```

## Distance Strategies

The in-memory store supports all three distance strategies:

```go
// Cosine similarity (default)
results, err := store.Search(ctx, queryVec, 10)

// Dot product
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithStrategy(vectorstore.DotProduct),
)

// Euclidean (negated so higher = more similar)
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithStrategy(vectorstore.Euclidean),
)
```

## Metadata Filtering

Full metadata filtering support:

```go
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithFilter(map[string]any{
        "lang":     "en",
        "category": "technical",
    }),
)
```

## Threshold Filtering

```go
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithThreshold(0.8),
)
```

## Thread Safety

The in-memory store uses `sync.RWMutex` for safe concurrent access. Reads (Search) can execute in parallel, while writes (Add, Delete) hold an exclusive lock.

## Upsert Behavior

Adding a document with an existing ID overwrites the previous entry:

```go
err = store.Add(ctx, []schema.Document{
    {ID: "doc1", Content: "Updated content"},
}, [][]float32{newEmbedding})
```

## Testing Usage

The in-memory store is the recommended choice for unit tests:

```go
func TestSearchPipeline(t *testing.T) {
    store := memstore.New()

    ctx := context.Background()

    // Seed test data
    docs := []schema.Document{
        {ID: "1", Content: "test document", Metadata: map[string]any{"type": "test"}},
    }
    embeddings := [][]float32{{0.1, 0.2, 0.3}}
    err := store.Add(ctx, docs, embeddings)
    if err != nil {
        t.Fatal(err)
    }

    // Search
    results, err := store.Search(ctx, []float32{0.1, 0.2, 0.3}, 1)
    if err != nil {
        t.Fatal(err)
    }
    if len(results) != 1 {
        t.Fatalf("expected 1 result, got %d", len(results))
    }
    if results[0].ID != "1" {
        t.Fatalf("expected ID 1, got %s", results[0].ID)
    }
}
```

## Limitations

- **Linear scan**: Search time is O(n) where n is the number of documents. Not suitable for large-scale production use.
- **No persistence**: Data is lost when the process exits.
- **No indexing**: No approximate nearest neighbor (ANN) index structures.
