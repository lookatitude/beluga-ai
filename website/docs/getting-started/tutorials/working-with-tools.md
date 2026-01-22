---
title: Working with Tools
sidebar_position: 4
---

# Part 4: Working with Tools

In this tutorial, you'll learn how to work with tools in Beluga AI. Tools extend agent capabilities by allowing them to interact with external systems, perform calculations, execute commands, and more.

## Learning Objectives

- ✅ Understand the tool interface
- ✅ Use built-in tools (Calculator, Echo, Shell, GoFunction)
- ✅ Create custom tools
- ✅ Register and discover tools
- ✅ Understand tool execution patterns

## Prerequisites

- Completed [Part 3: Creating Your First Agent](./03-first-agent)
- Basic understanding of Go interfaces
- API key for an LLM provider

## What are Tools?

Tools are functions that agents can call to:
- Perform calculations
- Execute shell commands
- Make API calls
- Access databases
- Interact with external services

Tools extend agent capabilities beyond what the LLM can do alone.

## Step 1: Understanding the Tool Interface

All tools implement the `Tool` interface:

```go
type Tool interface {
    Name() string
    Description() string
    Definition() ToolDefinition
    Execute(ctx context.Context, input any) (any, error)
    Batch(ctx context.Context, inputs []any) ([]any, error)
}
```

## Step 2: Built-in Tools

### Calculator Tool

Performs mathematical calculations:

```go
import "github.com/lookatitude/beluga-ai/pkg/agents/tools"

calculator := tools.NewCalculatorTool()

// Execute
result, err := calculator.Execute(ctx, "15 * 23")
// result: "345"
```

### Echo Tool

Echoes input back (useful for testing):

```go
echoTool := tools.NewEchoTool()

result, err := echoTool.Execute(ctx, "Hello, world!")
// result: "Hello, world!"
```

### Shell Tool

Executes shell commands (use with caution):

```go
import "github.com/lookatitude/beluga-ai/pkg/agents/tools/shell"
import "time"

shellTool, err := shell.NewShellTool(30 * time.Second)
if err != nil {
    log.Fatal(err)
}

// Input must be JSON with "command" key
input := map[string]interface{}{
    "command": "ls -la",
}
result, err := shellTool.Execute(ctx, input)
```

### GoFunction Tool

Wraps any Go function as a tool:

```go
import "github.com/lookatitude/beluga-ai/pkg/agents/tools/gofunc"

// Define your function
func myFunction(ctx context.Context, args map[string]any) (string, error) {
    name := args["name"].(string)
    return fmt.Sprintf("Hello, %s!", name), nil
}

// Create tool
inputSchema := `{
    "type": "object",
    "properties": {
        "name": {"type": "string"}
    },
    "required": ["name"]
}`

myTool, err := gofunc.NewGoFunctionTool(
    "greet",
    "Greets a person by name",
    inputSchema,
    myFunction,
)
```

## Step 3: Using Tools with Agents

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/agents"
    "github.com/lookatitude/beluga-ai/pkg/agents/tools"
    "github.com/lookatitude/beluga-ai/pkg/llms"
)

func main() {
    ctx := context.Background()

    // Setup LLM
    config := llms.NewConfig(
        llms.WithProvider("openai"),
        llms.WithModelName("gpt-3.5-turbo"),
        llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    )
    factory := llms.NewFactory()
    llm, _ := factory.CreateProvider("openai", config)

    // Create tools
    agentTools := []tools.Tool{
        tools.NewCalculatorTool(),
        tools.NewEchoTool(),
    }

    // Create agent with tools
    agent, _ := agents.NewBaseAgent("assistant", llm, agentTools)

    // Initialize and execute
    agent.Initialize(map[string]interface{}{})
    
    input := map[string]interface{}{
        "input": "Calculate 42 * 17 and echo the result",
    }
    result, _ := agent.Invoke(ctx, input)
    fmt.Printf("Result: %v\n", result)
}
```

## Step 4: Creating Custom Tools

Create a custom tool by implementing the `Tool` interface:

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/lookatitude/beluga-ai/pkg/agents/tools"
)

// WeatherTool is a custom tool that provides weather information
type WeatherTool struct {
    tools.BaseTool
}

func NewWeatherTool() *WeatherTool {
    inputSchema := map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "city": map[string]interface{}{
                "type":        "string",
                "description": "The city name",
            },
        },
        "required": []string{"city"},
    }

    return &WeatherTool{
        BaseTool: tools.BaseTool{
            name:        "weather",
            description: "Gets weather information for a city",
            inputSchema: inputSchema,
        },
    }
}

func (w *WeatherTool) Execute(ctx context.Context, input any) (any, error) {
    // Parse input
    inputMap, ok := input.(map[string]interface{})
    if !ok {
        // Try JSON string
        var parsed map[string]interface{}
        if err := json.Unmarshal([]byte(input.(string)), &parsed); err != nil {
            return nil, fmt.Errorf("invalid input format: %w", err)
        }
        inputMap = parsed
    }

    city, ok := inputMap["city"].(string)
    if !ok {
        return nil, fmt.Errorf("city must be a string")
    }

    // Simulate weather API call
    weather := map[string]interface{}{
        "city":    city,
        "temp":    "72°F",
        "condition": "Sunny",
    }

    result, _ := json.Marshal(weather)
    return string(result), nil
}

func (w *WeatherTool) Batch(ctx context.Context, inputs []any) ([]any, error) {
    results := make([]any, len(inputs))
    for i, input := range inputs {
        result, err := w.Execute(ctx, input)
        if err != nil {
            return nil, err
        }
        results[i] = result
    }
    return results, nil
}
```

## Step 5: Tool Registry

Use a registry to manage and discover tools:

```go
import "github.com/lookatitude/beluga-ai/pkg/agents/tools"

// Create registry
registry := tools.NewInMemoryToolRegistry()

// Register tools
registry.RegisterTool(tools.NewCalculatorTool())
registry.RegisterTool(tools.NewEchoTool())
registry.RegisterTool(NewWeatherTool())

// List available tools
toolNames := registry.ListTools()
fmt.Printf("Available tools: %v\n", toolNames)

// Get tool descriptions for LLM
descriptions := registry.GetToolDescriptions()
fmt.Printf("Tool descriptions:\n%s\n", descriptions)

// Get specific tool
calculator, _ := registry.GetTool("calculator")
result, _ := calculator.Execute(ctx, "10 + 5")
```

## Step 6: Tool Input/Output Formats

### Input Format

Tools accept input as:
- `map[string]interface\{\}` - Structured data
- `string` - JSON string (will be parsed)

### Output Format

Tools return:
- `string` - Text result
- `map[string]interface\{\}` - Structured data
- Any serializable type

### Example: Structured Input/Output

```go
// Tool that expects structured input
type DatabaseTool struct {
    tools.BaseTool
}

func (d *DatabaseTool) Execute(ctx context.Context, input any) (any, error) {
    inputMap := input.(map[string]interface{})
    
    query := inputMap["query"].(string)
    table := inputMap["table"].(string)
    
    // Execute query...
    result := map[string]interface{}{
        "rows": []map[string]interface{}{
            {"id": 1, "name": "Alice"},
            {"id": 2, "name": "Bob"},
        },
    }
    
    return result, nil
}
```

## Step 7: Error Handling in Tools

```go
func (t *MyTool) Execute(ctx context.Context, input any) (any, error) {
    // Validate input
    if input == nil {
        return nil, fmt.Errorf("input cannot be nil")
    }

    // Check context cancellation
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }

    // Handle errors gracefully
    result, err := doSomething(input)
    if err != nil {
        return nil, fmt.Errorf("tool execution failed: %w", err)
    }

    return result, nil
}
```

## Step 8: Best Practices

### 1. Clear Descriptions

```go
description := "Calculates the result of a mathematical expression. " +
    "Supports basic operations: +, -, *, /, and parentheses."
```

### 2. Validate Input

```go
func (t *MyTool) Execute(ctx context.Context, input any) (any, error) {
    inputMap, ok := input.(map[string]interface{})
    if !ok {
        return nil, fmt.Errorf("expected map[string]interface{}, got %T", input)
    }
    
    // Validate required fields
    if value, exists := inputMap["required_field"]; !exists {
        return nil, fmt.Errorf("required_field is missing")
    }
    
    // Continue execution...
}
```

### 3. Handle Timeouts

```go
func (t *MyTool) Execute(ctx context.Context, input any) (any, error) {
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    // Execute with timeout...
}
```

### 4. Provide Useful Error Messages

```go
if err != nil {
    return nil, fmt.Errorf("weather API call failed for city %s: %w", city, err)
}
```

## Step 9: Complete Example

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/agents"
    "github.com/lookatitude/beluga-ai/pkg/agents/tools"
    "github.com/lookatitude/beluga-ai/pkg/llms"
)

func main() {
    ctx := context.Background()

    // Setup
    config := llms.NewConfig(
        llms.WithProvider("openai"),
        llms.WithModelName("gpt-3.5-turbo"),
        llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    )
    factory := llms.NewFactory()
    llm, _ := factory.CreateProvider("openai", config)

    // Create and register tools
    registry := tools.NewInMemoryToolRegistry()
    registry.RegisterTool(tools.NewCalculatorTool())
    registry.RegisterTool(tools.NewEchoTool())
    registry.RegisterTool(NewWeatherTool())

    // Get all tools
    agentTools := make([]tools.Tool, 0)
    for _, name := range registry.ListTools() {
        tool, _ := registry.GetTool(name)
        agentTools = append(agentTools, tool)
    }

    // Create agent
    agent, _ := agents.NewBaseAgent("assistant", llm, agentTools)
    agent.Initialize(map[string]interface{}{})

    // Execute
    input := map[string]interface{}{
        "input": "What's the weather in San Francisco? Then calculate 20 * 5.",
    }
    result, _ := agent.Invoke(ctx, input)
    fmt.Printf("Result: %v\n", result)
}
```

## Exercises

1. **Create a time tool**: Build a tool that returns the current time
2. **Create a file tool**: Build a tool that reads file contents
3. **Create an API tool**: Build a tool that makes HTTP requests
4. **Tool composition**: Combine multiple tools in a single agent
5. **Error handling**: Add comprehensive error handling to your tools

## Next Steps

Congratulations! You've learned how to work with tools. Next, learn how to:

- **[Part 5: Memory Management](./memory-management)** - Add conversation memory
- **[Part 6: Orchestration Basics](./orchestration-basics)** - Build complex workflows
- **[Concepts: Agents](../../concepts/agents)** - Deep dive into agent concepts

---

**Ready for the next step?** Continue to [Part 5: Memory Management](./memory-management)!

