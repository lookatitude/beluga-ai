---
title: "SambaNova"
description: "Integration guide for SambaNova's high-throughput inference with Beluga AI."
---

The SambaNova provider connects Beluga AI to SambaNova's inference platform, which provides high-throughput inference on custom RDU (Reconfigurable Dataflow Unit) hardware. SambaNova exposes an OpenAI-compatible API, so this provider supports all standard features including streaming, tool calling, and structured output.

Choose SambaNova when you need high-throughput batch inference or consistent low-latency serving of large open-source models. SambaNova's custom RDU hardware is optimized for sustained throughput, making it well-suited for production workloads with predictable high-volume traffic.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/llm/providers/sambanova
```

## Configuration

| Field    | Required | Default                            | Description                        |
|----------|----------|------------------------------------|------------------------------------|
| `Model`  | Yes      | —                                  | Model ID (e.g. `"Meta-Llama-3.3-70B-Instruct"`) |
| `APIKey` | Yes      | —                                  | SambaNova API key (`sn-...`)      |
| `BaseURL`| No       | `https://api.sambanova.ai/v1`     | Override API endpoint              |
| `Timeout`| No       | `30s`                              | Request timeout                    |

**Environment variables:**

| Variable            | Maps to  |
|---------------------|----------|
| `SAMBANOVA_API_KEY` | `APIKey` |

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
    _ "github.com/lookatitude/beluga-ai/llm/providers/sambanova"
)

func main() {
    model, err := llm.New("sambanova", config.ProviderConfig{
        Model:  "Meta-Llama-3.3-70B-Instruct",
        APIKey: os.Getenv("SAMBANOVA_API_KEY"),
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
import "github.com/lookatitude/beluga-ai/llm/providers/sambanova"

model, err := sambanova.New(config.ProviderConfig{
    Model:  "Meta-Llama-3.3-70B-Instruct",
    APIKey: os.Getenv("SAMBANOVA_API_KEY"),
})
```

## Available Models

| Model ID                         | Description                                       |
|----------------------------------|---------------------------------------------------|
| `Meta-Llama-3.3-70B-Instruct`  | Llama 3.3 70B -- best quality/speed tradeoff      |
| `Meta-Llama-3.1-8B-Instruct`   | Llama 3.1 8B -- fastest, lowest cost              |
| `Meta-Llama-3.1-405B-Instruct` | Llama 3.1 405B -- most capable, highest throughput|

Refer to [SambaNova's documentation](https://community.sambanova.ai/docs) for the latest model list.
