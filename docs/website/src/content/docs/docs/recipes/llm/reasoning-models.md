---
title: "Reasoning Models"
description: "Recipe for using reasoning/chain-of-thought models (OpenAI o-series, Claude extended thinking) with Beluga AI in Go — configure effort, stream thinking, and track reasoning tokens."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, reasoning models, chain-of-thought, OpenAI o3, Claude thinking, extended thinking, reasoning tokens, Go LLM"
---

## Problem

You want to use reasoning models that expose their chain-of-thought process
(such as OpenAI o3/o4-mini or Claude with extended thinking), control how much
reasoning they perform, and observe or display the thinking process in real time.

## Solution

Beluga AI provides first-class support for reasoning models through
`ReasoningConfig`, streaming `ReasoningDelta`, the `ThinkingPart` content type,
and the `OnReasoning` hook.

### Basic Usage with ReasoningEffort

Set the reasoning effort level to control how much the model thinks before
answering:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"

    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

func main() {
    ctx := context.Background()

    model, err := llm.New("openai", config.ProviderConfig{
        APIKey: "sk-...",
        Model:  "o3",
    })
    if err != nil {
        log.Fatal(err)
    }

    msgs := []schema.Message{
        schema.NewHumanMessage("Prove that the square root of 2 is irrational."),
    }

    resp, err := model.Generate(ctx, msgs,
        llm.WithReasoningEffort(llm.ReasoningEffortHigh),
    )
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(resp.Text())
    fmt.Printf("Reasoning tokens used: %d\n", resp.Usage.ReasoningTokens)
}
```

### Setting a Token Budget

Limit the number of tokens the model can spend on reasoning:

```go
resp, err := model.Generate(ctx, msgs,
    llm.WithReasoning(llm.ReasoningConfig{
        Effort:       llm.ReasoningEffortHigh,
        BudgetTokens: 10000,
    }),
)
```

Or use the individual options which compose correctly:

```go
resp, err := model.Generate(ctx, msgs,
    llm.WithReasoningEffort(llm.ReasoningEffortMedium),
    llm.WithReasoningBudget(5000),
)
```

### Streaming Reasoning Output

Stream the model's thinking process to the console in real time:

```go
for chunk, err := range model.Stream(ctx, msgs,
    llm.WithReasoningEffort(llm.ReasoningEffortHigh),
) {
    if err != nil {
        log.Fatal(err)
    }

    // Print reasoning deltas (the model's thinking)
    if chunk.ReasoningDelta != "" {
        fmt.Fprintf(os.Stderr, "[thinking] %s", chunk.ReasoningDelta)
    }

    // Print the final answer deltas
    if chunk.Delta != "" {
        fmt.Print(chunk.Delta)
    }
}
```

### Using the OnReasoning Hook

The `OnReasoning` hook fires for each reasoning delta during streaming,
keeping reasoning observation separate from your main stream consumer:

```go
hooks := llm.Hooks{
    OnReasoning: func(ctx context.Context, delta string) {
        // Log reasoning to a file, send to a UI, etc.
        fmt.Fprintf(reasoningLog, "%s", delta)
    },
}

model = llm.ApplyMiddleware(model, llm.WithHooks(hooks))

// Now stream normally — the hook captures reasoning separately
for chunk, err := range model.Stream(ctx, msgs,
    llm.WithReasoningEffort(llm.ReasoningEffortHigh),
) {
    if err != nil {
        log.Fatal(err)
    }
    fmt.Print(chunk.Delta)
}
```

### Accessing Thinking Content in Responses

After a non-streaming `Generate` call, reasoning content appears as
`ThinkingPart` in the response's content parts:

```go
resp, err := model.Generate(ctx, msgs,
    llm.WithReasoningEffort(llm.ReasoningEffortHigh),
)
if err != nil {
    log.Fatal(err)
}

for _, part := range resp.Parts {
    switch p := part.(type) {
    case schema.ThinkingPart:
        fmt.Printf("[Reasoning]\n%s\n\n", p.Text)
    case schema.TextPart:
        fmt.Printf("[Answer]\n%s\n", p.Text)
    }
}
```

### Claude with Extended Thinking

The same API works with Anthropic's Claude models that support extended
thinking:

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/anthropic"

model, err := llm.New("anthropic", config.ProviderConfig{
    APIKey: "sk-ant-...",
    Model:  "claude-sonnet-4-20250514",
})
if err != nil {
    log.Fatal(err)
}

resp, err := model.Generate(ctx, msgs,
    llm.WithReasoningBudget(16000),
)
```

## How It Works

1. `ReasoningConfig` is passed through `GenerateOptions` to the provider.
2. The provider maps `Effort` and `BudgetTokens` to its native API parameters.
3. During streaming, providers emit `ReasoningDelta` in `StreamChunk`.
4. After completion, providers include `ThinkingPart` in the response's `Parts`
   and report `ReasoningTokens` in `Usage`.
5. The `OnReasoning` hook in `llm.Hooks` fires for each non-empty
   `ReasoningDelta` during streaming.
6. Observability attributes `gen_ai.usage.reasoning_tokens` and
   `gen_ai.request.reasoning_effort` are available for tracing.

## Related

- [LLM API Reference](/docs/reference/api/llm-agents/llm) -- full `ChatModel` API
- [Streaming Tool Calls](/docs/recipes/llm/streaming-tool-calls) -- handling tool calls in streams
- [Token Counting](/docs/recipes/llm/token-counting) -- tracking token usage
