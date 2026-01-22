# Cohere Multilingual Embedder

Welcome, colleague! In this integration guide, we're going to integrate Cohere's multilingual embedding models with Beluga AI. Cohere provides high-quality embeddings that work across 100+ languages.

## What you will build

You will configure Beluga AI to use Cohere's multilingual embedding models, enabling you to generate embeddings for text in multiple languages with a single model, perfect for international applications.

## Learning Objectives

- ✅ Configure Cohere embedder with Beluga AI
- ✅ Generate multilingual embeddings
- ✅ Handle Cohere API authentication
- ✅ Understand multilingual embedding best practices

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- Cohere API key
- Understanding of multilingual text processing

## Step 1: Setup and Installation

Get your Cohere API key from https://cohere.com

Set environment variable:
bash
```bash
export COHERE_API_KEY="your-api-key"
```

## Step 2: Basic Cohere Configuration

Create a Cohere embedder:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/embeddings"
    "github.com/lookatitude/beluga-ai/pkg/embeddings/providers/cohere"
    "go.opentelemetry.io/otel"
)

func main() {
    ctx := context.Background()

    // Create Cohere configuration
    config := &cohere.Config{
        APIKey:     os.Getenv("COHERE_API_KEY"),
        Model:      "embed-multilingual-v3.0",
        Timeout:    30 * time.Second,
        MaxRetries: 3,
        Enabled:    true,
    }

    // Create embedder
    tracer := otel.Tracer("beluga.embeddings.cohere")
    embedder, err := cohere.NewCohereEmbedder(config, tracer)
    if err != nil {
        log.Fatalf("Failed to create embedder: %v", err)
    }

    // Generate multilingual embeddings
    texts := []string{
        "Hello, world!",           // English
        "Bonjour le monde!",       // French
        "Hola, mundo!",            // Spanish
        "你好，世界！",              // Chinese
    }

    embeddings, err := embedder.EmbedDocuments(ctx, texts)
    if err != nil {
        log.Fatalf("Failed to generate embeddings: %v", err)
    }


    fmt.Printf("Generated %d multilingual embeddings\n", len(embeddings))
    fmt.Printf("Embedding dimension: %d\n", len(embeddings[0]))
}
```

### Verification

Run the example:
bash
```bash
export COHERE_API_KEY="your-api-key"
go run main.go
```

You should see:Generated 4 multilingual embeddings
Embedding dimension: 1024
```

## Step 3: Using Embeddings Factory

Use the factory for easier configuration:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/embeddings"
)

func main() {
    ctx := context.Background()

    // Create configuration
    config := embeddings.NewConfig()
    config.Cohere = &embeddings.CohereConfig{
        APIKey:     os.Getenv("COHERE_API_KEY"),
        Model:      "embed-multilingual-v3.0",
        Timeout:    30 * time.Second,
        MaxRetries: 3,
        Enabled:    true,
    }

    // Create factory
    factory, err := embeddings.NewEmbedderFactory(config)
    if err != nil {
        log.Fatalf("Failed to create factory: %v", err)
    }

    // Create embedder
    embedder, err := factory.CreateEmbedder("cohere")
    if err != nil {
        log.Fatalf("Failed to create embedder: %v", err)
    }

    // Generate embedding for query
    query := "What is machine learning?"
    embedding, err := embedder.EmbedQuery(ctx, query)
    if err != nil {
        log.Fatalf("Failed to generate embedding: %v", err)
    }


    fmt.Printf("Query embedding dimension: %d\n", len(embedding))
}
```

## Step 4: Available Models

Cohere offers several multilingual models:
// Multilingual v3.0 (recommended, 1024 dimensions)
config.Model = "embed-multilingual-v3.0"

// Multilingual v2.0 (768 dimensions)
config.Model = "embed-multilingual-v2.0"

// English-only models
config.Model = "embed-english-v3.0"  // 1024 dimensions
config.Model = "embed-english-light-v3.0"  // 384 dimensions
```

## Step 5: Input Types

Cohere supports different input types:
// Text input (default)
embedding, err := embedder.EmbedQuery(ctx, "Hello, world!")

// With input type specification
// Note: Cohere embedder handles input types automatically

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `APIKey` | Cohere API key | - | Yes |
| `Model` | Embedding model | `embed-multilingual-v3.0` | No |
| `Timeout` | Request timeout | `30s` | No |
| `MaxRetries` | Maximum retry attempts | `3` | No |
| `Truncate` | Truncate input text | `NONE` | No |

## Common Issues

### "Invalid API key"

**Problem**: API key not set or invalid.

**Solution**: Verify API key:export COHERE_API_KEY="your-api-key"
```
echo $COHERE_API_KEY

### "Rate limit exceeded"

**Problem**: Too many requests.

**Solution**: Implement rate limiting or use retry logic:config.MaxRetries = 5
config.Timeout = 60 * time.Second
```

## Production Considerations

When using Cohere in production:

- **Rate Limits**: Monitor and handle rate limits
- **Cost Management**: Track API usage and costs
- **Language Support**: Verify language support for your use case
- **Batch Processing**: Use batch endpoints for multiple texts
- **Error Handling**: Implement retry logic for transient failures

## Complete Example

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
    "github.com/lookatitude/beluga-ai/pkg/embeddings/providers/cohere"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    // Create configuration
    config := &cohere.Config{
        APIKey:     os.Getenv("COHERE_API_KEY"),
        Model:      "embed-multilingual-v3.0",
        Timeout:    30 * time.Second,
        MaxRetries: 5,
        Enabled:    true,
    }

    // Create embedder with tracing
    tracer := otel.Tracer("beluga.embeddings.cohere")
    embedder, err := cohere.NewCohereEmbedder(config, tracer)
    if err != nil {
        log.Fatalf("Failed to create embedder: %v", err)
    }

    // Generate multilingual embeddings
    ctx, span := tracer.Start(ctx, "embeddings.multilingual",
        trace.WithAttributes(
            attribute.String("model", config.Model),
            attribute.String("provider", "cohere"),
        ),
    )
    defer span.End()

    texts := []string{
        "Machine learning is a subset of artificial intelligence.",
        "L'apprentissage automatique est un sous-ensemble de l'intelligence artificielle.",
        "El aprendizaje automático es un subconjunto de la inteligencia artificial.",
    }

    embeddings, err := embedder.EmbedDocuments(ctx, texts)
    if err != nil {
        span.RecordError(err)
        log.Fatalf("Failed to generate embeddings: %v", err)
    }



    span.SetAttributes(
        attribute.Int("embedding_count", len(embeddings)),
        attribute.Int("embedding_dimension", len(embeddings[0])),
    )

    fmt.Printf("Successfully generated %d multilingual embeddings\n", len(embeddings))
    fmt.Printf("Embedding dimension: %d\n", len(embeddings[0]))
}
```

## Next Steps

Congratulations! You've integrated Cohere multilingual embeddings with Beluga AI. Next, learn how to:

- **[Ollama Local Embeddings](./ollama-local-embeddings.md)** - Local embedding generation
- **[Embeddings Package Documentation](../../api/packages/embeddings.md)** - Deep dive into embeddings package
- **[RAG Tutorial](../../getting-started/02-simple-rag.md)** - Use embeddings in RAG

---

**Ready for more?** Check out the [Integrations Index](./README.md) for more integration guides!
