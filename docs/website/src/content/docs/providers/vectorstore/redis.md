---
title: Redis
description: Vector store using Redis with the RediSearch module for vector similarity search.
---

The Redis provider implements the `vectorstore.VectorStore` interface using Redis hashes with the RediSearch module for vector similarity search. It stores documents as Redis hashes and leverages RediSearch's KNN vector search capabilities.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/rag/vectorstore/providers/redis
```

Start Redis with RediSearch:

```bash
docker run -p 6379:6379 redis/redis-stack
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
    _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/redis"
)

func main() {
    store, err := vectorstore.New("redis", config.ProviderConfig{
        BaseURL: "localhost:6379",
        Options: map[string]any{
            "index":     "idx:documents",
            "prefix":    "doc:",
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
| `BaseURL` | `string` | `localhost:6379` | Redis server address |
| `Options["index"]` | `string` | `idx:documents` | RediSearch index name |
| `Options["prefix"]` | `string` | `doc:` | Key prefix for document hashes |
| `Options["dimension"]` | `float64` | `1536` | Vector dimensionality |

## Index Setup

Use `EnsureIndex` to create the RediSearch index:

```go
import (
    redisstore "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/redis"
)

store := redisstore.New("localhost:6379",
    redisstore.WithIndex("idx:my_docs"),
    redisstore.WithPrefix("mydoc:"),
    redisstore.WithDimension(768),
)

err := store.EnsureIndex(ctx)
if err != nil {
    log.Fatal(err)
}
```

This creates a RediSearch index with:
- `content` as a TEXT field
- `embedding` as a VECTOR field using FLAT indexing with COSINE distance

If the index already exists, the call succeeds silently.

## Direct Construction

```go
store := redisstore.New("localhost:6379",
    redisstore.WithIndex("idx:vectors"),
    redisstore.WithPrefix("vec:"),
    redisstore.WithDimension(1024),
)
```

## Custom Redis Client

Provide a custom Redis client for advanced connection configuration:

```go
import goredis "github.com/redis/go-redis/v9"

client := goredis.NewClient(&goredis.Options{
    Addr:     "localhost:6379",
    Password: os.Getenv("REDIS_PASSWORD"),
    DB:       0,
})

store := redisstore.New("",
    redisstore.WithClient(client),
    redisstore.WithIndex("idx:my_docs"),
    redisstore.WithDimension(1536),
)
```

## Metadata Filtering

Redis supports tag-based filtering in the KNN search query:

```go
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithFilter(map[string]any{
        "lang": "en",
    }),
)
```

Filters are translated to RediSearch filter expressions (e.g., `@lang:{en}`).

## Search with Threshold

Cosine distance is converted to similarity: `1.0 - distance`:

```go
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithThreshold(0.8),
)
```

## Data Storage Format

Documents are stored as Redis hashes with the following fields:
- `content`: Document text content
- `embedding`: Binary-encoded float32 vector
- Custom metadata fields as string values

Keys follow the pattern `{prefix}{document_id}` (e.g., `doc:my-document`).
