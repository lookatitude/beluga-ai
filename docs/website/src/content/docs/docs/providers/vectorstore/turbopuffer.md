---
title: "Turbopuffer Vector Store Provider"
description: "Serverless vector search with Turbopuffer in Beluga AI. Pay-per-query vector database with automatic scaling and metadata filtering in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Turbopuffer, serverless vector store, pay-per-query, auto-scaling, vector database, Go, Beluga AI"
---

The Turbopuffer provider implements the `vectorstore.VectorStore` interface using the Turbopuffer serverless vector database. Turbopuffer offers a simple API with automatic scaling and supports cosine, dot-product, and Euclidean distance metrics.

Choose Turbopuffer when you want a serverless vector database with minimal configuration and pay-per-use pricing. Turbopuffer automatically scales storage and compute, with no capacity planning required. Its simple namespace-based data model makes it easy to isolate data across tenants or environments.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/rag/vectorstore/providers/turbopuffer
```

## Quick Start

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/rag/vectorstore"
    "github.com/lookatitude/beluga-ai/schema"
    _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/turbopuffer"
)

func main() {
    store, err := vectorstore.New("turbopuffer", config.ProviderConfig{
        APIKey: os.Getenv("TURBOPUFFER_API_KEY"),
        Options: map[string]any{
            "namespace": "my_documents",
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    docs := []schema.Document{
        {ID: "doc1", Content: "Go is a statically typed language", Metadata: map[string]any{"lang": "en"}},
    }
    embeddings := [][]float32{make([]float32, 1536)}

    err = store.Add(ctx, docs, embeddings)
    if err != nil {
        log.Fatal(err)
    }
}
```

## Configuration

| Parameter | Type | Default | Description |
|---|---|---|---|
| `APIKey` | `string` | (optional) | Turbopuffer API key |
| `BaseURL` | `string` | `https://api.turbopuffer.com/v1` | API base URL |
| `Options["namespace"]` | `string` | `documents` | Namespace for data isolation |

## Direct Construction

```go
import (
    tpstore "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/turbopuffer"
)

store := tpstore.New(
    tpstore.WithAPIKey(os.Getenv("TURBOPUFFER_API_KEY")),
    tpstore.WithNamespace("production"),
)
```

## Distance Strategies

Turbopuffer supports multiple distance metrics, selectable via search options:

```go
// Cosine distance (default)
results, err := store.Search(ctx, queryVec, 10)

// Dot product
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithStrategy(vectorstore.DotProduct),
)

// Euclidean squared
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithStrategy(vectorstore.Euclidean),
)
```

## Metadata Filtering

Turbopuffer supports attribute-based filtering with `Eq` operator:

```go
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithFilter(map[string]any{
        "lang": "en",
    }),
)
```

Multiple filters are combined with an `And` operator.

## Distance Threshold

```go
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithThreshold(0.3),
)
```

## Delete Behavior

Turbopuffer deletes documents by upserting with null vectors. This is handled transparently by the `Delete` method:

```go
err := store.Delete(ctx, []string{"doc1", "doc2"})
if err != nil {
    log.Fatal(err)
}
```

## Custom HTTP Client

```go
store := tpstore.New(
    tpstore.WithHTTPClient(customClient),
    tpstore.WithAPIKey(os.Getenv("TURBOPUFFER_API_KEY")),
    tpstore.WithNamespace("my_namespace"),
)
```
