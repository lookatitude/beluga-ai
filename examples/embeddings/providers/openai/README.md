# OpenAI Embedding Provider Example

This example demonstrates how to use the OpenAI embedding provider with Beluga AI.

## Prerequisites

- OpenAI API key (set as `OPENAI_API_KEY` environment variable)
- Go 1.21+

## Running the Example

```bash
export OPENAI_API_KEY="your-api-key-here"
go run main.go
```

## What This Example Shows

1. Creating an OpenAI embedder using the embedding factory
2. Configuring the provider with API key and model name
3. Getting embedding dimensions
4. Embedding single queries and multiple documents

## Configuration Options

- `APIKey`: Your OpenAI API key (required)
- `Model`: Embedding model to use (e.g., "text-embedding-3-small", "text-embedding-3-large")
- `BaseURL`: Optional custom API base URL
- `Timeout`: Request timeout duration
- `MaxRetries`: Maximum number of retry attempts

## See Also

- [Embeddings Package Documentation](../../../pkg/embeddings/README.md)
- [OpenAI Embedding Provider Documentation](../../../pkg/embeddings/providers/openai/README.md)
