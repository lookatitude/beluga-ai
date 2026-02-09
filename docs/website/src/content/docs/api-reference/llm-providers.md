---
title: "LLM Providers"
description: "All LLM provider implementations: OpenAI, Anthropic, Google, Ollama, Bedrock, and more"
---

## anthropic

```go
import "github.com/lookatitude/beluga-ai/llm/providers/anthropic"
```

Package anthropic provides the Anthropic (Claude) LLM provider for the
Beluga AI framework.

It implements the [llm.ChatModel] interface using the anthropic-sdk-go SDK,
with native support for the Anthropic Messages API including streaming,
tool use, vision (image inputs), and prompt caching.

## Registration

The provider registers itself as "anthropic" via init(). Import the package
for side effects to make it available through the llm registry:

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/anthropic"
```

## Usage

```go
model, err := llm.New("anthropic", config.ProviderConfig{
    Model:  "claude-sonnet-4-5-20250929",
    APIKey: "sk-ant-...",
})
if err != nil {
    log.Fatal(err)
}

resp, err := model.Generate(ctx, []schema.Message{
    schema.NewHumanMessage("Explain quantum computing"),
})
```

## Configuration

The following [config.ProviderConfig] fields are used:

- Model: the Anthropic model name (required; e.g. "claude-sonnet-4-5-20250929", "claude-haiku-4-5-20251001")
- APIKey: the Anthropic API key
- BaseURL: optional, defaults to Anthropic's API endpoint
- Timeout: optional request timeout

## Key Types

- [Model]: the ChatModel implementation with Generate, Stream, BindTools, and ModelID methods
- [New]: constructor that creates a Model from a [config.ProviderConfig]

## Implementation Notes

Unlike the OpenAI-compatible providers, this package implements the full
Anthropic Messages API natively. System messages are extracted and passed
as the dedicated system parameter. Tool use follows the Anthropic tool_use
content block format. The default max tokens is 4096 when not specified.

---

## azure

```go
import "github.com/lookatitude/beluga-ai/llm/providers/azure"
```

Package azure provides the Azure OpenAI LLM provider for the Beluga AI
framework.

Azure OpenAI uses a different authentication scheme (api-key header) and URL
structure (per-deployment endpoints with api-version query parameter) compared
to the standard OpenAI API, but the request/response format is otherwise
identical. This provider handles the Azure-specific authentication and URL
rewriting transparently using the shared openaicompat package.

## Registration

The provider registers itself as "azure" via init(). Import the package
for side effects to make it available through the llm registry:

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/azure"
```

## Usage

```go
model, err := llm.New("azure", config.ProviderConfig{
    Model:   "gpt-4o",
    APIKey:  "...",
    BaseURL: "https://myresource.openai.azure.com/openai/deployments/my-gpt4o",
    Options: map[string]any{"api_version": "2024-10-21"},
})
if err != nil {
    log.Fatal(err)
}

resp, err := model.Generate(ctx, []schema.Message{
    schema.NewHumanMessage("Hello from Azure!"),
})
```

## Configuration

The following [config.ProviderConfig] fields are used:

- Model: the deployment model name (defaults to "gpt-4o")
- APIKey: the Azure OpenAI API key (required)
- BaseURL: the Azure deployment endpoint (required; format: https://{resource}.openai.azure.com/openai/deployments/{deployment})
- Options["api_version"]: the Azure API version (defaults to "2024-10-21")

## Direct Construction

Use `New` to create a ChatModel directly without going through the registry.

---

## bedrock

```go
import "github.com/lookatitude/beluga-ai/llm/providers/bedrock"
```

Package bedrock provides the AWS Bedrock LLM provider for the Beluga AI
framework.

It implements the [llm.ChatModel] interface using the AWS SDK v2 Bedrock
Runtime Converse API, supporting all models available through Amazon
Bedrock including Anthropic Claude, Meta Llama, Mistral, and Amazon Titan.

## Registration

The provider registers itself as "bedrock" via init(). Import the package
for side effects to make it available through the llm registry:

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/bedrock"
```

## Usage

```go
model, err := llm.New("bedrock", config.ProviderConfig{
    Model:   "us.anthropic.claude-sonnet-4-5-20250929-v1:0",
    Options: map[string]any{"region": "us-east-1"},
})
if err != nil {
    log.Fatal(err)
}

resp, err := model.Generate(ctx, []schema.Message{
    schema.NewHumanMessage("Analyze this data"),
})
```

## Configuration

The following [config.ProviderConfig] fields are used:

- Model: the Bedrock model ID (required; e.g. "us.anthropic.claude-sonnet-4-5-20250929-v1:0")
- APIKey: optional AWS access key ID (uses default credentials if unset)
- BaseURL: optional custom Bedrock endpoint
- Options["region"]: AWS region (defaults to "us-east-1")
- Options["secret_key"]: AWS secret access key (used with APIKey)

## Key Types

- [Model]: the ChatModel implementation using the Bedrock Converse API
- [ConverseAPI]: interface for the subset of bedrockruntime.Client methods used, enabling mock injection for tests
- [New]: constructor from [config.ProviderConfig]
- [NewWithClient]: constructor accepting a custom [ConverseAPI] implementation for testing

## Implementation Notes

This package uses the Bedrock Converse API (not InvokeModel), which provides
a unified interface across all Bedrock-hosted models. Authentication uses
the standard AWS credential chain. System messages are passed as dedicated
system content blocks. Tool use follows the Bedrock tool specification format.

---

## bifrost

```go
import "github.com/lookatitude/beluga-ai/llm/providers/bifrost"
```

Package bifrost provides a ChatModel backed by a Bifrost gateway for the
Beluga AI framework.

Bifrost is an OpenAI-compatible proxy that routes requests to multiple LLM
providers with load balancing and failover. This provider is a thin wrapper
around the shared openaicompat package pointed at a Bifrost proxy endpoint.

## Registration

The provider registers itself as "bifrost" via init(). Import the package
for side effects to make it available through the llm registry:

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/bifrost"
```

## Usage

```go
model, err := llm.New("bifrost", config.ProviderConfig{
    Model:   "gpt-4o",
    APIKey:  "sk-...",
    BaseURL: "http://localhost:8080/v1",
})
if err != nil {
    log.Fatal(err)
}

resp, err := model.Generate(ctx, []schema.Message{
    schema.NewHumanMessage("Hello from Bifrost!"),
})
```

## Configuration

The following [config.ProviderConfig] fields are used:

- Model: the model name (required)
- APIKey: the API key for the Bifrost proxy
- BaseURL: the Bifrost proxy URL (required; no default)

## Direct Construction

Use `New` to create a ChatModel directly without going through the registry.

---

## cerebras

```go
import "github.com/lookatitude/beluga-ai/llm/providers/cerebras"
```

Package cerebras provides the Cerebras LLM provider for the Beluga AI
framework.

Cerebras exposes an OpenAI-compatible API running on Wafer-Scale Engine
hardware optimized for fast inference. This provider is a thin wrapper
around the shared openaicompat package with Cerebras' base URL.

## Registration

The provider registers itself as "cerebras" via init(). Import the package
for side effects to make it available through the llm registry:

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/cerebras"
```

## Usage

```go
model, err := llm.New("cerebras", config.ProviderConfig{
    Model:  "llama-3.3-70b",
    APIKey: "csk-...",
})
if err != nil {
    log.Fatal(err)
}

resp, err := model.Generate(ctx, []schema.Message{
    schema.NewHumanMessage("Explain wafer-scale computing"),
})
```

## Configuration

The following [config.ProviderConfig] fields are used:

- Model: the Cerebras model name (e.g. "llama-3.3-70b")
- APIKey: the Cerebras API key
- BaseURL: optional, defaults to "https://api.cerebras.ai/v1"

## Direct Construction

Use `New` to create a ChatModel directly without going through the registry.

---

## cohere

```go
import "github.com/lookatitude/beluga-ai/llm/providers/cohere"
```

Package cohere provides the Cohere LLM provider for the Beluga AI framework.

It implements the [llm.ChatModel] interface using the official Cohere Go SDK
(v2), with native support for the Cohere Chat API including streaming, tool
use, and the unique Cohere message format.

Cohere uses a different message structure than OpenAI: the last user message
becomes the "message" field, system messages go into the "preamble", and all
prior messages become "chat_history". This provider handles the mapping
transparently.

## Registration

The provider registers itself as "cohere" via init(). Import the package
for side effects to make it available through the llm registry:

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/cohere"
```

## Usage

```go
model, err := llm.New("cohere", config.ProviderConfig{
    Model:  "command-r-plus",
    APIKey: "...",
})
if err != nil {
    log.Fatal(err)
}

resp, err := model.Generate(ctx, []schema.Message{
    schema.NewHumanMessage("Explain RAG in three sentences"),
})
```

## Configuration

The following [config.ProviderConfig] fields are used:

- Model: the Cohere model name (defaults to "command-r-plus")
- APIKey: the Cohere API key (required)
- BaseURL: optional, overrides the default Cohere API endpoint

## Key Types

- [Model]: the ChatModel implementation using the Cohere SDK
- [New]: constructor from [config.ProviderConfig]

## Implementation Notes

This package uses the Cohere Go SDK directly rather than the OpenAI
compatibility layer. Tool definitions are converted to Cohere's
ParameterDefinitions format. Streaming uses Cohere's event-based
stream with TextGeneration, ToolCallsGeneration, and StreamEnd events.

---

## deepseek

```go
import "github.com/lookatitude/beluga-ai/llm/providers/deepseek"
```

Package deepseek provides the DeepSeek LLM provider for the Beluga AI
framework.

DeepSeek exposes an OpenAI-compatible API, so this provider is a thin
wrapper around the shared openaicompat package with DeepSeek's base URL.
It supports DeepSeek's chat and reasoning models.

## Registration

The provider registers itself as "deepseek" via init(). Import the package
for side effects to make it available through the llm registry:

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/deepseek"
```

## Usage

```go
model, err := llm.New("deepseek", config.ProviderConfig{
    Model:  "deepseek-chat",
    APIKey: "sk-...",
})
if err != nil {
    log.Fatal(err)
}

resp, err := model.Generate(ctx, []schema.Message{
    schema.NewHumanMessage("Solve this math problem"),
})
```

## Configuration

The following [config.ProviderConfig] fields are used:

- Model: the DeepSeek model name (defaults to "deepseek-chat")
- APIKey: the DeepSeek API key
- BaseURL: optional, defaults to "https://api.deepseek.com/v1"

## Direct Construction

Use `New` to create a ChatModel directly without going through the registry.

---

## fireworks

```go
import "github.com/lookatitude/beluga-ai/llm/providers/fireworks"
```

Package fireworks provides the Fireworks AI LLM provider for the Beluga AI
framework.

Fireworks AI exposes an OpenAI-compatible API optimized for fast inference
of open-source models. This provider is a thin wrapper around the shared
openaicompat package with Fireworks' base URL.

## Registration

The provider registers itself as "fireworks" via init(). Import the package
for side effects to make it available through the llm registry:

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/fireworks"
```

## Usage

```go
model, err := llm.New("fireworks", config.ProviderConfig{
    Model:  "accounts/fireworks/models/llama-v3p1-70b-instruct",
    APIKey: "fw_...",
})
if err != nil {
    log.Fatal(err)
}

resp, err := model.Generate(ctx, []schema.Message{
    schema.NewHumanMessage("Write a function to sort a list"),
})
```

## Configuration

The following [config.ProviderConfig] fields are used:

- Model: the Fireworks model path (defaults to "accounts/fireworks/models/llama-v3p1-70b-instruct")
- APIKey: the Fireworks API key
- BaseURL: optional, defaults to "https://api.fireworks.ai/inference/v1"

## Direct Construction

Use `New` to create a ChatModel directly without going through the registry.

---

## google

```go
import "github.com/lookatitude/beluga-ai/llm/providers/google"
```

Package google provides the Google Gemini LLM provider for the Beluga AI
framework.

It implements the [llm.ChatModel] interface using the google.golang.org/genai
SDK, with native support for the Gemini API including streaming, function
calling, vision (image and file inputs), and system instructions.

## Registration

The provider registers itself as "google" via init(). Import the package
for side effects to make it available through the llm registry:

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/google"
```

## Usage

```go
model, err := llm.New("google", config.ProviderConfig{
    Model:  "gemini-2.5-flash",
    APIKey: "...",
})
if err != nil {
    log.Fatal(err)
}

resp, err := model.Generate(ctx, []schema.Message{
    schema.NewHumanMessage("Summarize this document"),
})
```

## Configuration

The following [config.ProviderConfig] fields are used:

- Model: the Gemini model name (required; e.g. "gemini-2.5-flash", "gemini-2.5-pro")
- APIKey: the Google AI API key
- BaseURL: optional, overrides the default Gemini API endpoint

## Key Types

- [Model]: the ChatModel implementation
- [New]: constructor from [config.ProviderConfig]
- [NewWithHTTPClient]: constructor accepting a custom *http.Client for testing

## Implementation Notes

This package implements the Gemini API natively rather than using the
OpenAI compatibility layer. System messages are passed as system
instructions. Tool calling uses the Gemini function calling format with
FunctionDeclaration and FunctionResponse parts. Image inputs support
both inline data (base64) and file URIs.

---

## groq

```go
import "github.com/lookatitude/beluga-ai/llm/providers/groq"
```

Package groq provides the Groq LLM provider for the Beluga AI framework.

Groq exposes an OpenAI-compatible API optimized for fast inference on
custom LPU hardware. This provider is a thin wrapper around the shared
openaicompat package with Groq's base URL.

## Registration

The provider registers itself as "groq" via init(). Import the package
for side effects to make it available through the llm registry:

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/groq"
```

## Usage

```go
model, err := llm.New("groq", config.ProviderConfig{
    Model:  "llama-3.3-70b-versatile",
    APIKey: "gsk_...",
})
if err != nil {
    log.Fatal(err)
}

resp, err := model.Generate(ctx, []schema.Message{
    schema.NewHumanMessage("Explain Groq's LPU architecture"),
})
```

## Configuration

The following [config.ProviderConfig] fields are used:

- Model: the Groq model name (e.g. "llama-3.3-70b-versatile", "mixtral-8x7b-32768")
- APIKey: the Groq API key
- BaseURL: optional, defaults to "https://api.groq.com/openai/v1"

## Direct Construction

Use `New` to create a ChatModel directly without going through the registry.

---

## huggingface

```go
import "github.com/lookatitude/beluga-ai/llm/providers/huggingface"
```

Package huggingface provides the HuggingFace Inference API LLM provider for
the Beluga AI framework.

HuggingFace exposes an OpenAI-compatible chat completions endpoint through
its Inference API, so this provider is a thin wrapper around the shared
openaicompat package. It supports any model hosted on HuggingFace's
inference infrastructure.

## Registration

The provider registers itself as "huggingface" via init(). Import the package
for side effects to make it available through the llm registry:

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/huggingface"
```

## Usage

```go
model, err := llm.New("huggingface", config.ProviderConfig{
    Model:  "meta-llama/Meta-Llama-3.1-70B-Instruct",
    APIKey: "hf_...",
})
if err != nil {
    log.Fatal(err)
}

resp, err := model.Generate(ctx, []schema.Message{
    schema.NewHumanMessage("What is transfer learning?"),
})
```

## Configuration

The following [config.ProviderConfig] fields are used:

- Model: the HuggingFace model ID (e.g. "meta-llama/Meta-Llama-3.1-70B-Instruct")
- APIKey: the HuggingFace API token
- BaseURL: optional, defaults to "https://api-inference.huggingface.co/v1"

## Direct Construction

Use `New` to create a ChatModel directly without going through the registry.

---

## litellm

```go
import "github.com/lookatitude/beluga-ai/llm/providers/litellm"
```

Package litellm provides a ChatModel backed by a LiteLLM gateway for the
Beluga AI framework.

LiteLLM (https://litellm.ai) is a proxy that exposes an OpenAI-compatible
API in front of 100+ LLM providers. This provider is a thin wrapper around
the shared openaicompat package pointed at the LiteLLM proxy endpoint.

## Registration

The provider registers itself as "litellm" via init(). Import the package
for side effects to make it available through the llm registry:

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/litellm"
```

## Usage

```go
model, err := llm.New("litellm", config.ProviderConfig{
    Model:   "gpt-4o",
    APIKey:  "sk-...",
    BaseURL: "http://localhost:4000/v1",
})
if err != nil {
    log.Fatal(err)
}

resp, err := model.Generate(ctx, []schema.Message{
    schema.NewHumanMessage("Hello from LiteLLM!"),
})
```

## Configuration

The following [config.ProviderConfig] fields are used:

- Model: the model name as configured in LiteLLM (defaults to "gpt-4o")
- APIKey: the LiteLLM proxy API key
- BaseURL: the LiteLLM proxy URL (defaults to "http://localhost:4000/v1")

## Direct Construction

Use `New` to create a ChatModel directly without going through the registry.

---

## llama

```go
import "github.com/lookatitude/beluga-ai/llm/providers/llama"
```

Package llama provides a Meta Llama model provider for the Beluga AI
framework.

Since Meta does not offer a direct API for Llama models, this provider
delegates to one of the available hosting backends that serve Llama models:
Together, Fireworks, Groq, SambaNova, Cerebras, or Ollama. The backend
is selected via the "backend" option in [config.ProviderConfig].Options.

## Registration

The provider registers itself as "llama" via init(). Import the package
for side effects to make it available through the llm registry:

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/llama"
```

## Usage

```go
model, err := llm.New("llama", config.ProviderConfig{
    Model:   "meta-llama/Llama-3.3-70B-Instruct",
    APIKey:  "...",
    Options: map[string]any{"backend": "together"},
})
if err != nil {
    log.Fatal(err)
}

resp, err := model.Generate(ctx, []schema.Message{
    schema.NewHumanMessage("Explain the Llama architecture"),
})
```

## Configuration

The following [config.ProviderConfig] fields are used:

- Model: the Llama model name (required; e.g. "meta-llama/Llama-3.3-70B-Instruct")
- APIKey: the API key for the chosen backend
- BaseURL: optional, overrides the backend's default URL
- Options["backend"]: hosting backend to use (defaults to "together"; supported: "together", "fireworks", "groq", "sambanova", "cerebras", "ollama")

## Direct Construction

Use `New` to create a ChatModel directly without going through the registry.
The appropriate backend provider must be imported for delegation to work.

---

## mistral

```go
import "github.com/lookatitude/beluga-ai/llm/providers/mistral"
```

Package mistral provides the Mistral AI LLM provider for the Beluga AI
framework.

It implements the [llm.ChatModel] interface using the official Mistral Go
SDK, with native support for Mistral's chat API including streaming, tool
use, and JSON output mode.

## Registration

The provider registers itself as "mistral" via init(). Import the package
for side effects to make it available through the llm registry:

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/mistral"
```

## Usage

```go
model, err := llm.New("mistral", config.ProviderConfig{
    Model:  "mistral-large-latest",
    APIKey: "...",
})
if err != nil {
    log.Fatal(err)
}

resp, err := model.Generate(ctx, []schema.Message{
    schema.NewHumanMessage("Write a haiku about Go"),
})
```

## Configuration

The following [config.ProviderConfig] fields are used:

- Model: the Mistral model name (defaults to "mistral-large-latest")
- APIKey: the Mistral API key (required)
- BaseURL: optional, defaults to "https://api.mistral.ai"
- Timeout: optional request timeout (defaults to 30s)

## Key Types

- [Model]: the ChatModel implementation using the Mistral SDK
- [New]: constructor from [config.ProviderConfig]

## Implementation Notes

This package uses the Mistral Go SDK directly rather than the OpenAI
compatibility layer. Streaming is channel-based in the Mistral SDK and
is adapted to the iter.Seq2 pattern. Tool definitions are converted to
the Mistral function calling format.

---

## ollama

```go
import "github.com/lookatitude/beluga-ai/llm/providers/ollama"
```

Package ollama provides the Ollama LLM provider for the Beluga AI framework.

Ollama exposes an OpenAI-compatible API, so this provider is a thin wrapper
around the shared openaicompat package pointed at Ollama's local endpoint.
It supports all models available through Ollama including Llama, Mistral,
Phi, Gemma, and other open-source models.

## Registration

The provider registers itself as "ollama" via init(). Import the package
for side effects to make it available through the llm registry:

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/ollama"
```

## Usage

```go
model, err := llm.New("ollama", config.ProviderConfig{
    Model: "llama3.2",
})
if err != nil {
    log.Fatal(err)
}

resp, err := model.Generate(ctx, []schema.Message{
    schema.NewHumanMessage("Hello!"),
})
```

## Configuration

The following [config.ProviderConfig] fields are used:

- Model: the Ollama model name (e.g. "llama3.2", "mistral", "phi3")
- BaseURL: optional, defaults to "http://localhost:11434/v1"
- APIKey: optional, defaults to "ollama" (Ollama does not require authentication)

## Direct Construction

Use `New` to create a ChatModel directly without going through the registry.

---

## openai

```go
import "github.com/lookatitude/beluga-ai/llm/providers/openai"
```

Package openai provides the OpenAI LLM provider for the Beluga AI framework.

It implements the [llm.ChatModel] interface using the openai-go SDK via the
shared openaicompat package. This provider supports all OpenAI chat models
including GPT-4o, GPT-4, and GPT-3.5 Turbo, with full support for
streaming, tool calling, structured output, and multimodal inputs.

## Registration

The provider registers itself as "openai" via init(). Import the package
for side effects to make it available through the llm registry:

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
```

## Usage

```go
model, err := llm.New("openai", config.ProviderConfig{
    Model:  "gpt-4o",
    APIKey: "sk-...",
})
if err != nil {
    log.Fatal(err)
}

resp, err := model.Generate(ctx, []schema.Message{
    schema.NewHumanMessage("Hello, world!"),
})
```

## Configuration

The following [config.ProviderConfig] fields are used:

- Model: the OpenAI model name (e.g. "gpt-4o", "gpt-4o-mini")
- APIKey: the OpenAI API key
- BaseURL: optional, defaults to "https://api.openai.com/v1"

## Direct Construction

Use `New` to create a ChatModel directly without going through the registry.

---

## openrouter

```go
import "github.com/lookatitude/beluga-ai/llm/providers/openrouter"
```

Package openrouter provides the OpenRouter LLM provider for the Beluga AI
framework.

OpenRouter exposes an OpenAI-compatible API that routes requests to many
different model providers, enabling access to hundreds of models through a
single API key. This provider is a thin wrapper around the shared openaicompat
package with OpenRouter's base URL.

## Registration

The provider registers itself as "openrouter" via init(). Import the package
for side effects to make it available through the llm registry:

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/openrouter"
```

## Usage

```go
model, err := llm.New("openrouter", config.ProviderConfig{
    Model:  "anthropic/claude-sonnet-4-5-20250929",
    APIKey: "sk-or-...",
})
if err != nil {
    log.Fatal(err)
}

resp, err := model.Generate(ctx, []schema.Message{
    schema.NewHumanMessage("Compare GPT-4 and Claude"),
})
```

## Configuration

The following [config.ProviderConfig] fields are used:

- Model: the model path in provider/model format (e.g. "anthropic/claude-sonnet-4-5-20250929", "openai/gpt-4o")
- APIKey: the OpenRouter API key
- BaseURL: optional, defaults to "https://openrouter.ai/api/v1"

## Direct Construction

Use `New` to create a ChatModel directly without going through the registry.

---

## perplexity

```go
import "github.com/lookatitude/beluga-ai/llm/providers/perplexity"
```

Package perplexity provides the Perplexity LLM provider for the Beluga AI
framework.

Perplexity exposes an OpenAI-compatible API with models optimized for search
and retrieval-augmented generation. This provider is a thin wrapper around
the shared openaicompat package with Perplexity's base URL.

## Registration

The provider registers itself as "perplexity" via init(). Import the package
for side effects to make it available through the llm registry:

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/perplexity"
```

## Usage

```go
model, err := llm.New("perplexity", config.ProviderConfig{
    Model:  "sonar-pro",
    APIKey: "pplx-...",
})
if err != nil {
    log.Fatal(err)
}

resp, err := model.Generate(ctx, []schema.Message{
    schema.NewHumanMessage("What happened in tech news today?"),
})
```

## Configuration

The following [config.ProviderConfig] fields are used:

- Model: the Perplexity model name (e.g. "sonar-pro", "sonar")
- APIKey: the Perplexity API key
- BaseURL: optional, defaults to "https://api.perplexity.ai"

## Direct Construction

Use `New` to create a ChatModel directly without going through the registry.

---

## qwen

```go
import "github.com/lookatitude/beluga-ai/llm/providers/qwen"
```

Package qwen provides the Alibaba Qwen LLM provider for the Beluga AI
framework.

Qwen exposes an OpenAI-compatible API through Alibaba's DashScope platform,
so this provider is a thin wrapper around the shared openaicompat package
with Qwen's base URL.

## Registration

The provider registers itself as "qwen" via init(). Import the package
for side effects to make it available through the llm registry:

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/qwen"
```

## Usage

```go
model, err := llm.New("qwen", config.ProviderConfig{
    Model:  "qwen-plus",
    APIKey: "sk-...",
})
if err != nil {
    log.Fatal(err)
}

resp, err := model.Generate(ctx, []schema.Message{
    schema.NewHumanMessage("Explain the Qwen model family"),
})
```

## Configuration

The following [config.ProviderConfig] fields are used:

- Model: the Qwen model name (e.g. "qwen-plus", "qwen-turbo", "qwen-max")
- APIKey: the DashScope API key
- BaseURL: optional, defaults to "https://dashscope.aliyuncs.com/compatible-mode/v1"

## Direct Construction

Use `New` to create a ChatModel directly without going through the registry.

---

## sambanova

```go
import "github.com/lookatitude/beluga-ai/llm/providers/sambanova"
```

Package sambanova provides the SambaNova LLM provider for the Beluga AI
framework.

SambaNova exposes an OpenAI-compatible API running on custom RDU hardware
optimized for high-throughput inference. This provider is a thin wrapper
around the shared openaicompat package with SambaNova's base URL.

## Registration

The provider registers itself as "sambanova" via init(). Import the package
for side effects to make it available through the llm registry:

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/sambanova"
```

## Usage

```go
model, err := llm.New("sambanova", config.ProviderConfig{
    Model:  "Meta-Llama-3.3-70B-Instruct",
    APIKey: "sn-...",
})
if err != nil {
    log.Fatal(err)
}

resp, err := model.Generate(ctx, []schema.Message{
    schema.NewHumanMessage("Write a parallel algorithm"),
})
```

## Configuration

The following [config.ProviderConfig] fields are used:

- Model: the SambaNova model name (e.g. "Meta-Llama-3.3-70B-Instruct")
- APIKey: the SambaNova API key
- BaseURL: optional, defaults to "https://api.sambanova.ai/v1"

## Direct Construction

Use `New` to create a ChatModel directly without going through the registry.

---

## together

```go
import "github.com/lookatitude/beluga-ai/llm/providers/together"
```

Package together provides the Together AI LLM provider for the Beluga AI
framework.

Together AI exposes an OpenAI-compatible API for running open-source models
with optimized inference. This provider is a thin wrapper around the shared
openaicompat package with Together's base URL.

## Registration

The provider registers itself as "together" via init(). Import the package
for side effects to make it available through the llm registry:

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/together"
```

## Usage

```go
model, err := llm.New("together", config.ProviderConfig{
    Model:  "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo",
    APIKey: "...",
})
if err != nil {
    log.Fatal(err)
}

resp, err := model.Generate(ctx, []schema.Message{
    schema.NewHumanMessage("Compare Llama and Mistral models"),
})
```

## Configuration

The following [config.ProviderConfig] fields are used:

- Model: the Together model path (defaults to "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo")
- APIKey: the Together API key
- BaseURL: optional, defaults to "https://api.together.xyz/v1"

## Direct Construction

Use `New` to create a ChatModel directly without going through the registry.

---

## xai

```go
import "github.com/lookatitude/beluga-ai/llm/providers/xai"
```

Package xai provides the xAI Grok LLM provider for the Beluga AI framework.

xAI exposes an OpenAI-compatible API for Grok models. This provider is a
thin wrapper around the shared openaicompat package with xAI's base URL.

## Registration

The provider registers itself as "xai" via init(). Import the package
for side effects to make it available through the llm registry:

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/xai"
```

## Usage

```go
model, err := llm.New("xai", config.ProviderConfig{
    Model:  "grok-3",
    APIKey: "xai-...",
})
if err != nil {
    log.Fatal(err)
}

resp, err := model.Generate(ctx, []schema.Message{
    schema.NewHumanMessage("What makes Grok unique?"),
})
```

## Configuration

The following [config.ProviderConfig] fields are used:

- Model: the Grok model name (defaults to "grok-3")
- APIKey: the xAI API key
- BaseURL: optional, defaults to "https://api.x.ai/v1"

## Direct Construction

Use `New` to create a ChatModel directly without going through the registry.
