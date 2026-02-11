---
title: "LLM Providers"
description: "Overview of all supported LLM providers in Beluga AI."
---

Beluga AI provides a unified `llm.ChatModel` interface across 22 LLM providers. Every provider registers itself via `init()`, so a blank import is sufficient to make it available through the registry.

This unified interface means you can switch providers by changing a single line of configuration -- your application code, middleware, and hooks work identically across all providers. Start with any provider during development and switch to another for production without code changes.

## How It Works

All providers implement the same interface:

```go
type ChatModel interface {
    Generate(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error)
    Stream(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) iter.Seq2[schema.StreamChunk, error]
    BindTools(tools []schema.ToolDefinition) ChatModel
    ModelID() string
}
```

You can instantiate any provider two ways:

**Via the registry** (recommended for dynamic configuration):

```go
import (
    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/llm"
    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

model, err := llm.New("openai", config.ProviderConfig{
    Model:  "gpt-4o",
    APIKey: os.Getenv("OPENAI_API_KEY"),
})
```

**Via direct construction** (for compile-time type safety):

```go
import "github.com/lookatitude/beluga-ai/llm/providers/openai"

model, err := openai.New(config.ProviderConfig{
    Model:  "gpt-4o",
    APIKey: os.Getenv("OPENAI_API_KEY"),
})
```

## Configuration

All providers accept `config.ProviderConfig`:

| Field     | Type              | Description                                    |
|-----------|-------------------|------------------------------------------------|
| `Provider`| `string`          | Registered provider name (e.g. `"openai"`)     |
| `APIKey`  | `string`          | Authentication key                             |
| `Model`   | `string`          | Model identifier (e.g. `"gpt-4o"`)            |
| `BaseURL` | `string`          | Override the default API endpoint              |
| `Timeout` | `time.Duration`   | Maximum request duration (default: 30s)        |
| `Options` | `map[string]any`  | Provider-specific key-value configuration      |

## Generation Options

All providers support the same set of generation options passed via functional options:

```go
resp, err := model.Generate(ctx, msgs,
    llm.WithTemperature(0.7),
    llm.WithMaxTokens(1024),
    llm.WithTopP(0.9),
    llm.WithStopSequences("END"),
    llm.WithToolChoice(llm.ToolChoiceAuto),
    llm.WithResponseFormat(llm.ResponseFormat{Type: "json_object"}),
)
```

## Provider Categories

### Direct API Providers

These providers use their vendor's native SDK and offer the deepest feature integration, including provider-specific capabilities like vision, prompt caching, and extended context windows:

| Provider | Registry Name | Description |
|----------|---------------|-------------|
| [OpenAI](/providers/llm/openai) | `openai` | GPT-4o, GPT-4, o1, o3 models |
| [Anthropic](/providers/llm/anthropic) | `anthropic` | Claude 4.5, Claude 4 models |
| [Google](/providers/llm/google) | `google` | Gemini 2.5, Gemini 2.0 models |
| [Azure OpenAI](/providers/llm/azure) | `azure` | OpenAI models hosted on Azure |
| [AWS Bedrock](/providers/llm/bedrock) | `bedrock` | Multi-provider models via AWS |
| [Mistral](/providers/llm/mistral) | `mistral` | Mistral Large, Codestral models |
| [Cohere](/providers/llm/cohere) | `cohere` | Command R+ models |

### OpenAI-Compatible Providers

These providers expose an OpenAI-compatible API and share a common implementation layer via Beluga's `internal/openaicompat` package. They all support streaming, tool calling, and structured output through the same code path:

| Provider | Registry Name | Description |
|----------|---------------|-------------|
| [Groq](/providers/llm/groq) | `groq` | Ultra-fast inference with LPU hardware |
| [Together AI](/providers/llm/together) | `together` | Open-source model hosting |
| [Fireworks AI](/providers/llm/fireworks) | `fireworks` | Fast inference for open models |
| [DeepSeek](/providers/llm/deepseek) | `deepseek` | DeepSeek V3, R1 reasoning models |
| [OpenRouter](/providers/llm/openrouter) | `openrouter` | Multi-provider routing gateway |
| [Perplexity](/providers/llm/perplexity) | `perplexity` | Search-augmented generation |
| [HuggingFace](/providers/llm/huggingface) | `huggingface` | Inference API for hosted models |
| [xAI](/providers/llm/xai) | `xai` | Grok models |
| [Qwen](/providers/llm/qwen) | `qwen` | Alibaba Qwen models via DashScope |
| [SambaNova](/providers/llm/sambanova) | `sambanova` | High-throughput inference |
| [Cerebras](/providers/llm/cerebras) | `cerebras` | Wafer-scale inference |
| [Ollama](/providers/llm/ollama) | `ollama` | Local model serving |

### Meta-Providers

These providers delegate to other providers or gateways, adding a routing or abstraction layer between your application and the underlying LLM service:

| Provider | Registry Name | Description |
|----------|---------------|-------------|
| [Llama](/providers/llm/llama) | `llama` | Meta Llama models via any backend |
| [Bifrost](/providers/llm/bifrost) | `bifrost` | LLM gateway with load balancing |
| [LiteLLM](/providers/llm/litellm) | `litellm` | Universal LLM proxy (100+ models) |

## Choosing a Provider

| Use Case | Recommended Provider | Why |
|---|---|---|
| General-purpose default | [OpenAI](/providers/llm/openai) | Broadest ecosystem, mature tooling |
| Strong reasoning and safety | [Anthropic](/providers/llm/anthropic) | Large context, prompt caching |
| Multimodal (text + images + video) | [Google](/providers/llm/google) | Long context, Google Cloud integration |
| Enterprise Azure compliance | [Azure OpenAI](/providers/llm/azure) | Private networking, AAD, SLAs |
| AWS-native deployment | [AWS Bedrock](/providers/llm/bedrock) | IAM roles, multi-provider catalog |
| Lowest inference latency | [Groq](/providers/llm/groq), [Cerebras](/providers/llm/cerebras) | Custom hardware, fastest tokens/sec |
| Local/offline development | [Ollama](/providers/llm/ollama) | No API key, no network required |
| Search-augmented answers | [Perplexity](/providers/llm/perplexity) | Built-in web search |
| Model comparison and evaluation | [OpenRouter](/providers/llm/openrouter) | Single API key, hundreds of models |
| Infrastructure-level LLM management | [LiteLLM](/providers/llm/litellm) | Spend tracking, rate limiting, proxy |

## Middleware

All providers support the same middleware for cross-cutting concerns:

```go
model = llm.ApplyMiddleware(model,
    llm.WithLogging(logger),
    llm.WithFallback(backupModel),
)
```

See the [LLM middleware guide](/guides/middleware) for details.
