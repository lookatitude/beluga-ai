---
title: In-memory Vector Store
description: Use the in-memory vector store for rapid RAG prototyping with zero external dependencies.
---

Spinning up PostgreSQL with pgvector or a hosted vector database is necessary for production, but it slows down development when you need to test retrieval logic quickly. The in-memory vector store provides zero-dependency semantic search that runs entirely in RAM — ideal for prototyping, testing, and CI/CD pipelines. Because it implements the same `VectorStore` interface as pgvector, Pinecone, and other production stores, code written against it migrates to production with a one-line configuration change.

## What You Will Build

A local semantic search engine using the in-memory vector store, including document indexing, similarity search, and metadata filtering.

## Prerequisites

- Understanding of [embeddings](/tutorials/providers/multimodal-embeddings)
- A configured embedding provider (or a mock for zero-latency testing)

## Step 1: Initialize the Store

Both the embedder and vector store are created through their respective registries using `New()`. The blank imports trigger `init()` registration for each provider. The in-memory store requires no configuration (`config.ProviderConfig{}`) because it has no external dependencies — no connection strings, no table names, no credentials.

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/rag/embedding"
    _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/openai"
    "github.com/lookatitude/beluga-ai/rag/vectorstore"
    _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/inmemory"
    "github.com/lookatitude/beluga-ai/schema"
)

func main() {
    ctx := context.Background()

    embedder, err := embedding.New("openai", config.ProviderConfig{
        "api_key": os.Getenv("OPENAI_API_KEY"),
        "model":   "text-embedding-3-small",
    })
    if err != nil {
        log.Fatalf("create embedder: %v", err)
    }

    store, err := vectorstore.New("inmemory", config.ProviderConfig{})
    if err != nil {
        log.Fatalf("create store: %v", err)
    }

    fmt.Println("In-memory vector store initialized")
}
```

## Step 2: Add Documents

Create `schema.Document` objects with content and metadata, then add them to the store. Each document carries an embedding vector alongside its text content and metadata. The metadata fields (`source`, `topic`) enable filtering at query time, which narrows the search space before vector comparison and improves both relevance and performance.

```go
    // Define documents
    docs := []schema.Document{
        {
            ID:       "doc-1",
            Content:  "Beluga whales are white and live in the Arctic Ocean.",
            Metadata: map[string]any{"source": "encyclopedia", "topic": "animals"},
        },
        {
            ID:       "doc-2",
            Content:  "The beluga sturgeon is unrelated to the whale and produces caviar.",
            Metadata: map[string]any{"source": "encyclopedia", "topic": "animals"},
        },
        {
            ID:       "doc-3",
            Content:  "Go is a statically typed programming language designed at Google.",
            Metadata: map[string]any{"source": "docs", "topic": "programming"},
        },
    }

    // Embed and store documents
    for i := range docs {
        vec, err := embedder.EmbedQuery(ctx, docs[i].Content)
        if err != nil {
            log.Fatalf("embed doc %s: %v", docs[i].ID, err)
        }
        docs[i].Embedding = vec
    }

    ids, err := store.AddDocuments(ctx, docs)
    if err != nil {
        log.Fatalf("add documents: %v", err)
    }
    fmt.Printf("Indexed %d documents: %v\n", len(ids), ids)
```

## Step 3: Similarity Search

Search for documents similar to a query. The store computes cosine similarity between the query vector and all stored document vectors, returning the top-k results ranked by score. The `2` parameter limits results to the two most similar documents, which controls both response size and relevance — returning too many results dilutes the quality of context provided to the LLM.

```go
    query := "marine mammals in cold waters"
    queryVec, err := embedder.EmbedQuery(ctx, query)
    if err != nil {
        log.Fatalf("embed query: %v", err)
    }

    results, err := store.SimilaritySearch(ctx, queryVec, 2)
    if err != nil {
        log.Fatalf("search: %v", err)
    }

    fmt.Printf("\nQuery: %s\n", query)
    for _, doc := range results {
        fmt.Printf("  [%.4f] %s\n", doc.Score, doc.Content)
    }
```

## Step 4: Metadata Filtering

Filter results by metadata before the vector comparison. Metadata filtering is applied as a pre-filter that narrows the candidate set before similarity ranking. This is useful in multi-tenant applications where each tenant's documents should only match queries from that tenant, or when you need to restrict search to a specific document category.

```go
    // Search only within "programming" topic documents
    results, err = store.SimilaritySearch(ctx, queryVec, 2,
        vectorstore.WithFilter(map[string]any{
            "topic": "programming",
        }),
    )
```

## When to Use In-memory vs. External Stores

| Scenario | In-memory | pgvector/Pinecone |
|:---|:---|:---|
| Prototyping | Recommended | Overkill |
| Unit tests | Recommended | Slow, brittle |
| CI/CD | Recommended | Requires setup |
| < 50k documents | Works well | Optional |
| > 50k documents | Memory risk | Recommended |
| Multi-instance | Not suitable | Required |
| Persistence needed | Not suitable | Required |

## Migrating to Production

Switching from in-memory to pgvector is a one-line change — replace the provider name and configuration. This seamless migration is possible because both providers implement the identical `VectorStore` interface. All application code — document insertion, similarity search, metadata filtering — remains unchanged.

```go
// Development
store, _ := vectorstore.New("inmemory", config.ProviderConfig{})

// Production
store, _ := vectorstore.New("pgvector", config.ProviderConfig{
    "connection_string": os.Getenv("DATABASE_URL"),
    "table_name":        "documents",
})
```

The `VectorStore` interface is identical — all application code remains unchanged.

## Troubleshooting

**Search results seem random**: Verify your embedder is producing meaningful vectors. If using a mock embedder that returns constant vectors, all similarity scores will be identical.

**Out of memory**: The in-memory store holds all vectors in a Go slice. For datasets larger than ~50k documents, switch to a persistent vector store.

## Next Steps

- [pgvector Sharding](/tutorials/providers/pgvector-sharding) — Scale to millions of vectors
- [Hybrid Search](/tutorials/rag/hybrid-search) — Combine vector and keyword search
