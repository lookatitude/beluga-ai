---
title: "RAG Vector Store"
description: "VectorStore interface for similarity search over document embeddings"
---

```go
import "github.com/lookatitude/beluga-ai/rag/vectorstore"
```

Package vectorstore provides the VectorStore interface and registry for
storing and searching document embeddings. VectorStores are the storage
backend for the RAG pipeline, supporting similarity search over embedded
documents.

## Interface

The core interface is `VectorStore`:

```go
type VectorStore interface {
    Add(ctx context.Context, docs []schema.Document, embeddings [][]float32) error
    Search(ctx context.Context, query []float32, k int, opts ...SearchOption) ([]schema.Document, error)
    Delete(ctx context.Context, ids []string) error
}
```

Implementations must be safe for concurrent use.

## Registry

The package follows Beluga's registry pattern. Providers register via
init() and are instantiated with `New`:

```go
store, err := vectorstore.New("inmemory", cfg)
if err != nil {
    log.Fatal(err)
}

err = store.Add(ctx, docs, embeddings)
if err != nil {
    log.Fatal(err)
}

results, err := store.Search(ctx, queryVec, 10, vectorstore.WithThreshold(0.7))
if err != nil {
    log.Fatal(err)
}
```

Use `List` to discover all registered provider names.

## Search Options

Search behaviour is configurable via functional options:
- [WithFilter] — restrict results by metadata key-value pairs
- [WithThreshold] — set minimum similarity score
- [WithStrategy] — select distance metric ([Cosine], [DotProduct], [Euclidean])

## Providers

Available providers (imported via blank import):
- "inmemory" — in-memory store for testing and small-scale use
- "pgvector" — PostgreSQL with pgvector extension
- "chroma" — ChromaDB
- "elasticsearch" — Elasticsearch kNN search
- "milvus" — Milvus vector database
- "mongodb" — MongoDB Atlas Vector Search
- "pinecone" — Pinecone vector database
- "qdrant" — Qdrant vector database
- "redis" — Redis with RediSearch module
- "sqlitevec" — SQLite with sqlite-vec extension (requires CGO)
- "turbopuffer" — Turbopuffer serverless vector database
- "vespa" — Vespa search engine
- "weaviate" — Weaviate vector database

## Middleware and Hooks

Cross-cutting concerns are layered via `Middleware` and `Hooks`:

```go
store = vectorstore.ApplyMiddleware(store,
    vectorstore.WithHooks(vectorstore.Hooks{
        BeforeAdd: func(ctx context.Context, docs []schema.Document) error {
            log.Printf("adding %d documents", len(docs))
            return nil
        },
        AfterSearch: func(ctx context.Context, results []schema.Document, err error) {
            log.Printf("search returned %d results", len(results))
        },
    }),
)
```

## Custom Provider

To add a custom vector store provider:

```go
func init() {
    vectorstore.Register("custom", func(cfg config.ProviderConfig) (vectorstore.VectorStore, error) {
        return &myStore{connStr: cfg.BaseURL}, nil
    })
}
```
