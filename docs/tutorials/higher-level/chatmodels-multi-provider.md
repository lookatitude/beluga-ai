# Multi-provider Chat Integration

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this tutorial, we'll build applications that support multiple Chat Model providers (OpenAI, Anthropic, Bedrock) interchangeably using Beluga AI's unified interface.

## Learning Objectives
- ✅ Configure multiple providers
- ✅ Use the unified `ChatModel` interface
- ✅ Normalize tool calls across providers
- ✅ Handle provider-specific quirks

## Introduction
Welcome, colleague! In the fast-moving AI world, being tied to one vendor is a bottleneck. Let's see how Beluga AI makes it easy to swap providers with zero changes to your core logic.

## Prerequisites

- API Keys for at least 2 providers (e.g., OpenAI and Anthropic)

## Why Multi-provider?

- **Reliability**: Fallback if one API goes down.
- **Cost**: Use cheaper models (Haiku/GPT-3.5) for simple tasks, expensive ones (Opus/GPT-4) for complex ones.
- **Features**: Some models are better at coding, others at creative writing.

## Step 1: Unified Configuration
```go
package main

import (
    "github.com/lookatitude/beluga-ai/pkg/chatmodels"
    "github.com/lookatitude/beluga-ai/pkg/llms"
)

func main() {
    // OpenAI
    gpt4 := chatmodels.NewChatModel(llms.NewFactory().CreateProvider(
        "openai", 
        llms.NewConfig(llms.WithModelName("gpt-4")),
    ))
    
    // Anthropic
    claude := chatmodels.NewChatModel(llms.NewFactory().CreateProvider(
        "anthropic", 
        llms.NewConfig(llms.WithModelName("claude-3-opus")),
    ))
    
    // They share the same interface!
    generate(gpt4, "Hello")
    generate(claude, "Hello")
}

func generate(model chatmodels.ChatModel, input string) {
    // ...
}
```

## Step 2: Tool Calling Normalization

Different providers have different JSON schemas for tool calls. Beluga AI normalizes this.
```go
// Define tool once
tool := tools.NewCalculatorTool()



// Bind to both
gpt4.BindTools([]tool)
claude.BindTools([]tool)

```
// The framework handles converting the tool definition to OpenAI format vs Anthropic format

## Step 3: Provider Registry

Build a registry to retrieve models by name.






```go
type ModelRegistry struct \{
    models map[string]chatmodels.ChatModel
}

func (r *ModelRegistry) Get(name string) chatmodels.ChatModel \{
    return r.models[name]
}

## Verification

1. Write a prompt that uses a tool (e.g., "What is 25 * 48?").
2. Run it against OpenAI.
3. Run it against Anthropic.
4. Verify both execute the tool correctly.

## Next Steps

- **[Model Switching & Fallbacks](./chatmodels-model-switching.md)** - Automate the switch
- **[Adding a New LLM Provider](../providers/llms-new-provider.md)** - Add custom providers
