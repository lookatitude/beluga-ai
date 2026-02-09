---
title: Adding a New LLM Provider
description: Implement the ChatModel interface and register a custom LLM provider with Beluga AI's registry.
---

Beluga AI supports major LLM providers out of the box, but the AI landscape evolves rapidly. By implementing the `ChatModel` interface and registering your provider, you create a first-class citizen that works seamlessly with agents, middleware, routing, and structured output — all without modifying framework code.

## What You Will Build

A custom LLM provider that implements `ChatModel`, registers with the provider registry, and integrates with Beluga AI's middleware and routing infrastructure.

## Prerequisites

- Understanding of the [ChatModel interface](/guides/llm)
- Familiarity with Go interfaces and the [registry pattern](/guides/architecture)

## The ChatModel Interface

Every LLM provider implements this interface:

```go
type ChatModel interface {
    Generate(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error)
    Stream(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) iter.Seq2[schema.StreamChunk, error]
    BindTools(tools []schema.ToolDefinition) ChatModel
    ModelID() string
}
```

## Step 1: Define the Provider

Create a package for your provider under `llm/providers/`:

```go
package mycustom

import (
    "context"
    "fmt"
    "iter"
    "net/http"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
)

// Model implements llm.ChatModel for the MyCustomAI API.
type Model struct {
    apiKey  string
    model   string
    client  *http.Client
    tools   []schema.ToolDefinition
}

// Compile-time interface check.
var _ llm.ChatModel = (*Model)(nil)

func New(cfg config.ProviderConfig) (*Model, error) {
    apiKey, _ := cfg["api_key"].(string)
    if apiKey == "" {
        return nil, fmt.Errorf("mycustom: api_key is required")
    }

    model, _ := cfg["model"].(string)
    if model == "" {
        model = "mycustom-default"
    }

    return &Model{
        apiKey: apiKey,
        model:  model,
        client: &http.Client{},
    }, nil
}
```

## Step 2: Implement Generate

Convert Beluga AI messages to your API format, call the API, and convert the response back:

```go
func (m *Model) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
    options := llm.ApplyOptions(opts...)

    // Convert messages to your API's format
    apiReq := m.buildRequest(msgs, options)

    // Call your API
    apiResp, err := m.callAPI(ctx, apiReq)
    if err != nil {
        return nil, fmt.Errorf("mycustom: generate: %w", err)
    }

    // Convert response to Beluga AI's AIMessage
    resp := &schema.AIMessage{
        Parts: []schema.ContentPart{
            schema.TextPart{Text: apiResp.Text},
        },
        Usage: schema.Usage{
            InputTokens:  apiResp.InputTokens,
            OutputTokens: apiResp.OutputTokens,
            TotalTokens:  apiResp.InputTokens + apiResp.OutputTokens,
        },
        ModelID: m.model,
    }

    // Map tool calls if present
    for _, tc := range apiResp.ToolCalls {
        resp.ToolCalls = append(resp.ToolCalls, schema.ToolCall{
            ID:        tc.ID,
            Name:      tc.FunctionName,
            Arguments: tc.Arguments,
        })
    }

    return resp, nil
}
```

## Step 3: Implement Stream

Return an `iter.Seq2[schema.StreamChunk, error]` iterator. If your API supports server-sent events (SSE), consume them and yield chunks:

```go
func (m *Model) Stream(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
    return func(yield func(schema.StreamChunk, error) bool) {
        options := llm.ApplyOptions(opts...)
        apiReq := m.buildRequest(msgs, options)

        stream, err := m.callStreamAPI(ctx, apiReq)
        if err != nil {
            yield(schema.StreamChunk{}, fmt.Errorf("mycustom: stream: %w", err))
            return
        }
        defer stream.Close()

        for stream.Next() {
            chunk := stream.Value()
            sc := schema.StreamChunk{
                Delta:   chunk.Text,
                ModelID: m.model,
            }

            if !yield(sc, nil) {
                return // consumer stopped iterating
            }
        }

        if err := stream.Err(); err != nil {
            yield(schema.StreamChunk{}, err)
        }
    }
}
```

If your API does not support streaming, implement `Stream` by calling `Generate` and yielding the full response as a single chunk:

```go
func (m *Model) Stream(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
    return func(yield func(schema.StreamChunk, error) bool) {
        resp, err := m.Generate(ctx, msgs, opts...)
        if err != nil {
            yield(schema.StreamChunk{}, err)
            return
        }
        yield(schema.StreamChunk{
            Delta:   resp.Text(),
            ModelID: m.model,
        }, nil)
    }
}
```

## Step 4: Implement BindTools and ModelID

```go
func (m *Model) BindTools(tools []schema.ToolDefinition) llm.ChatModel {
    return &Model{
        apiKey: m.apiKey,
        model:  m.model,
        client: m.client,
        tools:  tools,
    }
}

func (m *Model) ModelID() string {
    return m.model
}
```

`BindTools` returns a new model instance — it does not modify the original. This allows safe concurrent use of the same base model with different tool sets.

## Step 5: Register with the Registry

Register your provider in an `init()` function so it becomes available through `llm.New`:

```go
func init() {
    llm.Register("mycustom", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
        return New(cfg)
    })
}
```

## Step 6: Use Your Provider

Import the provider package for its `init()` side effect, then create instances through the registry:

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/llm"
    _ "myproject/llm/providers/mycustom" // Register via init()
    "github.com/lookatitude/beluga-ai/schema"
)

func main() {
    ctx := context.Background()

    model, err := llm.New("mycustom", config.ProviderConfig{
        "api_key": os.Getenv("MYCUSTOM_API_KEY"),
        "model":   "super-model-v1",
    })
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    fmt.Println("Using model:", model.ModelID())

    // Discover all registered providers
    fmt.Println("Available providers:", llm.List())

    msgs := []schema.Message{
        schema.NewHumanMessage("Hello from my custom provider!"),
    }

    resp, err := model.Generate(ctx, msgs)
    if err != nil {
        fmt.Printf("Generate error: %v\n", err)
        return
    }
    fmt.Println(resp.Text())
}
```

## Troubleshooting

**"unknown provider" error**: Ensure you import the provider package with a blank identifier (`import _ "path/to/mycustom"`). Without the import, Go does not execute `init()` and the provider is not registered.

**Interface compliance error**: Verify you implement all four methods: `Generate`, `Stream`, `BindTools`, and `ModelID`. The compile-time check (`var _ llm.ChatModel = (*Model)(nil)`) catches this at build time.

## Next Steps

- [Advanced Inference Options](/tutorials/providers/advanced-inference) — Control generation parameters
- [Model Switching](/tutorials/agents/model-switching) — Build fallback chains across providers
