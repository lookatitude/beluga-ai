---
title: "xAI (Grok)"
description: "Integration guide for xAI's Grok models with Beluga AI."
---

The xAI provider connects Beluga AI to xAI's Grok family of models. xAI exposes an OpenAI-compatible API, so this provider supports all standard features including streaming, tool calling, and structured output.

Choose xAI when you want access to Grok models, which offer strong reasoning and conversational capabilities. Grok-3 is competitive with frontier models from other providers and provides large context windows for extended conversations.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/llm/providers/xai
```

## Configuration

| Field    | Required | Default                    | Description                    |
|----------|----------|----------------------------|--------------------------------|
| `Model`  | No       | `"grok-3"`                 | Model ID                      |
| `APIKey` | Yes      | —                          | xAI API key (`xai-...`)       |
| `BaseURL`| No       | `https://api.x.ai/v1`     | Override API endpoint          |
| `Timeout`| No       | `30s`                      | Request timeout                |

**Environment variables:**

| Variable      | Maps to  |
|---------------|----------|
| `XAI_API_KEY` | `APIKey` |

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
    _ "github.com/lookatitude/beluga-ai/llm/providers/xai"
)

func main() {
    model, err := llm.New("xai", config.ProviderConfig{
        Model:  "grok-3",
        APIKey: os.Getenv("XAI_API_KEY"),
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
import "github.com/lookatitude/beluga-ai/llm/providers/xai"

model, err := xai.New(config.ProviderConfig{
    Model:  "grok-3",
    APIKey: os.Getenv("XAI_API_KEY"),
})
```

## Available Models

| Model ID       | Description                                  |
|----------------|----------------------------------------------|
| `grok-3`       | Most capable Grok model, large context       |
| `grok-3-mini`  | Fast, cost-effective with strong reasoning   |
| `grok-2`       | Previous generation, still capable           |

Refer to [xAI's documentation](https://docs.x.ai/docs) for the latest model list.
