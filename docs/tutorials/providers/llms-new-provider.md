# Adding a New LLM Provider

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this guide, we'll build a custom **LLM Provider** from scratch. We will implement the `ChatModel` interface and plug it into the framework so it works seamlessly with Agents, Chains, and the standard configuration system.

## Learning Objectives
By the end of this tutorial, you will:
1.  Understand the core `ChatModel` interface.
2.  Implement a wrapper for a hypothetical API ("MyCustomAI").
3.  Register your provider with `pkg/llms`.
4.  Use it in an Agent.

## Introduction
Welcome, colleague! Beluga AI supports major LLMs like OpenAI and Anthropic out of the box, but the AI landscape changes weekly. Maybe you want to use a niche provider like **Cohere**, a local model via **llama.cpp**, or even your own internal fine-tuned model service.

## Why This Matters

*   **Future-Proofing**: Don't wait for the framework to support the latest hot model. Add it yourself today.
*   **Vendor Neutrality**: Switch between OpenAI and your custom on-prem model just by changing a config string.
*   **Testing**: Create specialized mock providers for chaos testing or regression suites.

## Prerequisites

*   A working Go environment.
*   Understanding of `pkg/schema` (Messages).
*   Familiarity with functional options (the pattern used for configuration).

## Concepts

### The ChatModel Interface
To be an LLM in Beluga AI, you must implement `pkg/llms/iface.ChatModel`.
```go
type ChatModel interface {
    // 1. The Core: Input messages -> Output message
    Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error)
    
    // 2. Streaming: Input messages -> Channel of chunks
    StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan AIMessageChunk, error)
    
    // 3. Identification
    GetModelName() string
}
```

## Step-by-Step Implementation

### Step 1: The Provider Struct

Let's create a new package `myprovider` and define our struct.
```go
package myprovider

import (
    "context"
    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

type Provider struct {
    client    *SomeAPIClient // Your internal API client
    modelName string
}

// Ensure interface compliance
var _ iface.ChatModel = (*Provider)(nil)

func NewProvider(opts ...llms.ConfigOption) (*Provider, error) {
    // 1. Initialize default config
    cfg := llms.DefaultConfig()
    
    // 2. Apply options (e.g., API Key, Model Name)
    for _, opt := range opts {
        opt(cfg)
    }
    
    // 3. Create the provider
    return &Provider{
        client:    NewClient(cfg.APIKey),
        modelName: cfg.ModelName,
    }, nil
}
```

### Step 2: Implementing Generate

This is where the magic happens. We translate Beluga AI's standard `[]schema.Message` into whatever format your API expects.
```go
func (p *Provider) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
    // 1. Convert Messages to API format
    apiMessages := convertToAPIMessages(messages)
    
    // 2. Call your API
    resp, err := p.client.CreateCompletion(ctx, req)
    if err != nil {
        return nil, err
    }
    
    // 3. Convert API response back to Beluga Message
    return schema.NewAIMessage(resp.Text), nil
}

func (p *Provider) GetModelName() string {
    return p.modelName
}
```

### Step 3: Registering with the Framework

To make your provider accessible via `llms.NewProvider("myproxy")`, you need to register it.
```text
import "github.com/lookatitude/beluga-ai/pkg/llms"
go
func init() {
    // Register the factory function
    llms.GetRegistry().Register("mycustomai", func(cfg *llms.Config) (iface.ChatModel, error) {
        return NewProvider(
            llms.WithAPIKey(cfg.APIKey),
            llms.WithModelName(cfg.ModelName),
        )
    })
}
```

### Step 4: Using Your New Provider

Now you can use it just like OpenAI or Anthropic!
```go
package main

import (
    _ "myproject/pkg/llms/myprovider" // Import for side-effects (registration)
    "github.com/lookatitude/beluga-ai/pkg/llms"
)

func main() {
    // Use the string name you registered
    model, err := llms.NewProvider(context.Background(), "mycustomai",
        llms.WithAPIKey("secret-key"),
        llms.WithModelName("super-model-v1"),
    )
    
    // Identify the component
    fmt.Println("Using model:", model.GetModelName())
    
    // Generate !
    response, _ := model.Generate(ctx, messages)
}
```

## Pro-Tips

*   **Helper Functions**: Use `llms/internal/common` if you need retry logic or HTTP client standardization.
*   **Streaming**: If your API doesn't support streaming, you can implement `StreamChat` by simply calling `Generate` and emitting a single chunk. This is valid!
*   **Config Validation**: In your `NewProvider`, check `cfg.APIKey` immediately and return an `ErrInvalidConfig` if it's missing.

## Troubleshooting

### "Provider not registered"
Make sure you are actually importing your package! In Go, if a package isn't used, `init()` won't run. Use the blank identifier import:
`import _ "path/to/myprovider"`

### "Interface compliance failed"
You might be missing a method. Typically `StreamChat` or `BindTools` are forgotten. If you don't support tools yet, just return the receiver and do nothing:

```go
func (p *Provider) BindTools(tools []tools.Tool) iface.ChatModel {
    // Not supported yet, just ignore
    return p
}
```

## Conclusion

You have just extended the capability of the Beluga AI Framework. By wrapping your custom API in the `ChatModel` interface, you gain access to the entire ecosystem of Agents, Chains, and Memory without writing any glue code. Your bespoke model is now a first-class citizen.
