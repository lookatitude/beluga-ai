# ChatModels Package

The `chatmodels` package provides chat-based language model implementations following the Beluga AI Framework design patterns. It implements chat models that can handle conversation-like interactions with various providers including OpenAI, Anthropic, and local models.

## Current Capabilities

**✅ Production Ready:**
- Multiple provider support (OpenAI, mock implementations)
- Streaming and non-streaming message generation
- Comprehensive error handling with custom error types
- Configuration management with functional options
- Clean architecture following SOLID principles
- Health checking and model information APIs
- Unit tests for core functionality

**🚧 Available as Framework:**
- OpenAI API integration (placeholder implementation)
- OpenTelemetry metrics/tracing interfaces (framework ready)
- Streaming support (basic implementation)
- Provider abstraction (extensible architecture)

## Features

- **Multiple Provider Support**: Support for different chat model providers (OpenAI, Anthropic, local models)
- **Streaming & Non-Streaming**: Both synchronous and streaming message generation
- **Observability**: OpenTelemetry tracing and metrics framework (interfaces ready)
- **Configurable Generation**: Retry logic, timeouts, and customizable parameters
- **Error Handling**: Comprehensive error types with proper error wrapping
- **Health Monitoring**: Built-in health checking for model availability
- **Dependency Injection**: Clean architecture with interface-based design

## Architecture

The package follows SOLID principles and the Beluga AI Framework patterns:

```
pkg/chatmodels/
├── iface/              # Interface definitions (ISP compliant)
│   └── chatmodel.go    # Core chat model interfaces
├── internal/           # Private implementations
│   └── mock/          # Mock implementations for testing
├── providers/         # Provider implementations
│   └── openai/        # OpenAI provider (framework ready)
├── config.go          # Configuration management and validation
├── errors.go          # Custom error types with proper wrapping
├── metrics.go         # OpenTelemetry observability framework
├── chatmodels.go      # Main package API and factory functions
└── README.md          # This documentation
```

### Key Design Principles

- **Interface Segregation**: Small, focused interfaces serving specific purposes
- **Dependency Inversion**: High-level modules don't depend on low-level modules
- **Single Responsibility**: Each package/component has one clear purpose
- **Composition over Inheritance**: Behaviors composed through embedding
- **Clean Architecture**: Clear separation between business logic and infrastructure

## Core Interfaces

### ChatModel Interface
```go
type ChatModel interface {
    MessageGenerator
    StreamMessageHandler
    ModelInfoProvider
    HealthChecker
    core.Runnable
}
```

### Focused Interfaces

- **`MessageGenerator`**: Focused interface for message generation only
- **`StreamMessageHandler`**: Interface for streaming message responses
- **`ModelInfoProvider`**: Interface for providing model metadata
- **`HealthChecker`**: Interface for health monitoring
- **`ChatModelFactory`**: Factory pattern for creating chat model instances

### Key Interfaces

- **`ChatModel`**: Core chat model interface combining all capabilities
- **`MessageGenerator`**: Single responsibility for message generation
- **`StreamMessageHandler`**: Handles streaming responses
- **`ModelInfoProvider`**: Provides model metadata
- **`HealthChecker`**: Health monitoring capabilities
- **`ChatModelFactory`**: Factory pattern for chat model creation

## Quick Start

### Creating a Chat Model

```go
package main

import (
    "context"
    "log"

    "github.com/lookatitude/beluga-ai/pkg/chatmodels"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

func main() {
    // Create a chat model with default configuration
    config := chatmodels.DefaultConfig()
    model, err := chatmodels.NewChatModel("gpt-4", config)
    if err != nil {
        log.Fatal(err)
    }

    // Generate messages
    ctx := context.Background()
    messages := []schema.Message{
        {Role: "user", Content: "Hello, how are you?"},
    }

    response, err := model.GenerateMessages(ctx, messages)
    if err != nil {
        log.Printf("Generation failed: %v", err)
        return
    }

    for _, msg := range response {
        log.Printf("Response: %s", msg.Content)
    }
}
```

### Creating an OpenAI Chat Model

```go
// Create OpenAI-specific model
model, err := chatmodels.NewOpenAIChatModel("gpt-4", "your-api-key",
    chatmodels.WithTemperature(0.7),
    chatmodels.WithMaxTokens(2000),
)
if err != nil {
    log.Fatal(err)
}
```

### Using Streaming

```go
// Create streaming response
stream, err := model.StreamMessages(ctx, messages)
if err != nil {
    log.Fatal(err)
}

// Process streaming chunks
for msg := range stream {
    log.Printf("Chunk: %s", msg.Content)

    // Check if this is the final chunk
    if finished, ok := msg.Metadata["finished"].(bool); ok && finished {
        log.Println("Streaming completed")
        break
    }
}
```

### Creating a Mock Model for Testing

```go
// Create mock model for testing
model, err := chatmodels.NewMockChatModel("mock-gpt-4",
    chatmodels.WithTemperature(0.5),
)
if err != nil {
    log.Fatal(err)
}

// Mock models return predetermined responses
response, err := model.GenerateMessages(ctx, messages)
```

**Status**: Mock implementation provides basic functionality for testing. Full provider implementations are planned for future updates.

## Configuration

The package supports configuration through functional options. Advanced features are designed to be extensible:

### Basic Configuration
```go
config := chatmodels.DefaultConfig()
config.DefaultTemperature = 0.8
config.DefaultMaxTokens = 2000

model, err := chatmodels.NewChatModel("gpt-4", config)
```

### Functional Options
```go
model, err := chatmodels.NewChatModel("gpt-4", config,
    // Generation parameters
    chatmodels.WithTemperature(0.7),
    chatmodels.WithMaxTokens(1000),
    chatmodels.WithTopP(0.9),
    chatmodels.WithStopSequences([]string{"\n", "END"}),

    // System configuration
    chatmodels.WithSystemPrompt("You are a helpful assistant."),
    chatmodels.WithFunctionCalling(true),

    // Runtime behavior
    chatmodels.WithTimeout(30*time.Second),
    chatmodels.WithMaxRetries(3),

    // Observability
    chatmodels.WithMetrics(true),
    chatmodels.WithTracing(true),
)
```

### Provider-Specific Configuration
```go
config := chatmodels.DefaultConfig()
config.DefaultProvider = "openai"

// Configure OpenAI provider
config.Providers["openai"] = &chatmodels.ProviderConfig{
    APIKey:     "your-api-key",
    BaseURL:    "https://api.openai.com/v1",
    Timeout:    30 * time.Second,
    MaxRetries: 3,
}

model, err := chatmodels.NewChatModel("gpt-4", config)
```

**Note**: Provider-specific configurations allow for fine-tuned control over different chat model providers.

### Provider Registry

The `chatmodels` package uses a global registry pattern for managing providers. Providers register themselves automatically via `init()` functions:

```go
// Providers register themselves automatically
import _ "github.com/lookatitude/beluga-ai/pkg/chatmodels/providers/openai"
import _ "github.com/lookatitude/beluga-ai/pkg/chatmodels/internal/mock"

// Access the registry
registry := chatmodels.GetRegistry()

// Check if a provider is registered
if registry.IsRegistered("openai") {
    // Provider is available
}

// List all registered providers
providers := registry.ListProviders()
fmt.Printf("Available providers: %v\n", providers)

// Create a provider using the registry
config := chatmodels.DefaultConfig()
config.DefaultProvider = "openai"
options := &chatmodels.iface.Options{...}
model, err := registry.CreateProvider("gpt-4", config, options)
```

**Note**: The `NewChatModel` function automatically uses the registry when creating providers. The registry pattern allows for easy extension with new providers without modifying core code.

## Error Handling

The package provides comprehensive error handling with custom error types:

```go
model, err := chatmodels.NewChatModel("invalid-model", config)
if err != nil {
    var chatErr *chatmodels.ChatModelError
    if errors.As(err, &chatErr) {
        switch chatErr.Code {
        case chatmodels.ErrCodeProviderNotSupported:
            log.Printf("Unsupported provider: %v", chatErr.Err)
        case chatmodels.ErrCodeModelNotFound:
            log.Printf("Model not found: %v", chatErr.Err)
        case chatmodels.ErrCodeAuthentication:
            if chatmodels.IsAuthenticationError(chatErr) {
                log.Printf("Authentication failed: %v", chatErr.Err)
            }
        }
    }
}
```

### Error Types
- `ChatModelError`: General chat model operation errors
- `ValidationError`: Configuration validation errors
- `ProviderError`: Provider-specific errors
- `GenerationError`: Message generation errors
- `StreamingError`: Streaming operation errors

### Error Checking Functions
```go
// Check for specific error types
if chatmodels.IsRetryable(err) {
    // Implement retry logic
}

if chatmodels.IsValidationError(err) {
    // Handle configuration errors
}

if chatmodels.IsAuthenticationError(err) {
    // Handle authentication failures
}
```

## Supported Providers and Models

### OpenAI Provider
- **Models**: `gpt-4`, `gpt-4-turbo`, `gpt-4o`, `gpt-4o-mini`, `gpt-3.5-turbo`
- **Features**: Text generation, streaming, function calling
- **Status**: Framework implemented, API integration placeholder

### Mock Provider
- **Models**: `mock-gpt-4`, `mock-claude`, `mock-general`
- **Features**: Predefined responses, streaming simulation
- **Status**: Fully implemented for testing

### Getting Available Options
```go
// List supported providers
providers := chatmodels.GetSupportedProviders()
fmt.Printf("Providers: %v\n", providers)

// List models for a provider
models := chatmodels.GetSupportedModels("openai")
fmt.Printf("OpenAI models: %v\n", models)
```

## Health Monitoring

```go
// Check model health
health := chatmodels.HealthCheck(model)
status := health["status"].(string)

if status == "error" {
    log.Warn("Model is in error state")
}

// Get model information
info := chatmodels.GetModelInfo(model)
fmt.Printf("Model: %s, Provider: %s, Max Tokens: %d\n",
    info.Name, info.Provider, info.MaxTokens)
```

## Observability

### Metrics Initialization

The package uses a standardized metrics initialization pattern with `InitMetrics()` and `GetMetrics()`:

```go
import (
    "go.opentelemetry.io/otel/metric"
    "github.com/lookatitude/beluga-ai/pkg/chatmodels"
)

// Initialize metrics once at application startup
meter := otel.Meter("beluga-chatmodels")
chatmodels.InitMetrics(meter)

// Get the global metrics instance
metrics := chatmodels.GetMetrics()
if metrics != nil {
    // Use metrics for observability
}
```

### Metrics
The package includes a metrics framework using OpenTelemetry interfaces:

- **Message Generation Metrics**: Track generation requests, tokens, latency
- **Streaming Performance**: Monitor streaming operations and throughput
- **Error Rates**: Track error rates by provider and error type
- **Token Usage**: Monitor token consumption across providers

### Tracing
Distributed tracing support for end-to-end observability:

```go
// Tracing is automatically integrated when metrics are initialized
ctx, span := metrics.StartProviderSpan(ctx, "openai", "generate")
defer span.End()

// Spans are automatically created for:
// - Message generation operations
// - Streaming operations
// - Provider interactions
```

**Status**: Observability framework is fully implemented with OpenTelemetry integration.

## Testing

The package includes unit tests and mock implementations:

```go
func TestChatModel_GenerateMessages(t *testing.T) {
    model, err := chatmodels.NewMockChatModel("test-model")
    if err != nil {
        t.Fatalf("Failed to create mock model: %v", err)
    }

    messages := []schema.Message{
        {Role: "user", Content: "Test message"},
    }

    response, err := model.GenerateMessages(context.Background(), messages)
    if err != nil {
        t.Fatalf("Generation failed: %v", err)
    }

    if len(response) == 0 {
        t.Fatal("Expected at least one response message")
    }
}
```

## Best Practices

### 1. Error Handling
```go
// Always check for specific error types
response, err := model.GenerateMessages(ctx, messages)
if err != nil {
    if chatmodels.IsRetryable(err) {
        // Implement retry logic with exponential backoff
        time.Sleep(backoffDuration)
        return retryGeneration()
    } else {
        // Handle permanent errors
        return fmt.Errorf("generation failed permanently: %w", err)
    }
}
```

### 2. Configuration Validation
```go
// Validate configuration before use
config := chatmodels.DefaultConfig()
if err := chatmodels.ValidateConfig(config); err != nil {
    log.Fatal("Invalid configuration:", err)
}
```

### 3. Resource Management
```go
// Always properly handle context cancellation
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

response, err := model.GenerateMessages(ctx, messages)
```

### 4. Health Monitoring
```go
// Regularly check model health
health := model.CheckHealth()
if health["status"] == "error" {
    // Implement fallback logic or model switching
    fallbackModel, err := chatmodels.NewChatModel("fallback-model", config)
    if err != nil {
        log.Printf("Failed to create fallback model: %v", err)
    }
    // Use fallback model
}
```

### 5. Streaming Best Practices
```go
// Handle streaming with proper error handling
stream, err := model.StreamMessages(ctx, messages)
if err != nil {
    return fmt.Errorf("streaming failed: %w", err)
}

var fullResponse strings.Builder
for msg := range stream {
    fullResponse.WriteString(msg.Content)

    // Handle streaming errors
    if err := msg.Metadata["error"]; err != nil {
        return fmt.Errorf("streaming error: %v", err)
    }
}
```

## Implementation Status

### ✅ Completed Features
- **Architecture**: SOLID principles with clean separation of concerns
- **Core Interfaces**: Comprehensive interface definitions (ISP compliant)
- **Configuration**: Basic configuration management and validation
- **Error Handling**: Custom error types with proper error wrapping
- **Mock Implementation**: Full mock provider for testing
- **Factory Pattern**: Clean factory functions for model creation
- **Health Monitoring**: Model health checking APIs
- **Testing**: Unit tests for core functionality

### 🚧 In Development / Placeholder
- **OpenAI Integration**: Full OpenAI API implementation (framework ready)
- **Additional Providers**: Anthropic, local model support
- **Metrics & Tracing**: OpenTelemetry integration (framework ready)
- **Advanced Streaming**: Real-time streaming with backpressure
- **Connection Pooling**: Efficient connection management

### 📋 Roadmap
1. **Complete OpenAI Integration**: Full API implementation with error handling
2. **Add Anthropic Provider**: Support for Claude models
3. **OpenTelemetry Integration**: Full metrics and tracing support
4. **Advanced Streaming**: WebSocket support and backpressure handling
5. **Local Model Support**: Integration with local LLM servers
6. **Caching Layer**: Response caching for improved performance
7. **Rate Limiting**: Built-in rate limiting and request queuing
8. **Batch Processing**: Support for batch message generation

## Contributing

When adding new providers:

1. **Create provider package** in `providers/` directory following existing patterns
2. **Follow SOLID principles** and interface segregation
3. **Add comprehensive tests** with mocks and edge cases
4. **Update documentation** with examples and usage patterns
5. **Maintain backward compatibility** with existing interfaces

### Development Guidelines
- Use functional options for configuration
- Implement proper error handling with custom error types
- Add health checking for all provider implementations
- Write tests that cover both success and failure scenarios
- Update this README when adding new features

## Migration Guide

### From Old ChatModels Interface
If migrating from the old single-file implementation:

1. **Update imports**: Change to use the new package structure
2. **Update interface usage**: Use focused interfaces where appropriate
3. **Update error handling**: Use new error types and checking functions
4. **Update configuration**: Use functional options pattern

### Example Migration
```go
// Old usage
chatModel := &oldChatModel{...}
response, err := chatModel.GenerateMessages(messages)

// New usage
config := chatmodels.DefaultConfig()
model, err := chatmodels.NewChatModel("gpt-4", config)
if err != nil {
    return err
}
response, err := model.GenerateMessages(ctx, messages)
```

## License

This package is part of the Beluga AI Framework and follows the same license terms.
