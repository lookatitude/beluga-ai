---
title: Pinecone Serverless
description: Integrate Pinecone Serverless with Beluga AI for fully managed, auto-scaling vector storage with pay-per-use pricing.
---

Pinecone Serverless provides fully managed, auto-scaling vector storage with pay-per-use pricing. This guide covers configuring Beluga AI to use Pinecone Serverless for vector storage and similarity search, enabling scalable RAG applications without infrastructure management.

## Overview

Beluga AI's `VectorStore` interface abstracts vector database operations behind a unified API. The Pinecone provider registers via `init()` and supports add, search, and delete operations against Pinecone Serverless indexes.

Key capabilities:
- Auto-scaling serverless indexes with no pod management
- Pay-per-use pricing based on operations and storage
- Metadata filtering for scoped similarity search
- OpenTelemetry instrumentation for observability

## Prerequisites

- Go 1.23 or later
- Beluga AI framework installed
- Pinecone account with API key ([pinecone.io](https://www.pinecone.io))
- A serverless index created in the Pinecone console

## Installation

Install the Beluga AI module:

```bash
go get github.com/lookatitude/beluga-ai
```

### Pinecone Setup

1. Sign in to the [Pinecone console](https://app.pinecone.io)
2. Create a new index with type **Serverless**
3. Set the dimension to match your embedding model (e.g., 1536 for OpenAI `text-embedding-ada-002`)
4. Note your API key, environment, and project ID

Set the required environment variables:

```bash
export PINECONE_API_KEY="your-api-key"
export PINECONE_ENVIRONMENT="us-west1-gcp"
export PINECONE_PROJECT_ID="your-project-id"
export PINECONE_INDEX_NAME="my-index"
export OPENAI_API_KEY="your-openai-key"
```

## Configuration

The Pinecone provider accepts the following options:

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `api_key` | Pinecone API key | - | Yes |
| `environment` | Pinecone environment (e.g., `us-west1-gcp`) | - | Yes |
| `project_id` | Pinecone project ID | - | Yes |
| `index_name` | Name of the serverless index | - | Yes |
| `embedding_dim` | Embedding vector dimension | - | Yes |
| `timeout` | Request timeout | `30s` | No |

## Usage

### Basic Vector Store

Create a Pinecone vector store, add documents, and run similarity search:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/rag/embedding"
    "github.com/lookatitude/beluga-ai/rag/vectorstore"
    "github.com/lookatitude/beluga-ai/schema"
)

func main() {
    ctx := context.Background()

    // Create an OpenAI embedder
    embedder, err := embedding.New("openai",
        embedding.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        embedding.WithModel("text-embedding-ada-002"),
    )
    if err != nil {
        log.Fatalf("Failed to create embedder: %v", err)
    }

    // Create a Pinecone vector store
    store, err := vectorstore.New("pinecone",
        vectorstore.WithEmbedder(embedder),
        vectorstore.WithOption("api_key", os.Getenv("PINECONE_API_KEY")),
        vectorstore.WithOption("environment", os.Getenv("PINECONE_ENVIRONMENT")),
        vectorstore.WithOption("project_id", os.Getenv("PINECONE_PROJECT_ID")),
        vectorstore.WithOption("index_name", os.Getenv("PINECONE_INDEX_NAME")),
        vectorstore.WithOption("embedding_dim", 1536),
    )
    if err != nil {
        log.Fatalf("Failed to create store: %v", err)
    }

    // Embed and add documents
    docs := []schema.Document{
        {PageContent: "Machine learning is transforming industries.", Metadata: map[string]any{"category": "tech"}},
        {PageContent: "Go is known for its simplicity and performance.", Metadata: map[string]any{"category": "programming"}},
    }

    embeddings, err := embedder.EmbedDocuments(ctx, docs)
    if err != nil {
        log.Fatalf("Failed to embed documents: %v", err)
    }

    err = store.Add(ctx, docs, embeddings)
    if err != nil {
        log.Fatalf("Failed to add documents: %v", err)
    }
    fmt.Printf("Added %d documents\n", len(docs))

    // Search by query
    queryVec, err := embedder.EmbedQuery(ctx, "programming languages")
    if err != nil {
        log.Fatalf("Failed to embed query: %v", err)
    }

    results, err := store.Search(ctx, queryVec, 5)
    if err != nil {
        log.Fatalf("Search failed: %v", err)
    }

    fmt.Printf("Found %d results\n", len(results))
    for i, result := range results {
        fmt.Printf("  %d. %s\n", i+1, result.PageContent)
    }
}
```

### Metadata Filtering

Use metadata filters to scope searches to specific document categories:

```go
results, err := store.Search(ctx, queryVec, 5,
    vectorstore.WithFilter(map[string]any{"category": "tech"}),
)
if err != nil {
    log.Fatalf("Filtered search failed: %v", err)
}
```

### Serverless Index Behavior

Serverless indexes differ from pod-based indexes:

- **Auto-scaling**: No pod type or replica configuration required
- **Pay-per-use**: Charges based on read/write operations and storage
- **Cold start**: Infrequently accessed indexes may have higher initial latency

No additional configuration is needed beyond the standard options above.

## Advanced Topics

### OpenTelemetry Instrumentation

Add tracing to Pinecone operations for production observability:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/lookatitude/beluga-ai/rag/embedding"
    "github.com/lookatitude/beluga-ai/rag/vectorstore"
    "github.com/lookatitude/beluga-ai/schema"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    embedder, err := embedding.New("openai",
        embedding.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        embedding.WithModel("text-embedding-ada-002"),
    )
    if err != nil {
        log.Fatalf("Failed to create embedder: %v", err)
    }

    tracer := otel.Tracer("beluga.vectorstore.pinecone")
    ctx, span := tracer.Start(ctx, "pinecone.ingest",
        trace.WithAttributes(
            attribute.String("provider", "pinecone"),
            attribute.String("index", os.Getenv("PINECONE_INDEX_NAME")),
        ),
    )
    defer span.End()

    store, err := vectorstore.New("pinecone",
        vectorstore.WithEmbedder(embedder),
        vectorstore.WithOption("api_key", os.Getenv("PINECONE_API_KEY")),
        vectorstore.WithOption("environment", os.Getenv("PINECONE_ENVIRONMENT")),
        vectorstore.WithOption("project_id", os.Getenv("PINECONE_PROJECT_ID")),
        vectorstore.WithOption("index_name", os.Getenv("PINECONE_INDEX_NAME")),
        vectorstore.WithOption("embedding_dim", 1536),
    )
    if err != nil {
        span.RecordError(err)
        log.Fatalf("Failed to create store: %v", err)
    }

    docs := []schema.Document{
        {PageContent: "AI is revolutionizing technology.", Metadata: map[string]any{"category": "ai", "source": "doc1"}},
        {PageContent: "Serverless computing scales automatically.", Metadata: map[string]any{"category": "cloud", "source": "doc2"}},
    }

    embeddings, err := embedder.EmbedDocuments(ctx, docs)
    if err != nil {
        span.RecordError(err)
        log.Fatalf("Failed to embed documents: %v", err)
    }

    err = store.Add(ctx, docs, embeddings)
    if err != nil {
        span.RecordError(err)
        log.Fatalf("Failed to add documents: %v", err)
    }

    span.SetAttributes(attribute.Int("documents_added", len(docs)))
    fmt.Printf("Successfully added %d documents\n", len(docs))
}
```

### Production Considerations

When using Pinecone Serverless in production:

- **Batch operations**: Group document additions into batches to reduce per-operation costs.
- **Rate limiting**: Implement backoff and retry logic for API rate limits. Beluga's `resilience` package provides built-in retry and circuit breaker support.
- **Index dimension limits**: Verify that your embedding model dimension matches the index configuration exactly.
- **Monitoring**: Track usage and costs in the Pinecone dashboard. Combine with OpenTelemetry traces for end-to-end visibility.
- **Backup**: Pinecone Serverless does not provide built-in backup. Maintain a source-of-truth data pipeline for re-indexing.

## Troubleshooting

### Index Not Found

The index name does not exist or is misspelled. Verify available indexes:

```bash
curl -X GET "https://api.pinecone.io/indexes" \
  -H "Api-Key: $PINECONE_API_KEY"
```

### Invalid API Key

The API key or environment does not match. Confirm that `PINECONE_API_KEY` and `PINECONE_ENVIRONMENT` correspond to the same Pinecone project.

### Dimension Mismatch

The embedding vector dimension does not match the index configuration. Ensure `embedding_dim` matches the dimension set when the index was created (e.g., 1536 for `text-embedding-ada-002`).

## Related Resources

- [Vector Stores Overview](/integrations/vector-stores) -- All supported vector store providers
- [Qdrant Cloud Integration](/integrations/qdrant-cloud) -- Managed Qdrant clusters
- [RAG Tutorial](/tutorials/rag-pipeline) -- Build end-to-end RAG applications
