---
title: "Weaviate Vector Store Provider"
description: "Vector search with Weaviate in Beluga AI. AI-native vector database with GraphQL API, hybrid search, and multi-tenancy support in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Weaviate, vector store, vector database, GraphQL, hybrid search, multi-tenancy, Go, Beluga AI"
---

The Weaviate provider implements the `vectorstore.VectorStore` interface using Weaviate's REST and GraphQL APIs. Weaviate supports both vector and hybrid (keyword + vector) search with a schema-based data model.

Choose Weaviate when you need a schema-enforced data model with built-in hybrid search (combining keyword BM25 with vector similarity). Weaviate's GraphQL API and class-based organization make it well-suited for applications with structured document types and complex query patterns.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/rag/vectorstore/providers/weaviate
```

Start Weaviate locally:

```bash
docker run -p 8080:8080 semitechnologies/weaviate
```

## Quick Start

```go
package main

import (
    "context"
    "log"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/rag/vectorstore"
    "github.com/lookatitude/beluga-ai/schema"
    _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/weaviate"
)

func main() {
    store, err := vectorstore.New("weaviate", config.ProviderConfig{
        BaseURL: "http://localhost:8080",
        Options: map[string]any{
            "class": "Document",
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
| `BaseURL` | `string` | (required) | Weaviate server URL |
| `APIKey` | `string` | (optional) | API key for authentication |
| `Options["class"]` | `string` | `Document` | Weaviate class name |

## Direct Construction

```go
import (
    weaviatestore "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/weaviate"
)

store := weaviatestore.New("http://localhost:8080",
    weaviatestore.WithClass("Article"),
    weaviatestore.WithAPIKey(os.Getenv("WEAVIATE_API_KEY")),
)
```

## Search

The provider uses GraphQL `nearVector` queries for similarity search:

```go
results, err := store.Search(ctx, queryVec, 10)
for _, doc := range results {
    fmt.Printf("ID: %s, Score: %.4f, Content: %s\n", doc.ID, doc.Score, doc.Content)
}
```

## Metadata Filtering

Weaviate supports property-based filtering via its `where` clause:

```go
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithFilter(map[string]any{
        "lang": "en",
    }),
)
```

Multiple filters are combined with an `And` operator.

## Distance Threshold

Set a maximum distance threshold (converted from similarity):

```go
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithThreshold(0.8),
)
```

## Weaviate Cloud

Connect to Weaviate Cloud Services (WCS):

```go
store, err := vectorstore.New("weaviate", config.ProviderConfig{
    BaseURL: "https://my-cluster.weaviate.network",
    APIKey:  os.Getenv("WEAVIATE_API_KEY"),
    Options: map[string]any{
        "class": "Document",
    },
})
```

## ID Mapping

Weaviate requires UUID-format IDs. The provider automatically generates deterministic UUIDs from document ID strings, and stores the original ID in a `_beluga_id` property for round-trip retrieval.
