---
title: "DeepSeek"
description: "Integration guide for DeepSeek models with Beluga AI."
---

The DeepSeek provider connects Beluga AI to DeepSeek's inference platform. DeepSeek exposes an OpenAI-compatible API, so this provider supports all standard features including streaming, tool calling, and structured output. DeepSeek is known for its DeepSeek-V3 and DeepSeek-R1 reasoning models.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/llm/providers/deepseek
```

## Configuration

| Field    | Required | Default                            | Description                    |
|----------|----------|------------------------------------|--------------------------------|
| `Model`  | No       | `"deepseek-chat"`                  | Model ID                      |
| `APIKey` | Yes      | —                                  | DeepSeek API key (`sk-...`)   |
| `BaseURL`| No       | `https://api.deepseek.com/v1`     | Override API endpoint          |
| `Timeout`| No       | `30s`                              | Request timeout                |

**Environment variables:**

| Variable           | Maps to  |
|--------------------|----------|
| `DEEPSEEK_API_KEY` | `APIKey` |

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
    _ "github.com/lookatitude/beluga-ai/llm/providers/deepseek"
)

func main() {
    model, err := llm.New("deepseek", config.ProviderConfig{
        Model:  "deepseek-chat",
        APIKey: os.Getenv("DEEPSEEK_API_KEY"),
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
    llm.WithMaxTokens(4096),
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
import "github.com/lookatitude/beluga-ai/llm/providers/deepseek"

model, err := deepseek.New(config.ProviderConfig{
    Model:  "deepseek-chat",
    APIKey: os.Getenv("DEEPSEEK_API_KEY"),
})
```

## Available Models

| Model ID           | Description                         |
|--------------------|-------------------------------------|
| `deepseek-chat`    | DeepSeek V3 general chat model      |
| `deepseek-reasoner`| DeepSeek R1 reasoning model         |

Refer to [DeepSeek's documentation](https://platform.deepseek.com/api-docs) for the latest model list.
