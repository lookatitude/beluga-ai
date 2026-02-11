---
title: Production pgvector Sharding
description: Configure PostgreSQL with pgvector for high-scale vector storage, including HNSW indexing and table partitioning.
---

As vector datasets grow beyond 1 million records, single-table scans become too slow and HNSW indexes may not fit in RAM. Table partitioning (sharding) combined with proper indexing provides the foundation for production-scale vector search. This approach leverages PostgreSQL's built-in partitioning, which means you retain full SQL capabilities (joins, transactions, aggregations) alongside vector search — a significant advantage over purpose-built vector databases that sacrifice query flexibility.

## What You Will Build

A production-grade pgvector setup with HNSW indexing, table partitioning by tenant, and query performance optimization.

## Prerequisites

- PostgreSQL with the `pgvector` extension installed
- Understanding of [In-memory Vector Store](/tutorials/providers/inmemory-vectorstore)

## Step 1: Basic pgvector Setup

Connect Beluga AI to PostgreSQL with pgvector. The connection uses the same `vectorstore.New()` registry pattern as all other vector store providers, so switching from in-memory to pgvector requires only a provider name and configuration change.

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/rag/vectorstore"
    _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/pgvector"
)

func main() {
    ctx := context.Background()

    store, err := vectorstore.New("pgvector", config.ProviderConfig{
        "connection_string": os.Getenv("DATABASE_URL"),
        "table_name":        "documents",
    })
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    fmt.Println("Connected to pgvector")
    _ = ctx
    _ = store
}
```

## Step 2: Create the Schema

Set up the table and HNSW index. HNSW (Hierarchical Navigable Small World) is preferred over IVFFlat for most workloads because it provides better recall with comparable latency and does not require a separate training step. The `vector_cosine_ops` operator class matches the cosine similarity metric used by most embedding models.

```sql
-- Enable the vector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Create the documents table
CREATE TABLE documents (
    id         bigserial PRIMARY KEY,
    content    text NOT NULL,
    metadata   jsonb DEFAULT '{}',
    embedding  vector(1536)  -- Match your embedder's dimension
);

-- Create HNSW index for cosine similarity
-- m: max connections per layer (16-64, higher = better recall, more memory)
-- ef_construction: build-time list size (64-512, higher = better recall, slower build)
CREATE INDEX idx_documents_embedding
    ON documents
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);
```

## Step 3: HNSW Parameter Tuning

HNSW parameters control the trade-off between recall, latency, and memory. The defaults (`m=16`, `ef_construction=64`) provide a good balance for most workloads. Increase these values when recall is more important than build time or memory usage — for example, in applications where returning the wrong document has high cost (medical, legal, financial).

| Parameter | Range | Effect |
|:---|:---|:---|
| `m` | 16-64 | Higher = better recall, more memory per node |
| `ef_construction` | 64-512 | Higher = better index quality, slower build |
| `ef_search` | 40-500 | Higher = better recall at query time, slower queries |

Set `ef_search` at query time:

```sql
-- Increase recall for important queries
SET hnsw.ef_search = 200;

SELECT id, content, embedding <=> $1 AS distance
FROM documents
ORDER BY embedding <=> $1
LIMIT 10;
```

## Step 4: Table Partitioning (Sharding)

For multi-tenant applications, partition by tenant to reduce search scope. Partitioning confines each query to a single partition's index rather than scanning the entire table, which reduces both latency and resource consumption. Each partition has its own HNSW index, so adding a new tenant creates a small, fast index rather than expanding a single large one.

```sql
-- Create partitioned table
CREATE TABLE documents_partitioned (
    id         bigserial,
    tenant_id  int NOT NULL,
    content    text NOT NULL,
    metadata   jsonb DEFAULT '{}',
    embedding  vector(1536),
    PRIMARY KEY (id, tenant_id)
) PARTITION BY LIST (tenant_id);

-- Create partitions for each tenant
CREATE TABLE documents_tenant_1
    PARTITION OF documents_partitioned
    FOR VALUES IN (1);

CREATE TABLE documents_tenant_2
    PARTITION OF documents_partitioned
    FOR VALUES IN (2);

-- Create HNSW index on each partition
CREATE INDEX ON documents_tenant_1
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

CREATE INDEX ON documents_tenant_2
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);
```

## Step 5: Route Queries to Partitions

In Beluga AI, route queries to specific partitions using dynamic table names or metadata filtering. Creating a per-tenant store instance is the simplest approach — it maps directly to the partitioned table and ensures complete data isolation between tenants.

```go
// Per-tenant store
tenantStore, err := vectorstore.New("pgvector", config.ProviderConfig{
    "connection_string": os.Getenv("DATABASE_URL"),
    "table_name":        fmt.Sprintf("documents_tenant_%d", tenantID),
})
```

## Step 6: Performance Optimization

### Pre-warm the index

Load the HNSW index into shared buffers at startup. This eliminates cold-start latency for the first queries after a database restart, which is important in auto-scaling environments where new database instances may serve traffic immediately.

```sql
-- Requires pg_prewarm extension
CREATE EXTENSION IF NOT EXISTS pg_prewarm;
SELECT pg_prewarm('idx_documents_embedding');
```

### Maintenance

Run regular maintenance on vector tables. `VACUUM ANALYZE` updates the statistics that the query planner uses to choose efficient execution plans, and `REINDEX` rebuilds the HNSW graph after large batch inserts that may have degraded its structure.

```sql
-- Update statistics for the query planner
VACUUM ANALYZE documents;

-- Reindex periodically after large batch inserts
REINDEX INDEX CONCURRENTLY idx_documents_embedding;
```

### Connection pooling

Use a connection pooler (PgBouncer, pgcat) in front of PostgreSQL. Vector similarity queries can be CPU-intensive — limit concurrent queries to avoid overloading the database.

## Scaling Guidelines

| Dataset Size | Recommendation |
|:---|:---|
| < 100k vectors | Single table, HNSW index |
| 100k - 1M | HNSW with tuned parameters |
| 1M - 10M | Table partitioning by tenant or category |
| > 10M | Horizontal sharding across PostgreSQL instances |

## Verification

1. Load 10,000 vectors into a table without an index — measure query time.
2. Create the HNSW index — measure query time again (should be < 10ms).
3. Create tenant partitions — verify queries only scan the relevant partition.

## Next Steps

- [Hybrid Search](/tutorials/rag/hybrid-search) — Combine vector and keyword search
- [In-memory Vector Store](/tutorials/providers/inmemory-vectorstore) — Zero-dependency development
