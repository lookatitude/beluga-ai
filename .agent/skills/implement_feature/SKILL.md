---
name: Implement Feature
description: End-to-end feature implementation following Beluga AI patterns
personas:
  - backend-developer
---

# Implement Feature

This skill guides you through implementing a new feature in Beluga AI, ensuring compliance with all framework patterns and standards.

## Prerequisites

- Understand the feature requirements
- Identify which package(s) will be affected
- Review similar implementations in the codebase

## Steps

### 1. Plan the Implementation

Before writing code:

1. Identify affected packages
2. Define interfaces needed (following ISP)
3. Plan the configuration structure
4. List dependencies (following DIP)
5. Design error handling approach

### 2. Create/Update Package Structure

If creating a new package:

```bash
mkdir -p pkg/<package_name>/{iface,internal,providers}
```

Required files:
- `pkg/<package_name>/<package_name>.go` - Main API
- `pkg/<package_name>/config.go` - Configuration
- `pkg/<package_name>/metrics.go` - OTEL metrics
- `pkg/<package_name>/errors.go` - Error definitions
- `pkg/<package_name>/test_utils.go` - Test helpers
- `pkg/<package_name>/README.md` - Documentation

### 3. Define Interfaces

In `iface/interfaces.go`:

```go
package iface

import "context"

// Use -er suffix for single-method interfaces
type Processor interface {
    Process(ctx context.Context, input Input) (Output, error)
}

// Use descriptive nouns for multi-method interfaces
type DataStore interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte) error
    Delete(ctx context.Context, key string) error
}
```

### 4. Implement Configuration

In `config.go`:

```go
package mypackage

import "github.com/go-playground/validator/v10"

type Config struct {
    Timeout    time.Duration `mapstructure:"timeout" yaml:"timeout" validate:"required,min=1s"`
    MaxRetries int           `mapstructure:"max_retries" yaml:"max_retries" validate:"gte=0,lte=10"`
}

func (c *Config) Validate() error {
    return validator.New().Struct(c)
}

// Functional options
type Option func(*Component)

func WithTimeout(t time.Duration) Option {
    return func(c *Component) { c.timeout = t }
}
```

### 5. Implement OTEL Metrics

In `metrics.go`:

```go
package mypackage

import (
    "go.opentelemetry.io/otel/metric"
    "go.opentelemetry.io/otel/trace"
)

type Metrics struct {
    operationsTotal   metric.Int64Counter
    operationDuration metric.Float64Histogram
    errorsTotal       metric.Int64Counter
    tracer            trace.Tracer
}

func NewMetrics(meter metric.Meter, tracer trace.Tracer) (*Metrics, error) {
    ops, err := meter.Int64Counter("mypackage_operations_total")
    if err != nil {
        return nil, err
    }
    dur, err := meter.Float64Histogram("mypackage_operation_duration_seconds")
    if err != nil {
        return nil, err
    }
    errs, err := meter.Int64Counter("mypackage_errors_total")
    if err != nil {
        return nil, err
    }
    return &Metrics{
        operationsTotal:   ops,
        operationDuration: dur,
        errorsTotal:       errs,
        tracer:            tracer,
    }, nil
}
```

### 6. Implement Error Handling

In `errors.go`:

```go
package mypackage

type ErrorCode string

const (
    ErrCodeInvalidInput ErrorCode = "INVALID_INPUT"
    ErrCodeTimeout      ErrorCode = "TIMEOUT"
    ErrCodeRateLimit    ErrorCode = "RATE_LIMIT"
)

type Error struct {
    Op   string    // Operation that failed
    Err  error     // Underlying error
    Code ErrorCode // Classification
}

func (e *Error) Error() string {
    return fmt.Sprintf("%s: %s: %v", e.Op, e.Code, e.Err)
}

func (e *Error) Unwrap() error {
    return e.Err
}
```

### 7. Implement Core Logic

In `<package_name>.go`:

```go
package mypackage

func NewComponent(opts ...Option) *Component {
    c := &Component{
        // defaults
    }
    for _, opt := range opts {
        opt(c)
    }
    return c
}

func (c *Component) Process(ctx context.Context, input Input) (Output, error) {
    ctx, span := c.metrics.tracer.Start(ctx, "component.process")
    defer span.End()

    start := time.Now()
    defer func() {
        c.metrics.operationDuration.Record(ctx, time.Since(start).Seconds())
    }()

    c.metrics.operationsTotal.Add(ctx, 1)

    // Implementation...

    if err != nil {
        c.metrics.errorsTotal.Add(ctx, 1)
        span.RecordError(err)
        return Output{}, &Error{Op: "Process", Err: err, Code: ErrCodeInvalidInput}
    }

    return output, nil
}
```

### 8. Write Tests

Create comprehensive tests following test standards:

- Table-driven tests
- Concurrency tests
- Error scenario tests
- OTEL validation tests

### 9. Run Quality Checks

```bash
make fmt
make lint
make test-unit
make security
```

### 10. Update Documentation

- Update package README.md
- Add examples if appropriate
- Update main documentation if needed

## Validation Checklist

- [ ] Interfaces follow ISP (small, focused)
- [ ] Configuration has validation tags
- [ ] OTEL metrics implemented
- [ ] Error handling uses Op/Err/Code pattern
- [ ] All public methods have tracing
- [ ] Tests cover happy path, errors, edge cases
- [ ] `make lint` passes
- [ ] `make test-unit` passes

## Output

A complete feature implementation with:
- Clean interfaces
- Configuration with validation
- OTEL observability
- Comprehensive tests
- Documentation
