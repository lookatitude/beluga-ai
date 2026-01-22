# Ollama Local Embeddings

Welcome, colleague! In this integration guide, we're going to integrate Ollama for local embeddings with Beluga AI's embeddings package. Ollama allows you to run embedding models locally, providing privacy and eliminating API costs.

## What you will build

You will configure Beluga AI to use Ollama for generating embeddings locally, enabling private, offline embedding generation without external API calls. This is perfect for sensitive data or air-gapped environments.

## Learning Objectives

- ✅ Install and configure Ollama
- ✅ Use Ollama embedding models with Beluga AI
- ✅ Generate embeddings locally
- ✅ Understand security considerations

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- Ollama installed and running locally
- Experimental build tag enabled (for security)

## Step 1: Setup and Installation

Install Ollama:
# macOS/Linux
```bash
curl -fsSL https://ollama.com/install.sh | sh
```

# Or download from https://ollama.com
```

Start Ollama server:
ollama serve
```

Pull an embedding model:
ollama pull nomic-embed-text
```

## Step 2: Build with Experimental Tag

Ollama provider requires the experimental build tag due to security considerations:
bash
```bash
go build -tags experimental ./...
```

Or in your code:
```
//go:build experimental

## Step 3: Basic Ollama Configuration

Create an Ollama embedder:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/embeddings"
    "github.com/lookatitude/beluga-ai/pkg/embeddings/providers/ollama"
    "go.opentelemetry.io/otel"
)

func main() {
    ctx := context.Background()

    // Create Ollama configuration
    config := &ollama.Config{
        ServerURL: "http://localhost:11434",
        Model:     "nomic-embed-text",
        Timeout:   30 * time.Second,
        MaxRetries: 3,
        Enabled:   true,
    }

    // Create embedder
    tracer := otel.Tracer("beluga.embeddings.ollama")
    embedder, err := ollama.NewOllamaEmbedder(config, tracer)
    if err != nil {
        log.Fatalf("Failed to create embedder: %v", err)
    }

    // Generate embeddings
    texts := []string{
        "The capital of France is Paris.",
        "Go is a programming language.",
    }

    embeddings, err := embedder.EmbedDocuments(ctx, texts)
    if err != nil {
        log.Fatalf("Failed to generate embeddings: %v", err)
    }


    fmt.Printf("Generated %d embeddings\n", len(embeddings))
    fmt.Printf("First embedding dimension: %d\n", len(embeddings[0]))
}
```

### Verification

Run the example:
bash
```bash
go run -tags experimental main.go
```

You should see:Generated 2 embeddings
First embedding dimension: 768
```

## Step 4: Using Embeddings Factory

Use the embeddings factory for easier configuration:
```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/pkg/embeddings"
)

func main() {
    ctx := context.Background()

    // Create configuration
    config := embeddings.NewConfig()
    config.Ollama = &embeddings.OllamaConfig{
        ServerURL:  "http://localhost:11434",
        Model:      "nomic-embed-text",
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
    embedder, err := factory.CreateEmbedder("ollama")
    if err != nil {
        log.Fatalf("Failed to create embedder: %v", err)
    }

    // Generate embeddings
    text := "Hello, world!"
    embedding, err := embedder.EmbedQuery(ctx, text)
    if err != nil {
        log.Fatalf("Failed to generate embedding: %v", err)
    }


    fmt.Printf("Embedding dimension: %d\n", len(embedding))
}
```

## Step 5: Available Models

Ollama supports various embedding models:
// Nomic Embed Text (recommended)
config.Model = "nomic-embed-text"  // 768 dimensions

// All-MiniLM
config.Model = "all-minilm"  // 384 dimensions

// Custom models
config.Model = "your-custom-model"
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `ServerURL` | Ollama server URL | `http://localhost:11434` | No |
| `Model` | Embedding model name | `nomic-embed-text` | Yes |
| `Timeout` | Request timeout | `30s` | No |
| `MaxRetries` | Maximum retry attempts | `3` | No |
| `KeepAlive` | Model keep-alive duration | `5m` | No |

## Common Issues

### "Connection refused"

**Problem**: Ollama server not running.

**Solution**: Start Ollama server:ollama serve
```

### "Model not found"

**Problem**: Model not pulled.

**Solution**: Pull the model:ollama pull nomic-embed-text
```

### "Build tag required"

**Problem**: Experimental tag not enabled.

**Solution**: Build with experimental tag:go build -tags experimental ./...
```

## Production Considerations

When using Ollama in production:

- **Security**: Ollama has known CVEs - use only in isolated environments
- **Performance**: Local models may be slower than cloud APIs
- **Resource Usage**: Embedding models require significant memory
- **Monitoring**: Monitor model performance and resource usage
- **Isolation**: Use in air-gapped or properly secured networks

## Complete Example

Here's a complete, production-ready example:
//go:build experimental

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/embeddings"
    "github.com/lookatitude/beluga-ai/pkg/embeddings/providers/ollama"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    // Create configuration
    config := &ollama.Config{
        ServerURL:  "http://localhost:11434",
        Model:      "nomic-embed-text",
        Timeout:    30 * time.Second,
        MaxRetries: 3,
        KeepAlive:  "5m",
        Enabled:    true,
    }

    // Create embedder with tracing
    tracer := otel.Tracer("beluga.embeddings.ollama")
    embedder, err := ollama.NewOllamaEmbedder(config, tracer)
    if err != nil {
        log.Fatalf("Failed to create embedder: %v", err)
    }

    // Generate embeddings with tracing
    ctx, span := tracer.Start(ctx, "embeddings.generate",
        trace.WithAttributes(
            attribute.String("model", config.Model),
            attribute.Int("text_count", 2),
        ),
    )
    defer span.End()

    texts := []string{
        "The capital of France is Paris.",
        "Go is a programming language developed by Google.",
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

    fmt.Printf("Successfully generated %d embeddings\n", len(embeddings))
    fmt.Printf("Embedding dimension: %d\n", len(embeddings[0]))
}
```

## Next Steps

Congratulations! You've integrated Ollama local embeddings with Beluga AI. Next, learn how to:

- **[Cohere Multilingual Embedder](./cohere-multilingual-embedder.md)** - Multilingual embeddings
- **[Embeddings Package Documentation](../../api/packages/embeddings.md)** - Deep dive into embeddings package
- **[RAG Tutorial](../../getting-started/02-simple-rag.md)** - Use embeddings in RAG

---

**Ready for more?** Check out the [Integrations Index](./README.md) for more integration guides!
