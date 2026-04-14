---
title: "LiteLLM LLM Provider"
description: "Integrate LiteLLM universal proxy with Beluga AI. Route requests to 100+ LLM providers through a single OpenAI-compatible endpoint in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "LiteLLM, LLM proxy, universal provider, OpenAI compatible, multi-provider, streaming, Go, Beluga AI"
---

The LiteLLM provider connects Beluga AI to a [LiteLLM](https://litellm.ai) proxy gateway. LiteLLM provides a unified OpenAI-compatible API in front of 100+ LLM providers, handling format translation, load balancing, spend tracking, and rate limiting. This provider is a thin wrapper that points to your LiteLLM deployment.

Choose LiteLLM when you need centralized LLM management at the infrastructure layer -- spend tracking, rate limiting, virtual API keys, and provider abstraction managed outside your application code. LiteLLM is well-suited for teams with multiple applications sharing LLM access.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/llm/providers/litellm
```

**Prerequisites:** A running LiteLLM proxy instance. The quickest way to start one:

```bash
pip install litellm[proxy]
litellm --model gpt-4o --port 4000
```

## Configuration

| Field    | Required | Default                        | Description                              |
|----------|----------|--------------------------------|------------------------------------------|
| `Model`  | No       | `"gpt-4o"`                    | Model ID (LiteLLM handles routing)       |
| `APIKey` | No       | —                              | API key (if LiteLLM requires auth)       |
| `BaseURL`| No       | `http://localhost:4000/v1`    | LiteLLM proxy endpoint                   |
| `Timeout`| No       | `30s`                          | Request timeout                          |

## Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
    _ "github.com/lookatitude/beluga-ai/llm/providers/litellm"
)

func main() {
    model, err := llm.New("litellm", config.ProviderConfig{
        Model:   "gpt-4o",
        BaseURL: "http://localhost:4000/v1",
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

LiteLLM translates between provider formats, so all features available in the OpenAI-compatible interface work through LiteLLM:

### Accessing Different Providers

LiteLLM routes to different providers based on model naming:

```go
// Route to Anthropic via LiteLLM
model, err := llm.New("litellm", config.ProviderConfig{
    Model:   "anthropic/claude-sonnet-4-5-20250929",
    BaseURL: "http://localhost:4000/v1",
})

// Route to Google via LiteLLM
model, err := llm.New("litellm", config.ProviderConfig{
    Model:   "gemini/gemini-2.5-flash",
    BaseURL: "http://localhost:4000/v1",
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
import "github.com/lookatitude/beluga-ai/llm/providers/litellm"

model, err := litellm.New(config.ProviderConfig{
    Model:   "gpt-4o",
    BaseURL: "http://localhost:4000/v1",
})
```

## When to Use LiteLLM

LiteLLM is a good choice when you need:

- **Provider abstraction** at the infrastructure level rather than in application code
- **Spend tracking and rate limiting** across multiple providers and teams
- **Load balancing** across multiple API keys or provider accounts
- **Virtual keys** for team-level access control
- **Logging and observability** at the proxy layer
