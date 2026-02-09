---
title: Advanced Inference Options
description: Fine-tune LLM generation with temperature, sampling, penalties, and response format controls.
---

Default LLM settings work for general queries, but specific tasks require tuning. Code generation needs low temperature for determinism. Creative writing benefits from higher temperature for variety. Beluga AI v2 provides per-call `GenerateOption` functions that control these parameters across all providers through a unified API.

## What You Will Build

Configurations for different use cases — deterministic code generation, creative writing, and structured JSON output — using Beluga AI's `GenerateOption` system.

## Prerequisites

- A configured LLM provider (OpenAI, Anthropic, or any registered provider)
- Understanding of the [ChatModel interface](/guides/llm)

## Generate Options

Options are applied per-call to `Generate` or `Stream`:

```go
resp, err := model.Generate(ctx, msgs,
    llm.WithTemperature(0.2),
    llm.WithMaxTokens(500),
)
```

All options are collected into a `GenerateOptions` struct that providers read to configure their API requests.

## Step 1: Temperature and TopP

Temperature controls randomness. TopP (nucleus sampling) restricts the token pool to the top cumulative probability mass.

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/llm"
    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
    "github.com/lookatitude/beluga-ai/schema"
)

func main() {
    ctx := context.Background()

    model, err := llm.New("openai", config.ProviderConfig{
        "model":   "gpt-4o",
        "api_key": os.Getenv("OPENAI_API_KEY"),
    })
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    msgs := []schema.Message{
        schema.NewHumanMessage("Write a haiku about Go programming"),
    }

    // High creativity: temperature 1.2, TopP 0.9
    creative, err := model.Generate(ctx, msgs,
        llm.WithTemperature(1.2),
        llm.WithTopP(0.9),
    )
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    fmt.Printf("Creative: %s\n\n", creative.Text())

    // Deterministic: temperature 0
    deterministic, err := model.Generate(ctx, msgs,
        llm.WithTemperature(0.0),
    )
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    fmt.Printf("Deterministic: %s\n", deterministic.Text())
}
```

Temperature guidelines:
- **0.0** — Nearly deterministic. Use for code generation, data extraction, classification.
- **0.3-0.7** — Balanced. Use for Q&A, summarization, analysis.
- **0.8-1.5** — Creative. Use for writing, brainstorming, storytelling.

## Step 2: Max Tokens and Stop Sequences

Control output length and stopping conditions:

```go
// Limit response to 100 tokens
resp, err := model.Generate(ctx, msgs,
    llm.WithMaxTokens(100),
)

// Stop generating when specific sequences appear
resp, err = model.Generate(ctx, msgs,
    llm.WithStopSequences("```", "\n\n"),
)
```

## Step 3: Structured JSON Output

Force the model to produce valid JSON using `ResponseFormat`:

```go
// JSON mode — model returns valid JSON
resp, err := model.Generate(ctx, msgs,
    llm.WithResponseFormat(llm.ResponseFormat{
        Type: "json_object",
    }),
)

// JSON Schema — model output conforms to a specific schema
resp, err = model.Generate(ctx, msgs,
    llm.WithResponseFormat(llm.ResponseFormat{
        Type: "json_schema",
        Schema: map[string]any{
            "type": "object",
            "properties": map[string]any{
                "sentiment": map[string]any{
                    "type": "string",
                    "enum": []any{"positive", "negative", "neutral"},
                },
                "confidence": map[string]any{
                    "type": "number",
                },
            },
            "required": []any{"sentiment", "confidence"},
        },
    }),
)
```

## Step 4: Tool Choice Control

When tools are bound to a model, control how the model selects them:

```go
// Let the model decide (default)
resp, err := model.Generate(ctx, msgs, llm.WithToolChoice(llm.ToolChoiceAuto))

// Prevent tool calls
resp, err = model.Generate(ctx, msgs, llm.WithToolChoice(llm.ToolChoiceNone))

// Force at least one tool call
resp, err = model.Generate(ctx, msgs, llm.WithToolChoice(llm.ToolChoiceRequired))

// Force a specific tool
resp, err = model.Generate(ctx, msgs, llm.WithSpecificTool("calculator"))
```

## Step 5: Provider-Specific Options

Use `WithMetadata` for options that are specific to a particular provider:

```go
resp, err := model.Generate(ctx, msgs,
    llm.WithTemperature(0.5),
    llm.WithMetadata(map[string]any{
        "frequency_penalty": 0.5,
        "presence_penalty":  0.3,
        "seed":              42,
    }),
)
```

## Recommended Configurations by Task

| Task | Temperature | Max Tokens | Other |
|:---|:---|:---|:---|
| Code generation | 0.0 | 2000 | Stop on `\`\`\`` |
| Data extraction | 0.0 | 500 | JSON mode |
| Q&A | 0.3 | 1000 | — |
| Summarization | 0.5 | 500 | — |
| Creative writing | 1.0 | 2000 | TopP 0.9 |
| Brainstorming | 1.2 | 1000 | Presence penalty 0.5 |

## Verification

1. Generate responses with temperature 0.0 — verify they are consistent across runs.
2. Generate with temperature 1.2 — verify more variation.
3. Use JSON mode — verify the response is valid JSON.

## Next Steps

- [Adding a New LLM Provider](/tutorials/providers/new-llm-provider) — Extend the framework with custom providers
- [Multi-provider Chat](/tutorials/agents/multi-provider) — Use multiple providers together
