---
title: LLM Providers
description: Configure and use language model providers with Beluga AI, including OpenAI, Anthropic, Google, Ollama, and 18 more.
sidebar:
  order: 0
---

Choosing an LLM provider involves tradeoffs between cost, latency, capability, data residency, and vendor lock-in. Beluga AI supports 22 LLM providers through a unified `ChatModel` interface so you can evaluate these tradeoffs without rewriting application code. All providers register via `init()` and are created through the same registry pattern -- switching between providers requires changing only an import path and a configuration struct.

This design also enables multi-provider strategies: route complex queries to Claude or GPT-4o while sending simpler tasks to faster, cheaper models like Groq or local Ollama instances.

## Provider Overview

| Provider | Registry Name | Models | Import Path |
|----------|--------------|--------|-------------|
| OpenAI | `openai` | GPT-4o, GPT-4, o1, o3 | `llm/providers/openai` |
| Anthropic | `anthropic` | Claude Opus 4, Sonnet 4.5, Haiku | `llm/providers/anthropic` |
| Google | `google` | Gemini 2.5, 2.0, 1.5 | `llm/providers/google` |
| Ollama | `ollama` | Llama 3, Mistral, Phi, any local | `llm/providers/ollama` |
| AWS Bedrock | `bedrock` | Claude, Titan, Llama via AWS | `llm/providers/bedrock` |
| Azure OpenAI | `azure` | GPT-4o, GPT-4 via Azure | `llm/providers/azure` |
| Groq | `groq` | Llama, Mixtral on Groq LPUs | `llm/providers/groq` |
| Together AI | `together` | Open-source models hosted | `llm/providers/together` |
| Fireworks | `fireworks` | Open-source models, fast inference | `llm/providers/fireworks` |
| Mistral | `mistral` | Mistral Large, Medium, Small | `llm/providers/mistral` |
| Cohere | `cohere` | Command R+, Command R | `llm/providers/cohere` |
| DeepSeek | `deepseek` | DeepSeek V3, R1 | `llm/providers/deepseek` |
| xAI | `xai` | Grok-2, Grok-3 | `llm/providers/xai` |
| Perplexity | `perplexity` | Sonar models with search | `llm/providers/perplexity` |
| OpenRouter | `openrouter` | Multi-provider routing | `llm/providers/openrouter` |
| Qwen | `qwen` | Qwen-2.5, QwQ | `llm/providers/qwen` |
| Cerebras | `cerebras` | Llama on Cerebras hardware | `llm/providers/cerebras` |
| SambaNova | `sambanova` | Llama, Mistral on SambaNova | `llm/providers/sambanova` |
| HuggingFace | `huggingface` | Inference API models | `llm/providers/huggingface` |
| LiteLLM | `litellm` | Proxy for 100+ providers | `llm/providers/litellm` |
| Llama.cpp | `llama` | Local GGUF models | `llm/providers/llama` |
| Bifrost | `bifrost` | Multi-provider gateway | `llm/providers/bifrost` |

## Common Pattern

Every provider follows the same three-step pattern:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"

    // 1. Import the provider — its init() registers with the llm registry
    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

func main() {
    ctx := context.Background()

    // 2. Create the model via the registry
    model, err := llm.New("openai", config.ProviderConfig{
        Model:  "gpt-4o",
        APIKey: "sk-...",
    })
    if err != nil {
        log.Fatal(err)
    }

    // 3. Use the unified ChatModel interface
    resp, err := model.Generate(ctx, []schema.Message{
        schema.NewUserMessage(schema.Text("What is Go?")),
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(resp.Content())
}
```

## OpenAI

OpenAI is the default provider for GPT-4o, o1, and o3 models.

```bash
export OPENAI_API_KEY="sk-..."
```

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/openai"

model, err := llm.New("openai", config.ProviderConfig{
    Model:  "gpt-4o",
    APIKey: os.Getenv("OPENAI_API_KEY"),
})
```

**Available models**: `gpt-4o`, `gpt-4o-mini`, `gpt-4-turbo`, `o1`, `o1-mini`, `o3-mini`

**Options**:
| Key | Type | Description |
|-----|------|-------------|
| `temperature` | `float64` | Sampling temperature (0.0-2.0) |
| `max_tokens` | `float64` | Maximum response tokens |
| `top_p` | `float64` | Nucleus sampling |
| `frequency_penalty` | `float64` | Repetition penalty |

```go
model, err := llm.New("openai", config.ProviderConfig{
    Model:  "gpt-4o",
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Options: map[string]any{
        "temperature": 0.7,
        "max_tokens":  4096.0,
    },
})
```

## Anthropic

Anthropic provides the Claude family of models with native tool use and extended context windows.

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
```

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/anthropic"

model, err := llm.New("anthropic", config.ProviderConfig{
    Model:  "claude-sonnet-4-5-20250929",
    APIKey: os.Getenv("ANTHROPIC_API_KEY"),
})
```

**Available models**: `claude-opus-4-6`, `claude-sonnet-4-5-20250929`, `claude-haiku-4-5-20251001`

Anthropic uses a dedicated SDK rather than the OpenAI-compatible wrapper, which provides native support for Claude's extended thinking, tool use, and content block streaming.

## Google Gemini

Google provides Gemini models with multimodal capabilities.

```bash
export GOOGLE_API_KEY="AIza..."
```

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/google"

model, err := llm.New("google", config.ProviderConfig{
    Model:  "gemini-2.5-pro",
    APIKey: os.Getenv("GOOGLE_API_KEY"),
})
```

**Available models**: `gemini-2.5-pro`, `gemini-2.5-flash`, `gemini-2.0-flash`, `gemini-1.5-pro`

## Ollama (Local Models)

Ollama runs open-source models locally with no API key required.

```bash
# Start Ollama server
ollama serve
ollama pull llama3.1
```

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/ollama"

model, err := llm.New("ollama", config.ProviderConfig{
    Model:   "llama3.1",
    BaseURL: "http://localhost:11434/v1",
})
```

Ollama uses the OpenAI-compatible API format, so all standard options apply. No API key is needed for local usage.

## AWS Bedrock

AWS Bedrock provides access to multiple model families through your AWS account.

```bash
export AWS_ACCESS_KEY_ID="..."
export AWS_SECRET_ACCESS_KEY="..."
export AWS_REGION="us-east-1"
```

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/bedrock"

model, err := llm.New("bedrock", config.ProviderConfig{
    Model: "anthropic.claude-sonnet-4-5-20250929-v1:0",
    Options: map[string]any{
        "region": "us-east-1",
    },
})
```

Bedrock reads AWS credentials from the standard AWS credential chain (environment variables, shared config, IAM role).

## Azure OpenAI

Azure OpenAI provides GPT models via your Azure subscription with enterprise data residency.

```bash
export AZURE_OPENAI_API_KEY="..."
export AZURE_OPENAI_ENDPOINT="https://my-resource.openai.azure.com"
```

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/azure"

model, err := llm.New("azure", config.ProviderConfig{
    Model:   "gpt-4o",
    APIKey:  os.Getenv("AZURE_OPENAI_API_KEY"),
    BaseURL: os.Getenv("AZURE_OPENAI_ENDPOINT"),
    Options: map[string]any{
        "api_version":    "2024-06-01",
        "deployment_name": "my-gpt4o-deployment",
    },
})
```

## Groq

Groq provides ultra-fast inference on custom LPU hardware.

```bash
export GROQ_API_KEY="gsk_..."
```

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/groq"

model, err := llm.New("groq", config.ProviderConfig{
    Model:  "llama-3.1-70b-versatile",
    APIKey: os.Getenv("GROQ_API_KEY"),
})
```

## Together AI

Together AI hosts open-source models with OpenAI-compatible API.

```bash
export TOGETHER_API_KEY="..."
```

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/together"

model, err := llm.New("together", config.ProviderConfig{
    Model:  "meta-llama/Llama-3.1-70B-Instruct-Turbo",
    APIKey: os.Getenv("TOGETHER_API_KEY"),
})
```

## Fireworks

Fireworks provides fast inference for open-source models.

```bash
export FIREWORKS_API_KEY="..."
```

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/fireworks"

model, err := llm.New("fireworks", config.ProviderConfig{
    Model:  "accounts/fireworks/models/llama-v3p1-70b-instruct",
    APIKey: os.Getenv("FIREWORKS_API_KEY"),
})
```

## Mistral

Mistral AI provides European-hosted models.

```bash
export MISTRAL_API_KEY="..."
```

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/mistral"

model, err := llm.New("mistral", config.ProviderConfig{
    Model:  "mistral-large-latest",
    APIKey: os.Getenv("MISTRAL_API_KEY"),
})
```

## DeepSeek

DeepSeek provides high-performance reasoning models.

```bash
export DEEPSEEK_API_KEY="..."
```

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/deepseek"

model, err := llm.New("deepseek", config.ProviderConfig{
    Model:  "deepseek-chat",
    APIKey: os.Getenv("DEEPSEEK_API_KEY"),
})
```

## Streaming

All providers support streaming via `iter.Seq2`:

```go
for chunk, err := range model.Stream(ctx, messages) {
    if err != nil {
        log.Fatal(err)
    }
    fmt.Print(chunk.Delta)
}
```

## Middleware

Add cross-cutting concerns to any provider using middleware:

```go
model = llm.ApplyMiddleware(model,
    llm.WithLogging(slog.Default()),
    llm.WithRetry(3),
)
```

Middleware is composable and applies to Generate, Stream, and BindTools uniformly across all providers.

## Discovering Providers at Runtime

List all registered providers:

```go
providers := llm.List()
fmt.Println("Available providers:", providers)
```

This returns all providers whose import-side-effect `init()` has run. Use this for dynamic provider selection in configuration-driven applications.

## Configuration Reference

All providers accept `config.ProviderConfig`:

| Field | Type | Description |
|-------|------|-------------|
| `Provider` | `string` | Registry name (e.g. `"openai"`) |
| `APIKey` | `string` | Authentication key |
| `Model` | `string` | Model identifier |
| `BaseURL` | `string` | Override default API endpoint |
| `Timeout` | `time.Duration` | Request timeout (default: 30s) |
| `Options` | `map[string]any` | Provider-specific options |

## OpenAI-Compatible Providers

Most providers (18 of 22) use Beluga's shared `internal/openaicompat` package, which means they accept the same `Options` keys: `temperature`, `max_tokens`, `top_p`, `frequency_penalty`, `presence_penalty`, `stop`, and `response_format`. The only providers with custom implementations are Anthropic, Google, Bedrock, and Bifrost.
