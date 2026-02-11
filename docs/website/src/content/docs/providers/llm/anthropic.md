---
title: "Anthropic"
description: "Integration guide for Anthropic Claude models with Beluga AI."
---

The Anthropic provider connects Beluga AI to Anthropic's Claude family of models using the official `anthropic-sdk-go` SDK. It supports chat completions, streaming, tool calling, vision (images via URL or base64), and system messages as a first-class concept.

Choose Anthropic when you need strong reasoning capabilities, large context windows (up to 200K tokens), or prompt caching for cost optimization. Claude models are well-suited for complex agentic workflows, code generation, and tasks requiring careful instruction following.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/llm/providers/anthropic
```

## Configuration

| Field    | Required | Default | Description                              |
|----------|----------|---------|------------------------------------------|
| `Model`  | Yes      | —       | Model ID (e.g. `"claude-sonnet-4-5-20250929"`) |
| `APIKey` | Yes      | —       | Anthropic API key (`sk-ant-...`)         |
| `BaseURL`| No       | Anthropic default | Override API endpoint          |
| `Timeout`| No       | `30s`   | Request timeout                          |

The default max tokens for generation is 4096 unless overridden with `llm.WithMaxTokens()`.

**Environment variables:**

| Variable             | Maps to  |
|----------------------|----------|
| `ANTHROPIC_API_KEY`  | `APIKey` |

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
    _ "github.com/lookatitude/beluga-ai/llm/providers/anthropic"
)

func main() {
    model, err := llm.New("anthropic", config.ProviderConfig{
        Model:  "claude-sonnet-4-5-20250929",
        APIKey: os.Getenv("ANTHROPIC_API_KEY"),
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

Stream chunks include tool call start events (`content_block_start`) and delta events (`content_block_delta`) for incremental tool argument delivery.

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

Anthropic supports the following tool choice modes:

| Beluga ToolChoice         | Anthropic Equivalent |
|---------------------------|----------------------|
| `llm.ToolChoiceAuto`     | `auto`               |
| `llm.ToolChoiceNone`     | `none`               |
| `llm.ToolChoiceRequired` | `any`                |
| `llm.WithSpecificTool()` | Named tool           |

### Vision (Multimodal)

```go
msgs := []schema.Message{
    schema.NewHumanMessageWithParts(
        schema.TextPart{Text: "What's in this image?"},
        schema.ImagePart{URL: "https://example.com/image.png"},
    ),
}

resp, err := model.Generate(ctx, msgs)
```

Base64-encoded images are also supported:

```go
schema.ImagePart{
    Data:     imageBytes,
    MimeType: "image/png",
}
```

### System Messages

Anthropic handles system messages separately from the conversation. The provider automatically extracts `schema.SystemMessage` entries and sends them via the `system` parameter:

```go
msgs := []schema.Message{
    schema.NewSystemMessage("You are a code reviewer."),
    schema.NewHumanMessage("Review this function..."),
}
```

### Generation Options

```go
resp, err := model.Generate(ctx, msgs,
    llm.WithTemperature(0.7),
    llm.WithMaxTokens(8192),
    llm.WithTopP(0.9),
    llm.WithStopSequences("\n\nHuman:"),
)
```

## Error Handling

```go
resp, err := model.Generate(ctx, msgs)
if err != nil {
    // Errors are wrapped with the "anthropic:" prefix
    log.Fatal(err)
}
```

The response includes token usage information:

```go
fmt.Printf("Input: %d, Output: %d, Cached: %d\n",
    resp.Usage.InputTokens,
    resp.Usage.OutputTokens,
    resp.Usage.CachedTokens,
)
```

## Direct Construction

```go
import "github.com/lookatitude/beluga-ai/llm/providers/anthropic"

model, err := anthropic.New(config.ProviderConfig{
    Model:  "claude-sonnet-4-5-20250929",
    APIKey: os.Getenv("ANTHROPIC_API_KEY"),
})
```

## Available Models

| Model ID                          | Description                          |
|-----------------------------------|--------------------------------------|
| `claude-opus-4-0-20250514`        | Most capable Claude model            |
| `claude-sonnet-4-5-20250929`      | Balanced performance and cost        |
| `claude-haiku-3-5-20241022`       | Fast, cost-effective model           |

Refer to [Anthropic's model documentation](https://docs.anthropic.com/en/docs/about-claude/models) for the latest model list.
