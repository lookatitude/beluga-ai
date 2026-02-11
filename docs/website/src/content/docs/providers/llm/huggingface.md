---
title: "HuggingFace"
description: "Integration guide for HuggingFace Inference API with Beluga AI."
---

The HuggingFace provider connects Beluga AI to HuggingFace's Inference API, which provides hosted access to thousands of open-source models. HuggingFace exposes an OpenAI-compatible chat completions endpoint, so this provider supports all standard features including streaming and tool calling.

Choose HuggingFace when you need access to specialized or fine-tuned models from the HuggingFace ecosystem. The free Inference API is suitable for prototyping, while Dedicated Inference Endpoints provide production-grade hosting with guaranteed compute for your chosen model.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/llm/providers/huggingface
```

## Configuration

| Field    | Required | Default                                        | Description                          |
|----------|----------|-------------------------------------------------|--------------------------------------|
| `Model`  | Yes      | —                                               | Model ID (HuggingFace repo format)  |
| `APIKey` | Yes      | —                                               | HuggingFace token (`hf_...`)        |
| `BaseURL`| No       | `https://api-inference.huggingface.co/v1`      | Override API endpoint                |
| `Timeout`| No       | `30s`                                           | Request timeout                      |

**Environment variables:**

| Variable                 | Maps to  |
|--------------------------|----------|
| `HUGGINGFACE_API_KEY`    | `APIKey` |
| `HF_TOKEN`              | `APIKey` |

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
    _ "github.com/lookatitude/beluga-ai/llm/providers/huggingface"
)

func main() {
    model, err := llm.New("huggingface", config.ProviderConfig{
        Model:  "meta-llama/Meta-Llama-3.1-70B-Instruct",
        APIKey: os.Getenv("HF_TOKEN"),
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

Tool calling support depends on the model:

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

### Dedicated Inference Endpoints

To use a HuggingFace Dedicated Inference Endpoint, set the `BaseURL`:

```go
model, err := llm.New("huggingface", config.ProviderConfig{
    Model:   "meta-llama/Meta-Llama-3.1-70B-Instruct",
    APIKey:  os.Getenv("HF_TOKEN"),
    BaseURL: "https://your-endpoint.us-east-1.aws.endpoints.huggingface.cloud/v1",
})
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
import "github.com/lookatitude/beluga-ai/llm/providers/huggingface"

model, err := huggingface.New(config.ProviderConfig{
    Model:  "meta-llama/Meta-Llama-3.1-70B-Instruct",
    APIKey: os.Getenv("HF_TOKEN"),
})
```

## Available Models

HuggingFace hosts thousands of models. Popular choices for the Inference API include:

| Model ID                                    | Description              |
|---------------------------------------------|--------------------------|
| `meta-llama/Meta-Llama-3.1-70B-Instruct`  | Llama 3.1 70B            |
| `meta-llama/Meta-Llama-3.1-8B-Instruct`   | Llama 3.1 8B             |
| `mistralai/Mixtral-8x7B-Instruct-v0.1`    | Mixtral 8x7B             |
| `microsoft/Phi-3-medium-4k-instruct`      | Phi-3 Medium             |

Refer to the [HuggingFace model hub](https://huggingface.co/models?pipeline_tag=text-generation) for the full catalog.
