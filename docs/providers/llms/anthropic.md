# Anthropic Provider Guide

Complete guide to using Anthropic Claude models with Beluga AI Framework.

## Overview

Anthropic provides Claude models with focus on safety and long context windows.

## Setup

### Get API Key

1. Visit https://console.anthropic.com/
2. Create API key
3. Store securely

### Configuration

**Basic Configuration:**
```go
config := llms.NewConfig(
    llms.WithProvider("anthropic"),                    // Required: Provider name
    llms.WithModelName("claude-3-sonnet-20240229"),    // Required: Model identifier
    llms.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),  // Required: API key
    llms.WithTemperatureConfig(0.7),                  // Optional: Creativity (0.0-1.0)
    llms.WithMaxTokensConfig(4096),                   // Optional: Max response length
)
```

**Complete Configuration:**
```go
config := llms.NewConfig(
    // Required options
    llms.WithProvider("anthropic"),
    llms.WithModelName("claude-3-sonnet-20240229"),
    llms.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
    
    // Generation parameters
    llms.WithTemperatureConfig(0.7),              // Randomness: 0.0 (deterministic) to 1.0 (creative)
    llms.WithMaxTokensConfig(4096),              // Maximum tokens in response
    llms.WithTopPConfig(0.9),                    // Nucleus sampling: 0.0 to 1.0
    llms.WithStopSequences([]string{"\n\nHuman:"}), // Stop generation at these sequences
    
    // Retry configuration
    llms.WithRetryConfig(
        5,                    // Max retries for transient errors
        2 * time.Second,      // Initial retry delay
        2.0,                  // Exponential backoff multiplier
    ),
    
    // Timeout configuration
    llms.WithTimeout(60 * time.Second),           // Request timeout
    
    // Advanced options
    llms.WithEnableStreaming(true),               // Enable streaming support
    llms.WithEnableToolCalling(true),             // Enable tool use
)
```

**Creating Provider:**
```go
factory := llms.NewFactory()
provider, err := factory.CreateProvider("anthropic", config)
if err != nil {
    return fmt.Errorf("failed to create provider: %w", err)
}
```

## Available Models

- `claude-3-opus-20240229` - Most capable
- `claude-3-sonnet-20240229` - Balanced
- `claude-3-haiku-20240307` - Fast, cost-effective

## API Reference

### Generate

Generates a response from Claude models. See [OpenAI Provider Guide](./openai.md#generate) for detailed API documentation - the interface is identical across providers.

**Example:**
```go
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()

messages := []schema.Message{
    schema.NewSystemMessage("You are a helpful assistant."),
    schema.NewHumanMessage("Explain quantum computing in simple terms."),
}

response, err := provider.Generate(ctx, messages)
if err != nil {
    if llms.IsLLMError(err) {
        code := llms.GetLLMErrorCode(err)
        if code == "rate_limit" && llms.IsRetryableError(err) {
            // Implement retry logic
            return retryWithBackoff(ctx, provider, messages)
        }
    }
    return fmt.Errorf("generation failed: %w", err)
}

fmt.Printf("Response: %s\n", response.GetContent())
```

### StreamChat

Streams responses in real-time. See [OpenAI Provider Guide](./openai.md#streamchat) for detailed API documentation.

**Example:**
```go
streamChan, err := provider.StreamChat(ctx, messages)
if err != nil {
    return fmt.Errorf("failed to start stream: %w", err)
}

var fullResponse strings.Builder
for chunk := range streamChan {
    if chunk.Err != nil {
        return fmt.Errorf("stream error: %w", chunk.Err)
    }
    fmt.Print(chunk.Content)
    fullResponse.WriteString(chunk.Content)
}
```

### BindTools

Binds tools for function calling. See [OpenAI Provider Guide](./openai.md#bindtools) for detailed API documentation.

## Features

- **Long context windows**: Up to 200k tokens for Claude 3.5 Sonnet
- **Strong safety features**: Built-in safety mechanisms
- **Tool use support**: Native function calling support
- **Streaming support**: Real-time response generation

## Best Practices

- Use appropriate model for task complexity
- Leverage long context windows
- Implement proper error handling

---

**Next:** [Provider Comparison](./comparison.md)

