# Retrievers Basic Example

This example demonstrates how to use the Retrievers package for document retrieval in RAG pipelines.

## Prerequisites

- Go 1.21+

## Running the Example

```bash
go run main.go
```

## What This Example Shows

1. Creating an embedder for generating embeddings
2. Creating a vector store for document storage
3. Adding documents to the vector store
4. Creating a retriever with configuration
5. Retrieving relevant documents for a query

## Configuration Options

- `WithDefaultK`: Default number of documents to retrieve
- `WithScoreThreshold`: Minimum similarity score threshold
- `WithMaxRetries`: Maximum retry attempts
- `WithTimeout`: Request timeout duration
- `WithTracing`: Enable OpenTelemetry tracing
- `WithMetrics`: Enable metrics collection

## Using Real Vector Stores

To use a real vector store:

```go
store, err := vectorstores.NewVectorStore(ctx, "qdrant",
	vectorstores.WithEmbedder(embedder),
	vectorstores.WithProviderConfig("url", "http://localhost:6333"),
	vectorstores.WithProviderConfig("collection_name", "documents"),
)
```

## Use Cases

- RAG (Retrieval-Augmented Generation) pipelines
- Semantic search
- Document similarity matching
- Knowledge base queries

## See Also

- [Retrievers Package Documentation](../../../pkg/retrievers/README.md)
- [RAG Examples](../../rag/simple/main.go)
