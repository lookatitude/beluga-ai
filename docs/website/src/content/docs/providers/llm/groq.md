---
title: "Groq"
description: "Integration guide for Groq's ultra-fast inference with Beluga AI."
---

The Groq provider connects Beluga AI to Groq's inference platform, which uses custom LPU (Language Processing Unit) hardware for extremely fast token generation. Groq exposes an OpenAI-compatible API, so this provider supports all standard features including streaming, tool calling, and structured output.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/llm/providers/groq
```

## Configuration

| Field    | Required | Default                              | Description                        |
|----------|----------|--------------------------------------|------------------------------------|
| `Model`  | Yes      | —                                    | Model ID (e.g. `"llama-3.3-70b-versatile"`) |
| `APIKey` | Yes      | —                                    | Groq API key (`gsk_...`)          |
| `BaseURL`| No       | `https://api.groq.com/openai/v1`    | Override API endpoint              |
| `Timeout`| No       | `30s`                                | Request timeout                    |

**Environment variables:**

| Variable       | Maps to  |
|----------------|----------|
| `GROQ_API_KEY` | `APIKey` |

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
    _ "github.com/lookatitude/beluga-ai/llm/providers/groq"
)

func main() {
    model, err := llm.New("groq", config.ProviderConfig{
        Model:  "llama-3.3-70b-versatile",
        APIKey: os.Getenv("GROQ_API_KEY"),
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
    llm.WithStopSequences("END"),
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
import "github.com/lookatitude/beluga-ai/llm/providers/groq"

model, err := groq.New(config.ProviderConfig{
    Model:  "llama-3.3-70b-versatile",
    APIKey: os.Getenv("GROQ_API_KEY"),
})
```

## Available Models

| Model ID                    | Description                  |
|-----------------------------|------------------------------|
| `llama-3.3-70b-versatile`  | Llama 3.3 70B                |
| `llama-3.1-8b-instant`     | Fast Llama 3.1 8B            |
| `mixtral-8x7b-32768`       | Mixtral 8x7B                 |
| `gemma2-9b-it`             | Gemma 2 9B                   |

Refer to [Groq's documentation](https://console.groq.com/docs/models) for the latest model list.
