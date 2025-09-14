# VectorStores Package

The `vectorstores` package provides a comprehensive vector storage and retrieval system for retrieval-augmented generation (RAG) applications in the Beluga AI Framework.

## Overview

This package implements the Beluga AI Framework design patterns including:
- **Interface Segregation Principle (ISP)** - Focused, single-responsibility interfaces
- **Dependency Inversion Principle (DIP)** - Abstractions over concrete implementations
- **OpenTelemetry Observability** - Comprehensive metrics, tracing, and logging
- **Factory Pattern** - Extensible provider registration and creation
- **Functional Options** - Type-safe configuration management

## Key Features

- **Multiple Vector Store Providers**: In-memory, PostgreSQL (pgvector), and extensible architecture
- **Efficient Similarity Search**: Cosine similarity with configurable algorithms
- **Comprehensive Observability**: Metrics, tracing, and structured logging
- **Type-Safe Configuration**: Functional options with validation
- **Batch Operations**: Efficient bulk document processing
- **Error Handling**: Structured error types with proper error wrapping

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"

    "github.com/lookatitude/beluga-ai/pkg/vectorstores"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

func main() {
    ctx := context.Background()

    // Create an in-memory vector store
    store, err := vectorstores.NewInMemoryStore(ctx,
        vectorstores.WithEmbedder(embedder),
        vectorstores.WithSearchK(10),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Add documents
    docs := []schema.Document{
        schema.NewDocument("Machine learning is awesome", map[string]string{"topic": "ml"}),
        schema.NewDocument("Deep learning uses neural networks", map[string]string{"topic": "dl"}),
    }
    ids, err := store.AddDocuments(ctx, docs)
    if err != nil {
        log.Fatal(err)
    }

    // Search by query
    results, scores, err := store.SimilaritySearchByQuery(ctx, "ML basics", 5, embedder)
    if err != nil {
        log.Fatal(err)
    }

    for i, doc := range results {
        log.Printf("Result %d: %s (score: %.3f)", i+1, doc.GetContent(), scores[i])
    }
}
```

### Advanced Configuration

```go
// Configure with custom settings
store, err := vectorstores.NewVectorStore(ctx, "pgvector",
    vectorstores.WithEmbedder(embedder),
    vectorstores.WithSearchK(20),
    vectorstores.WithScoreThreshold(0.8),
    vectorstores.WithProviderConfig("connection_string", "postgres://user:pass@localhost/db"),
    vectorstores.WithProviderConfig("table_name", "my_documents"),
    vectorstores.WithMetadataFilter("category", "tech"),
)
```

### Batch Operations

```go
// Efficient batch processing
ids, err := vectorstores.BatchAddDocuments(ctx, store, allDocs, 100, embedder)

// Multiple queries at once
results, scores, err := vectorstores.BatchSearch(ctx, store, queries, 5, embedder)
```

### Observability Setup

```go
// Set up global observability
import "go.opentelemetry.io/otel/metric"
import "log/slog"

meter := metric.NewMeterProvider().Meter("my-app")
metrics, _ := vectorstores.NewMetricsCollector(meter)
vectorstores.SetGlobalMetrics(metrics)

tracer := vectorstores.NewTracerProvider("my-app")
vectorstores.SetGlobalTracer(tracer)

logger := vectorstores.NewLogger(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
vectorstores.SetGlobalLogger(logger)
```

## Architecture

### Package Structure

```
pkg/vectorstores/
├── iface/              # Interfaces and core types
│   ├── vectorstore.go  # Main interfaces (VectorStore, Retriever, etc.)
│   ├── errors.go       # Custom error types
│   └── options.go      # Functional options
├── internal/           # Private implementation details
├── providers/          # Provider implementations
│   ├── inmemory/       # In-memory provider
│   ├── pgvector/       # PostgreSQL provider
│   └── pinecone/       # Pinecone provider
├── config.go           # Configuration management
├── metrics.go          # Observability metrics
├── logging.go          # Structured logging
├── vectorstores.go     # Factory functions and utilities
└── README.md           # This documentation
```

### Core Interfaces

#### VectorStore Interface

The main interface for vector storage and retrieval:

```go
type VectorStore interface {
    AddDocuments(ctx context.Context, documents []schema.Document, opts ...Option) ([]string, error)
    DeleteDocuments(ctx context.Context, ids []string, opts ...Option) error
    SimilaritySearch(ctx context.Context, queryVector []float32, k int, opts ...Option) ([]schema.Document, []float32, error)
    SimilaritySearchByQuery(ctx context.Context, query string, k int, embedder Embedder, opts ...Option) ([]schema.Document, []float32, error)
    AsRetriever(opts ...Option) Retriever
    GetName() string
}
```

#### Retriever Interface

For integration with retrieval chains:

```go
type Retriever interface {
    GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error)
}
```

#### Embedder Interface

For generating vector embeddings:

```go
type Embedder interface {
    EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error)
    EmbedQuery(ctx context.Context, text string) ([]float32, error)
}
```

## Providers

### In-Memory Provider

**Best for**: Development, testing, small datasets

```go
store, err := vectorstores.NewInMemoryStore(ctx,
    vectorstores.WithEmbedder(embedder),
)
```

**Features**:
- Fast in-memory operations
- Cosine similarity search
- Thread-safe concurrent access
- Automatic ID generation

**Limitations**:
- Data lost on restart
- Memory scales with document count
- Not suitable for large datasets

### PostgreSQL Provider

**Best for**: Production, large datasets, ACID compliance

```go
store, err := vectorstores.NewPgVectorStore(ctx,
    vectorstores.WithEmbedder(embedder),
    vectorstores.WithProviderConfig("connection_string", "postgres://user:pass@localhost/db"),
    vectorstores.WithProviderConfig("table_name", "documents"),
    vectorstores.WithProviderConfig("embedding_dimension", 768),
)
```

**Requirements**:
- PostgreSQL with pgvector extension
- Proper database permissions

### Pinecone Provider

**Status**: Not yet implemented

**Best for**: Cloud-native, managed vector search (when implemented)

```go
// TODO: Implement Pinecone provider
store, err := vectorstores.NewPineconeStore(ctx,
    vectorstores.WithEmbedder(embedder),
    vectorstores.WithProviderConfig("api_key", "your-api-key"),
    vectorstores.WithProviderConfig("environment", "us-west1-gcp"),
    vectorstores.WithProviderConfig("project_id", "your-project"),
    vectorstores.WithProviderConfig("index_name", "my-index"),
)
// Note: This will return an error until the provider is implemented
```

## Configuration

### Functional Options

The package uses functional options for type-safe configuration:

```go
// Search configuration
vectorstores.WithSearchK(10)                    // Number of results to return
vectorstores.WithScoreThreshold(0.8)           // Minimum similarity score

// Embedder configuration
vectorstores.WithEmbedder(embedder)            // Embedder to use

// Metadata filtering
vectorstores.WithMetadataFilter("category", "tech")
vectorstores.WithMetadataFilters(map[string]interface{}{
    "category": "tech",
    "status": "published",
})

// Provider-specific configuration
vectorstores.WithProviderConfig("table_name", "my_documents")
vectorstores.WithProviderConfigs(map[string]interface{}{
    "connection_string": "postgres://...",
    "max_connections": 10,
})
```

### Configuration Validation

All configurations include validation:

```go
config := vectorstores.NewDefaultConfig()
config.SearchK = 20
config.ScoreThreshold = 0.8

store, err := vectorstores.NewVectorStore(ctx, "pgvector", config)
if err != nil {
    // Handle configuration validation error
}
```

## Observability

### Metrics

Automatic collection of operational metrics:

- **Document Operations**: Documents added/deleted/stored counts
- **Search Operations**: Request count, duration, result counts
- **Embedding Operations**: Request count, duration
- **Error Tracking**: Errors by type and provider
- **Resource Usage**: Memory and disk usage

### Tracing

Distributed tracing with OpenTelemetry:

- Document addition spans
- Search operation spans
- Embedding generation spans
- Error context propagation

### Logging

Structured logging with context:

```go
// Automatic logging includes:
// - Operation type and duration
// - Document/query counts
// - Error details with stack traces
// - Trace and span IDs for correlation
```

## Error Handling

### Custom Error Types

Structured errors with codes for programmatic handling:

```go
import "github.com/lookatitude/beluga-ai/pkg/vectorstores"

// Check for specific error types
if vectorstores.IsVectorStoreError(err, vectorstores.ErrCodeConnectionFailed) {
    // Handle connection error
}

// Error codes
const (
    ErrCodeUnknownProvider      = "unknown_provider"
    ErrCodeInvalidConfig        = "invalid_config"
    ErrCodeConnectionFailed     = "connection_failed"
    ErrCodeEmbeddingFailed      = "embedding_failed"
    ErrCodeStorageFailed        = "storage_failed"
    ErrCodeRetrievalFailed      = "retrieval_failed"
)
```

### Error Wrapping

Proper error chaining with context:

```go
err := vectorstores.WrapError(cause, vectorstores.ErrCodeStorageFailed, "failed to store document %s", docID)
```

## Extensibility

### Adding New Providers

Implement the `VectorStore` interface and register with the factory:

```go
type CustomStore struct {
    // implementation
}

func (s *CustomStore) AddDocuments(ctx context.Context, docs []schema.Document, opts ...vectorstores.Option) ([]string, error) {
    // implementation
}

// Register with factory
vectorstores.RegisterProvider("custom", func(ctx context.Context, config vectorstores.Config) (vectorstores.VectorStore, error) {
    return NewCustomStore(config)
})

// Use the provider
store, err := vectorstores.NewVectorStore(ctx, "custom", config)
```

### Custom Embedders

Implement the `Embedder` interface:

```go
type CustomEmbedder struct{}

func (e *CustomEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
    // implementation
}

func (e *CustomEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
    // implementation
}
```

## Testing

### Table-Driven Tests

```go
func TestVectorStore(t *testing.T) {
    tests := []struct {
        name     string
        docs     []schema.Document
        query    string
        wantErr  bool
    }{
        {
            name: "successful search",
            docs: []schema.Document{
                schema.NewDocument("test content", nil),
            },
            query: "test",
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            store := inmemory.NewInMemoryVectorStore(embedder)
            _, err := store.AddDocuments(ctx, tt.docs)
            if (err != nil) != tt.wantErr {
                t.Errorf("AddDocuments() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Mocking

Use provided mock implementations for testing:

```go
mockEmbedder := &iface.MockEmbedder{
    MockEmbedQuery: func(ctx context.Context, text string) ([]float32, error) {
        return []float32{0.1, 0.2, 0.3}, nil
    },
}
```

## Performance Considerations

### In-Memory Provider
- **Best for**: < 100K documents
- **Memory usage**: ~4KB per document (embedding + metadata)
- **Search speed**: O(n) where n = document count

### PostgreSQL Provider
- **Best for**: Production, large datasets
- **Indexing**: HNSW for sub-linear search
- **Concurrent access**: Full ACID compliance

### Batch Operations
- Use `BatchAddDocuments` for bulk inserts
- Configure batch size based on memory constraints
- Consider rate limiting for external APIs

## Migration Guide

### From Legacy Implementation

1. **Update imports**:
   ```go
   // Old
   "github.com/lookatitude/beluga-ai/pkg/vectorstores/providers/inmemory"

   // New
   "github.com/lookatitude/beluga-ai/pkg/vectorstores"
   ```

2. **Update constructor calls**:
   ```go
   // Old
   store := inmemory.NewInMemoryVectorStore(embedder)

   // New
   store, err := vectorstores.NewInMemoryStore(ctx, vectorstores.WithEmbedder(embedder))
   ```

3. **Update method signatures**:
   ```go
   // Old
   ids, err := store.AddDocuments(ctx, docs)

   // New
   ids, err := store.AddDocuments(ctx, docs, vectorstores.WithEmbedder(embedder))
   ```

## Best Practices

1. **Provider Selection**:
   - Use in-memory for development/testing
   - Use PostgreSQL for production with large datasets
   - Use cloud providers for managed solutions

2. **Configuration**:
   - Validate configurations at startup
   - Use environment variables for sensitive data
   - Document configuration requirements

3. **Error Handling**:
   - Check error codes for programmatic handling
   - Log errors with appropriate levels
   - Implement retry logic for transient failures

4. **Observability**:
   - Set up metrics and tracing in production
   - Use structured logging for better debugging
   - Monitor resource usage and performance

5. **Performance**:
   - Use batch operations for bulk processing
   - Configure appropriate search limits
   - Consider embedding caching for repeated queries

## Troubleshooting

### Common Issues

1. **Embedder Required Error**:
   ```
   Solution: Provide embedder via WithEmbedder() option or constructor
   ```

2. **Connection Failed**:
   ```
   Solution: Verify database credentials and network connectivity
   ```

3. **Invalid Configuration**:
   ```
   Solution: Use config validation and check error messages
   ```

### Debug Logging

Enable debug logging to troubleshoot issues:

```go
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))
vectorstores.SetGlobalLogger(vectorstores.NewLogger(logger))
```

## Contributing

When contributing to the vectorstores package:

1. Follow the established design patterns
2. Add comprehensive tests
3. Update documentation
4. Implement proper error handling
5. Add observability support

## License

This package is part of the Beluga AI Framework and follows the same licensing terms.
