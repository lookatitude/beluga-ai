# Beluga AI Framework - Package Design Patterns

This document outlines the design patterns, conventions, and rules that all packages in the Beluga AI Framework must follow to maintain consistency, extendability, configuration management, and observability.

## Table of Contents

1. [Core Principles](#core-principles)
2. [Package Structure](#package-structure)
3. [Interface Design](#interface-design)
4. [Configuration Management](#configuration-management)
5. [Observability and Monitoring](#observability-and-monitoring)
6. [Error Handling](#error-handling)
7. [Dependency Management](#dependency-management)
8. [Testing Patterns](#testing-patterns)
9. [Documentation Standards](#documentation-standards)
10. [Code Generation and Automation](#code-generation-and-automation)

## Core Principles

### 1. Interface Segregation Principle (ISP)
- Define small, focused interfaces that serve specific purposes
- Avoid "god interfaces" that force implementations to depend on unused methods
- Prefer multiple small interfaces over one large interface

```go
// Good: Focused interfaces
type LLMCaller interface {
    Generate(ctx context.Context, prompt string) (string, error)
}

type Embedder interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// Avoid: Kitchen sink interface
type EverythingProvider interface {
    Generate(ctx context.Context, prompt string) (string, error)
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    Store(ctx context.Context, docs []Document) error
    Search(ctx context.Context, query string) ([]Document, error)
}
```

### 2. Dependency Inversion Principle (DIP)
- High-level modules should not depend on low-level modules
- Both should depend on abstractions (interfaces)
- Use constructor injection for dependencies

```go
// Good: Constructor injection with interfaces
type Agent struct {
    llm LLMCaller
    memory MemoryStore
}

func NewAgent(llm LLMCaller, memory MemoryStore, opts ...Option) *Agent {
    // implementation
}
```

### 3. Single Responsibility Principle (SRP)
- Each package should have one primary responsibility
- Each struct/function should have one reason to change
- Keep packages focused and cohesive

### 4. Composition over Inheritance
- Prefer embedding interfaces/structs over type hierarchies
- Use functional options for configuration
- Enable flexible composition of behaviors

## Package Structure

### Standard Package Layout

Every package must follow this structure:

```
pkg/{package_name}/
├── iface/           # Interfaces and types (if separate from main logic)
├── internal/        # Private implementation details
├── config.go        # Configuration structs and validation
├── {package_name}.go # Main interfaces and factory functions
├── providers/       # Provider implementations (openai/, anthropic/, etc.)
├── metrics.go       # Package-specific metrics
└── {package_name}_test.go # Tests
```

### Package Naming Conventions

- Use lowercase, descriptive names: `llms`, `vectorstores`, `embeddings`
- Avoid abbreviations unless they're widely understood (e.g., `llms` is acceptable)
- Use singular forms: `agent` not `agents`, `tool` not `tools`

### Internal Package Organization

```
pkg/llms/
├── internal/
│   ├── openai/
│   ├── anthropic/
│   └── mock/
├── iface/
│   └── llm.go
├── providers/
│   ├── openai.go
│   ├── anthropic.go
│   └── mock.go
├── config.go
├── metrics.go
├── llms.go
└── llms_test.go
```

## Interface Design

### Interface Naming
- Use descriptive names ending with "er" for single-method interfaces: `Embedder`, `Retriever`
- Use noun-based names for multi-method interfaces: `VectorStore`, `Agent`
- Always provide a comment explaining the interface's purpose

### Interface Stability
- Once an interface is released in a stable version, it must maintain backward compatibility
- Use embedding to extend interfaces without breaking changes:

```go
// v1
type LLMCaller interface {
    Generate(ctx context.Context, prompt string) (string, error)
}

// v2 - backward compatible
type LLMCaller interface {
    LLMCallerV1 // embed v1 interface
    GenerateWithOptions(ctx context.Context, prompt string, opts GenerateOptions) (string, error)
}
```

### Factory Pattern
Every package must provide factory functions for creating instances:

```go
// Factory function with functional options
func NewLLM(caller LLMCaller, opts ...Option) (*LLM, error) {
    l := &LLM{
        caller: caller,
    }
    for _, opt := range opts {
        opt(l)
    }
    return l, nil
}

// Provider-specific factory
func NewOpenAILLM(config OpenAIConfig) (*LLM, error) {
    caller := openai.NewCaller(config.APIKey)
    return NewLLM(caller, WithModel(config.Model))
}
```

## Configuration Management

### Configuration Structs
- Define configuration structs in `config.go`
- Use struct tags for viper mapping: `mapstructure`, `yaml`, `env`
- Include validation tags
- Provide sensible defaults

```go
type Config struct {
    APIKey      string        `mapstructure:"api_key" yaml:"api_key" env:"API_KEY" validate:"required"`
    Model       string        `mapstructure:"model" yaml:"model" env:"MODEL" default:"gpt-3.5-turbo"`
    Timeout     time.Duration `mapstructure:"timeout" yaml:"timeout" env:"TIMEOUT" default:"30s"`
    MaxRetries  int           `mapstructure:"max_retries" yaml:"max_retries" env:"MAX_RETRIES" default:"3"`
    Enabled     bool          `mapstructure:"enabled" yaml:"enabled" env:"ENABLED" default:"true"`
}
```

### Functional Options Pattern
Use functional options for runtime configuration:

```go
type Option func(*LLM)

func WithTimeout(timeout time.Duration) Option {
    return func(l *LLM) {
        l.timeout = timeout
    }
}

func WithRetryPolicy(policy RetryPolicy) Option {
    return func(l *LLM) {
        l.retryPolicy = policy
    }
}
```

### Configuration Validation
- Use struct validation tags with a validation library (e.g., go-playground/validator)
- Validate configuration at creation time
- Return descriptive error messages

```go
import "github.com/go-playground/validator/v10"

func NewLLM(config Config) (*LLM, error) {
    validate := validator.New()
    if err := validate.Struct(config); err != nil {
        return nil, fmt.Errorf("invalid configuration: %w", err)
    }
    // implementation
}
```

## Observability and Monitoring

### OpenTelemetry Integration

All packages must integrate OpenTelemetry for metrics and tracing as the default observability solution.

#### Tracing
- Create spans for all public method calls
- Include relevant context in span tags
- Propagate context through all operations
- Handle errors by setting span status

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

func (l *LLM) Generate(ctx context.Context, prompt string) (string, error) {
    ctx, span := l.tracer.Start(ctx, "llm.generate",
        trace.WithAttributes(
            attribute.String("llm.model", l.model),
            attribute.Int("prompt.length", len(prompt)),
        ))
    defer span.End()

    result, err := l.caller.Generate(ctx, prompt)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return "", err
    }

    span.SetAttributes(attribute.Int("response.length", len(result)))
    return result, nil
}
```

#### Metrics
- Define package-specific metrics in `metrics.go`
- Use OpenTelemetry metrics API
- Include relevant labels for aggregation

```go
import "go.opentelemetry.io/otel/metric"

type Metrics struct {
    requestsTotal   metric.Int64Counter
    requestDuration metric.Float64Histogram
    errorsTotal     metric.Int64Counter
}

func NewMetrics(meter metric.Meter) *Metrics {
    return &Metrics{
        requestsTotal: metric.Must(meter).NewInt64Counter(
            "llm_requests_total",
            metric.WithDescription("Total number of LLM requests"),
        ),
        requestDuration: metric.Must(meter).NewFloat64Histogram(
            "llm_request_duration_seconds",
            metric.WithDescription("Duration of LLM requests"),
        ),
        errorsTotal: metric.Must(meter).NewInt64Counter(
            "llm_errors_total",
            metric.WithDescription("Total number of LLM errors"),
        ),
    }
}
```

#### Structured Logging
- Use structured logging with context
- Include trace IDs and span IDs when available
- Log at appropriate levels (DEBUG, INFO, WARN, ERROR)

```go
func (l *LLM) Generate(ctx context.Context, prompt string) (string, error) {
    l.logger.Info("generating response",
        "model", l.model,
        "prompt_length", len(prompt),
        "trace_id", trace.SpanFromContext(ctx).SpanContext().TraceID().String(),
    )

    // implementation
}
```

### Health Checks
- Implement health check interfaces where appropriate
- Provide meaningful health status information
- Include in service discovery and load balancing

```go
type HealthChecker interface {
    Check(ctx context.Context) error
}

func (l *LLM) Check(ctx context.Context) error {
    // Perform a lightweight health check
    _, err := l.caller.Generate(ctx, "ping")
    return err
}
```

## Error Handling

### Error Types
- Define custom error types for package-specific errors
- Use error wrapping to preserve error chains
- Provide context about what operation failed

```go
type LLMError struct {
    Op   string // operation that failed
    Err  error  // underlying error
    Code string // error code for programmatic handling
}

func (e *LLMError) Error() string {
    return fmt.Sprintf("llm %s: %v", e.Op, e.Err)
}

func (e *LLMError) Unwrap() error {
    return e.Err
}
```

### Error Codes
- Define standard error codes for common failure modes
- Allow programmatic error handling by clients

```go
const (
    ErrCodeRateLimit     = "rate_limit"
    ErrCodeInvalidConfig = "invalid_config"
    ErrCodeNetworkError  = "network_error"
    ErrCodeTimeout       = "timeout"
)
```

### Context Cancellation
- Always respect context cancellation
- Use context.WithTimeout for operations with deadlines
- Propagate context through all async operations

```go
func (l *LLM) Generate(ctx context.Context, prompt string) (string, error) {
    ctx, cancel := context.WithTimeout(ctx, l.timeout)
    defer cancel()

    // implementation that respects ctx.Done()
    select {
    case <-ctx.Done():
        return "", ctx.Err()
    case result := <-l.generateAsync(ctx, prompt):
        return result.response, result.err
    }
}
```

## Dependency Management

### Import Organization
- Group imports by standard library, third-party, and internal
- Use blank lines between import groups
- Keep import paths clean and unambiguous

```go
import (
    "context"
    "fmt"
    "time"

    "go.opentelemetry.io/otel/trace"

    "github.com/lookatitude/beluga-ai/pkg/config"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)
```

### Dependency Injection
- Prefer explicit dependencies over global state
- Use interfaces for external dependencies
- Enable testing with mocks/stubs

### Version Compatibility
- Specify version constraints in go.mod
- Test against minimum supported versions
- Document breaking changes clearly

## Testing Patterns

### Test Structure
- Place tests in `*_test.go` files in the same package
- Use table-driven tests for multiple test cases
- Test both success and failure scenarios

```go
func TestLLM_Generate(t *testing.T) {
    tests := []struct {
        name     string
        prompt   string
        mockResp string
        mockErr  error
        want     string
        wantErr  bool
    }{
        {
            name:     "successful generation",
            prompt:   "Hello",
            mockResp: "Hi there!",
            want:     "Hi there!",
            wantErr:  false,
        },
        // more test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

### Mocking
- Use interfaces to enable easy mocking
- Provide mock implementations in `internal/mock/` or `providers/mock.go`
- Test error conditions and edge cases

### Benchmarking
- Include benchmarks for performance-critical code
- Use realistic data sizes and scenarios
- Compare performance across versions

```go
func BenchmarkLLM_Generate(b *testing.B) {
    // benchmark implementation
}
```

## Documentation Standards

### Package Documentation
- Every package must have a package comment explaining its purpose
- Document interface methods with clear descriptions
- Explain parameters, return values, and error conditions

```go
// Package llms provides interfaces and implementations for Large Language Model interactions.
// It supports multiple LLM providers including OpenAI, Anthropic, and local models.
package llms
```

### Function Documentation
- Document all exported functions and methods
- Explain purpose, parameters, return values, and potential errors
- Include usage examples where helpful

```go
// Generate sends a prompt to the LLM and returns the generated response.
// It handles retries, timeouts, and error propagation automatically.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - prompt: The text prompt to send to the LLM
//
// Returns:
//   - string: The generated response from the LLM
//   - error: Any error that occurred during generation
//
// Example:
//
//	response, err := llm.Generate(ctx, "What is the capital of France?")
//	if err != nil {
//	    log.Printf("Generation failed: %v", err)
//	    return
//	}
//	fmt.Println(response)
func (l *LLM) Generate(ctx context.Context, prompt string) (string, error)
```

### README Files
- Include README.md in complex packages
- Document setup, configuration, and usage examples
- Provide troubleshooting information

## Code Generation and Automation

### Interface Generation
- Use code generation for boilerplate interface implementations
- Generate mock implementations for testing
- Automate repetitive patterns

### Configuration Validation
- Generate validation code from struct tags
- Provide compile-time guarantees where possible
- Automate configuration documentation

### Metrics Registration
- Generate metrics registration code
- Ensure consistent metric naming across packages
- Automate metric collection setup

## Migration and Evolution

### Backward Compatibility
- Avoid breaking changes in stable APIs
- Deprecate old APIs with clear migration paths
- Provide migration guides for major version changes

### Versioning Strategy
- Use semantic versioning (MAJOR.MINOR.PATCH)
- Document breaking changes clearly
- Support multiple major versions during transition periods

### Deprecation Policy
- Mark deprecated APIs with deprecation notices
- Provide replacement APIs before removing deprecated code
- Give users sufficient time to migrate (typically one major version cycle)

---

This document serves as the authoritative guide for package design in the Beluga AI Framework. All new packages must adhere to these patterns, and existing packages should be gradually migrated to comply with these standards. Questions about these patterns should be directed to the framework maintainers.
