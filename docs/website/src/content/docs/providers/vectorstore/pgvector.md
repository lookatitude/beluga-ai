---
title: pgvector
description: PostgreSQL-based vector store using the pgvector extension.
---

The pgvector provider implements the `vectorstore.VectorStore` interface using PostgreSQL with the [pgvector](https://github.com/pgvector/pgvector) extension. It uses `pgx` for connection management and supports cosine, dot-product, and Euclidean distance strategies.

Choose pgvector when you already operate PostgreSQL and want to add vector search without introducing a separate database. This avoids the operational overhead of managing a dedicated vector database while keeping your vectors co-located with relational data. pgvector supports HNSW and IVFFlat indexing for production-scale deployments.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/rag/vectorstore/providers/pgvector
```

Ensure the pgvector extension is installed in your PostgreSQL database:

```sql
CREATE EXTENSION IF NOT EXISTS vector;
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
    _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/pgvector"
)

func main() {
    store, err := vectorstore.New("pgvector", config.ProviderConfig{
        BaseURL: os.Getenv("DATABASE_URL"),
        Options: map[string]any{
            "table":     "documents",
            "dimension": float64(1536),
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Add documents with embeddings
    docs := []schema.Document{
        {ID: "doc1", Content: "Go is a statically typed language", Metadata: map[string]any{"lang": "en"}},
    }
    embeddings := [][]float32{make([]float32, 1536)} // Your actual embeddings

    err = store.Add(ctx, docs, embeddings)
    if err != nil {
        log.Fatal(err)
    }
}
```

## Configuration

| Parameter | Type | Default | Description |
|---|---|---|---|
| `BaseURL` | `string` | (required) | PostgreSQL connection string |
| `Options["table"]` | `string` | `documents` | Table name for storing vectors |
| `Options["dimension"]` | `float64` | `1536` | Vector dimensionality |

## Table Setup

Use `EnsureTable` to create the table and enable the vector extension automatically:

```go
import (
    pgvecstore "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/pgvector"
)

store, err := pgvecstore.NewFromConfig(config.ProviderConfig{
    BaseURL: os.Getenv("DATABASE_URL"),
    Options: map[string]any{
        "table":     "documents",
        "dimension": float64(1536),
    },
})
if err != nil {
    log.Fatal(err)
}

err = store.EnsureTable(ctx)
if err != nil {
    log.Fatal(err)
}
```

This creates:

```sql
CREATE EXTENSION IF NOT EXISTS vector;
CREATE TABLE IF NOT EXISTS documents (
    id TEXT PRIMARY KEY,
    embedding vector(1536),
    content TEXT,
    metadata JSONB
);
```

## Direct Construction

For more control, construct with a custom `pgx` pool:

```go
import (
    "github.com/jackc/pgx/v5/pgxpool"
    pgvecstore "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/pgvector"
)

pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
if err != nil {
    log.Fatal(err)
}

store := pgvecstore.New(pool,
    pgvecstore.WithTable("my_vectors"),
    pgvecstore.WithDimension(768),
)
```

## Search with Distance Strategies

pgvector supports three distance strategies with dedicated PostgreSQL operators:

```go
// Cosine similarity (default) -- uses <=> operator
results, err := store.Search(ctx, queryVec, 10)

// Dot product -- uses <#> operator
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithStrategy(vectorstore.DotProduct),
)

// Euclidean distance -- uses <-> operator
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithStrategy(vectorstore.Euclidean),
)
```

## Metadata Filtering

Filter search results by JSONB metadata fields:

```go
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithFilter(map[string]any{
        "lang": "en",
    }),
    vectorstore.WithThreshold(0.7),
)
```

## Upsert Behavior

The `Add` method uses `INSERT ... ON CONFLICT DO UPDATE`, so adding a document with an existing ID overwrites the previous entry.
