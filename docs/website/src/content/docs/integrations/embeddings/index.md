---
title: Embedding Providers
description: Generate text embeddings with OpenAI, Cohere, Google, Ollama, Jina, Voyage, Mistral, and more for RAG pipelines.
sidebar:
  order: 0
---

Embeddings convert text into numerical vectors that capture semantic meaning, enabling similarity search, clustering, and retrieval-augmented generation (RAG). The choice of embedding provider affects retrieval quality, latency, cost, and data privacy -- there is no single best option for all use cases.

Beluga AI provides a unified `Embedder` interface for converting text into vector embeddings. All embedding providers register via `init()` and follow the same registry pattern used across the framework, so you can evaluate different providers without changing your pipeline code.

## Provider Overview

| Provider | Registry Name | Dimensions | Import Path |
|----------|--------------|-----------|-------------|
| OpenAI | `openai` | 1536, 3072 | `rag/embedding/providers/openai` |
| Cohere | `cohere` | 384-1024 | `rag/embedding/providers/cohere` |
| Google | `google` | 768 | `rag/embedding/providers/google` |
| Ollama | `ollama` | Varies by model | `rag/embedding/providers/ollama` |
| Jina | `jina` | 768-1024 | `rag/embedding/providers/jina` |
| Voyage | `voyage` | 1024 | `rag/embedding/providers/voyage` |
| Mistral | `mistral` | 1024 | `rag/embedding/providers/mistral` |
| Sentence Transformers | `sentence_transformers` | Varies | `rag/embedding/providers/sentence_transformers` |
| In-Memory | `inmemory` | Configurable | `rag/embedding/providers/inmemory` |

## Common Pattern

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/rag/embedding"

    // Import the provider
    _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/openai"
)

func main() {
    ctx := context.Background()

    // Create embedder via registry
    emb, err := embedding.New("openai", config.ProviderConfig{
        APIKey: "sk-...",
        Model:  "text-embedding-3-small",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Embed text
    vectors, err := emb.Embed(ctx, []string{"Hello, world!", "How are you?"})
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Generated %d embeddings of dimension %d\n", len(vectors), emb.Dimensions())
}
```

## Embedder Interface

All providers implement:

```go
type Embedder interface {
    // Embed produces embeddings for a batch of texts.
    Embed(ctx context.Context, texts []string) ([][]float32, error)

    // EmbedSingle embeds a single text and returns its vector.
    EmbedSingle(ctx context.Context, text string) ([]float32, error)

    // Dimensions returns the vector dimensionality.
    Dimensions() int
}
```

## OpenAI Embeddings

OpenAI provides `text-embedding-3-small` (1536d) and `text-embedding-3-large` (3072d).

```bash
export OPENAI_API_KEY="sk-..."
```

```go
import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/openai"

emb, err := embedding.New("openai", config.ProviderConfig{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "text-embedding-3-small",
})
```

| Model | Dimensions | Use Case |
|-------|-----------|----------|
| `text-embedding-3-small` | 1536 | General purpose, cost-effective |
| `text-embedding-3-large` | 3072 | Higher accuracy, more expensive |
| `text-embedding-ada-002` | 1536 | Legacy, widely compatible |

## Cohere Embeddings

Cohere provides multilingual embeddings across 100+ languages.

```bash
export COHERE_API_KEY="..."
```

```go
import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/cohere"

emb, err := embedding.New("cohere", config.ProviderConfig{
    APIKey: os.Getenv("COHERE_API_KEY"),
    Model:  "embed-multilingual-v3.0",
})
```

| Model | Dimensions | Languages |
|-------|-----------|-----------|
| `embed-multilingual-v3.0` | 1024 | 100+ languages |
| `embed-english-v3.0` | 1024 | English only |
| `embed-english-light-v3.0` | 384 | English only, lightweight |

Cohere embeddings are particularly suited for cross-language retrieval tasks where queries and documents may be in different languages.

## Google Embeddings

Google Vertex AI provides embeddings via the Gemini API.

```bash
export GOOGLE_API_KEY="AIza..."
```

```go
import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/google"

emb, err := embedding.New("google", config.ProviderConfig{
    APIKey: os.Getenv("GOOGLE_API_KEY"),
    Model:  "text-embedding-004",
})
```

## Ollama (Local Embeddings)

Run embedding models locally with no external API calls.

```bash
ollama pull nomic-embed-text
```

```go
import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/ollama"

emb, err := embedding.New("ollama", config.ProviderConfig{
    Model:   "nomic-embed-text",
    BaseURL: "http://localhost:11434",
})
```

Local embeddings are useful for air-gapped environments, reducing latency, and avoiding per-token costs. Popular models include `nomic-embed-text`, `mxbai-embed-large`, and `all-minilm`.

## Jina Embeddings

Jina AI provides embeddings optimized for retrieval tasks.

```bash
export JINA_API_KEY="..."
```

```go
import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/jina"

emb, err := embedding.New("jina", config.ProviderConfig{
    APIKey: os.Getenv("JINA_API_KEY"),
    Model:  "jina-embeddings-v3",
})
```

## Voyage Embeddings

Voyage AI provides high-quality embeddings for code and text.

```bash
export VOYAGE_API_KEY="..."
```

```go
import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/voyage"

emb, err := embedding.New("voyage", config.ProviderConfig{
    APIKey: os.Getenv("VOYAGE_API_KEY"),
    Model:  "voyage-3",
})
```

Voyage embeddings are popular for code-related retrieval. Use `voyage-code-3` for code search applications.

## Mistral Embeddings

Mistral provides European-hosted embeddings.

```bash
export MISTRAL_API_KEY="..."
```

```go
import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/mistral"

emb, err := embedding.New("mistral", config.ProviderConfig{
    APIKey: os.Getenv("MISTRAL_API_KEY"),
    Model:  "mistral-embed",
})
```

## Sentence Transformers

Use sentence-transformer models via a local inference server.

```go
import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/sentence_transformers"

emb, err := embedding.New("sentence_transformers", config.ProviderConfig{
    BaseURL: "http://localhost:8080",
    Model:   "all-MiniLM-L6-v2",
})
```

## In-Memory Embeddings

The in-memory embedder generates deterministic hash-based vectors for testing and development.

```go
import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/inmemory"

emb, err := embedding.New("inmemory", config.ProviderConfig{
    Options: map[string]any{
        "dimensions": 384.0,
    },
})
```

Do not use the in-memory embedder for production — it does not produce semantically meaningful vectors.

## Middleware

Add caching, logging, or tracing to any embedder:

```go
emb = embedding.ApplyMiddleware(emb,
    embedding.WithLogging(slog.Default()),
)
```

## Hooks

Attach callbacks to embedding operations:

```go
hooks := embedding.Hooks{
    BeforeEmbed: func(ctx context.Context, texts []string) error {
        slog.Info("embedding", "count", len(texts))
        return nil
    },
    AfterEmbed: func(ctx context.Context, embeddings [][]float32, err error) {
        if err == nil {
            slog.Info("embedded", "vectors", len(embeddings))
        }
    },
}
```

## Batch Processing

For large document sets, embed in batches to manage memory and rate limits:

```go
batchSize := 100
for i := 0; i < len(texts); i += batchSize {
    end := min(i+batchSize, len(texts))
    batch := texts[i:end]

    vectors, err := emb.Embed(ctx, batch)
    if err != nil {
        return fmt.Errorf("batch %d: %w", i/batchSize, err)
    }
    // Store vectors...
}
```

## Choosing an Embedding Provider

| Need | Recommended Provider |
|------|---------------------|
| General purpose | OpenAI `text-embedding-3-small` |
| Multilingual | Cohere `embed-multilingual-v3.0` |
| Code search | Voyage `voyage-code-3` |
| Local/offline | Ollama `nomic-embed-text` |
| Cost-sensitive | Ollama or Sentence Transformers |
| European data residency | Mistral `mistral-embed` |

Match the embedding provider to your vector store's dimension requirements. The embedding dimension must match the dimension configured in your vector store index.
