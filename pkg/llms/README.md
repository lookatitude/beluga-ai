# LLMs Package

The `llms` package provides Large Language Model (LLM) implementations following the Beluga AI Framework design patterns. It supports multiple LLM providers with consistent interfaces, comprehensive error handling, observability, and extensible architecture.

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Core Interfaces](#core-interfaces)
4. [Configuration](#configuration)
5. [Usage Examples](#usage-examples)
6. [Provider Support](#provider-support)
7. [Error Handling](#error-handling)
8. [Observability](#observability)
9. [Testing](#testing)
10. [Extensibility](#extensibility)
11. [Best Practices](#best-practices)
12. [Migration Guide](#migration-guide)

## Overview

The LLMs package is a comprehensive framework for integrating Large Language Models into applications. It provides:

### ✅ Production Ready Features
- **Unified Interface**: Consistent API across all LLM providers
- **Streaming Support**: Real-time response streaming with tool call chunks
- **Tool Calling**: Cross-provider function calling capabilities
- **Error Handling**: Comprehensive error types with retry logic
- **Configuration Management**: Functional options with validation
- **Observability**: OpenTelemetry tracing, metrics, and structured logging
- **Factory Pattern**: Clean provider registration and instantiation
- **Batch Processing**: Concurrent batch operations with configurable limits
- **Dependency Injection**: Interface-based design for testability

### 🚧 Framework Features
- **Multi-modal Support**: Image and file processing capabilities
- **Advanced Caching**: Response caching with invalidation
- **Cost Tracking**: Token usage and cost calculation
- **Model Routing**: Intelligent model selection based on complexity
- **Rate Limiting**: Advanced rate limiting with token bucket algorithms

## Architecture

The package follows SOLID principles and the Beluga AI Framework patterns:

```
pkg/llms/
├── iface/                 # Interface definitions (ISP compliant)
│   └── chat_model.go      # Core ChatModel and LLM interfaces
├── internal/              # Private implementations and utilities
│   ├── common/            # Shared utilities and helpers
│   │   ├── retry.go       # Retry logic with backoff
│   │   ├── tracing.go     # Tracing utilities
│   │   ├── validation.go  # Input validation helpers
│   │   └── utils.go       # General utilities
│   └── providers/         # Provider-specific implementations
├── providers/             # Provider implementations
│   ├── anthropic/         # Anthropic Claude implementation
│   ├── openai/            # OpenAI GPT implementation
│   ├── bedrock/           # AWS Bedrock implementation
│   ├── ollama/            # Ollama implementation
│   └── mock/              # Mock implementation for testing
├── config.go              # Configuration management and validation
├── errors.go              # Custom error types with proper wrapping
├── tracing.go             # OpenTelemetry tracing integration
├── metrics.go             # Metrics collection framework
├── llms.go                # Main package API and factory functions
├── llms_test.go           # Comprehensive test suite
├── examples_test.go       # Usage examples and documentation
└── README.md              # This documentation
```

### Design Principles

#### 1. Interface Segregation Principle (ISP)
```go
// Good: Focused interfaces
type ChatModel interface {
    Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error)
    StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan AIMessageChunk, error)
    BindTools(toolsToBind []tools.Tool) ChatModel
    GetModelName() string
}

// Avoid: Kitchen sink interface
type EverythingProvider interface {
    Generate(ctx context.Context, prompt string) (string, error)
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    Store(ctx context.Context, docs []Document) error
    Search(ctx context.Context, query string) ([]Document, error)
}
```

#### 2. Dependency Inversion Principle (DIP)
```go
// High-level modules don't depend on low-level modules
type Agent struct {
    llm    ChatModel    // Interface, not concrete implementation
    memory MemoryStore  // Interface, not concrete implementation
}

func NewAgent(llm ChatModel, memory MemoryStore) *Agent {
    return &Agent{llm: llm, memory: memory}
}
```

#### 3. Single Responsibility Principle (SRP)
- **Package Level**: Each package has one primary responsibility
- **Function Level**: Each function has one reason to change
- **Struct Level**: Each struct has one clear purpose

#### 4. Composition over Inheritance
```go
// Prefer embedding interfaces/structs over type hierarchies
type EnhancedProvider struct {
    ChatModel           // Embedded interface
    *RetryConfig        // Embedded configuration
    TracingHelper       // Embedded tracing utilities
}
```

## Core Interfaces

### ChatModel Interface
```go
type ChatModel interface {
	core.Runnable

	// Generate takes a series of messages and returns an AI message.
	// It is a single call to the model with no streaming.
	Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error)

	// StreamChat takes a series of messages and returns a channel of AIMessageChunk.
	// This allows for streaming responses from the model.
	StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan AIMessageChunk, error)

	// BindTools binds a list of tools to the ChatModel. The returned ChatModel
	// will then be able to call these tools.
	BindTools(toolsToBind []tools.Tool) ChatModel

	// GetModelName returns the model name used by this ChatModel instance.
	GetModelName() string
}
```

### AIMessageChunk
```go
type AIMessageChunk struct {
	Content        string                    // Text content of the chunk
	ToolCallChunks []schema.ToolCallChunk    // Tool call information if present
	AdditionalArgs map[string]interface{}    // Provider-specific arguments or metadata
	Err            error                     // Error encountered during streaming for this chunk
}
```

### LLM Interface (Simplified)
```go
type LLM interface {
	// Invoke sends a single request to the LLM and gets a single response.
	Invoke(ctx context.Context, prompt string, options ...core.Option) (string, error)

	// GetModelName returns the specific model name being used.
	GetModelName() string

	// GetProviderName returns the name of the LLM provider.
	GetProviderName() string
}
```

### Factory Interface
```go
type Factory struct {
	providers         map[string]ChatModel
	llms              map[string]LLM
	providerFactories map[string]func(*Config) (ChatModel, error)
}

func (f *Factory) RegisterProvider(name string, provider ChatModel)
func (f *Factory) GetProvider(name string) (ChatModel, error)
func (f *Factory) CreateProvider(providerName string, config *Config) (ChatModel, error)
```

### Unification with ChatModels
The ChatModel interface embeds the LLM interface for better composition. This allows ChatModels to be used where an LLM is required, providing greater flexibility.

## Quick Start

### Creating a ChatModel

```go
package main

import (
    "context"
    "log"

    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

func main() {
    // Initialize metrics (optional)
    // metrics := llms.NewMetrics(meter)
    // llms.InitMetrics(meter)

    // Create Anthropic ChatModel
    anthropicChat, err := llms.NewAnthropicChat(
        llms.WithProvider("anthropic"),
        llms.WithModelName("claude-3-sonnet-20240229"),
        llms.WithAPIKey("your-anthropic-api-key"),
        llms.WithTemperature(0.7),
        llms.WithMaxTokens(1024),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Create messages
    messages := []schema.Message{
        schema.NewSystemMessage("You are a helpful assistant."),
        schema.NewHumanMessage("What is the capital of France?"),
    }

    // Generate response
    response, err := anthropicChat.Generate(context.Background(), messages)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Response: %s", response.GetContent())
}
```

### Using Streaming

```go
// Stream chat responses
streamChan, err := anthropicChat.StreamChat(context.Background(), messages)
if err != nil {
    log.Fatal(err)
}

for chunk := range streamChan {
    if chunk.Err != nil {
        log.Printf("Stream error: %v", chunk.Err)
        break
    }
    log.Printf("Chunk: %s", chunk.Content)
}
```

### Using Tool Calling

```go
// Bind tools to the model
calculatorTool := tools.NewCalculatorTool()
modelWithTools := anthropicChat.BindTools([]tools.Tool{calculatorTool})

// Use tool calling
messages := []schema.Message{
    schema.NewHumanMessage("What is 15 * 23?"),
}

response, err := modelWithTools.Generate(context.Background(), messages)
if err != nil {
    log.Fatal(err)
}

// Check for tool calls in response
if aiMsg, ok := response.(*schema.AIMessage); ok {
    for _, toolCall := range aiMsg.ToolCalls {
        log.Printf("Tool call: %s with args %s", toolCall.Name, toolCall.Arguments)
    }
}
```

### Using the Factory Pattern

```go
// Create factory
factory := llms.NewFactory()

// Register providers
factory.RegisterProvider("claude", anthropicChat)

// Get provider by name
model, err := factory.GetProvider("claude")
if err != nil {
    log.Fatal(err)
}

// Use the model
response, err := model.Generate(context.Background(), messages)
```

### Batch Processing

```go
// Prepare multiple inputs
inputs := []any{
    []schema.Message{schema.NewHumanMessage("Hello!")},
    []schema.Message{schema.NewHumanMessage("How are you?")},
}

// Process batch
results, err := anthropicChat.Batch(context.Background(), inputs)
if err != nil {
    log.Fatal(err)
}

for i, result := range results {
    if msg, ok := result.(schema.Message); ok {
        log.Printf("Response %d: %s", i+1, msg.GetContent())
    }
}
```

## Configuration

The package supports configuration through functional options with comprehensive validation:

### Basic Configuration

```go
config := llms.NewConfig(
    llms.WithProvider("anthropic"),
    llms.WithModelName("claude-3-sonnet-20240229"),
    llms.WithAPIKey("your-api-key"),
    llms.WithTemperature(0.7),
    llms.WithMaxTokens(2048),
    llms.WithMaxConcurrentBatches(10),
    llms.WithRetryConfig(3, time.Second, 2.0),
    llms.WithObservability(true, true, true), // tracing, metrics, logging
    llms.WithToolCalling(true),
)
```

### Provider-Specific Configuration

```go
// Anthropic-specific configuration
anthropicConfig := llms.NewConfig(
    llms.WithProvider("anthropic"),
    llms.WithModelName("claude-3-opus-20240229"),
    llms.WithAPIKey("your-anthropic-key"),
    llms.WithProviderSpecific("api_version", "2023-06-01"),
)

// OpenAI-specific configuration
openaiConfig := llms.NewConfig(
    llms.WithProvider("openai"),
    llms.WithModelName("gpt-4-turbo-preview"),
    llms.WithAPIKey("your-openai-key"),
    llms.WithBaseURL("https://api.openai.com/v1"),
    llms.WithProviderSpecific("organization", "your-org-id"),
)

// AWS Bedrock configuration
bedrockConfig := llms.NewConfig(
    llms.WithProvider("bedrock"),
    llms.WithModelName("anthropic.claude-3-sonnet-20240229-v1:0"),
    llms.WithProviderSpecific("region", "us-east-1"),
)
```

### Validation

```go
if err := llms.ValidateProviderConfig(context.Background(), config); err != nil {
    log.Fatal("Invalid configuration:", err)
}
```

## Error Handling

The package provides comprehensive error handling with custom error types:

```go
response, err := model.Generate(context.Background(), messages)
if err != nil {
    // Check for specific error types
    if llmErr := llms.GetLLMError(err); llmErr != nil {
        switch llmErr.Code {
        case llms.ErrCodeRateLimit:
            // Implement backoff and retry
            time.Sleep(time.Minute)
            // retry...
        case llms.ErrCodeAuthentication:
            log.Fatal("Authentication failed - check API key")
        case llms.ErrCodeInvalidRequest:
            log.Printf("Invalid request: %v", llmErr.Err)
        default:
            log.Printf("LLM error (%s): %v", llmErr.Code, llmErr.Err)
        }
    } else {
        log.Printf("Unknown error: %v", err)
    }
}
```

### Error Types

- **`LLMError`**: General LLM operation errors with operation name, error code, and underlying error
- **`ValidationError`**: Configuration validation errors
- **`ConfigValidationError`**: Multiple configuration validation errors
- **`ProviderError`**: Provider-specific errors
- **`StreamError`**: Streaming-specific errors

### Retry Logic

```go
// Automatic retry with backoff
result, err := llms.RetryWithBackoff(ctx,
    &llms.RetryConfig{MaxRetries: 3, Delay: time.Second, Backoff: 2.0},
    "generate",
    func() (interface{}, error) {
        return model.Generate(ctx, messages)
    })
```

## Observability

### Metrics

The package includes comprehensive metrics collection using OpenTelemetry:

```go
// Initialize metrics
meter := otel.Meter("beluga-ai-llms")
metrics := llms.NewMetrics(meter)
llms.InitMetrics(meter)

// Metrics are automatically collected for:
// - Request count by provider and model
// - Request duration histograms
// - Error counts by error code
// - Token usage metrics
// - Active request counters
// - Tool call metrics
```

### Tracing

Distributed tracing support for end-to-end observability:

```go
// Create tracer
tracer := llms.NewTracer("llms")

// Start operation span
ctx, span := llms.StartSpan(context.Background(), tracer, "generate", "anthropic", "claude-3-sonnet")
defer span.End()

// Record operation
response, err := model.Generate(ctx, messages)
if err != nil {
    llms.RecordSpanError(span, err)
    return err
}

// Add custom attributes
span.SetAttributes(
    attribute.Int("response_length", len(response.GetContent())),
    attribute.Bool("has_tool_calls", len(response.ToolCalls) > 0),
)

return response, nil
```

### Structured Logging

```go
// Log with context
log.WithFields(llms.LoggerAttrs("anthropic", "claude-3-sonnet", "generate")).
    WithField("input_tokens", usage.InputTokens).
    WithField("output_tokens", usage.OutputTokens).
    Info("LLM request completed")
```

## Provider Support

### Supported Providers

| Provider | Status | Features |
|----------|--------|----------|
| **Anthropic** | ✅ Production | Streaming, Tool Calling, Messages API |
| **OpenAI** | ✅ Production | Streaming, Tool Calling, Completions API |
| **AWS Bedrock** | ✅ Production | Multi-model support (Claude, Titan), Streaming |
| **Ollama** | ✅ Production | Local models, Streaming, Open-source models |
| **Mock** | ✅ Production | Testing and development |

### Provider-Specific Usage

#### Anthropic Claude

```go
anthropic, err := llms.NewAnthropicChat(
    llms.WithModelName("claude-3-opus-20240229"),
    llms.WithAPIKey("your-key"),
    llms.WithMaxTokens(4096),
    llms.WithTemperature(0.1), // Lower temperature for more focused responses
)
```

#### OpenAI GPT

```go
openai, err := llms.NewOpenAIChat(
    llms.WithModelName("gpt-4-turbo-preview"),
    llms.WithAPIKey("your-key"),
    llms.WithBaseURL("https://api.openai.com/v1"),
    llms.WithOrganization("your-org-id"),
)
```

#### AWS Bedrock

```go
bedrock, err := llms.NewBedrockLLM(context.Background(),
    "anthropic.claude-3-sonnet-20240229-v1:0",
    llms.WithBedrockMaxConcurrentBatches(5),
)

// Or using configuration
bedrockConfig := llms.NewConfig(
    llms.WithProvider("bedrock"),
    llms.WithModelName("anthropic.claude-3-sonnet-20240229-v1:0"),
    llms.WithProviderSpecific("region", "us-east-1"),
)
```

#### Ollama (Local Models)

```go
ollama, err := llms.NewOllamaChat(
    llms.WithModelName("llama2"),
    llms.WithProviderSpecific("base_url", "http://localhost:11434"),
)

// Or using configuration
ollamaConfig := llms.NewConfig(
    llms.WithProvider("ollama"),
    llms.WithModelName("codellama"),
    llms.WithProviderSpecific("base_url", "http://localhost:11434"),
)
```

## Tool Integration

The package supports tool calling across all major providers:

```go
// Define tools
calculator := tools.NewCalculatorTool()
webSearch := tools.NewWebSearchTool()

// Bind tools to model
modelWithTools := model.BindTools([]tools.Tool{calculator, webSearch})

// Use tool calling
messages := []schema.Message{
    schema.NewHumanMessage("Calculate 15 * 23 and search for the result online"),
}

response, err := modelWithTools.Generate(context.Background(), messages)
if err != nil {
    log.Fatal(err)
}

// Handle tool calls
for _, toolCall := range response.ToolCalls {
    switch toolCall.Name {
    case "calculator":
        result, err := calculator.Execute(context.Background(), toolCall.Arguments)
        // Handle result...
    case "web_search":
        result, err := webSearch.Execute(context.Background(), toolCall.Arguments)
        // Handle result...
    }
}
```

## Testing

The package includes comprehensive testing utilities:

### Mock LLM for Testing

```go
func TestChatModel(t *testing.T) {
    // Create mock LLM
    mockLLM := llms.NewMockLLM(llms.MockConfig{
        ModelName:     "mock-model",
        Responses:     []string{"Mock response 1", "Mock response 2"},
        ExpectedError: nil,
    })

    // Use in tests
    messages := []schema.Message{schema.NewHumanMessage("Test")}
    response, err := mockLLM.Generate(context.Background(), messages)

    assert.NoError(t, err)
    assert.Equal(t, "Mock response 1", response.GetContent())
}
```

### Table-Driven Tests

```go
func TestEnsureMessages(t *testing.T) {
    tests := []struct {
        name     string
        input    any
        expected []schema.Message
        wantErr  bool
    }{
        {
            name:     "string input",
            input:    "hello world",
            expected: []schema.Message{schema.NewHumanMessage("hello world")},
            wantErr:  false,
        },
        {
            name:     "message slice",
            input:    []schema.Message{schema.NewHumanMessage("test")},
            expected: []schema.Message{schema.NewHumanMessage("test")},
            wantErr:  false,
        },
        {
            name:     "invalid input",
            input:    123,
            expected: nil,
            wantErr:  true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := llms.EnsureMessages(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expected, result)
            }
        })
    }
}
```

## Best Practices

### 1. Error Handling

```go
// Always check for specific error types and implement appropriate handling
response, err := model.Generate(ctx, messages)
if err != nil {
    if llms.IsRetryableError(err) {
        // Implement exponential backoff retry
        time.Sleep(backoff.NextBackOff())
        return retry()
    }

    if llms.GetLLMErrorCode(err) == llms.ErrCodeRateLimit {
        // Specific rate limit handling
        return handleRateLimit()
    }

    // Log and return error
    log.Printf("LLM error: %v", err)
    return err
}
```

### 2. Resource Management

```go
// Always clean up resources
model := llms.NewAnthropicChat(llms.WithAPIKey("key"))
defer func() {
    // Close connections, cleanup resources
    if closer, ok := model.(io.Closer); ok {
        closer.Close()
    }
}()
```

### 3. Configuration Validation

```go
// Validate configuration before use
config := llms.NewConfig(llms.WithProvider("anthropic"))
if err := llms.ValidateProviderConfig(ctx, config); err != nil {
    log.Fatal("Invalid LLM configuration:", err)
}
```

### 4. Observability

```go
// Always use tracing for operations
ctx, span := tracer.Start(ctx, "llm.generate")
defer span.End()

response, err := model.Generate(ctx, messages)
if err != nil {
    span.RecordError(err)
    span.SetStatus(codes.Error, err.Error())
    return err
}

span.SetAttributes(
    attribute.String("model", model.GetModelName()),
    attribute.Int("response_length", len(response.GetContent())),
)
```

### 5. Streaming Best Practices

```go
// Handle streaming responses properly
streamChan, err := model.StreamChat(ctx, messages)
if err != nil {
    return err
}

var fullResponse strings.Builder
for chunk := range streamChan {
    if chunk.Err != nil {
        return fmt.Errorf("stream error: %w", chunk.Err)
    }

    fullResponse.WriteString(chunk.Content)

    // Handle tool calls
    for _, toolCall := range chunk.ToolCallChunks {
        log.Printf("Tool call: %s", toolCall.Name)
    }
}
```

## Implementation Status

### ✅ Completed Features

- **Architecture**: SOLID principles with clean separation of concerns
- **Core Interfaces**: Comprehensive ChatModel and LLM interfaces (ISP compliant)
- **Provider Implementations**: Anthropic, OpenAI, Bedrock, Gemini, Ollama, Mock
- **Configuration**: Advanced configuration management with validation
- **Error Handling**: Custom error types with retry logic and error codes
- **Observability**: OpenTelemetry metrics, tracing, and logging
- **Factory Pattern**: Provider registration and management
- **Tool Calling**: Cross-provider tool calling support
- **Streaming**: Real-time streaming with tool call chunks
- **Batch Processing**: Concurrent batch operations
- **Testing**: Comprehensive unit tests with mocks

### 🚧 In Development / Enhancement

- **Advanced Tool Calling**: Enhanced tool call validation and error handling
- **Multi-modal Support**: Image and file processing capabilities
- **Caching Layer**: Response caching for improved performance
- **Rate Limiting**: Advanced rate limiting with token bucket algorithms
- **Cost Tracking**: Token usage cost calculation and tracking
- **Model Fine-tuning**: Integration with fine-tuned models

### 📋 Roadmap

1. **Enhanced Multi-modal**: Support for vision models and file processing
2. **Advanced Caching**: Intelligent response caching with invalidation
3. **Cost Optimization**: Token usage tracking and cost optimization
4. **Model Router**: Intelligent model selection based on task complexity
5. **Streaming Optimizations**: Enhanced streaming with backpressure handling
6. **Provider Extensions**: Support for additional LLM providers
7. **Performance Monitoring**: Advanced performance metrics and alerting

## Provider-Specific Notes

### Anthropic Claude
- **Best for**: Complex reasoning, analysis, and creative writing
- **Models**: Claude 3 Opus, Sonnet, Haiku
- **Features**: Excellent tool calling, strong reasoning capabilities
- **Limitations**: No image generation, higher latency than some providers

### OpenAI GPT
- **Best for**: General-purpose tasks, code generation, conversation
- **Models**: GPT-4 Turbo, GPT-4, GPT-3.5 Turbo
- **Features**: Strong coding capabilities, fast inference
- **Limitations**: Higher cost, less reasoning-focused than Claude

### AWS Bedrock
- **Best for**: Enterprise deployments, multi-model access, secure environments
- **Models**: Claude (Anthropic), Titan (Amazon), Llama, Mistral, Cohere
- **Features**: Single API for multiple models, enterprise security, fine-grained access control
- **Limitations**: More complex setup, region-specific availability, AWS account required

### Ollama
- **Best for**: Local development, offline usage, open-source models
- **Models**: Llama, CodeLlama, Mistral, Phi, and many others
- **Features**: Local execution, no API costs, privacy-focused, extensive model support
- **Limitations**: Requires local hardware resources, model download and setup needed

### Google Gemini
- **Best for**: Multi-modal tasks, fast inference
- **Models**: Gemini 1.5 Pro, Gemini 1.5 Flash
- **Features**: Multi-modal support, fast responses
- **Limitations**: Newer models, less tool calling maturity

## Contributing

When adding new providers or features:

1. **Follow the architecture**: Use the established patterns in `iface/`, `internal/`, and `config.go`
2. **Add comprehensive tests**: Include unit tests, integration tests, and error scenarios
3. **Implement observability**: Add metrics, tracing, and structured logging
4. **Handle errors properly**: Use custom error types and proper error wrapping
5. **Update documentation**: Add examples and update this README
6. **Maintain compatibility**: Don't break existing interfaces

### Development Guidelines

- Use functional options for configuration
- Implement proper error handling with custom error types
- Add OpenTelemetry tracing and metrics
- Write comprehensive tests with table-driven patterns
- Follow the existing code style and naming conventions
- Add documentation comments for all exported functions
- Update this README when adding new features

### Adding a New Provider

1. **Create provider implementation** in `providers/` directory:
   ```go
   type NewProvider struct {
       config *llms.Config
       client *ProviderClient
   }
   ```

2. **Implement ChatModel interface**:
   ```go
   func (p *NewProvider) Generate(ctx context.Context, messages []schema.Message, opts ...core.Option) (schema.Message, error) {
       // Implementation
   }
   ```

3. **Add factory function**:
   ```go
   func NewNewProvider(opts ...llms.Option) (*NewProvider, error) {
       // Implementation
   }
   ```

4. **Add tests** with mocks and integration tests

5. **Update documentation** with usage examples

## License

This package is part of the Beluga AI Framework and follows the same license terms.
