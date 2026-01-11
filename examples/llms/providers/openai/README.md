# OpenAI LLM Provider Example

This example demonstrates how to use the OpenAI LLM provider with Beluga AI.

## Prerequisites

- OpenAI API key (set as `OPENAI_API_KEY` environment variable)
- Go 1.21+

## Running the Example

```bash
export OPENAI_API_KEY="your-api-key-here"
go run main.go
```

## What This Example Shows

1. Creating an OpenAI provider using the LLM factory
2. Configuring the provider with API key and model name
3. Generating text responses using the OpenAI GPT models

## Configuration Options

- `APIKey`: Your OpenAI API key (required)
- `ModelName`: Model to use (e.g., "gpt-4", "gpt-3.5-turbo")
- `BaseURL`: API base URL (defaults to OpenAI's endpoint)

## See Also

- [LLM Package Documentation](../../../pkg/llms/README.md)
- [OpenAI Provider Documentation](../../../pkg/llms/providers/openai/README.md)
