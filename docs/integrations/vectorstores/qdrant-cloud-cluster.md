# Qdrant Cloud Cluster

Welcome, colleague! In this integration guide, we're going to integrate Qdrant Cloud with Beluga AI's vectorstores package. Qdrant Cloud provides managed, scalable vector storage with high-performance similarity search.

## What you will build

You will configure Beluga AI to use Qdrant Cloud clusters for vector storage and similarity search, enabling scalable, production-ready RAG applications with managed infrastructure.

## Learning Objectives

- ✅ Configure Qdrant Cloud with Beluga AI
- ✅ Create and manage collections
- ✅ Perform similarity search
- ✅ Understand Qdrant Cloud best practices

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- Qdrant Cloud account and API key
- Qdrant cluster URL

## Step 1: Setup and Installation

Sign up for Qdrant Cloud at https://cloud.qdrant.io

Create a cluster and get your:
- Cluster URL (e.g., `https://xyz.us-east-1-0.aws.cloud.qdrant.io`)
- API key

Set environment variables:
bash
```bash
export QDRANT_URL="https://your-cluster.qdrant.io"
export QDRANT_API_KEY="your-api-key"
```

## Step 2: Basic Qdrant Configuration

Create a Qdrant vector store:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/embeddings"
    "github.com/lookatitude/beluga-ai/pkg/schema"
    "github.com/lookatitude/beluga-ai/pkg/vectorstores"
)

func main() {
    ctx := context.Background()

    // Create embedder (required for Qdrant)
    embedderConfig := embeddings.NewConfig()
    embedderConfig.OpenAI = &embeddings.OpenAIConfig{
        APIKey: os.Getenv("OPENAI_API_KEY"),
        Model:  "text-embedding-ada-002",
    }
    
    embedderFactory, _ := embeddings.NewEmbedderFactory(embedderConfig)
    embedder, _ := embedderFactory.CreateEmbedder("openai")

    // Create Qdrant vector store
    store, err := vectorstores.NewVectorStore(ctx, "qdrant",
        vectorstores.WithEmbedder(embedder),
        vectorstores.WithProviderConfig("url", os.Getenv("QDRANT_URL")),
        vectorstores.WithProviderConfig("api_key", os.Getenv("QDRANT_API_KEY")),
        vectorstores.WithProviderConfig("collection_name", "my-documents"),
        vectorstores.WithProviderConfig("embedding_dim", 1536),
    )
    if err != nil {
        log.Fatalf("Failed to create store: %v", err)
    }

    // Add documents
    docs := []schema.Document{
        schema.NewDocument("Machine learning is a subset of AI.", map[string]string{"topic": "ml"}),
        schema.NewDocument("Go is a programming language.", map[string]string{"topic": "programming"}),
    }

    ids, err := store.AddDocuments(ctx, docs)
    if err != nil {
        log.Fatalf("Failed to add documents: %v", err)
    }

    fmt.Printf("Added %d documents with IDs: %v\n", len(ids), ids)

    // Search
    results, scores, err := store.SimilaritySearchByQuery(ctx, "artificial intelligence", 5, embedder)
    if err != nil {
        log.Fatalf("Search failed: %v", err)
    }

    fmt.Printf("Found %d results\n", len(results))
    for i, result := range results {
        fmt.Printf("Result %d (score: %.3f): %s\n", i+1, scores[i], result.PageContent)
    }
}
```

### Verification

Run the example:
bash
```bash
export QDRANT_URL="https://your-cluster.qdrant.io"
export QDRANT_API_KEY="your-api-key"
export OPENAI_API_KEY="your-openai-key"
go run main.go
```

You should see documents added and search results.

## Step 3: Collection Management

Create and manage collections:
// Collection is created automatically on first use
// Or create explicitly via Qdrant API

```go
import (
    "github.com/qdrant/go-client/qdrant"
)
go
func createCollection(ctx context.Context, client *qdrant.Client, collectionName string, dim int) error {
    _, err := client.CreateCollection(ctx, &qdrant.CreateCollection{
        CollectionName: collectionName,
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

## Step 4: Metadata Filtering

Use metadata filters for advanced queries:
// Qdrant supports metadata filtering
// Filter by metadata when searching
results, scores, err := store.SimilaritySearchByQuery(ctx, "query", 5, embedder,
```
    vectorstores.WithMetadataFilter("topic", "ml"),
)

## Step 5: Complete Integration

Here's a complete, production-ready example:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/embeddings"
    "github.com/lookatitude/beluga-ai/pkg/schema"
    "github.com/lookatitude/beluga-ai/pkg/vectorstores"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    // Setup embedder
    embedderConfig := embeddings.NewConfig()
    embedderConfig.OpenAI = &embeddings.OpenAIConfig{
        APIKey:     os.Getenv("OPENAI_API_KEY"),
        Model:      "text-embedding-ada-002",
        Timeout:    30 * time.Second,
        MaxRetries: 3,
    }
    
    embedderFactory, err := embeddings.NewEmbedderFactory(embedderConfig)
    if err != nil {
        log.Fatalf("Failed to create embedder factory: %v", err)
    }
    
    embedder, err := embedderFactory.CreateEmbedder("openai")
    if err != nil {
        log.Fatalf("Failed to create embedder: %v", err)
    }

    // Create Qdrant store
    tracer := otel.Tracer("beluga.vectorstores.qdrant")
    ctx, span := tracer.Start(ctx, "qdrant.setup",
        trace.WithAttributes(
            attribute.String("provider", "qdrant"),
            attribute.String("collection", "my-documents"),
        ),
    )
    defer span.End()

    store, err := vectorstores.NewVectorStore(ctx, "qdrant",
        vectorstores.WithEmbedder(embedder),
        vectorstores.WithProviderConfig("url", os.Getenv("QDRANT_URL")),
        vectorstores.WithProviderConfig("api_key", os.Getenv("QDRANT_API_KEY")),
        vectorstores.WithProviderConfig("collection_name", "my-documents"),
        vectorstores.WithProviderConfig("embedding_dim", 1536),
    )
    if err != nil {
        span.RecordError(err)
        log.Fatalf("Failed to create store: %v", err)
    }

    // Add documents
    docs := []schema.Document{
        schema.NewDocument("Machine learning enables computers to learn.", 
            map[string]string{"topic": "ml", "source": "doc1"}),
        schema.NewDocument("Go is statically typed and compiled.", 
            map[string]string{"topic": "programming", "source": "doc2"}),
    }

    ids, err := store.AddDocuments(ctx, docs)
    if err != nil {
        span.RecordError(err)
        log.Fatalf("Failed to add documents: %v", err)
    }

    span.SetAttributes(
        attribute.Int("documents_added", len(ids)),
    )

    fmt.Printf("Successfully added %d documents\n", len(ids))

    // Search
    results, scores, err := store.SimilaritySearchByQuery(ctx, "programming languages", 5, embedder)
    if err != nil {
        log.Fatalf("Search failed: %v", err)
    }

    fmt.Printf("Found %d results:\n", len(results))
    for i, result := range results {
        fmt.Printf("  %d. Score: %.3f - %s\n", i+1, scores[i], result.PageContent)
    }
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `url` | Qdrant cluster URL | - | Yes |
| `api_key` | Qdrant API key | - | Yes (for cloud) |
| `collection_name` | Collection name | `default` | No |
| `embedding_dim` | Embedding dimension | - | Yes |
| `timeout` | Request timeout | `30s` | No |

## Common Issues

### "Connection refused"

**Problem**: Invalid Qdrant URL or cluster not accessible.

**Solution**: Verify cluster URL and network connectivity:curl https://your-cluster.qdrant.io/collections
```

### "Unauthorized"

**Problem**: Invalid or missing API key.

**Solution**: Verify API key:export QDRANT_API_KEY="your-api-key"
```

### "Collection not found"

**Problem**: Collection doesn't exist.

**Solution**: Collection is created automatically, or create it explicitly via API.

## Production Considerations

When using Qdrant Cloud in production:

- **Choose right cluster size**: Select appropriate cluster size for your workload
- **Monitor usage**: Track API calls and storage usage
- **Use indexes**: Configure HNSW indexes for better performance
- **Set timeouts**: Configure appropriate timeouts
- **Handle rate limits**: Implement retry logic

## Next Steps

Congratulations! You've integrated Qdrant Cloud with Beluga AI. Next, learn how to:

- **[Pinecone Serverless](./pinecone-serverless.md)** - Serverless vector storage
- **[Vectorstores Package Documentation](../../api/packages/vectorstores.md)** - Deep dive into vectorstores
- **[RAG Tutorial](../../getting-started/02-simple-rag.md)** - Build RAG applications

---

**Ready for more?** Check out the [Integrations Index](./README.md) for more integration guides!
