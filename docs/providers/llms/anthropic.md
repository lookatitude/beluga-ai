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

```go
config := llms.NewConfig(
    llms.WithProvider("anthropic"),
    llms.WithModelName("claude-3-sonnet-20240229"),
    llms.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
)
```

## Available Models

- `claude-3-opus-20240229` - Most capable
- `claude-3-sonnet-20240229` - Balanced
- `claude-3-haiku-20240307` - Fast, cost-effective

## Features

- Long context windows (200k tokens)
- Strong safety features
- Tool use support
- Streaming support

## Best Practices

- Use appropriate model for task complexity
- Leverage long context windows
- Implement proper error handling

---

**Next:** [Provider Comparison](./comparison.md)

