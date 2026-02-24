---
title: "Voyage AI Embedding Provider"
description: "Generate text embeddings with Voyage AI specialized models in Beluga AI. Code and domain-specific embeddings with high retrieval accuracy in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Voyage AI, embeddings, embedding provider, code embeddings, domain-specific, retrieval, Go, Beluga AI"
---

The Voyage AI embedding provider implements the `embedding.Embedder` interface using the Voyage Embed API. Voyage models are optimized for retrieval tasks and support input type differentiation for asymmetric search.

Choose Voyage embeddings when retrieval quality is critical, especially for code search (`voyage-code-2`) or when you need asymmetric document/query embeddings. Voyage models consistently rank highly on retrieval benchmarks and offer specialized models for code, legal, and financial domains.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/rag/embedding/providers/voyage
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/rag/embedding"
    _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/voyage"
)

func main() {
    emb, err := embedding.New("voyage", config.ProviderConfig{
        APIKey: os.Getenv("VOYAGE_API_KEY"),
    })
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()
    vec, err := emb.EmbedSingle(ctx, "Beluga AI is a Go framework for agentic systems")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Vector length: %d\n", len(vec))
    fmt.Printf("Dimensions: %d\n", emb.Dimensions())
}
```

## Configuration

| Parameter | Type | Default | Description |
|---|---|---|---|
| `APIKey` | `string` | (required) | Voyage AI API key |
| `Model` | `string` | `voyage-2` | Embedding model name |
| `BaseURL` | `string` | `https://api.voyageai.com/v1` | API base URL |
| `Timeout` | `time.Duration` | 0 (no timeout) | Request timeout |
| `Options["input_type"]` | `string` | `document` | Input type for embedding |
| `Options["dimensions"]` | `float64` | Model-dependent | Override vector dimensions |

## Supported Models

| Model | Default Dimensions |
|---|---|
| `voyage-2` | 1024 |
| `voyage-large-2` | 1536 |
| `voyage-code-2` | 1536 |
| `voyage-lite-02-instruct` | 1024 |
| `voyage-3` | 1024 |
| `voyage-3-lite` | 512 |

## Input Types

Voyage supports input type differentiation for asymmetric retrieval. Set the `input_type` option accordingly:

- `document` (default) -- Use when embedding documents for storage
- `query` -- Use when embedding queries for retrieval

```go
// Embed documents for storage
docEmb, err := embedding.New("voyage", config.ProviderConfig{
    APIKey: os.Getenv("VOYAGE_API_KEY"),
    Options: map[string]any{
        "input_type": "document",
    },
})

// Embed queries for retrieval
queryEmb, err := embedding.New("voyage", config.ProviderConfig{
    APIKey: os.Getenv("VOYAGE_API_KEY"),
    Options: map[string]any{
        "input_type": "query",
    },
})
```

## Direct Construction

```go
import (
    voyageemb "github.com/lookatitude/beluga-ai/rag/embedding/providers/voyage"
)

emb, err := voyageemb.New(config.ProviderConfig{
    APIKey: os.Getenv("VOYAGE_API_KEY"),
    Model:  "voyage-code-2",
    Options: map[string]any{
        "input_type": "document",
    },
})
if err != nil {
    log.Fatal(err)
}
```

## Code Embeddings

Voyage's `voyage-code-2` model is specifically tuned for code retrieval tasks:

```go
emb, err := embedding.New("voyage", config.ProviderConfig{
    APIKey: os.Getenv("VOYAGE_API_KEY"),
    Model:  "voyage-code-2",
})
if err != nil {
    log.Fatal(err)
}

vectors, err := emb.Embed(ctx, []string{
    "func main() { fmt.Println(\"hello\") }",
    "def main(): print('hello')",
})
```
