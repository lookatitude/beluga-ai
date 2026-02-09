---
title: Qdrant Cloud
description: Integrate Qdrant Cloud managed clusters with Beluga AI for high-performance vector storage and similarity search.
---

Qdrant Cloud provides managed, scalable vector storage with high-performance similarity search. This guide covers configuring Beluga AI to use Qdrant Cloud clusters for vector storage, enabling production-ready RAG applications with managed infrastructure.

## Overview

Beluga AI's `VectorStore` interface abstracts vector database operations behind a unified API. The Qdrant provider registers via `init()` and supports add, search, and delete operations against Qdrant Cloud clusters. Qdrant offers HNSW-based approximate nearest neighbor search with rich metadata filtering.

Key capabilities:
- Managed clusters with configurable sizing and replication
- HNSW indexing for fast approximate nearest neighbor search
- Rich payload-based metadata filtering
- Collections auto-created on first use
- OpenTelemetry instrumentation for observability

## Prerequisites

- Go 1.23 or later
- Beluga AI framework installed
- Qdrant Cloud account ([cloud.qdrant.io](https://cloud.qdrant.io))
- A Qdrant cluster with its URL and API key

## Installation

Install the Beluga AI module:

```bash
go get github.com/lookatitude/beluga-ai
```

### Qdrant Cloud Setup

1. Sign in to the [Qdrant Cloud console](https://cloud.qdrant.io)
2. Create a new cluster
3. Note the cluster URL (e.g., `https://xyz.us-east-1-0.aws.cloud.qdrant.io`) and API key

Set the required environment variables:

```bash
export QDRANT_URL="https://your-cluster.qdrant.io"
export QDRANT_API_KEY="your-api-key"
export OPENAI_API_KEY="your-openai-key"
```

## Configuration

The Qdrant provider accepts the following options:

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `url` | Qdrant cluster URL | - | Yes |
| `api_key` | Qdrant API key | - | Yes (for cloud) |
| `collection_name` | Collection name | `default` | No |
| `embedding_dim` | Embedding vector dimension | - | Yes |
| `timeout` | Request timeout | `30s` | No |

## Usage

### Basic Vector Store

Create a Qdrant vector store, add documents, and run similarity search:

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

    // Create a Qdrant vector store
    store, err := vectorstore.New("qdrant",
        vectorstore.WithEmbedder(embedder),
        vectorstore.WithOption("url", os.Getenv("QDRANT_URL")),
        vectorstore.WithOption("api_key", os.Getenv("QDRANT_API_KEY")),
        vectorstore.WithOption("collection_name", "my-documents"),
        vectorstore.WithOption("embedding_dim", 1536),
    )
    if err != nil {
        log.Fatalf("Failed to create store: %v", err)
    }

    // Embed and add documents
    docs := []schema.Document{
        {PageContent: "Machine learning is a subset of AI.", Metadata: map[string]any{"topic": "ml"}},
        {PageContent: "Go is a programming language.", Metadata: map[string]any{"topic": "programming"}},
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
    queryVec, err := embedder.EmbedQuery(ctx, "artificial intelligence")
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

Use payload-based metadata filters to scope searches:

```go
results, err := store.Search(ctx, queryVec, 5,
    vectorstore.WithFilter(map[string]any{"topic": "ml"}),
)
if err != nil {
    log.Fatalf("Filtered search failed: %v", err)
}
```

### Collection Management

Collections are created automatically on first use. To create a collection explicitly with custom parameters, use the Qdrant Go client directly:

```go
import (
    "github.com/qdrant/go-client/qdrant"
)

func createCollection(ctx context.Context, client *qdrant.Client, name string, dim int) error {
    _, err := client.CreateCollection(ctx, &qdrant.CreateCollection{
        CollectionName: name,
        VectorsConfig: &qdrant.VectorsConfig{
            Config: &qdrant.VectorsConfig_Params{
                Params: &qdrant.VectorParams{
                    Size:     uint64(dim),
                    Distance: qdrant.Distance_Cosine,
                },
            },
        },
    })
    return err
}
```

## Advanced Topics

### OpenTelemetry Instrumentation

Add tracing to Qdrant operations for production observability:

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

    tracer := otel.Tracer("beluga.vectorstore.qdrant")
    ctx, span := tracer.Start(ctx, "qdrant.ingest",
        trace.WithAttributes(
            attribute.String("provider", "qdrant"),
            attribute.String("collection", "my-documents"),
        ),
    )
    defer span.End()

    store, err := vectorstore.New("qdrant",
        vectorstore.WithEmbedder(embedder),
        vectorstore.WithOption("url", os.Getenv("QDRANT_URL")),
        vectorstore.WithOption("api_key", os.Getenv("QDRANT_API_KEY")),
        vectorstore.WithOption("collection_name", "my-documents"),
        vectorstore.WithOption("embedding_dim", 1536),
    )
    if err != nil {
        span.RecordError(err)
        log.Fatalf("Failed to create store: %v", err)
    }

    docs := []schema.Document{
        {PageContent: "Machine learning enables computers to learn.", Metadata: map[string]any{"topic": "ml", "source": "doc1"}},
        {PageContent: "Go is statically typed and compiled.", Metadata: map[string]any{"topic": "programming", "source": "doc2"}},
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

When using Qdrant Cloud in production:

- **Cluster sizing**: Select a cluster size appropriate for your document volume and query throughput. Monitor resource utilization in the Qdrant Cloud dashboard.
- **HNSW tuning**: Configure HNSW index parameters (`m`, `ef_construct`) for your accuracy/performance tradeoff. Higher values improve recall at the cost of memory and indexing speed.
- **Timeouts**: Set appropriate client timeouts for your network latency. The default 30-second timeout may need adjustment for large batch operations.
- **Retry logic**: Use Beluga's `resilience` package for automatic retry with backoff on transient failures.
- **Monitoring**: Combine Qdrant Cloud metrics with OpenTelemetry traces for end-to-end observability.

## Troubleshooting

### Connection Refused

The cluster URL is invalid or the cluster is not accessible. Verify connectivity:

```bash
curl https://your-cluster.qdrant.io/collections
```

### Unauthorized

The API key is invalid or missing. Confirm that `QDRANT_API_KEY` matches the key generated in the Qdrant Cloud console.

### Collection Not Found

Collections are created automatically on first add operation. If you need to create a collection with specific parameters before adding documents, use the Qdrant Go client as shown in the [Collection Management](#collection-management) section.

## Related Resources

- [Vector Stores Overview](/integrations/vector-stores) -- All supported vector store providers
- [Pinecone Serverless Integration](/integrations/pinecone-serverless) -- Serverless vector storage
- [RAG Tutorial](/tutorials/rag-pipeline) -- Build end-to-end RAG applications
