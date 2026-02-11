---
title: Cohere Multilingual Embeddings
description: Generate multilingual text embeddings across 100+ languages using Cohere's embed-multilingual models with Beluga AI's embedding interface.
---

Most embedding models are trained primarily on English text and degrade on other languages. Cohere's multilingual models are trained across 100+ languages and produce vectors where semantically similar text clusters together regardless of language. This means a query in Spanish can match a document written in French or Japanese.

This cross-language capability is essential for organizations serving international users, processing multilingual knowledge bases, or building customer support systems that need to retrieve answers across language boundaries.

## Overview

Beluga AI's `Embedder` interface in the `rag/embedding` package provides a uniform API for all embedding providers. Cohere registers as `"cohere"` in the global registry and is instantiated via the standard `embedding.New` factory. No Cohere-specific API is needed beyond configuration.

The recommended model is `embed-multilingual-v3.0`, which produces 1024-dimensional vectors and supports 100+ languages.

## Prerequisites

- Go 1.23 or later
- Beluga AI framework installed
- Cohere API key (obtain from [cohere.com](https://cohere.com))

## Installation

The Cohere embedding provider is included in the Beluga AI module. No additional dependencies are required beyond the framework itself.

Set your API key:

```bash
export COHERE_API_KEY="your-api-key"
```

## The Embedder Interface

All embedding providers implement:

```go
type Embedder interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    EmbedSingle(ctx context.Context, text string) ([]float32, error)
    Dimensions() int
}
```

## Usage

### Basic Multilingual Embeddings

Create a Cohere embedder via the registry and generate embeddings for text in multiple languages:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/rag/embedding"

    // Register the Cohere provider
    _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/cohere"
)

func main() {
    ctx := context.Background()

    emb, err := embedding.New("cohere", config.ProviderConfig{
        APIKey: os.Getenv("COHERE_API_KEY"),
        Model:  "embed-multilingual-v3.0",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Embed text in multiple languages
    texts := []string{
        "Machine learning is a subset of artificial intelligence.",  // English
        "L'apprentissage automatique est un sous-ensemble de l'IA.", // French
        "El aprendizaje automatico es un subconjunto de la IA.",     // Spanish
    }

    vectors, err := emb.Embed(ctx, texts)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Generated %d embeddings of dimension %d\n", len(vectors), emb.Dimensions())
}
```

### Single Text Embedding

For embedding a query or individual text:

```go
vector, err := emb.EmbedSingle(ctx, "What is machine learning?")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Query vector dimension: %d\n", len(vector))
```

### Cross-Language Retrieval

Cohere multilingual embeddings map semantically similar text to nearby vectors regardless of language. This enables queries in one language to retrieve documents written in another:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "math"
    "os"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/rag/embedding"

    _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/cohere"
)

func main() {
    ctx := context.Background()

    emb, err := embedding.New("cohere", config.ProviderConfig{
        APIKey: os.Getenv("COHERE_API_KEY"),
        Model:  "embed-multilingual-v3.0",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Documents in different languages
    docs := []string{
        "The capital of France is Paris.",
        "La capitale de la France est Paris.",
        "Go is a statically typed programming language.",
    }
    docVecs, err := emb.Embed(ctx, docs)
    if err != nil {
        log.Fatal(err)
    }

    // Query in Spanish
    queryVec, err := emb.EmbedSingle(ctx, "Cual es la capital de Francia?")
    if err != nil {
        log.Fatal(err)
    }

    // Compute cosine similarity
    for i, dv := range docVecs {
        sim := cosineSimilarity(queryVec, dv)
        fmt.Printf("Doc %d: %.4f - %s\n", i, sim, docs[i])
    }
}

func cosineSimilarity(a, b []float32) float64 {
    var dot, normA, normB float64
    for i := range a {
        dot += float64(a[i]) * float64(b[i])
        normA += float64(a[i]) * float64(a[i])
        normB += float64(b[i]) * float64(b[i])
    }
    return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
```

The French and English documents about Paris will score higher than the unrelated programming document, demonstrating cross-language semantic matching.

## Available Models

| Model | Dimensions | Languages | Use Case |
|-------|-----------|-----------|----------|
| `embed-multilingual-v3.0` | 1024 | 100+ | Recommended for multilingual applications |
| `embed-multilingual-v2.0` | 768 | 100+ | Legacy multilingual model |
| `embed-english-v3.0` | 1024 | English | Higher accuracy for English-only workloads |
| `embed-english-light-v3.0` | 384 | English | Lightweight English model, lower cost |

## Advanced Topics

### Batch Processing

For large document sets, process embeddings in batches to manage memory and respect API rate limits:

```go
batchSize := 96 // Cohere supports up to 96 texts per request
for i := 0; i < len(texts); i += batchSize {
    end := min(i+batchSize, len(texts))
    batch := texts[i:end]

    vectors, err := emb.Embed(ctx, batch)
    if err != nil {
        return fmt.Errorf("batch %d: %w", i/batchSize, err)
    }
    // Store vectors in your vector store...
}
```

### Hooks for Observability

Attach hooks to monitor embedding operations:

```go
import "log/slog"

hooks := embedding.Hooks{
    BeforeEmbed: func(ctx context.Context, texts []string) error {
        slog.Info("embedding texts", "count", len(texts), "provider", "cohere")
        return nil
    },
    AfterEmbed: func(ctx context.Context, embeddings [][]float32, err error) {
        if err != nil {
            slog.Error("embedding failed", "error", err)
        } else {
            slog.Info("embedding complete", "vectors", len(embeddings))
        }
    },
}
```

### Using with a Vector Store

Connect Cohere embeddings to a vector store for similarity search:

```go
import (
    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/rag/embedding"
    "github.com/lookatitude/beluga-ai/rag/vectorstore"

    _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/cohere"
    _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/pgvector"
)

emb, err := embedding.New("cohere", config.ProviderConfig{
    APIKey: os.Getenv("COHERE_API_KEY"),
    Model:  "embed-multilingual-v3.0",
})
if err != nil {
    log.Fatal(err)
}

store, err := vectorstore.New("pgvector", config.ProviderConfig{
    Options: map[string]any{
        "connection_string": os.Getenv("PGVECTOR_URL"),
        "dimensions":        1024.0, // Must match embedding dimensions
        "collection":        "multilingual_docs",
    },
})
if err != nil {
    log.Fatal(err)
}
```

## Configuration

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `APIKey` | Cohere API key | - | Yes |
| `Model` | Embedding model name | `embed-multilingual-v3.0` | No |
| `Timeout` | Request timeout | `30s` | No |

Provider-specific options can be passed via the `Options` map in `config.ProviderConfig`.

## Troubleshooting

**"invalid api token"** -- Verify your API key is set correctly and has not been revoked. Check with `echo $COHERE_API_KEY`.

**"rate limit exceeded"** -- Reduce batch sizes or add delays between requests. Cohere's rate limits vary by plan tier. Use the `resilience` package for automatic retry with exponential backoff.

**Dimension mismatch with vector store** -- Ensure the vector store index is configured with the same dimensionality as the chosen model (1024 for `embed-multilingual-v3.0`, 768 for v2.0, 384 for light models).

## Production Considerations

- Monitor API usage and costs through the Cohere dashboard
- Use batch endpoints for bulk embedding to reduce request overhead
- Verify language support for your specific use case at [Cohere's documentation](https://docs.cohere.com)
- Store the model name alongside embeddings to prevent mixing vectors from different models
- Consider the `embed-english-light-v3.0` model for English-only workloads to reduce cost and latency

## Related Resources

- [Embedding Providers](/integrations/embedding-providers) -- All embedding provider integrations
- [Ollama Local Embeddings](/integrations/ollama-local-embeddings) -- Local embedding generation
- [Vector Stores](/integrations/vector-stores) -- Storing and searching embeddings
