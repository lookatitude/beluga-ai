---
title: Qdrant
description: High-performance vector store using Qdrant's HTTP REST API.
---

The Qdrant provider implements the `vectorstore.VectorStore` interface using Qdrant's HTTP REST API. It avoids gRPC dependencies for broad compatibility while supporting cosine, dot-product, and Euclidean distance strategies.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/rag/vectorstore/providers/qdrant
```

Start Qdrant locally:

```bash
docker run -p 6333:6333 qdrant/qdrant
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
    _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/qdrant"
)

func main() {
    store, err := vectorstore.New("qdrant", config.ProviderConfig{
        BaseURL: "http://localhost:6333",
        Options: map[string]any{
            "collection": "my_documents",
            "dimension":  float64(1536),
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
| `BaseURL` | `string` | (required) | Qdrant server URL |
| `APIKey` | `string` | (optional) | API key for authentication |
| `Options["collection"]` | `string` | `documents` | Collection name |
| `Options["dimension"]` | `float64` | `1536` | Vector dimensionality |

## Collection Setup

Use `EnsureCollection` to create the collection with cosine distance:

```go
import (
    qdrantstore "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/qdrant"
)

store, err := qdrantstore.NewFromConfig(config.ProviderConfig{
    BaseURL: "http://localhost:6333",
    Options: map[string]any{
        "collection": "my_documents",
        "dimension":  float64(768),
    },
})
if err != nil {
    log.Fatal(err)
}

err = store.EnsureCollection(ctx)
if err != nil {
    log.Fatal(err)
}
```

## Direct Construction

```go
store := qdrantstore.New("http://localhost:6333",
    qdrantstore.WithCollection("my_documents"),
    qdrantstore.WithDimension(768),
    qdrantstore.WithAPIKey(os.Getenv("QDRANT_API_KEY")),
)
```

## Metadata Filtering

Qdrant supports payload-based filtering using its `must` filter clause:

```go
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithFilter(map[string]any{
        "lang":     "en",
        "category": "technical",
    }),
)
```

## Score Threshold

Set a minimum similarity score to exclude low-relevance results at the Qdrant API level:

```go
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithThreshold(0.7),
)
```

## Qdrant Cloud

Connect to Qdrant Cloud by providing the cluster URL and API key:

```go
store, err := vectorstore.New("qdrant", config.ProviderConfig{
    BaseURL: "https://abc123.us-east4-0.gcp.cloud.qdrant.io:6333",
    APIKey:  os.Getenv("QDRANT_API_KEY"),
    Options: map[string]any{
        "collection": "production",
    },
})
```

## Custom HTTP Client

```go
store := qdrantstore.New("http://localhost:6333",
    qdrantstore.WithHTTPClient(customClient),
    qdrantstore.WithCollection("my_documents"),
)
```
