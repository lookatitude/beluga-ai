---
title: "Pinecone Vector Store Provider"
description: "Managed vector search with Pinecone in Beluga AI. Serverless vector database with namespace isolation, metadata filtering, and fast queries in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Pinecone, vector database, managed vector store, serverless, metadata filtering, Go, RAG, Beluga AI"
---

The Pinecone provider implements the `vectorstore.VectorStore` interface using Pinecone's REST API. Pinecone is a fully managed vector database with automatic scaling, requiring no infrastructure management.

Choose Pinecone when you want a fully managed vector database with zero operational overhead. Pinecone handles scaling, replication, and index optimization automatically. It is well-suited for teams that want to focus on application logic rather than infrastructure, and supports namespaces for multi-tenant data isolation.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/rag/vectorstore/providers/pinecone
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
    _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/pinecone"
)

func main() {
    store, err := vectorstore.New("pinecone", config.ProviderConfig{
        APIKey:  os.Getenv("PINECONE_API_KEY"),
        BaseURL: os.Getenv("PINECONE_INDEX_HOST"),
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
| `APIKey` | `string` | (required) | Pinecone API key |
| `BaseURL` | `string` | (required) | Index host URL (e.g., `https://index-name-project.svc.environment.pinecone.io`) |
| `Options["namespace"]` | `string` | (none) | Pinecone namespace for data isolation |

## Direct Construction

```go
import (
    pineconestore "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/pinecone"
)

store := pineconestore.New(
    "https://my-index-abc123.svc.us-east1-gcp.pinecone.io",
    os.Getenv("PINECONE_API_KEY"),
    pineconestore.WithNamespace("production"),
)
```

## Namespaces

Use namespaces to partition data within a single index:

```go
store, err := vectorstore.New("pinecone", config.ProviderConfig{
    APIKey:  os.Getenv("PINECONE_API_KEY"),
    BaseURL: os.Getenv("PINECONE_INDEX_HOST"),
    Options: map[string]any{
        "namespace": "user_123",
    },
})
```

## Metadata Filtering

Pinecone supports metadata filtering using its native `$eq` operator:

```go
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithFilter(map[string]any{
        "lang":     "en",
        "category": "technical",
    }),
)
```

## Search with Threshold

```go
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithThreshold(0.8),
)
for _, doc := range results {
    fmt.Printf("ID: %s, Score: %.4f, Content: %s\n", doc.ID, doc.Score, doc.Content)
}
```

## Content Storage

Document content is stored in Pinecone metadata under the `content` key. Custom metadata fields are stored alongside it. On retrieval, content is extracted back into the `Document.Content` field automatically.

## Custom HTTP Client

```go
store := pineconestore.New(
    baseURL, apiKey,
    pineconestore.WithHTTPClient(customClient),
)
```
