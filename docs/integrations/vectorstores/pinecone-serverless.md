# Pinecone Serverless Integration

Welcome, colleague! In this integration guide, we're going to integrate Pinecone Serverless with Beluga AI's vectorstores package. Pinecone Serverless provides fully managed, auto-scaling vector storage with pay-per-use pricing.

## What you will build

You will configure Beluga AI to use Pinecone Serverless for vector storage and similarity search, enabling scalable, cost-effective RAG applications without infrastructure management.

## Learning Objectives

- ✅ Configure Pinecone Serverless with Beluga AI
- ✅ Create and manage indexes
- ✅ Perform similarity search
- ✅ Understand serverless pricing and limits

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- Pinecone account and API key
- Pinecone project and index

## Step 1: Setup and Installation

Sign up for Pinecone at https://www.pinecone.io

Create a serverless index:
1. Go to Pinecone console
2. Create a new index
3. Choose "Serverless" type
4. Set dimensions (e.g., 1536 for OpenAI embeddings)
5. Get your API key and environment

Set environment variables:
bash
```bash
export PINECONE_API_KEY="your-api-key"
export PINECONE_ENVIRONMENT="us-west1-gcp"  # or your environment
export PINECONE_PROJECT_ID="your-project-id"
export PINECONE_INDEX_NAME="my-index"
```

## Step 2: Basic Pinecone Configuration

Create a Pinecone vector store:
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

    // Create embedder
    embedderConfig := embeddings.NewConfig()
    embedderConfig.OpenAI = &embeddings.OpenAIConfig{
        APIKey: os.Getenv("OPENAI_API_KEY"),
        Model:  "text-embedding-ada-002",
    }
    
    embedderFactory, _ := embeddings.NewEmbedderFactory(embedderConfig)
    embedder, _ := embedderFactory.CreateEmbedder("openai")

    // Create Pinecone store
    store, err := vectorstores.NewVectorStore(ctx, "pinecone",
        vectorstores.WithEmbedder(embedder),
        vectorstores.WithProviderConfig("api_key", os.Getenv("PINECONE_API_KEY")),
        vectorstores.WithProviderConfig("environment", os.Getenv("PINECONE_ENVIRONMENT")),
        vectorstores.WithProviderConfig("project_id", os.Getenv("PINECONE_PROJECT_ID")),
        vectorstores.WithProviderConfig("index_name", os.Getenv("PINECONE_INDEX_NAME")),
        vectorstores.WithProviderConfig("embedding_dim", 1536),
    )
    if err != nil {
        log.Fatalf("Failed to create store: %v", err)
    }

    // Add documents
    docs := []schema.Document{
        schema.NewDocument("Machine learning is transforming industries.", 
            map[string]string{"category": "tech"}),
        schema.NewDocument("Go is known for its simplicity and performance.", 
            map[string]string{"category": "programming"}),
    }

    ids, err := store.AddDocuments(ctx, docs)
    if err != nil {
        log.Fatalf("Failed to add documents: %v", err)
    }

    fmt.Printf("Added %d documents\n", len(ids))

    // Search
    results, scores, err := store.SimilaritySearchByQuery(ctx, "programming languages", 5, embedder)
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
export PINECONE_API_KEY="your-api-key"
export PINECONE_ENVIRONMENT="us-west1-gcp"
export PINECONE_PROJECT_ID="your-project"
export PINECONE_INDEX_NAME="my-index"
export OPENAI_API_KEY="your-openai-key"
go run main.go
```

You should see documents added and search results.

## Step 3: Serverless Index Configuration

Configure serverless index settings:
// Serverless indexes auto-scale
// No need to specify pod type or replicas
// Pay only for what you use

store, err := vectorstores.NewVectorStore(ctx, "pinecone",
```
    vectorstores.WithEmbedder(embedder),
    vectorstores.WithProviderConfig("api_key", apiKey),
    vectorstores.WithProviderConfig("environment", environment),
    vectorstores.WithProviderConfig("project_id", projectID),
    vectorstores.WithProviderConfig("index_name", indexName),
    vectorstores.WithProviderConfig("embedding_dim", 1536),
    // Serverless-specific: no pod configuration needed
)

## Step 4: Metadata Filtering

Use metadata filters with Pinecone:
// Pinecone supports metadata filtering
results, scores, err := store.SimilaritySearchByQuery(ctx, "query", 5, embedder,
    vectorstores.WithMetadataFilter("category", "tech"),
)
```

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

    // Create Pinecone store
    tracer := otel.Tracer("beluga.vectorstores.pinecone")
    ctx, span := tracer.Start(ctx, "pinecone.setup",
        trace.WithAttributes(
            attribute.String("provider", "pinecone"),
            attribute.String("index", os.Getenv("PINECONE_INDEX_NAME")),
        ),
    )
    defer span.End()

    store, err := vectorstores.NewVectorStore(ctx, "pinecone",
        vectorstores.WithEmbedder(embedder),
        vectorstores.WithProviderConfig("api_key", os.Getenv("PINECONE_API_KEY")),
        vectorstores.WithProviderConfig("environment", os.Getenv("PINECONE_ENVIRONMENT")),
        vectorstores.WithProviderConfig("project_id", os.Getenv("PINECONE_PROJECT_ID")),
        vectorstores.WithProviderConfig("index_name", os.Getenv("PINECONE_INDEX_NAME")),
        vectorstores.WithProviderConfig("embedding_dim", 1536),
    )
    if err != nil {
        span.RecordError(err)
        log.Fatalf("Failed to create store: %v", err)
    }

    // Add documents
    docs := []schema.Document{
        schema.NewDocument("AI is revolutionizing technology.", 
            map[string]string{"category": "ai", "source": "doc1"}),
        schema.NewDocument("Serverless computing scales automatically.", 
            map[string]string{"category": "cloud", "source": "doc2"}),
    }

    ids, err := store.AddDocuments(ctx, docs)
    if err != nil {
        span.RecordError(err)
        log.Fatalf("Failed to add documents: %v", err)
    }

    span.SetAttributes(attribute.Int("documents_added", len(ids)))
    fmt.Printf("Successfully added %d documents\n", len(ids))

    // Search
    results, scores, err := store.SimilaritySearchByQuery(ctx, "cloud computing", 5, embedder)
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
| `api_key` | Pinecone API key | - | Yes |
| `environment` | Pinecone environment | - | Yes |
| `project_id` | Pinecone project ID | - | Yes |
| `index_name` | Index name | - | Yes |
| `embedding_dim` | Embedding dimension | - | Yes |
| `timeout` | Request timeout | `30s` | No |

## Common Issues

### "Index not found"

**Problem**: Index doesn't exist or wrong name.

**Solution**: Create index in Pinecone console or verify index name:# List indexes via Pinecone API
```bash
curl -X GET "https://api.pinecone.io/indexes" \
```
  -H "Api-Key: $PINECONE_API_KEY"
```

### "Invalid API key"

**Problem**: Wrong API key or environment.

**Solution**: Verify API key and environment match:export PINECONE_API_KEY="your-api-key"
bash
```bash
export PINECONE_ENVIRONMENT="us-west1-gcp"
```

### "Dimension mismatch"

**Problem**: Embedding dimension doesn't match index.

**Solution**: Ensure embedding dimension matches index configuration:vectorstores.WithProviderConfig("embedding_dim", 1536), // Must match index
```

## Production Considerations

When using Pinecone Serverless in production:

- **Cost optimization**: Serverless charges per operation - optimize batch operations
- **Rate limits**: Be aware of rate limits and implement backoff
- **Index limits**: Serverless has dimension and size limits
- **Monitoring**: Monitor usage and costs in Pinecone dashboard
- **Backup**: Implement backup strategy for critical data

## Next Steps

Congratulations! You've integrated Pinecone Serverless with Beluga AI. Next, learn how to:

- **[Qdrant Cloud Cluster](./qdrant-cloud-cluster.md)** - Managed Qdrant clusters
- **[Vectorstores Package Documentation](../../api/packages/vectorstores.md)** - Deep dive into vectorstores
- **[RAG Tutorial](../../getting-started/02-simple-rag.md)** - Build RAG applications

---

**Ready for more?** Check out the [Integrations Index](./README.md) for more integration guides!
