---
title: Multi-Provider Chat Integration
description: "Build Go applications that use multiple LLM providers interchangeably through Beluga AI's unified ChatModel interface with normalized tool calling."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go, tutorial, multi-provider, LLM, ChatModel, OpenAI, Anthropic, provider switching"
---

In a fast-moving AI landscape, locking into a single vendor creates risk -- risk of outages, pricing changes, capability gaps, and vendor lock-in. Beluga AI addresses this by defining a single `ChatModel` interface that every provider implements identically. Tool calls, streaming, and configuration all work the same way regardless of whether you are talking to OpenAI, Anthropic, Google, or any other registered provider. This design follows the principle that application logic should never contain provider-specific code.

## What You Will Build

An application that uses OpenAI and Anthropic interchangeably, with normalized tool calling and provider-agnostic application code.

## Prerequisites

- API keys for at least two providers (OpenAI, Anthropic, or others)
- Understanding of the [registry pattern](/guides/architecture)

## Step 1: Create Models from the Registry

Beluga AI uses the **registry pattern** (`Register()` + `New()` + `List()`) for all extensible components. Each provider package registers itself via Go's `init()` function, which means simply importing the package (with a blank identifier `_`) is enough to make the provider available at runtime. This design eliminates the need for hardcoded factory functions or manual wiring -- the registry handles discovery and instantiation.

The `llm.New()` function looks up the provider by name and returns a `ChatModel` interface. From that point forward, all operations are provider-agnostic. The `llm.List()` function lets you discover which providers are available at runtime, which is useful for configuration-driven deployments where provider selection happens through environment variables or config files rather than code changes.

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/llm"
    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
    _ "github.com/lookatitude/beluga-ai/llm/providers/anthropic"
    "github.com/lookatitude/beluga-ai/schema"
)

func main() {
    ctx := context.Background()

    // Discover available providers
    fmt.Println("Registered providers:", llm.List())

    // Create models from the registry
    gpt4, err := llm.New("openai", config.ProviderConfig{
        "api_key": os.Getenv("OPENAI_API_KEY"),
        "model":   "gpt-4o",
    })
    if err != nil {
        fmt.Printf("OpenAI error: %v\n", err)
        return
    }

    claude, err := llm.New("anthropic", config.ProviderConfig{
        "api_key": os.Getenv("ANTHROPIC_API_KEY"),
        "model":   "claude-sonnet-4-5-20250929",
    })
    if err != nil {
        fmt.Printf("Anthropic error: %v\n", err)
        return
    }

    // Same function works with any provider
    generate(ctx, gpt4, "Explain channels in Go.")
    generate(ctx, claude, "Explain channels in Go.")
}

// generate works with any ChatModel — provider agnostic
func generate(ctx context.Context, model llm.ChatModel, prompt string) {
    msgs := []schema.Message{
        schema.NewHumanMessage(prompt),
    }

    resp, err := model.Generate(ctx, msgs)
    if err != nil {
        fmt.Printf("[%s] Error: %v\n", model.ModelID(), err)
        return
    }

    fmt.Printf("[%s] %s\n\n", model.ModelID(), resp.Text())
    fmt.Printf("  Tokens: %d input, %d output\n", resp.Usage.InputTokens, resp.Usage.OutputTokens)
}
```

## Step 2: Normalized Tool Calling

Each LLM provider uses a different JSON format for tool definitions and tool call responses. OpenAI uses a `functions` array, Anthropic uses a `tools` array with a different structure, and Google uses yet another format. Beluga AI normalizes all of this behind `schema.ToolDefinition` -- you define tools once using a standard JSON Schema format, and the provider adapter translates it into the wire format each API expects. This is why `BindTools` returns a new `ChatModel` rather than modifying the original: it wraps the model with tool-aware behavior while preserving immutability.

```go
calcTool := schema.ToolDefinition{
    Name:        "calculator",
    Description: "Evaluate a mathematical expression.",
    InputSchema: map[string]any{
        "type": "object",
        "properties": map[string]any{
            "expression": map[string]any{
                "type":        "string",
                "description": "Math expression to evaluate",
            },
        },
        "required": []any{"expression"},
    },
}

// Same tool definition works for both providers
gptWithTools := gpt4.BindTools([]schema.ToolDefinition{calcTool})
claudeWithTools := claude.BindTools([]schema.ToolDefinition{calcTool})
```

## Step 3: Streaming Across Providers

Beluga AI uses `iter.Seq2[schema.StreamChunk, error]` for all streaming operations. This is a Go 1.23+ iterator type that provides a pull-based consumption model with standard `for...range` syntax. Unlike channel-based streaming, iterators compose naturally, support early termination via `break`, and propagate context cancellation without goroutine leaks. Every provider implements the same streaming contract, so your streaming consumption code works identically regardless of the backend.

```go
func streamResponse(ctx context.Context, model llm.ChatModel, prompt string) {
    msgs := []schema.Message{
        schema.NewHumanMessage(prompt),
    }

    fmt.Printf("[%s] ", model.ModelID())
    for chunk, err := range model.Stream(ctx, msgs) {
        if err != nil {
            fmt.Printf("\nStream error: %v\n", err)
            return
        }
        fmt.Print(chunk.Delta)
    }
    fmt.Println()
}
```

## Step 4: Provider Registry for Dynamic Selection

In production systems, you often need to select a provider at runtime based on user configuration, request metadata, or cost constraints. A `ModelPool` provides a simple lookup layer on top of the registry-created models. This pattern separates model creation (which happens at startup) from model selection (which happens per-request), giving you flexibility to route different requests to different providers without changing application logic.

```go
type ModelPool struct {
    models map[string]llm.ChatModel
}

func NewModelPool() *ModelPool {
    return &ModelPool{models: make(map[string]llm.ChatModel)}
}

func (p *ModelPool) Add(name string, model llm.ChatModel) {
    p.models[name] = model
}

func (p *ModelPool) Get(name string) (llm.ChatModel, error) {
    m, ok := p.models[name]
    if !ok {
        return nil, fmt.Errorf("model %q not registered in pool", name)
    }
    return m, nil
}

// Usage: select model at runtime based on configuration or request
func handleRequest(ctx context.Context, pool *ModelPool, providerName, prompt string) (string, error) {
    model, err := pool.Get(providerName)
    if err != nil {
        return "", err
    }

    resp, err := model.Generate(ctx, []schema.Message{
        schema.NewHumanMessage(prompt),
    })
    if err != nil {
        return "", err
    }
    return resp.Text(), nil
}
```

## Verification

1. Send the same prompt to OpenAI and Anthropic -- verify both return valid responses.
2. Bind the same tool to both models -- verify both generate valid tool calls with the same schema.
3. Stream from both providers -- verify the `iter.Seq2` pattern works identically.

## Next Steps

- [Model Switching and Fallbacks](/tutorials/agents/model-switching) -- Automate provider selection
- [Adding a New LLM Provider](/tutorials/providers/new-llm-provider) -- Register custom providers
