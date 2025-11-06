# PgVector Provider Guide

Complete guide to using PostgreSQL with pgvector extension.

## Overview

PgVector provides vector storage using PostgreSQL with the pgvector extension.

## Prerequisites

- PostgreSQL 12+ installed
- pgvector extension installed

## Setup

### Install pgvector Extension

```sql
CREATE EXTENSION IF NOT EXISTS vector;
```

### Configuration

```go
store, err := vectorstores.NewPgVectorStore(ctx,
    vectorstores.WithEmbedder(embedder),
    vectorstores.WithProviderConfig("connection_string", 
        "postgres://user:pass@localhost/db"),
    vectorstores.WithProviderConfig("table_name", "documents"),
    vectorstores.WithProviderConfig("embedding_dimension", 1536),
)
```

## Schema Design

```sql
CREATE TABLE documents (
    id SERIAL PRIMARY KEY,
    content TEXT,
    embedding vector(1536),
    metadata JSONB
);

CREATE INDEX ON documents USING ivfflat (embedding vector_cosine_ops);
```

## Best Practices

1. **Index optimization**: Use appropriate index type
2. **Connection pooling**: Reuse connections
3. **Batch operations**: Insert documents in batches
4. **Metadata indexing**: Index JSONB metadata fields

## Production Deployment

- Use connection pooling
- Monitor query performance
- Optimize indexes
- Set appropriate timeouts

## Troubleshooting

See [Troubleshooting Guide](../../TROUBLESHOOTING.md) for common issues.

---

**Next:** [Vector Store Comparison](./comparison.md)

