---
title: "Fireworks AI"
description: "Integration guide for Fireworks AI's fast inference with Beluga AI."
---

The Fireworks AI provider connects Beluga AI to Fireworks' inference platform, which specializes in fast, cost-effective serving of open-source models. Fireworks exposes an OpenAI-compatible API, so this provider supports all standard features including streaming, tool calling, and structured output.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/llm/providers/fireworks
```

## Configuration

| Field    | Required | Default                                                  | Description                    |
|----------|----------|----------------------------------------------------------|--------------------------------|
| `Model`  | No       | `"accounts/fireworks/models/llama-v3p1-70b-instruct"`   | Model ID                      |
| `APIKey` | Yes      | —                                                        | Fireworks API key (`fw_...`)   |
| `BaseURL`| No       | `https://api.fireworks.ai/inference/v1`                  | Override API endpoint          |
| `Timeout`| No       | `30s`                                                    | Request timeout                |

**Environment variables:**

| Variable             | Maps to  |
|----------------------|----------|
| `FIREWORKS_API_KEY`  | `APIKey` |

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
    _ "github.com/lookatitude/beluga-ai/llm/providers/fireworks"
)

func main() {
    model, err := llm.New("fireworks", config.ProviderConfig{
        Model:  "accounts/fireworks/models/llama-v3p1-70b-instruct",
        APIKey: os.Getenv("FIREWORKS_API_KEY"),
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
import "github.com/lookatitude/beluga-ai/llm/providers/fireworks"

model, err := fireworks.New(config.ProviderConfig{
    Model:  "accounts/fireworks/models/llama-v3p1-70b-instruct",
    APIKey: os.Getenv("FIREWORKS_API_KEY"),
})
```

## Available Models

| Model ID                                             | Description              |
|------------------------------------------------------|--------------------------|
| `accounts/fireworks/models/llama-v3p1-70b-instruct`  | Llama 3.1 70B            |
| `accounts/fireworks/models/llama-v3p1-8b-instruct`   | Llama 3.1 8B             |
| `accounts/fireworks/models/mixtral-8x7b-instruct`    | Mixtral 8x7B             |
| `accounts/fireworks/models/qwen2p5-72b-instruct`     | Qwen 2.5 72B             |

Refer to [Fireworks AI's documentation](https://docs.fireworks.ai/guides/querying-text-models) for the full model catalog.
