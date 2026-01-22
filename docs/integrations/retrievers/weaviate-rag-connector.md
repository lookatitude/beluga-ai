# Weaviate RAG Connector

Welcome, colleague! In this integration guide, we're going to integrate Weaviate for RAG with Beluga AI's retrievers package. Weaviate is a vector database that combines vector search with keyword search for powerful hybrid retrieval.

## What you will build

You will configure Beluga AI to use Weaviate for RAG retrieval, enabling hybrid search that combines semantic similarity with keyword matching, metadata filtering, and graph-based relationships.

## Learning Objectives

- ✅ Configure Weaviate with Beluga AI
- ✅ Create Weaviate retriever
- ✅ Perform hybrid vector + keyword search
- ✅ Use Weaviate's graph capabilities

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- Weaviate instance (local or cloud)
- Weaviate Go client

## Step 1: Setup and Installation

Install Weaviate Go client:
bash
```bash
go get github.com/weaviate/weaviate-go-client/v4
```

Start Weaviate (local):
```bash
docker run -d -p 8080:8080 -p 50051:50051 \
```
  -e QUERY_DEFAULTS_LIMIT=25 \
  -e AUTHENTICATION_ANONYMOUS_ACCESS_ENABLED=true \
  -e PERSISTENCE_DATA_PATH='/var/lib/weaviate' \
  semitechnologies/weaviate:latest
```

## Step 2: Create Weaviate Retriever

Create a Weaviate-based retriever:
```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/weaviate/weaviate-go-client/v4/weaviate"
    "github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
    "github.com/lookatitude/beluga-ai/pkg/core"
    "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

type WeaviateRetriever struct {
    client     *weaviate.Client
    className  string
    embedder   iface.Embedder
    defaultK   int
}

func NewWeaviateRetriever(weaviateURL, className string, embedder iface.Embedder, defaultK int) (*WeaviateRetriever, error) {
    cfg := weaviate.Config{
        Host:   weaviateURL,
        Scheme: "http",
    }
    
    client, err := weaviate.NewClient(cfg)
    if err != nil {
        return nil, fmt.Errorf("failed to create client: %w", err)
    }
    
    return &WeaviateRetriever{
        client:    client,
        className: className,
        embedder:  embedder,
        defaultK:  defaultK,
    }, nil
}

func (r *WeaviateRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
    // Generate query embedding
    queryEmbedding, err := r.embedder.EmbedQuery(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("failed to embed query: %w", err)
    }
    
    // Convert to float32
    vector := make([]float32, len(queryEmbedding))
    for i, v := range queryEmbedding {
        vector[i] = float32(v)
    }
    
    // Build GraphQL query
    builder := r.client.GraphQL().Get().
        WithClassName(r.className).
        WithFields(graphql.Field{
            Name: "content",
        }, graphql.Field{
            Name: "metadata",
        }).
        WithNearVector(&graphql.NearVectorArgumentBuilder{}.
            WithVector(vector).
            WithCertainty(0.7))
    
    result, err := builder.Do(ctx)
    if err != nil {
        return nil, fmt.Errorf("search failed: %w", err)
    }
    
    // Parse results
    get := result.Data["Get"].(map[string]interface{})
    items := get[r.className].([]interface{})
    
    documents := make([]schema.Document, 0, len(items))
    for _, item := range items {
        itemMap := item.(map[string]interface{})
        content, _ := itemMap["content"].(string)
        metadata, _ := itemMap["metadata"].(map[string]interface{})
        
        meta := make(map[string]string)
        for k, v := range metadata {
            meta[k] = fmt.Sprintf("%v", v)
        }
        
        documents = append(documents, schema.NewDocument(content, meta))
    }
    
    return documents, nil
}
```

## Step 3: Hybrid Search

Use Weaviate's hybrid search (vector + keyword):
```go
func (r *WeaviateRetriever) HybridSearch(ctx context.Context, query string, alpha float32) ([]schema.Document, error) {
    // Generate embedding
    queryEmbedding, err := r.embedder.EmbedQuery(ctx, query)
    if err != nil {
        return nil, err
    }
    
    vector := make([]float32, len(queryEmbedding))
    for i, v := range queryEmbedding {
        vector[i] = float32(v)
    }
    
    // Hybrid search
    builder := r.client.GraphQL().Get().
        WithClassName(r.className).
        WithFields(graphql.Field{Name: "content"}, graphql.Field{Name: "metadata"}).
        WithHybrid(&graphql.HybridArgumentBuilder{}.
            WithQuery(query).
            WithVector(vector).
            WithAlpha(alpha))
    
    result, err := builder.Do(ctx)
    if err != nil {
        return nil, err
    }
    
    // Parse and return documents
    // ... (similar to GetRelevantDocuments)
    return documents, nil
}
```

## Step 4: Use with Beluga AI

Use Weaviate retriever in RAG pipeline:
```go
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
    
    // Create retriever
    retriever, err := NewWeaviateRetriever("http://localhost:8080", "Document", embedder, 5)
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }
    
    // Retrieve documents
    docs, err := retriever.GetRelevantDocuments(ctx, "machine learning")
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }

    
    fmt.Printf("Retrieved %d documents\n", len(docs))
}
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
    "github.com/weaviate/weaviate-go-client/v4/weaviate"
    "github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

type ProductionWeaviateRetriever struct {
    client    *weaviate.Client
    className string
    embedder  iface.Embedder
    defaultK  int
    tracer    trace.Tracer
}

func NewProductionWeaviateRetriever(weaviateURL, className string, embedder iface.Embedder, defaultK int) (*ProductionWeaviateRetriever, error) {
    cfg := weaviate.Config{
        Host:   weaviateURL,
        Scheme: "http",
    }
    
    client, err := weaviate.NewClient(cfg)
    if err != nil {
        return nil, fmt.Errorf("failed to create client: %w", err)
    }
    
    return &ProductionWeaviateRetriever{
        client:    client,
        className: className,
        embedder:  embedder,
        defaultK:  defaultK,
        tracer:    otel.Tracer("beluga.retrievers.weaviate"),
    }, nil
}

func (r *ProductionWeaviateRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
    ctx, span := r.tracer.Start(ctx, "weaviate.search",
        trace.WithAttributes(
            attribute.String("class", r.className),
            attribute.String("query", query),
        ),
    )
    defer span.End()
    
    // Generate embedding
    queryEmbedding, err := r.embedder.EmbedQuery(ctx, query)
    if err != nil {
        span.RecordError(err)
        return nil, fmt.Errorf("embedding failed: %w", err)
    }
    
    vector := make([]float32, len(queryEmbedding))
    for i, v := range queryEmbedding {
        vector[i] = float32(v)
    }
    
    // Search
    builder := r.client.GraphQL().Get().
        WithClassName(r.className).
        WithFields(graphql.Field{Name: "content"}, graphql.Field{Name: "metadata"}).
        WithLimit(r.defaultK).
        WithNearVector(&graphql.NearVectorArgumentBuilder{}.
            WithVector(vector).
            WithCertainty(0.7))
    
    result, err := builder.Do(ctx)
    if err != nil {
        span.RecordError(err)
        return nil, fmt.Errorf("search failed: %w", err)
    }
    
    // Parse results
    get := result.Data["Get"].(map[string]interface{})
    items := get[r.className].([]interface{})
    
    documents := make([]schema.Document, 0, len(items))
    for _, item := range items {
        itemMap := item.(map[string]interface{})
        content, _ := itemMap["content"].(string)
        metadata, _ := itemMap["metadata"].(map[string]interface{})
        
        meta := make(map[string]string)
        for k, v := range metadata {
            meta[k] = fmt.Sprintf("%v", v)
        }
        
        documents = append(documents, schema.NewDocument(content, meta))
    }
    
    span.SetAttributes(attribute.Int("documents_retrieved", len(documents)))
    return documents, nil
}

func main() {
    ctx := context.Background()
    
    // Setup embedder
    embedderConfig := embeddings.NewConfig()
    embedderConfig.OpenAI = &embeddings.OpenAIConfig{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    }
    embedderFactory, _ := embeddings.NewEmbedderFactory(embedderConfig)
    embedder, _ := embedderFactory.CreateEmbedder("openai")
    
    // Create retriever
    retriever, err := NewProductionWeaviateRetriever("http://localhost:8080", "Document", embedder, 5)
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }
    
    docs, err := retriever.GetRelevantDocuments(ctx, "AI and machine learning")
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }

    
    fmt.Printf("Retrieved %d documents\n", len(docs))
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `WeaviateURL` | Weaviate server URL | `http://localhost:8080` | No |
| `ClassName` | Weaviate class name | `Document` | No |
| `DefaultK` | Default number of results | `5` | No |
| `Alpha` | Hybrid search weight | `0.5` | No |

## Common Issues

### "Connection refused"

**Problem**: Weaviate not running.

**Solution**: Start Weaviate:docker run -p 8080:8080 semitechnologies/weaviate:latest
```

### "Class not found"

**Problem**: Weaviate class doesn't exist.

**Solution**: Create class via Weaviate API or client.

## Production Considerations

When using Weaviate in production:

- **Schema design**: Design classes for your data model
- **Vectorization**: Configure vectorization modules
- **Hybrid search**: Use hybrid search for best results
- **GraphQL**: Leverage Weaviate's GraphQL API
- **Monitoring**: Monitor query performance

## Next Steps

Congratulations! You've integrated Weaviate with Beluga AI. Next, learn how to:

- **[Elasticsearch Keyword Search](./elasticsearch-keyword-search.md)** - Elasticsearch integration
- **[Retrievers Package Documentation](../../api/packages/retrievers.md)** - Deep dive into retrievers
- **[RAG Tutorial](../../getting-started/02-simple-rag.md)** - Build RAG applications

---

**Ready for more?** Check out the [Integrations Index](./README.md) for more integration guides!
