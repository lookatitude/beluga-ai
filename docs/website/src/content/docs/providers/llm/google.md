---
title: "Google Gemini"
description: "Integration guide for Google Gemini models with Beluga AI."
---

The Google provider connects Beluga AI to Google's Gemini family of models using the official `google.golang.org/genai` SDK. It supports chat completions, streaming, tool calling, vision, and system instructions.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/llm/providers/google
```

## Configuration

| Field    | Required | Default | Description                             |
|----------|----------|---------|-----------------------------------------|
| `Model`  | Yes      | —       | Model ID (e.g. `"gemini-2.5-flash"`)   |
| `APIKey` | Yes      | —       | Google AI API key                       |
| `BaseURL`| No       | Gemini API default | Override API endpoint        |
| `Timeout`| No       | `30s`   | Request timeout                         |

**Environment variables:**

| Variable                  | Maps to  |
|---------------------------|----------|
| `GOOGLE_API_KEY`          | `APIKey` |
| `GOOGLE_GENAI_API_KEY`    | `APIKey` |

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
    _ "github.com/lookatitude/beluga-ai/llm/providers/google"
)

func main() {
    model, err := llm.New("google", config.ProviderConfig{
        Model:  "gemini-2.5-flash",
        APIKey: os.Getenv("GOOGLE_API_KEY"),
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

Gemini supports the following tool choice modes:

| Beluga ToolChoice         | Gemini Equivalent  |
|---------------------------|--------------------|
| `llm.ToolChoiceAuto`     | `AUTO`             |
| `llm.ToolChoiceNone`     | `NONE`             |
| `llm.ToolChoiceRequired` | `ANY`              |
| `llm.WithSpecificTool()` | `ANY` + allowed function names |

### Vision (Multimodal)

```go
msgs := []schema.Message{
    schema.NewHumanMessageWithParts(
        schema.TextPart{Text: "Describe this image."},
        schema.ImagePart{
            Data:     imageBytes,
            MimeType: "image/png",
        },
    ),
}

resp, err := model.Generate(ctx, msgs)
```

File URIs are also supported for images stored in Google Cloud:

```go
schema.ImagePart{URL: "gs://bucket/image.png"}
```

### System Instructions

System messages are automatically mapped to Gemini's `SystemInstruction` parameter:

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
    llm.WithMaxTokens(4096),
    llm.WithTopP(0.9),
    llm.WithStopSequences("END"),
)
```

## Error Handling

```go
resp, err := model.Generate(ctx, msgs)
if err != nil {
    // Errors are wrapped with the "google:" prefix
    log.Fatal(err)
}
```

Token usage is available on the response:

```go
fmt.Printf("Input: %d, Output: %d, Cached: %d\n",
    resp.Usage.InputTokens,
    resp.Usage.OutputTokens,
    resp.Usage.CachedTokens,
)
```

## Direct Construction

```go
import "github.com/lookatitude/beluga-ai/llm/providers/google"

model, err := google.New(config.ProviderConfig{
    Model:  "gemini-2.5-flash",
    APIKey: os.Getenv("GOOGLE_API_KEY"),
})
```

For testing with a custom HTTP client:

```go
model, err := google.NewWithHTTPClient(cfg, httpClient)
```

## Available Models

| Model ID              | Description                       |
|-----------------------|-----------------------------------|
| `gemini-2.5-pro`     | Most capable Gemini model         |
| `gemini-2.5-flash`   | Fast, balanced model              |
| `gemini-2.0-flash`   | Previous generation fast model    |

Refer to [Google AI's model documentation](https://ai.google.dev/gemini-api/docs/models) for the latest model list.
