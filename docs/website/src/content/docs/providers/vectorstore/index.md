---
title: Vector Store Providers
description: Overview of all vector store providers available in Beluga AI v2.
---

Beluga AI v2 provides a unified `vectorstore.VectorStore` interface for storing and searching document embeddings. All providers register via `init()` and are instantiated through the global registry.

## Interface

```go
type VectorStore interface {
    Add(ctx context.Context, docs []schema.Document, embeddings [][]float32) error
    Search(ctx context.Context, query []float32, k int, opts ...SearchOption) ([]schema.Document, error)
    Delete(ctx context.Context, ids []string) error
}
```

## Registry Usage

```go
import (
    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/rag/vectorstore"

    // Register the provider you need via blank import
    _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/inmemory"
)

func main() {
    store, err := vectorstore.New("inmemory", config.ProviderConfig{})
    if err != nil {
        log.Fatal(err)
    }

    err = store.Add(ctx, docs, embeddings)
    if err != nil {
        log.Fatal(err)
    }

    results, err := store.Search(ctx, queryVec, 10,
        vectorstore.WithThreshold(0.7),
    )
    if err != nil {
        log.Fatal(err)
    }
}
```

## Available Providers

| Provider | Registry Name | Type | Distance Strategies |
|---|---|---|---|
| [pgvector](/providers/vectorstore/pgvector) | `pgvector` | PostgreSQL extension | Cosine, Dot Product, Euclidean |
| [Pinecone](/providers/vectorstore/pinecone) | `pinecone` | Managed cloud | Cosine |
| [Qdrant](/providers/vectorstore/qdrant) | `qdrant` | Self-hosted / cloud | Cosine, Dot Product, Euclidean |
| [Milvus](/providers/vectorstore/milvus) | `milvus` | Self-hosted / cloud | Cosine |
| [Weaviate](/providers/vectorstore/weaviate) | `weaviate` | Self-hosted / cloud | Cosine |
| [ChromaDB](/providers/vectorstore/chroma) | `chroma` | Self-hosted | Cosine |
| [Elasticsearch](/providers/vectorstore/elasticsearch) | `elasticsearch` | Self-hosted / cloud | Cosine |
| [Redis](/providers/vectorstore/redis) | `redis` | Self-hosted / cloud | Cosine |
| [MongoDB Atlas](/providers/vectorstore/mongodb) | `mongodb` | Managed cloud | Cosine |
| [SQLite-vec](/providers/vectorstore/sqlitevec) | `sqlitevec` | Embedded (CGO) | Euclidean (L2) |
| [Turbopuffer](/providers/vectorstore/turbopuffer) | `turbopuffer` | Managed cloud | Cosine, Dot Product, Euclidean |
| [Vespa](/providers/vectorstore/vespa) | `vespa` | Self-hosted / cloud | Cosine, Dot Product, Euclidean |
| [In-Memory](/providers/vectorstore/inmemory) | `inmemory` | In-process | Cosine, Dot Product, Euclidean |

## Search Options

All providers support the same functional options for search configuration:

```go
// Filter by metadata
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithFilter(map[string]any{"category": "technical"}),
)

// Set minimum similarity threshold
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithThreshold(0.8),
)

// Select distance strategy
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithStrategy(vectorstore.DotProduct),
)
```

## Provider Discovery

List all registered providers at runtime:

```go
names := vectorstore.List()
// Returns sorted list: ["chroma", "elasticsearch", "inmemory", ...]
```

## Hooks

Hooks allow observing vector store operations:

```go
// BeforeAdd is called before each Add operation
// AfterSearch is called after each Search operation
hooks := vectorstore.Hooks{
    BeforeAdd: func(ctx context.Context, docs []schema.Document) error {
        log.Printf("Adding %d documents", len(docs))
        return nil
    },
    AfterSearch: func(ctx context.Context, results []schema.Document, err error) {
        log.Printf("Search returned %d results", len(results))
    },
}
```

## Choosing a Provider

| Use Case | Recommended Provider |
|---|---|
| Testing and development | [In-Memory](/providers/vectorstore/inmemory) |
| Existing PostgreSQL deployment | [pgvector](/providers/vectorstore/pgvector) |
| Embedded/edge applications | [SQLite-vec](/providers/vectorstore/sqlitevec) |
| Managed cloud with zero ops | [Pinecone](/providers/vectorstore/pinecone), [Turbopuffer](/providers/vectorstore/turbopuffer) |
| Full-text + vector search | [Elasticsearch](/providers/vectorstore/elasticsearch), [Weaviate](/providers/vectorstore/weaviate) |
| High-performance self-hosted | [Qdrant](/providers/vectorstore/qdrant), [Milvus](/providers/vectorstore/milvus) |
| Existing Redis infrastructure | [Redis](/providers/vectorstore/redis) |
| Existing MongoDB Atlas | [MongoDB Atlas](/providers/vectorstore/mongodb) |
