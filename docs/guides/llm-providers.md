# LLM Provider Integration Guide

> **Learn how to add custom LLM providers to Beluga AI, extending the framework's capabilities to support any language model API.**

## Introduction

One of Beluga AI's most powerful features is its extensible provider system. Whether you're integrating a new LLM service, connecting to a self-hosted model, or creating a mock for testing, the process follows the same elegant pattern: implement the interface, register it, and you're ready to go.

In this guide, you'll learn:

- How the provider registry pattern works and why it's designed this way
- How to implement the `ChatModel` interface for your custom provider
- How to add proper OTEL instrumentation for observability
- How to register your provider with the global registry
- How to test your provider thoroughly

By the end, you'll have a fully functional custom LLM provider that integrates seamlessly with the rest of the Beluga AI framework.

## Prerequisites

Before diving in, make sure you have:

| Requirement | Why You Need It |
|-------------|-----------------|
| **Go 1.24+** | Required for the Beluga AI framework |
| **Beluga AI Framework** | `go get github.com/lookatitude/beluga-ai` |
| **Understanding of Go interfaces** | We'll implement the `ChatModel` interface |
| **API credentials** (for real providers) | To test against actual LLM APIs |

If you're new to Go interfaces, we recommend reviewing the [Go documentation on interfaces](https://go.dev/tour/methods/9) first.

## Concepts

Before we write any code, let's understand the key concepts that make provider extensibility work.

### The Provider Registry Pattern

At the heart of Beluga AI's extensibility is the **Provider Registry**. Think of it as a phonebook for LLM providers - when you need an LLM, you look it up by name, and the registry creates the right instance for you.

```
┌─────────────────────────────────────────────────────────────┐
│                     Global Registry                          │
├─────────────────────────────────────────────────────────────┤
│  "openai"     → OpenAI Provider Factory                     │
│  "anthropic"  → Anthropic Provider Factory                  │
│  "ollama"     → Ollama Provider Factory                     │
│  "custom"     → Your Custom Provider Factory  ← You add this│
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
              registry.GetProvider("custom", config)
                              │
                              ▼
                 ┌─────────────────────────┐
                 │  Custom ChatModel       │
                 │  Instance (ready to use)│
                 └─────────────────────────┘
```

**Why this pattern?**

1. **Decoupling**: Your application code doesn't need to know which provider it's using
2. **Configuration-driven**: Switch providers by changing config, not code
3. **Testing**: Easily swap in mocks for testing
4. **Lazy initialization**: Providers are created only when needed

### The ChatModel Interface

All LLM providers in Beluga AI implement the `ChatModel` interface. This interface defines the contract that every provider must fulfill:

```go
type ChatModel interface {
    core.Runnable  // Provides Invoke, Batch, Stream methods
    LLM            // Provides GetModelName, GetProviderName

    // Generate takes messages and returns an AI response
    Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error)

    // StreamChat returns a channel of response chunks for streaming
    StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan AIMessageChunk, error)

    // BindTools attaches tools that the model can call
    BindTools(toolsToBind []tools.Tool) ChatModel

    // GetModelName returns the model identifier
    GetModelName() string

    // CheckHealth returns health status information
    CheckHealth() map[string]any
}
```

**Interface Segregation**: Notice how `ChatModel` embeds smaller interfaces (`core.Runnable`, `LLM`). This follows the Interface Segregation Principle - you can use just the parts you need.

### Factory Functions

Instead of constructors, we use **factory functions**. A factory function takes configuration and returns a provider instance:

```go
func NewCustomProviderFactory() func(*llms.Config) (iface.ChatModel, error) {
    return func(config *llms.Config) (iface.ChatModel, error) {
        return NewCustomProvider(config)
    }
}
```

**Why factories?**

- They allow the registry to create instances lazily
- Configuration is validated at creation time
- Different configurations can create different provider instances

## Step-by-Step Tutorial

Now let's build a custom LLM provider from scratch. We'll create a provider that integrates with a hypothetical "CustomLLM" API.

### Step 1: Set Up Your Provider Package

Create a new directory for your provider:

```bash
mkdir -p pkg/llms/providers/custom
```

Create the main provider file:

```go
// pkg/llms/providers/custom/provider.go
package custom

import (
    "context"
    "fmt"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/agents/tools"
    "github.com/lookatitude/beluga-ai/pkg/core"
    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/llms/iface"
    "github.com/lookatitude/beluga-ai/pkg/llms/internal/common"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

// ProviderName identifies this provider in the registry
const ProviderName = "custom"

// DefaultModel is used when no model is specified
const DefaultModel = "custom-v1"
```

**What you'll see**: A clean package structure that mirrors the built-in providers.

**Why this works**: By following the same structure as OpenAI and Anthropic providers, your custom provider integrates naturally with the framework.

### Step 2: Define the Provider Struct

```go
// CustomProvider implements the ChatModel interface for CustomLLM.
// We store configuration, metrics, tracing, and any bound tools.
type CustomProvider struct {
    config      *llms.Config
    metrics     llms.MetricsRecorder
    tracing     *common.TracingHelper
    retryConfig *common.RetryConfig
    modelName   string
    tools       []tools.Tool
    
    // Add your API client here
    // client *customllm.Client
}
```

**Key design decisions**:

- **Composition over inheritance**: We embed helper components rather than inheriting
- **Immutable after creation**: Configuration is set once during construction
- **Tools are copied, not shared**: `BindTools` returns a new instance to avoid mutation

### Step 3: Implement the Constructor

```go
// NewCustomProvider creates a new CustomLLM provider instance.
// This validates configuration and sets up the provider for use.
//
// Parameters:
//   - config: LLM configuration with API key, model name, etc.
//
// Returns:
//   - *CustomProvider: Ready-to-use provider instance
//   - error: Configuration validation errors
func NewCustomProvider(config *llms.Config) (*CustomProvider, error) {
    // Validate the configuration first - fail fast if something's wrong
    if err := llms.ValidateProviderConfig(context.Background(), config); err != nil {
        return nil, fmt.Errorf("invalid Custom configuration: %w", err)
    }

    // Use default model if none specified
    modelName := config.ModelName
    if modelName == "" {
        modelName = DefaultModel
    }

    // Initialize your API client here
    // client, err := customllm.NewClient(config.APIKey, config.BaseURL)
    // if err != nil {
    //     return nil, fmt.Errorf("failed to create Custom client: %w", err)
    // }

    provider := &CustomProvider{
        config:    config,
        modelName: modelName,
        metrics:   llms.GetMetrics(),  // Get global OTEL metrics
        tracing:   common.NewTracingHelper(),
        retryConfig: &common.RetryConfig{
            MaxRetries: config.MaxRetries,
            Delay:      config.RetryDelay,
            Backoff:    config.RetryBackoff,
        },
    }

    return provider, nil
}
```

**What you'll see**: A clean validation and initialization flow.

**Common pitfall**: Don't forget to validate configuration before using it. Fail fast with clear error messages.

### Step 4: Implement the Generate Method

This is the core method - it takes messages and returns an AI response:

```go
// Generate sends messages to the LLM and returns the response.
// It includes OTEL tracing, metrics, and retry logic.
func (c *CustomProvider) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
    // Start tracing span - this creates visibility into each request
    ctx = c.tracing.StartOperation(ctx, "custom.generate", ProviderName, c.modelName)

    // Calculate input size for metrics
    inputSize := 0
    for _, m := range messages {
        inputSize += len(m.GetContent())
    }
    c.tracing.AddSpanAttributes(ctx, map[string]any{"input_size": inputSize})

    start := time.Now()

    // Track active requests for load monitoring
    c.metrics.IncrementActiveRequests(ctx, ProviderName, c.modelName)
    defer c.metrics.DecrementActiveRequests(ctx, ProviderName, c.modelName)

    // Build call options from defaults and overrides
    callOpts := c.buildCallOptions(options...)

    // Execute with retry logic for resilience
    var result schema.Message
    var err error

    retryErr := common.RetryWithBackoff(ctx, c.retryConfig, "custom.generate", func() error {
        result, err = c.generateInternal(ctx, messages, callOpts)
        return err
    })

    if retryErr != nil {
        duration := time.Since(start)
        c.metrics.RecordError(ctx, ProviderName, c.modelName, llms.GetLLMErrorCode(retryErr), duration)
        c.tracing.RecordError(ctx, retryErr)
        return nil, retryErr
    }

    // Record success metrics
    duration := time.Since(start)
    c.metrics.RecordRequest(ctx, ProviderName, c.modelName, duration)

    return result, nil
}

// generateInternal performs the actual API call
func (c *CustomProvider) generateInternal(ctx context.Context, messages []schema.Message, opts *llms.CallOptions) (schema.Message, error) {
    // Convert messages to your API's format
    apiMessages := c.convertMessages(messages)
    
    // Make the API call
    // response, err := c.client.Chat(ctx, apiMessages, opts)
    // if err != nil {
    //     return nil, c.handleAPIError("generate", err)
    // }
    
    // For demonstration, return a mock response
    response := "This is a response from CustomLLM"
    
    return schema.NewAIMessage(response), nil
}
```

**Why separate methods?** The retry wrapper calls `generateInternal`, keeping the retry logic separate from the business logic. This makes testing easier.

### Step 5: Implement Streaming

Streaming allows responses to arrive in chunks, enabling real-time display:

```go
// StreamChat streams response chunks for real-time display.
func (c *CustomProvider) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan iface.AIMessageChunk, error) {
    // Start tracing
    ctx = c.tracing.StartOperation(ctx, "custom.stream", ProviderName, c.modelName)

    callOpts := c.buildCallOptions(options...)

    // Create the output channel
    outputChan := make(chan iface.AIMessageChunk)

    go func() {
        defer close(outputChan)

        // Your streaming API call here
        // stream, err := c.client.StreamChat(ctx, messages)
        // if err != nil {
        //     outputChan <- iface.AIMessageChunk{Err: err}
        //     return
        // }
        
        // Simulate streaming for demonstration
        words := []string{"Hello", " from", " CustomLLM", "!"}
        for _, word := range words {
            select {
            case <-ctx.Done():
                return
            case outputChan <- iface.AIMessageChunk{Content: word}:
            }
        }
    }()

    return outputChan, nil
}
```

**Important**: Always check `ctx.Done()` in streaming loops to support cancellation.

### Step 6: Implement Tool Binding

Tool binding allows the LLM to call functions:

```go
// BindTools returns a new provider instance with tools attached.
// We return a copy to avoid mutating the original provider.
func (c *CustomProvider) BindTools(toolsToBind []tools.Tool) iface.ChatModel {
    // Create a shallow copy
    newProvider := *c
    
    // Deep copy the tools
    newProvider.tools = make([]tools.Tool, len(toolsToBind))
    copy(newProvider.tools, toolsToBind)
    
    return &newProvider
}
```

**Why copy?** This ensures thread safety and allows the same provider to be used with different tool sets concurrently.

### Step 7: Implement Remaining Interface Methods

```go
// GetModelName returns the model identifier
func (c *CustomProvider) GetModelName() string {
    return c.modelName
}

// GetProviderName returns "custom"
func (c *CustomProvider) GetProviderName() string {
    return ProviderName
}

// Invoke implements the Runnable interface
func (c *CustomProvider) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
    messages, err := llms.EnsureMessages(input)
    if err != nil {
        return nil, err
    }
    return c.Generate(ctx, messages, options...)
}

// Batch implements the Runnable interface for batch processing
func (c *CustomProvider) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
    results := make([]any, len(inputs))
    
    // Use semaphore for concurrency control
    sem := make(chan struct{}, c.config.MaxConcurrentBatches)
    errChan := make(chan error, len(inputs))

    for i, input := range inputs {
        sem <- struct{}{}
        
        go func(index int, currentInput any) {
            defer func() { <-sem }()
            
            result, err := c.Invoke(ctx, currentInput, options...)
            results[index] = result
            if err != nil {
                errChan <- err
            }
        }(i, input)
    }

    // Wait for completion
    for i := 0; i < c.config.MaxConcurrentBatches; i++ {
        sem <- struct{}{}
    }
    close(errChan)

    // Collect errors
    var combinedErr error
    for err := range errChan {
        if combinedErr == nil {
            combinedErr = err
        }
    }

    return results, combinedErr
}

// Stream implements the Runnable interface
func (c *CustomProvider) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
    messages, err := llms.EnsureMessages(input)
    if err != nil {
        return nil, err
    }

    chunkChan, err := c.StreamChat(ctx, messages, options...)
    if err != nil {
        return nil, err
    }

    outputChan := make(chan any)
    go func() {
        defer close(outputChan)
        for chunk := range chunkChan {
            select {
            case outputChan <- chunk:
            case <-ctx.Done():
                return
            }
        }
    }()

    return outputChan, nil
}

// CheckHealth returns health status information
func (c *CustomProvider) CheckHealth() map[string]any {
    return map[string]any{
        "state":       "healthy",
        "provider":    ProviderName,
        "model":       c.modelName,
        "timestamp":   time.Now().Unix(),
        "api_key_set": c.config.APIKey != "",
        "tools_count": len(c.tools),
    }
}
```

### Step 8: Create the Factory and Register

Create an init file that registers your provider:

```go
// pkg/llms/providers/custom/init.go
package custom

import "github.com/lookatitude/beluga-ai/pkg/llms"

// NewCustomProviderFactory returns a factory function for the registry
func NewCustomProviderFactory() func(*llms.Config) (iface.ChatModel, error) {
    return func(config *llms.Config) (iface.ChatModel, error) {
        return NewCustomProvider(config)
    }
}

func init() {
    // Register with the global registry on package import
    llms.GetRegistry().Register(ProviderName, NewCustomProviderFactory())
}
```

**How it works**: When your package is imported (even with `import _ "path/to/custom"`), the `init()` function runs automatically, registering your provider.

### Step 9: Use Your Provider

Now you can use your provider like any built-in provider:

```go
package main

import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/schema"
    _ "github.com/lookatitude/beluga-ai/pkg/llms/providers/custom" // Register provider
)

func main() {
    ctx := context.Background()

    // Create config
    config := llms.NewConfig(
        llms.WithProvider("custom"),
        llms.WithModelName("custom-v1"),
        llms.WithAPIKey("your-api-key"),
    )

    // Get provider from registry
    provider, err := llms.NewProvider(ctx, "custom", config)
    if err != nil {
        panic(err)
    }

    // Use it!
    messages := []schema.Message{
        schema.NewHumanMessage("Hello, CustomLLM!"),
    }

    response, err := provider.Generate(ctx, messages)
    if err != nil {
        panic(err)
    }

    fmt.Println(response.GetContent())
}
```

## Code Examples

For a complete, production-ready example, see:

- [Custom LLM Provider Example](/examples/llms/custom_provider/custom_llm_provider.go)
- [Test Suite](/examples/llms/custom_provider/custom_llm_provider_test.go)

## Testing

Testing your provider thoroughly is crucial. Here's how to approach it:

### Interface Compliance Tests

Verify your provider implements all required methods:

```go
func TestCustomProviderImplementsChatModel(t *testing.T) {
    config := &llms.Config{
        Provider:  "custom",
        ModelName: "test-model",
        APIKey:    "test-key",
    }
    
    provider, err := NewCustomProvider(config)
    require.NoError(t, err)
    
    // Compile-time check that we implement ChatModel
    var _ iface.ChatModel = provider
}
```

### Registration Tests

Verify the provider registers correctly:

```go
func TestCustomProviderRegistration(t *testing.T) {
    // Import triggers init()
    registry := llms.GetRegistry()
    
    assert.True(t, registry.IsRegistered("custom"))
    assert.Contains(t, registry.ListProviders(), "custom")
}
```

### Table-Driven Generation Tests

```go
func TestCustomProviderGenerate(t *testing.T) {
    tests := []struct {
        name        string
        messages    []schema.Message
        wantErr     bool
        errContains string
    }{
        {
            name: "simple message",
            messages: []schema.Message{
                schema.NewHumanMessage("Hello"),
            },
            wantErr: false,
        },
        {
            name: "system and human messages",
            messages: []schema.Message{
                schema.NewSystemMessage("You are helpful"),
                schema.NewHumanMessage("Hi"),
            },
            wantErr: false,
        },
        {
            name:        "empty messages",
            messages:    []schema.Message{},
            wantErr:     true,
            errContains: "no messages",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            provider := setupTestProvider(t)
            
            result, err := provider.Generate(context.Background(), tt.messages)
            
            if tt.wantErr {
                require.Error(t, err)
                assert.Contains(t, err.Error(), tt.errContains)
            } else {
                require.NoError(t, err)
                assert.NotNil(t, result)
            }
        })
    }
}
```

### OTEL Metrics Verification

```go
func TestCustomProviderMetrics(t *testing.T) {
    // Set up test meter
    reader := sdkmetric.NewManualReader()
    meterProvider := sdkmetric.NewMeterProvider(
        sdkmetric.WithReader(reader),
    )
    meter := meterProvider.Meter("test")
    llms.InitMetrics(meter)
    
    provider := setupTestProvider(t)
    
    // Make a request
    _, _ = provider.Generate(context.Background(), []schema.Message{
        schema.NewHumanMessage("Test"),
    })
    
    // Verify metrics were recorded
    metrics := &metricdata.ResourceMetrics{}
    err := reader.Collect(context.Background(), metrics)
    require.NoError(t, err)
    
    // Check for expected metrics
    found := false
    for _, sm := range metrics.ScopeMetrics {
        for _, m := range sm.Metrics {
            if m.Name == "beluga.llm.request_duration_seconds" {
                found = true
            }
        }
    }
    assert.True(t, found, "Expected metrics not found")
}
```

## Best Practices

### Error Handling

Create provider-specific error codes for better debugging:

```go
const (
    ErrCodeInvalidAPIKey  = "custom_invalid_api_key"
    ErrCodeRateLimit      = "custom_rate_limit"
    ErrCodeModelNotFound  = "custom_model_not_found"
)

func (c *CustomProvider) handleAPIError(operation string, err error) error {
    errStr := err.Error()
    
    switch {
    case strings.Contains(errStr, "rate limit"):
        return llms.NewLLMError(operation, ErrCodeRateLimit, err)
    case strings.Contains(errStr, "authentication"):
        return llms.NewLLMError(operation, ErrCodeInvalidAPIKey, err)
    default:
        return llms.WrapError(operation, err)
    }
}
```

### Configuration Management

Use functional options for optional configuration:

```go
// Provider-specific options
func WithCustomEndpoint(endpoint string) llms.ConfigOption {
    return llms.WithProviderSpecific("custom_endpoint", endpoint)
}

func WithCustomTimeout(timeout time.Duration) llms.ConfigOption {
    return llms.WithProviderSpecific("custom_timeout", timeout)
}
```

### Rate Limiting

Respect API rate limits:

```go
type CustomProvider struct {
    // ... other fields
    rateLimiter *rate.Limiter
}

func (c *CustomProvider) generateInternal(ctx context.Context, ...) (schema.Message, error) {
    // Wait for rate limiter
    if err := c.rateLimiter.Wait(ctx); err != nil {
        return nil, llms.NewLLMError("generate", "rate_limit_wait", err)
    }
    
    // Proceed with API call
    // ...
}
```

## Troubleshooting

### Q: My provider isn't appearing in the registry

**A:** Make sure you:
1. Import the package somewhere (even with `_`)
2. The `init()` function is calling `llms.GetRegistry().Register()`
3. There are no compilation errors in your package

### Q: OTEL metrics aren't being recorded

**A:** Verify that:
1. `llms.InitMetrics(meter)` is called during application startup
2. Your provider calls `c.metrics.RecordRequest()` and `c.metrics.RecordError()`
3. The OTEL exporter is configured correctly

### Q: Streaming stops unexpectedly

**A:** Check:
1. You're checking `ctx.Done()` in your streaming goroutine
2. The channel is being closed properly with `defer close(outputChan)`
3. API errors are being sent through the channel: `outputChan <- iface.AIMessageChunk{Err: err}`

### Q: Tool calls aren't working

**A:** Ensure:
1. `BindTools()` returns a new instance (don't mutate)
2. Your API request includes the tools in the correct format
3. Tool responses are being parsed correctly from the API response

## Related Resources

- **[Streaming LLM with Tool Calls Guide](./llm-streaming-tool-calls.md)**: Learn how to implement streaming with tool calling
- **[Extensibility Guide](./extensibility.md)**: Broader patterns for extending Beluga AI
- **[Observability Tracing Guide](./observability-tracing.md)**: Deep dive into OTEL integration
- **[LLM Error Handling Cookbook](../cookbook/llm-error-handling.md)**: Recipes for handling LLM errors gracefully
- **[Custom LLM Provider Example](/examples/llms/custom_provider/)**: Complete, runnable example code
