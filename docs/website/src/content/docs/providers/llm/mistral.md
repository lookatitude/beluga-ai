---
title: "Mistral AI"
description: "Integration guide for Mistral AI models with Beluga AI."
---

The Mistral provider connects Beluga AI to Mistral AI's models using the `mistral-go` SDK. It supports chat completions, streaming, tool calling, and JSON mode.

Choose Mistral when you need competitive model quality with European data residency (hosted in EU). Mistral Large offers strong reasoning and multilingual capabilities, while Codestral specializes in code generation. Mistral also offers efficient open-weight models that can be self-hosted.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/llm/providers/mistral
```

## Configuration

| Field    | Required | Default                     | Description                              |
|----------|----------|-----------------------------|------------------------------------------|
| `Model`  | No       | `"mistral-large-latest"`    | Model ID                                 |
| `APIKey` | Yes      | —                           | Mistral API key                          |
| `BaseURL`| No       | `https://api.mistral.ai`    | Override API endpoint                    |
| `Timeout`| No       | `30s`                       | Request timeout                          |

**Environment variables:**

| Variable           | Maps to  |
|--------------------|----------|
| `MISTRAL_API_KEY`  | `APIKey` |

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
    _ "github.com/lookatitude/beluga-ai/llm/providers/mistral"
)

func main() {
    model, err := llm.New("mistral", config.ProviderConfig{
        Model:  "mistral-large-latest",
        APIKey: os.Getenv("MISTRAL_API_KEY"),
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

The Mistral streaming implementation respects context cancellation and returns channel-based events converted to `iter.Seq2`.

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

Mistral tool choice mapping:

| Beluga ToolChoice         | Mistral Equivalent |
|---------------------------|--------------------|
| `llm.ToolChoiceAuto`     | `auto`             |
| `llm.ToolChoiceNone`     | `none`             |
| `llm.ToolChoiceRequired` | `any`              |

### JSON Mode

```go
resp, err := model.Generate(ctx, msgs,
    llm.WithResponseFormat(llm.ResponseFormat{Type: "json_object"}),
)
```

### Generation Options

The provider defaults to `Temperature: 0.7` and `TopP: 1.0` unless overridden:

```go
resp, err := model.Generate(ctx, msgs,
    llm.WithTemperature(0.3),
    llm.WithMaxTokens(2048),
    llm.WithTopP(0.9),
)
```

## Error Handling

```go
resp, err := model.Generate(ctx, msgs)
if err != nil {
    // Errors are wrapped with the "mistral:" prefix
    log.Fatal(err)
}
```

## Direct Construction

```go
import "github.com/lookatitude/beluga-ai/llm/providers/mistral"

model, err := mistral.New(config.ProviderConfig{
    Model:  "mistral-large-latest",
    APIKey: os.Getenv("MISTRAL_API_KEY"),
})
```

## Available Models

| Model ID                 | Description                        |
|--------------------------|------------------------------------|
| `mistral-large-latest`   | Most capable Mistral model         |
| `mistral-small-latest`   | Fast, efficient model              |
| `codestral-latest`       | Code generation specialist         |
| `open-mistral-nemo`      | Open-weight 12B model              |

Refer to [Mistral AI's documentation](https://docs.mistral.ai/getting-started/models/models_overview/) for the latest model list.
