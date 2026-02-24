---
title: "Vespa Vector Store Provider"
description: "Vector search with Vespa engine in Beluga AI. Enterprise search platform with nearest-neighbor search, ranking, and hybrid retrieval in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Vespa, vector store, enterprise search, nearest-neighbor, hybrid retrieval, ranking, Go, Beluga AI"
---

The Vespa provider implements the `vectorstore.VectorStore` interface using Vespa's document and search APIs. Vespa is a production-grade search engine that supports real-time vector similarity search with YQL queries, combining keyword and vector search capabilities.

Choose Vespa when you need a production-grade search engine that combines vector search with structured queries, ranking, and real-time indexing. Vespa excels at large-scale deployments with complex ranking models and is well-suited for applications that need both keyword and semantic search with custom ranking logic.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/rag/vectorstore/providers/vespa
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
    _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/vespa"
)

func main() {
    store, err := vectorstore.New("vespa", config.ProviderConfig{
        BaseURL: "http://localhost:8080",
        Options: map[string]any{
            "namespace": "default",
            "doc_type":  "document",
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
| `BaseURL` | `string` | (required) | Vespa endpoint URL |
| `Options["namespace"]` | `string` | `default` | Vespa namespace |
| `Options["doc_type"]` | `string` | `document` | Vespa document type |

## Prerequisites

Configure a Vespa schema with a vector field:

```xml
schema document {
    document document {
        field content type string { indexing: summary | index }
        field embedding type tensor<float>(x[1536]) {
            indexing: attribute | index
            attribute { distance-metric: angular }
        }
    }
    rank-profile default {
        first-phase { expression: closeness(embedding) }
    }
}
```

## Direct Construction

```go
import (
    vespastore "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/vespa"
)

store := vespastore.New("http://localhost:8080",
    vespastore.WithNamespace("production"),
    vespastore.WithDocType("article"),
)
```

## Distance Strategies

Vespa supports multiple ranking profiles based on the distance strategy:

```go
// Cosine similarity (default) -- uses closeness(embedding)
results, err := store.Search(ctx, queryVec, 10)

// Dot product -- uses dotProduct(embedding)
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithStrategy(vectorstore.DotProduct),
)

// Euclidean distance -- uses euclidean(embedding)
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithStrategy(vectorstore.Euclidean),
)
```

## Metadata Filtering

Vespa supports YQL-based filtering for metadata fields:

```go
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithFilter(map[string]any{
        "lang": "en",
    }),
)
```

Filters are appended to the YQL query as `AND` conditions.

## Search with Threshold

```go
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithThreshold(0.5),
)
for _, doc := range results {
    fmt.Printf("ID: %s, Score: %.4f\n", doc.ID, doc.Score)
}
```

## Document API

Documents are stored via Vespa's document PUT API at:

```
/document/v1/{namespace}/{doc_type}/docid/{id}
```

The provider handles URL construction and document ID encoding automatically.

## Custom HTTP Client

```go
store := vespastore.New("http://localhost:8080",
    vespastore.WithHTTPClient(customClient),
    vespastore.WithNamespace("my_ns"),
    vespastore.WithDocType("my_type"),
)
```
