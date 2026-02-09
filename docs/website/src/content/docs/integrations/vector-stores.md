---
title: Vector Stores
description: Store and search document embeddings with pgvector, Qdrant, Pinecone, Weaviate, Milvus, and 8 more vector databases.
---

Beluga AI provides a unified `VectorStore` interface for storing document embeddings and performing similarity search. All vector store providers register via `init()` and support the same Add, Search, and Delete operations.

## Provider Overview

| Provider | Registry Name | Type | Import Path |
|----------|--------------|------|-------------|
| pgvector | `pgvector` | PostgreSQL extension | `rag/vectorstore/providers/pgvector` |
| Qdrant | `qdrant` | Dedicated vector DB | `rag/vectorstore/providers/qdrant` |
| Pinecone | `pinecone` | Managed serverless | `rag/vectorstore/providers/pinecone` |
| Weaviate | `weaviate` | Vector + graph DB | `rag/vectorstore/providers/weaviate` |
| Milvus | `milvus` | Distributed vector DB | `rag/vectorstore/providers/milvus` |
| ChromaDB | `chroma` | Lightweight vector DB | `rag/vectorstore/providers/chroma` |
| Redis | `redis` | In-memory + vector search | `rag/vectorstore/providers/redis` |
| Elasticsearch | `elasticsearch` | Full-text + vector search | `rag/vectorstore/providers/elasticsearch` |
| MongoDB | `mongodb` | Atlas Vector Search | `rag/vectorstore/providers/mongodb` |
| SQLite-vec | `sqlitevec` | Embedded vector search | `rag/vectorstore/providers/sqlitevec` |
| Turbopuffer | `turbopuffer` | Serverless vector DB | `rag/vectorstore/providers/turbopuffer` |
| Vespa | `vespa` | Search + vector engine | `rag/vectorstore/providers/vespa` |
| In-Memory | `inmemory` | Testing and prototyping | `rag/vectorstore/providers/inmemory` |

## VectorStore Interface

All providers implement:

```go
type VectorStore interface {
    // Add inserts documents with their corresponding embeddings.
    Add(ctx context.Context, docs []schema.Document, embeddings [][]float32) error

    // Search finds the k most similar documents to the query vector.
    Search(ctx context.Context, query []float32, k int, opts ...SearchOption) ([]schema.Document, error)

    // Delete removes documents by ID.
    Delete(ctx context.Context, ids []string) error
}
```

## Common Pattern

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/rag/embedding"
    "github.com/lookatitude/beluga-ai/rag/vectorstore"
    "github.com/lookatitude/beluga-ai/schema"

    _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/openai"
    _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/pgvector"
)

func main() {
    ctx := context.Background()

    // Create embedder
    emb, err := embedding.New("openai", config.ProviderConfig{
        APIKey: "sk-...",
        Model:  "text-embedding-3-small",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Create vector store
    store, err := vectorstore.New("pgvector", config.ProviderConfig{
        Options: map[string]any{
            "connection_string": "postgres://user:pass@localhost:5432/mydb",
            "table_name":        "documents",
            "dimensions":        1536.0,
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    // Embed and store documents
    docs := []schema.Document{
        {ID: "1", Content: "Go is a statically typed language.", Metadata: map[string]any{"topic": "go"}},
        {ID: "2", Content: "Python is dynamically typed.", Metadata: map[string]any{"topic": "python"}},
    }

    vectors, err := emb.Embed(ctx, []string{docs[0].Content, docs[1].Content})
    if err != nil {
        log.Fatal(err)
    }

    if err := store.Add(ctx, docs, vectors); err != nil {
        log.Fatal(err)
    }

    // Search
    queryVec, err := emb.EmbedSingle(ctx, "static typing")
    if err != nil {
        log.Fatal(err)
    }

    results, err := store.Search(ctx, queryVec, 5, vectorstore.WithThreshold(0.7))
    if err != nil {
        log.Fatal(err)
    }

    for _, doc := range results {
        fmt.Printf("%.3f: %s\n", doc.Score, doc.Content)
    }
}
```

## pgvector (PostgreSQL)

pgvector adds vector similarity search to PostgreSQL. Use it when you already have a PostgreSQL deployment and want to avoid adding a separate vector database.

```bash
# Enable the extension in PostgreSQL
CREATE EXTENSION IF NOT EXISTS vector;
```

```go
import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/pgvector"

store, err := vectorstore.New("pgvector", config.ProviderConfig{
    Options: map[string]any{
        "connection_string": "postgres://user:pass@localhost:5432/mydb",
        "table_name":        "documents",
        "dimensions":        1536.0,
    },
})
```

| Option | Type | Description |
|--------|------|-------------|
| `connection_string` | `string` | PostgreSQL connection URI |
| `table_name` | `string` | Table for document storage |
| `dimensions` | `float64` | Vector dimension (must match embedder) |

## Qdrant

Qdrant is a dedicated vector database with filtering, payload indexing, and clustering support.

```go
import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/qdrant"

store, err := vectorstore.New("qdrant", config.ProviderConfig{
    Options: map[string]any{
        "url":             "http://localhost:6333",
        "api_key":         os.Getenv("QDRANT_API_KEY"),
        "collection_name": "documents",
        "dimensions":      1536.0,
    },
})
```

For Qdrant Cloud, use your cluster URL and API key. For local development, run Qdrant via Docker:

```bash
docker run -p 6333:6333 qdrant/qdrant
```

## Pinecone

Pinecone provides fully managed, serverless vector search.

```go
import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/pinecone"

store, err := vectorstore.New("pinecone", config.ProviderConfig{
    APIKey: os.Getenv("PINECONE_API_KEY"),
    Options: map[string]any{
        "index_name":  "my-index",
        "environment": "us-east-1-aws",
        "namespace":   "production",
    },
})
```

## Weaviate

Weaviate combines vector search with graph-based data modeling.

```go
import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/weaviate"

store, err := vectorstore.New("weaviate", config.ProviderConfig{
    Options: map[string]any{
        "url":        "http://localhost:8080",
        "api_key":    os.Getenv("WEAVIATE_API_KEY"),
        "class_name": "Document",
    },
})
```

## Milvus

Milvus is a distributed vector database designed for billion-scale workloads.

```go
import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/milvus"

store, err := vectorstore.New("milvus", config.ProviderConfig{
    Options: map[string]any{
        "address":         "localhost:19530",
        "collection_name": "documents",
        "dimensions":      1536.0,
    },
})
```

## ChromaDB

ChromaDB is a lightweight, developer-friendly vector database.

```go
import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/chroma"

store, err := vectorstore.New("chroma", config.ProviderConfig{
    Options: map[string]any{
        "url":             "http://localhost:8000",
        "collection_name": "documents",
    },
})
```

## Redis

Redis Stack provides vector search alongside your existing cache layer.

```go
import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/redis"

store, err := vectorstore.New("redis", config.ProviderConfig{
    Options: map[string]any{
        "address":    "localhost:6379",
        "index_name": "doc-idx",
        "dimensions": 1536.0,
    },
})
```

## Elasticsearch

Elasticsearch 8+ provides dense vector search alongside full-text search.

```go
import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/elasticsearch"

store, err := vectorstore.New("elasticsearch", config.ProviderConfig{
    Options: map[string]any{
        "addresses":  []any{"http://localhost:9200"},
        "index_name": "documents",
        "dimensions": 1536.0,
    },
})
```

## MongoDB Atlas

MongoDB Atlas Vector Search integrates vectors into your document database.

```go
import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/mongodb"

store, err := vectorstore.New("mongodb", config.ProviderConfig{
    Options: map[string]any{
        "uri":             "mongodb+srv://...",
        "database":        "mydb",
        "collection_name": "documents",
        "index_name":      "vector_index",
    },
})
```

## SQLite-vec

SQLite-vec provides embedded vector search for edge deployments and single-node applications. Requires CGO and the `sqlite-vec` extension.

```go
import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/sqlitevec"

store, err := vectorstore.New("sqlitevec", config.ProviderConfig{
    Options: map[string]any{
        "path":       "./data/vectors.db",
        "table_name": "documents",
        "dimensions": 384.0,
    },
})
```

## In-Memory

The in-memory store is useful for testing, prototyping, and small datasets. Data is not persisted.

```go
import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/inmemory"

store, err := vectorstore.New("inmemory", config.ProviderConfig{})
```

## Search Options

All providers support the same search options:

```go
// Filter by metadata
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithFilter(map[string]any{"topic": "go"}),
)

// Set minimum similarity threshold
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithThreshold(0.75),
)

// Change distance strategy
results, err := store.Search(ctx, queryVec, 10,
    vectorstore.WithStrategy(vectorstore.DotProduct),
)
```

## Choosing a Vector Store

| Need | Recommended |
|------|------------|
| Already using PostgreSQL | pgvector |
| Managed, serverless | Pinecone |
| Advanced filtering | Qdrant |
| Billion-scale | Milvus |
| Full-text + vector | Elasticsearch |
| Document database | MongoDB Atlas |
| Embedded / edge | SQLite-vec |
| Development / testing | In-Memory |
| Existing Redis infra | Redis |

Key considerations:
- **Dimension alignment**: Your vector store dimension must match your embedder output. Mismatches cause runtime errors.
- **Distance metric**: Most providers default to cosine similarity. Ensure your index metric matches `vectorstore.Cosine`, `vectorstore.DotProduct`, or `vectorstore.Euclidean`.
- **Metadata indexing**: Filter performance depends on the provider's metadata index support. Qdrant, Pinecone, and Weaviate have first-class metadata filtering.
