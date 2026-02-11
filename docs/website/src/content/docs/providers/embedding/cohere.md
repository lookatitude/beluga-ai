---
title: Cohere Embeddings
description: Generate text embeddings using Cohere's Embed API with input type control.
---

The Cohere embedding provider implements the `embedding.Embedder` interface using Cohere's v2 Embed API. It supports input type differentiation between documents and queries, which is critical for asymmetric search scenarios.

Choose Cohere embeddings when building search systems that benefit from asymmetric retrieval (different embeddings for documents vs. queries). Cohere also offers multilingual models that embed 100+ languages into a shared vector space, making them ideal for cross-language search applications.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/rag/embedding/providers/cohere
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
    _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/cohere"
)

func main() {
    emb, err := embedding.New("cohere", config.ProviderConfig{
        APIKey: os.Getenv("COHERE_API_KEY"),
    })
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()
    vectors, err := emb.Embed(ctx, []string{
        "Beluga AI provides a unified embedding interface",
        "Vector databases store high-dimensional embeddings",
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
| `APIKey` | `string` | (required) | Cohere API key |
| `Model` | `string` | `embed-english-v3.0` | Embedding model name |
| `BaseURL` | `string` | `https://api.cohere.com/v2` | API base URL |
| `Timeout` | `time.Duration` | 0 (no timeout) | Request timeout |
| `Options["input_type"]` | `string` | `search_document` | Input type for embedding |
| `Options["dimensions"]` | `float64` | Model-dependent | Override vector dimensions |

## Supported Models

| Model | Default Dimensions |
|---|---|
| `embed-english-v3.0` | 1024 |
| `embed-multilingual-v3.0` | 1024 |
| `embed-english-light-v3.0` | 384 |
| `embed-multilingual-light-v3.0` | 384 |
| `embed-english-v2.0` | 4096 |

## Input Types

Cohere embedding models differentiate between document and query embeddings. Set the `input_type` option to control this behavior:

- `search_document` (default) -- Use when embedding documents for storage
- `search_query` -- Use when embedding queries for retrieval
- `classification` -- Use for classification tasks
- `clustering` -- Use for clustering tasks

```go
// Embed documents for storage
docEmb, err := embedding.New("cohere", config.ProviderConfig{
    APIKey: os.Getenv("COHERE_API_KEY"),
    Options: map[string]any{
        "input_type": "search_document",
    },
})

// Embed queries for retrieval
queryEmb, err := embedding.New("cohere", config.ProviderConfig{
    APIKey: os.Getenv("COHERE_API_KEY"),
    Options: map[string]any{
        "input_type": "search_query",
    },
})
```

## Direct Construction

```go
import (
    cohereemb "github.com/lookatitude/beluga-ai/rag/embedding/providers/cohere"
)

emb, err := cohereemb.New(config.ProviderConfig{
    APIKey: os.Getenv("COHERE_API_KEY"),
    Model:  "embed-multilingual-v3.0",
    Options: map[string]any{
        "input_type": "search_document",
    },
})
if err != nil {
    log.Fatal(err)
}
```

## Multilingual Support

Cohere's multilingual models support 100+ languages in a shared embedding space:

```go
emb, err := embedding.New("cohere", config.ProviderConfig{
    APIKey: os.Getenv("COHERE_API_KEY"),
    Model:  "embed-multilingual-v3.0",
})
if err != nil {
    log.Fatal(err)
}

vectors, err := emb.Embed(ctx, []string{
    "Hello world",           // English
    "Bonjour le monde",      // French
    "Hola mundo",            // Spanish
})
```
