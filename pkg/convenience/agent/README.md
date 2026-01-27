# Agent Package

The agent package provides a simplified API for creating agents with memory and tools. It reduces boilerplate by providing a fluent builder pattern for agent configuration.

## Features

- **Fluent Builder Pattern**: Chain configuration methods for easy agent setup
- **Multiple LLM Types**: Support for both LLM and ChatModel interfaces
- **Memory Integration**: Built-in buffer and window memory support
- **Tool Support**: Add single or multiple tools to agents
- **OpenTelemetry Integration**: Full observability with metrics and tracing
- **Structured Errors**: Op/Err/Code error pattern for clear error handling

## Installation

```go
import "github.com/lookatitude/beluga-ai/pkg/convenience/agent"
```

## Quick Start

```go
import (
    "context"
    "github.com/lookatitude/beluga-ai/pkg/convenience/agent"
    "github.com/lookatitude/beluga-ai/pkg/llms/providers/openai"
)

// Create an LLM
llm, _ := openai.NewOpenAI(ctx, openai.WithAPIKey("your-key"))

// Build the agent
myAgent, err := agent.NewBuilder().
    WithLLM(llm).
    WithName("my-assistant").
    WithSystemPrompt("You are a helpful assistant.").
    WithBufferMemory(50).
    Build(ctx)
if err != nil {
    log.Fatal(err)
}

// Run the agent
result, err := myAgent.Run(ctx, "Hello, how are you?")
```

## Builder API

### Creating a Builder

```go
builder := agent.NewBuilder()
```

### Configuration Methods

#### LLM Configuration

```go
// Use an LLM instance
builder.WithLLM(llm)

// Use a ChatModel instance (alternative to LLM)
builder.WithChatModel(chatModel)

// Use provider-based resolution (not yet implemented)
builder.WithLLMProvider("openai", "api-key")
```

#### Agent Configuration

```go
builder.WithName("research-agent")           // Set agent name (default: "assistant")
builder.WithSystemPrompt("You are an expert...")  // Set system prompt
builder.WithMaxTurns(15)                     // Maximum conversation turns (default: 10)
builder.WithVerbose(true)                    // Enable verbose logging
builder.WithAgentType("tool_calling")        // Agent type: "react" or "tool_calling" (default: "react")
builder.WithTimeout(5 * time.Minute)         // Set execution timeout
```

#### Memory Configuration

```go
// Buffer memory - stores last N messages
builder.WithBufferMemory(100)

// Window memory - stores last N turns
builder.WithWindowMemory(20)

// Use a pre-configured memory instance
builder.WithMemory(customMemory)
```

#### Tool Configuration

```go
// Add a single tool
builder.WithTool(calculatorTool)

// Add multiple tools
builder.WithTools([]core.Tool{searchTool, browserTool})
```

#### Metrics Configuration

```go
// Use custom metrics instance
builder.WithMetrics(customMetrics)
```

### Building the Agent

```go
agent, err := builder.Build(ctx)
if err != nil {
    // Handle error
}
```

## Agent Interface

The built agent implements the `Agent` interface:

```go
type Agent interface {
    Run(ctx context.Context, input string) (string, error)
    RunWithInputs(ctx context.Context, inputs map[string]any) (map[string]any, error)
    Stream(ctx context.Context, input string) (<-chan string, error)
    GetName() string
    GetTools() []core.Tool
    GetMemory() memoryiface.Memory
    Shutdown() error
}
```

### Running the Agent

```go
// Simple string input/output
result, err := myAgent.Run(ctx, "What is 2+2?")

// Map-based input/output
outputs, err := myAgent.RunWithInputs(ctx, map[string]any{
    "input": "Calculate the sum of 5 and 3",
})

// Streaming (returns channel)
stream, err := myAgent.Stream(ctx, "Tell me a story")
for chunk := range stream {
    fmt.Print(chunk)
}
```

### Accessing Agent Components

```go
name := myAgent.GetName()
tools := myAgent.GetTools()
memory := myAgent.GetMemory()
```

### Cleanup

```go
err := myAgent.Shutdown()
```

## Error Handling

The package uses structured errors with Op/Err/Code pattern:

```go
agent, err := builder.Build(ctx)
if err != nil {
    var agentErr *agent.Error
    if errors.As(err, &agentErr) {
        switch agentErr.Code {
        case agent.ErrCodeMissingLLM:
            // No LLM or ChatModel configured
        case agent.ErrCodeLLMCreation:
            // Failed to create LLM from provider
        case agent.ErrCodeMemoryCreation:
            // Failed to create memory
        case agent.ErrCodeAgentCreation:
            // Failed to create underlying agent
        }
    }
}
```

### Error Codes

| Code | Description |
|------|-------------|
| `missing_llm` | No LLM or ChatModel configured |
| `llm_creation_failed` | Failed to create LLM from provider name |
| `memory_creation_failed` | Failed to create memory |
| `agent_creation_failed` | Failed to create underlying agent |
| `invalid_llm_type` | Invalid LLM type provided |
| `execution_failed` | Agent execution failed |

## Observability

The package includes OpenTelemetry metrics and tracing:

```go
// Get global metrics instance
metrics := agent.GetMetrics()

// Create custom metrics
metrics, err := agent.NewMetrics("custom-prefix")

// Use no-op metrics (for testing)
metrics := agent.NoOpMetrics()
```

### Metrics Recorded

- `agent_builds_total` - Counter for build operations
- `agent_build_duration_seconds` - Histogram for build duration
- `agent_executions_total` - Counter for agent executions
- `agent_execution_duration_seconds` - Histogram for execution duration
- `agent_errors_total` - Counter for errors by type

## Examples

### Agent with Tools

```go
// Create a calculator tool
calculator := tools.NewGoFunc("calculator", "Calculate math expressions", func(expr string) (string, error) {
    // calculation logic
    return result, nil
})

agent, err := agent.NewBuilder().
    WithLLM(llm).
    WithTool(calculator).
    WithSystemPrompt("You can use the calculator tool for math.").
    Build(ctx)
```

### Agent with Memory

```go
agent, err := agent.NewBuilder().
    WithChatModel(chatModel).
    WithBufferMemory(100).  // Remember last 100 messages
    WithSystemPrompt("You are a conversational assistant.").
    Build(ctx)

// First turn
agent.Run(ctx, "My name is Alice")

// Second turn - agent remembers the name
agent.Run(ctx, "What's my name?")  // "Your name is Alice"
```

### ReAct Agent

```go
agent, err := agent.NewBuilder().
    WithLLM(llm).
    WithAgentType("react").
    WithTools([]core.Tool{searchTool, browserTool}).
    WithMaxTurns(20).
    Build(ctx)
```

## Default Values

| Option | Default |
|--------|---------|
| Name | "assistant" |
| Max Turns | 10 |
| Agent Type | "react" |
| Verbose | false |
| Memory | nil (disabled) |

## Thread Safety

The built agent is safe for concurrent use. Multiple goroutines can call `Run()` simultaneously.

## See Also

- [pkg/agents](../../agents/) - Lower-level agent framework
- [pkg/memory](../../memory/) - Memory implementations
- [pkg/tools](../../tools/) - Tool implementations
- [pkg/llms](../../llms/) - LLM providers
