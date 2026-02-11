---
title: SQLite-vec
description: Embedded vector store using SQLite with the sqlite-vec extension.
---

The SQLite-vec provider implements the `vectorstore.VectorStore` interface using SQLite with the [sqlite-vec](https://github.com/asg017/sqlite-vec) extension. It provides an embedded, zero-dependency vector store suitable for edge deployments and local applications.

Choose SQLite-vec when you need an embedded vector store with no external server dependencies. It is ideal for edge deployments, CLI tools, desktop applications, and scenarios where a file-based database is preferable to a network service. The tradeoff is that it requires CGO and has lower throughput than server-based alternatives.

**Note:** This provider requires CGO and the sqlite-vec extension.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/rag/vectorstore/providers/sqlitevec
```

Build with CGO enabled:

```bash
CGO_ENABLED=1 go build
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
    _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/sqlitevec"
)

func main() {
    store, err := vectorstore.New("sqlitevec", config.ProviderConfig{
        BaseURL: "./vectors.db",
        Options: map[string]any{
            "table":     "documents",
            "dimension": float64(768),
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    docs := []schema.Document{
        {ID: "doc1", Content: "Go is a statically typed language", Metadata: map[string]any{"lang": "en"}},
    }
    embeddings := [][]float32{make([]float32, 768)}

    err = store.Add(ctx, docs, embeddings)
    if err != nil {
        log.Fatal(err)
    }
}
```

## Configuration

| Parameter | Type | Default | Description |
|---|---|---|---|
| `BaseURL` | `string` | (required) | Path to SQLite database file |
| `Options["table"]` | `string` | `documents` | Base table name |
| `Options["dimension"]` | `float64` | `1536` | Vector dimensionality |

## Table Setup

Use `EnsureTable` to create both the metadata table and the virtual vector table:

```go
import (
    sqlitevecstore "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/sqlitevec"
)

store, err := sqlitevecstore.NewFromConfig(config.ProviderConfig{
    BaseURL: "./vectors.db",
    Options: map[string]any{
        "table":     "documents",
        "dimension": float64(768),
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

This creates two tables:

```sql
-- Metadata table
CREATE TABLE IF NOT EXISTS documents (
    id TEXT PRIMARY KEY,
    content TEXT NOT NULL,
    metadata TEXT
);

-- Virtual vector table using vec0
CREATE VIRTUAL TABLE IF NOT EXISTS vec_documents USING vec0(
    id TEXT PRIMARY KEY,
    embedding float[768]
);
```

## Direct Construction

Provide a custom `*sql.DB` connection:

```go
import (
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
    sqlitevecstore "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/sqlitevec"
)

db, err := sql.Open("sqlite3", "./my_vectors.db")
if err != nil {
    log.Fatal(err)
}

store, err := sqlitevecstore.New(
    sqlitevecstore.WithDB(db),
    sqlitevecstore.WithTable("my_docs"),
    sqlitevecstore.WithDimension(384),
)
if err != nil {
    log.Fatal(err)
}
```

## Search

SQLite-vec uses L2 (Euclidean) distance by default. Distances are converted to similarity scores using `1 / (1 + distance)`:

```go
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithThreshold(0.5),
)
```

## Metadata Filtering

Metadata filtering is applied in-memory after the vector search:

```go
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithFilter(map[string]any{
        "lang": "en",
    }),
)
```

## In-Memory Database

Use `:memory:` for a fully in-memory store (lost on process exit):

```go
store, err := vectorstore.New("sqlitevec", config.ProviderConfig{
    BaseURL: ":memory:",
    Options: map[string]any{
        "dimension": float64(384),
    },
})
```

## Build Requirements

- **CGO_ENABLED=1** is required at build time
- The sqlite-vec extension is automatically loaded via `sqlite_vec.Auto()` during `init()`
- Tests skip automatically when the `vec0` module is unavailable
