---
name: Write API Docs
description: API reference documentation from code
personas:
  - documentation-writer
---

# Write API Docs

This skill guides you through creating API reference documentation for Beluga AI packages.

## Prerequisites

- Package code to document
- Understanding of package purpose
- Access to test files for usage examples

## Steps

### 1. Analyze Package Structure

Review the package to identify:

```markdown
## Package Analysis: pkg/[name]

### Public Types
- [ ] `Config` - Configuration struct
- [ ] `Component` - Main type
- [ ] `Interface` - Public interface
- [ ] `Error` - Error type
- [ ] `Option` - Functional option type

### Public Functions
- [ ] `New...()` - Constructor
- [ ] `RegisterGlobal()` - Registry (if applicable)
- [ ] `NewProvider()` - Provider factory (if applicable)

### Public Methods
- [ ] `Component.Method1()`
- [ ] `Component.Method2()`

### Constants/Variables
- [ ] Error codes
- [ ] Default values
```

### 2. Document Package Overview

```markdown
# Package [name]

`import "github.com/lookatitude/beluga-ai/pkg/[name]"`

## Overview

[Brief description of what this package provides]

## Key Features

- [Feature 1]
- [Feature 2]
- [Feature 3]

## Quick Start

```go
package main

import (
    "context"
    "github.com/lookatitude/beluga-ai/pkg/[name]"
)

func main() {
    // Minimal working example
    component := name.NewComponent(
        name.WithTimeout(30 * time.Second),
    )
    result, err := component.Process(context.Background(), input)
    if err != nil {
        // handle error
    }
    // use result
}
```

## Architecture

[Optional: Mermaid diagram showing component relationships]
```

### 3. Document Types

#### Configuration

```markdown
## Types

### Config

Configuration for creating a [Component].

```go
type Config struct {
    // Name is the identifier for this component.
    // Required.
    Name string `mapstructure:"name" validate:"required"`

    // Timeout is the maximum duration for operations.
    // Default: 30s
    Timeout time.Duration `mapstructure:"timeout" validate:"min=1s"`

    // MaxRetries is the number of retry attempts on transient failures.
    // Default: 3, Range: 0-10
    MaxRetries int `mapstructure:"max_retries" validate:"gte=0,lte=10"`
}
```

#### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `Name` | `string` | Yes | - | Identifier for this component |
| `Timeout` | `time.Duration` | No | `30s` | Maximum operation duration |
| `MaxRetries` | `int` | No | `3` | Retry attempts (0-10) |

#### Example

```go
config := name.Config{
    Name:       "my-component",
    Timeout:    60 * time.Second,
    MaxRetries: 5,
}
```
```

#### Main Type

```markdown
### Component

Component is the main type for [purpose].

```go
type Component struct {
    // contains filtered or unexported fields
}
```

#### Creating a Component

```go
// Using default configuration
component := name.NewComponent()

// With functional options
component := name.NewComponent(
    name.WithTimeout(60 * time.Second),
    name.WithLogger(logger),
    name.WithMetrics(metrics),
)

// From configuration
component, err := name.NewComponentFromConfig(config)
```
```

#### Interface

```markdown
### Interface

Interface defines the contract for [purpose].

```go
type Interface interface {
    // Process performs the main operation.
    // Returns ErrCodeInvalidInput if input is invalid.
    // Returns ErrCodeTimeout if the operation exceeds the configured timeout.
    Process(ctx context.Context, input Input) (Output, error)

    // Close releases resources held by the component.
    Close() error
}
```
```

### 4. Document Functions

```markdown
## Functions

### NewComponent

```go
func NewComponent(opts ...Option) *Component
```

NewComponent creates a new Component with the given options.

#### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `opts` | `...Option` | Functional options to configure the component |

#### Returns

| Type | Description |
|------|-------------|
| `*Component` | Configured component instance |

#### Example

```go
component := name.NewComponent(
    name.WithTimeout(30 * time.Second),
    name.WithMaxRetries(5),
)
```

---

### NewProvider

```go
func NewProvider(ctx context.Context, providerName string, config Config) (Interface, error)
```

NewProvider creates a new provider instance by name from the global registry.

#### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | Context for cancellation and tracing |
| `providerName` | `string` | Name of the registered provider |
| `config` | `Config` | Provider configuration |

#### Returns

| Type | Description |
|------|-------------|
| `Interface` | Provider implementing the Interface |
| `error` | Error if provider not found or creation fails |

#### Errors

- `ErrCodeNotFound`: Provider name not registered
- `ErrCodeInvalidConfig`: Configuration validation failed

#### Example

```go
provider, err := name.NewProvider(ctx, "openai", name.Config{
    Timeout: 30 * time.Second,
})
if err != nil {
    log.Fatalf("Failed to create provider: %v", err)
}
```
```

### 5. Document Methods

```markdown
## Methods

### Component.Process

```go
func (c *Component) Process(ctx context.Context, input Input) (Output, error)
```

Process performs the main operation on the given input.

#### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | Context for cancellation, timeout, and tracing |
| `input` | `Input` | Input data to process |

#### Returns

| Type | Description |
|------|-------------|
| `Output` | Processed result |
| `error` | Error if processing fails |

#### Errors

| Error Code | Condition |
|------------|-----------|
| `ErrCodeInvalidInput` | Input validation failed |
| `ErrCodeTimeout` | Operation exceeded timeout |
| `ErrCodeRateLimit` | Rate limit exceeded |

#### Example

```go
result, err := component.Process(ctx, name.Input{
    Value: "test data",
})
if err != nil {
    var e *name.Error
    if errors.As(err, &e) {
        switch e.Code {
        case name.ErrCodeTimeout:
            // Handle timeout
        case name.ErrCodeRateLimit:
            // Handle rate limit
        default:
            // Handle other errors
        }
    }
    return err
}
fmt.Printf("Result: %s\n", result.Value)
```
```

### 6. Document Options

```markdown
## Options

### WithTimeout

```go
func WithTimeout(d time.Duration) Option
```

WithTimeout sets the maximum duration for operations.

#### Example

```go
component := name.NewComponent(
    name.WithTimeout(60 * time.Second),
)
```

---

### WithLogger

```go
func WithLogger(l Logger) Option
```

WithLogger sets the logger for the component.

---

### WithMetrics

```go
func WithMetrics(m *Metrics) Option
```

WithMetrics sets the OTEL metrics collector.
```

### 7. Document Errors

```markdown
## Errors

### Error Type

```go
type Error struct {
    Op   string    // Operation that failed
    Err  error     // Underlying error
    Code ErrorCode // Error classification
}
```

### Error Codes

| Code | Description | Retryable |
|------|-------------|-----------|
| `ErrCodeInvalidInput` | Input validation failed | No |
| `ErrCodeTimeout` | Operation timed out | Yes |
| `ErrCodeRateLimit` | Rate limit exceeded | Yes (with backoff) |
| `ErrCodeNotFound` | Resource not found | No |

### Checking Error Codes

```go
result, err := component.Process(ctx, input)
if err != nil {
    var e *name.Error
    if errors.As(err, &e) {
        if e.Code == name.ErrCodeRateLimit {
            // Implement backoff and retry
        }
    }
    return err
}
```
```

### 8. Document Providers (if applicable)

```markdown
## Providers

### Available Providers

| Provider | Description | Import |
|----------|-------------|--------|
| `openai` | OpenAI API | Built-in |
| `anthropic` | Anthropic API | Built-in |
| `ollama` | Local Ollama | Built-in |

### Registering Custom Providers

```go
func init() {
    name.RegisterGlobal("custom", func(ctx context.Context, config name.Config) (name.Interface, error) {
        return NewCustomProvider(config)
    })
}
```
```

## Documentation Checklist

- [ ] Package overview with import path
- [ ] Quick start example
- [ ] All public types documented
- [ ] All public functions documented
- [ ] All public methods documented
- [ ] All options documented
- [ ] Error codes with descriptions
- [ ] Providers listed (if applicable)
- [ ] Examples compile and run
- [ ] Links to guides and tutorials

## Output

Complete API reference with:
- Package overview
- Type documentation with fields
- Function signatures with parameters/returns
- Method documentation with errors
- Functional options
- Error handling guide
- Provider documentation
- Working code examples
