---
title: Custom Tools and Tool Registry
description: Build custom tools and manage them with the tool registry for use in Beluga AI agents, including FuncTool, manual Tool implementations, and registry patterns.
---

Tools give agents the ability to take actions beyond text generation -- querying databases, calling APIs, running calculations, or interacting with external systems. Without tools, an agent can only reason about information already in its context; with tools, it can retrieve live data and modify external state.

Beluga AI provides a composable tool system that lets you define custom tools, register them in a thread-safe registry, and pass them to agents for execution. Tools implement the `tool.Tool` interface and are created either manually or with the type-safe `FuncTool` generic wrapper.

## Overview

The tool system consists of three components:

- **`tool.Tool` interface** — defines `Name()`, `Description()`, `InputSchema()`, and `Execute()`
- **`tool.FuncTool`** — a generic wrapper that turns a typed Go function into a `Tool` with auto-generated JSON Schema
- **`tool.Registry`** — a thread-safe, name-based collection for organizing and discovering tools

Agents receive tools at construction time via `agent.WithTools()` and invoke them during reasoning loops when the LLM produces a tool call.

## Prerequisites

- Go 1.23 or later
- Beluga AI framework installed (`go get github.com/lookatitude/beluga-ai`)
- An LLM provider configured (any provider that supports tool calling)

## Installation

```bash
go get github.com/lookatitude/beluga-ai
```

## Creating Tools with FuncTool

The recommended approach uses `tool.NewFuncTool` with a typed input struct. The JSON Schema is generated automatically from struct tags.

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/agent"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/tool"

    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

// GreetInput defines the tool's input parameters.
// Struct tags control JSON Schema generation.
type GreetInput struct {
    Name string `json:"name" description:"The person to greet" required:"true"`
}

func main() {
    ctx := context.Background()

    // Create a typed tool with auto-generated schema
    greet := tool.NewFuncTool("greet", "Greets a person by name",
        func(ctx context.Context, input GreetInput) (*tool.Result, error) {
            return tool.TextResult(fmt.Sprintf("Hello, %s!", input.Name)), nil
        },
    )

    // Create the LLM
    model, err := llm.New("openai", llm.ProviderConfig{
        APIKey: "your-api-key",
        Model:  "gpt-4o",
    })
    if err != nil {
        log.Fatalf("llm: %v", err)
    }

    // Create an agent with the tool
    a := agent.New("assistant",
        agent.WithLLM(model),
        agent.WithTools([]tool.Tool{greet}),
    )

    result, err := a.Invoke(ctx, "Say hello to Alice")
    if err != nil {
        log.Fatalf("invoke: %v", err)
    }
    fmt.Println(result)
}
```

## Implementing the Tool Interface Directly

For tools that need custom schema construction or special execution logic, implement `tool.Tool` directly.

```go
package main

import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/schema"
    "github.com/lookatitude/beluga-ai/tool"
)

// WeatherTool implements tool.Tool manually.
type WeatherTool struct{}

func (t *WeatherTool) Name() string        { return "get_weather" }
func (t *WeatherTool) Description() string { return "Returns current weather for a city" }

func (t *WeatherTool) InputSchema() map[string]any {
    return map[string]any{
        "type": "object",
        "properties": map[string]any{
            "city": map[string]any{
                "type":        "string",
                "description": "City name",
            },
        },
        "required": []string{"city"},
    }
}

func (t *WeatherTool) Execute(ctx context.Context, input map[string]any) (*tool.Result, error) {
    city, ok := input["city"].(string)
    if !ok || city == "" {
        return nil, fmt.Errorf("get_weather: city is required")
    }

    // In production, call a weather API here
    weather := fmt.Sprintf("Sunny, 22C in %s", city)
    return tool.TextResult(weather), nil
}
```

## Using the Tool Registry

The `tool.Registry` provides thread-safe tool storage with name-based lookup and discovery.

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/agent"
    "github.com/lookatitude/beluga-ai/tool"
)

func main() {
    ctx := context.Background()

    // Create a registry and register tools
    reg := tool.NewRegistry()

    greet := tool.NewFuncTool("greet", "Greets a person",
        func(ctx context.Context, input GreetInput) (*tool.Result, error) {
            return tool.TextResult(fmt.Sprintf("Hello, %s!", input.Name)), nil
        },
    )

    if err := reg.Add(greet); err != nil {
        log.Fatalf("register greet: %v", err)
    }
    if err := reg.Add(&WeatherTool{}); err != nil {
        log.Fatalf("register weather: %v", err)
    }

    // Discover registered tools
    names := reg.List() // sorted: ["get_weather", "greet"]
    fmt.Println("Available tools:", names)

    // Retrieve tools by name
    t, err := reg.Get("greet")
    if err != nil {
        log.Fatalf("get tool: %v", err)
    }

    // Collect all tools for an agent
    var agentTools []tool.Tool
    for _, name := range reg.List() {
        t, err := reg.Get(name)
        if err != nil {
            log.Fatalf("get tool %s: %v", name, err)
        }
        agentTools = append(agentTools, t)
    }

    // Pass tools to an agent
    a := agent.New("assistant",
        agent.WithTools(agentTools),
        // ... agent.WithLLM(model),
    )
    _ = a
}
```

## Advanced Topics

### Tool Middleware

Wrap tools with middleware to add cross-cutting concerns like logging, metrics, or timeouts.

```go
import "github.com/lookatitude/beluga-ai/tool"

// Apply middleware to a tool
wrapped := tool.ApplyMiddleware(myTool,
    tool.WithTimeout(5 * time.Second),
    tool.WithLogging(logger),
)
```

### Tool Hooks

Attach lifecycle hooks to observe tool execution events.

```go
hooks := tool.Hooks{
    OnStart: func(ctx context.Context, name string, input map[string]any) error {
        log.Printf("tool %s called with %v", name, input)
        return nil
    },
    OnEnd: func(ctx context.Context, name string, result *tool.Result, err error) {
        if err != nil {
            log.Printf("tool %s failed: %v", name, err)
        }
    },
}
```

### Validating Tool Input

Always validate input in `Execute` and return clear error messages. The LLM uses error responses to self-correct.

```go
func (t *MyTool) Execute(ctx context.Context, input map[string]any) (*tool.Result, error) {
    query, ok := input["query"].(string)
    if !ok || query == "" {
        return tool.ErrorResult("query parameter is required and must be a string"), nil
    }
    // proceed with validated input
}
```

### Context Timeouts for External Calls

When a tool calls external services, use context-based timeouts to prevent agent stalls.

```go
func (t *APITool) Execute(ctx context.Context, input map[string]any) (*tool.Result, error) {
    ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()

    resp, err := t.client.Call(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("api call: %w", err)
    }
    return tool.TextResult(resp.Body), nil
}
```

## Configuration Reference

| Component | API | Description |
|-----------|-----|-------------|
| `tool.NewFuncTool` | `NewFuncTool[I](name, desc, fn)` | Create a typed tool from a Go function |
| `tool.NewRegistry` | `NewRegistry()` | Create an empty tool registry |
| `Registry.Add` | `Add(t Tool) error` | Register a tool (fails on duplicates) |
| `Registry.Get` | `Get(name string) (Tool, error)` | Retrieve a tool by name |
| `Registry.List` | `List() []string` | List all registered tool names (sorted) |

## Troubleshooting

### Tool not found

**Problem**: Agent reports a tool is not available.

**Solution**: Verify the tool is registered in the registry and passed to the agent via `agent.WithTools()`. Use `reg.List()` to confirm registration.

### Input schema mismatch

**Problem**: The LLM produces input that does not match the tool's schema.

**Solution**: Ensure `InputSchema()` accurately describes the expected input. When using `FuncTool`, check that struct tags include `json`, `description`, and `required` annotations. Validate and normalize in `Execute` as a fallback.

### Agent ignores available tools

**Problem**: The LLM generates text responses instead of calling tools.

**Solution**: Use an LLM that supports tool calling (e.g., GPT-4o, Claude). Verify tools are bound to the model via `BindTools`. Adjust the agent's persona or system prompt to encourage tool use.

## Related Resources

- [MCP Tools Integration](/integrations/agents-mcp-integration) — Expose tools over the Model Context Protocol
- [Tool System API Reference](/api-reference/tool) — Complete tool package documentation
- [Working with Tools](/getting-started/quick-start) — Getting started guide
