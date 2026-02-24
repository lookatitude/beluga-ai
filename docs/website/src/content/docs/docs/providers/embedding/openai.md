---
title: "OpenAI Embedding Provider"
description: "Generate text embeddings with OpenAI text-embedding-3 models in Beluga AI. Configurable dimensions, batch processing, and high retrieval quality in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "OpenAI embeddings, text-embedding-3, embedding provider, vector embeddings, RAG, Go, Beluga AI"
---

The OpenAI embedding provider implements the `embedding.Embedder` interface using the official `openai-go` SDK. It supports all OpenAI embedding models including the latest `text-embedding-3` family with configurable dimensions.

Choose OpenAI embeddings for the best general-purpose quality-to-cost ratio. The `text-embedding-3-small` model offers strong retrieval performance at low cost, while `text-embedding-3-large` provides higher quality with configurable dimension reduction for storage optimization.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/rag/embedding/providers/openai
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
    _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/openai"
)

func main() {
    emb, err := embedding.New("openai", config.ProviderConfig{
        APIKey: os.Getenv("OPENAI_API_KEY"),
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
| `APIKey` | `string` | (required) | OpenAI API key |
| `Model` | `string` | `text-embedding-3-small` | Embedding model name |
| `BaseURL` | `string` | `https://api.openai.com/v1` | API base URL |
| `Timeout` | `time.Duration` | 0 (no timeout) | Request timeout |
| `Options["dimensions"]` | `float64` | Model-dependent | Override vector dimensions |

## Supported Models

| Model | Default Dimensions |
|---|---|
| `text-embedding-3-small` | 1536 |
| `text-embedding-3-large` | 3072 |
| `text-embedding-ada-002` | 1536 |

## Direct Construction

For more control, construct the embedder directly:

```go
import (
    oaiemb "github.com/lookatitude/beluga-ai/rag/embedding/providers/openai"
)

emb, err := oaiemb.New(config.ProviderConfig{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "text-embedding-3-large",
    Options: map[string]any{
        "dimensions": float64(1024),
    },
})
if err != nil {
    log.Fatal(err)
}
```

## Batch Embedding

The OpenAI provider handles batch embedding natively in a single API call:

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

## Custom Base URL

Use a custom endpoint for Azure OpenAI, local proxies, or compatible APIs:

```go
emb, err := embedding.New("openai", config.ProviderConfig{
    APIKey:  os.Getenv("AZURE_OPENAI_KEY"),
    BaseURL: "https://your-resource.openai.azure.com/openai/deployments/your-deployment",
    Model:   "text-embedding-3-small",
})
```
