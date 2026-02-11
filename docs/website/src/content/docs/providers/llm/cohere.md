---
title: "Cohere"
description: "Integration guide for Cohere models with Beluga AI."
---

The Cohere provider connects Beluga AI to Cohere's Command R family of models using the official `cohere-go` SDK (v2). Cohere uses a distinct message format where system messages are sent as a preamble, the last user message is the primary input, and prior messages become chat history. The provider handles this mapping transparently.

Choose Cohere when building enterprise search and RAG applications. Command R models are specifically designed for retrieval-augmented generation with strong grounding capabilities and citation support. Cohere also provides matching embedding models for an end-to-end search stack.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/llm/providers/cohere
```

## Configuration

| Field    | Required | Default           | Description                        |
|----------|----------|-------------------|------------------------------------|
| `Model`  | No       | `"command-r-plus"` | Model ID                          |
| `APIKey` | Yes      | —                 | Cohere API key                     |
| `BaseURL`| No       | Cohere default    | Override API endpoint              |
| `Timeout`| No       | `30s`             | Request timeout                    |

**Environment variables:**

| Variable         | Maps to  |
|------------------|----------|
| `COHERE_API_KEY` | `APIKey` |

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
    _ "github.com/lookatitude/beluga-ai/llm/providers/cohere"
)

func main() {
    model, err := llm.New("cohere", config.ProviderConfig{
        Model:  "command-r-plus",
        APIKey: os.Getenv("COHERE_API_KEY"),
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

Cohere's stream emits `TextGeneration`, `ToolCallsGeneration`, and `StreamEnd` events. The provider maps these to Beluga's `StreamChunk` type.

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
resp, err := modelWithTools.Generate(ctx, msgs)
if err != nil {
    log.Fatal(err)
}

for _, tc := range resp.ToolCalls {
    fmt.Printf("Tool: %s, Args: %s\n", tc.Name, tc.Arguments)
}
```

Cohere tool definitions are automatically converted from the standard JSON Schema format to Cohere's `ParameterDefinitions` format.

### Generation Options

```go
resp, err := model.Generate(ctx, msgs,
    llm.WithTemperature(0.7),
    llm.WithMaxTokens(2048),
    llm.WithTopP(0.9),
    llm.WithStopSequences("END"),
)
```

### Message Mapping

The provider automatically maps Beluga messages to Cohere's format:

| Beluga Message          | Cohere Field       |
|-------------------------|--------------------|
| `schema.SystemMessage`  | `preamble`         |
| Last `schema.HumanMessage` | `message`       |
| Prior messages          | `chat_history`     |

This is handled transparently so you can use standard Beluga message types.

## Error Handling

```go
resp, err := model.Generate(ctx, msgs)
if err != nil {
    // Errors are wrapped with the "cohere:" prefix
    log.Fatal(err)
}
```

## Direct Construction

```go
import "github.com/lookatitude/beluga-ai/llm/providers/cohere"

model, err := cohere.New(config.ProviderConfig{
    Model:  "command-r-plus",
    APIKey: os.Getenv("COHERE_API_KEY"),
})
```

## Available Models

| Model ID         | Description                         |
|------------------|-------------------------------------|
| `command-r-plus` | Most capable Cohere model           |
| `command-r`      | Balanced performance and cost       |
| `command-light`  | Fast, lightweight model             |

Refer to [Cohere's documentation](https://docs.cohere.com/docs/models) for the latest model list.
