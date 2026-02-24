---
title: "OpenAI LLM Provider"
description: "Integrate OpenAI GPT and o-series models with Beluga AI. Streaming, tool calling, structured output, and vision support via the official Go SDK."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "OpenAI, GPT-4, LLM provider, Go OpenAI SDK, streaming, tool calling, structured output, Beluga AI"
---

The OpenAI provider connects Beluga AI to OpenAI's GPT and o-series models through the official `openai-go` SDK. It supports chat completions, streaming, tool calling, structured output, and vision.

Choose OpenAI when you need the broadest ecosystem support, the most mature tool-calling implementation, or access to both GPT and o-series reasoning models. OpenAI is a strong default for general-purpose applications and has the widest third-party tooling compatibility.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/llm/providers/openai
```

## Configuration

| Field    | Required | Default                        | Description                  |
|----------|----------|--------------------------------|------------------------------|
| `Model`  | Yes      | —                              | Model ID (e.g. `"gpt-4o"`)  |
| `APIKey` | Yes      | —                              | OpenAI API key (`sk-...`)    |
| `BaseURL`| No       | `https://api.openai.com/v1`    | Override API endpoint        |
| `Timeout`| No       | `30s`                          | Request timeout              |

**Environment variables:**

| Variable         | Maps to  |
|------------------|----------|
| `OPENAI_API_KEY` | `APIKey` |

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
    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

func main() {
    model, err := llm.New("openai", config.ProviderConfig{
        Model:  "gpt-4o",
        APIKey: os.Getenv("OPENAI_API_KEY"),
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

The stream automatically includes usage metadata when available.

## Advanced Features

### Tool Calling

```go
tools := []schema.ToolDefinition{
    {
        Name:        "get_weather",
        Description: "Get current weather for a location",
        InputSchema: map[string]any{
            "type": "object",
            "properties": map[string]any{
                "location": map[string]any{
                    "type":        "string",
                    "description": "City name",
                },
            },
            "required": []any{"location"},
        },
    },
}

modelWithTools := model.BindTools(tools)
resp, err := modelWithTools.Generate(ctx, msgs, llm.WithToolChoice(llm.ToolChoiceAuto))
if err != nil {
    log.Fatal(err)
}

for _, tc := range resp.ToolCalls {
    fmt.Printf("Tool: %s, Args: %s\n", tc.Name, tc.Arguments)
}
```

### Structured Output (JSON Mode)

```go
resp, err := model.Generate(ctx, msgs,
    llm.WithResponseFormat(llm.ResponseFormat{Type: "json_object"}),
)
```

### JSON Schema Mode

```go
resp, err := model.Generate(ctx, msgs,
    llm.WithResponseFormat(llm.ResponseFormat{
        Type: "json_schema",
        Schema: map[string]any{
            "name": "person",
            "strict": true,
            "schema": map[string]any{
                "type": "object",
                "properties": map[string]any{
                    "name": map[string]any{"type": "string"},
                    "age":  map[string]any{"type": "integer"},
                },
                "required": []any{"name", "age"},
            },
        },
    }),
)
```

### Generation Options

```go
resp, err := model.Generate(ctx, msgs,
    llm.WithTemperature(0.7),
    llm.WithMaxTokens(2048),
    llm.WithTopP(0.9),
    llm.WithStopSequences("END"),
)
```

## Error Handling

```go
resp, err := model.Generate(ctx, msgs)
if err != nil {
    // Errors are wrapped with the "openaicompat:" prefix
    // Check for retryable errors in production:
    if core.IsRetryable(err) {
        // Retry with backoff
    }
    log.Fatal(err)
}
```

## Direct Construction

For compile-time type safety, construct the provider directly instead of using the registry:

```go
import "github.com/lookatitude/beluga-ai/llm/providers/openai"

model, err := openai.New(config.ProviderConfig{
    Model:  "gpt-4o",
    APIKey: os.Getenv("OPENAI_API_KEY"),
})
```

## Available Models

| Model ID         | Description                      |
|------------------|----------------------------------|
| `gpt-4o`         | Most capable multimodal model    |
| `gpt-4o-mini`    | Fast, cost-effective model       |
| `o1`             | Reasoning model                  |
| `o3`             | Advanced reasoning model         |
| `o3-mini`        | Fast reasoning model             |

Refer to [OpenAI's model documentation](https://platform.openai.com/docs/models) for the latest model list.
