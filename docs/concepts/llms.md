# LLM Concepts

This document explains how Large Language Models (LLMs) work in Beluga AI, including provider abstraction, streaming, tool calling, and batch processing.

## Provider Abstraction

Beluga AI provides a unified interface for different LLM providers.

### ChatModel Interface

```go
type ChatModel interface {
    Generate(ctx context.Context, messages []Message, options ...Option) (Message, error)
    StreamChat(ctx context.Context, messages []Message, options ...Option) (<-chan AIMessageChunk, error)
    BindTools(toolsToBind []Tool) ChatModel
    GetModelName() string
}
```

### Supported Providers

- **OpenAI**: GPT-3.5, GPT-4, and variants
- **Anthropic**: Claude models
- **AWS Bedrock**: Various foundation models
- **Ollama**: Local models

### Provider Factory

```go
factory := llms.NewFactory()
provider, err := factory.CreateProvider("openai", config)
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

```go
streamChan, err := provider.StreamChat(ctx, messages)
if err != nil {
    return err
}

for chunk := range streamChan {
    if chunk.Err != nil {
        return chunk.Err
    }
    fmt.Print(chunk.Content)
}
```

### Streaming Benefits

- **Lower latency**: See responses as they're generated
- **Better UX**: Progressive rendering
- **Efficiency**: Process chunks incrementally

## Tool Calling

Tool calling allows LLMs to invoke functions.

### Binding Tools

```go
tools := []tools.Tool{
    tools.NewCalculatorTool(),
    tools.NewEchoTool(),
}

providerWithTools := provider.BindTools(tools)
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

```go
inputs := []any{
    []Message{schema.NewHumanMessage("Question 1")},
    []Message{schema.NewHumanMessage("Question 2")},
    []Message{schema.NewHumanMessage("Question 3")},
}

results, err := provider.Batch(ctx, inputs)
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

```go
// Check if LLM error
if llms.IsLLMError(err) {
    code := llms.GetLLMErrorCode(err)
    
    // Check if retryable
    if llms.IsRetryableError(err) {
        // Implement retry
    }
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

