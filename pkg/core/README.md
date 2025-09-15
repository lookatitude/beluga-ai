# Core Package

The `core` package serves as the foundational "glue" layer of the Beluga AI Framework, providing essential abstractions, dependency injection, observability, and error handling that orchestrates components throughout the system.

## Overview

This package implements the framework's core principles:
- **Interface Segregation Principle (ISP)**: Small, focused interfaces
- **Dependency Inversion Principle (DIP)**: Depend on abstractions, use constructor injection
- **Single Responsibility Principle (SRP)**: One responsibility per component
- **Composition over Inheritance**: Embed interfaces for extensibility

## Key Components

### Runnable Interface

The central abstraction representing executable components that can be invoked, batched, or streamed:

```go
type Runnable interface {
    Invoke(ctx context.Context, input any, options ...Option) (any, error)
    Batch(ctx context.Context, inputs []any, options ...Option) ([]any, error)
    Stream(ctx context.Context, input any, options ...Option) (<-chan any, error)
}
```

All AI components (LLMs, retrievers, chains, etc.) implement this interface for unified orchestration.

### Dependency Injection Container

Provides type-safe dependency resolution with monitoring integration:

```go
container := core.NewContainerWithOptions(
    core.WithLogger(logger),
    core.WithTracerProvider(tracerProvider),
)

// Register dependencies
container.Register(func() MyService {
    return &myServiceImpl{}
})

// Resolve dependencies
var service MyService
container.Resolve(&service)
```

### Observability

Built-in OpenTelemetry tracing and metrics for all core operations:

```go
// Automatic tracing for Runnable operations
tracedRunnable := core.NewTracedRunnable(
    runnable,
    tracer,
    metrics,
    "component_type",
    "component_name",
)

// Health checks for components
if err := container.CheckHealth(ctx); err != nil {
    // Handle unhealthy component
}
```

### Standardized Error Handling

Framework-wide error types with codes and context:

```go
// Create typed errors
err := core.NewValidationError("invalid input", cause)

// Check error types
if core.IsErrorType(err, core.ErrorTypeValidation) {
    // Handle validation error
}
```

## Usage Examples

### Basic Component Creation

```go
// Create a container with monitoring
container := core.NewContainerWithOptions(
    core.WithLogger(logger),
    core.WithTracerProvider(tracerProvider),
)

// Register services
container.Register(func() LLMService {
    return &openaiService{}
})

// Build components
builder := core.NewBuilder(container)
builder.RegisterMetrics(func() (*core.Metrics, error) {
    return core.NewMetrics(meter), nil
})
```

### Component Orchestration

```go
// Components implement Runnable for unified execution
chain := core.NewChain([]core.Runnable{
    promptTemplate,
    llm,
    outputParser,
})

result, err := chain.Invoke(ctx, input)
```

### Health Monitoring

```go
// Check component health
healthChecker := container.(core.HealthChecker)
if err := healthChecker.CheckHealth(ctx); err != nil {
    log.Printf("Component unhealthy: %v", err)
}
```

## Configuration

The core package supports functional options for flexible configuration:

```go
container := core.NewContainerWithOptions(
    core.WithLogger(customLogger),
    core.WithTracerProvider(customTracer),
)
```

## Extensibility

The package is designed for extension:
- Embed interfaces for backward compatibility
- Use functional options for configuration
- Small interfaces enable easy mocking and testing
- Factory functions allow custom implementations

## Testing

Comprehensive test coverage with table-driven tests and mocks:

```go
// Mock implementations available
mockRunnable := &core.MockRunnable{}

// Test utilities
container := core.NewContainer()
assert.NoError(t, container.CheckHealth(ctx))
```

## Migration Guide

When upgrading from previous versions:
- Core error types moved from `utils/errors.go` to `core/errors.go`
- DI container now includes monitoring by default
- All components should implement `Runnable` interface where applicable

## Package Structure

```
pkg/core/
├── di.go              # Dependency injection container
├── errors.go          # Standardized error types
├── interfaces.go      # Core interface definitions
├── metrics.go         # Metrics collection
├── runnable.go        # Runnable interface and utilities
├── traced_runnable.go # Tracing instrumentation
├── model/             # Core data models
├── utils/             # Utility functions
└── *_test.go          # Comprehensive tests
```

This package provides the foundation that other Beluga AI packages build upon, ensuring consistency, observability, and maintainability across the entire framework.
