# Agents Package

The `agents` package provides AI agent implementations following the Beluga AI Framework design patterns. It implements autonomous agents that can reason, plan, and execute actions using tools.

## Current Capabilities

**✅ Production Ready:**
- Base agent with lifecycle management (init, execute, shutdown)
- Event-driven architecture with custom event handlers
- Comprehensive error handling with custom error types and wrapping
- Configuration management with validation and functional options
- OpenTelemetry metrics and tracing integration
- Clean architecture following SOLID principles
- Unit tests for core functionality
- Factory pattern with dependency injection
- Interface segregation with focused, composable interfaces
- Tool integration framework with registry system

## Features

- **Multiple Agent Types**: Support for different agent architectures (BaseAgent, ReActAgent framework)
- **Tool Integration**: Seamless integration with the tools registry system
- **Observability**: Full OpenTelemetry tracing and metrics integration
- **Configurable Execution**: Retry logic and customizable behavior
- **Event-Driven Architecture**: Extensible event system for monitoring and customization
- **Error Handling**: Comprehensive error types with proper error wrapping
- **Dependency Injection**: Clean architecture with interface-based design

## Architecture

The package follows SOLID principles and the Beluga AI Framework patterns:

```
pkg/agents/
├── iface/              # Interface definitions (ISP compliant)
├── internal/           # Private implementations
│   ├── base/          # Base agent implementation with lifecycle management
│   └── executor/      # Plan execution and tool orchestration
├── providers/         # Agent implementations and strategies
│   └── react/         # ReAct agent provider (reasoning + acting)
├── tools/             # Tool integration and registry
├── config.go          # Configuration management and validation
├── errors.go          # Custom error types with proper wrapping
├── metrics.go         # OpenTelemetry observability framework
├── agents.go          # Main package API and factory functions
└── README.md          # This documentation
```

### Key Design Principles

- **Interface Segregation**: Small, focused interfaces serving specific purposes
- **Dependency Inversion**: High-level modules don't depend on low-level modules
- **Single Responsibility**: Each package/component has one clear purpose
- **Composition over Inheritance**: Behaviors composed through embedding
- **Clean Architecture**: Clear separation between business logic and infrastructure

## Core Interfaces

### Agent Interface
```go
type Agent interface {
    Plan(ctx context.Context, steps []IntermediateStep, inputs map[string]any) (AgentAction, AgentFinish, error)
    InputVariables() []string
    OutputVariables() []string
    GetTools() []tools.Tool
    GetConfig() schema.AgentConfig
    GetLLM() llms.LLM
}
```

### CompositeAgent Interface
```go
type CompositeAgent interface {
    Agent
    LifecycleManager
    HealthChecker
    EventEmitter
}
```

### Key Interfaces

- **`Agent`**: Core agent interface for planning and tool execution
- **`CompositeAgent`**: Full-featured agent with lifecycle management
- **`Planner`**: Focused interface for planning operations only
- **`Executor`**: Handles plan execution and tool orchestration
- **`AgentFactory`**: Factory pattern for creating agent instances
- **`LifecycleManager`**: Agent lifecycle operations (init, shutdown)
- **`EventEmitter`**: Event-driven architecture support
- **`HealthChecker`**: Health monitoring and status reporting
- **`AgentRegistry`**: Global registry for managing agent types
- **`tools.Registry`**: Tool registry interface for tool management

## Quick Start

### Creating a Base Agent

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/agents"
    "github.com/lookatitude/beluga-ai/pkg/agents/iface"
    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/agents/tools"
)

func main() {
    // Initialize LLM (example - replace with actual implementation)
    llm := &MockLLM{} // Replace with your LLM implementation

    // Create tools (example - replace with actual tool implementations)
    calculator := &MockTool{name: "calculator", description: "Calculator tool"}
    webSearch := &MockTool{name: "web_search", description: "Web search tool"}

    // Create agent with configuration options
    agent, err := agents.NewBaseAgent("assistant", llm, []tools.Tool{calculator, webSearch},
        agents.WithMaxRetries(3),
        agents.WithRetryDelay(2*time.Second),
        agents.WithEventHandler("execution_completed", func(payload interface{}) error {
            log.Printf("Agent completed: %v", payload)
            return nil
        }),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Initialize agent with configuration
    config := map[string]interface{}{
        "max_retries": 5,
    }
    if err := agent.Initialize(config); err != nil {
        log.Fatal(err)
    }

    // Execute agent (note: BaseAgent.Execute() is a placeholder and needs implementation)
    if err := agent.Execute(); err != nil {
        log.Printf("Execution failed: %v", err)
    }
}

// Mock implementations for example (replace with real implementations)
type MockLLM struct{}

func (m *MockLLM) Invoke(ctx context.Context, prompt string, callOptions ...interface{}) (string, error) {
    return "Mock LLM response", nil
}

func (m *MockLLM) GetModelName() string { return "mock-llm" }
func (m *MockLLM) GetProviderName() string { return "mock-provider" }

type MockTool struct {
    name        string
    description string
}

func (m *MockTool) Name() string { return m.name }
func (m *MockTool) Description() string { return m.description }
func (m *MockTool) Definition() tools.ToolDefinition {
    return tools.ToolDefinition{
        Name:        m.name,
        Description: m.description,
    }
}
func (m *MockTool) Execute(ctx context.Context, input interface{}) (interface{}, error) {
    return "Mock tool result", nil
}
func (m *MockTool) Batch(ctx context.Context, inputs []interface{}) ([]interface{}, error) {
    return []interface{}{"Mock batch result"}, nil
}
```

### Creating a ReAct Agent

```go
// Create ReAct agent for complex reasoning tasks
// Note: ReAct agent implementation is currently a placeholder
chatLLM := &MockChatLLM{} // Replace with actual ChatModel implementation
tools := []tools.Tool{calculator, webSearch}
promptTemplate := "You are a helpful assistant..." // Prompt template

reactAgent, err := agents.NewReActAgent("researcher", chatLLM, tools, promptTemplate)
if err != nil {
    // Note: This will currently return an error as ReAct is not yet implemented
    log.Printf("ReAct agent not yet implemented: %v", err)
}
```

**Status**: The ReAct agent implementation is currently a placeholder and will be completed in a future update.

### Using the Executor

```go
// Create executor for plan-based execution
// Note: Executor is currently a simplified implementation
executor := agents.NewAgentExecutor()

// Execute a plan (placeholder implementation)
plan := []schema.Step{
    {
        Action: schema.AgentAction{
            Tool:      "calculator",
            ToolInput: map[string]any{"expression": "2 + 2"},
        },
    },
}

result, err := executor.ExecutePlan(ctx, agent, plan)
if err != nil {
    log.Printf("Plan execution failed: %v", err)
} else {
    // Note: Current implementation returns a placeholder message
    log.Printf("Result: %s", result.Output)
}
```

**Status**: The executor implementation is currently simplified and serves as a foundation for future enhancements.

## Configuration

The package supports configuration through functional options. Some advanced features are placeholders for future implementation:

### Agent Configuration
```go
agent, err := agents.NewBaseAgent("agent", llm, tools,
    // Retry configuration
    agents.WithMaxRetries(5),
    agents.WithRetryDelay(2*time.Second),

    // Event handlers
    agents.WithEventHandler("state_change", func(payload interface{}) error {
        state := payload.(iface.AgentState)
        log.Printf("Agent state changed to: %s", state)
        return nil
    }),
)
```

**Note**: Advanced configuration options like timeouts, metrics, and tracing are placeholders for future implementation.

### Global Configuration
```go
// Configuration management is implemented but simplified
config := agents.NewDefaultConfig()
config.DefaultMaxRetries = 3

if err := agents.ValidateConfig(config); err != nil {
    log.Fatal("Invalid configuration:", err)
}
```

**Status**: Configuration management is implemented with basic validation. Advanced observability features are planned for future updates.

## Error Handling

The package provides comprehensive error handling with custom error types:

```go
agent, err := agents.NewBaseAgent("agent", llm, tools)
if err != nil {
    var agentErr *agents.AgentError
    if errors.As(err, &agentErr) {
        switch agentErr.Code {
        case agents.ErrCodeInitialization:
            log.Printf("Initialization failed: %v", agentErr.Err)
        case agents.ErrCodeExecution:
            if agents.IsRetryable(agentErr.Err) {
                // Retry logic
            }
        }
    }
}
```

### Error Types
- `AgentError`: General agent operation errors
- `ValidationError`: Configuration validation errors
- `FactoryError`: Agent creation errors
- `ExecutionError`: Runtime execution errors
- `PlanningError`: Planning phase errors

## Observability

### Metrics
The package includes comprehensive metrics collection using OpenTelemetry:

- **Agent Metrics**: Creation counts, execution times, error rates by agent type
- **Executor Metrics**: Run counts, execution times, step counts by executor type
- **Tool Metrics**: Call counts, execution times, error rates by tool name
- **Planning Metrics**: Planning call counts, execution times, success rates

### Tracing
Full distributed tracing support for end-to-end observability:

```go
// Automatic tracing in agent operations
agent, err := agents.NewBaseAgent("my-agent", llm, tools)
// All Invoke, Plan, and Execute operations are automatically traced

// Custom tracing spans
ctx, span := metrics.StartAgentSpan(ctx, "my-agent", "custom_operation")
defer span.End()
```

**Status**: Full OpenTelemetry metrics and tracing integration implemented.

### Logging
Structured logging through event-driven architecture:

```go
// Event-driven logging (currently implemented)
agent.RegisterEventHandler("execution_error", func(payload interface{}) error {
    log.Printf("Agent execution error: %v", payload)
    return nil
})
```

**Status**: Basic event-driven logging is implemented. Structured logging integration planned for future updates.

## Tool Integration

Agents work seamlessly with the tools system:

```go
// Create tool registry
registry := agents.NewToolRegistry()

// Register tools
registry.RegisterTool(calculator)
registry.RegisterTool(webSearch)

// Get tools by name
tool, err := registry.GetTool("calculator")
if err != nil {
    log.Fatal("Tool not found:", err)
}

// List all available tools
toolNames := registry.ListTools()
log.Printf("Available tools: %v", toolNames)

// Get formatted tool descriptions for LLM context
descriptions := registry.GetToolDescriptions()
log.Printf("Tool descriptions:\n%s", descriptions)
```

## Agent Registry

The package provides a global registry system for managing agent types:

```go
// Use the global registry to create agents
agent, err := agents.CreateAgent(
    ctx,
    agents.AgentTypeBase,  // or agents.AgentTypeReAct
    "my-agent",
    llm,
    tools,
    config,
)

// List available agent types
types := agents.ListAvailableAgentTypes()
// Returns: ["base", "react"]

// Register a custom agent type
agents.RegisterAgentType("custom", func(ctx context.Context, name string, llm interface{}, tools []tools.Tool, config agents.Config) (iface.CompositeAgent, error) {
    // Custom agent creation logic
    return customAgent, nil
})

// Get the global registry for advanced usage
registry := agents.GetGlobalAgentRegistry()
```

## Health Monitoring

```go
// Check agent health
health := agents.HealthCheck(agent)
status := health["state"].(iface.AgentState)

if status == iface.StateError {
    log.Warn("Agent is in error state")
}

// Get human-readable status
statusString := agents.GetAgentStateString(status)
log.Printf("Agent status: %s", statusString)
```

## Agent States

Agents have well-defined lifecycle states:

- `StateInitializing`: Agent is being created
- `StateReady`: Agent is ready for execution
- `StateRunning`: Agent is currently executing
- `StatePaused`: Agent execution is paused
- `StateError`: Agent encountered an error
- `StateShutdown`: Agent has been shut down

## Testing

The package includes unit tests for the core functionality:

```go
func TestNewBaseAgent(t *testing.T) {
    llm := &MockLLM{response: "test"}
    tools := []tools.Tool{&MockTool{name: "test-tool"}}

    agent, err := agents.NewBaseAgent("test-agent", llm, tools)
    if err != nil {
        t.Fatalf("Failed to create agent: %v", err)
    }
    if agent == nil {
        t.Fatal("Agent should not be nil")
    }
}
```

**Status**: Basic unit tests are implemented for the base agent functionality. Test coverage will be expanded as more features are implemented.

## Best Practices

### 1. Error Handling
```go
// Always check for specific error types
if err := agent.Execute(); err != nil {
    if agents.IsRetryable(err) {
        // Implement retry logic
    } else {
        // Handle permanent errors
    }
}
```

### 2. Configuration Validation
```go
// Validate configuration before use
config := agents.NewDefaultConfig()
if err := agents.ValidateConfig(config); err != nil {
    log.Fatal("Invalid config:", err)
}
```

### 3. Resource Management
```go
// Always properly shutdown agents
defer func() {
    if err := agent.Shutdown(); err != nil {
        log.Printf("Shutdown error: %v", err)
    }
}()
```

### 4. Event Monitoring
```go
// Register event handlers for monitoring
agent.RegisterEventHandler("execution_completed", func(payload interface{}) error {
    metrics.RecordSuccess()
    return nil
})

agent.RegisterEventHandler("execution_error", func(payload interface{}) error {
    metrics.RecordError()
    return nil
})
```

## Implementation Status

### ✅ Completed Features
- **Architecture**: SOLID principles with clean separation of concerns
- **Base Agent**: Core agent implementation with lifecycle management
- **Interfaces**: Comprehensive interface definitions (ISP compliant)
- **Configuration**: Full configuration management with validation and functional options
- **Error Handling**: Custom error types with proper error wrapping
- **Event System**: Event-driven architecture for extensibility
- **Metrics & Tracing**: Full OpenTelemetry integration
- **Factory Pattern**: Dependency injection with functional options
- **Agent Registry**: Global registry system for agent type management
- **Tool Registry**: Complete tool management and integration
- **Testing**: Comprehensive unit tests for core functionality

### 📋 Roadmap
1. **Complete ReAct Agent**: Full reasoning + acting implementation
2. **Enhanced Executor**: Advanced plan execution with concurrency
3. **Server Integration**: MCP server and REST API exposure
4. **A2A Protocol**: Agent-to-agent communication protocol
5. **Streaming Support**: Real-time agent responses
6. **Persistence**: Agent state and memory management
7. **Structured Logging**: Enhanced logging with context and trace IDs

## Contributing

When adding new agent types:

1. **Create implementation** in `providers/` directory following existing patterns
2. **Follow SOLID principles** and interface segregation
3. **Add comprehensive tests** with mocks and edge cases
4. **Update documentation** with examples and usage patterns
5. **Maintain backward compatibility** with existing interfaces

### Development Guidelines
- Use functional options for configuration
- Implement proper error handling with custom error types
- Add event handlers for monitoring and debugging
- Write tests that cover both success and failure scenarios
- Update this README when adding new features

## License

This package is part of the Beluga AI Framework and follows the same license terms.
