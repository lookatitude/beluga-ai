---
title: Tools Registry and Custom Tools
description: Build custom tools, register them in a tool registry, and bind them to agents using Beluga AI's tool system.
---

Tools extend agent capabilities beyond text generation. A tool is a function that the LLM can invoke -- search APIs, calculators, database queries, code execution. Beluga AI's tool system uses `schema.ToolDefinition` for model-facing descriptions and a registry for runtime discovery and management. The registry pattern (`Register()` + `New()` + `List()`) is the same one used throughout the framework for LLM providers, embedding models, vector stores, and every other extensible component. This consistency means that once you understand the pattern for tools, you understand the extension mechanism for the entire framework.

## What You Will Build

Custom tools with JSON schema input validation, a tool registry for centralized management, and integration with the `ChatModel` tool binding system.

## Prerequisites

- Understanding of [Building a Research Agent](/tutorials/agents/research-agent)
- Familiarity with JSON Schema basics

## Step 1: Define a Custom Tool

A tool consists of two parts: a **definition** that the model sees (name, description, input schema in JSON Schema format), and an **execution function** that your code calls when the model requests the tool. The definition tells the model when and how to use the tool -- the description quality directly affects how reliably the model will choose the right tool. The execution function receives the raw JSON arguments string from the model and must handle its own deserialization and validation.

The JSON Schema in `InputSchema` serves double duty: it guides the model in generating valid arguments, and it can be used for server-side validation before executing the tool. The `required` field is particularly important because it prevents the model from omitting mandatory parameters.

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"
    "time"

    "github.com/lookatitude/beluga-ai/schema"
)

// Tool combines a definition with an execution function.
type Tool struct {
    Definition schema.ToolDefinition
    Execute    func(ctx context.Context, args string) (string, error)
}

// Create an echo tool
var echoTool = Tool{
    Definition: schema.ToolDefinition{
        Name:        "echo",
        Description: "Echoes the input back. Useful for testing.",
        InputSchema: map[string]any{
            "type": "object",
            "properties": map[string]any{
                "message": map[string]any{
                    "type":        "string",
                    "description": "The message to echo",
                },
            },
            "required": []any{"message"},
        },
    },
    Execute: func(ctx context.Context, args string) (string, error) {
        var input struct {
            Message string `json:"message"`
        }
        if err := json.Unmarshal([]byte(args), &input); err != nil {
            return "", fmt.Errorf("parse args: %w", err)
        }
        return fmt.Sprintf("Echo: %s", input.Message), nil
    },
}

// Create a timestamp tool
var timestampTool = Tool{
    Definition: schema.ToolDefinition{
        Name:        "current_time",
        Description: "Returns the current date and time in UTC.",
        InputSchema: map[string]any{
            "type":       "object",
            "properties": map[string]any{},
        },
    },
    Execute: func(ctx context.Context, args string) (string, error) {
        return time.Now().UTC().Format(time.RFC3339), nil
    },
}

// Create a word counter tool
var wordCountTool = Tool{
    Definition: schema.ToolDefinition{
        Name:        "word_count",
        Description: "Counts the number of words in a text.",
        InputSchema: map[string]any{
            "type": "object",
            "properties": map[string]any{
                "text": map[string]any{
                    "type":        "string",
                    "description": "The text to count words in",
                },
            },
            "required": []any{"text"},
        },
    },
    Execute: func(ctx context.Context, args string) (string, error) {
        var input struct {
            Text string `json:"text"`
        }
        if err := json.Unmarshal([]byte(args), &input); err != nil {
            return "", fmt.Errorf("parse args: %w", err)
        }
        count := len(strings.Fields(input.Text))
        return fmt.Sprintf("%d words", count), nil
    },
}
```

## Step 2: Build a Tool Registry

A tool registry centralizes tool management with registration, lookup, and listing -- the same `Register()` + `Get()` + `List()` pattern used across Beluga AI. Centralizing tools in a registry instead of passing them around as slices provides several advantages: you can register tools from different packages via `init()`, look up tools by name for dynamic dispatch, and extract all definitions at once for model binding. The `Definitions()` method returns the slice of `schema.ToolDefinition` values that `BindTools` expects, bridging the registry to the model interface.

```go
type ToolRegistry struct {
    tools map[string]Tool
}

func NewToolRegistry() *ToolRegistry {
    return &ToolRegistry{
        tools: make(map[string]Tool),
    }
}

func (r *ToolRegistry) Register(tool Tool) {
    r.tools[tool.Definition.Name] = tool
}

func (r *ToolRegistry) Get(name string) (Tool, error) {
    t, ok := r.tools[name]
    if !ok {
        return Tool{}, fmt.Errorf("tool not found: %s", name)
    }
    return t, nil
}

func (r *ToolRegistry) List() []string {
    names := make([]string, 0, len(r.tools))
    for name := range r.tools {
        names = append(names, name)
    }
    return names
}

func (r *ToolRegistry) Definitions() []schema.ToolDefinition {
    defs := make([]schema.ToolDefinition, 0, len(r.tools))
    for _, tool := range r.tools {
        defs = append(defs, tool.Definition)
    }
    return defs
}
```

## Step 3: Execute Tool Calls

When the model generates tool calls, each call includes the tool name and a JSON arguments string. The dispatch function looks up the tool by name in the registry, calls its execution function, and packages the result as a `schema.ToolMessage` with the matching tool call ID. The ID correlation is critical -- each tool message must reference the ID of the tool call it responds to, so the model can match results to requests when there are multiple concurrent tool calls.

Errors during tool execution are returned as text content rather than Go errors. This is intentional: the model can read the error message and decide how to proceed (retry with different arguments, try a different tool, or inform the user), whereas a Go error would terminate the agent loop.

```go
func executeToolCalls(ctx context.Context, registry *ToolRegistry, calls []schema.ToolCall) []schema.Message {
    var results []schema.Message

    for _, tc := range calls {
        tool, err := registry.Get(tc.Name)
        if err != nil {
            results = append(results, schema.NewToolMessage(tc.ID, fmt.Sprintf("Error: %v", err)))
            continue
        }

        result, err := tool.Execute(ctx, tc.Arguments)
        if err != nil {
            results = append(results, schema.NewToolMessage(tc.ID, fmt.Sprintf("Error: %v", err)))
            continue
        }

        results = append(results, schema.NewToolMessage(tc.ID, result))
    }

    return results
}
```

## Step 4: Integrate with ChatModel

This step ties everything together: bind all registered tools to the model, then run the standard agent loop of generate-check-execute. The `BindTools` call returns a new `ChatModel` that includes tool definitions in every request. The loop terminates when the model responds without tool calls, indicating it has enough information to produce a final answer.

```go
func runWithTools(ctx context.Context, model llm.ChatModel, registry *ToolRegistry, input string) (string, error) {
    // Bind all registered tools to the model
    toolModel := model.BindTools(registry.Definitions())

    messages := []schema.Message{
        schema.NewSystemMessage("You are a helpful assistant. Use the available tools when needed."),
        schema.NewHumanMessage(input),
    }

    for i := 0; i < 5; i++ {
        resp, err := toolModel.Generate(ctx, messages)
        if err != nil {
            return "", err
        }

        messages = append(messages, resp)

        if len(resp.ToolCalls) == 0 {
            return resp.Text(), nil
        }

        // Execute tool calls and append results
        results := executeToolCalls(ctx, registry, resp.ToolCalls)
        messages = append(messages, results...)
    }

    return "", fmt.Errorf("exceeded max tool iterations")
}

func main() {
    ctx := context.Background()

    // Set up registry
    registry := NewToolRegistry()
    registry.Register(echoTool)
    registry.Register(timestampTool)
    registry.Register(wordCountTool)

    fmt.Println("Available tools:", registry.List())

    // Create model and run
    model, err := llm.New("openai", config.ProviderConfig{
        "api_key": os.Getenv("OPENAI_API_KEY"),
        "model":   "gpt-4o",
    })
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    result, err := runWithTools(ctx, model, registry, "What time is it? Also, count the words in: The quick brown fox jumps.")
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    fmt.Println(result)
}
```

## Verification

1. Register at least two tools.
2. Send a query that requires tool use ("What time is it?").
3. Verify the model calls the correct tool and incorporates the result.

## Next Steps

- [Multi-Agent Orchestration](/tutorials/agents/multi-agent-orchestration) -- Coordinate multiple tool-equipped agents
- [Research Agent](/tutorials/agents/research-agent) -- Build a complex agent with multiple tools
