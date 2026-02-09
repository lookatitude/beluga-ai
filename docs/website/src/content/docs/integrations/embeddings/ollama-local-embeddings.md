---
title: Ollama Local Embeddings
description: Generate text embeddings locally using Ollama with Beluga AI, enabling private, offline embedding generation for air-gapped and cost-sensitive environments.
---

Ollama runs embedding models locally, eliminating external API calls and per-token costs. When integrated with Beluga AI, Ollama provides fully private embedding generation suitable for air-gapped environments, sensitive data workloads, and development setups where cloud API access is unavailable or undesirable.

## Overview

Beluga AI's `Embedder` interface in the `rag/embedding` package provides a uniform API for all embedding providers. Ollama registers as `"ollama"` in the global registry and is instantiated via the standard `embedding.New` factory. The Ollama provider communicates with a local Ollama server over HTTP.

The recommended model is `nomic-embed-text`, which produces 768-dimensional vectors and provides strong general-purpose embeddings.

## Prerequisites

- Go 1.23 or later
- Beluga AI framework installed
- Ollama installed and running ([ollama.com](https://ollama.com))

## Installation

### Install Ollama

On Linux and macOS:

```bash
curl -fsSL https://ollama.com/install.sh | sh
```

On macOS, you can also install via Homebrew or download the application from [ollama.com](https://ollama.com).

### Start the Server and Pull a Model

```bash
ollama serve
```

In a separate terminal, pull an embedding model:

```bash
ollama pull nomic-embed-text
```

### Verify the Server

```bash
curl http://localhost:11434/api/tags
```

This should return a JSON response listing the installed models.

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

### Basic Local Embeddings

Create an Ollama embedder via the registry and generate embeddings:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/rag/embedding"

    // Register the Ollama provider
    _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/ollama"
)

func main() {
    ctx := context.Background()

    emb, err := embedding.New("ollama", config.ProviderConfig{
        Model:   "nomic-embed-text",
        BaseURL: "http://localhost:11434",
    })
    if err != nil {
        log.Fatal(err)
    }

    texts := []string{
        "The capital of France is Paris.",
        "Go is a statically typed programming language.",
    }

    vectors, err := emb.Embed(ctx, texts)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Generated %d embeddings of dimension %d\n", len(vectors), emb.Dimensions())
}
```

### Single Text Embedding

For embedding queries:

```go
vector, err := emb.EmbedSingle(ctx, "What is the capital of France?")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Query vector dimension: %d\n", len(vector))
```

### Complete Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "math"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/rag/embedding"

    _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/ollama"
)

func main() {
    ctx := context.Background()

    emb, err := embedding.New("ollama", config.ProviderConfig{
        Model:   "nomic-embed-text",
        BaseURL: "http://localhost:11434",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Index some documents
    docs := []string{
        "Paris is the capital and most populous city of France.",
        "Berlin is the capital of Germany.",
        "Go was designed at Google by Robert Griesemer, Rob Pike, and Ken Thompson.",
    }
    docVecs, err := emb.Embed(ctx, docs)
    if err != nil {
        log.Fatal(err)
    }

    // Query
    queryVec, err := emb.EmbedSingle(ctx, "capital of France")
    if err != nil {
        log.Fatal(err)
    }

    // Find the most similar document
    for i, dv := range docVecs {
        sim := cosineSimilarity(queryVec, dv)
        fmt.Printf("Doc %d (%.4f): %s\n", i, sim, docs[i])
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

## Available Models

| Model | Dimensions | Size | Use Case |
|-------|-----------|------|----------|
| `nomic-embed-text` | 768 | ~274 MB | General purpose, recommended |
| `mxbai-embed-large` | 1024 | ~670 MB | Higher accuracy, larger vectors |
| `all-minilm` | 384 | ~45 MB | Lightweight, fast inference |
| `snowflake-arctic-embed` | 1024 | ~670 MB | High-quality retrieval |

Pull additional models as needed:

```bash
ollama pull mxbai-embed-large
ollama pull all-minilm
```

## Advanced Topics

### Custom Server Configuration

Point to a remote Ollama instance or a non-default port:

```go
emb, err := embedding.New("ollama", config.ProviderConfig{
    Model:   "nomic-embed-text",
    BaseURL: "http://gpu-server.internal:11434",
    Timeout: 60 * time.Second,
})
```

### Hooks for Observability

Monitor local embedding performance:

```go
import "log/slog"

hooks := embedding.Hooks{
    BeforeEmbed: func(ctx context.Context, texts []string) error {
        slog.Info("embedding locally", "count", len(texts), "model", "nomic-embed-text")
        return nil
    },
    AfterEmbed: func(ctx context.Context, embeddings [][]float32, err error) {
        if err != nil {
            slog.Error("local embedding failed", "error", err)
        } else {
            slog.Info("local embedding complete", "vectors", len(embeddings))
        }
    },
}
```

### Using with a Vector Store

Connect Ollama embeddings to a vector store for similarity search:

```go
import (
    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/rag/embedding"
    "github.com/lookatitude/beluga-ai/rag/vectorstore"

    _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/ollama"
    _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/pgvector"
)

emb, err := embedding.New("ollama", config.ProviderConfig{
    Model:   "nomic-embed-text",
    BaseURL: "http://localhost:11434",
})
if err != nil {
    log.Fatal(err)
}

store, err := vectorstore.New("pgvector", config.ProviderConfig{
    Options: map[string]any{
        "connection_string": os.Getenv("PGVECTOR_URL"),
        "dimensions":        768.0, // Must match nomic-embed-text dimensions
        "collection":        "local_docs",
    },
})
if err != nil {
    log.Fatal(err)
}
```

### Batch Processing

Ollama processes texts sequentially on most hardware. For large document sets, consider batching to provide progress feedback:

```go
batchSize := 50
for i := 0; i < len(texts); i += batchSize {
    end := min(i+batchSize, len(texts))
    batch := texts[i:end]

    vectors, err := emb.Embed(ctx, batch)
    if err != nil {
        return fmt.Errorf("batch %d: %w", i/batchSize, err)
    }

    log.Printf("embedded batch %d/%d (%d texts)",
        i/batchSize+1, (len(texts)+batchSize-1)/batchSize, len(batch))

    // Store vectors...
}
```

## Configuration

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `Model` | Ollama model name | - | Yes |
| `BaseURL` | Ollama server URL | `http://localhost:11434` | No |
| `Timeout` | Request timeout | `30s` | No |

Provider-specific options can be passed via the `Options` map in `config.ProviderConfig`.

## Troubleshooting

**"connection refused"** -- The Ollama server is not running. Start it with `ollama serve`. Verify it is accessible at the configured URL with `curl http://localhost:11434/api/tags`.

**"model not found"** -- The requested model has not been pulled. Download it with `ollama pull <model-name>`. List installed models with `ollama list`.

**Slow inference** -- Local embedding speed depends on hardware. GPU acceleration significantly improves throughput. Ensure Ollama detects your GPU with `ollama ps`. For CPU-only systems, use smaller models like `all-minilm`.

**High memory usage** -- Embedding models are loaded into memory when first used. The `nomic-embed-text` model requires approximately 600 MB of RAM. Monitor with `ollama ps` and unload unused models with `ollama stop <model>`.

## Production Considerations

- Ollama is designed for local and development use. For production deployments, evaluate the security posture of the Ollama server carefully
- Run the Ollama server in an isolated network segment, not exposed to the public internet
- Monitor resource usage -- embedding models consume significant GPU memory or CPU resources
- Use GPU acceleration (NVIDIA, AMD, or Apple Silicon) for acceptable throughput on large document sets
- Consider running Ollama in a container with resource limits to prevent runaway memory usage
- For multi-tenant deployments, run separate Ollama instances per tenant to isolate model loading and memory

## Related Resources

- [Embedding Providers](/integrations/embedding-providers) -- All embedding provider integrations
- [Cohere Multilingual Embeddings](/integrations/cohere-multilingual) -- Cloud-based multilingual embeddings
- [Vector Stores](/integrations/vector-stores) -- Storing and searching embeddings
