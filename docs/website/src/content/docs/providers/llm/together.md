---
title: "Together AI"
description: "Integration guide for Together AI's open-source model hosting with Beluga AI."
---

The Together AI provider connects Beluga AI to Together's inference platform, which hosts a wide selection of open-source models. Together exposes an OpenAI-compatible API, so this provider supports all standard features including streaming, tool calling, and structured output.

Choose Together AI when you want access to a broad catalog of open-source models (Llama, Mixtral, Qwen, and more) with "Turbo" variants optimized for speed. Together is a good choice for prototyping across different model architectures and for workloads where open-weight model access is important.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/llm/providers/together
```

## Configuration

| Field    | Required | Default                                              | Description                        |
|----------|----------|------------------------------------------------------|------------------------------------|
| `Model`  | No       | `"meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo"`   | Model ID                          |
| `APIKey` | Yes      | —                                                    | Together API key                   |
| `BaseURL`| No       | `https://api.together.xyz/v1`                       | Override API endpoint              |
| `Timeout`| No       | `30s`                                                | Request timeout                    |

**Environment variables:**

| Variable            | Maps to  |
|---------------------|----------|
| `TOGETHER_API_KEY`  | `APIKey` |

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
    _ "github.com/lookatitude/beluga-ai/llm/providers/together"
)

func main() {
    model, err := llm.New("together", config.ProviderConfig{
        Model:  "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo",
        APIKey: os.Getenv("TOGETHER_API_KEY"),
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
import "github.com/lookatitude/beluga-ai/llm/providers/together"

model, err := together.New(config.ProviderConfig{
    Model:  "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo",
    APIKey: os.Getenv("TOGETHER_API_KEY"),
})
```

## Available Models

| Model ID                                         | Description              |
|--------------------------------------------------|--------------------------|
| `meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo`  | Llama 3.1 70B (Turbo)   |
| `meta-llama/Meta-Llama-3.1-8B-Instruct-Turbo`   | Llama 3.1 8B (Turbo)    |
| `mistralai/Mixtral-8x7B-Instruct-v0.1`          | Mixtral 8x7B             |
| `Qwen/Qwen2.5-72B-Instruct-Turbo`              | Qwen 2.5 72B             |

Refer to [Together AI's model list](https://docs.together.ai/docs/inference-models) for the full catalog.
