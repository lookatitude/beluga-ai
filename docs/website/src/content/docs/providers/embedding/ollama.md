---
title: Ollama Embeddings
description: Generate text embeddings locally using Ollama's embedding models.
---

The Ollama embedding provider implements the `embedding.Embedder` interface using the Ollama REST API. It enables fully local embedding generation with no external API dependencies, making it suitable for development, testing, and privacy-sensitive deployments.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/rag/embedding/providers/ollama
```

Ensure Ollama is running locally:

```bash
# Install Ollama (see https://ollama.ai)
ollama pull nomic-embed-text
ollama serve
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
    _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/ollama"
)

func main() {
    emb, err := embedding.New("ollama", config.ProviderConfig{})
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
| `Model` | `string` | `nomic-embed-text` | Ollama model name |
| `BaseURL` | `string` | `http://localhost:11434` | Ollama server URL |
| `Timeout` | `time.Duration` | 0 (no timeout) | Request timeout |
| `Options["dimensions"]` | `float64` | Model-dependent | Override vector dimensions |

## Supported Models

| Model | Default Dimensions |
|---|---|
| `nomic-embed-text` | 768 |
| `mxbai-embed-large` | 1024 |
| `all-minilm` | 384 |
| `snowflake-arctic-embed` | 1024 |

Any model available in Ollama that supports the embed API can be used. Pull the model first with `ollama pull <model>`.

## Direct Construction

```go
import (
    ollamaemb "github.com/lookatitude/beluga-ai/rag/embedding/providers/ollama"
)

emb, err := ollamaemb.New(config.ProviderConfig{
    BaseURL: "http://localhost:11434",
    Model:   "mxbai-embed-large",
})
if err != nil {
    log.Fatal(err)
}
```

## Remote Ollama Server

Connect to a remote Ollama instance by specifying the base URL:

```go
emb, err := embedding.New("ollama", config.ProviderConfig{
    BaseURL: "http://gpu-server.internal:11434",
    Model:   "nomic-embed-text",
})
```

## Batch Behavior

The Ollama provider processes batch requests sequentially, embedding one text at a time via the `/api/embed` endpoint. For high-throughput scenarios, consider using a cloud-based provider that supports native batch embedding.
