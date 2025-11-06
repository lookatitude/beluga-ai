# Core Package

The `core` package serves as the foundational "glue" layer of the Beluga AI Framework, providing essential abstractions, dependency injection, observability, and error handling that orchestrates components throughout the system.

## Table of Contents

1. [Overview](#overview)
2. [Current Status](#current-status)
3. [Key Components](#key-components)
4. [Core Interfaces](#core-interfaces)
5. [Dependency Injection](#dependency-injection)
6. [Error Handling](#error-handling)
7. [Observability](#observability)
8. [Usage Examples](#usage-examples)
9. [API Reference](#api-reference)
10. [Testing](#testing)
11. [Best Practices](#best-practices)

## Overview

This package implements the framework's core principles:
- **Interface Segregation Principle (ISP)**: Small, focused interfaces
- **Dependency Inversion Principle (DIP)**: Depend on abstractions, use constructor injection
- **Single Responsibility Principle (SRP)**: One responsibility per component
- **Composition over Inheritance**: Embed interfaces for extensibility

## Current Status

**✅ Production Ready:**
- Runnable interface with Invoke, Batch, and Stream operations
- Dependency injection container with factory pattern and singleton support
- Comprehensive error handling with typed errors and error codes
- OpenTelemetry metrics and tracing integration
- Health check interface and implementation
- Builder pattern for fluent dependency registration
- Utility functions for common operations

## Key Components

### Runnable Interface

The central abstraction representing executable components that can be invoked, batched, or streamed. All AI components (LLMs, retrievers, chains, agents, etc.) implement this interface for unified orchestration.

### Dependency Injection Container

Type-safe dependency resolution with automatic dependency injection, singleton support, and monitoring integration.

### Error Handling

Framework-wide standardized error types with error codes, context, and proper error wrapping for programmatic error handling.

### Observability

Built-in OpenTelemetry tracing and metrics for all core operations, including Runnable executions, DI operations, and health checks.

## Core Interfaces

### Runnable Interface

```go
type Runnable interface {
    // Invoke executes the runnable component with a single input and returns a single output.
    // It is the primary method for synchronous execution.
    Invoke(ctx context.Context, input any, options ...Option) (any, error)

    // Batch executes the runnable component with multiple inputs concurrently or sequentially,
    // returning a corresponding list of outputs.
    Batch(ctx context.Context, inputs []any, options ...Option) ([]any, error)

    // Stream executes the runnable component with a single input and returns a channel
    // from which output chunks can be read asynchronously.
    Stream(ctx context.Context, input any, options ...Option) (<-chan any, error)
}
```

### Option Interface

```go
type Option interface {
    // Apply modifies a configuration map. Implementations should add or update
    // key-value pairs relevant to their specific option.
    Apply(config *map[string]any)
}
```

### Container Interface

```go
type Container interface {
    // Register registers a factory function for a type
    Register(factoryFunc interface{}) error

    // Resolve resolves a dependency by type
    Resolve(target interface{}) error

    // MustResolve resolves a dependency or panics
    MustResolve(target interface{})

    // Has checks if a type is registered
    Has(typ reflect.Type) bool

    // Clear removes all registered dependencies
    Clear()

    // Singleton registers a singleton instance
    Singleton(instance interface{})

    // HealthChecker provides health check functionality
    HealthChecker
}
```

### HealthChecker Interface

```go
type HealthChecker interface {
    // CheckHealth performs a health check and returns an error if the component is unhealthy.
    CheckHealth(ctx context.Context) error
}
```

### Loader Interface

```go
type Loader interface {
    // Load reads all data from the source and returns it as a slice of Documents.
    Load(ctx context.Context) ([]schema.Document, error)
    
    // LazyLoad provides an alternative way to load data, returning a channel that yields
    // documents one by one as they become available.
    LazyLoad(ctx context.Context) (<-chan any, error)
}
```

### Retriever Interface

```go
type Retriever interface {
    Runnable // Input: string (query), Output: []schema.Document

    // GetRelevantDocuments retrieves documents considered relevant to the given query string.
    GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error)
}
```

## Dependency Injection

### Creating a Container

```go
// Basic container with no-op monitoring
container := core.NewContainer()

// Container with custom monitoring
container := core.NewContainerWithOptions(
    core.WithLogger(logger),
    core.WithTracerProvider(tracerProvider),
)
```

### Registering Dependencies

```go
// Register a factory function
container.Register(func() MyService {
    return &myServiceImpl{}
})

// Register a singleton instance
container.Singleton(&mySingletonService{})

// Register with dependencies
container.Register(func(dep Dependency) MyService {
    return &myServiceImpl{dep: dep}
})
```

### Resolving Dependencies

```go
// Resolve a dependency
var service MyService
if err := container.Resolve(&service); err != nil {
    log.Fatal(err)
}

// Resolve or panic
var service MyService
container.MustResolve(&service)
```

### Builder Pattern

```go
builder := core.NewBuilder(container)

// Register services
builder.Register(func() MyService {
    return &myServiceImpl{}
})

// Register monitoring components
builder.RegisterLogger(func() core.Logger {
    return myLogger
})

builder.RegisterTracerProvider(func() core.TracerProvider {
    return myTracerProvider
})

builder.RegisterMetrics(func() (*core.Metrics, error) {
    return core.NewMetrics(meter), nil
})

// Build components
var service MyService
if err := builder.Build(&service); err != nil {
    log.Fatal(err)
}
```

### Container Options

```go
// WithContainer sets a custom container
core.WithContainer(customContainer)

// WithLogger sets the logger
core.WithLogger(logger)

// WithTracerProvider sets the tracer provider
core.WithTracerProvider(tracerProvider)
```

## Error Handling

### Error Types

The package provides standardized error types:

```go
type ErrorType string

const (
    ErrorTypeValidation     ErrorType = "validation"
    ErrorTypeNetwork        ErrorType = "network"
    ErrorTypeAuthentication ErrorType = "authentication"
    ErrorTypeRateLimit      ErrorType = "rate_limit"
    ErrorTypeInternal       ErrorType = "internal"
    ErrorTypeExternal       ErrorType = "external"
    ErrorTypeConfiguration  ErrorType = "configuration"
)
```

### Error Codes

```go
type ErrorCode string

const (
    ErrorCodeInvalidInput   ErrorCode = "INVALID_INPUT"
    ErrorCodeNotFound       ErrorCode = "NOT_FOUND"
    ErrorCodeUnauthorized   ErrorCode = "UNAUTHORIZED"
    ErrorCodeTimeout        ErrorCode = "TIMEOUT"
    ErrorCodeRateLimited    ErrorCode = "RATE_LIMITED"
    ErrorCodeInternalError  ErrorCode = "INTERNAL_ERROR"
)
```

### FrameworkError

```go
type FrameworkError struct {
    Type    ErrorType
    Message string
    Cause   error
    Code    string                 // Optional error code
    Context map[string]interface{} // Additional context
}
```

### Creating Errors

```go
// Create typed errors
err := core.NewValidationError("invalid input", cause)
err := core.NewNetworkError("connection failed", cause)
err := core.NewAuthenticationError("unauthorized", cause)
err := core.NewInternalError("internal error", cause)
err := core.NewConfigurationError("invalid config", cause)

// Create error with code
err := core.NewFrameworkErrorWithCode(
    core.ErrorTypeValidation,
    core.ErrorCodeInvalidInput,
    "invalid input parameter",
    cause,
)

// Add context to error
err.AddContext("field", "username")
err.AddContext("value", userInput)
```

### Checking Errors

```go
// Check error type
if core.IsErrorType(err, core.ErrorTypeValidation) {
    // Handle validation error
}

// Extract FrameworkError
var fwErr *core.FrameworkError
if core.AsFrameworkError(err, &fwErr) {
    switch fwErr.Code {
    case string(core.ErrorCodeInvalidInput):
        // Handle invalid input
    case string(core.ErrorCodeTimeout):
        // Handle timeout
    }
}

// Unwrap error
cause := core.UnwrapError(err)

// Wrap error
wrapped := core.WrapError(err, "additional context")
```

## Observability

### Metrics

The package provides comprehensive metrics for Runnable operations:

```go
// Create metrics instance
meter := otel.Meter("beluga-ai-core")
metrics, err := core.NewMetrics(meter)

// Metrics collected:
// - runnable_invokes_total: Total number of Invoke calls
// - runnable_batches_total: Total number of Batch calls
// - runnable_streams_total: Total number of Stream calls
// - runnable_errors_total: Total number of errors
// - runnable_duration_seconds: Duration histogram for all operations
// - runnable_batch_size: Batch size histogram
// - runnable_batch_duration_seconds: Batch operation duration
// - runnable_stream_duration_seconds: Stream operation duration
// - runnable_stream_chunks_total: Total number of stream chunks

// Record metrics manually
metrics.RecordRunnableInvoke(ctx, "component_type", duration, err)
metrics.RecordRunnableBatch(ctx, "component_type", batchSize, duration, err)
metrics.RecordRunnableStream(ctx, "component_type", duration, chunkCount, err)

// No-op metrics for testing
noOpMetrics := core.NoOpMetrics()
```

### Tracing

Automatic tracing is provided through TracedRunnable:

```go
// Create traced runnable
tracedRunnable := core.NewTracedRunnable(
    runnable,
    tracer,
    metrics,
    "component_type",
    "component_name",
)

// Helper functions
runnable := core.RunnableWithTracing(
    myRunnable,
    tracer,
    metrics,
    "llm",
)

runnable := core.RunnableWithTracingAndName(
    myRunnable,
    tracer,
    metrics,
    "llm",
    "gpt-4",
)
```

### Health Checks

```go
// Check container health
if err := container.CheckHealth(ctx); err != nil {
    log.Printf("Container unhealthy: %v", err)
}
```

## Usage Examples

### Basic Runnable Usage

```go
package main

import (
    "context"
    "fmt"
    "github.com/lookatitude/beluga-ai/pkg/core"
)

type MyRunnable struct{}

func (r *MyRunnable) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
    return fmt.Sprintf("Processed: %v", input), nil
}

func (r *MyRunnable) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
    results := make([]any, len(inputs))
    for i, input := range inputs {
        results[i] = fmt.Sprintf("Processed: %v", input)
    }
    return results, nil
}

func (r *MyRunnable) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
    ch := make(chan any)
    go func() {
        defer close(ch)
        ch <- fmt.Sprintf("Chunk 1: %v", input)
        ch <- fmt.Sprintf("Chunk 2: %v", input)
    }()
    return ch, nil
}

func main() {
    runnable := &MyRunnable{}
    
    // Invoke
    result, err := runnable.Invoke(context.Background(), "test")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(result)
    
    // Batch
    results, err := runnable.Batch(context.Background(), []any{"a", "b", "c"})
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(results)
    
    // Stream
    stream, err := runnable.Stream(context.Background(), "test")
    if err != nil {
        log.Fatal(err)
    }
    for chunk := range stream {
        fmt.Println(chunk)
    }
}
```

### Dependency Injection Example

```go
package main

import (
    "context"
    "github.com/lookatitude/beluga-ai/pkg/core"
)

type Database interface {
    Query(query string) ([]string, error)
}

type Service interface {
    Process(data string) (string, error)
}

type databaseImpl struct{}

func (d *databaseImpl) Query(query string) ([]string, error) {
    return []string{"result1", "result2"}, nil
}

type serviceImpl struct {
    db Database
}

func (s *serviceImpl) Process(data string) (string, error) {
    results, err := s.db.Query("SELECT * FROM table")
    if err != nil {
        return "", err
    }
    return fmt.Sprintf("Processed %d results", len(results)), nil
}

func main() {
    // Create container
    container := core.NewContainer()
    
    // Register dependencies
    container.Register(func() Database {
        return &databaseImpl{}
    })
    
    container.Register(func(db Database) Service {
        return &serviceImpl{db: db}
    })
    
    // Resolve service (database will be automatically injected)
    var service Service
    if err := container.Resolve(&service); err != nil {
        log.Fatal(err)
    }
    
    // Use service
    result, err := service.Process("data")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(result)
}
```

### Using Options

```go
// Create option
option := core.WithOption("temperature", 0.7)

// Apply option
config := make(map[string]any)
option.Apply(&config)
// config["temperature"] = 0.7
```

### Error Handling Example

```go
package main

import (
    "errors"
    "github.com/lookatitude/beluga-ai/pkg/core"
)

func processInput(input string) error {
    if input == "" {
        return core.NewValidationError(
            "input cannot be empty",
            errors.New("empty string provided"),
        )
    }
    
    if len(input) > 100 {
        return core.NewFrameworkErrorWithCode(
            core.ErrorTypeValidation,
            core.ErrorCodeInvalidInput,
            "input too long",
            nil,
        ).AddContext("max_length", 100).AddContext("actual_length", len(input))
    }
    
    return nil
}

func main() {
    err := processInput("")
    if err != nil {
        if core.IsErrorType(err, core.ErrorTypeValidation) {
            var fwErr *core.FrameworkError
            if core.AsFrameworkError(err, &fwErr) {
                fmt.Printf("Validation error: %s\n", fwErr.Message)
                fmt.Printf("Context: %v\n", fwErr.Context)
            }
        }
    }
}
```

### TracedRunnable Example

```go
package main

import (
    "context"
    "github.com/lookatitude/beluga-ai/pkg/core"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/metric"
)

func main() {
    // Create runnable
    runnable := &MyRunnable{}
    
    // Create tracer and metrics
    tracer := otel.Tracer("my-service")
    meter := otel.Meter("my-service")
    metrics, _ := core.NewMetrics(meter)
    
    // Wrap with tracing
    traced := core.NewTracedRunnable(
        runnable,
        tracer,
        metrics,
        "my_component",
        "my_instance",
    )
    
    // All operations are now traced and metered
    result, err := traced.Invoke(context.Background(), "input")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(result)
}
```

## API Reference

### Container Functions

```go
// NewContainer creates a new dependency injection container with no-op monitoring
func NewContainer() Container

// NewContainerWithOptions creates a new container with custom options
func NewContainerWithOptions(opts ...DIOption) Container
```

### Builder Functions

```go
// NewBuilder creates a new builder with the given container
func NewBuilder(container Container) *Builder

// Builder methods
func (b *Builder) Build(target interface{}) error
func (b *Builder) Register(factoryFunc interface{}) error
func (b *Builder) Singleton(instance interface{})
func (b *Builder) RegisterLogger(factory func() Logger) error
func (b *Builder) RegisterTracerProvider(factory func() TracerProvider) error
func (b *Builder) RegisterMetrics(factory func() (*Metrics, error)) error
func (b *Builder) WithLogger(logger Logger) *Builder
func (b *Builder) WithTracerProvider(tracerProvider TracerProvider) *Builder
```

### Error Functions

```go
// Error constructors
func NewValidationError(message string, cause error) *FrameworkError
func NewNetworkError(message string, cause error) *FrameworkError
func NewAuthenticationError(message string, cause error) *FrameworkError
func NewInternalError(message string, cause error) *FrameworkError
func NewConfigurationError(message string, cause error) *FrameworkError
func NewFrameworkErrorWithCode(errorType ErrorType, code ErrorCode, message string, cause error) *FrameworkError

// Error utilities
func IsErrorType(err error, errorType ErrorType) bool
func AsFrameworkError(err error, target **FrameworkError) bool
func UnwrapError(err error) error
func WrapError(err error, message string) error
```

### Metrics Functions

```go
// NewMetrics creates a new Metrics instance
func NewMetrics(meter metric.Meter) (*Metrics, error)

// NoOpMetrics returns a no-op metrics instance
func NoOpMetrics() *Metrics

// Metrics recording methods
func (m *Metrics) RecordRunnableInvoke(ctx context.Context, componentType string, duration time.Duration, err error)
func (m *Metrics) RecordRunnableBatch(ctx context.Context, componentType string, batchSize int, duration time.Duration, err error)
func (m *Metrics) RecordRunnableStream(ctx context.Context, componentType string, duration time.Duration, chunkCount int, err error)
```

### TracedRunnable Functions

```go
// NewTracedRunnable creates a new TracedRunnable
func NewTracedRunnable(
    runnable Runnable,
    tracer trace.Tracer,
    metrics *Metrics,
    componentType string,
    componentName string,
) *TracedRunnable

// Helper functions
func RunnableWithTracing(
    runnable Runnable,
    tracer trace.Tracer,
    metrics *Metrics,
    componentType string,
) Runnable

func RunnableWithTracingAndName(
    runnable Runnable,
    tracer trace.Tracer,
    metrics *Metrics,
    componentType string,
    componentName string,
) Runnable
```

### Option Functions

```go
// WithOption creates a new Option that sets a key-value pair
func WithOption(key string, value any) Option
```

### Utility Functions

```go
// GenerateRandomString generates a random hex string
func GenerateRandomString(length int) (string, error)

// ContainsString checks if a slice contains a string
func ContainsString(slice []string, s string) bool
```

## Testing

The package includes comprehensive test coverage:

```go
// Mock Runnable for testing
mockRunnable := &core.MockRunnable{}
mockRunnable.WithInvokeResult("result")
mockRunnable.WithInvokeError(nil)

// Test container
container := core.NewContainer()
container.Register(func() string {
    return "test"
})

var result string
container.MustResolve(&result)
// result == "test"

// Test health check
err := container.CheckHealth(context.Background())
assert.NoError(t, err)
```

### Running Tests

```bash
# Run all tests
go test ./pkg/core/...

# Run with coverage
go test ./pkg/core/... -cover

# Run benchmarks
go test ./pkg/core/... -bench=.

# Run with race detection
go test ./pkg/core/... -race
```

## Best Practices

### 1. Use Runnable Interface

All components that can be executed should implement the Runnable interface:

```go
type MyComponent struct{}

func (c *MyComponent) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
    // Implementation
}

func (c *MyComponent) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
    // Implementation
}

func (c *MyComponent) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
    // Implementation
}
```

### 2. Use Dependency Injection

Prefer dependency injection over global state:

```go
// Good: Use DI
container.Register(func(db Database) Service {
    return &serviceImpl{db: db}
})

// Bad: Global state
var globalDB Database
```

### 3. Use Typed Errors

Always use typed errors for better error handling:

```go
// Good: Typed error
return core.NewValidationError("invalid input", cause)

// Bad: Generic error
return fmt.Errorf("invalid input: %w", cause)
```

### 4. Wrap Runnables with Tracing

Always wrap Runnables with tracing for observability:

```go
traced := core.NewTracedRunnable(
    runnable,
    tracer,
    metrics,
    "component_type",
    "component_name",
)
```

### 5. Check Health Regularly

Implement health checks for long-running services:

```go
if err := container.CheckHealth(ctx); err != nil {
    // Handle unhealthy state
}
```

### 6. Use Context for Cancellation

Always respect context cancellation:

```go
func (r *MyRunnable) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
        // Process
    }
}
```

## Package Structure

```
pkg/core/
├── di.go              # Dependency injection container
├── errors.go          # Standardized error types
├── interfaces.go      # Core interface definitions (Loader, Retriever, HealthChecker)
├── metrics.go         # Metrics collection
├── runnable.go        # Runnable interface and Option utilities
├── traced_runnable.go # Tracing instrumentation
├── model/             # Core data models (placeholder)
│   └── model.go
├── utils/             # Utility functions
│   └── utils.go
├── *_test.go          # Comprehensive tests
└── README.md          # This documentation
```

## Related Packages

- [`pkg/schema`](../schema/README.md): Core data structures and message types
- [`pkg/config`](../config/README.md): Configuration management
- [`pkg/monitoring`](../monitoring/README.md): Advanced observability features

## License

This package is part of the Beluga AI Framework and follows the same license terms.
