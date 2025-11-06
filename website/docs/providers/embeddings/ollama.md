---
title: Ollama Embeddings
sidebar_position: 2
---

# Ollama Embeddings Guide

Complete guide to using local Ollama embedding models.

## Overview

Ollama provides local embedding models for privacy-sensitive applications.

## Setup

```bash
# Pull embedding model
ollama pull nomic-embed-text
```

## Configuration

```go
config := &embeddings.Config{
    Ollama: &embeddings.OllamaConfig{
        ServerURL: "http://localhost:11434",
        Model:     "nomic-embed-text",
    },
}

factory, _ := embeddings.NewEmbedderFactory(config)
embedder, _ := factory.NewEmbedder("ollama")
```

## Benefits

- Privacy: Data stays local
- Cost: Free
- Control: Full control

## Limitations

- Quality may be lower than OpenAI
- Requires local resources
- Setup complexity

## Best Practices

- Use for privacy-sensitive applications
- Monitor resource usage
- Test quality for your use case

---

**Next:** [Provider Selection Guide](./selection.md)

