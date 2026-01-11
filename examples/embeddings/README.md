# Embeddings Examples

This directory contains examples demonstrating how to use the Beluga AI Embeddings package.

## Examples

### [basic](basic/)

Basic embedding operations including factory creation, single text embedding, and batch document embedding.

**What you'll learn:**
- Creating an embedder factory
- Creating embedder instances
- Embedding single queries
- Embedding multiple documents
- Health checks
- Provider management

**Run:**
```bash
cd basic
go run main.go
```

## Prerequisites

- Go 1.21 or later
- Beluga AI Framework
- (Optional) OpenAI API key for real embeddings

## Learning Path

1. Start with `basic` to understand embedding fundamentals
2. Try `examples/rag/simple` to see embeddings in RAG pipelines
3. Explore `examples/rag/advanced` for advanced patterns

## Related Documentation

- [Embeddings Concepts](../../docs/concepts/)
- [RAG Examples](../rag/)
- [Package Documentation](../../pkg/embeddings/README.md)
