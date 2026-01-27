# Agent Package

The agent package provides a simplified API for creating agents with memory and tools. It reduces boilerplate by providing a fluent builder pattern for agent configuration.

> **Note**: This package is a work in progress. For production use, please use the agents package directly (`github.com/lookatitude/beluga-ai/pkg/agents`).

## Features

- **Fluent Builder Pattern**: Chain configuration methods for easy agent setup
- **Configuration Storage**: Store agent parameters before building
- **Future Integration**: Designed for seamless integration with the main agents package

## Usage

### Basic Builder Usage

```go
import "github.com/lookatitude/beluga-ai/pkg/convenience/agent"

// Create a new agent builder
builder := agent.NewBuilder().
    WithSystemPrompt("You are a helpful assistant.").
    WithName("my-assistant").
    WithMaxTurns(20).
    WithVerbose(true).
    WithAgentType("react")

// Access configuration
fmt.Println(builder.GetSystemPrompt())  // "You are a helpful assistant."
fmt.Println(builder.GetName())          // "my-assistant"
fmt.Println(builder.GetMaxTurns())      // 20
fmt.Println(builder.GetAgentType())     // "react"
```

## Configuration Options

### System Prompt
```go
builder.WithSystemPrompt("You are an expert in...")
```

### Agent Name
```go
builder.WithName("research-agent")
```

### Maximum Turns
```go
builder.WithMaxTurns(15)  // Default: 10
```

### Verbose Mode
```go
builder.WithVerbose(true)  // Enable verbose logging
```

### Agent Type
```go
builder.WithAgentType("tool_calling")  // Options: "react", "tool_calling"
```

## Default Values

- **Name**: "assistant"
- **Max Turns**: 10
- **Agent Type**: "react"

## Future API (Planned)

The intended future API will look like:

```go
agent, err := agent.NewBuilder().
    WithLLM(llm).
    WithBufferMemory(50).
    WithTool(calculator).
    WithSystemPrompt("You are helpful").
    Build(ctx)

result, err := agent.Run(ctx, "Calculate 2+2")
```

## Production Usage

For production use, use the main agents package directly:

```go
import "github.com/lookatitude/beluga-ai/pkg/agents"

agent, err := agents.NewBaseAgent("my-agent", llm, tools)
result, err := agent.Invoke(ctx, input)
```
