# LLM Concepts

This document explains how Large Language Models (LLMs) work in Beluga AI, including provider abstraction, streaming, tool calling, and batch processing.

## Provider Abstraction

Beluga AI provides a unified interface for different LLM providers.

### ChatModel Interface

The `ChatModel` interface provides a unified API for all LLM providers.

**Interface Definition:**
```go
type ChatModel interface {
    // Generate creates a complete response from messages
    // Parameters:
    //   - ctx: Context for cancellation and timeout
    //   - messages: Conversation history (system, human, AI messages)
    //   - options: Optional generation parameters (temperature, max_tokens, etc.)
    // Returns:
    //   - Message: Generated response with content and metadata
    //   - error: Error if generation fails (check with llms.IsLLMError())
    Generate(ctx context.Context, messages []Message, options ...Option) (Message, error)
    
    // StreamChat streams responses in real-time
    // Parameters:
    //   - ctx: Context for cancellation
    //   - messages: Conversation history
    //   - options: Optional generation parameters
    // Returns:
    //   - <-chan AIMessageChunk: Channel receiving response chunks
    //   - error: Initial error if stream setup fails
    StreamChat(ctx context.Context, messages []Message, options ...Option) (<-chan AIMessageChunk, error)
    
    // BindTools attaches tools for function calling
    // Parameters:
    //   - toolsToBind: Array of tools implementing tools.Tool interface
    // Returns:
    //   - ChatModel: New ChatModel instance with tools bound
    BindTools(toolsToBind []Tool) ChatModel
    
    // GetModelName returns the configured model name
    // Returns:
    //   - string: Model identifier (e.g., "gpt-4", "claude-3-sonnet")
    GetModelName() string
}
```

**Complete Usage Example:**
```go
// 1. Create configuration
config := llms.NewConfig(
    llms.WithProvider("openai"),
    llms.WithModelName("gpt-4"),
    llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    llms.WithTemperatureConfig(0.7),
    llms.WithMaxTokensConfig(1000),
    llms.WithTimeout(30 * time.Second),
    llms.WithRetryConfig(3, 1*time.Second, 2.0),
)

// 2. Create provider using factory
factory := llms.NewFactory()
provider, err := factory.CreateProvider("openai", config)
if err != nil {
    return fmt.Errorf("failed to create provider: %w", err)
}

// 3. Prepare messages
messages := []schema.Message{
    schema.NewSystemMessage("You are a helpful AI assistant."),
    schema.NewHumanMessage("Explain quantum computing in simple terms."),
}

// 4. Generate response with error handling
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

response, err := provider.Generate(ctx, messages)
if err != nil {
    // Handle different error types
    if llms.IsLLMError(err) {
        code := llms.GetLLMErrorCode(err)
        switch code {
        case "rate_limit":
            // Implement retry with exponential backoff
            return handleRateLimit(ctx, provider, messages)
        case "authentication_error":
            return fmt.Errorf("invalid API key: %w", err)
        default:
            return fmt.Errorf("LLM error [%s]: %w", code, err)
        }
    }
    return fmt.Errorf("generation failed: %w", err)
}

// 5. Process response
fmt.Printf("Model: %s\n", provider.GetModelName())
fmt.Printf("Response: %s\n", response.GetContent())
```

### Supported Providers

- **OpenAI**: GPT-3.5, GPT-4, and variants
- **Anthropic**: Claude models
- **AWS Bedrock**: Various foundation models
- **Ollama**: Local models

### Provider Factory

The factory pattern allows creating providers dynamically based on configuration.

**Factory API:**
```go
// NewFactory creates a new LLM factory instance
factory := llms.NewFactory()

// CreateProvider creates a provider instance
// Parameters:
//   - providerName: Provider identifier ("openai", "anthropic", "ollama", etc.)
//   - config: Configuration struct with provider settings
// Returns:
//   - ChatModel: Provider instance implementing ChatModel interface
//   - error: Error if provider creation fails (invalid config, unsupported provider, etc.)
provider, err := factory.CreateProvider("openai", config)
if err != nil {
    return fmt.Errorf("failed to create provider: %w", err)
}

// ListAvailableProviders returns all registered provider names
providers := factory.ListAvailableProviders()
fmt.Printf("Available providers: %v\n", providers)
```

**Complete Factory Example:**
```go
// Create factory
factory := llms.NewFactory()

// Create configuration
config := llms.NewConfig(
    llms.WithProvider("openai"),
    llms.WithModelName("gpt-4"),
    llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
)

// Validate config before creating provider
if err := config.Validate(); err != nil {
    return fmt.Errorf("invalid config: %w", err)
}

// Create provider
provider, err := factory.CreateProvider("openai", config)
if err != nil {
    // Check if provider is not found
    if strings.Contains(err.Error(), "not registered") {
        return fmt.Errorf("provider 'openai' not available: %w", err)
    }
    return fmt.Errorf("failed to create provider: %w", err)
}

// Use provider
response, err := provider.Generate(ctx, messages)
```

## ChatModel vs LLM Interfaces

### ChatModel

Optimized for chat/conversation:
- Message-based input/output
- Streaming support
- Tool calling
- Conversation context

### LLM

Lower-level interface:
- Text input/output
- Direct prompt processing
- More control over generation

## Streaming Responses

Streaming provides real-time response generation.

### Basic Streaming

**StreamChat API:**
```go
// StreamChat streams responses in real-time
// Parameters:
//   - ctx: Context for cancellation (use context.WithTimeout for timeouts)
//   - messages: Conversation messages
//   - options: Optional generation parameters
// Returns:
//   - <-chan AIMessageChunk: Channel of message chunks
//     Each chunk contains:
//       - Content (string): Text content of the chunk
//       - Err (error): Error if chunk generation failed (nil otherwise)
//   - error: Initial error if stream setup fails
streamChan, err := provider.StreamChat(ctx, messages)
if err != nil {
    return fmt.Errorf("failed to start stream: %w", err)
}

// Process chunks as they arrive
var fullResponse strings.Builder
for chunk := range streamChan {
    if chunk.Err != nil {
        // Handle streaming errors
        if chunk.Err == context.DeadlineExceeded {
            return fmt.Errorf("stream timeout: %w", chunk.Err)
        }
        if llms.IsRetryableError(chunk.Err) {
            // Optionally restart stream
            return restartStream(ctx, provider, messages)
        }
        return fmt.Errorf("stream error: %w", chunk.Err)
    }
    
    // Process chunk content
    fmt.Print(chunk.Content) // Print as it arrives
    fullResponse.WriteString(chunk.Content)
}

fmt.Printf("\nComplete response: %s\n", fullResponse.String())
```

**Streaming with Context Cancellation:**
```go
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()

streamChan, err := provider.StreamChat(ctx, messages)
if err != nil {
    return err
}

// Process with cancellation support
done := make(chan bool)
go func() {
    for chunk := range streamChan {
        if chunk.Err != nil {
            if chunk.Err == context.Canceled {
                fmt.Println("Stream cancelled")
                return
            }
            fmt.Printf("Stream error: %v\n", chunk.Err)
            return
        }
        fmt.Print(chunk.Content)
    }
    done <- true
}()

// Cancel after 30 seconds if needed
select {
case <-time.After(30 * time.Second):
    cancel()
    fmt.Println("\nStream cancelled due to timeout")
case <-done:
    fmt.Println("\nStream completed")
}
```

### Streaming Benefits

- **Lower latency**: See responses as they're generated
- **Better UX**: Progressive rendering
- **Efficiency**: Process chunks incrementally

## Tool Calling

Tool calling allows LLMs to invoke functions.

### Binding Tools

**BindTools API:**
```go
// BindTools attaches tools for function calling
// Parameters:
//   - toolsToBind: Array of tools implementing tools.Tool interface
//     Each tool must implement:
//       - Name() string: Tool identifier
//       - Description() string: Tool description for LLM
//       - Execute(ctx, args) (string, error): Tool execution logic
// Returns:
//   - ChatModel: New ChatModel instance with tools bound
//     Use this instance for subsequent Generate() calls

// Create tools
calculator := tools.NewCalculatorTool()
echoTool := tools.NewEchoTool()
customTool := tools.NewCustomTool("weather", getWeather)

// Bind tools
tools := []tools.Tool{calculator, echoTool, customTool}
providerWithTools := provider.BindTools(tools)

// Use provider with tools
response, err := providerWithTools.Generate(ctx, messages)
```

**Complete Tool Calling Example:**
```go
import (
    "context"
    "encoding/json"
    "fmt"
    
    "github.com/lookatitude/beluga-ai/pkg/agents/tools"
    "github.com/lookatitude/beluga-ai/pkg/agents/tools/providers"
    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

// 1. Create and bind tools
calculator := providers.NewCalculatorTool()
echoTool := providers.NewEchoTool()
tools := []tools.Tool{calculator, echoTool}

providerWithTools := provider.BindTools(tools)

// 2. Make request that may trigger tool calls
messages := []schema.Message{
    schema.NewSystemMessage("You can use tools to help answer questions."),
    schema.NewHumanMessage("Calculate 15 * 23 and echo the result"),
}

response, err := providerWithTools.Generate(ctx, messages)
if err != nil {
    return err
}

// 3. Check for tool calls
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
        
        fmt.Printf("Tool called: %s (ID: %s) with args: %s\n", 
            toolName, toolCall.ID, toolCall.Function.Arguments)
        
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
        
        // Convert result to string
        resultStr := fmt.Sprintf("%v", result)
        
        // Add tool result (use toolCall.ID, not name)
        toolResults = append(toolResults,
            schema.NewToolMessage(resultStr, toolCall.ID))
    }
    
    // 4. Continue conversation with tool results
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

### Tool Call Handling

```go
response, err := provider.Generate(ctx, messages)
if err != nil {
    return err
}

// Check for tool calls
if len(response.ToolCalls()) > 0 {
    for _, toolCall := range response.ToolCalls() {
        // Execute tool
        result := executeTool(toolCall)
        
        // Continue conversation with tool result
        messages = append(messages, schema.NewToolMessage(result, toolCall.Name))
    }
}
```

## Batch Processing

Batch processing enables efficient bulk operations.

### Batch Execution

**Batch API:**
```go
// Batch processes multiple requests in parallel
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - inputs: Array of inputs, each should be []schema.Message or convertible
// Returns:
//   - []interface{}: Array of results, each is a schema.Message
//   - error: Error if batch processing fails (individual failures may be in results)

// Prepare batch inputs
inputs := []interface{}{
    []schema.Message{schema.NewHumanMessage("What is 2+2?")},
    []schema.Message{schema.NewHumanMessage("What is the capital of France?")},
    []schema.Message{schema.NewHumanMessage("Explain quantum computing briefly.")},
}

// Process batch
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()

results, err := provider.Batch(ctx, inputs)
if err != nil {
    return fmt.Errorf("batch processing failed: %w", err)
}

// Process results
for i, result := range results {
    if msg, ok := result.(schema.Message); ok {
        if msg.Err != nil {
            fmt.Printf("Item %d failed: %v\n", i, msg.Err)
            continue
        }
        fmt.Printf("Result %d: %s\n", i+1, msg.GetContent())
    }
}
```

**Batch with Error Handling:**
```go
results, err := provider.Batch(ctx, inputs)
if err != nil {
    // Batch-level error (e.g., all requests failed)
    if llms.IsLLMError(err) {
        code := llms.GetLLMErrorCode(err)
        if code == "rate_limit" {
            // Implement batch retry with backoff
            return retryBatchWithBackoff(ctx, provider, inputs)
        }
    }
    return fmt.Errorf("batch error: %w", err)
}

// Check individual results
successCount := 0
for i, result := range results {
    if msg, ok := result.(schema.Message); ok {
        if msg.Err != nil {
            fmt.Printf("Item %d error: %v\n", i, msg.Err)
            continue
        }
        processResult(msg)
        successCount++
    }
}

fmt.Printf("Processed %d/%d items successfully\n", successCount, len(inputs))
```

### Batch Configuration

```go
config := llms.NewConfig(
    llms.WithMaxConcurrentBatches(10), // Parallel batch limit
)
```

## Configuration Patterns

### Temperature

Controls randomness (0.0 = deterministic, 2.0 = creative):

```go
llms.WithTemperatureConfig(0.7) // Balanced
```

### Max Tokens

Limits response length:

```go
llms.WithMaxTokensConfig(1000)
```

### Retry Configuration

Handles transient failures:

```go
llms.WithRetryConfig(
    3,                    // Max retries
    1 * time.Second,      // Initial delay
    2.0,                  // Backoff multiplier
)
```

## Error Handling

### Error Types

**Error Checking Functions:**
```go
// IsLLMError checks if error is an LLM-specific error
// Returns: true if error is from LLM provider
if llms.IsLLMError(err) {
    // Get error code for programmatic handling
    code := llms.GetLLMErrorCode(err)
    
    // Check if error can be retried
    if llms.IsRetryableError(err) {
        // Implement retry logic with exponential backoff
        return retryWithBackoff(ctx, provider, messages, maxRetries)
    }
}
```

**Complete Error Handling Example:**
```go
response, err := provider.Generate(ctx, messages)
if err != nil {
    // Check if it's an LLM-specific error
    if llms.IsLLMError(err) {
        code := llms.GetLLMErrorCode(err)
        
        switch code {
        case "rate_limit":
            // Rate limit exceeded - retry with exponential backoff
            if llms.IsRetryableError(err) {
                return retryWithBackoff(ctx, provider, messages, 5)
            }
            return fmt.Errorf("rate limit exceeded: %w", err)
            
        case "authentication_error":
            // Invalid API key - don't retry
            return fmt.Errorf("authentication failed: check API key")
            
        case "invalid_request":
            // Invalid request format - check messages
            return fmt.Errorf("invalid request format: %w", err)
            
        case "network_error":
            // Network issue - retry
            if llms.IsRetryableError(err) {
                return retryWithBackoff(ctx, provider, messages, 3)
            }
            return fmt.Errorf("network error: %w", err)
            
        case "internal_error":
            // Provider internal error - retry
            if llms.IsRetryableError(err) {
                return retryWithBackoff(ctx, provider, messages, 3)
            }
            return fmt.Errorf("provider error: %w", err)
            
        default:
            return fmt.Errorf("LLM error [%s]: %w", code, err)
        }
    }
    
    // Non-LLM error (e.g., context cancelled, validation error)
    return fmt.Errorf("unexpected error: %w", err)
}
```

**Retry Helper Function:**
```go
func retryWithBackoff(ctx context.Context, provider ChatModel, 
    messages []schema.Message, maxRetries int) (schema.Message, error) {
    
    var lastErr error
    delay := 1 * time.Second
    
    for i := 0; i < maxRetries; i++ {
        // Check context cancellation
        select {
        case <-ctx.Done():
            return schema.Message{}, ctx.Err()
        default:
        }
        
        // Attempt request
        response, err := provider.Generate(ctx, messages)
        if err == nil {
            return response, nil
        }
        
        lastErr = err
        
        // Check if error is still retryable
        if !llms.IsRetryableError(err) {
            return schema.Message{}, err
        }
        
        // Wait before retry
        if i < maxRetries-1 {
            time.Sleep(delay)
            delay *= 2 // Exponential backoff
        }
    }
    
    return schema.Message{}, fmt.Errorf("max retries exceeded: %w", lastErr)
}
```

### Common Error Codes

- `rate_limit`: Rate limit exceeded
- `authentication_error`: Invalid API key
- `invalid_request`: Invalid request format
- `network_error`: Network issues
- `internal_error`: Provider error

## Best Practices

1. **Use appropriate models**: Choose models based on task complexity
2. **Set timeouts**: Always use context with timeout
3. **Handle errors**: Implement retry logic for retryable errors
4. **Monitor usage**: Track token usage and costs
5. **Cache responses**: Cache identical requests when appropriate

## Related Concepts

- [Core Concepts](./core.md) - Foundation patterns
- [Agent Concepts](./agents.md) - Using LLMs in agents
- [Provider Documentation](../providers/llms/) - Provider-specific guides

---

**Next:** Learn about [Agent Concepts](./agents.md) or [RAG Concepts](./rag.md)

