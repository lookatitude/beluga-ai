---
title: In-Memory Embeddings
description: Deterministic hash-based embeddings for testing and development.
---

The in-memory embedding provider implements the `embedding.Embedder` interface using a deterministic FNV-1a hash function. It generates reproducible, normalized vectors without any external API calls, making it suitable for unit tests and local development.

Use the in-memory embedder exclusively for testing and development. It produces deterministic output (same text always yields the same vector) but vectors are not semantically meaningful -- this provider is not suitable for production retrieval. It eliminates the need for API keys, network access, or mocking in test suites.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/rag/embedding/providers/inmemory
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/rag/embedding"
    _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/inmemory"
)

func main() {
    emb, err := embedding.New("inmemory", config.ProviderConfig{})
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()
    vec, err := emb.EmbedSingle(ctx, "hello world")
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
| `Options["dimensions"]` | `float64` | `128` | Vector dimensionality |

No API key or network access is required.

## Properties

- **Deterministic**: The same input text always produces the same vector
- **Normalized**: All vectors are unit-length (L2 norm = 1)
- **Hash-based**: Uses FNV-1a hashing to generate pseudo-random vector components
- **No external dependencies**: Runs entirely in-process with no network calls

## Custom Dimensions

```go
emb, err := embedding.New("inmemory", config.ProviderConfig{
    Options: map[string]any{
        "dimensions": float64(256),
    },
})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Dimensions: %d\n", emb.Dimensions()) // 256
```

## Testing Usage

The in-memory embedder is designed for use in tests where you need a real `embedding.Embedder` implementation without mocking:

```go
func TestRAGPipeline(t *testing.T) {
    emb, err := embedding.New("inmemory", config.ProviderConfig{})
    if err != nil {
        t.Fatal(err)
    }

    ctx := context.Background()

    // Same text produces same embedding
    vec1, err := emb.EmbedSingle(ctx, "test text")
    if err != nil {
        t.Fatal(err)
    }

    vec2, err := emb.EmbedSingle(ctx, "test text")
    if err != nil {
        t.Fatal(err)
    }

    for i := range vec1 {
        if vec1[i] != vec2[i] {
            t.Fatalf("expected deterministic output at index %d", i)
        }
    }
}
```

## Limitations

- Vectors are not semantically meaningful -- similar texts do not produce similar embeddings
- Intended for testing and development only, not for production retrieval
- Hash collisions are theoretically possible but practically irrelevant for testing
