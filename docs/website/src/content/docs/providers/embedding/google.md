---
title: "Google Embedding Provider"
description: "Generate text embeddings with Google Gemini models in Beluga AI. High-quality embeddings with task type support and batch processing in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Google embeddings, Gemini embeddings, embedding provider, Google AI, batch processing, Go, Beluga AI"
---

The Google embedding provider implements the `embedding.Embedder` interface using the Google AI Gemini embedding API. It uses the batch `batchEmbedContents` endpoint for efficient multi-text embedding.

Choose Google embeddings when you are already using the Gemini LLM provider and want to unify your API key and billing, or when you need multilingual embeddings through Google's `text-multilingual-embedding-002` model. The batch endpoint provides efficient processing for large document sets.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/rag/embedding/providers/google
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
    _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/google"
)

func main() {
    emb, err := embedding.New("google", config.ProviderConfig{
        APIKey: os.Getenv("GOOGLE_API_KEY"),
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
| `APIKey` | `string` | (required) | Google AI API key |
| `Model` | `string` | `text-embedding-004` | Embedding model name |
| `BaseURL` | `string` | `https://generativelanguage.googleapis.com/v1beta` | API base URL |
| `Timeout` | `time.Duration` | 0 (no timeout) | Request timeout |
| `Options["dimensions"]` | `float64` | Model-dependent | Override vector dimensions |

## Supported Models

| Model | Default Dimensions |
|---|---|
| `text-embedding-004` | 768 |
| `embedding-001` | 768 |
| `text-multilingual-embedding-002` | 768 |

## Direct Construction

```go
import (
    googleemb "github.com/lookatitude/beluga-ai/rag/embedding/providers/google"
)

emb, err := googleemb.New(config.ProviderConfig{
    APIKey: os.Getenv("GOOGLE_API_KEY"),
    Model:  "text-embedding-004",
})
if err != nil {
    log.Fatal(err)
}
```

## Batch Embedding

The Google provider uses the `batchEmbedContents` API for efficient batch processing:

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

## Vertex AI

To use Vertex AI instead of the public Gemini API, configure a custom base URL:

```go
emb, err := embedding.New("google", config.ProviderConfig{
    APIKey:  os.Getenv("GOOGLE_API_KEY"),
    BaseURL: "https://us-central1-aiplatform.googleapis.com/v1",
    Model:   "text-embedding-004",
})
```
