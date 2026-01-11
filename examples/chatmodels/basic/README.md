# ChatModels Basic Example

This example demonstrates how to use the ChatModels package for chat-based language model interactions.

## Prerequisites

- Go 1.21+
- Optional: API keys for real providers (OpenAI, Anthropic, etc.)

## Running the Example

```bash
go run main.go
```

## What This Example Shows

1. Creating chat model configuration with default settings
2. Creating a chat model instance
3. Creating conversation messages
4. Generating responses from the model
5. Streaming responses in real-time

## Configuration Options

- `DefaultProvider`: Provider to use (e.g., "openai", "anthropic")
- `DefaultModel`: Model name (e.g., "gpt-4", "claude-3-opus")
- `DefaultTemperature`: Temperature for generation (0.0-2.0)
- `DefaultMaxTokens`: Maximum tokens in response
- `DefaultTimeout`: Request timeout duration

## Using Real Providers

To use a real chat model provider:

```go
config := chatmodels.NewDefaultConfig()
config.DefaultProvider = "openai"
config.DefaultModel = "gpt-4"
// Set API key in environment or config

model, err := chatmodels.NewChatModel("gpt-4", config)
```

## See Also

- [ChatModels Package Documentation](../../../pkg/chatmodels/README.md)
- [LLM Usage Examples](../../llm-usage/main.go)
