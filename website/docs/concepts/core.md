---
title: Core
sidebar_position: 1
---

# Core Concepts

This document explains the foundational concepts that underpin the Beluga AI Framework.

## Runnable Interface

The `Runnable` interface is the central abstraction in Beluga AI. It provides a unified way to execute different components.

### Interface Definition

```go
type Runnable interface {
    Invoke(ctx context.Context, input any, options ...Option) (any, error)
    Batch(ctx context.Context, inputs []any, options ...Option) ([]any, error)
    Stream(ctx context.Context, input any, options ...Option) (<-chan any, error)
}
```

### Key Methods

- **Invoke**: Synchronous execution with a single input/output
- **Batch**: Concurrent execution with multiple inputs
- **Stream**: Asynchronous streaming execution

### Components Implementing Runnable

- LLMs and ChatModels
- Prompts and Templates
- Tools
- Agents
- Chains and Graphs
- Retrievers

## Message Types and Schemas

Beluga AI uses a message-based communication system.

### Message Types

```go
const (
    RoleHuman     MessageType = "human"     // User input
    RoleAssistant MessageType = "ai"        // AI response
    RoleSystem    MessageType = "system"    // System instructions
    RoleTool      MessageType = "tool"      // Tool execution results
    RoleFunction  MessageType = "function"  // Function calls
)
```

### Message Interface

```go
type Message interface {
    GetType() MessageType
    GetContent() string
    ToolCalls() []ToolCall
    AdditionalArgs() map[string]interface{}
}
```

### Creating Messages

```go
// System message - Sets assistant behavior
systemMsg := schema.NewSystemMessage("You are a helpful assistant.")

// Human message - User input
humanMsg := schema.NewHumanMessage("Hello!")

// AI message - Assistant response
aiMsg := schema.NewAIMessage("Hi there!")

// Tool message - Tool execution result
toolMsg := schema.NewToolMessage("result", "tool_name")
```

## Context Propagation

Context (`context.Context`) is used throughout Beluga AI for:

- **Cancellation**: Request cancellation and timeouts
- **Tracing**: Distributed tracing with OpenTelemetry
- **Request-scoped values**: Passing metadata through the call chain

### Using Context

```go
// Create context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// Check for cancellation
select {
case <-ctx.Done():
    return ctx.Err()
default:
    // Continue execution
}

// Pass context through calls
result, err := component.Invoke(ctx, input)
```

## Dependency Injection

Beluga AI uses dependency injection for flexible component composition.

### Interface-Based Design

```go
type Agent struct {
    llm    ChatModel    // Interface, not concrete type
    memory Memory       // Interface, not concrete type
    tools  []Tool       // Interface, not concrete type
}
```

### Factory Pattern

```go
// Create factory
factory := llms.NewFactory()

// Create provider using interface
provider, err := factory.CreateProvider("openai", config)
```

### Constructor Injection

```go
func NewAgent(llm ChatModel, memory Memory, tools []Tool) *Agent {
    return &Agent{
        llm:    llm,
        memory: memory,
        tools:  tools,
    }
}
```

## Error Handling Patterns

Beluga AI uses structured error handling with custom error types.

### Error Structure

```go
type Error struct {
    Op    string // Operation name
    Err   error  // Wrapped error
    Code  string // Error code
}
```

### Error Wrapping

```go
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

### Error Checking

```go
if llms.IsLLMError(err) {
    code := llms.GetLLMErrorCode(err)
    if llms.IsRetryableError(err) {
        // Implement retry logic
    }
}
```

## Options Pattern

Functional options provide flexible configuration.

### Option Interface

```go
type Option interface {
    Apply(config *map[string]any)
}
```

### Using Options

```go
config := llms.NewConfig(
    llms.WithProvider("openai"),
    llms.WithModelName("gpt-4"),
    llms.WithTemperatureConfig(0.7),
    llms.WithMaxTokensConfig(1000),
)
```

## Configuration Management

Configuration is managed through structured configs with validation.

### Config Structure

```go
type Config struct {
    Provider    string
    ModelName   string
    APIKey      string
    Temperature float64
    MaxTokens   int
}
```

### Validation

```go
if err := config.Validate(); err != nil {
    return fmt.Errorf("invalid config: %w", err)
}
```

## Observability

OpenTelemetry integration provides comprehensive observability.

### Tracing

```go
ctx, span := tracer.StartSpan(ctx, "operation")
defer span.End()

span.SetAttributes(
    attribute.String("key", "value"),
)
```

### Metrics

```go
metrics.Counter(ctx, "requests_total", "Total requests", 1, 
    map[string]string{"status": "success"})
```

### Logging

```go
logger.Info(ctx, "Operation completed", map[string]interface{}{
    "duration_ms": 150,
    "status": "success",
})
```

## Best Practices

1. **Always use context**: Pass context through all function calls
2. **Implement Runnable**: Make components composable
3. **Use interfaces**: Depend on abstractions, not implementations
4. **Handle errors properly**: Wrap errors with context
5. **Validate configuration**: Check configs before use
6. **Add observability**: Instrument code with tracing and metrics

## Related Concepts

- [LLM Concepts](./llms) - LLM-specific patterns
- [Agent Concepts](./agents) - Agent architecture
- [Architecture Documentation](../../guides/architecture) - System design

---

**Next:** Learn about [LLM Concepts](./llms) or [Agent Concepts](./agents)

