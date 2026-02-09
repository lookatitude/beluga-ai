---
title: Jina Embeddings
description: Generate text embeddings using Jina AI's embedding models.
---

The Jina AI embedding provider implements the `embedding.Embedder` interface using the Jina Embeddings API. It supports Jina's multilingual embedding models with native batch processing and token usage reporting.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/rag/embedding/providers/jina
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
    _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/jina"
)

func main() {
    emb, err := embedding.New("jina", config.ProviderConfig{
        APIKey: os.Getenv("JINA_API_KEY"),
    })
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()
    vectors, err := emb.Embed(ctx, []string{
        "Beluga AI provides a unified embedding interface",
        "Jina AI specializes in search and embeddings",
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Embedded %d texts, dimensions: %d\n", len(vectors), emb.Dimensions())
}
```

## Configuration

| Parameter | Type | Default | Description |
|---|---|---|---|
| `APIKey` | `string` | (required) | Jina AI API key |
| `Model` | `string` | `jina-embeddings-v2-base-en` | Embedding model name |
| `BaseURL` | `string` | `https://api.jina.ai/v1` | API base URL |
| `Timeout` | `time.Duration` | 0 (no timeout) | Request timeout |
| `Options["dimensions"]` | `float64` | Model-dependent | Override vector dimensions |

## Supported Models

| Model | Default Dimensions |
|---|---|
| `jina-embeddings-v2-base-en` | 768 |
| `jina-embeddings-v2-small-en` | 512 |
| `jina-embeddings-v2-base-de` | 768 |
| `jina-embeddings-v2-base-zh` | 768 |
| `jina-embeddings-v3` | 1024 |

## Direct Construction

```go
import (
    jinaemb "github.com/lookatitude/beluga-ai/rag/embedding/providers/jina"
)

emb, err := jinaemb.New(config.ProviderConfig{
    APIKey: os.Getenv("JINA_API_KEY"),
    Model:  "jina-embeddings-v3",
})
if err != nil {
    log.Fatal(err)
}
```

## Multilingual Models

Jina offers language-specific models that can be selected via the model parameter:

```go
// German-language embeddings
emb, err := embedding.New("jina", config.ProviderConfig{
    APIKey: os.Getenv("JINA_API_KEY"),
    Model:  "jina-embeddings-v2-base-de",
})

// Chinese-language embeddings
emb, err := embedding.New("jina", config.ProviderConfig{
    APIKey: os.Getenv("JINA_API_KEY"),
    Model:  "jina-embeddings-v2-base-zh",
})
```

## Batch Embedding

The Jina provider sends all texts in a single API request and correctly maps results back by index:

```go
texts := []string{
    "First document",
    "Second document",
    "Third document",
}

vectors, err := emb.Embed(ctx, texts)
if err != nil {
    log.Fatal(err)
}
```
