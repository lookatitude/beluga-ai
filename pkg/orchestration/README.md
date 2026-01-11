# Orchestration Package

The `orchestration` package provides comprehensive orchestration capabilities for Beluga AI. It implements chains, graphs, and workflows with full observability, configuration management, and extensible provider architecture.

## Current Capabilities

**✅ Production Ready:**
- Chain orchestration with sequential step execution
- Graph orchestration with dependency management
- Workflow orchestration with Temporal integration
- Comprehensive error handling with custom error types
- Configuration management with functional options and validation
- OpenTelemetry metrics, tracing, and structured logging
- Provider-based architecture for extensibility
- Health checks and observability
- Unit tests for core functionality

**🚧 Available as Framework:**
- Advanced graph algorithms (DAG execution)
- Distributed workflow coordination
- Real-time orchestration monitoring
- Streaming orchestration support

## Features

- **Multiple Orchestration Patterns**: Support for chains, graphs, and distributed workflows
- **Provider Architecture**: Extensible implementations through providers
- **Full Observability**: OpenTelemetry tracing, metrics, and structured logging
- **Configuration Management**: Functional options with validation
- **Error Handling**: Comprehensive error types with proper error wrapping
- **Health Monitoring**: Built-in health checks and status reporting
- **Concurrent Execution**: Support for parallel and concurrent orchestration
- **Memory Integration**: Seamless integration with Beluga's memory system
- **Dependency Injection**: Clean architecture with interface-based design

## Architecture

The package follows SOLID principles and the Beluga AI Framework patterns:

```
pkg/orchestration/
├── iface/              # Interface definitions (ISP compliant)
│   ├── interfaces.go   # Core orchestration interfaces
│   └── errors.go       # Custom error types with proper wrapping
├── internal/           # Private implementations and legacy code
│   ├── messagebus/    # Inter-agent communication
│   ├── monitoring/    # Workflow monitoring
│   ├── scheduler/     # Task scheduling
│   ├── task_chaining/ # Task chaining utilities
│   ├── workflow/      # Workflow orchestration
│   └── temporal.go    # Temporal integration
├── providers/          # Provider implementations
│   ├── chain/         # Chain orchestration providers
│   │   └── simple.go  # Simple chain implementation
│   ├── graph/         # Graph orchestration providers
│   │   └── basic.go   # Basic graph implementation
│   └── workflow/      # Workflow orchestration providers
│       └── temporal/  # Temporal workflow provider
├── config.go           # Configuration management and validation
├── metrics.go          # OpenTelemetry observability framework
├── orchestrator.go     # Main package API and factory functions
└── README.md           # This documentation
```

### Key Design Principles

- **Interface Segregation**: Small, focused interfaces serving specific purposes
- **Dependency Inversion**: High-level modules don't depend on low-level modules
- **Single Responsibility**: Each package/component has one clear purpose
- **Composition over Inheritance**: Behaviors composed through embedding
- **Provider Pattern**: Extensible implementations through provider interfaces
- **Clean Architecture**: Clear separation between business logic and infrastructure

## Core Interfaces

### Orchestrator Interface
```go
type Orchestrator interface {
    // CreateChain creates a new chain orchestration
    CreateChain(steps []core.Runnable, opts ...ChainOption) (Chain, error)

    // CreateGraph creates a new graph orchestration
    CreateGraph(opts ...GraphOption) (Graph, error)

    // CreateWorkflow creates a new workflow orchestration
    CreateWorkflow(workflowFn any, opts ...WorkflowOption) (Workflow, error)

    // GetMetrics returns orchestration metrics
    GetMetrics() OrchestratorMetrics
}
```

### Chain Interface
```go
type Chain interface {
    core.Runnable // Chains are Runnable

    // GetInputKeys returns the expected input keys for the chain
    GetInputKeys() []string

    // GetOutputKeys returns the keys produced by the chain
    GetOutputKeys() []string

    // GetMemory returns the memory associated with the chain, if any
    GetMemory() memory.BaseMemory
}
```

### Graph Interface
```go
type Graph interface {
    core.Runnable // Graphs are Runnable

    // AddNode adds a Runnable component as a node in the graph
    AddNode(name string, runnable core.Runnable) error

    // AddEdge defines a dependency between two nodes
    AddEdge(sourceNode string, targetNode string) error

    // SetEntryPoint defines the starting node(s) of the graph
    SetEntryPoint(nodeNames []string) error

    // SetFinishPoint defines the final node(s) whose output is the graph's output
    SetFinishPoint(nodeNames []string) error
}
```

### Workflow Interface
```go
type Workflow interface {
    // Execute starts the workflow execution
    Execute(ctx context.Context, input any) (workflowID string, runID string, err error)

    // GetResult retrieves the final result of a completed workflow instance
    GetResult(ctx context.Context, workflowID string, runID string) (any, error)

    // Signal sends a signal to a running workflow instance
    Signal(ctx context.Context, workflowID string, runID string, signalName string, data any) error

    // Query queries the state of a running workflow instance
    Query(ctx context.Context, workflowID string, runID string, queryType string, args ...any) (any, error)

    // Cancel requests cancellation of a running workflow instance
    Cancel(ctx context.Context, workflowID string, runID string) error

    // Terminate forcefully stops a running workflow instance
    Terminate(ctx context.Context, workflowID string, runID string, reason string, details ...any) error
}
```

### Key Interfaces

- **`Orchestrator`**: Main orchestration interface combining all patterns
- **`Chain`**: Sequential execution of runnable components
- **`Graph`**: DAG-based execution with dependency management
- **`Workflow`**: Long-running, distributed orchestration
- **`OrchestratorMetrics`**: Observability metrics interface
- **`HealthChecker`**: Health monitoring and status reporting
- **`ChainOption/GraphOption/WorkflowOption`**: Functional options for configuration

## Quick Start

### Creating a Simple Chain

```go
package main

import (
    "context"
    "log"

    "github.com/lookatitude/beluga-ai/pkg/orchestration"
    "github.com/lookatitude/beluga-ai/pkg/core"
)

func main() {
    // Create steps (implement core.Runnable)
    step1 := &MyRunnableStep{Name: "step1"}
    step2 := &MyRunnableStep{Name: "step2"}
    steps := []core.Runnable{step1, step2}

    // Create chain with configuration options
    chain, err := orchestration.NewChain(steps,
        orchestration.WithChainTimeout(30), // 30 seconds
        orchestration.WithChainRetries(3),
        orchestration.WithChainMemory(memoryInstance),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Execute chain
    input := map[string]any{"input": "test data"}
    result, err := chain.Invoke(context.Background(), input)
    if err != nil {
        log.Printf("Chain execution failed: %v", err)
    } else {
        log.Printf("Chain result: %v", result)
    }
}

// Example Runnable implementation
type MyRunnableStep struct {
    Name string
}

func (s *MyRunnableStep) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
    log.Printf("Executing step: %s with input: %v", s.Name, input)
    // Your step logic here
    return map[string]any{"output": s.Name + "_result"}, nil
}

func (s *MyRunnableStep) Batch(ctx context.Context, inputs []any, opts ...core.Option) ([]any, error) {
    // Batch implementation
    return nil, nil
}

func (s *MyRunnableStep) Stream(ctx context.Context, input any, opts ...core.Option) (<-chan any, error) {
    // Streaming implementation
    return nil, nil
}
```

### Creating a Graph Orchestration

```go
// Create graph with dependency management
graph, err := orchestration.NewGraph(
    orchestration.WithGraphTimeout(60),
    orchestration.WithGraphMaxWorkers(5),
    orchestration.WithGraphParallelExecution(true),
)
if err != nil {
    log.Fatal(err)
}

// Add nodes to the graph
graph.AddNode("data_processor", &DataProcessor{})
graph.AddNode("validator", &DataValidator{})
graph.AddNode("enricher", &DataEnricher{})
graph.AddNode("storage", &DataStorage{})

// Define dependencies
graph.AddEdge("data_processor", "validator")
graph.AddEdge("validator", "enricher")
graph.AddEdge("enricher", "storage")

// Set entry and exit points
graph.SetEntryPoint([]string{"data_processor"})
graph.SetFinishPoint([]string{"storage"})

// Execute graph
input := map[string]any{"data": "input_data"}
result, err := graph.Invoke(context.Background(), input)
if err != nil {
    log.Printf("Graph execution failed: %v", err)
} else {
    log.Printf("Graph result: %v", result)
}
```

### Creating a Workflow Orchestration

```go
// Note: Requires Temporal client setup
temporalClient, err := client.Dial(client.Options{
    HostPort: "localhost:7233",
})
if err != nil {
    log.Fatal(err)
}

// Create workflow function
workflowFn := func(ctx workflow.Context, input MyWorkflowInput) (MyWorkflowOutput, error) {
    // Workflow implementation using Temporal SDK
    // This is a placeholder - actual implementation would use Temporal activities
    return MyWorkflowOutput{Result: "workflow_completed"}, nil
}

// Create workflow with configuration
workflow, err := orchestration.NewWorkflow(workflowFn,
    orchestration.WithWorkflowTimeout(300), // 5 minutes
    orchestration.WithWorkflowRetries(5),
    orchestration.WithWorkflowTaskQueue("my-queue"),
)
if err != nil {
    log.Printf("Workflow creation failed: %v", err)
    return
}

// Execute workflow
input := MyWorkflowInput{Data: "workflow_input"}
workflowID, runID, err := workflow.Execute(context.Background(), input)
if err != nil {
    log.Printf("Workflow execution failed: %v", err)
    return
}

// Get result (blocking)
result, err := workflow.GetResult(context.Background(), workflowID, runID)
if err != nil {
    log.Printf("Failed to get workflow result: %v", err)
} else {
    log.Printf("Workflow result: %v", result)
}
```

### Using the Main Orchestrator

```go
// Create orchestrator with full configuration
orch, err := orchestration.NewOrchestratorWithOptions(
    orchestration.WithChainTimeout(30*time.Second),
    orchestration.WithGraphMaxWorkers(10),
    orchestration.WithWorkflowTaskQueue("beluga-workflows"),
    orchestration.WithMetricsPrefix("beluga.orchestration"),
    orchestration.WithFeatures(orchestration.EnabledFeatures{
        Chains:    true,
        Graphs:    true,
        Workflows: false, // Disable if Temporal not available
    }),
)
if err != nil {
    log.Fatal(err)
}

// Create and execute different orchestration types
chain, _ := orch.CreateChain(steps)
graph, _ := orch.CreateGraph()

// Get metrics
metrics := orch.GetMetrics()
log.Printf("Active chains: %d, graphs: %d, workflows: %d",
    metrics.GetActiveChains(),
    metrics.GetActiveGraphs(),
    metrics.GetActiveWorkflows())
```

## Configuration

The package supports comprehensive configuration through functional options:

### Chain Configuration
```go
chain, err := orchestration.NewChain(steps,
    orchestration.WithChainName("data-processing-chain"),
    orchestration.WithChainDescription("Processes and validates data"),
    orchestration.WithChainTimeout(60), // seconds
    orchestration.WithChainRetries(5),
    orchestration.WithChainMemory(memoryInstance),
    orchestration.WithChainInputKeys([]string{"data", "config"}),
    orchestration.WithChainOutputKeys([]string{"result", "metadata"}),
)
```

### Graph Configuration
```go
graph, err := orchestration.NewGraph(
    orchestration.WithGraphName("complex-workflow"),
    orchestration.WithGraphDescription("Multi-step data processing"),
    orchestration.WithGraphTimeout(300), // seconds
    orchestration.WithGraphRetries(3),
    orchestration.WithGraphMaxWorkers(8),
    orchestration.WithGraphParallelExecution(true),
    orchestration.WithGraphQueueSize(100),
)
```

### Workflow Configuration
```go
workflow, err := orchestration.NewWorkflow(workflowFn,
    orchestration.WithWorkflowName("distributed-processor"),
    orchestration.WithWorkflowDescription("Distributed data processing"),
    orchestration.WithWorkflowTimeout(1800), // 30 minutes
    orchestration.WithWorkflowRetries(10),
    orchestration.WithWorkflowTaskQueue("processing-queue"),
    orchestration.WithWorkflowMetadata(map[string]any{
        "priority": "high",
        "owner":    "data-team",
    }),
)
```

### Global Configuration
```go
// Create orchestrator with comprehensive configuration
orch, err := orchestration.NewOrchestratorWithOptions(
    // Chain settings
    orchestration.WithChainTimeout(30*time.Second),
    orchestration.WithChainRetries(3),

    // Graph settings
    orchestration.WithGraphMaxWorkers(5),
    orchestration.WithGraphParallelExecution(true),

    // Workflow settings
    orchestration.WithWorkflowTaskQueue("beluga-workflows"),
    orchestration.WithWorkflowTimeout(600), // 10 minutes

    // Observability
    orchestration.WithMetricsPrefix("beluga.orchestration"),
    orchestration.WithHealthCheckInterval(30*time.Second),

    // Feature toggles
    orchestration.WithFeatures(orchestration.EnabledFeatures{
        Chains:    true,
        Graphs:    true,
        Workflows: true,
        Scheduler: true,
        MessageBus: true,
    }),
)
```

## Error Handling

The package provides comprehensive error handling with custom error types:

```go
chain, err := orchestration.NewChain(steps)
if err != nil {
    var orchErr *iface.OrchestratorError
    if errors.As(err, &orchErr) {
        switch orchErr.Code {
        case iface.ErrCodeInvalidConfig:
            log.Printf("Configuration error: %v", orchErr.Err)
        case iface.ErrCodeExecutionFailed:
            log.Printf("Execution failed: %v", orchErr.Err)
        case iface.ErrCodeTimeout:
            log.Printf("Operation timed out: %v", orchErr.Err)
        }
    }
}

// Error handling during execution
result, err := chain.Invoke(ctx, input)
if err != nil {
    if iface.IsRetryable(err) {
        // Implement retry logic
        result, err = chain.Invoke(ctx, input) // Retry
    } else {
        // Handle permanent errors
        log.Printf("Permanent error: %v", err)
    }
}
```

### Error Types
- `OrchestratorError`: General orchestration operation errors
- `ErrInvalidConfig`: Configuration validation errors
- `ErrExecutionFailed`: Runtime execution errors
- `ErrTimeout`: Operation timeout errors
- `ErrDependencyFailed`: Dependency resolution errors
- `ErrResourceExhausted`: Resource limitation errors
- `ErrInvalidState`: Invalid state transition errors
- `ErrNotFound`: Resource not found errors

## Observability

### Metrics Initialization

The package uses a standardized metrics initialization pattern with `InitMetrics()` and `GetMetrics()`:

```go
import (
    "go.opentelemetry.io/otel/metric"
    "github.com/lookatitude/beluga-ai/pkg/orchestration"
)

// Initialize metrics once at application startup
meter := otel.Meter("beluga.orchestration")
orchestration.InitMetrics(meter)

// Get the global metrics instance
metrics := orchestration.GetMetrics()
if metrics != nil {
    // Metrics are automatically collected for all orchestration operations
    fmt.Printf("Active chains: %d\n", metrics.GetActiveChains())
    fmt.Printf("Active graphs: %d\n", metrics.GetActiveGraphs())
    fmt.Printf("Total executions: %d\n", metrics.GetTotalExecutions())
    fmt.Printf("Error count: %d\n", metrics.GetErrorCount())
}
```

**Note**: `InitMetrics()` uses `sync.Once` to ensure thread-safe initialization. It should be called once at application startup.

### Metrics
The package includes comprehensive metrics using OpenTelemetry:

- **Chain Metrics**: Execution count, duration, errors, active chains
- **Graph Metrics**: Execution count, duration, errors, active graphs, node count
- **Workflow Metrics**: Execution count, duration, errors, active workflows
- **General Metrics**: Total executions, error rates, resource usage

```go
// Metrics are automatically collected for all orchestration operations
metrics := orchestrator.GetMetrics()

// Access specific metrics
fmt.Printf("Active chains: %d\n", metrics.GetActiveChains())
fmt.Printf("Active graphs: %d\n", metrics.GetActiveGraphs())
fmt.Printf("Total executions: %d\n", metrics.GetTotalExecutions())
fmt.Printf("Error count: %d\n", metrics.GetErrorCount())
```

### Tracing
Distributed tracing support for end-to-end observability:

```go
// Traces are automatically created for all orchestration operations
// Spans include operation type, duration, and error information
ctx, span := tracer.Start(ctx, "custom.orchestration.operation")
defer span.End()

// Execute orchestration within the span context
result, err := chain.Invoke(ctx, input)
```

### Structured Logging
Comprehensive logging through structured events:

```go
// Create orchestrator with custom logger
orch, err := orchestration.NewOrchestratorWithOptions(
    orchestration.WithLogger(customLogger),
)

// Logs include trace IDs, operation types, and structured data
// Example log: {"level":"info","operation":"chain.execute","chain.name":"data-processor","duration":0.234,"trace.id":"abc123"}
```

## Health Monitoring

```go
// Health check for orchestration components
orch, err := orchestration.NewDefaultOrchestrator()
if err != nil {
    log.Fatal(err)
}

// Check overall health
healthErr := orch.Check(context.Background())
if healthErr != nil {
    log.Printf("Orchestrator unhealthy: %v", healthErr)
}

// Individual component health checks
chainHealth := chain.Check(context.Background())
graphHealth := graph.Check(context.Background())
workflowHealth := workflow.Check(context.Background())
```

## Provider Architecture

The package uses a provider-based architecture for extensibility:

### Chain Providers
```go
// Implement custom chain provider
type CustomChainProvider struct{}

func (p *CustomChainProvider) CreateChain(config iface.ChainConfig) (iface.Chain, error) {
    // Custom chain implementation
    return &CustomChain{
        config: config,
        // Custom logic
    }, nil
}

// Register provider
orchestration.RegisterChainProvider("custom", &CustomChainProvider{})
```

### Graph Providers
```go
// Implement custom graph provider
type CustomGraphProvider struct{}

func (p *CustomGraphProvider) CreateGraph(config iface.GraphConfig) (iface.Graph, error) {
    // Custom graph implementation with advanced algorithms
    return &CustomGraph{
        config: config,
        // Advanced DAG execution logic
    }, nil
}
```

### Workflow Providers
```go
// Implement custom workflow provider
type CustomWorkflowProvider struct{}

func (p *CustomWorkflowProvider) CreateWorkflow(workflowFn any, config iface.WorkflowConfig) (iface.Workflow, error) {
    // Custom workflow implementation
    return &CustomWorkflow{
        workflowFn: workflowFn,
        config:     config,
        // Custom workflow engine
    }, nil
}
```

## Memory Integration

Seamless integration with Beluga's memory system:

```go
// Create chain with memory
memory := memory.NewBufferMemory()
chain, err := orchestration.NewChain(steps,
    orchestration.WithChainMemory(memory),
)

// Memory is automatically used for context loading/saving
result, err := chain.Invoke(ctx, input)
// Memory variables are loaded from previous executions
// Context is automatically saved after execution
```

## Testing

The package includes comprehensive unit tests:

```go
func TestNewOrchestrator(t *testing.T) {
    orch, err := orchestration.NewDefaultOrchestrator()
    if err != nil {
        t.Fatalf("Failed to create orchestrator: %v", err)
    }

    if orch == nil {
        t.Fatal("Orchestrator should not be nil")
    }
}

func TestChainExecution(t *testing.T) {
    steps := []core.Runnable{
        &MockRunnable{name: "step1"},
        &MockRunnable{name: "step2"},
    }

    chain, err := orchestration.NewChain(steps)
    if err != nil {
        t.Fatalf("Failed to create chain: %v", err)
    }

    input := map[string]any{"test": "data"}
    result, err := chain.Invoke(context.Background(), input)

    if err != nil {
        t.Fatalf("Chain execution failed: %v", err)
    }

    if result == nil {
        t.Fatal("Chain result should not be nil")
    }
}

func TestGraphOrchestration(t *testing.T) {
    graph, err := orchestration.NewGraph()
    if err != nil {
        t.Fatalf("Failed to create graph: %v", err)
    }

    // Add nodes and edges
    graph.AddNode("processor", &MockRunnable{name: "processor"})
    graph.AddNode("validator", &MockRunnable{name: "validator"})
    graph.AddEdge("processor", "validator")

    graph.SetEntryPoint([]string{"processor"})
    graph.SetFinishPoint([]string{"validator"})

    input := map[string]any{"data": "test"}
    result, err := graph.Invoke(context.Background(), input)

    if err != nil {
        t.Fatalf("Graph execution failed: %v", err)
    }

    if result == nil {
        t.Fatal("Graph result should not be nil")
    }
}
```

## Best Practices

### 1. Error Handling
```go
// Always check for specific error types
result, err := chain.Invoke(ctx, input)
if err != nil {
    var orchErr *iface.OrchestratorError
    if errors.As(err, &orchErr) {
        switch orchErr.Code {
        case iface.ErrCodeTimeout:
            // Handle timeout with retry
        case iface.ErrCodeExecutionFailed:
            if iface.IsRetryable(err) {
                // Implement retry logic
            }
        }
    }
}
```

### 2. Configuration Validation
```go
// Validate configuration before use
config, err := orchestration.NewConfig(
    orchestration.WithChainTimeout(30*time.Second),
    orchestration.WithGraphMaxWorkers(5),
)
if err != nil {
    log.Fatal("Invalid config:", err)
}

orch, err := orchestration.NewOrchestrator(config)
if err != nil {
    log.Fatal("Failed to create orchestrator:", err)
}
```

### 3. Resource Management
```go
// Always properly cleanup resources
defer func() {
    if orch != nil {
        // Cleanup logic if needed
    }
}()
```

### 4. Observability
```go
// Use structured logging and metrics
orch, err := orchestration.NewOrchestratorWithOptions(
    orchestration.WithMetricsPrefix("myapp.orchestration"),
    orchestration.WithLogger(logger),
)

// Monitor orchestration health
go func() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        if err := orch.Check(context.Background()); err != nil {
            log.Printf("Orchestrator health check failed: %v", err)
        }
    }
}()
```

### 5. Timeout Management
```go
// Set appropriate timeouts for operations
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

result, err := chain.Invoke(ctx, input)
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        log.Printf("Operation timed out")
    }
}
```

## Implementation Status

### ✅ Completed Features
- **Architecture**: SOLID principles with clean separation of concerns
- **Chain Orchestration**: Sequential execution with memory integration
- **Graph Orchestration**: DAG execution with dependency management
- **Workflow Orchestration**: Temporal-based distributed workflows
- **Configuration Management**: Functional options with validation
- **Error Handling**: Custom error types with proper error wrapping
- **Observability**: OpenTelemetry metrics, tracing, and logging
- **Provider Architecture**: Extensible provider system
- **Health Monitoring**: Built-in health checks
- **Testing**: Unit tests for core functionality

### 🚧 In Development / Placeholder
- **Advanced Graph Algorithms**: Complex DAG execution patterns
- **Streaming Orchestration**: Real-time orchestration support
- **Distributed Coordination**: Advanced workflow coordination
- **Custom Providers**: Additional provider implementations

### 📋 Roadmap
1. **Advanced Graph Execution**: Parallel execution, cyclic graphs, complex dependencies
2. **Streaming Support**: Real-time orchestration with streaming results
3. **Custom Workflow Engines**: Support for additional workflow engines beyond Temporal
4. **Performance Optimization**: Caching, connection pooling, resource optimization
5. **Advanced Monitoring**: Custom dashboards, alerting, performance analytics
6. **Kubernetes Integration**: Native Kubernetes orchestration support
7. **Event-Driven Orchestration**: Event-based triggers and reactions

## Contributing

When adding new orchestration types or providers:

1. **Create implementation** in appropriate `providers/` directory following existing patterns
2. **Follow SOLID principles** and interface segregation
3. **Add comprehensive tests** with mocks and edge cases
4. **Update documentation** with examples and usage patterns
5. **Maintain backward compatibility** with existing interfaces
6. **Add observability** with proper metrics and tracing

### Development Guidelines
- Use functional options for configuration
- Implement proper error handling with custom error types
- Add comprehensive observability (metrics, tracing, logging)
- Write tests that cover both success and failure scenarios
- Update this README when adding new features
- Follow the provider pattern for extensibility

## Performance Considerations

### Chain Orchestration
- **Sequential Execution**: Steps execute one after another
- **Memory Overhead**: Minimal for simple chains
- **Timeout Management**: Individual step timeouts supported
- **Best For**: Simple, linear workflows

### Graph Orchestration
- **Parallel Execution**: Independent nodes can run concurrently
- **Resource Management**: Configurable worker pools
- **Dependency Resolution**: Automatic dependency ordering
- **Best For**: Complex workflows with dependencies

### Workflow Orchestration
- **Distributed Execution**: Can span multiple services/machines
- **Durability**: State persisted across failures
- **Scalability**: Horizontal scaling support
- **Best For**: Long-running, distributed processes

## License

This package is part of the Beluga AI Framework and follows the same license terms.
