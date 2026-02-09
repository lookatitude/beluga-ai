---
title: "Qwen (Alibaba)"
description: "Integration guide for Alibaba's Qwen models with Beluga AI."
---

The Qwen provider connects Beluga AI to Alibaba's Qwen family of models via the DashScope API. Qwen exposes an OpenAI-compatible API, so this provider supports all standard features including streaming, tool calling, and structured output.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/llm/providers/qwen
```

## Configuration

| Field    | Required | Default                                              | Description                    |
|----------|----------|------------------------------------------------------|--------------------------------|
| `Model`  | Yes      | —                                                    | Model ID (e.g. `"qwen-plus"`) |
| `APIKey` | Yes      | —                                                    | DashScope API key (`sk-...`)   |
| `BaseURL`| No       | `https://dashscope.aliyuncs.com/compatible-mode/v1` | Override API endpoint          |
| `Timeout`| No       | `30s`                                                | Request timeout                |

**Environment variables:**

| Variable           | Maps to  |
|--------------------|----------|
| `DASHSCOPE_API_KEY`| `APIKey` |

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
    _ "github.com/lookatitude/beluga-ai/llm/providers/qwen"
)

func main() {
    model, err := llm.New("qwen", config.ProviderConfig{
        Model:  "qwen-plus",
        APIKey: os.Getenv("DASHSCOPE_API_KEY"),
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
import "github.com/lookatitude/beluga-ai/llm/providers/qwen"

model, err := qwen.New(config.ProviderConfig{
    Model:  "qwen-plus",
    APIKey: os.Getenv("DASHSCOPE_API_KEY"),
})
```

## Available Models

| Model ID        | Description                     |
|-----------------|---------------------------------|
| `qwen-plus`     | Balanced cost and performance   |
| `qwen-turbo`    | Fast, cost-effective model      |
| `qwen-max`      | Most capable Qwen model         |
| `qwen-long`     | Extended context model          |

Refer to [Alibaba Cloud's documentation](https://help.aliyun.com/zh/model-studio/developer-reference/what-is-qwen-llm) for the latest model list.
