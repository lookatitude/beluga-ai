---
title: "Mistral AI Embedding Provider"
description: "Generate text embeddings with Mistral AI models in Beluga AI. High-quality European AI embeddings with batch support and efficient inference in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Mistral embeddings, embedding provider, European AI, batch embeddings, text embeddings, Go, Beluga AI"
---

The Mistral embedding provider implements the `embedding.Embedder` interface using the Mistral AI embeddings API. It provides high-quality embeddings through the `mistral-embed` model.

Choose Mistral embeddings when you are already using Mistral for LLM inference and want to consolidate your provider stack under a single API key and billing account. The `mistral-embed` model delivers solid retrieval quality with 1024-dimensional vectors.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/rag/embedding/providers/mistral
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
    _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/mistral"
)

func main() {
    emb, err := embedding.New("mistral", config.ProviderConfig{
        APIKey: os.Getenv("MISTRAL_API_KEY"),
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
| `APIKey` | `string` | (required) | Mistral AI API key |
| `Model` | `string` | `mistral-embed` | Embedding model name |
| `BaseURL` | `string` | `https://api.mistral.ai/v1` | API base URL |
| `Timeout` | `time.Duration` | 0 (no timeout) | Request timeout |
| `Options["dimensions"]` | `float64` | Model-dependent | Override vector dimensions |

## Supported Models

| Model | Default Dimensions |
|---|---|
| `mistral-embed` | 1024 |

## Direct Construction

```go
import (
    mistralemb "github.com/lookatitude/beluga-ai/rag/embedding/providers/mistral"
)

emb, err := mistralemb.New(config.ProviderConfig{
    APIKey: os.Getenv("MISTRAL_API_KEY"),
    Model:  "mistral-embed",
})
if err != nil {
    log.Fatal(err)
}
```

## Batch Embedding

The Mistral provider sends all texts in a single API request and correctly orders results by index:

```go
texts := []string{
    "First document about Go programming",
    "Second document about vector databases",
    "Third document about machine learning",
}

vectors, err := emb.Embed(ctx, texts)
if err != nil {
    log.Fatal(err)
}

for i, vec := range vectors {
    fmt.Printf("Text %d: %d dimensions\n", i, len(vec))
}
```

## Error Handling

The provider requires an API key and returns an error if none is provided:

```go
// This will return an error: "mistral embedding: api_key is required"
emb, err := embedding.New("mistral", config.ProviderConfig{})
if err != nil {
    log.Fatal(err) // Handle missing API key
}
```
