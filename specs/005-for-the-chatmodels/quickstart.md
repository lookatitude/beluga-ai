# ChatModels Package Framework Compliance - Quickstart Guide

## Overview

This guide demonstrates how to use the enhanced ChatModels package with full framework compliance including global registry pattern, complete OTEL integration, and structured error handling, while preserving the existing runnable interface that developers already know and love.

## Prerequisites

- Beluga AI Framework installed and configured  
- ChatModels package with framework compliance enhancements
- Provider API keys (OpenAI, Anthropic, etc.) for real usage
- Go 1.21+ for development

## Quick Start Examples

### 1. Using the Global Registry (New Compliance Feature)

```go
package main

import (
    "context"
    "log"
    
    "github.com/lookatitude/beluga-ai/pkg/chatmodels"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

func main() {
    ctx := context.Background()
    
    // Register providers globally (typically done at application startup)
    err := chatmodels.RegisterGlobal("openai", chatmodels.NewOpenAICreator())
    if err != nil {
        log.Fatal("Failed to register OpenAI provider:", err)
    }
    
    err = chatmodels.RegisterGlobal("anthropic", chatmodels.NewAnthropicCreator())
    if err != nil {
        log.Fatal("Failed to register Anthropic provider:", err)
    }
    
    // Create provider using registry
    config := chatmodels.Config{
        Provider: "openai",
        Model:    "gpt-4",
        APIKey:   "your-openai-key",
        Temperature: 0.7,
        MaxTokens: 1024,
        EnableMetrics: true,
        EnableTracing: true,
    }
    
    model, err := chatmodels.NewProvider(ctx, "openai", config)
    if err != nil {
        log.Fatal("Failed to create provider:", err)
    }
    
    // Use exactly the same interface as before - no breaking changes!
    messages := []schema.Message{
        schema.NewSystemMessage("You are a helpful assistant."),
        schema.NewHumanMessage("What is the capital of France?"),
    }
    
    response, err := model.GenerateMessages(ctx, messages)
    if err != nil {
        // Enhanced error handling with structured errors
        if chatErr := chatmodels.AsChatModelError(err); chatErr != nil {
            log.Printf("Operation: %s, Code: %s, Provider: %s", 
                chatErr.GetOperation(), chatErr.GetCode(), chatErr.GetProvider())
            
            if chatErr.IsRetryable() {
                log.Printf("Error is retryable, retry after: %v", chatErr.GetRetryAfter())
            }
        }
        return
    }
    
    log.Printf("Response: %s", response[0].GetContent())
}
```

### 2. Provider Discovery and Capabilities (Registry Features)

```go
func discoverProviders() {
    // List all registered providers
    providers := chatmodels.ListProviders()
    log.Printf("Available providers: %v", providers)
    
    // Get detailed provider information
    for _, providerName := range providers {
        metadata, err := chatmodels.GetProviderMetadata(providerName)
        if err != nil {
            continue
        }
        
        log.Printf("Provider: %s", metadata.Name)
        log.Printf("  Description: %s", metadata.Description)
        log.Printf("  Capabilities: %v", metadata.Capabilities)
        log.Printf("  Supported Models: %v", metadata.SupportedModels)
        log.Printf("  Supports Streaming: %t", metadata.SupportsStreaming)
        log.Printf("  Supports Tool Calls: %t", metadata.SupportsToolCalls)
    }
    
    // Find providers with specific capabilities
    streamingProviders, err := chatmodels.GetProvidersWithCapability("streaming")
    if err == nil {
        log.Printf("Providers supporting streaming: %v", streamingProviders)
    }
    
    // Find providers supporting a specific model
    gpt4Providers, err := chatmodels.GetProvidersForModel("gpt-4")
    if err == nil {
        log.Printf("Providers supporting GPT-4: %v", gpt4Providers)
    }
}
```

### 3. Advanced OTEL Observability (New Feature)

```go
func demonstrateObservability() {
    ctx := context.Background()
    
    // Initialize OTEL (typically done in main)
    meter := otel.Meter("chatmodels-example")
    tracer := otel.Tracer("chatmodels-example")
    
    // Create metrics-enabled configuration
    config := chatmodels.Config{
        Provider: "anthropic",
        Model:    "claude-3-sonnet",
        APIKey:   "your-anthropic-key",
        EnableMetrics: true,
        EnableTracing: true,
        EnableLogging: true,
    }
    
    model, err := chatmodels.NewProvider(ctx, "anthropic", config)
    if err != nil {
        log.Fatal(err)
    }
    
    // All operations are automatically instrumented
    messages := []schema.Message{
        schema.NewHumanMessage("Explain quantum computing"),
    }
    
    // This operation will generate:
    // - Metrics: request count, duration, success/failure
    // - Tracing: distributed trace with spans
    // - Logging: structured logs with context
    response, err := model.GenerateMessages(ctx, messages)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Response generated with full observability: %s", response[0].GetContent())
    
    // Access health metrics
    if metricsProvider, ok := model.(chatmodels.MetricsProvider); ok {
        healthMetrics := metricsProvider.GetHealthMetrics()
        log.Printf("Success Rate: %.2f%%", healthMetrics.SuccessRate*100)
        log.Printf("Average Latency: %v", healthMetrics.AverageLatency)
        log.Printf("Active Operations: %d", healthMetrics.ActiveOperations)
    }
}
```

### 4. Structured Error Handling (Enhanced Feature)

```go
func handleErrors() {
    ctx := context.Background()
    
    // Create provider with invalid configuration to demonstrate error handling
    config := chatmodels.Config{
        Provider: "openai",
        Model:    "invalid-model",
        APIKey:   "invalid-key",
        Timeout:  time.Second * 5,
    }
    
    model, err := chatmodels.NewProvider(ctx, "openai", config)
    if err != nil {
        // Demonstrate structured error handling
        chatErr := chatmodels.AsChatModelError(err)
        if chatErr != nil {
            log.Printf("Error Details:")
            log.Printf("  Operation: %s", chatErr.GetOperation())
            log.Printf("  Code: %s", chatErr.GetCode())
            log.Printf("  Provider: %s", chatErr.GetProvider())
            log.Printf("  Retryable: %t", chatErr.IsRetryable())
            log.Printf("  Timestamp: %v", chatErr.GetTimestamp())
            
            // Handle specific error codes
            switch chatErr.GetCode() {
            case chatmodels.ErrCodeAuthenticationFailed:
                log.Println("Check your API key configuration")
            case chatmodels.ErrCodeModelUnsupported:
                log.Println("Choose a different model from supported list")
            case chatmodels.ErrCodeProviderUnavailable:
                log.Println("Try a different provider or wait and retry")
            }
            
            // Access additional context
            if context := chatErr.GetContext(); len(context) > 0 {
                log.Printf("Additional Context: %+v", context)
            }
        }
        return
    }
    
    // Example of retry logic with structured errors
    messages := []schema.Message{
        schema.NewHumanMessage("Hello"),
    }
    
    maxRetries := 3
    for attempt := 1; attempt <= maxRetries; attempt++ {
        response, err := model.GenerateMessages(ctx, messages)
        if err == nil {
            log.Printf("Success on attempt %d: %s", attempt, response[0].GetContent())
            return
        }
        
        chatErr := chatmodels.AsChatModelError(err)
        if chatErr == nil || !chatErr.IsRetryable() {
            log.Printf("Non-retryable error: %v", err)
            return
        }
        
        if attempt < maxRetries {
            delay := chatErr.GetRetryAfter()
            if delay == 0 {
                delay = time.Duration(attempt) * time.Second // exponential backoff
            }
            log.Printf("Retrying in %v (attempt %d/%d)", delay, attempt, maxRetries)
            time.Sleep(delay)
        }
    }
    
    log.Printf("All retry attempts exhausted")
}
```

### 5. Backward Compatibility (Existing Code Works Unchanged)

```go
func demonstrateBackwardCompatibility() {
    // This is exactly how developers used ChatModels before compliance enhancements
    // All existing code continues to work without any changes!
    
    ctx := context.Background()
    
    // Original factory functions still work (now use registry internally)
    model, err := chatmodels.NewOpenAIChatModel(
        chatmodels.WithModel("gpt-3.5-turbo"),
        chatmodels.WithAPIKey("your-key"),
        chatmodels.WithTemperature(0.8),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // Same interface methods work exactly the same
    messages := []schema.Message{
        schema.NewHumanMessage("Hello, world!"),
    }
    
    // Generate method unchanged
    response, err := model.GenerateMessages(ctx, messages)
    if err != nil {
        log.Fatal(err)
    }
    
    // Streaming unchanged
    streamChan, err := model.StreamMessages(ctx, messages)
    if err != nil {
        log.Fatal(err)
    }
    
    for chunk := range streamChan {
        log.Printf("Streamed: %s", chunk.GetContent())
    }
    
    // Health checking unchanged
    health := model.CheckHealth()
    log.Printf("Model health: %+v", health)
    
    // Model info unchanged  
    if infoProvider, ok := model.(chatmodels.ModelInfoProvider); ok {
        info := infoProvider.GetModelInfo()
        log.Printf("Model: %s, Provider: %s", info.Name, info.Provider)
    }
}
```

### 6. Advanced Registry Usage

```go
func advancedRegistryUsage() {
    ctx := context.Background()
    
    // Register custom provider with enhanced metadata
    metadata := chatmodels.ProviderMetadata{
        Name:        "custom-provider",
        Description: "Custom LLM provider implementation",
        Capabilities: []string{"generation", "streaming", "tool-calls"},
        SupportedModels: []string{"custom-model-v1", "custom-model-v2"},
        RequiredConfig: []string{"api_key", "endpoint"},
        OptionalConfig: []string{"temperature", "max_tokens"},
        SupportsStreaming: true,
        SupportsToolCalls: true,
        DefaultTimeout: time.Second * 30,
        MaxRetries: 3,
    }
    
    creator := func(ctx context.Context, config chatmodels.Config) (chatmodels.ChatModel, error) {
        // Custom provider creation logic
        return &CustomChatModel{config: config}, nil
    }
    
    err := chatmodels.RegisterGlobalWithMetadata("custom", creator, metadata)
    if err != nil {
        log.Fatal("Failed to register custom provider:", err)
    }
    
    // Use registry validation
    config := chatmodels.Config{
        Provider: "custom",
        Model:    "custom-model-v1",
        APIKey:   "custom-key",
        ProviderSpecific: map[string]interface{}{
            "endpoint": "https://api.custom-llm.com/v1",
            "custom_param": "value",
        },
    }
    
    // Configuration is validated against provider requirements
    err = chatmodels.ValidateConfig("custom", config)
    if err != nil {
        log.Printf("Configuration validation failed: %v", err)
        return
    }
    
    // Create provider with validated configuration
    model, err := chatmodels.NewProvider(ctx, "custom", config)
    if err != nil {
        log.Fatal("Failed to create custom provider:", err)
    }
    
    // Use custom provider with same interface
    messages := []schema.Message{
        schema.NewHumanMessage("Test custom provider"),
    }
    
    response, err := model.GenerateMessages(ctx, messages)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Custom provider response: %s", response[0].GetContent())
}
```

## Migration Guide

### For Existing Code

**No changes required!** All existing ChatModels code continues to work exactly as before. The compliance enhancements are additive and don't break any existing APIs.

### To Adopt New Features Gradually

1. **Start using registry for new code:**
   ```go
   // Old way (still works)
   model, err := chatmodels.NewOpenAIChatModel(options...)
   
   // New way with registry
   model, err := chatmodels.NewProvider(ctx, "openai", config)
   ```

2. **Enable observability:**
   ```go
   config.EnableMetrics = true
   config.EnableTracing = true
   config.EnableLogging = true
   ```

3. **Improve error handling:**
   ```go
   if chatErr := chatmodels.AsChatModelError(err); chatErr != nil {
       // Use structured error information
       log.Printf("Error code: %s", chatErr.GetCode())
   }
   ```

### Best Practices

1. **Registry Management:**
   ```go
   // Register providers at application startup
   func init() {
       chatmodels.RegisterGlobal("openai", chatmodels.NewOpenAICreator())
       chatmodels.RegisterGlobal("anthropic", chatmodels.NewAnthropicCreator())
   }
   ```

2. **Configuration Validation:**
   ```go
   // Always validate configuration
   if err := chatmodels.ValidateConfig(providerName, config); err != nil {
       return fmt.Errorf("invalid config: %w", err)
   }
   ```

3. **Error Handling:**
   ```go
   // Use structured error handling for better debugging
   if chatErr := chatmodels.AsChatModelError(err); chatErr != nil {
       if chatErr.IsRetryable() {
           // Implement retry logic
       }
       // Log with proper context
       log.WithFields(log.Fields{
           "operation": chatErr.GetOperation(),
           "provider": chatErr.GetProvider(),
           "error_code": chatErr.GetCode(),
       }).Error("ChatModel operation failed")
   }
   ```

4. **Observability:**
   ```go
   // Always enable observability in production
   config.EnableMetrics = true
   config.EnableTracing = true
   config.EnableLogging = true
   ```

## Testing with Enhanced Features

### Using Advanced Mocks

```go
func TestWithAdvancedMocks(t *testing.T) {
    // Create advanced mock with registry support
    mock := chatmodels.NewAdvancedMockChatModel("test-model", "mock-provider",
        chatmodels.WithMockResponses([]schema.Message{
            schema.NewAIMessage("Mock response 1"),
            schema.NewAIMessage("Mock response 2"),
        }),
        chatmodels.WithMockLatency(time.Millisecond * 100),
        chatmodels.WithMockErrorRate(0.1), // 10% error rate for testing
    )
    
    // Register mock provider
    err := chatmodels.RegisterGlobal("mock", func(ctx context.Context, config chatmodels.Config) (chatmodels.ChatModel, error) {
        return mock, nil
    })
    require.NoError(t, err)
    
    // Use mock through registry
    model, err := chatmodels.NewProvider(context.Background(), "mock", chatmodels.Config{
        Provider: "mock",
        Model: "test-model",
    })
    require.NoError(t, err)
    
    // Test with realistic mock behavior
    messages := []schema.Message{schema.NewHumanMessage("test")}
    response, err := model.GenerateMessages(context.Background(), messages)
    require.NoError(t, err)
    assert.NotEmpty(t, response)
}
```

This quickstart guide demonstrates how the ChatModels package maintains complete backward compatibility while providing powerful new compliance features that enhance observability, error handling, and extensibility through the global registry pattern.
