# RAG Pipeline Examples

This directory contains examples demonstrating Retrieval-Augmented Generation (RAG) pipelines in the Beluga AI Framework.

## Examples

### [simple](simple/)

Complete RAG pipeline from document ingestion to generation.

**What you'll learn:**
- Document preparation
- Embedding generation
- Vector store operations
- Query and retrieval
- Context-augmented generation

**Run:**
```bash
cd simple
go run main.go
```

### [with_memory](with_memory/)

RAG pipeline with conversation memory for multi-turn interactions.

**What you'll learn:**
- Combining RAG with memory
- Multi-turn RAG conversations
- Context building from multiple sources

**Run:**
```bash
cd with_memory
go run main.go
```

### [advanced](advanced/)

Advanced RAG patterns including filtering and multiple retrieval strategies.

**What you'll learn:**
- Advanced retrieval strategies
- Metadata filtering
- Score thresholds
- Multiple retrieval methods

**Run:**
```bash
cd advanced
go run main.go
```

## Prerequisites

- Go 1.21 or later
- Beluga AI Framework
- (Optional) OpenAI API key for embeddings and LLM

## Learning Path

1. Start with `simple` to understand RAG fundamentals
2. Add `with_memory` for conversation context
3. Explore `advanced` for optimization techniques

## Related Documentation

- [RAG Concepts](../../docs/concepts/rag.md)
- [RAG Recipes](../../docs/cookbook/rag-recipes.md)
- [Vector Stores](../../docs/providers/vectorstores/)
