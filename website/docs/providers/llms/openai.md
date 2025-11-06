---
title: Openai
sidebar_position: 1
---

# OpenAI Provider Guide

Complete guide to using OpenAI models with Beluga AI Framework.

## Overview

OpenAI provides access to GPT-3.5, GPT-4, and other models through their API.

## Setup

### Get API Key

1. Visit https://platform.openai.com/api-keys
2. Create a new API key
3. Store securely (environment variable recommended)

### Configuration

**Basic Configuration:**
```go
config := llms.NewConfig(
    llms.WithProvider("openai"),                    // Required: Provider name
    llms.WithModelName("gpt-4"),                    // Required: Model identifier
    llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),  // Required: API key
    llms.WithTemperatureConfig(0.7),                // Optional: Creativity (0.0-2.0)
    llms.WithMaxTokensConfig(1000),                 // Optional: Max response length
)
```

**Complete Configuration with All Options:**
```go
config := llms.NewConfig(
    // Required options
    llms.WithProvider("openai"),
    llms.WithModelName("gpt-4"),
    llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    
    // Generation parameters
    llms.WithTemperatureConfig(0.7),              // Randomness: 0.0 (deterministic) to 2.0 (creative)
    llms.WithMaxTokensConfig(2000),               // Maximum tokens in response
    llms.WithTopPConfig(0.9),                     // Nucleus sampling: 0.0 to 1.0
    llms.WithTopKConfig(40),                      // Top-K sampling: 1 to 100
    llms.WithStopSequences([]string{"\n\n"}),     // Stop generation at these sequences
    
    // Retry configuration
    llms.WithRetryConfig(
        5,                    // Max retries for transient errors
        2 * time.Second,      // Initial retry delay
        2.0,                  // Exponential backoff multiplier
    ),
    
    // Timeout configuration
    llms.WithTimeout(60 * time.Second),           // Request timeout
    
    // Advanced options
    llms.WithBaseURL("https://api.openai.com/v1"), // Custom API endpoint (optional)
    llms.WithEnableStreaming(true),               // Enable streaming support
    llms.WithEnableToolCalling(true),             // Enable function calling
)
```

**Configuration Validation:**
```go
config := llms.NewConfig(
    llms.WithProvider("openai"),
    llms.WithModelName("gpt-4"),
    llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
)

// Validate configuration
if err := config.Validate(); err != nil {
    return fmt.Errorf("invalid configuration: %w", err)
}
```

**Creating Provider from Config:**
```go
factory := llms.NewFactory()
provider, err := factory.CreateProvider("openai", config)
if err != nil {
    return fmt.Errorf("failed to create provider: %w", err)
}

// Verify provider is ready
modelName := provider.GetModelName()
fmt.Printf("Provider ready: %s\n", modelName)
```

## Available Models

- `gpt-4` - Most capable model
- `gpt-4-turbo` - Faster GPT-4 variant
- `gpt-3.5-turbo` - Cost-effective, fast
- `gpt-4o` - Latest GPT-4 variant

## API Reference

### Generate

Generates a response from the LLM based on the provided messages.

**Signature:**
```go
Generate(ctx context.Context, messages []schema.Message, options ...GenerateOption) (schema.Message, error)
```

**Parameters:**
- `ctx` (context.Context): Context for cancellation and timeout control. Use `context.WithTimeout()` for request timeouts.
- `messages` ([]schema.Message): Array of conversation messages. Must include at least one message. Use `schema.NewHumanMessage()`, `schema.NewSystemMessage()`, or `schema.NewAIMessage()` to create messages.
- `options` (...GenerateOption): Optional generation parameters. Can include `WithTemperature()`, `WithMaxTokens()`, etc.

**Returns:**
- `schema.Message`: The generated response message. Use `message.GetContent()` to access the text content.
- `error`: Error if generation fails. Check with `llms.IsLLMError()` and `llms.IsRetryableError()`.

**Example:**
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

messages := []schema.Message{
    schema.NewSystemMessage("You are a helpful assistant."),
    schema.NewHumanMessage("What is the capital of France?"),
}

response, err := provider.Generate(ctx, messages)
if err != nil {
    if llms.IsLLMError(err) {
        code := llms.GetLLMErrorCode(err)
        if code == "rate_limit" && llms.IsRetryableError(err) {
            // Implement retry logic
            return handleRetryableError(err)
        }
    }
    return fmt.Errorf("generation failed: %w", err)
}

fmt.Printf("Response: %s\n", response.GetContent())
```

**Error Handling:**
```go
response, err := provider.Generate(ctx, messages)
if err != nil {
    // Check error type
    if llms.IsLLMError(err) {
        code := llms.GetLLMErrorCode(err)
        switch code {
        case "rate_limit":
            // Rate limit exceeded - retry with backoff
            return retryWithBackoff(ctx, provider, messages)
        case "authentication_error":
            // Invalid API key - don't retry
            return fmt.Errorf("authentication failed: check API key")
        case "invalid_request":
            // Invalid request format - check messages
            return fmt.Errorf("invalid request: %w", err)
        default:
            return fmt.Errorf("LLM error [%s]: %w", code, err)
        }
    }
    return fmt.Errorf("unexpected error: %w", err)
}
```

### StreamChat

Streams responses in real-time as they're generated.

**Signature:**
```go
StreamChat(ctx context.Context, messages []schema.Message, options ...GenerateOption) (<-chan schema.AIMessageChunk, error)
```

**Parameters:**
- `ctx` (context.Context): Context for cancellation. The stream will stop when context is cancelled.
- `messages` ([]schema.Message): Array of conversation messages.
- `options` (...GenerateOption): Optional generation parameters.

**Returns:**
- `<-chan schema.AIMessageChunk`: Channel that receives message chunks. Each chunk contains:
  - `Content` (string): Text content of the chunk
  - `Err` (error): Error if chunk generation failed (nil otherwise)
  - Channel closes when stream completes or errors
- `error`: Initial error if streaming setup fails (nil if stream starts successfully)

**Example:**
```go
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()

messages := []schema.Message{
    schema.NewHumanMessage("Write a short story about AI."),
}

streamChan, err := provider.StreamChat(ctx, messages)
if err != nil {
    return fmt.Errorf("failed to start stream: %w", err)
}

var fullResponse strings.Builder
for chunk := range streamChan {
    if chunk.Err != nil {
        // Handle streaming error
        if chunk.Err == context.DeadlineExceeded {
            return fmt.Errorf("stream timeout: %w", chunk.Err)
        }
        return fmt.Errorf("stream error: %w", chunk.Err)
    }
    
    // Process chunk
    fmt.Print(chunk.Content) // Print as it arrives
    fullResponse.WriteString(chunk.Content)
}

fmt.Printf("\n\nFull response: %s\n", fullResponse.String())
```

**Error Handling:**
```go
streamChan, err := provider.StreamChat(ctx, messages)
if err != nil {
    return fmt.Errorf("stream setup failed: %w", err)
}

for chunk := range streamChan {
    if chunk.Err != nil {
        // Check if error is retryable
        if llms.IsRetryableError(chunk.Err) {
            // Optionally restart stream
            return restartStream(ctx, provider, messages)
        }
        return fmt.Errorf("stream error: %w", chunk.Err)
    }
    processChunk(chunk)
}
```

### BindTools

Binds tools to the provider for function calling support.

**Signature:**
```go
BindTools(toolsToBind []tools.Tool) ChatModel
```

**Parameters:**
- `toolsToBind` ([]tools.Tool): Array of tools to bind. Tools must implement the `tools.Tool` interface with `Name()`, `Description()`, and `Execute()` methods.

**Returns:**
- `ChatModel`: New ChatModel instance with tools bound. Use this instance for subsequent calls.

**Example:**
```go
import (
    "encoding/json"
    "github.com/lookatitude/beluga-ai/pkg/agents/tools"
    "github.com/lookatitude/beluga-ai/pkg/agents/tools/providers"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

// Create tools
calculator := providers.NewCalculatorTool()
echoTool := providers.NewEchoTool()

// Bind tools to provider
tools := []tools.Tool{calculator, echoTool}
providerWithTools := provider.BindTools(tools)

// Use provider with tools
messages := []schema.Message{
    schema.NewSystemMessage("You can use tools to help answer questions."),
    schema.NewHumanMessage("Calculate 15 * 23 and tell me the result"),
}

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

response, err := providerWithTools.Generate(ctx, messages)
if err != nil {
    return fmt.Errorf("generation failed: %w", err)
}

// Check if tool was called
if len(response.ToolCalls()) > 0 {
    for _, toolCall := range response.ToolCalls() {
        // Access tool name via Function.Name or Name field
        toolName := toolCall.Function.Name
        if toolName == "" {
            toolName = toolCall.Name
        }
        fmt.Printf("Tool called: %s (ID: %s) with args: %s\n", 
            toolName, toolCall.ID, toolCall.Function.Arguments)
    }
}
```

**Tool Call Handling:**
```go
import (
    "context"
    "encoding/json"
    "fmt"
    
    "github.com/lookatitude/beluga-ai/pkg/agents/tools"
    "github.com/lookatitude/beluga-ai/pkg/agents/tools/providers"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

response, err := providerWithTools.Generate(ctx, messages)
if err != nil {
    return "", err
}

// Process tool calls
if len(response.ToolCalls()) > 0 {
    var toolResults []schema.Message
    
    // Create tool registry for lookup
    toolMap := make(map[string]tools.Tool)
    for _, tool := range tools {
        toolMap[tool.Name()] = tool
    }
    
    for _, toolCall := range response.ToolCalls() {
        // Get tool name from Function.Name or Name field
        toolName := toolCall.Function.Name
        if toolName == "" {
            toolName = toolCall.Name
        }
        
        // Find tool by name
        tool, exists := toolMap[toolName]
        if !exists {
            toolResults = append(toolResults, 
                schema.NewToolMessage("Tool not found", toolCall.ID))
            continue
        }
        
        // Parse arguments (JSON string) to map for tool execution
        var args map[string]interface{}
        argsStr := toolCall.Function.Arguments
        if argsStr == "" {
            argsStr = toolCall.Arguments
        }
        if argsStr != "" {
            if err := json.Unmarshal([]byte(argsStr), &args); err != nil {
                toolResults = append(toolResults,
                    schema.NewToolMessage(fmt.Sprintf("Error parsing arguments: %v", err), toolCall.ID))
                continue
            }
        }
        
        // Execute tool (Execute takes any as input, returns any)
        result, err := tool.Execute(ctx, args)
        if err != nil {
            toolResults = append(toolResults,
                schema.NewToolMessage(fmt.Sprintf("Error: %v", err), toolCall.ID))
            continue
        }
        
        // Convert result to string if needed
        resultStr := fmt.Sprintf("%v", result)
        
        // Add tool result to conversation (use toolCall.ID, not name)
        toolResults = append(toolResults,
            schema.NewToolMessage(resultStr, toolCall.ID))
    }
    
    // Continue conversation with tool results
    messages = append(messages, response)
    messages = append(messages, toolResults...)
    
    // Get final response
    finalResponse, err := providerWithTools.Generate(ctx, messages)
    if err != nil {
        return "", err
    }
    return finalResponse.GetContent(), nil
}

return response.GetContent(), nil
```

### Batch

Processes multiple requests in parallel for efficiency.

**Signature:**
```go
Batch(ctx context.Context, inputs []interface{}) ([]interface{}, error)
```

**Parameters:**
- `ctx` (context.Context): Context for cancellation and timeout.
- `inputs` ([]interface{}): Array of inputs. Each input should be `[]schema.Message` or convertible to messages.

**Returns:**
- `[]interface{}`: Array of results. Each result is a `schema.Message`.
- `error`: Error if batch processing fails. Individual item failures may be in result messages.

**Example:**
```go
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()

// Prepare batch inputs
inputs := []interface{}{
    []schema.Message{schema.NewHumanMessage("What is 2+2?")},
    []schema.Message{schema.NewHumanMessage("What is the capital of France?")},
    []schema.Message{schema.NewHumanMessage("Explain quantum computing briefly.")},
}

// Process batch
results, err := provider.Batch(ctx, inputs)
if err != nil {
    return fmt.Errorf("batch processing failed: %w", err)
}

// Process results
for i, result := range results {
    if msg, ok := result.(schema.Message); ok {
        fmt.Printf("Result %d: %s\n", i+1, msg.GetContent())
    }
}
```

**Error Handling:**
```go
results, err := provider.Batch(ctx, inputs)
if err != nil {
    // Check if it's a batch-level error
    if llms.IsLLMError(err) {
        return fmt.Errorf("batch error: %w", err)
    }
    return err
}

// Check individual results for errors
for i, result := range results {
    if msg, ok := result.(schema.Message); ok {
        if msg.Err != nil {
            fmt.Printf("Item %d failed: %v\n", i, msg.Err)
            continue
        }
        processResult(msg)
    }
}
```

### GetModelName

Returns the model name configured for this provider.

**Signature:**
```go
GetModelName() string
```

**Returns:**
- `string`: The model name (e.g., "gpt-4", "gpt-3.5-turbo")

**Example:**
```go
modelName := provider.GetModelName()
fmt.Printf("Using model: %s\n", modelName)
```

## Features

### Streaming

See [StreamChat API Reference](#streamchat) above for detailed documentation.

### Tool Calling

See [BindTools API Reference](#bindtools) above for detailed documentation.

### Batch Processing

See [Batch API Reference](#batch) above for detailed documentation.

## Rate Limiting

OpenAI enforces rate limits. Implement retry logic:

```go
config := llms.NewConfig(
    llms.WithRetryConfig(5, 2*time.Second, 2.0),
)
```

## Cost Optimization

1. Use GPT-3.5 for simple tasks
2. Reduce max_tokens when possible
3. Cache identical requests
4. Batch requests when possible

## Best Practices

- Always set timeouts
- Handle rate limits gracefully
- Monitor token usage
- Use appropriate models for tasks

## Troubleshooting

See [Troubleshooting Guide](../../guides/troubleshooting) for common issues.

---

**Next:** [Anthropic Guide](./anthropic) or [Provider Comparison](./comparison)

