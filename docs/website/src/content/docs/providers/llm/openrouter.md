---
title: "OpenRouter"
description: "Integration guide for OpenRouter's multi-provider routing with Beluga AI."
---

The OpenRouter provider connects Beluga AI to [OpenRouter](https://openrouter.ai), a unified gateway that routes requests to many different model providers (OpenAI, Anthropic, Google, Meta, Mistral, and more). OpenRouter exposes an OpenAI-compatible API, so this provider supports all standard features including streaming, tool calling, and structured output.

Choose OpenRouter when you want to compare models across providers without managing separate API keys, or when you need access to niche models not available through other providers. OpenRouter's single API key simplifies billing and model switching during development and evaluation.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/llm/providers/openrouter
```

## Configuration

| Field    | Required | Default                            | Description                        |
|----------|----------|------------------------------------|------------------------------------|
| `Model`  | Yes      | —                                  | Model ID (provider/model format)   |
| `APIKey` | Yes      | —                                  | OpenRouter API key (`sk-or-...`)   |
| `BaseURL`| No       | `https://openrouter.ai/api/v1`    | Override API endpoint              |
| `Timeout`| No       | `30s`                              | Request timeout                    |

**Environment variables:**

| Variable              | Maps to  |
|-----------------------|----------|
| `OPENROUTER_API_KEY`  | `APIKey` |

## Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
    _ "github.com/lookatitude/beluga-ai/llm/providers/openrouter"
)

func main() {
    model, err := llm.New("openrouter", config.ProviderConfig{
        Model:  "anthropic/claude-sonnet-4-5-20250929",
        APIKey: os.Getenv("OPENROUTER_API_KEY"),
    })
    if err != nil {
        log.Fatal(err)
    }

    msgs := []schema.Message{
        schema.NewSystemMessage("You are a helpful assistant."),
        schema.NewHumanMessage("What is the capital of France?"),
    }

    resp, err := model.Generate(context.Background(), msgs)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(resp.Text())
}
```

## Streaming

```go
for chunk, err := range model.Stream(context.Background(), msgs) {
    if err != nil {
        log.Fatal(err)
    }
    fmt.Print(chunk.Delta)
}
fmt.Println()
```

## Advanced Features

### Accessing Different Providers

OpenRouter uses a `provider/model` naming convention:

```go
// Use Claude via OpenRouter
model, err := llm.New("openrouter", config.ProviderConfig{
    Model:  "anthropic/claude-sonnet-4-5-20250929",
    APIKey: os.Getenv("OPENROUTER_API_KEY"),
})

// Use GPT-4o via OpenRouter
model, err := llm.New("openrouter", config.ProviderConfig{
    Model:  "openai/gpt-4o",
    APIKey: os.Getenv("OPENROUTER_API_KEY"),
})

// Use Llama via OpenRouter
model, err := llm.New("openrouter", config.ProviderConfig{
    Model:  "meta-llama/llama-3.1-70b-instruct",
    APIKey: os.Getenv("OPENROUTER_API_KEY"),
})
```

### Tool Calling

```go
modelWithTools := model.BindTools(tools)
resp, err := modelWithTools.Generate(ctx, msgs, llm.WithToolChoice(llm.ToolChoiceAuto))
```

### Structured Output

```go
resp, err := model.Generate(ctx, msgs,
    llm.WithResponseFormat(llm.ResponseFormat{Type: "json_object"}),
)
```

### Generation Options

```go
resp, err := model.Generate(ctx, msgs,
    llm.WithTemperature(0.7),
    llm.WithMaxTokens(2048),
    llm.WithTopP(0.9),
)
```

## Error Handling

```go
resp, err := model.Generate(ctx, msgs)
if err != nil {
    log.Fatal(err)
}
```

## Direct Construction

```go
import "github.com/lookatitude/beluga-ai/llm/providers/openrouter"

model, err := openrouter.New(config.ProviderConfig{
    Model:  "anthropic/claude-sonnet-4-5-20250929",
    APIKey: os.Getenv("OPENROUTER_API_KEY"),
})
```

## Available Models

OpenRouter provides access to hundreds of models. Some popular options:

| Model ID                                 | Provider  | Description          |
|------------------------------------------|-----------|----------------------|
| `openai/gpt-4o`                         | OpenAI    | GPT-4o               |
| `anthropic/claude-sonnet-4-5-20250929`   | Anthropic | Claude Sonnet 4.5    |
| `google/gemini-2.5-flash`               | Google    | Gemini 2.5 Flash     |
| `meta-llama/llama-3.1-70b-instruct`     | Meta      | Llama 3.1 70B        |

Refer to [OpenRouter's model list](https://openrouter.ai/models) for the full catalog.
