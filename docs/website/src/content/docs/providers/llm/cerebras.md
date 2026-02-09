---
title: "Cerebras"
description: "Integration guide for Cerebras wafer-scale inference with Beluga AI."
---

The Cerebras provider connects Beluga AI to Cerebras' inference platform, which uses wafer-scale engine (WSE) hardware for extremely fast inference. Cerebras exposes an OpenAI-compatible API, so this provider supports all standard features including streaming, tool calling, and structured output.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/llm/providers/cerebras
```

## Configuration

| Field    | Required | Default                          | Description                        |
|----------|----------|----------------------------------|------------------------------------|
| `Model`  | Yes      | —                                | Model ID (e.g. `"llama-3.3-70b"`) |
| `APIKey` | Yes      | —                                | Cerebras API key (`csk-...`)      |
| `BaseURL`| No       | `https://api.cerebras.ai/v1`    | Override API endpoint              |
| `Timeout`| No       | `30s`                            | Request timeout                    |

**Environment variables:**

| Variable            | Maps to  |
|---------------------|----------|
| `CEREBRAS_API_KEY`  | `APIKey` |

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
    _ "github.com/lookatitude/beluga-ai/llm/providers/cerebras"
)

func main() {
    model, err := llm.New("cerebras", config.ProviderConfig{
        Model:  "llama-3.3-70b",
        APIKey: os.Getenv("CEREBRAS_API_KEY"),
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
import "github.com/lookatitude/beluga-ai/llm/providers/cerebras"

model, err := cerebras.New(config.ProviderConfig{
    Model:  "llama-3.3-70b",
    APIKey: os.Getenv("CEREBRAS_API_KEY"),
})
```

## Available Models

| Model ID          | Description              |
|-------------------|--------------------------|
| `llama-3.3-70b`  | Llama 3.3 70B            |
| `llama-3.1-8b`   | Llama 3.1 8B             |

Refer to [Cerebras' documentation](https://inference-docs.cerebras.ai/introduction) for the latest model list.
