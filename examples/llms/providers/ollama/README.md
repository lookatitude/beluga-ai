# Ollama LLM Provider Example

This example demonstrates how to use the Ollama LLM provider with Beluga AI for local LLM inference.

## Prerequisites

- Ollama installed and running locally
- A model pulled (e.g., `ollama pull llama2`)
- Go 1.21+

## Running the Example

```bash
# Make sure Ollama is running
ollama serve

# In another terminal, run the example
go run main.go
```

## What This Example Shows

1. Creating an Ollama provider using the LLM factory
2. Configuring the provider with model name (no API key needed)
3. Generating text responses using local Ollama models

## Configuration Options

- `ModelName`: Model to use (e.g., "llama2", "mistral", "codellama")
- `ProviderSpecific["base_url"]`: Optional custom Ollama server URL (defaults to "http://localhost:11434")

## Available Models

You can use any model available in Ollama. Common models include:
- `llama2` - Meta's Llama 2
- `mistral` - Mistral AI model
- `codellama` - Code-focused Llama variant
- `phi` - Microsoft's Phi model

## See Also

- [LLM Package Documentation](../../../pkg/llms/README.md)
- [Ollama Provider Documentation](../../../pkg/llms/providers/ollama/README.md)
