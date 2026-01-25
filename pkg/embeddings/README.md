# Embeddings Package

The embeddings package provides interfaces and implementations for text embedding generation within the Beluga AI Framework. It follows the framework's design patterns with clean separation of interfaces, implementations, and configuration management.

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Supported Providers](#supported-providers)
4. [Configuration](#configuration)
5. [Usage](#usage)
6. [Extending the Package](#extending-the-package)
7. [Observability](#observability)
8. [Testing](#testing)
9. [Migration Guide](#migration-guide)

## Overview

The embeddings package enables the generation of vector representations (embeddings) for text data. These embeddings can be used for various AI applications including:

- Semantic search and retrieval
- Text similarity comparison
- Clustering and classification
- Recommendation systems
- Question answering systems

The package supports multiple embedding providers and includes comprehensive observability features including OpenTelemetry tracing and metrics.

## Architecture

The package follows the Beluga AI Framework design patterns:

```
pkg/embeddings/
├── iface/              # Interface definitions
├── providers/          # Provider implementations
│   ├── openai/        # OpenAI embedding provider
│   ├── ollama/        # Ollama local model provider
│   └── mock/          # Mock provider for testing
├── config.go           # Configuration structs and validation
├── metrics.go          # OpenTelemetry metrics integration
└── embeddings.go       # Main factory and interfaces
```

### Key Components

- **`iface.Embedder`**: Core interface defining embedding operations (following ISP with focused, embedding-specific methods)
- **`EmbedderFactory`**: Factory for creating embedder instances
- **`Config`**: Configuration management with validation
- **`Metrics`**: OpenTelemetry metrics collection
- **Provider Implementations**: Concrete implementations for different embedding services

## Supported Providers

### OpenAI
Uses OpenAI's embedding models for high-quality embeddings.

**Supported Models:**
- `text-embedding-ada-002` (1536 dimensions)
- `text-embedding-3-small` (1536 dimensions)
- `text-embedding-3-large` (3072 dimensions)

### Ollama (Experimental)
Uses local Ollama models for private, offline embeddings.

**⚠️ Security Notice:** Ollama contains high-severity CVEs (GO-2025-3824, GO-2025-3695) allowing cross-domain token exposure and DoS attacks. This provider is **only available when building with the experimental tag**:

```bash
go build -tags experimental ./...
```

**Features:**
- Local model execution
- No API costs
- Full data privacy
- Custom models support

**Security Recommendation:** Use only in isolated, air-gapped environments or with proper network security controls.

### Mock
Provides deterministic mock embeddings for testing and development.

**Features:**
- Configurable dimensions
- Deterministic output with seed control
- Zero vectors for empty inputs (configurable)
- No external dependencies

## Configuration

The package uses structured configuration with validation and defaults.

### Example Configuration (YAML)

```yaml
embeddings:
  openai:
    api_key: "sk-..."
    model: "text-embedding-ada-002"
    timeout: "30s"
    max_retries: 3
    enabled: true

  ollama:
    server_url: "http://localhost:11434"
    model: "nomic-embed-text"
    timeout: "30s"
    max_retries: 3
    enabled: true

  mock:
    dimension: 128
    seed: 42
    randomize_nil: false
    enabled: true
```

### Configuration Structs

```go
type Config struct {
    OpenAI *OpenAIConfig `mapstructure:"openai" yaml:"openai"`
    Ollama *OllamaConfig `mapstructure:"ollama" yaml:"ollama"`
    Mock   *MockConfig   `mapstructure:"mock" yaml:"mock"`
}
```

### Functional Options

The package supports functional options for runtime configuration:

```go
factory, err := embeddings.NewEmbedderFactory(config,
    embeddings.WithTimeout(60*time.Second),
    embeddings.WithMaxRetries(5),
    embeddings.WithModel("text-embedding-3-small"),
)
```

Available options:
- `WithTimeout(duration)`: Set request timeout
- `WithMaxRetries(count)`: Set maximum retry attempts
- `WithModel(name)`: Override the default model

## Usage

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/embeddings"
)

func main() {
    // Create configuration
    config := &embeddings.Config{
        OpenAI: &embeddings.OpenAIConfig{
            APIKey: "sk-...",
            Model:  "text-embedding-ada-002",
        },
    }

    // Create factory with functional options
    factory, err := embeddings.NewEmbedderFactory(config,
        embeddings.WithTimeout(30*time.Second),
        embeddings.WithMaxRetries(3),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Create embedder instance
    embedder, err := factory.NewEmbedder("openai")
    if err != nil {
        log.Fatal(err)
    }

    // Generate embeddings
    ctx := context.Background()
    texts := []string{"Hello world", "How are you?"}

    embeddings, err := embedder.EmbedDocuments(ctx, texts)
    if err != nil {
        log.Fatal(err)
    }

    // embeddings is [][]float32 with shape [len(texts)][dimension]
    log.Printf("Generated %d embeddings with dimension %d",
        len(embeddings), len(embeddings[0]))
}
```

### Single Query Embedding

```go
query := "What is machine learning?"
embedding, err := embedder.EmbedQuery(ctx, query)
if err != nil {
    log.Fatal(err)
}
// embedding is []float32 with length = dimension
```

### Health Checks

```go
// Check if provider is healthy
err := factory.CheckHealth(ctx, "openai")
if err != nil {
    log.Printf("Provider health check failed: %v", err)
}
```

### Available Providers

```go
providers := factory.GetAvailableProviders()
// Returns []string of enabled provider types
```

## Extending the Package

### Adding a New Provider

1. **Create Provider Directory**

```bash
mkdir -p pkg/embeddings/providers/yourprovider
```

2. **Implement the Embedder Interface**

```go
package yourprovider

import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/pkg/embeddings"
    "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
    "go.opentelemetry.io/otel/trace"
)

type YourProviderEmbedder struct {
    config  *embeddings.YourProviderConfig
    metrics *embeddings.Metrics
    tracer  trace.Tracer
}

func NewYourProviderEmbedder(config *embeddings.YourProviderConfig, metrics *embeddings.Metrics, tracer trace.Tracer) (*YourProviderEmbedder, error) {
    // Implementation
    return &YourProviderEmbedder{
        config:  config,
        metrics: metrics,
        tracer:  tracer,
    }, nil
}

func (e *YourProviderEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
    ctx, span := e.tracer.Start(ctx, "yourprovider.embed_documents")
    defer span.End()

    e.metrics.StartRequest(ctx, "yourprovider", e.config.Model)
    start := time.Now()
    defer func() {
        duration := time.Since(start)
        e.metrics.EndRequest(ctx, "yourprovider", e.config.Model)
    }()

    // Your implementation here
    // ...

    e.metrics.RecordRequest(ctx, "yourprovider", e.config.Model, time.Since(start), len(texts), dimension)
    return embeddings, nil
}

func (e *YourProviderEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
    // Implementation similar to EmbedDocuments but for single text
}

func (e *YourProviderEmbedder) GetDimension(ctx context.Context) (int, error) {
    // Return embedding dimension
}

func (e *YourProviderEmbedder) Check(ctx context.Context) error {
    // Health check implementation
    return nil
}

var _ iface.Embedder = (*YourProviderEmbedder)(nil)
var _ embeddings.HealthChecker = (*YourProviderEmbedder)(nil)
```

3. **Add Configuration**

Update `pkg/embeddings/config.go`:

```go
type Config struct {
    // ... existing fields
    YourProvider *YourProviderConfig `mapstructure:"yourprovider" yaml:"yourprovider"`
}

type YourProviderConfig struct {
    APIKey      string        `mapstructure:"api_key" yaml:"api_key" validate:"required"`
    Model       string        `mapstructure:"model" yaml:"model" validate:"required"`
    Timeout     time.Duration `mapstructure:"timeout" yaml:"timeout" default:"30s"`
    MaxRetries  int           `mapstructure:"max_retries" yaml:"max_retries" default:"3"`
    Enabled     bool          `mapstructure:"enabled" yaml:"enabled" default:"true"`
}
```

4. **Update Factory**

Update `pkg/embeddings/embeddings.go`:

```go
import (
    // ... existing imports
    "github.com/lookatitude/beluga-ai/pkg/embeddings/providers/yourprovider"
)

// Add to NewEmbedder method
case "yourprovider":
    return f.newYourProviderEmbedder()

// Add factory method
func (f *EmbedderFactory) newYourProviderEmbedder() (iface.Embedder, error) {
    if f.config.YourProvider == nil || !f.config.YourProvider.Enabled {
        return nil, fmt.Errorf("YourProvider provider is not configured or disabled")
    }

    if err := f.config.YourProvider.Validate(); err != nil {
        return nil, fmt.Errorf("invalid YourProvider configuration: %w", err)
    }

    return yourprovider.NewYourProviderEmbedder(f.config.YourProvider, f.metrics, f.tracer)
}
```

## Observability

The package includes comprehensive observability features:

### Metrics Initialization

The package uses a standardized metrics initialization pattern with `InitMetrics()` and `GetMetrics()`:

```go
import (
    "go.opentelemetry.io/otel/metric"
    "github.com/lookatitude/beluga-ai/pkg/embeddings"
)

// Initialize metrics once at application startup
meter := otel.Meter("beluga-embeddings")
embeddings.InitMetrics(meter)

// Get the global metrics instance
metrics := embeddings.GetMetrics()
if metrics != nil {
    // Metrics are automatically collected for embedding operations
}
```

**Note**: `InitMetrics()` uses `sync.Once` to ensure thread-safe initialization. It should be called once at application startup.

### Metrics

All implementations automatically collect metrics:

- `embeddings_requests_total`: Total number of embedding requests
- `embeddings_request_duration_seconds`: Request duration histogram
- `embeddings_requests_in_flight`: Number of concurrent requests
- `embeddings_errors_total`: Total number of errors
- `embeddings_tokens_processed_total`: Token processing counts

Metrics include labels for `provider`, `model`, and `error_type`.

### Tracing

All operations are traced with OpenTelemetry:

- `embeddings.openai.embed_documents`
- `embeddings.openai.embed_query`
- `embeddings.ollama.embed_documents`
- `embeddings.ollama.embed_query`
- `embeddings.mock.embed_documents`
- `embeddings.mock.embed_query`

Traces include relevant attributes like document count, dimensions, and error information.

### Structured Logging

The package uses structured logging with context propagation and includes trace/span IDs when available.

## Testing

### Unit Tests

The package includes comprehensive unit tests with table-driven tests, mocks, and benchmarks:

```bash
# Run all embedding tests
go test ./pkg/embeddings/...

# Run specific provider tests
go test ./pkg/embeddings/providers/openai/
go test ./pkg/embeddings/providers/ollama/
go test ./pkg/embeddings/providers/mock/

# Run with coverage
go test ./pkg/embeddings/... -cover

# Run benchmarks
go test ./pkg/embeddings/... -bench=.
```

Test coverage includes:
- Configuration validation and defaults
- Factory creation and provider instantiation
- Interface compliance verification
- Error handling with custom error types
- Functional options pattern
- Mock provider with deterministic output
- Performance benchmarks for critical operations

### Mock Testing

Use the mock provider for testing components that depend on embeddings:

```go
config := &embeddings.Config{
    Mock: &embeddings.MockConfig{
        Dimension: 128,
        Seed: 42,
    },
}

factory, _ := embeddings.NewEmbedderFactory(config)
embedder, _ := factory.NewEmbedder("mock")

// Use embedder in tests - output is deterministic with seed
```

### Integration Tests

For integration testing with real providers:

```go
// Set environment variables
os.Setenv("OPENAI_API_KEY", "sk-test...")

config := &embeddings.Config{
    OpenAI: &embeddings.OpenAIConfig{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    },
}

// Test with real API (use sparingly due to costs)
```

## Migration Guide

### From Legacy Factory Pattern

**Old Usage:**
```go
// Old way - direct factory function calls
embedder := embeddings.NewOpenAIEmbedder(apiKey, model)
```

**New Usage:**
```go
// New way - configuration-based factory
config := &embeddings.Config{
    OpenAI: &embeddings.OpenAIConfig{
        APIKey: apiKey,
        Model:  model,
    },
}

factory, _ := embeddings.NewEmbedderFactory(config)
embedder, _ := factory.NewEmbedder("openai")
```

### Breaking Changes

1. **Factory Methods**: Direct constructor calls replaced with configuration-based factory
2. **Configuration**: All configuration now uses structured config structs
3. **Error Handling**: Improved error messages and validation
4. **Observability**: Automatic tracing and metrics (may affect performance monitoring)

### Compatibility

The core `iface.Embedder` interface remains unchanged, ensuring existing implementations continue to work. Only the creation/factory pattern has changed.

---

This package follows the Beluga AI Framework design patterns and provides a robust, extensible foundation for text embedding operations. For questions or contributions, please refer to the framework maintainers.
