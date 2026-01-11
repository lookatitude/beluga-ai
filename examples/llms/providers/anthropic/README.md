# Anthropic Claude LLM Provider Example

This example demonstrates how to use the Anthropic Claude LLM provider with Beluga AI.

## Prerequisites

- Anthropic API key (set as `ANTHROPIC_API_KEY` environment variable)
- Go 1.21+

## Running the Example

```bash
export ANTHROPIC_API_KEY="your-api-key-here"
go run main.go
```

## What This Example Shows

1. Creating an Anthropic provider using the LLM factory
2. Configuring the provider with API key and model name
3. Generating text responses using Claude models

## Configuration Options

- `APIKey`: Your Anthropic API key (required)
- `ModelName`: Model to use (e.g., "claude-3-opus-20240229", "claude-3-sonnet-20240229")
- `BaseURL`: API base URL (defaults to Anthropic's endpoint)

## See Also

- [LLM Package Documentation](../../../pkg/llms/README.md)
- [Anthropic Provider Documentation](../../../pkg/llms/providers/anthropic/README.md)
