---
title: Milvus
description: Scalable vector store using Milvus's REST API.
---

The Milvus provider implements the `vectorstore.VectorStore` interface using the Milvus v2 REST API. Milvus is an open-source vector database designed for scalable similarity search, supporting both self-hosted and managed (Zilliz Cloud) deployments.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/rag/vectorstore/providers/milvus
```

Start Milvus locally:

```bash
docker compose up -d  # Using the official Milvus docker-compose
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
    _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/milvus"
)

func main() {
    store, err := vectorstore.New("milvus", config.ProviderConfig{
        BaseURL: "http://localhost:19530",
        Options: map[string]any{
            "collection": "documents",
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
| `BaseURL` | `string` | (required) | Milvus server URL |
| `APIKey` | `string` | (optional) | API key for Zilliz Cloud |
| `Options["collection"]` | `string` | `documents` | Collection name |
| `Options["dimension"]` | `float64` | `1536` | Vector dimensionality |

## Direct Construction

```go
import (
    milvusstore "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/milvus"
)

store := milvusstore.New("http://localhost:19530",
    milvusstore.WithCollection("my_collection"),
    milvusstore.WithDimension(768),
)
```

## Metadata Filtering

Milvus supports expression-based filtering:

```go
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithFilter(map[string]any{
        "lang": "en",
    }),
)
```

Filters are translated to Milvus filter expressions (e.g., `lang == "en"`).

## Zilliz Cloud

Connect to Zilliz Cloud (managed Milvus) using the API key:

```go
store, err := vectorstore.New("milvus", config.ProviderConfig{
    BaseURL: "https://in03-abc123.api.gcp-us-west1.zillizcloud.com",
    APIKey:  os.Getenv("ZILLIZ_API_KEY"),
    Options: map[string]any{
        "collection": "production",
    },
})
```

## Search with Threshold

```go
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithThreshold(0.7),
)
for _, doc := range results {
    fmt.Printf("ID: %s, Score: %.4f\n", doc.ID, doc.Score)
}
```

## Custom HTTP Client

```go
store := milvusstore.New("http://localhost:19530",
    milvusstore.WithHTTPClient(customClient),
    milvusstore.WithCollection("my_collection"),
)
```
