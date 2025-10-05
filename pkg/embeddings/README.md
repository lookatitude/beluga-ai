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
Uses OpenAI's embedding models for high-quality embeddings with cloud-based processing.

**Supported Models:**
- `text-embedding-ada-002` (1536 dimensions) - Legacy model, good balance of quality and cost
- `text-embedding-3-small` (1536 dimensions) - Newer model, optimized for speed and cost
- `text-embedding-3-large` (3072 dimensions) - Highest quality model for complex tasks

#### Setup Instructions

1. **Get API Key:**
   ```bash
   # Visit https://platform.openai.com/api-keys
   # Create a new API key with appropriate permissions
   export OPENAI_API_KEY="sk-your-api-key-here"
   ```

2. **Verify API Access:**
   ```bash
   # Test your API key
   curl -H "Authorization: Bearer $OPENAI_API_KEY" \
        -H "Content-Type: application/json" \
        -d '{"model": "text-embedding-ada-002", "input": "test"}' \
        https://api.openai.com/v1/embeddings
   ```

3. **Configuration:**
   ```yaml
   embeddings:
     openai:
       api_key: "${OPENAI_API_KEY}"
       model: "text-embedding-3-small"  # Recommended for most use cases
       timeout: "30s"
       max_retries: 3
       enabled: true
   ```

4. **Cost Optimization:**
   - Use `text-embedding-3-small` for most applications (80% cost reduction vs ada-002)
   - Batch requests when possible (up to 2048 inputs per request)
   - Implement caching for repeated texts

#### Performance Characteristics
- **Latency:** 100-500ms per request
- **Throughput:** Limited by OpenAI rate limits (varies by account tier)
- **Cost:** $0.00002-$0.00013 per 1K tokens
- **Reliability:** High (99.9% uptime SLA)

### Ollama
Uses local Ollama models for private, offline embeddings with full data privacy.

**Supported Models:**
- `nomic-embed-text` (768 dimensions) - Optimized for English text, fast and efficient
- `all-minilm` (384 dimensions) - Multi-language support, compact model
- `mxbai-embed-large` (1024 dimensions) - High-quality embeddings for complex tasks
- Any custom embedding model supported by Ollama

#### Setup Instructions

1. **Install Ollama:**
   ```bash
   # macOS
   brew install ollama

   # Linux
   curl -fsSL https://ollama.ai/install.sh | sh

   # Windows
   # Download from https://ollama.ai/download
   ```

2. **Start Ollama Service:**
   ```bash
   # Start Ollama in the background
   ollama serve

   # Or run as a service (Linux)
   sudo systemctl start ollama
   ```

3. **Pull Embedding Models:**
   ```bash
   # Pull recommended models
   ollama pull nomic-embed-text    # Fast, good for English
   ollama pull all-minilm          # Multi-language support
   ollama pull mxbai-embed-large   # High quality
   ```

4. **Verify Installation:**
   ```bash
   # List available models
   ollama list

   # Test model availability
   ollama show nomic-embed-text
   ```

5. **Configuration:**
   ```yaml
   embeddings:
     ollama:
       server_url: "http://localhost:11434"  # Default Ollama port
       model: "nomic-embed-text"             # Recommended for most use cases
       timeout: "30s"
       max_retries: 3
       enabled: true
   ```

6. **Custom Server Configuration:**
   ```bash
   # Run Ollama on a custom port
   OLLAMA_HOST=0.0.0.0:8080 ollama serve

   # Update configuration accordingly
   server_url: "http://localhost:8080"
   ```

#### Performance Characteristics
- **Latency:** 50-200ms per request (varies by model and hardware)
- **Throughput:** Limited by local hardware capabilities
- **Cost:** Free (one-time model download)
- **Reliability:** Depends on local hardware stability
- **Privacy:** Complete data privacy (no external API calls)

#### Hardware Recommendations
- **CPU-only:** 4+ CPU cores, 8GB+ RAM
- **GPU acceleration:** NVIDIA GPU with CUDA support (significantly faster)
- **Memory:** 8GB+ RAM recommended, 16GB+ for large models
- **Storage:** 2GB+ free space per model

### Mock
Provides deterministic mock embeddings for testing, development, and load testing scenarios.

**Features:**
- **Deterministic output:** Same input always produces same embedding (with seed control)
- **Configurable dimensions:** Support for any embedding dimension size
- **Load simulation:** Built-in support for delays, rate limiting, and error injection
- **Performance testing:** Ideal for benchmarking and performance regression testing
- **No external dependencies:** Works offline without API keys or network access

#### Setup Instructions

The mock provider requires no external setup and works immediately:

```yaml
embeddings:
  mock:
    dimension: 128        # Any positive integer
    seed: 42             # For deterministic output (optional)
    randomize_nil: false # Return zero vectors for empty strings
    enabled: true
    # Optional load simulation features
    simulate_delay: "0s"          # Add artificial delay
    simulate_errors: false        # Enable error injection
    error_rate: 0.0               # Error probability (0.0-1.0)
    rate_limit_per_second: 0      # Rate limiting (0 = disabled)
    memory_pressure: false        # Simulate memory pressure
    performance_degrade: false    # Gradual performance degradation
```

#### Use Cases

**Unit Testing:**
```go
// Deterministic embeddings for reliable tests
config := &embeddings.Config{
    Mock: &embeddings.MockConfig{
        Dimension: 128,
        Seed:      12345,  // Same seed = same results across test runs
        Enabled:   true,
    },
}
```

**Load Testing:**
```go
// Simulate production conditions
config := &embeddings.Config{
    Mock: &embeddings.MockConfig{
        Dimension:          256,
        Seed:               42,
        Enabled:            true,
        SimulateDelay:      10 * time.Millisecond,  // Add latency
        RateLimitPerSecond: 100,                    // Rate limiting
        ErrorRate:          0.05,                   // 5% error rate
    },
}
```

**Performance Benchmarking:**
```go
// Fast, deterministic performance tests
config := &embeddings.Config{
    Mock: &embeddings.MockConfig{
        Dimension: 512,     // Larger dimensions for memory testing
        Seed:      999,     // Different seed for variety
        Enabled:   true,
    },
}
```

#### Performance Characteristics
- **Latency:** <1ms per request (configurable with delays)
- **Throughput:** 10,000+ ops/sec (limited only by hardware)
- **Cost:** Free
- **Reliability:** 100% (no network dependencies)
- **Determinism:** Perfect reproducibility with fixed seeds

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

### Advanced Configuration Examples

#### Mock Provider with Load Simulation

The mock provider supports advanced load simulation features for testing under various conditions:

```yaml
embeddings:
  mock:
    dimension: 128
    seed: 42
    randomize_nil: false
    enabled: true
    # Load simulation settings
    simulate_delay: "50ms"          # Add artificial delay
    simulate_errors: true           # Enable error simulation
    error_rate: 0.1                 # 10% error rate
    rate_limit_per_second: 100      # Rate limiting
    memory_pressure: false          # Simulate memory pressure
    performance_degrade: false      # Gradual performance degradation
```

#### Environment Variable Configuration

```bash
# OpenAI Configuration
export OPENAI_API_KEY="sk-your-api-key"
export OPENAI_MODEL="text-embedding-3-small"
export OPENAI_TIMEOUT="45s"
export OPENAI_MAX_RETRIES="5"

# Ollama Configuration
export OLLAMA_SERVER_URL="http://localhost:11434"
export OLLAMA_MODEL="nomic-embed-text"
export OLLAMA_TIMEOUT="30s"

# Mock Configuration (for testing)
export MOCK_EMBEDDING_DIMENSION="256"
export MOCK_EMBEDDING_SEED="12345"
```

#### Programmatic Configuration with Load Testing

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/embeddings"
)

func main() {
    // Advanced mock configuration for load testing
    config := &embeddings.Config{
        Mock: &embeddings.MockConfig{
            Dimension:          256,
            Seed:               12345,
            Enabled:            true,
            SimulateDelay:      10 * time.Millisecond,  // Add 10ms delay
            SimulateErrors:     false,                   // No errors
            ErrorRate:          0.0,                     // 0% error rate
            RateLimitPerSecond: 1000,                    // 1000 requests/second limit
            MemoryPressure:     false,                   // No memory pressure
            PerformanceDegrade: false,                   // No degradation
        },
    }

    factory, err := embeddings.NewEmbedderFactory(config)
    if err != nil {
        log.Fatal(err)
    }

    embedder, err := factory.NewEmbedder("mock")
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Test batch processing with various document sizes
    smallDocs := []string{"Short doc 1", "Short doc 2"}
    largeDocs := make([]string, 50)
    for i := range largeDocs {
        largeDocs[i] = fmt.Sprintf("This is document number %d with more content for testing batch processing capabilities.", i)
    }

    // Process small batch
    smallEmbeddings, err := embedder.EmbedDocuments(ctx, smallDocs)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Processed %d small documents", len(smallEmbeddings))

    // Process large batch
    largeEmbeddings, err := embedder.EmbedDocuments(ctx, largeDocs)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Processed %d large documents", len(largeEmbeddings))
}
```

#### Multi-Provider Configuration

```go
package main

import (
    "context"
    "log"

    "github.com/lookatitude/beluga-ai/pkg/embeddings"
)

func main() {
    // Configure multiple providers
    config := &embeddings.Config{
        OpenAI: &embeddings.OpenAIConfig{
            APIKey:     "sk-your-openai-key",
            Model:      "text-embedding-3-small",
            Timeout:    30 * time.Second,
            MaxRetries: 3,
            Enabled:    true,
        },
        Ollama: &embeddings.OllamaConfig{
            ServerURL:  "http://localhost:11434",
            Model:      "nomic-embed-text",
            Timeout:    30 * time.Second,
            MaxRetries: 3,
            Enabled:    true,
        },
        Mock: &embeddings.MockConfig{
            Dimension: 128,
            Seed:      42,
            Enabled:   true,
        },
    }

    factory, err := embeddings.NewEmbedderFactory(config)
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Test all available providers
    providers := factory.GetAvailableProviders()
    for _, provider := range providers {
        embedder, err := factory.NewEmbedder(provider)
        if err != nil {
            log.Printf("Failed to create %s embedder: %v", provider, err)
            continue
        }

        // Test basic functionality
        testText := fmt.Sprintf("Test text for %s provider", provider)
        embedding, err := embedder.EmbedQuery(ctx, testText)
        if err != nil {
            log.Printf("%s provider error: %v", provider, err)
            continue
        }

        dimension, err := embedder.GetDimension(ctx)
        if err != nil {
            log.Printf("%s dimension query failed: %v", provider, err)
            continue
        }

        log.Printf("%s provider: generated %d-dimensional embedding", provider, dimension)
        log.Printf("Embedding sample: %v", embedding[:5]) // First 5 dimensions
    }
}
```

#### Error Handling and Resilience

```go
package main

import (
    "context"
    "errors"
    "log"

    "github.com/lookatitude/beluga-ai/pkg/embeddings"
    "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
)

func main() {
    // Configure with fallback providers
    config := &embeddings.Config{
        OpenAI: &embeddings.OpenAIConfig{
            APIKey:     "sk-your-key",
            Model:      "text-embedding-ada-002",
            Timeout:    10 * time.Second,  // Shorter timeout for resilience
            MaxRetries: 2,
            Enabled:    true,
        },
        Ollama: &embeddings.OllamaConfig{
            ServerURL:  "http://localhost:11434",
            Model:      "nomic-embed-text",
            Timeout:    5 * time.Second,
            MaxRetries: 1,
            Enabled:    true,
        },
        Mock: &embeddings.MockConfig{
            Dimension: 128,
            Seed:      42,
            Enabled:   true,  // Always available as fallback
        },
    }

    factory, err := embeddings.NewEmbedderFactory(config)
    if err != nil {
        log.Fatal(err)
    }

    // Resilient embedding function with provider fallback
    embedWithFallback := func(ctx context.Context, text string, preferredProvider string) ([]float32, error) {
        // Try preferred provider first
        embedder, err := factory.NewEmbedder(preferredProvider)
        if err == nil {
            embedding, embedErr := embedder.EmbedQuery(ctx, text)
            if embedErr == nil {
                return embedding, nil
            }
            log.Printf("Preferred provider %s failed: %v", preferredProvider, embedErr)
        }

        // Fallback to available providers
        providers := factory.GetAvailableProviders()
        for _, provider := range providers {
            if provider == preferredProvider {
                continue // Already tried
            }

            embedder, err := factory.NewEmbedder(provider)
            if err != nil {
                continue
            }

            embedding, err := embedder.EmbedQuery(ctx, text)
            if err == nil {
                log.Printf("Successfully used fallback provider: %s", provider)
                return embedding, nil
            }
        }

        return nil, errors.New("all embedding providers failed")
    }

    ctx := context.Background()

    // Test with fallback
    texts := []string{
        "Test document 1",
        "Test document 2",
        "Test document 3",
    }

    for _, text := range texts {
        embedding, err := embedWithFallback(ctx, text, "openai") // Prefer OpenAI, fallback to others
        if err != nil {
            log.Printf("Failed to embed text '%s': %v", text, err)
            continue
        }

        dimension, _ := factory.NewEmbedder("mock") // Get dimension from mock for consistency
        if dim, err := dimension.GetDimension(ctx); err == nil {
            log.Printf("Embedded text (dim=%d): %v...", dim, embedding[:3])
        }
    }
}
```

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

### Load Testing

The package provides comprehensive load testing capabilities for realistic performance evaluation under various conditions.

#### Running Load Tests

```bash
# Run all load tests
go test ./pkg/embeddings/ -run TestLoadTesting -v

# Run specific load scenarios
go test ./pkg/embeddings/ -run TestLoadTestingScenarios -v

# Run load tests with different configurations
go test ./pkg/embeddings/ -run TestLoadTestingWithDifferentConfigurations -v

# Run load testing benchmarks
go test ./pkg/embeddings/ -bench=BenchmarkLoadTest -benchtime=10s

# Run performance regression tests
go test ./pkg/embeddings/ -bench=BenchmarkPerformanceRegressionDetection
```

#### Load Testing Scenarios

The package includes several realistic load testing scenarios:

**Realistic User Session (`testRealisticUserSession`):**
- Simulates typical user interaction patterns
- Includes document processing, queries, and follow-up searches
- Tests both batch and individual operations

**API Burst Traffic (`testAPIBurstTraffic`):**
- Tests sudden spikes in request volume
- Validates system resilience under burst loads
- Measures response times during traffic spikes

**Gradual Load Increase (`testGradualLoadIncrease`):**
- Tests system behavior under slowly increasing load
- Validates scaling characteristics
- Identifies performance degradation points

**Mixed Workload Patterns (`testMixedWorkloadPatterns`):**
- Tests various embedding operation types
- Validates handling of different input sizes and types
- Ensures consistent behavior across operation types

**Error Recovery Scenarios (`testErrorRecoveryScenarios`):**
- Tests system behavior under error conditions
- Validates fallback and recovery mechanisms
- Measures error handling performance

#### Load Testing Configuration

Configure load simulation parameters using the mock provider:

```yaml
embeddings:
  mock:
    dimension: 256
    seed: 12345
    enabled: true
    # Load simulation settings
    simulate_delay: "10ms"          # Add artificial latency
    simulate_errors: true           # Enable error injection
    error_rate: 0.05                # 5% error rate
    rate_limit_per_second: 1000     # Rate limiting
    memory_pressure: false          # Memory pressure simulation
    performance_degrade: false      # Performance degradation
```

#### Custom Load Testing

Create custom load tests using the provided utilities:

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/embeddings"
)

func main() {
    config := &embeddings.Config{
        Mock: &embeddings.MockConfig{
            Dimension:          128,
            Seed:               42,
            Enabled:            true,
            SimulateDelay:      5 * time.Millisecond,
            RateLimitPerSecond: 100,
        },
    }

    factory, err := embeddings.NewEmbedderFactory(config)
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Create test documents
    documents := make([]string, 100)
    for i := range documents {
        documents[i] = fmt.Sprintf("Test document %d for load testing", i)
    }

    // Run concurrent load test
    config := embeddings.ConcurrentLoadTestConfig{
        NumWorkers:           5,
        OperationsPerWorker:  20,
        TestDocuments:        documents,
        Timeout:              30 * time.Second,
    }

    result, err := embeddings.RunConcurrentLoadTest(ctx, embedder, config)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Load test completed:")
    log.Printf("  Total operations: %d", result.TotalOperations)
    log.Printf("  Total errors: %d", result.TotalErrors)
    log.Printf("  Duration: %v", result.Duration)
    log.Printf("  Operations/sec: %.2f", result.OpsPerSecond)
    log.Printf("  Error rate: %.2f%%", result.ErrorRate*100)

    // Run load test scenario
    scenario := embeddings.LoadTestScenario{
        Name:         "custom_scenario",
        Pattern:      embeddings.RampUpLoad,
        Duration:     30 * time.Second,
        Concurrency:  10,
        TargetOpsSec: 500,
    }

    err = embeddings.RunLoadTestScenario(ctx, embedder, scenario, func(result *embeddings.ConcurrentLoadTestResult) {
        log.Printf("Progress: %d ops/sec, %.2f%% error rate",
            int(result.OpsPerSecond), result.ErrorRate*100)
    })

    if err != nil {
        log.Fatal(err)
    }
}
```

#### Interpreting Load Test Results

**Key Metrics to Monitor:**
- **Operations per second**: Overall throughput
- **Error rate**: System reliability under load
- **Latency percentiles**: Response time distribution
- **Memory usage**: Resource consumption patterns
- **CPU utilization**: Processing efficiency

**Performance Baselines:**
- Single query: 10,000+ ops/sec (mock provider)
- Batch processing: 5,000+ docs/sec
- Concurrent users: Scales linearly with worker count

**Common Issues:**
- High error rates indicate configuration problems
- Declining throughput suggests resource bottlenecks
- Memory leaks show as continuously increasing usage
- Inconsistent latency indicates GC pressure

#### Performance Regression Detection

The package includes automatic performance regression detection:

```bash
# Run regression tests
go test ./pkg/embeddings/ -bench=BenchmarkPerformanceRegressionDetection -v

# The tests will fail if performance drops below baseline thresholds:
# - Single query performance: < 80% of baseline
# - Batch processing: < 80% of baseline
# - Concurrent operations: < 70% of baseline
```

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

## Troubleshooting

### Common Issues and Solutions

#### OpenAI Provider Issues

**"invalid api key" error:**
```bash
# Verify your API key is set correctly
echo $OPENAI_API_KEY

# Test API key with curl
curl -H "Authorization: Bearer $OPENAI_API_KEY" \
     https://api.openai.com/v1/models
```
**Solution:** Ensure your API key starts with `sk-` and has sufficient credits.

**"rate limit exceeded" error:**
```go
// Implement exponential backoff
factory, _ := embeddings.NewEmbedderFactory(config,
    embeddings.WithMaxRetries(5),
    embeddings.WithTimeout(60*time.Second),
)
```
**Solution:** Increase timeout and max retries, or implement client-side rate limiting.

**Timeout errors:**
```yaml
embeddings:
  openai:
    timeout: "60s"  # Increase from default 30s
    max_retries: 3
```
**Solution:** Increase timeout values for slower models or network conditions.

#### Ollama Provider Issues

**"connection refused" error:**
```bash
# Check if Ollama is running
curl http://localhost:11434/api/tags

# Start Ollama if not running
ollama serve
```
**Solution:** Ensure Ollama server is running on the configured port (default: 11434).

**"model not found" error:**
```bash
# List available models
ollama list

# Pull the required model
ollama pull nomic-embed-text
```
**Solution:** Pull the required model using `ollama pull <model-name>`.

**Dimension discovery fails:**
```go
// Check model information manually
dimension, err := embedder.GetDimension(ctx)
if err != nil {
    log.Printf("Dimension discovery failed: %v", err)
    // Fallback to known dimensions
    switch embedder.Model() {
    case "nomic-embed-text":
        dimension = 768
    case "all-minilm":
        dimension = 384
    default:
        dimension = 128 // Safe fallback
    }
}
```
**Solution:** The package attempts automatic dimension discovery, but you can implement fallbacks for known models.

#### Configuration Issues

**"config validation failed" error:**
```go
// Check configuration validation
config := &embeddings.Config{...}
if err := config.Validate(); err != nil {
    log.Printf("Configuration error: %v", err)
    // Print detailed validation errors
}
```
**Solution:** Use `config.Validate()` to get detailed validation error messages.

**Environment variables not working:**
```bash
# Export variables before running
export OPENAI_API_KEY="sk-..."
export OLLAMA_MODEL="nomic-embed-text"

# Verify variables are set
env | grep -E "(OPENAI|OLLAMA)"
```
**Solution:** Ensure environment variables are exported in the same shell session.

#### Performance Issues

**Slow embedding generation:**
```go
// Profile performance
go test -bench=. -benchmem ./pkg/embeddings/

// Check concurrent performance
go test -run TestLoadTesting ./pkg/embeddings/ -v
```
**Solution:** Use benchmarks to identify bottlenecks. Consider batch processing for multiple documents.

**Memory usage problems:**
```go
// Monitor memory usage
go test -bench=. -benchmem -memprofile=mem.out ./pkg/embeddings/
// Analyze with: go tool pprof mem.out
```
**Solution:** Profile memory usage and optimize batch sizes or implement streaming for large datasets.

**High error rates in production:**
```yaml
embeddings:
  openai:
    max_retries: 5
    timeout: "45s"
  ollama:
    max_retries: 3
    timeout: "30s"
```
**Solution:** Increase retry counts and timeouts for unreliable network conditions.

#### Testing Issues

**Mock embeddings not deterministic:**
```go
config := &embeddings.Config{
    Mock: &embeddings.MockConfig{
        Seed: 42,  // Use fixed seed for deterministic results
        Enabled: true,
    },
}
```
**Solution:** Use fixed seeds for reproducible test results.

**Load testing shows unexpected failures:**
```go
// Debug load test failures
config := &embeddings.Config{
    Mock: &embeddings.MockConfig{
        SimulateErrors: false,  // Disable error simulation
        ErrorRate: 0.0,
        Enabled: true,
    },
}
```
**Solution:** Disable load simulation features when debugging test failures.

#### Integration Issues

**Vector store compatibility problems:**
```go
// Verify embedding dimensions match vector store requirements
ctx := context.Background()
embedder, _ := factory.NewEmbedder("openai")
dimension, _ := embedder.GetDimension(ctx)

// Ensure vector store is configured for this dimension
vectorStoreConfig := map[string]interface{}{
    "dimension": dimension,
    "metric": "cosine",
}
```
**Solution:** Ensure embedding dimensions match vector store configuration.

**Concurrent access deadlocks:**
```go
// Use context with timeout for all operations
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

embeddings, err := embedder.EmbedDocuments(ctx, documents)
```
**Solution:** Always use contexts with timeouts to prevent hanging operations.

### Debug Logging

Enable detailed logging for troubleshooting:

```go
import "log"

config := &embeddings.Config{
    // ... your config
}

// The package uses OpenTelemetry for observability
// Check traces and metrics for detailed debugging information

// For manual debugging, add logging around operations
ctx := context.Background()
log.Printf("Starting embedding generation...")

embeddings, err := embedder.EmbedDocuments(ctx, texts)
if err != nil {
    log.Printf("Embedding failed: %v", err)
    return err
}

log.Printf("Successfully generated %d embeddings", len(embeddings))
```

### Health Checks

Use built-in health checks to diagnose issues:

```go
// Check provider health
err := factory.CheckHealth(ctx, "openai")
if err != nil {
    log.Printf("OpenAI provider unhealthy: %v", err)
    // Switch to alternative provider
    err = factory.CheckHealth(ctx, "ollama")
    if err == nil {
        log.Println("Falling back to Ollama provider")
        embedder, _ := factory.NewEmbedder("ollama")
        // Continue with Ollama
    }
}
```

### Performance Tuning

Optimize performance based on your use case:

```yaml
# For high-throughput applications
embeddings:
  openai:
    timeout: "30s"
    max_retries: 2  # Fewer retries for speed
  ollama:
    timeout: "15s"  # Faster local processing

# For high-reliability applications
embeddings:
  openai:
    timeout: "60s"
    max_retries: 5  # More retries for reliability
  ollama:
    timeout: "30s"
    max_retries: 3
```

### Getting Help

1. **Check the logs:** Enable debug logging to see detailed operation traces
2. **Run tests:** Use `go test ./pkg/embeddings/... -v` to identify issues
3. **Verify configuration:** Use `config.Validate()` for detailed error messages
4. **Check provider status:** Use `factory.CheckHealth()` to diagnose connectivity
5. **Review documentation:** Check this README for configuration examples
6. **File an issue:** If problems persist, create an issue with full error logs and configuration

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
