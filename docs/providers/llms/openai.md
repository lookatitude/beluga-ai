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

```go
config := llms.NewConfig(
    llms.WithProvider("openai"),
    llms.WithModelName("gpt-4"),
    llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    llms.WithTemperatureConfig(0.7),
    llms.WithMaxTokensConfig(1000),
)
```

## Available Models

- `gpt-4` - Most capable model
- `gpt-4-turbo` - Faster GPT-4 variant
- `gpt-3.5-turbo` - Cost-effective, fast
- `gpt-4o` - Latest GPT-4 variant

## Features

### Streaming

```go
streamChan, err := provider.StreamChat(ctx, messages)
for chunk := range streamChan {
    fmt.Print(chunk.Content)
}
```

### Tool Calling

```go
tools := []tools.Tool{calculator, echo}
providerWithTools := provider.BindTools(tools)
```

### Batch Processing

```go
results, err := provider.Batch(ctx, inputs)
```

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

See [Troubleshooting Guide](../../TROUBLESHOOTING.md) for common issues.

---

**Next:** [Anthropic Guide](./anthropic.md) or [Provider Comparison](./comparison.md)

