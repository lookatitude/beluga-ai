---
title: ChromaDB
description: Vector store using ChromaDB's HTTP REST API.
---

The ChromaDB provider implements the `vectorstore.VectorStore` interface using ChromaDB's REST API. ChromaDB is an open-source embedding database with a simple API, supporting multi-tenant deployments.

Choose ChromaDB for its simplicity and ease of setup. ChromaDB is straightforward to run locally with Docker and requires minimal configuration, making it a good choice for prototyping and small-to-medium workloads. Its multi-tenant support (tenants and databases) allows data isolation without separate deployments.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/rag/vectorstore/providers/chroma
```

Start ChromaDB locally:

```bash
docker run -p 8000:8000 chromadb/chroma
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
    _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/chroma"
)

func main() {
    store, err := vectorstore.New("chroma", config.ProviderConfig{
        BaseURL: "http://localhost:8000",
        Options: map[string]any{
            "collection": "my_documents",
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
| `BaseURL` | `string` | (required) | ChromaDB server URL |
| `Options["collection"]` | `string` | (none) | Collection name |
| `Options["tenant"]` | `string` | `default_tenant` | Tenant name |
| `Options["database"]` | `string` | `default_database` | Database name |

## Collection Management

The provider automatically creates collections via `get_or_create` semantics. You can also explicitly ensure a collection exists:

```go
import (
    chromastore "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/chroma"
)

store := chromastore.New("http://localhost:8000",
    chromastore.WithCollection("my_collection"),
)

err := store.EnsureCollection(ctx)
if err != nil {
    log.Fatal(err)
}
```

## Direct Construction

```go
store := chromastore.New("http://localhost:8000",
    chromastore.WithCollection("my_collection"),
    chromastore.WithTenant("my_tenant"),
    chromastore.WithDatabase("my_database"),
)
```

## Pre-Resolved Collection ID

If you already know the collection ID, skip the resolution step:

```go
store := chromastore.New("http://localhost:8000",
    chromastore.WithCollectionID("abc-123-def-456"),
)
```

## Metadata Filtering

ChromaDB supports metadata filtering with its `$eq` operator:

```go
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithFilter(map[string]any{
        "lang": "en",
    }),
)
```

## Search with Threshold

ChromaDB returns distances (lower = more similar). The provider converts them to similarity scores using the formula `1 / (1 + distance)`:

```go
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithThreshold(0.5),
)
```

## Multi-Tenancy

ChromaDB supports multi-tenant deployments via the tenant and database parameters:

```go
store, err := vectorstore.New("chroma", config.ProviderConfig{
    BaseURL: "http://localhost:8000",
    Options: map[string]any{
        "collection": "shared_docs",
        "tenant":     "org_123",
        "database":   "prod",
    },
})
```
