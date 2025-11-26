# LLM Package Usage Examples

This document demonstrates how to use the refactored LLM package following the Beluga AI Framework design patterns.

## Basic Usage

### Creating a Provider with Factory Pattern

```go
package main

import (
    "context"
    "log"

    "github.com/lookatitude/beluga-ai/pkg/llms"
)

func main() {
    // Initialize metrics (optional)
    llms.InitMetrics(nil) // Pass your OpenTelemetry meter if available

    // Create configuration
    config := llms.NewConfig(
        llms.WithProvider("anthropic"),
        llms.WithModelName("claude-3-sonnet-20240229"),
        llms.WithAPIKey("your-anthropic-api-key"),
        llms.WithTemperature(0.7),
        llms.WithMaxTokens(1024),
    )

    // Validate configuration
    if err := llms.ValidateProviderConfig(context.Background(), config); err != nil {
        log.Fatal("Invalid configuration:", err)
    }

    // Create provider using factory
    factory := llms.NewFactory()

    // Note: You would register provider factories like this:
    // factory.RegisterProviderFactory("anthropic", anthropic.NewAnthropicProviderFactory())

    // For now, create providers directly
    // provider, err := factory.CreateProvider("anthropic", config)
    // if err != nil {
    //     log.Fatal(err)
    // }
}
```

### Using the ChatModel Interface

```go
// Create messages
messages := []schema.Message{
    schema.NewSystemMessage("You are a helpful assistant."),
    schema.NewHumanMessage("What is the capital of France?"),
}

// Generate response
response, err := provider.Generate(context.Background(), messages)
if err != nil {
    log.Printf("Generation failed: %v", err)
    return
}

log.Printf("Response: %s", response.GetContent())
```

### Streaming Responses

```go
// Stream chat responses
streamChan, err := provider.StreamChat(context.Background(), messages)
if err != nil {
    log.Fatal(err)
}

for chunk := range streamChan {
    if chunk.Err != nil {
        log.Printf("Stream error: %v", chunk.Err)
        break
    }
    log.Printf("Chunk: %s", chunk.Content)

    // Handle tool calls
    for _, toolCall := range chunk.ToolCallChunks {
        log.Printf("Tool call: %s", toolCall.Name)
    }
}
```

### Tool Calling

```go
// Bind tools to the model
calculatorTool := tools.NewCalculatorTool()
modelWithTools := provider.BindTools([]tools.Tool{calculatorTool})

// Use tool calling
messages := []schema.Message{
    schema.NewHumanMessage("Calculate 15 * 23"),
}

response, err := modelWithTools.Generate(context.Background(), messages)
if err != nil {
    log.Fatal(err)
}

// Handle tool calls in response
if aiMsg, ok := response.(*schema.AIMessage); ok {
    for _, toolCall := range aiMsg.ToolCalls {
        log.Printf("Tool call: %s with args %s", toolCall.Name, toolCall.Arguments)
    }
}
```

### Batch Processing

```go
// Prepare multiple inputs
inputs := []any{
    []schema.Message{schema.NewHumanMessage("Hello!")},
    []schema.Message{schema.NewHumanMessage("How are you?")},
    []schema.Message{schema.NewHumanMessage("What's the weather?")},
}

// Process batch with concurrency control
results, err := provider.Batch(context.Background(), inputs)
if err != nil {
    log.Printf("Batch processing failed: %v", err)
    return
}

for i, result := range results {
    if msg, ok := result.(schema.Message); ok {
        log.Printf("Response %d: %s", i+1, msg.GetContent())
    }
}
```

## Configuration Options

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
```

## Error Handling

```go
response, err := provider.Generate(ctx, messages)
if err != nil {
    // Check for specific error types
    if llmErr := llms.GetLLMError(err); llmErr != nil {
        switch llmErr.Code {
        case llms.ErrCodeRateLimit:
            // Implement backoff and retry
            time.Sleep(time.Minute)
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

## Testing with Mock Provider

```go
// Create mock provider for testing
mockConfig := llms.NewConfig(
    llms.WithProvider("mock"),
    llms.WithModelName("mock-model"),
    llms.WithProviderSpecific("responses", []string{
        "Mock response 1",
        "Mock response 2",
        "Mock response 3",
    }),
)

// Use in tests
messages := []schema.Message{schema.NewHumanMessage("Test")}
response, err := mockProvider.Generate(context.Background(), messages)
require.NoError(t, err)
assert.Equal(t, "Mock response 1", response.GetContent())
```

## Utility Functions

```go
// Convert various input types to messages
messages, err := llms.EnsureMessages("Hello, world!")
if err != nil {
    log.Fatal(err)
}

// Extract system and human prompts
system, human := llms.GetSystemAndHumanPrompts(messages)
log.Printf("System: %s", system)
log.Printf("Human: %s", human)

// Validate model names
err = llms.ValidateModelName("openai", "gpt-4")
if err != nil {
    log.Printf("Invalid model: %v", err)
}
```

## Factory Pattern Usage

```go
// Create factory
factory := llms.NewFactory()

// Register providers
factory.RegisterProvider("my-anthropic", anthropicProvider)
factory.RegisterProvider("my-openai", openaiProvider)

// Get provider by name
provider, err := factory.GetProvider("my-anthropic")
if err != nil {
    log.Fatal(err)
}

// List all registered providers
providers := factory.ListProviders()
log.Printf("Available providers: %v", providers)
```

## Advanced Configuration

### Retry and Timeout Configuration

```go
config := llms.NewConfig(
    llms.WithProvider("anthropic"),
    llms.WithTimeout(30 * time.Second),
    llms.WithRetryConfig(5, time.Second, 2.0), // 5 retries, 1s delay, 2x backoff
    llms.WithMaxConcurrentBatches(20),
)
```

### Observability Configuration

```go
config := llms.NewConfig(
    llms.WithProvider("openai"),
    llms.WithObservability(true, true, true), // Enable tracing, metrics, logging
)

// Initialize metrics
meter := otel.Meter("beluga-ai-llms")
llms.InitMetrics(meter)
```

### Tool Calling Configuration

```go
config := llms.NewConfig(
    llms.WithProvider("anthropic"),
    llms.WithToolCalling(true),
)

// Use with tool-bound model
modelWithTools := provider.BindTools([]tools.Tool{calculator, webSearch})
response, err := modelWithTools.Generate(ctx, messages)
```

## Best Practices

1. **Always validate configuration** before creating providers
2. **Use proper error handling** with specific error code checks
3. **Implement retry logic** for transient failures
4. **Clean up resources** by properly shutting down providers
5. **Use the factory pattern** for provider management
6. **Enable observability** for production deployments
7. **Use mock providers** for testing and development

## Migration from Old API

### Before (old API)
```go
// Old way
client := anthropic.NewChat("api-key")
response, err := client.Generate(messages)
```

### After (new API)
```go
// New way with configuration and factory
config := llms.NewConfig(
    llms.WithProvider("anthropic"),
    llms.WithAPIKey("api-key"),
    llms.WithModelName("claude-3-sonnet"),
)

factory := llms.NewFactory()
provider, err := factory.CreateProvider("anthropic", config)
if err != nil {
    log.Fatal(err)
}

response, err := provider.Generate(ctx, messages)
```

This refactoring provides a more robust, maintainable, and extensible foundation for LLM interactions in the Beluga AI Framework.
