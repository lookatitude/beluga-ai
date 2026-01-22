# Production pgvector Sharding

In this tutorial, you'll learn how to configure PostgreSQL with `pgvector` for high-scale, production-grade vector storage, including sharding strategies for handling millions of vectors.

## Learning Objectives

- ✅ Configure Beluga AI with `pgvector`
- ✅ Understand HNSW indexing parameters
- ✅ Implement table partitioning (sharding)
- ✅ Optimize query performance

## Prerequisites

- PostgreSQL with `pgvector` extension installed
- Basic Vector Store knowledge (see [Local Development](./vectorstores-inmemory-local.md))
- Go 1.24+

## Why Sharding?

As your vector dataset grows (1M+ vectors), a single table scan becomes too slow, and a single index might not fit in RAM.
- **Partitioning**: Splits data by category/date/hash to reduce search scope.
- **Indexing**: HNSW (Hierarchical Navigable Small World) speeds up approximate nearest neighbor search.

## Step 1: Basic pgvector Setup
```go
package main

import (
    "context"
    "fmt"
    
    "github.com/lookatitude/beluga-ai/pkg/vectorstores"
    "github.com/lookatitude/beluga-ai/pkg/embeddings"
)

func main() {
    ctx := context.Background()
    // Assume embedder is created...
    
    store, err := vectorstores.NewPgVectorStore(ctx,
        vectorstores.WithEmbedder(embedder),
        vectorstores.WithProviderConfig("connection_string", 
            "postgres://user:pass@localhost:5432/vectordb"),
        vectorstores.WithProviderConfig("table_name", "documents"),
        // Dimension MUST match your embedder (e.g., 1536 for OpenAI)
        vectorstores.WithProviderConfig("dimension", 1536),
    )
}
```

## Step 2: Creating HNSW Index

Before loading data, ensure you create an index for speed.
-- Run this in your database
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE documents (
    id bigserial PRIMARY KEY,
    content text,
    metadata jsonb,
    embedding vector(1536)
);

-- Create HNSW index
-- m: max connections per layer (16-64)
-- ef_construction: size of dynamic list during construction (64-512)
CREATE INDEX ON documents USING hnsw (embedding vector_cosine_ops) 
WITH (m = 16, ef_construction = 64);
```

## Step 3: Sharding Strategy (Partitioning)

If you have multi-tenant data, partition by `tenant_id`.
CREATE TABLE documents_partitioned (
    id bigserial,
    tenant_id int,
    content text,
    metadata jsonb,
    embedding vector(1536),
    PRIMARY KEY (id, tenant_id)
) PARTITION BY LIST (tenant_id);

-- Create partition for tenant 1
CREATE TABLE documents_tenant_1 PARTITION OF documents_partitioned
    FOR VALUES IN (1);
```

## Step 4: Connecting to a Partition

In Beluga AI, you can route queries to specific tables or partitions using metadata filtering or dynamic table names.
// Dynamic table name per request
store, _ := vectorstores.NewPgVectorStore(ctx,
```
    vectorstores.WithProviderConfig("table_name", "documents_tenant_1"),
    // ...
)

## Step 5: Optimization Tips

1. **Pre-warm**: `pg_prewarm` to load index into RAM.
2. **Vacuum**: Run `VACUUM ANALYZE` frequently on vector tables.
3. **EF Search**: Tune `hnsw.ef_search` at query time for trade-off between speed/recall.
SET hnsw.ef_search = 100;
SELECT ... FROM documents ORDER BY embedding \<-> query_vector LIMIT 5;
```

## Verification

1. Load 10,000 vectors.
2. Run a similarity search without index (measure time).
3. Create HNSW index.
4. Run search again (should be \<10ms).

## Next Steps

- **[Hybrid Search](../higher-level/retrievers-hybrid-search.md)** - Combine SQL + Vector search
- **[Memory Management](../../getting-started/05-memory-management.md)** - Use vector store for agent memory
