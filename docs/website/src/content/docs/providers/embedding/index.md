---
title: Embedding Providers
description: Overview of all embedding providers available in Beluga AI v2.
---

Beluga AI v2 provides a unified `embedding.Embedder` interface for converting text into dense vector representations. All providers register via `init()` and are instantiated through the global registry.

## Interface

```go
type Embedder interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    EmbedSingle(ctx context.Context, text string) ([]float32, error)
    Dimensions() int
}
```

## Registry Usage

```go
import (
    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/rag/embedding"

    // Register the provider you need via blank import
    _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/openai"
)

func main() {
    emb, err := embedding.New("openai", config.ProviderConfig{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })
    if err != nil {
        log.Fatal(err)
    }

    vectors, err := emb.Embed(context.Background(), []string{"hello world"})
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Dimensions: %d\n", emb.Dimensions())
}
```

## Available Providers

| Provider | Registry Name | Default Model | Default Dimensions |
|---|---|---|---|
| [OpenAI](/providers/embedding/openai) | `openai` | `text-embedding-3-small` | 1536 |
| [Cohere](/providers/embedding/cohere) | `cohere` | `embed-english-v3.0` | 1024 |
| [Google](/providers/embedding/google) | `google` | `text-embedding-004` | 768 |
| [Ollama](/providers/embedding/ollama) | `ollama` | `nomic-embed-text` | 768 |
| [Jina](/providers/embedding/jina) | `jina` | `jina-embeddings-v2-base-en` | 768 |
| [Voyage](/providers/embedding/voyage) | `voyage` | `voyage-2` | 1024 |
| [Mistral](/providers/embedding/mistral) | `mistral` | `mistral-embed` | 1024 |
| [Sentence Transformers](/providers/embedding/sentence-transformers) | `sentence_transformers` | `all-MiniLM-L6-v2` | 384 |
| [In-Memory](/providers/embedding/inmemory) | `inmemory` | N/A (hash-based) | 128 |

## Provider Discovery

List all registered providers at runtime:

```go
names := embedding.List()
// Returns sorted list: ["cohere", "google", "inmemory", "jina", ...]
```

## Middleware

All embedders support middleware for cross-cutting concerns such as logging, caching, and tracing:

```go
emb := embedding.ApplyMiddleware(baseEmb,
    loggingMiddleware,
    cachingMiddleware,
)
```

## Hooks

Hooks allow observing embedding operations without wrapping the interface:

```go
emb = embedding.ApplyMiddleware(baseEmb,
    embedding.WithHooks(embedding.Hooks{
        BeforeEmbed: func(ctx context.Context, texts []string) error {
            log.Printf("Embedding %d texts", len(texts))
            return nil
        },
        AfterEmbed: func(ctx context.Context, embeddings [][]float32, err error) {
            if err != nil {
                log.Printf("Embedding failed: %v", err)
            }
        },
    }),
)
```
