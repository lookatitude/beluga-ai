---
title: MongoDB Atlas
description: Vector store using MongoDB Atlas Vector Search via the Data API.
---

The MongoDB Atlas provider implements the `vectorstore.VectorStore` interface using MongoDB Atlas Vector Search via the HTTP Data API. It supports cosine similarity search with metadata filtering through the `$vectorSearch` aggregation stage.

Choose MongoDB Atlas when your application already uses MongoDB and you want to add vector search alongside your existing document data. Atlas Vector Search integrates with MongoDB's aggregation pipeline, allowing you to combine vector similarity with traditional document queries without managing a separate vector database.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/rag/vectorstore/providers/mongodb
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
    _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/mongodb"
)

func main() {
    store, err := vectorstore.New("mongodb", config.ProviderConfig{
        BaseURL: os.Getenv("MONGODB_DATA_API_URL"),
        APIKey:  os.Getenv("MONGODB_API_KEY"),
        Options: map[string]any{
            "database":   "my_db",
            "collection": "documents",
            "index":      "vector_index",
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
| `BaseURL` | `string` | (required) | MongoDB Data API endpoint |
| `APIKey` | `string` | (optional) | Data API key |
| `Options["database"]` | `string` | `beluga` | Database name |
| `Options["collection"]` | `string` | `documents` | Collection name |
| `Options["index"]` | `string` | `vector_index` | Atlas Vector Search index name |

## Prerequisites

1. Create a MongoDB Atlas cluster
2. Enable the Data API for your cluster
3. Create an Atlas Vector Search index on the `embedding` field:

```json
{
  "fields": [
    {
      "type": "vector",
      "path": "embedding",
      "numDimensions": 1536,
      "similarity": "cosine"
    }
  ]
}
```

## Direct Construction

```go
import (
    mongostore "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/mongodb"
)

store := mongostore.New(
    os.Getenv("MONGODB_DATA_API_URL"),
    mongostore.WithDatabase("production"),
    mongostore.WithCollection("docs"),
    mongostore.WithIndex("prod_vector_index"),
    mongostore.WithAPIKey(os.Getenv("MONGODB_API_KEY")),
)
```

## Metadata Filtering

Metadata filters are applied to the `metadata` field path:

```go
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithFilter(map[string]any{
        "lang": "en",
    }),
)
```

This translates to a MongoDB filter on `metadata.lang`.

## Search with Threshold

```go
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithThreshold(0.7),
)
for _, doc := range results {
    fmt.Printf("ID: %s, Score: %.4f, Content: %s\n", doc.ID, doc.Score, doc.Content)
}
```

## Document Structure

Documents are stored with the following structure:

```json
{
  "_id": "doc1",
  "content": "Document text content",
  "embedding": [0.1, 0.2, ...],
  "metadata": {
    "lang": "en",
    "category": "technical"
  }
}
```

## Custom HTTP Client

```go
store := mongostore.New(
    os.Getenv("MONGODB_DATA_API_URL"),
    mongostore.WithHTTPClient(customClient),
    mongostore.WithDatabase("my_db"),
)
```
