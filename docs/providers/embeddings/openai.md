# OpenAI Embeddings Guide

Complete guide to using OpenAI embedding models.

## Overview

OpenAI provides high-quality embedding models for semantic search and RAG.

## Setup

```go
config := &embeddings.Config{
    OpenAI: &embeddings.OpenAIConfig{
        APIKey: os.Getenv("OPENAI_API_KEY"),
        Model:  "text-embedding-ada-002",
    },
}

factory, _ := embeddings.NewEmbedderFactory(config)
embedder, _ := factory.NewEmbedder("openai")
```

## Available Models

- `text-embedding-ada-002`: 1536 dims, cost-effective
- `text-embedding-3-small`: 1536 dims, improved
- `text-embedding-3-large`: 3072 dims, best quality

## Usage

```go
// Embed documents
texts := []string{"Document 1", "Document 2"}
embeddings, _ := embedder.EmbedDocuments(ctx, texts)

// Embed query
query := "search query"
queryEmbedding, _ := embedder.EmbedQuery(ctx, query)
```

## Best Practices

- Use batch processing for multiple texts
- Choose model based on quality needs
- Monitor token usage and costs

---

**Next:** [Provider Selection Guide](./selection.md)

