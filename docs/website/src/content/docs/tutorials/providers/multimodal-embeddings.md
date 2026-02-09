---
title: Multimodal Embeddings
description: Generate embeddings for text and images in a shared vector space using Beluga AI's embedding providers.
---

Traditional search is limited to text matching. Multimodal embeddings project text, images, and other modalities into the same high-dimensional vector space, enabling cross-modal retrieval — search a product catalog of images using text queries, or find documents that relate to an uploaded photo.

## What You Will Build

A multimodal embedding pipeline that embeds both text and images into a shared vector space, then computes similarity scores across modalities.

## Prerequisites

- A Google Cloud API key with the Generative Language API enabled (or equivalent multimodal embedding provider)
- Understanding of the [RAG pipeline](/guides/rag)

## The Embedder Interface

Beluga AI's embedding interface in the `rag/embedding` package:

```go
type Embedder interface {
    EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error)
    EmbedQuery(ctx context.Context, text string) ([]float32, error)
}
```

For multimodal use cases, embedding providers accept `schema.Document` objects with metadata that specifies the content type.

## Step 1: Initialize the Embedding Provider

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
    ctx := context.Background()

    embedder, err := embedding.New("google", config.ProviderConfig{
        "api_key": os.Getenv("GOOGLE_API_KEY"),
        "model":   "text-embedding-004",
    })
    if err != nil {
        log.Fatalf("create embedder: %v", err)
    }

    // Embed text documents
    texts := []string{
        "A golden retriever playing in a park",
        "Financial quarterly report Q3 2025",
        "Kubernetes deployment configuration",
    }

    vectors, err := embedder.EmbedDocuments(ctx, texts)
    if err != nil {
        log.Fatalf("embed documents: %v", err)
    }

    for i, v := range vectors {
        fmt.Printf("Text %d: %d dimensions\n", i, len(v))
    }
}
```

## Step 2: Embed Queries

Use `EmbedQuery` for search queries, which may use a different embedding strategy optimized for retrieval:

```go
queryVector, err := embedder.EmbedQuery(ctx, "dog playing outside")
if err != nil {
    log.Fatalf("embed query: %v", err)
}

fmt.Printf("Query vector: %d dimensions\n", len(queryVector))
```

## Step 3: Batch Processing for Performance

Embedding one document at a time is slow. Process documents in batches:

```go
func batchEmbed(ctx context.Context, embedder embedding.Embedder, texts []string, batchSize int) ([][]float32, error) {
    var allVectors [][]float32

    for i := 0; i < len(texts); i += batchSize {
        end := i + batchSize
        if end > len(texts) {
            end = len(texts)
        }

        batch := texts[i:end]
        vectors, err := embedder.EmbedDocuments(ctx, batch)
        if err != nil {
            return nil, fmt.Errorf("batch %d-%d: %w", i, end, err)
        }
        allVectors = append(allVectors, vectors...)
    }

    return allVectors, nil
}
```

## Step 4: Compute Cosine Similarity

Compare vectors to find semantic matches:

```go
import "math"

func cosineSimilarity(a, b []float32) float64 {
    if len(a) != len(b) {
        return 0
    }

    var dot, normA, normB float64
    for i := range a {
        dot += float64(a[i]) * float64(b[i])
        normA += float64(a[i]) * float64(a[i])
        normB += float64(b[i]) * float64(b[i])
    }

    if normA == 0 || normB == 0 {
        return 0
    }
    return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
```

Usage:

```go
// Compare query to each document
for i, docVec := range vectors {
    score := cosineSimilarity(queryVector, docVec)
    fmt.Printf("Similarity to doc %d: %.4f\n", i, score)
}
```

## Step 5: Choosing the Right Model

| Model | Dimensions | Best For |
|:---|:---|:---|
| OpenAI text-embedding-3-large | 3072 | High accuracy, general purpose |
| OpenAI text-embedding-3-small | 1536 | Cost-effective, general purpose |
| Google text-embedding-004 | 768 | Multimodal, cross-language |
| Ollama nomic-embed-text | 768 | Local, privacy-sensitive, free |

Always use the same model for both document indexing and query embedding. Mixing models produces incompatible vector spaces.

## Verification

1. Embed a list of 100 texts using batch processing — measure the time versus sequential embedding.
2. Compute similarity between semantically related texts — verify scores above 0.7.
3. Compute similarity between unrelated texts — verify lower scores.

## Next Steps

- [Fine-tuning Embedding Strategies](/tutorials/providers/finetuning-embeddings) — Optimize retrieval performance
- [In-memory Vector Store](/tutorials/providers/inmemory-vectorstore) — Store and search embeddings locally
