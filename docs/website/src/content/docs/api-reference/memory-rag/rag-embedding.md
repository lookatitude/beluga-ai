---
title: "Embedding API — Embedder Interface"
description: "RAG embedding API reference for Beluga AI. Embedder interface for text-to-vector conversion, provider registry, middleware, and hooks."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "embedding API, Embedder, vector embeddings, RAG, provider registry, middleware, Beluga AI, Go, reference"
---

```go
import "github.com/lookatitude/beluga-ai/rag/embedding"
```

Package embedding provides the Embedder interface and registry for
converting text into vector embeddings. Embedders are the first stage of
the RAG pipeline, producing dense vector representations that enable
similarity search over documents.

## Interface

The core interface is `Embedder`:

```go
type Embedder interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    EmbedSingle(ctx context.Context, text string) ([]float32, error)
    Dimensions() int
}
```

Implementations must be safe for concurrent use.

## Registry

The package follows Beluga's registry pattern. Providers register via
init() and are instantiated with `New`:

```go
emb, err := embedding.New("openai", cfg)
if err != nil {
    log.Fatal(err)
}

vectors, err := emb.Embed(ctx, []string{"hello world"})
if err != nil {
    log.Fatal(err)
}
```

Use `List` to discover all registered provider names.

## Providers

Available providers (imported via blank import):
- "openai" — OpenAI text-embedding-3-small/large
- "cohere" — Cohere Embed v3
- "google" — Google AI Gemini embedding
- "jina" — Jina AI embeddings
- "mistral" — Mistral AI embeddings
- "ollama" — Ollama local embedding models
- "sentence_transformers" — HuggingFace Sentence Transformers
- "voyage" — Voyage AI embeddings
- "inmemory" — Deterministic hash-based embedder for testing

## Middleware and Hooks

Cross-cutting concerns like logging, caching, and tracing are layered
via `Middleware` and `Hooks`:

```go
emb = embedding.ApplyMiddleware(emb,
    embedding.WithHooks(embedding.Hooks{
        BeforeEmbed: func(ctx context.Context, texts []string) error {
            log.Printf("embedding %d texts", len(texts))
            return nil
        },
    }),
)
```

Use `ComposeHooks` to merge multiple hook sets into one. For BeforeEmbed,
the first error returned short-circuits execution.

## Custom Provider

To add a custom embedding provider:

```go
func init() {
    embedding.Register("custom", func(cfg config.ProviderConfig) (embedding.Embedder, error) {
        return &myEmbedder{apiKey: cfg.APIKey}, nil
    })
}
```
