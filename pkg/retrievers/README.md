# Retrievers Package

The `retrievers` package provides components for retrieving relevant documents from various data sources in Retrieval Augmented Generation (RAG) pipelines. It offers a clean, extensible architecture for document retrieval with support for observability, configuration management, and multiple retrieval strategies.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Core Components](#core-components)
- [Usage](#usage)
- [Configuration](#configuration)
- [Extensibility](#extensibility)
- [Observability](#observability)
- [Error Handling](#error-handling)
- [Testing](#testing)
- [Performance](#performance)
- [Migration Guide](#migration-guide)

## Overview

The retrievers package is designed to support the retrieval phase of RAG pipelines. It provides:

- **Multiple Retrieval Strategies**: Vector similarity search, keyword-based retrieval, hybrid approaches
- **Configurable Behavior**: Functional options for fine-tuning retrieval parameters
- **Observability**: Built-in metrics, tracing, and structured logging
- **Extensible Design**: Easy to add new retriever implementations
- **Production Ready**: Error handling, timeouts, retries, and health checks

## Architecture

### Package Structure

```
pkg/retrievers/
├── iface/                    # Core interfaces
│   ├── interfaces.go        # Retriever interfaces and types
│   └── options.go           # Configuration options
├── internal/                # Private implementation details
│   └── mock/               # Mock implementations for testing
│       ├── mock.go         # Mock retriever implementation
│       └── mock_test.go    # Mock-specific tests
├── providers/               # Directory for future concrete implementations
├── config.go                # Configuration structs and validation
├── errors.go                # Custom error types
├── metrics.go               # OpenTelemetry metrics integration
├── retrievers.go            # Main factory functions and types
├── retrievers_test.go       # Comprehensive test suite
├── vectorstore.go           # VectorStoreRetriever implementation
└── README.md               # This documentation
```

### Design Principles

The package follows these core design principles:

1. **Interface Segregation**: Small, focused interfaces for specific retrieval operations
2. **Dependency Inversion**: High-level modules depend on abstractions, not concretions
3. **Functional Options**: Flexible configuration using functional options pattern
4. **Composition over Inheritance**: Prefer embedding and composition
5. **Observability First**: Built-in support for metrics, tracing, and logging

## Core Components

### Interfaces

#### Retriever Interface

The core `Retriever` interface defines the contract for all retriever implementations:

```go
type Retriever interface {
    core.Retriever
}
```

This extends the `core.Retriever` interface which provides:

```go
type Retriever interface {
    GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error)
    Invoke(ctx context.Context, input any, options ...core.Option) (any, error)
    Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error)
    Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error)
}
```

#### VectorStoreRetriever

The `VectorStoreRetriever` is the primary implementation that retrieves documents from vector stores:

```go
type VectorStoreRetriever struct {
    // Configuration fields
    vectorStore    vectorstores.VectorStore
    defaultK       int
    scoreThreshold float32
    maxRetries     int
    timeout        time.Duration

    // Observability
    enableTracing  bool
    enableMetrics  bool
    logger         *slog.Logger
    tracer         trace.Tracer
    metrics        *Metrics
}
```

## Usage

### Basic Usage

#### Creating a Vector Store Retriever

```go
package main

import (
    "context"
    "log"

    "github.com/lookatitude/beluga-ai/pkg/retrievers"
    "github.com/lookatitude/beluga-ai/pkg/vectorstores"
)

func main() {
    // Assume you have a configured vector store
    var vectorStore vectorstores.VectorStore

    // Create a retriever with default configuration
    retriever, err := retrievers.NewVectorStoreRetriever(vectorStore)
    if err != nil {
        log.Fatal(err)
    }

    // Retrieve documents
    ctx := context.Background()
    docs, err := retriever.GetRelevantDocuments(ctx, "What is machine learning?")
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Retrieved %d documents", len(docs))
}
```

#### Using Functional Options

```go
// Create a retriever with custom configuration
retriever, err := retrievers.NewVectorStoreRetriever(vectorStore,
    retrievers.WithDefaultK(10),
    retrievers.WithScoreThreshold(0.7),
    retrievers.WithTimeout(30*time.Second),
    retrievers.WithTracing(true),
    retrievers.WithMetrics(true),
)
```

#### Configuration-Based Creation

```go
// Create from configuration struct
config := retrievers.VectorStoreRetrieverConfig{
    K:              5,
    ScoreThreshold: 0.8,
    Timeout:        45 * time.Second,
}

retriever, err := retrievers.NewVectorStoreRetrieverFromConfig(vectorStore, config)
```

### Advanced Usage

#### Using with Runnable Interface

```go
// Use as a Runnable in chains/graphs
result, err := retriever.Invoke(ctx, "query string")
if err != nil {
    log.Fatal(err)
}

// Batch processing
queries := []any{"query1", "query2", "query3"}
results, err := retriever.Batch(ctx, queries)
```

#### Custom Options

```go
// Use call-specific options
docs, err := retriever.GetRelevantDocuments(ctx, "query",
    retrievers.WithK(15),                    // Override default K
    retrievers.WithScoreThreshold(0.9),      // Override threshold
    retrievers.WithMetadataFilter(map[string]any{
        "category": "technical",
    }),
)
```

## Configuration

### Configuration Options

The package supports multiple levels of configuration:

1. **Package-level defaults**: Sensible defaults for all retrievers
2. **Retriever-level options**: Set when creating the retriever instance
3. **Call-level options**: Override settings for specific retrieval calls

### Configuration Structs

#### Main Config

```go
type Config struct {
    DefaultK           int           `mapstructure:"default_k"`
    ScoreThreshold     float32       `mapstructure:"score_threshold"`
    MaxRetries         int           `mapstructure:"max_retries"`
    Timeout            time.Duration `mapstructure:"timeout"`
    EnableTracing      bool          `mapstructure:"enable_tracing"`
    EnableMetrics      bool          `mapstructure:"enable_metrics"`
    VectorStoreConfig  VectorStoreConfig `mapstructure:",squash"`
}
```

#### Functional Options

```go
// Available options
WithDefaultK(k int)
WithScoreThreshold(threshold float32)
WithMaxRetries(retries int)
WithTimeout(timeout time.Duration)
WithTracing(enabled bool)
WithMetrics(enabled bool)
WithLogger(logger *slog.Logger)
WithTracer(tracer trace.Tracer)
WithMeter(meter metric.Meter)
```

### Validation

All configuration is validated at creation time:

```go
config := retrievers.DefaultConfig()
err := retrievers.ValidateRetrieverConfig(config)
if err != nil {
    log.Fatal("Invalid configuration:", err)
}
```

## Extensibility

### Adding New Retriever Types

To add a new retriever implementation:

1. **Implement the interfaces**:

```go
type CustomRetriever struct {
    // Your fields
}

func (cr *CustomRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
    // Your implementation
}

func (cr *CustomRetriever) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
    // Your implementation
}

// Implement Batch and Stream methods
```

2. **Add factory functions**:

```go
func NewCustomRetriever(config CustomConfig, options ...Option) (*CustomRetriever, error) {
    // Validate and create
}
```

3. **Add to providers**:

Place your implementation in `pkg/retrievers/providers/custom.go`

### Extending Configuration

Add new functional options:

```go
func WithCustomParameter(value string) Option {
    return func(opts *RetrieverOptions) {
        opts.CustomParam = value
    }
}
```

### Provider Pattern

The package uses a provider pattern for extensibility:

- **Core interfaces** in `iface/`
- **Concrete implementations** in `providers/`
- **Factory functions** in main package
- **Configuration structs** with validation

## Observability

### Metrics Initialization

The package uses a standardized metrics initialization pattern with `InitMetrics()` and `GetMetrics()`:

```go
import (
    "go.opentelemetry.io/otel/metric"
    "github.com/lookatitude/beluga-ai/pkg/retrievers"
)

// Initialize metrics once at application startup
meter := otel.Meter("beluga-retrievers")
retrievers.InitMetrics(meter)

// Get the global metrics instance
metrics := retrievers.GetMetrics()
if metrics != nil {
    // Metrics are automatically recorded for retrieval operations
}
```

**Note**: `InitMetrics()` uses `sync.Once` to ensure thread-safe initialization. It should be called once at application startup.

### Metrics

The package provides comprehensive metrics using OpenTelemetry:

```go
// Retrieval metrics
retrieval_requests_total
retrieval_duration_seconds
retrieval_errors_total
documents_retrieved_total
retrieval_score_avg

// Vector store metrics
vector_store_requests_total
vector_store_duration_seconds
vector_store_errors_total
documents_stored_total
documents_deleted_total

// Performance metrics
batch_size_avg
```

### Tracing

All retrieval operations are traced:

### Health Checks

The package implements the `core.HealthChecker` interface for monitoring component health:

```go
// Check health of a retriever
err := retriever.CheckHealth(ctx)
if err != nil {
    log.Printf("Retriever health check failed: %v", err)
}
```

Health checks validate:
- Configuration parameters (K, ScoreThreshold, etc.)
- Required dependencies (vector store availability)
- Component state and connectivity

### Logging

Structured logging with configurable levels:

```go
// Info level: Successful operations
logger.Info("retrieval completed",
    "documents_returned", count,
    "duration", duration,
)

// Error level: Failed operations
logger.Error("retrieval failed",
    "error", err,
    "query", query,
    "duration", duration,
)
```

## Error Handling

### Custom Error Types

The package defines specific error types:

```go
// General retriever errors
type RetrieverError struct {
    Op      string
    Err     error
    Code    string
    Message string
}

// Configuration validation errors
type ValidationError struct {
    Field string
    Value interface{}
    Msg   string
}

// Timeout errors
type TimeoutError struct {
    Op      string
    Timeout time.Duration
    Err     error
}
```

### Error Codes

Standardized error codes for programmatic handling:

```go
ErrCodeInvalidConfig    = "invalid_config"
ErrCodeInvalidInput     = "invalid_input"
ErrCodeRetrievalFailed  = "retrieval_failed"
ErrCodeEmbeddingFailed  = "embedding_failed"
ErrCodeVectorStoreError = "vector_store_error"
ErrCodeTimeout          = "timeout"
ErrCodeRateLimit        = "rate_limit"
ErrCodeNetworkError     = "network_error"
```

### Error Handling Best Practices

```go
docs, err := retriever.GetRelevantDocuments(ctx, query)
if err != nil {
    var retrieverErr *retrievers.RetrieverError
    if errors.As(err, &retrieverErr) {
        switch retrieverErr.Code {
        case retrievers.ErrCodeTimeout:
            // Handle timeout
        case retrievers.ErrCodeInvalidInput:
            // Handle invalid input
        default:
            // Handle other errors
        }
    }
    return err
}
```

## Testing

### Test Structure

Comprehensive test suite covering:

- **Unit tests** for all public functions
- **Integration tests** with mock vector stores
- **Configuration validation tests**
- **Error handling tests**
- **Performance benchmarks**

### Running Tests

```bash
# Run all tests
go test ./pkg/retrievers/...

# Run with coverage
go test -cover ./pkg/retrievers/...

# Run benchmarks
go test -bench=. ./pkg/retrievers/...
```

### Mock Implementation

The package includes a `MockRetriever` for testing:

```go
mockRetriever := providers.NewMockRetriever("test", testDocuments,
    providers.WithDefaultK(5),
    providers.WithScoreThreshold(0.7),
)
```

## Performance

### Benchmarks

The package includes comprehensive benchmarks for performance-critical operations:

```bash
# Run all benchmarks
go test -bench=. ./pkg/retrievers/...

# Key benchmark results:
# - Config validation: ~2ns/op
# - Option application: ~32ns/op
# - Default config creation: ~0.12ns/op
# - Mock retrieval: ~30ms/op (with 100 documents)
```

Available benchmarks:
- Configuration validation performance
- Functional options application
- Default configuration creation
- Mock retriever document retrieval

### Optimization Guidelines

1. **Batch Processing**: Use `Batch()` for multiple queries
2. **Connection Pooling**: Reuse retriever instances
3. **Timeout Configuration**: Set appropriate timeouts
4. **Score Threshold**: Use thresholds to reduce result sets
5. **Caching**: Implement caching at higher levels if needed

### Performance Metrics

Monitor these key metrics:

- Retrieval latency (p50, p95, p99)
- Throughput (requests/second)
- Error rates by type
- Memory usage patterns

## Migration Guide

### From Legacy Retrievers

If migrating from an older version:

1. **Update imports**:
   ```go
   // Old
   import "github.com/lookatitude/beluga-ai/pkg/retrievers/base"

   // New
   import "github.com/lookatitude/beluga-ai/pkg/retrievers"
   ```

2. **Update factory calls**:
   ```go
   // Old
   retriever := retrievers.NewVectorStoreRetriever(store)

   // New - with options
   retriever, err := retrievers.NewVectorStoreRetriever(store,
       retrievers.WithDefaultK(5),
       retrievers.WithTracing(true),
   )
   ```

3. **Update error handling**:
   ```go
   // Old
   if err != nil {
       return err
   }

   // New - with typed errors
   if err != nil {
       var retrieverErr *retrievers.RetrieverError
       if errors.As(err, &retrieverErr) {
           // Handle specific error types
       }
       return err
   }
   ```

### Breaking Changes

- Factory functions now return errors
- Configuration validation is stricter
- Error types have changed
- Some internal APIs may have changed

## Examples

### Complete RAG Pipeline

```go
package main

import (
    "context"
    "log"

    "github.com/lookatitude/beluga-ai/pkg/retrievers"
    "github.com/lookatitude/beluga-ai/pkg/vectorstores"
)

func main() {
    ctx := context.Background()

    // Set up vector store (assume configured)
    var vectorStore vectorstores.VectorStore

    // Create retriever
    retriever, err := retrievers.NewVectorStoreRetriever(vectorStore,
        retrievers.WithDefaultK(3),
        retrievers.WithScoreThreshold(0.7),
        retrievers.WithTimeout(30*time.Second),
    )
    if err != nil {
        log.Fatal("Failed to create retriever:", err)
    }

    // Retrieve relevant documents
    query := "What are the benefits of microservices architecture?"
    docs, err := retriever.GetRelevantDocuments(ctx, query)
    if err != nil {
        log.Fatal("Failed to retrieve documents:", err)
    }

    log.Printf("Retrieved %d documents for query: %s", len(docs), query)

    // Process documents (e.g., pass to LLM)
    for i, doc := range docs {
        log.Printf("Document %d: %s...", i+1, doc.PageContent[:100])
    }
}
```

### Custom Retriever Implementation

```go
package main

import (
    "context"

    "github.com/lookatitude/beluga-ai/pkg/core"
    "github.com/lookatitude/beluga-ai/pkg/retrievers"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

type KeywordRetriever struct {
    documents []schema.Document
    index     map[string][]int // keyword -> document indices
}

func (kr *KeywordRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
    // Simple keyword-based retrieval
    var results []schema.Document

    // Tokenize query and find matching documents
    // Implementation details...

    return results, nil
}

func (kr *KeywordRetriever) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
    query, ok := input.(string)
    if !ok {
        return nil, retrievers.NewRetrieverErrorWithMessage("Invoke", nil, retrievers.ErrCodeInvalidInput,
            "KeywordRetriever expects string input")
    }
    return kr.GetRelevantDocuments(ctx, query)
}

// Implement Batch and Stream methods...

// Factory function
func NewKeywordRetriever(documents []schema.Document) *KeywordRetriever {
    return &KeywordRetriever{
        documents: documents,
        index:     buildIndex(documents),
    }
}
```

---

## Contributing

When contributing to the retrievers package:

1. **Follow the design patterns** outlined in the Beluga AI framework documentation
2. **Add comprehensive tests** for new functionality
3. **Update documentation** including this README
4. **Follow Go best practices** for code style and error handling
5. **Add metrics and tracing** for new operations

## License

This package is part of the Beluga AI framework and follows the same license terms.
