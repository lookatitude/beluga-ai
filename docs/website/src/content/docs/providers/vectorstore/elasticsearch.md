---
title: Elasticsearch
description: Vector store using Elasticsearch's kNN search with dense_vector fields.
---

The Elasticsearch provider implements the `vectorstore.VectorStore` interface using Elasticsearch's approximate kNN search with `dense_vector` fields. It uses the bulk API for efficient batch operations.

Choose Elasticsearch when you need both full-text search and vector similarity search in the same engine. If your organization already runs Elasticsearch for logging, analytics, or text search, adding vector search avoids introducing a separate system. Elasticsearch's kNN search integrates with its existing query DSL for combined keyword and vector queries.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/rag/vectorstore/providers/elasticsearch
```

Start Elasticsearch locally:

```bash
docker run -p 9200:9200 -e "discovery.type=single-node" elasticsearch:8.12.0
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
    _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/elasticsearch"
)

func main() {
    store, err := vectorstore.New("elasticsearch", config.ProviderConfig{
        BaseURL: "http://localhost:9200",
        Options: map[string]any{
            "index":     "documents",
            "dimension": float64(1536),
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
| `BaseURL` | `string` | (required) | Elasticsearch URL |
| `APIKey` | `string` | (optional) | API key for authentication |
| `Options["index"]` | `string` | `documents` | Index name |
| `Options["dimension"]` | `float64` | `1536` | Vector dimensionality |

## Index Setup

Use `EnsureIndex` to create the index with the correct mapping:

```go
import (
    esstore "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/elasticsearch"
)

store, err := esstore.NewFromConfig(config.ProviderConfig{
    BaseURL: "http://localhost:9200",
    Options: map[string]any{
        "index":     "documents",
        "dimension": float64(768),
    },
})
if err != nil {
    log.Fatal(err)
}

err = store.EnsureIndex(ctx)
if err != nil {
    log.Fatal(err)
}
```

This creates an index with the following mapping:

```json
{
  "mappings": {
    "properties": {
      "content": { "type": "text" },
      "embedding": {
        "type": "dense_vector",
        "dims": 768,
        "index": true,
        "similarity": "cosine"
      }
    }
  }
}
```

## Direct Construction

```go
store := esstore.New("http://localhost:9200",
    esstore.WithIndex("my_vectors"),
    esstore.WithDimension(768),
    esstore.WithAPIKey(os.Getenv("ES_API_KEY")),
)
```

## Bulk Operations

The provider uses the Elasticsearch `_bulk` API for both `Add` and `Delete` operations, enabling efficient batch processing.

## Metadata Filtering

Elasticsearch filters are built as `term` queries in a `bool/must` clause:

```go
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithFilter(map[string]any{
        "lang": "en",
    }),
)
```

## Similarity Threshold

Set a minimum similarity score at the kNN query level:

```go
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithThreshold(0.7),
)
```

## Elastic Cloud

Connect to Elastic Cloud with an API key:

```go
store, err := vectorstore.New("elasticsearch", config.ProviderConfig{
    BaseURL: "https://my-deployment.es.us-east-1.aws.elastic.cloud:9243",
    APIKey:  os.Getenv("ELASTIC_API_KEY"),
    Options: map[string]any{
        "index": "production_vectors",
    },
})
```

## Custom HTTP Client

```go
store := esstore.New("http://localhost:9200",
    esstore.WithHTTPClient(customClient),
    esstore.WithIndex("my_vectors"),
)
```
