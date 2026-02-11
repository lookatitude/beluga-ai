---
title: "Ollama"
description: "Integration guide for running local models with Ollama and Beluga AI."
---

The Ollama provider connects Beluga AI to locally-hosted models via [Ollama](https://ollama.ai). Ollama serves an OpenAI-compatible API on `localhost`, making it straightforward to run open-source models without any cloud dependency or API key.

Choose Ollama for local development, offline environments, or when data privacy requires that no data leaves your machine. Ollama is also the fastest way to prototype with open-source models without setting up cloud accounts or managing API keys.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/llm/providers/ollama
```

**Prerequisites:** Install and start Ollama, then pull a model:

```bash
# Install Ollama (see https://ollama.ai for platform-specific instructions)
curl -fsSL https://ollama.com/install.sh | sh

# Pull a model
ollama pull llama3.2

# Start serving (if not already running)
ollama serve
```

## Configuration

| Field    | Required | Default                          | Description                    |
|----------|----------|----------------------------------|--------------------------------|
| `Model`  | Yes      | —                                | Model name (e.g. `"llama3.2"`) |
| `APIKey` | No       | `"ollama"`                       | Not required for local use     |
| `BaseURL`| No       | `http://localhost:11434/v1`      | Ollama API endpoint            |
| `Timeout`| No       | `30s`                            | Request timeout                |

No API key is needed for local Ollama. The provider uses a placeholder value of `"ollama"` automatically.

## Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
    _ "github.com/lookatitude/beluga-ai/llm/providers/ollama"
)

func main() {
    model, err := llm.New("ollama", config.ProviderConfig{
        Model: "llama3.2",
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

Tool calling support depends on the model. Models like `llama3.2` and `mistral` support function calling through Ollama's OpenAI-compatible endpoint:

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
```

### Remote Ollama

To connect to a remote Ollama instance, set the `BaseURL`:

```go
model, err := llm.New("ollama", config.ProviderConfig{
    Model:   "llama3.2",
    BaseURL: "http://192.168.1.100:11434/v1",
})
```

### Generation Options

```go
resp, err := model.Generate(ctx, msgs,
    llm.WithTemperature(0.7),
    llm.WithMaxTokens(2048),
    llm.WithTopP(0.9),
)
```

## Error Handling

```go
resp, err := model.Generate(ctx, msgs)
if err != nil {
    // Common error: Ollama not running or model not pulled
    log.Fatal(err)
}
```

## Direct Construction

```go
import "github.com/lookatitude/beluga-ai/llm/providers/ollama"

model, err := ollama.New(config.ProviderConfig{
    Model: "llama3.2",
})
```

## Available Models

Ollama supports hundreds of open-source models. Popular choices include:

| Model Name     | Description                    |
|----------------|--------------------------------|
| `llama3.2`     | Meta Llama 3.2 (1B/3B)        |
| `llama3.1`     | Meta Llama 3.1 (8B/70B)       |
| `mistral`      | Mistral 7B                     |
| `mixtral`      | Mixtral 8x7B                   |
| `phi4`         | Microsoft Phi-4                |
| `qwen2.5`      | Alibaba Qwen 2.5              |
| `deepseek-r1`  | DeepSeek R1 reasoning model   |

Run `ollama list` to see installed models, or browse the [Ollama model library](https://ollama.com/library).
