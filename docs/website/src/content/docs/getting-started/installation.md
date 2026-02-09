---
title: Installation
description: System requirements, installation steps, and provider setup for Beluga AI v2.
---

## System Requirements

- **Go 1.23+** — Beluga AI uses `iter.Seq2[T, error]` for streaming, which requires Go 1.23 or later
- **Git** — For version control and `go get` operations

Verify your Go version:

```bash
go version
# go version go1.23.0 linux/amd64  (or later)
```

## Install the Core Module

Add Beluga AI to your Go project:

```bash
go get github.com/lookatitude/beluga-ai@latest
```

This installs the core framework. LLM providers, vector stores, and other integrations are separate packages — you only import what you need.

## Provider Setup

Beluga AI uses a registry pattern: import a provider package and it registers itself via `init()`. You then create instances through the unified `llm.New()` factory.

### OpenAI

```bash
go get github.com/lookatitude/beluga-ai/llm/providers/openai
```

```bash
export OPENAI_API_KEY="sk-..."
```

```go
import (
    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/llm"
    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

model, err := llm.New("openai", config.ProviderConfig{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "gpt-4o",
})
```

### Anthropic

```bash
go get github.com/lookatitude/beluga-ai/llm/providers/anthropic
```

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
```

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/anthropic"

model, err := llm.New("anthropic", config.ProviderConfig{
    APIKey: os.Getenv("ANTHROPIC_API_KEY"),
    Model:  "claude-sonnet-4-5-20250929",
})
```

### Google (Gemini)

```bash
go get github.com/lookatitude/beluga-ai/llm/providers/google
```

```bash
export GOOGLE_API_KEY="AI..."
```

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/google"

model, err := llm.New("google", config.ProviderConfig{
    APIKey: os.Getenv("GOOGLE_API_KEY"),
    Model:  "gemini-2.5-flash",
})
```

### Ollama (Local Models)

```bash
go get github.com/lookatitude/beluga-ai/llm/providers/ollama
```

No API key required. Ollama must be running locally:

```bash
# Install and start Ollama, then pull a model
ollama pull llama3.2
```

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/ollama"

model, err := llm.New("ollama", config.ProviderConfig{
    Model:   "llama3.2",
    BaseURL: "http://localhost:11434",
})
```

### Groq

```bash
go get github.com/lookatitude/beluga-ai/llm/providers/groq
```

```bash
export GROQ_API_KEY="gsk_..."
```

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/groq"

model, err := llm.New("groq", config.ProviderConfig{
    APIKey: os.Getenv("GROQ_API_KEY"),
    Model:  "llama-3.3-70b-versatile",
})
```

### All Available Providers

| Provider | Registry Name | Package |
|----------|--------------|---------|
| OpenAI | `openai` | `llm/providers/openai` |
| Anthropic | `anthropic` | `llm/providers/anthropic` |
| Google Gemini | `google` | `llm/providers/google` |
| Ollama | `ollama` | `llm/providers/ollama` |
| AWS Bedrock | `bedrock` | `llm/providers/bedrock` |
| Azure OpenAI | `azure` | `llm/providers/azure` |
| Groq | `groq` | `llm/providers/groq` |
| Mistral | `mistral` | `llm/providers/mistral` |
| DeepSeek | `deepseek` | `llm/providers/deepseek` |
| xAI (Grok) | `xai` | `llm/providers/xai` |
| Cohere | `cohere` | `llm/providers/cohere` |
| Together | `together` | `llm/providers/together` |
| Fireworks | `fireworks` | `llm/providers/fireworks` |
| OpenRouter | `openrouter` | `llm/providers/openrouter` |
| Perplexity | `perplexity` | `llm/providers/perplexity` |
| HuggingFace | `huggingface` | `llm/providers/huggingface` |
| Cerebras | `cerebras` | `llm/providers/cerebras` |
| SambaNova | `sambanova` | `llm/providers/sambanova` |
| LiteLLM | `litellm` | `llm/providers/litellm` |
| Llama.cpp | `llama` | `llm/providers/llama` |
| Qwen | `qwen` | `llm/providers/qwen` |
| Bifrost | `bifrost` | `llm/providers/bifrost` |

## Environment Variables

Beluga AI reads provider credentials from environment variables. Set the relevant variables for your providers:

```bash
# LLM Providers
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
export GOOGLE_API_KEY="AI..."
export GROQ_API_KEY="gsk_..."
export MISTRAL_API_KEY="..."
export DEEPSEEK_API_KEY="..."
export XAI_API_KEY="..."
export COHERE_API_KEY="..."
export TOGETHER_API_KEY="..."
export FIREWORKS_API_KEY="..."

# AWS Bedrock (uses standard AWS credentials)
export AWS_ACCESS_KEY_ID="..."
export AWS_SECRET_ACCESS_KEY="..."
export AWS_REGION="us-east-1"
```

You can also use the `config` package to load settings from a JSON file with environment variable overrides:

```go
type AppConfig struct {
    LLM config.ProviderConfig `json:"llm" required:"true"`
}

cfg, err := config.Load[AppConfig]("config.json")
// Environment variables override: BELUGA_LLM_API_KEY, etc.
config.MergeEnv(&cfg, "BELUGA")
```

## Verifying Your Installation

Create a simple program to verify everything works:

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

func main() {
    model, err := llm.New("openai", config.ProviderConfig{
        APIKey: os.Getenv("OPENAI_API_KEY"),
        Model:  "gpt-4o",
    })
    if err != nil {
        fmt.Fprintf(os.Stderr, "failed to create model: %v\n", err)
        os.Exit(1)
    }

    ctx := context.Background()
    resp, err := model.Generate(ctx, []schema.Message{
        schema.NewHumanMessage("Say hello from Beluga AI!"),
    })
    if err != nil {
        fmt.Fprintf(os.Stderr, "generate failed: %v\n", err)
        os.Exit(1)
    }

    fmt.Println(resp.Text())
}
```

Run it:

```bash
go run main.go
```

If you see a response from the model, your installation is working.

## Optional Dependencies

Some packages require additional system dependencies:

### CGO-Dependent Packages

| Package | Requires | Notes |
|---------|----------|-------|
| `rag/vectorstore/providers/sqlitevec` | CGO + sqlite-vec extension | For embedded vector search |
| `voice/vad/silero` | CGO + ONNX Runtime | For Silero VAD voice activity detection |

Enable CGO:

```bash
export CGO_ENABLED=1
```

### External Services

| Package | Requires | Notes |
|---------|----------|-------|
| `memory/stores/neo4j` | Neo4j instance | For graph memory |
| `memory/stores/memgraph` | Memgraph instance | For in-memory graph |
| `memory/stores/redis` | Redis instance | For Redis-backed memory/cache/state |
| `memory/stores/postgres` | PostgreSQL instance | For persistent storage |

## IDE Setup

### VS Code

Install the [Go extension](https://marketplace.visualstudio.com/items?itemName=golang.Go) and ensure `gopls` is configured:

```json
{
    "go.useLanguageServer": true,
    "gopls": {
        "build.buildFlags": ["-tags=integration"]
    }
}
```

### GoLand / IntelliJ

Go support is built in. For integration tests, add `-tags=integration` to your run configuration build tags.

### General Tips

- Run `go mod tidy` after adding new provider imports to clean up dependencies
- Use `go vet ./...` to catch common issues
- Beluga AI's interfaces are small (1-4 methods) — your IDE's autocomplete will surface them quickly

## Next Steps

- [Quick Start](/getting-started/quick-start/) — Build your first agent in 5 minutes
- [Working with LLMs](/guides/working-with-llms/) — Deep dive into LLM configuration
- [LLM Providers](/integrations/llm-providers/) — Detailed provider setup guides
