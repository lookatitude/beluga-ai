# Implementing Providers in Beluga AI Framework

This guide provides step-by-step instructions for implementing new providers across different package types in the Beluga AI Framework.

## Table of Contents

1. [Overview](#overview)
2. [Common Patterns](#common-patterns)
3. [LLM Provider Implementation](#llm-provider-implementation)
4. [Vector Store Provider Implementation](#vector-store-provider-implementation)
5. [Embeddings Provider Implementation](#embeddings-provider-implementation)
6. [S2S Provider Implementation](#s2s-provider-implementation)
7. [Memory Implementation](#memory-implementation)
8. [Testing Requirements](#testing-requirements)
9. [Best Practices](#best-practices)

---

## Overview

All providers in the Beluga AI Framework follow consistent design patterns:

- **Interface Segregation Principle (ISP)**: Small, focused interfaces
- **Dependency Inversion Principle (DIP)**: Depend on abstractions
- **Factory Pattern**: Provider creation through factories
- **Functional Options**: Flexible configuration
- **Observability**: OTEL metrics, tracing, and structured logging
- **Error Handling**: Custom error types with codes

---

## Common Patterns

### 1. Package Structure

```
pkg/{package}/providers/{provider_name}/
├── config.go          # Provider-specific configuration
├── provider.go        # Main provider implementation
├── streaming.go       # Streaming support (if applicable)
├── init.go            # Auto-registration
├── provider_test.go   # Unit tests
└── streaming_test.go  # Streaming tests (if applicable)
```

### 2. Auto-Registration Pattern

All providers should auto-register using `init()`:

```go
// pkg/{package}/providers/{provider}/init.go
package {provider}

import "github.com/lookatitude/beluga-ai/pkg/{package}"

func init() {
    {package}.GetRegistry().Register("{provider_name}", New{Provider}Factory)
}
```

### 3. Configuration Pattern

```go
// config.go
package {provider}

import (
    "time"
    "github.com/lookatitude/beluga-ai/pkg/{package}"
)

// {Provider}Config extends the base config
type {Provider}Config struct {
    *{package}.Config
    
    // Provider-specific fields
    APIEndpoint string `mapstructure:"api_endpoint" yaml:"api_endpoint" default:"https://api.example.com"`
    Model       string `mapstructure:"model" yaml:"model" default:"default-model"`
    Timeout     time.Duration `mapstructure:"timeout" yaml:"timeout" default:"30s"`
}

// Default{Provider}Config returns default configuration
func Default{Provider}Config() *{Provider}Config {
    return &{Provider}Config{
        Config: {package}.DefaultConfig(),
        APIEndpoint: "https://api.example.com",
        Model: "default-model",
        Timeout: 30 * time.Second,
    }
}

// Validate validates the configuration
func (c *{Provider}Config) Validate() error {
    if c.APIKey == "" {
        return errors.New("API key is required")
    }
    if c.Model == "" {
        return errors.New("model is required")
    }
    return nil
}
```

### 4. Provider Structure Pattern

```go
// provider.go
package {provider}

import (
    "context"
    "time"
    
    "github.com/lookatitude/beluga-ai/pkg/{package}"
    "github.com/lookatitude/beluga-ai/pkg/{package}/iface"
)

// Provider constants
const (
    ProviderName = "{provider_name}"
    DefaultModel = "default-model"
    
    // Error codes
    ErrCodeInvalidAPIKey  = "{provider}_invalid_api_key"
    ErrCodeRateLimit      = "{provider}_rate_limit"
    ErrCodeModelNotFound  = "{provider}_model_not_found"
)

// {Provider}Provider implements the interface
type {Provider}Provider struct {
    config      *{package}.Config
    client      *{ExternalClient} // External SDK client
    metrics     {package}.MetricsRecorder
    tracing     *common.TracingHelper
    retryConfig *common.RetryConfig
    modelName   string
}

// New{Provider}Provider creates a new provider instance
func New{Provider}Provider(config *{package}.Config) (iface.{Interface}, error) {
    // Validate configuration
    if err := {package}.ValidateProviderConfig(context.Background(), config); err != nil {
        return nil, fmt.Errorf("invalid {provider} configuration: %w", err)
    }
    
    // Set defaults
    modelName := config.ModelName
    if modelName == "" {
        modelName = DefaultModel
    }
    
    // Initialize external client
    client := {ExternalClient}.NewClient(config.APIKey)
    
    // Create provider
    provider := &{Provider}Provider{
        config:    config,
        client:    client,
        modelName: modelName,
        metrics:   {package}.GetMetrics(),
        tracing:   common.NewTracingHelper(),
        retryConfig: &common.RetryConfig{
            MaxRetries: config.MaxRetries,
            Delay:      config.RetryDelay,
            Backoff:    config.RetryBackoff,
        },
    }
    
    return provider, nil
}

// Factory function for registration
func New{Provider}Factory() func(*{package}.Config) (iface.{Interface}, error) {
    return func(config *{package}.Config) (iface.{Interface}, error) {
        return New{Provider}Provider(config)
    }
}
```

### 5. Error Handling Pattern

```go
// Handle provider-specific errors
func (p *{Provider}Provider) handle{Provider}Error(operation string, err error) error {
    if err == nil {
        return nil
    }
    
    var errorCode string
    var message string
    
    errStr := err.Error()
    if strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "429") {
        errorCode = ErrCodeRateLimit
        message = "{Provider} API rate limit exceeded"
    } else if strings.Contains(errStr, "authentication") || strings.Contains(errStr, "401") {
        errorCode = ErrCodeInvalidAPIKey
        message = "{Provider} API authentication failed"
    } else {
        errorCode = ErrCodeInvalidRequest
        message = "{Provider} API request failed"
    }
    
    return {package}.New{ErrorType}WithMessage(operation, errorCode, message, err)
}
```

### 6. Observability Pattern

```go
// Generate implements the interface with observability
func (p *{Provider}Provider) Generate(ctx context.Context, input any, options ...core.Option) (any, error) {
    // Start tracing
    ctx = p.tracing.StartOperation(ctx, "{provider}.generate", ProviderName, p.modelName)
    
    start := time.Now()
    
    // Record metrics
    p.metrics.IncrementActiveRequests(ctx, ProviderName, p.modelName)
    defer p.metrics.DecrementActiveRequests(ctx, ProviderName, p.modelName)
    
    // Execute with retry
    var result any
    var err error
    
    retryErr := common.RetryWithBackoff(ctx, p.retryConfig, "{provider}.generate", func() error {
        result, err = p.generateInternal(ctx, input, options...)
        return err
    })
    
    duration := time.Since(start)
    
    if retryErr != nil {
        p.metrics.RecordError(ctx, ProviderName, p.modelName, {package}.GetErrorCode(retryErr), duration)
        p.tracing.RecordError(ctx, retryErr)
        return nil, retryErr
    }
    
    p.metrics.RecordRequest(ctx, ProviderName, p.modelName, duration)
    
    return result, nil
}
```

---

## LLM Provider Implementation

### Step 1: Create Package Structure

```bash
mkdir -p pkg/llms/providers/{provider_name}
cd pkg/llms/providers/{provider_name}
```

### Step 2: Implement Configuration

Create `config.go` with provider-specific configuration extending `llms.Config`.

### Step 3: Implement Provider

Create `provider.go` implementing `iface.ChatModel`:

**Required Methods:**
- `Generate(ctx, messages, options) (Message, error)`
- `StreamChat(ctx, messages, options) (<-chan AIMessageChunk, error)`
- `BindTools(tools) ChatModel`
- `GetModelName() string`
- `GetProviderName() string`

**Also implement `core.Runnable`:**
- `Invoke(ctx, input, options) (any, error)`
- `Batch(ctx, inputs, options) ([]any, error)`
- `Stream(ctx, input, options) (<-chan any, error)`

### Step 4: Add Auto-Registration

Create `init.go`:

```go
package {provider}

import "github.com/lookatitude/beluga-ai/pkg/llms"

func init() {
    llms.GetRegistry().Register("{provider_name}", New{Provider}ProviderFactory)
}
```

### Step 5: Add Tests

Create comprehensive tests in `provider_test.go`:
- Configuration validation
- Successful generation
- Error handling
- Streaming support
- Tool calling (if supported)

**Example Test:**
```go
func Test{Provider}Provider_Generate(t *testing.T) {
    config := llms.DefaultConfig()
    config.Provider = "{provider_name}"
    config.APIKey = "test-key"
    config.ModelName = "test-model"
    
    provider, err := New{Provider}Provider(config)
    require.NoError(t, err)
    
    messages := []schema.Message{
        schema.NewHumanMessage("Hello"),
    }
    
    result, err := provider.Generate(context.Background(), messages)
    require.NoError(t, err)
    assert.NotNil(t, result)
}
```

---

## Vector Store Provider Implementation

### Step 1: Create Package Structure

```bash
mkdir -p pkg/vectorstores/providers/{provider_name}
cd pkg/vectorstores/providers/{provider_name}
```

### Step 2: Implement Configuration

Create `config.go` with provider-specific configuration.

### Step 3: Implement Provider

Create `provider.go` implementing `vectorstores.VectorStore`:

**Required Methods:**
- `AddDocuments(ctx, documents, opts) ([]string, error)`
- `DeleteDocuments(ctx, ids, opts) error`
- `SimilaritySearch(ctx, queryVector, k, opts) ([]Document, []float32, error)`
- `SimilaritySearchByQuery(ctx, query, k, embedder, opts) ([]Document, []float32, error)`
- `AsRetriever(opts) Retriever`
- `GetName() string`

### Step 4: Add Auto-Registration

Create `init.go`:

```go
package {provider}

import "github.com/lookatitude/beluga-ai/pkg/vectorstores"

func init() {
    vectorstores.RegisterProvider("{provider_name}", func(ctx context.Context, config vectorstoresiface.Config) (vectorstores.VectorStore, error) {
        return New{Provider}Store(ctx, config)
    })
}
```

### Step 5: Add Tests

Test all required methods with:
- Successful operations
- Error handling
- Edge cases (empty vectors, invalid IDs, etc.)
- Concurrent access (if applicable)

---

## Embeddings Provider Implementation

### Step 1: Create Package Structure

```bash
mkdir -p pkg/embeddings/providers/{provider_name}
cd pkg/embeddings/providers/{provider_name}
```

### Step 2: Implement Configuration

Create `config.go` with provider-specific configuration extending `embeddings.Config`.

### Step 3: Implement Provider

Create `provider.go` implementing `iface.Embedder`:

**Required Methods:**
- `EmbedDocuments(ctx, texts) ([][]float32, error)`
- `EmbedQuery(ctx, text) ([]float32, error)`
- `GetDimension(ctx) (int, error)` (if supported)

### Step 4: Add Factory Integration

Update `pkg/embeddings/embeddings.go` to include your provider in the factory:

```go
func (f *EmbedderFactory) NewEmbedder(providerType string) (iface.Embedder, error) {
    switch providerType {
    case "openai":
        return f.newOpenAIEmbedder()
    case "{provider_name}":
        return f.new{Provider}Embedder()
    // ...
    }
}
```

### Step 5: Add Tests

Test embedding generation, dimension retrieval, and error handling.

---

## S2S Provider Implementation

### Step 1: Create Package Structure

```bash
mkdir -p pkg/voice/s2s/providers/{provider_name}
cd pkg/voice/s2s/providers/{provider_name}
```

### Step 2: Implement Configuration

Create `config.go` extending `s2s.Config`.

### Step 3: Implement Provider

Create `provider.go` implementing `iface.S2SProvider`:

**Required Methods:**
- `Process(ctx, input, context, opts) (*AudioOutput, error)`
- `Name() string`

**For Streaming Support:**
- Implement `iface.StreamingS2SProvider`
- `StartStreaming(ctx, context, opts) (StreamingSession, error)`

### Step 4: Implement Streaming (if supported)

Create `streaming.go` implementing `iface.StreamingSession`:

**Required Methods:**
- `SendAudio(ctx, audio) error`
- `ReceiveAudio() <-chan AudioOutputChunk`
- `Close() error`

### Step 5: Add Auto-Registration

Create `init.go`:

```go
package {provider}

import "github.com/lookatitude/beluga-ai/pkg/voice/s2s"

func init() {
    s2s.GetRegistry().Register("{provider_name}", New{Provider}Provider)
}
```

### Step 6: Add Tests

Test both non-streaming and streaming modes.

---

## Memory Implementation

### Step 1: Create Package Structure

```bash
mkdir -p pkg/memory/internal/{memory_type}
cd pkg/memory/internal/{memory_type}
```

### Step 2: Implement Memory Type

Create `{memory_type}.go` implementing `iface.Memory`:

**Required Methods:**
- `MemoryVariables() []string`
- `LoadMemoryVariables(ctx, inputs) (map[string]any, error)`
- `SaveContext(ctx, inputs, outputs) error`
- `Clear(ctx) error`

### Step 3: Add Factory Integration

Update `pkg/memory/memory.go` to include your memory type:

```go
func (f *DefaultFactory) CreateMemory(ctx context.Context, config Config) (Memory, error) {
    switch config.Type {
    case MemoryTypeBuffer:
        return f.createBufferMemory(ctx, config)
    case MemoryType{YourType}:
        return f.create{YourType}Memory(ctx, config)
    // ...
    }
}
```

### Step 4: Add Tests

Test all memory operations with various scenarios.

---

## Testing Requirements

### Unit Tests

Every provider must have:
- ✅ Configuration validation tests
- ✅ Successful operation tests
- ✅ Error handling tests
- ✅ Edge case tests
- ✅ Concurrent access tests (if applicable)

### Integration Tests

For providers that make external API calls:
- ✅ Mock external API responses
- ✅ Test retry logic
- ✅ Test timeout handling
- ✅ Test rate limiting

### Test Structure

```go
func Test{Provider}Provider_{Method}(t *testing.T) {
    tests := []struct {
        name    string
        config  *Config
        input   any
        wantErr bool
        errCode string
    }{
        {
            name: "successful operation",
            config: validConfig(),
            input: validInput(),
            wantErr: false,
        },
        {
            name: "invalid config",
            config: invalidConfig(),
            wantErr: true,
            errCode: ErrCodeInvalidConfig,
        },
        // ...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            provider, err := New{Provider}Provider(tt.config)
            if tt.wantErr && err == nil {
                t.Fatal("expected error but got none")
            }
            if !tt.wantErr && err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
            // ... test implementation
        })
    }
}
```

---

## Best Practices

### 1. Error Handling

- Always use custom error types with error codes
- Wrap external errors with context
- Provide actionable error messages
- Log errors with structured logging

### 2. Observability

- Always start OTEL spans for public methods
- Record metrics for all operations
- Include relevant attributes in spans
- Handle context cancellation properly

### 3. Configuration

- Always validate configuration
- Provide sensible defaults
- Support both struct-based and functional options
- Document all configuration options

### 4. Testing

- Aim for 100% code coverage
- Test both success and failure paths
- Test edge cases and boundary conditions
- Use table-driven tests for multiple scenarios

### 5. Documentation

- Document all public functions
- Include usage examples
- Document error codes
- Update README with provider information

### 6. Performance

- Implement connection pooling (if applicable)
- Support batch operations
- Implement proper retry logic with backoff
- Handle rate limiting gracefully

### 7. Security

- Never log API keys or sensitive data
- Validate all inputs
- Use secure defaults
- Follow provider-specific security best practices

---

## Checklist

Before submitting a provider implementation, ensure:

- [ ] All required interface methods implemented
- [ ] Configuration validation implemented
- [ ] Error handling with custom error types
- [ ] OTEL metrics and tracing integrated
- [ ] Auto-registration via `init()`
- [ ] Comprehensive unit tests (>90% coverage)
- [ ] Integration tests with mocked APIs
- [ ] Documentation updated
- [ ] README includes provider information
- [ ] Examples provided
- [ ] Health check implemented (if applicable)
- [ ] Streaming support (if applicable)
- [ ] Retry logic with exponential backoff
- [ ] Rate limiting handled
- [ ] Context cancellation respected

---

## Examples

See existing implementations for reference:

- **LLM Provider**: `pkg/llms/providers/openai/`
- **Vector Store**: `pkg/vectorstores/providers/pgvector/`
- **Embeddings**: `pkg/embeddings/providers/openai/`
- **S2S Provider**: `pkg/voice/s2s/providers/amazon_nova/` (structure, needs API implementation)

---

## Getting Help

- Review existing provider implementations
- Check `docs/package_design_patterns.md` for design patterns
- Review `docs/architecture/` for architecture details
- Ask questions in project discussions
