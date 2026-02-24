---
title: "VectorStore Providers API — 13 Backends"
description: "VectorStore providers API for Beluga AI. 13 backends: pgvector, Pinecone, Qdrant, Weaviate, Milvus, Elasticsearch, Chroma, Redis, and more."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "VectorStore providers API, pgvector, Pinecone, Qdrant, Weaviate, Milvus, Elasticsearch, Chroma, Beluga AI, Go, reference"
---

## chroma

```go
import "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/chroma"
```

Package chroma provides a VectorStore backed by ChromaDB. It communicates
with ChromaDB via its HTTP REST API.

## Registration

The provider registers as "chroma" in the vectorstore registry:

```go
import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/chroma"

store, err := vectorstore.New("chroma", config.ProviderConfig{
    BaseURL: "http://localhost:8000",
    Options: map[string]any{
        "collection": "my_collection",
        "tenant":     "default_tenant",
        "database":   "default_database",
    },
})
```

## Configuration

ProviderConfig fields:
- BaseURL — ChromaDB server URL (required)
- Options["collection"] — collection name
- Options["tenant"] — tenant name (default: "default_tenant")
- Options["database"] — database name (default: "default_database")

## Collection Management

Use `Store.EnsureCollection` to create the collection if it does not exist.
The collection ID is resolved automatically on first Add or Search call.

---

## elasticsearch

```go
import "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/elasticsearch"
```

Package elasticsearch provides a VectorStore backed by Elasticsearch's kNN search.
It uses Elasticsearch's dense_vector field type and approximate kNN search.

## Registration

The provider registers as "elasticsearch" in the vectorstore registry:

```go
import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/elasticsearch"

store, err := vectorstore.New("elasticsearch", config.ProviderConfig{
    BaseURL: "http://localhost:9200",
    APIKey:  "optional-api-key",
    Options: map[string]any{
        "index":     "documents",
        "dimension": float64(1536),
    },
})
```

## Configuration

ProviderConfig fields:
- BaseURL — Elasticsearch server URL (required)
- APIKey — API key for authentication (optional)
- Options["index"] — index name (default: "documents")
- Options["dimension"] — vector dimension (default: 1536)

## Index Management

Use `Store.EnsureIndex` to create the Elasticsearch index with the
appropriate dense_vector mapping if it does not exist.

---

## inmemory

```go
import "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/inmemory"
```

Package inmemory provides an in-memory VectorStore for testing and
small-scale use. It uses linear scan with cosine similarity for search
and is safe for concurrent use.

## Registration

The provider registers as "inmemory" in the vectorstore registry:

```go
import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/inmemory"

store, err := vectorstore.New("inmemory", config.ProviderConfig{})
```

## Features

- Thread-safe with sync.RWMutex
- Supports cosine, dot-product, and Euclidean distance strategies
- Documents keyed by ID; re-adding overwrites the previous entry
- No external dependencies

---

## milvus

```go
import "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/milvus"
```

Package milvus provides a VectorStore backed by the Milvus vector database.
It communicates with Milvus via its REST API for broad compatibility.

## Registration

The provider registers as "milvus" in the vectorstore registry:

```go
import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/milvus"

store, err := vectorstore.New("milvus", config.ProviderConfig{
    BaseURL: "http://localhost:19530",
    Options: map[string]any{
        "collection": "documents",
        "dimension":  float64(1536),
    },
})
```

## Configuration

ProviderConfig fields:
- BaseURL — Milvus server URL (required)
- APIKey — API key for authentication (optional)
- Options["collection"] — collection name (default: "documents")
- Options["dimension"] — vector dimension (default: 1536)

---

## mongodb

```go
import "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/mongodb"
```

Package mongodb provides a VectorStore backed by MongoDB Atlas Vector Search.
It communicates with MongoDB via its HTTP Data API to avoid requiring the
full MongoDB Go driver as a dependency, and supports cosine, dot-product,
and Euclidean distance strategies.

## Registration

The provider registers as "mongodb" in the vectorstore registry:

```go
import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/mongodb"

store, err := vectorstore.New("mongodb", config.ProviderConfig{
    BaseURL: "https://data.mongodb-api.com/app/<app-id>/endpoint/data/v1",
    APIKey:  "your-api-key",
    Options: map[string]any{
        "database":   "my_db",
        "collection": "documents",
        "index":      "vector_index",
    },
})
```

## Configuration

ProviderConfig fields:
- BaseURL — MongoDB Data API endpoint (required)
- APIKey — API key for authentication (required)
- Options["database"] — database name (default: "beluga")
- Options["collection"] — collection name (default: "documents")
- Options["index"] — vector search index name (default: "vector_index")

---

## pgvector

```go
import "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/pgvector"
```

Package pgvector provides a VectorStore backed by PostgreSQL with the
pgvector extension. It uses pgx for connection management and supports
cosine, dot-product, and Euclidean distance strategies.

## Registration

The provider registers as "pgvector" in the vectorstore registry:

```go
import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/pgvector"

store, err := vectorstore.New("pgvector", config.ProviderConfig{
    BaseURL: "postgres://user:pass@localhost:5432/db",
    Options: map[string]any{
        "table":     "documents",
        "dimension": float64(1536),
    },
})
```

## Configuration

ProviderConfig fields:
- BaseURL — PostgreSQL connection string (required)
- Options["table"] — table name (default: "documents")
- Options["dimension"] — vector dimension (default: 1536)

## Table Management

Use `Store.EnsureTable` to create the documents table and pgvector
extension if they do not exist.

## Distance Operators

The provider maps [vectorstore.SearchStrategy] to pgvector SQL operators:
- Cosine — <=> (returns 1 - distance as similarity score)
- DotProduct — <#> (returns negated inner product)
- Euclidean — <-> (returns negated distance)

---

## pinecone

```go
import "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/pinecone"
```

Package pinecone provides a VectorStore backed by the Pinecone vector
database. It communicates with Pinecone via its REST API.

## Registration

The provider registers as "pinecone" in the vectorstore registry:

```go
import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/pinecone"

store, err := vectorstore.New("pinecone", config.ProviderConfig{
    APIKey:  "your-api-key",
    BaseURL: "https://index-name-project.svc.environment.pinecone.io",
    Options: map[string]any{
        "namespace": "my_namespace",
    },
})
```

## Configuration

ProviderConfig fields:
- APIKey — Pinecone API key (required)
- BaseURL — index endpoint URL (required)
- Options["namespace"] — namespace for document isolation

---

## qdrant

```go
import "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/qdrant"
```

Package qdrant provides a VectorStore backed by the Qdrant vector database.
It communicates with Qdrant via its HTTP REST API to avoid heavy gRPC
dependencies, and supports cosine, dot-product, and Euclidean distance.

## Registration

The provider registers as "qdrant" in the vectorstore registry:

```go
import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/qdrant"

store, err := vectorstore.New("qdrant", config.ProviderConfig{
    BaseURL: "http://localhost:6333",
    APIKey:  "optional-api-key",
    Options: map[string]any{
        "collection": "my_collection",
        "dimension":  float64(1536),
    },
})
```

## Configuration

ProviderConfig fields:
- BaseURL — Qdrant server URL (required)
- APIKey — API key for authentication (optional)
- Options["collection"] — collection name (required)
- Options["dimension"] — vector dimension (default: 1536)

---

## redis

```go
import "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/redis"
```

Package redis provides a VectorStore backed by Redis with the RediSearch module.
It uses Redis hashes to store documents and RediSearch's vector similarity
search for retrieval.

## Registration

The provider registers as "redis" in the vectorstore registry:

```go
import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/redis"

store, err := vectorstore.New("redis", config.ProviderConfig{
    BaseURL: "localhost:6379",
    Options: map[string]any{
        "index":     "idx:documents",
        "prefix":    "doc:",
        "dimension": float64(1536),
    },
})
```

## Configuration

ProviderConfig fields:
- BaseURL — Redis server address (required)
- Options["index"] — RediSearch index name (default: "idx:documents")
- Options["prefix"] — key prefix for document hashes (default: "doc:")
- Options["dimension"] — vector dimension (default: 1536)

---

## sqlitevec

```go
import "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/sqlitevec"
```

Package sqlitevec provides a VectorStore backed by SQLite with the
sqlite-vec extension for vector similarity search.

This provider requires CGO and the sqlite-vec extension. Build with:

```go
CGO_ENABLED=1 go build
```

## Registration

The provider registers as "sqlitevec" in the vectorstore registry:

```go
import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/sqlitevec"

store, err := vectorstore.New("sqlitevec", config.ProviderConfig{
    BaseURL: "/path/to/database.db",
    Options: map[string]any{
        "table":     "documents",
        "dimension": float64(1536),
    },
})
```

## Configuration

ProviderConfig fields:
- BaseURL — path to the SQLite database file (required)
- Options["table"] — table name (default: "documents")
- Options["dimension"] — vector dimension (default: 1536)

---

## turbopuffer

```go
import "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/turbopuffer"
```

Package turbopuffer provides a VectorStore backed by the Turbopuffer
serverless vector database. It communicates with Turbopuffer via its REST API.

## Registration

The provider registers as "turbopuffer" in the vectorstore registry:

```go
import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/turbopuffer"

store, err := vectorstore.New("turbopuffer", config.ProviderConfig{
    APIKey: "...",
    Options: map[string]any{
        "namespace": "my_namespace",
    },
})
```

## Configuration

ProviderConfig fields:
- APIKey — Turbopuffer API key (required)
- Options["namespace"] — namespace for document isolation

---

## vespa

```go
import "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/vespa"
```

Package vespa provides a VectorStore backed by the Vespa search engine.
It communicates with Vespa's document and search APIs via HTTP REST,
supporting cosine, dot-product, and Euclidean distance strategies.

## Registration

The provider registers as "vespa" in the vectorstore registry:

```go
import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/vespa"

store, err := vectorstore.New("vespa", config.ProviderConfig{
    BaseURL: "http://localhost:8080",
    Options: map[string]any{
        "namespace":  "my_namespace",
        "doc_type":   "my_doc_type",
    },
})
```

## Configuration

ProviderConfig fields:
- BaseURL — Vespa endpoint URL (required)
- Options["namespace"] — document namespace
- Options["doc_type"] — document type

---

## weaviate

```go
import "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/weaviate"
```

Package weaviate provides a VectorStore backed by the Weaviate vector database.
It communicates with Weaviate via its REST API using the internal httpclient.

## Registration

The provider registers as "weaviate" in the vectorstore registry:

```go
import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/weaviate"

store, err := vectorstore.New("weaviate", config.ProviderConfig{
    BaseURL: "http://localhost:8080",
    APIKey:  "optional-api-key",
    Options: map[string]any{
        "class": "Document",
    },
})
```

## Configuration

ProviderConfig fields:
- BaseURL — Weaviate server URL (required)
- APIKey — API key for authentication (optional)
- Options["class"] — Weaviate class name (default: "Document")
