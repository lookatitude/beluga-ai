# Advanced Retrieval Example

This example demonstrates multi-strategy retrieval with similarity search, keyword search, hybrid search using Reciprocal Rank Fusion (RRF), and intelligent query analysis for optimal strategy selection.

## Prerequisites

- **Go 1.24+**: Required for the Beluga AI framework
- **Vector store configured**: Qdrant, Pinecone, or PostgreSQL with pgvector
- **Embeddings provider**: OpenAI, Voyage, or custom embeddings

## What You'll Learn

- Implementing similarity search with embeddings
- Implementing keyword search (BM25-style)
- Combining both with hybrid search using RRF
- Query analysis for automatic strategy selection
- OTEL instrumentation for retrieval metrics

## Files

| File | Description |
|------|-------------|
| `advanced_retrieval.go` | Multi-strategy retrieval implementation |
| `advanced_retrieval_test.go` | Comprehensive test suite |
| `advanced_retrieval_guide.md` | Detailed guide with explanations |

## Usage

```go
package main

import (
    "context"
    "log"
)

func main() {
    ctx := context.Background()
    
    // Create retriever with hybrid strategy as default
    retriever, err := NewAdvancedRetriever(
        vectorStore,
        embedder,
        keywordStore,
        WithDefaultStrategy(StrategyHybrid),
        WithTopK(10),
        WithMinScore(0.7),
        WithHybridAlpha(0.5), // Balanced similarity/keyword
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // Hybrid search combines similarity and keyword
    results, err := retriever.HybridSearch(ctx, "error handling in Go")
    
    // Multi-strategy automatically selects best approach
    results, err = retriever.MultiStrategySearch(ctx, "What is RAG?")
}
```

## Testing

```bash
go test -v ./...

# Run benchmarks
go test -bench=. -benchmem ./...
```

## Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| `WithDefaultStrategy` | `hybrid` | Default search strategy |
| `WithTopK` | `10` | Number of results to return |
| `WithMinScore` | `0.7` | Minimum score threshold |
| `WithHybridAlpha` | `0.5` | Weight for similarity (0=keyword, 1=similarity) |
| `WithQueryAnalysis` | `true` | Enable automatic strategy selection |
| `WithFallback` | `true` | Fallback to hybrid on poor results |

## Retrieval Strategies

| Strategy | When to Use |
|----------|-------------|
| `StrategySimilarity` | Semantic queries, Q&A, concepts |
| `StrategyKeyword` | Technical terms, codes, exact matches |
| `StrategyHybrid` | General purpose, mixed queries |
| `StrategyMulti` | Complex domains, diverse query types |

## Related Examples

- **[RAG Evaluation](../../rag/evaluation/)**: Measuring retrieval quality
- **[Multimodal RAG](../../rag/multimodal/)**: RAG with images

## Related Documentation

- **[RAG Strategies Use Case](../../../docs/use-cases/rag-strategies.md)**: Strategy comparison
- **[Multimodal RAG Guide](../../../docs/guides/rag-multimodal.md)**: RAG with images
- **[Extensibility Guide](../../../docs/guides/extensibility.md)**: Vector store integration
