# RAG Evaluation Example

This example demonstrates how to measure and benchmark your RAG pipeline's retrieval quality using standard Information Retrieval metrics: Precision@K, Recall@K, Mean Reciprocal Rank (MRR), and Normalized Discounted Cumulative Gain (NDCG).

## Prerequisites

- **Go 1.24+**: Required for the Beluga AI framework
- **Working RAG pipeline**: A retriever to evaluate
- **Evaluation dataset**: Ground truth query-document pairs

## What You'll Learn

- Calculating Precision@K and Recall@K
- Implementing MRR for ranking quality
- Using NDCG for graded relevance
- Creating evaluation datasets
- Running automated benchmarks
- OTEL instrumentation for evaluation metrics

## Files

| File | Description |
|------|-------------|
| `rag_evaluation.go` | Complete evaluation implementation |
| `rag_evaluation_test.go` | Test suite with metric verification |
| `rag_evaluation_guide.md` | Detailed guide with examples |

## Usage

```go
package main

import (
    "context"
    "log"
)

func main() {
    ctx := context.Background()
    
    // Create evaluator for your retriever
    evaluator, err := NewRAGEvaluator(retriever)
    if err != nil {
        log.Fatal(err)
    }
    
    // Define evaluation queries with ground truth
    queries := []EvaluationQuery{
        {
            ID:             "q1",
            Query:          "How do I handle errors?",
            RelevantDocIDs: []string{"doc1", "doc3"},
        },
        {
            ID:             "q2",
            Query:          "API design patterns",
            RelevantDocIDs: []string{"doc2"},
            GradedRelevance: map[string]int{
                "doc2": 3, // Highly relevant
                "doc5": 1, // Marginally relevant
            },
        },
    }
    
    // Run evaluation
    agg, results, err := evaluator.EvaluateDataset(ctx, queries, 10)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Mean Precision@5: %.2f", agg.MeanPrecisionAt5)
    log.Printf("Mean Recall@5: %.2f", agg.MeanRecallAt5)
    log.Printf("Mean MRR: %.2f", agg.MeanMRR)
    log.Printf("Mean NDCG: %.2f", agg.MeanNDCG)
}
```

## Testing

```bash
go test -v ./...

# Run benchmarks
go test -bench=. -benchmem ./...
```

## Metrics Explained

| Metric | Description | Range |
|--------|-------------|-------|
| **Precision@K** | Fraction of top K that are relevant | 0.0 - 1.0 |
| **Recall@K** | Fraction of all relevant found in top K | 0.0 - 1.0 |
| **MRR** | 1/position of first relevant result | 0.0 - 1.0 |
| **NDCG** | Ranking quality with graded relevance | 0.0 - 1.0 |

## Evaluation Dataset Format

```json
{
  "queries": [
    {
      "id": "q1",
      "query": "How to handle errors?",
      "relevant_doc_ids": ["doc1", "doc3"],
      "graded_relevance": {
        "doc1": 3,
        "doc3": 2
      }
    }
  ]
}
```

## Related Examples

- **[Advanced Retrieval](../../vectorstores/advanced_retrieval/)**: Multi-strategy retrieval
- **[Multimodal RAG](../multimodal/)**: RAG with images

## Related Documentation

- **[RAG Strategies Use Case](../../../docs/use-cases/rag-strategies.md)**: When to use which strategy
- **[Multimodal RAG Guide](../../../docs/guides/rag-multimodal.md)**: RAG with images
- **[Observability Tracing](../../../docs/guides/observability-tracing.md)**: Monitoring your RAG
