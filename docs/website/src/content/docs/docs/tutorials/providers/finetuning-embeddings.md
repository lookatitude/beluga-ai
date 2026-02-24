---
title: Fine-Tuning Embedding Strategies
description: "Optimize embedding pipelines for retrieval in Go with Beluga AI — model selection, efficient batch processing, and chunking strategies for maximum RAG accuracy."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go, tutorial, embeddings, fine-tuning, RAG, batch processing, chunking, retrieval"
---

Standard embeddings work well for general use, but production RAG systems require optimization across three dimensions: model selection (accuracy vs. cost), batch processing (throughput), and text chunking (embedding quality). Each dimension directly impacts retrieval quality — the wrong model produces poor vectors, inefficient batching causes ingestion bottlenecks, and poor chunking creates embeddings that dilute or fragment the meaning of the source text. This tutorial covers strategies for each dimension.

## What You Will Build

An optimized embedding pipeline with model selection guidelines, efficient batch processing, and chunking strategies that maximize retrieval accuracy.

## Prerequisites

- Understanding of [Multimodal Embeddings](/docs/tutorials/providers/multimodal-embeddings)
- A configured embedding provider

## Step 1: Model Selection

Choose your embedding model based on the trade-off between accuracy, cost, and latency. Both providers are created through the same `embedding.New()` registry, which means your application code does not change when switching between models — only the provider name and configuration differ.

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/rag/embedding"
    _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/openai"
    _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/ollama"
)

func main() {
    ctx := context.Background()

    // High accuracy (paid API)
    openaiEmb, err := embedding.New("openai", config.ProviderConfig{
        "api_key": os.Getenv("OPENAI_API_KEY"),
        "model":   "text-embedding-3-large",
    })
    if err != nil {
        fmt.Printf("OpenAI error: %v\n", err)
        return
    }

    // Local, free, private
    ollamaEmb, err := embedding.New("ollama", config.ProviderConfig{
        "base_url": "http://localhost:11434",
        "model":    "nomic-embed-text",
    })
    if err != nil {
        fmt.Printf("Ollama error: %v\n", err)
        return
    }

    text := "Kubernetes pod scheduling and resource allocation"
    vec1, _ := openaiEmb.EmbedQuery(ctx, text)
    vec2, _ := ollamaEmb.EmbedQuery(ctx, text)

    fmt.Printf("OpenAI dimensions: %d\n", len(vec1))
    fmt.Printf("Ollama dimensions: %d\n", len(vec2))
}
```

| Scenario | Recommended Model | Why |
|:---|:---|:---|
| Production RAG | text-embedding-3-large | Best accuracy |
| Cost-sensitive | text-embedding-3-small | 1/6 the cost |
| Air-gapped / privacy | nomic-embed-text (Ollama) | No API calls |
| Multi-language | text-embedding-004 (Google) | Strong cross-language |
| Domain-specific | Fine-tuned local model | Captures jargon |

## Step 2: Efficient Batch Processing

Process documents in optimal batch sizes to balance throughput and API rate limits. The batch processing function amortizes HTTP round-trip overhead across multiple documents per request. The optimal batch size depends on the provider — API-based providers can handle large batches efficiently, while local models are limited by GPU memory.

```go
func batchEmbed(ctx context.Context, embedder embedding.Embedder, texts []string, batchSize int) ([][]float32, error) {
    var allVectors [][]float32

    for i := 0; i < len(texts); i += batchSize {
        end := i + batchSize
        if end > len(texts) {
            end = len(texts)
        }

        vectors, err := embedder.EmbedDocuments(ctx, texts[i:end])
        if err != nil {
            return nil, fmt.Errorf("batch %d-%d: %w", i, end, err)
        }
        allVectors = append(allVectors, vectors...)
    }

    return allVectors, nil
}
```

Recommended batch sizes:
- **OpenAI**: 100-500 texts per batch (API limit: 2048)
- **Google**: 50-100 texts per batch
- **Ollama**: 10-50 texts (local compute bound)

## Step 3: Text Chunking for Quality

The quality of embeddings depends directly on chunk size and strategy. Embedding models produce a single vector for the entire input text, which means the vector represents the average meaning of all content in the chunk. If a chunk mixes multiple topics, the resulting vector sits between them in the vector space rather than close to any one topic, reducing retrieval precision.

- **Too small** (< 100 tokens): Loss of context, fragmented meaning
- **Too large** (> 1000 tokens): Diluted meaning, averaging too many topics
- **Optimal** (200-500 tokens): Enough context for coherent meaning

### Fixed-size chunking

Split text into chunks of a fixed token count with overlap. The overlap parameter ensures that sentences or ideas that span chunk boundaries are captured in at least one chunk's embedding. An overlap of 10-20% of the chunk size provides good boundary coverage without excessive duplication.

```go
func chunkText(text string, chunkSize, overlap int) []string {
    words := strings.Fields(text)
    var chunks []string

    for i := 0; i < len(words); i += chunkSize - overlap {
        end := i + chunkSize
        if end > len(words) {
            end = len(words)
        }
        chunk := strings.Join(words[i:end], " ")
        chunks = append(chunks, chunk)
        if end == len(words) {
            break
        }
    }

    return chunks
}
```

### Paragraph-aware chunking

Preserve natural text boundaries. Paragraph-aware chunking produces higher quality embeddings than fixed-size chunking because paragraphs typically contain a single coherent idea. The algorithm accumulates paragraphs until the token limit is reached, then starts a new chunk. This ensures that no chunk splits a paragraph mid-sentence.

```go
func chunkByParagraphs(text string, maxTokens int) []string {
    paragraphs := strings.Split(text, "\n\n")
    var chunks []string
    var current strings.Builder

    for _, para := range paragraphs {
        paraLen := len(strings.Fields(para))
        currentLen := len(strings.Fields(current.String()))

        if currentLen+paraLen > maxTokens && current.Len() > 0 {
            chunks = append(chunks, strings.TrimSpace(current.String()))
            current.Reset()
        }
        if current.Len() > 0 {
            current.WriteString("\n\n")
        }
        current.WriteString(para)
    }

    if current.Len() > 0 {
        chunks = append(chunks, strings.TrimSpace(current.String()))
    }

    return chunks
}
```

## Step 4: Hybrid Approach

Combine embedding search with keyword search (BM25) to cover both semantic similarity and exact matches. Beluga AI defaults to hybrid search (vector + BM25 + RRF fusion) because embedding models alone struggle with domain-specific jargon, error codes, and proper nouns. A search for "SIGTERM signal handling" retrieves better results when exact keyword matching supplements the semantic search.

See [Hybrid Search](/docs/tutorials/rag/hybrid-search) for the full implementation.

## Verification

1. Embed 100 documents using batch processing — measure throughput improvement over sequential.
2. Compare retrieval accuracy between different chunk sizes for a fixed query set.
3. Test with domain-specific queries to identify where keyword search outperforms embeddings.

## Next Steps

- [Multimodal Embeddings](/docs/tutorials/providers/multimodal-embeddings) — Embed images and text
- [pgvector Sharding](/docs/tutorials/providers/pgvector-sharding) — Scale to millions of vectors
