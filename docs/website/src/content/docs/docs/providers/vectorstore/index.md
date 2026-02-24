---
title: "Vector Store Providers — 13 Backends"
description: "13 vector store providers with unified VectorStore interface: pgvector, Pinecone, Qdrant, Weaviate, Milvus, and more. Similarity search in Go with Beluga AI."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "vector store providers, Go vector database, pgvector, Pinecone, Qdrant, similarity search, RAG, Beluga AI"
---

Beluga AI v2 provides a unified `vectorstore.VectorStore` interface for storing and searching document embeddings. All providers register via `init()` and are instantiated through the global registry.

The unified interface means your RAG pipeline works identically across all 13 vector store backends. Start with the in-memory store for development, switch to pgvector or ChromaDB for local testing, and deploy to Pinecone or Qdrant in production -- all without changing your application code.

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
| [pgvector](/docs/providers/vectorstore/pgvector) | `pgvector` | PostgreSQL extension | Cosine, Dot Product, Euclidean |
| [Pinecone](/docs/providers/vectorstore/pinecone) | `pinecone` | Managed cloud | Cosine |
| [Qdrant](/docs/providers/vectorstore/qdrant) | `qdrant` | Self-hosted / cloud | Cosine, Dot Product, Euclidean |
| [Milvus](/docs/providers/vectorstore/milvus) | `milvus` | Self-hosted / cloud | Cosine |
| [Weaviate](/docs/providers/vectorstore/weaviate) | `weaviate` | Self-hosted / cloud | Cosine |
| [ChromaDB](/docs/providers/vectorstore/chroma) | `chroma` | Self-hosted | Cosine |
| [Elasticsearch](/docs/providers/vectorstore/elasticsearch) | `elasticsearch` | Self-hosted / cloud | Cosine |
| [Redis](/docs/providers/vectorstore/redis) | `redis` | Self-hosted / cloud | Cosine |
| [MongoDB Atlas](/docs/providers/vectorstore/mongodb) | `mongodb` | Managed cloud | Cosine |
| [SQLite-vec](/docs/providers/vectorstore/sqlitevec) | `sqlitevec` | Embedded (CGO) | Euclidean (L2) |
| [Turbopuffer](/docs/providers/vectorstore/turbopuffer) | `turbopuffer` | Managed cloud | Cosine, Dot Product, Euclidean |
| [Vespa](/docs/providers/vectorstore/vespa) | `vespa` | Self-hosted / cloud | Cosine, Dot Product, Euclidean |
| [In-Memory](/docs/providers/vectorstore/inmemory) | `inmemory` | In-process | Cosine, Dot Product, Euclidean |

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

| Use Case | Recommended Provider | Why |
|---|---|---|
| Testing and development | [In-Memory](/docs/providers/vectorstore/inmemory) | No setup required, deterministic |
| Existing PostgreSQL deployment | [pgvector](/docs/providers/vectorstore/pgvector) | Vectors co-located with relational data |
| Embedded/edge applications | [SQLite-vec](/docs/providers/vectorstore/sqlitevec) | File-based, no server needed |
| Managed cloud with zero ops | [Pinecone](/docs/providers/vectorstore/pinecone), [Turbopuffer](/docs/providers/vectorstore/turbopuffer) | Automatic scaling, no infrastructure |
| Full-text + vector search | [Elasticsearch](/docs/providers/vectorstore/elasticsearch), [Weaviate](/docs/providers/vectorstore/weaviate) | Combined keyword and semantic queries |
| High-performance self-hosted | [Qdrant](/docs/providers/vectorstore/qdrant), [Milvus](/docs/providers/vectorstore/milvus) | Purpose-built for vector workloads at scale |
| Existing Redis infrastructure | [Redis](/docs/providers/vectorstore/redis) | Low-latency, reuses existing deployment |
| Existing MongoDB Atlas | [MongoDB Atlas](/docs/providers/vectorstore/mongodb) | Vector search integrated with document DB |
| Production search engine | [Vespa](/docs/providers/vectorstore/vespa) | Real-time indexing, custom ranking models |
