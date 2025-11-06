# LLM Provider Documentation

This directory contains detailed guides for each LLM provider supported by Beluga AI Framework.

## Available Providers

- [OpenAI](./openai.md) - GPT-3.5, GPT-4, and variants
- [Anthropic](./anthropic.md) - Claude models
- [AWS Bedrock](./bedrock.md) - AWS foundation models
- [Ollama](./ollama.md) - Local models
- [Provider Comparison](./comparison.md) - Compare all providers

## Quick Start

### OpenAI

```go
config := llms.NewConfig(
    llms.WithProvider("openai"),
    llms.WithModelName("gpt-4"),
    llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
)
```

### Anthropic

```go
config := llms.NewConfig(
    llms.WithProvider("anthropic"),
    llms.WithModelName("claude-3-sonnet-20240229"),
    llms.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
)
```

### Ollama

```go
config := llms.NewConfig(
    llms.WithProvider("ollama"),
    llms.WithModelName("llama2"),
    llms.WithBaseURL("http://localhost:11434"),
)
```

## Choosing a Provider

See [Provider Comparison](./comparison.md) for detailed guidance.

---

**Next:** Read provider-specific guides or [compare providers](./comparison.md)

