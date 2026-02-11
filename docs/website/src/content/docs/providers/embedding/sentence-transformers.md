---
title: Sentence Transformers Embeddings
description: Generate text embeddings using HuggingFace Inference API with Sentence Transformers models.
---

The Sentence Transformers embedding provider implements the `embedding.Embedder` interface using the HuggingFace Inference API's feature-extraction pipeline. It supports any Sentence Transformers model hosted on HuggingFace, including the popular all-MiniLM, all-mpnet, and BGE model families.

Choose Sentence Transformers when you need access to the widest selection of community embedding models or when you want to use BGE models that rank competitively on MTEB benchmarks. This provider can also target a self-hosted HuggingFace Text Embeddings Inference (TEI) server for full control over hosting.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/rag/embedding/providers/sentence_transformers
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
    _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/sentence_transformers"
)

func main() {
    emb, err := embedding.New("sentence_transformers", config.ProviderConfig{
        APIKey: os.Getenv("HUGGINGFACE_API_KEY"),
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
| `APIKey` | `string` | (required) | HuggingFace API token |
| `Model` | `string` | `sentence-transformers/all-MiniLM-L6-v2` | HuggingFace model ID |
| `BaseURL` | `string` | `https://api-inference.huggingface.co` | Inference API base URL |
| `Timeout` | `time.Duration` | 0 (no timeout) | Request timeout |
| `Options["dimensions"]` | `float64` | Model-dependent | Override vector dimensions |

## Supported Models

| Model | Default Dimensions |
|---|---|
| `sentence-transformers/all-MiniLM-L6-v2` | 384 |
| `sentence-transformers/all-MiniLM-L12-v2` | 384 |
| `sentence-transformers/all-mpnet-base-v2` | 768 |
| `sentence-transformers/paraphrase-MiniLM-L6-v2` | 384 |
| `BAAI/bge-small-en-v1.5` | 384 |
| `BAAI/bge-base-en-v1.5` | 768 |
| `BAAI/bge-large-en-v1.5` | 1024 |

Any model on HuggingFace that supports the feature-extraction pipeline can be used.

## Direct Construction

```go
import (
    stemb "github.com/lookatitude/beluga-ai/rag/embedding/providers/sentence_transformers"
)

emb, err := stemb.New(config.ProviderConfig{
    APIKey: os.Getenv("HUGGINGFACE_API_KEY"),
    Model:  "BAAI/bge-large-en-v1.5",
})
if err != nil {
    log.Fatal(err)
}
```

## Using BGE Models

The BGE (BAAI General Embedding) family provides competitive quality at different size tradeoffs:

```go
// Small model -- fast, 384 dimensions
emb, err := embedding.New("sentence_transformers", config.ProviderConfig{
    APIKey: os.Getenv("HUGGINGFACE_API_KEY"),
    Model:  "BAAI/bge-small-en-v1.5",
})

// Large model -- higher quality, 1024 dimensions
emb, err := embedding.New("sentence_transformers", config.ProviderConfig{
    APIKey: os.Getenv("HUGGINGFACE_API_KEY"),
    Model:  "BAAI/bge-large-en-v1.5",
})
```

## Self-Hosted Inference

Use the `BaseURL` parameter to point to a self-hosted HuggingFace Text Embeddings Inference (TEI) server:

```go
emb, err := embedding.New("sentence_transformers", config.ProviderConfig{
    APIKey:  "unused",
    BaseURL: "http://localhost:8080",
    Model:   "sentence-transformers/all-MiniLM-L6-v2",
})
```
