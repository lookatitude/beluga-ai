---
title: Quick Start
description: Build your first AI agent with Beluga AI v2 in 5 minutes — from installation to a working agent with tools and memory.
---

This guide walks you through building a working AI agent with tools and streaming in under 5 minutes. By the end, you'll have an agent that can answer questions and use tools to perform actions.

## Prerequisites

- Go 1.23+ installed ([download](https://go.dev/dl/))
- An OpenAI API key (or any [supported provider](/getting-started/installation/#all-available-providers))

## Step 1: Create a New Project

```bash
mkdir my-agent && cd my-agent
go mod init my-agent
```

## Step 2: Install Beluga AI

```bash
go get github.com/lookatitude/beluga-ai@latest
```

## Step 3: Set Your API Key

```bash
export OPENAI_API_KEY="sk-..."
```

## Step 4: Create a Simple Chat

Create `main.go` with a direct LLM call to verify everything works:

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

func main() {
    // Create an LLM instance
    model, err := llm.New("openai", config.ProviderConfig{
        APIKey: os.Getenv("OPENAI_API_KEY"),
        Model:  "gpt-4o",
    })
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }

    // Send a message
    ctx := context.Background()
    resp, err := model.Generate(ctx, []schema.Message{
        schema.NewSystemMessage("You are a helpful assistant."),
        schema.NewHumanMessage("What is Go best known for?"),
    })
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }

    fmt.Println(resp.Text())
}
```

```bash
go mod tidy && go run main.go
```

## Step 5: Stream the Response

Replace the `Generate` call with streaming to see tokens arrive in real time:

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

func main() {
    model, err := llm.New("openai", config.ProviderConfig{
        APIKey: os.Getenv("OPENAI_API_KEY"),
        Model:  "gpt-4o",
    })
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }

    ctx := context.Background()
    msgs := []schema.Message{
        schema.NewSystemMessage("You are a helpful assistant."),
        schema.NewHumanMessage("Explain goroutines in 3 sentences."),
    }

    // Stream returns iter.Seq2[schema.StreamChunk, error]
    for chunk, err := range model.Stream(ctx, msgs) {
        if err != nil {
            fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
            break
        }
        fmt.Print(chunk.Delta)
    }
    fmt.Println()
}
```

## Step 6: Build an Agent with Tools

Now let's build a proper agent. Agents combine an LLM, a persona, and tools into a reasoning loop that can decide when to call tools and how to use their results.

```go
package main

import (
    "context"
    "fmt"
    "math"
    "os"
    "strconv"

    "github.com/lookatitude/beluga-ai/agent"
    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/tool"
    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

// CalculateInput defines the tool's input parameters.
// Struct tags generate the JSON Schema sent to the LLM.
type CalculateInput struct {
    Expression string `json:"expression" description:"A math expression like '2+2' or 'sqrt(16)'" required:"true"`
}

func main() {
    // 1. Create the LLM
    model, err := llm.New("openai", config.ProviderConfig{
        APIKey: os.Getenv("OPENAI_API_KEY"),
        Model:  "gpt-4o",
    })
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }

    // 2. Create a tool — wraps a Go function with auto-generated JSON Schema
    calculator := tool.NewFuncTool("calculate", "Evaluate a math expression",
        func(ctx context.Context, input CalculateInput) (*tool.Result, error) {
            // Simple calculator for demonstration
            result := evalExpression(input.Expression)
            return tool.TextResult(fmt.Sprintf("Result: %s", result)), nil
        },
    )

    // 3. Create an agent with persona, LLM, and tools
    assistant := agent.New("math-assistant",
        agent.WithLLM(model),
        agent.WithTools([]tool.Tool{calculator}),
        agent.WithPersona(agent.Persona{
            Role: "math tutor",
            Goal: "help users solve math problems step by step",
        }),
    )

    // 4. Invoke the agent
    ctx := context.Background()
    result, err := assistant.Invoke(ctx, "What is the square root of 144?")
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }

    fmt.Println(result)
}

func evalExpression(expr string) string {
    // Simplified — in production, use a proper expression parser
    if expr == "sqrt(144)" || expr == "√144" {
        return "12"
    }
    f, err := strconv.ParseFloat(expr, 64)
    if err == nil {
        return fmt.Sprintf("%g", math.Sqrt(f))
    }
    return "Unable to evaluate: " + expr
}
```

The agent's reasoning loop (ReAct by default) will:
1. Read the user's question
2. Decide to call the `calculate` tool
3. Receive the tool result
4. Formulate a response incorporating the result

## Step 7: Stream Agent Events

For real-time feedback, use `Stream` instead of `Invoke` to see each step of the reasoning loop:

```go
ctx := context.Background()
for event, err := range assistant.Stream(ctx, "What is the square root of 144?") {
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        break
    }
    switch event.Type {
    case agent.EventText:
        fmt.Print(event.Text)
    case agent.EventToolCall:
        fmt.Printf("\n[Calling tool: %s]\n", event.ToolCall.Name)
    case agent.EventToolResult:
        fmt.Printf("[Tool result received]\n")
    }
}
fmt.Println()
```

## Step 8: Add Multiple Tools

Agents become powerful when they have access to multiple tools. The LLM decides which tool to use based on the user's input:

```go
type WeatherInput struct {
    City string `json:"city" description:"City name" required:"true"`
}

type SearchInput struct {
    Query string `json:"query" description:"Search query" required:"true"`
    Limit int    `json:"limit" description:"Max results" default:"5"`
}

weather := tool.NewFuncTool("get_weather", "Get current weather for a city",
    func(ctx context.Context, input WeatherInput) (*tool.Result, error) {
        // In production, call a real weather API
        return tool.TextResult(fmt.Sprintf("72°F and sunny in %s", input.City)), nil
    },
)

search := tool.NewFuncTool("web_search", "Search the web for information",
    func(ctx context.Context, input SearchInput) (*tool.Result, error) {
        // In production, call a real search API
        return tool.TextResult(fmt.Sprintf("Top results for '%s': ...", input.Query)), nil
    },
)

assistant := agent.New("research-assistant",
    agent.WithLLM(model),
    agent.WithTools([]tool.Tool{calculator, weather, search}),
    agent.WithPersona(agent.Persona{
        Role: "research assistant",
        Goal: "help users find information and answer questions",
    }),
)
```

## What's Next?

You've built a working agent with tools and streaming. Here's where to go from here:

| Topic | Guide |
|-------|-------|
| In-depth agent tutorial | [Building Your First Agent](/guides/first-agent/) |
| LLM configuration and routing | [Working with LLMs](/guides/working-with-llms/) |
| RAG and document retrieval | [RAG Pipeline](/guides/rag-pipeline/) |
| Conversation memory | [Memory System](/guides/memory-system/) |
| Voice AI applications | [Voice AI Pipeline](/guides/voice-ai/) |
| MCP and remote tools | [Tools & MCP](/guides/tools-and-mcp/) |
| Multi-agent orchestration | [Orchestration & Workflows](/guides/orchestration/) |
| Production deployment | [Deploying to Production](/guides/deployment/) |
