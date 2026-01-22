# Comprehensive Middleware Implementation

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this tutorial, we'll implement robust middleware for Beluga AI applications. Middleware allows you to intercept, modify, and monitor requests and responses across your AI pipeline.

## Learning Objectives
- ✅ Understand the middleware pattern in Beluga AI
- ✅ Create request/response interceptors
- ✅ Implement error handling middleware
- ✅ Build logging and monitoring middleware

## Introduction
Welcome, colleague! Middleware wraps a `Runnable` component to add behavior without modifying the component itself. Common use cases include logging, validation, error handling, and rate limiting.

## Step 1: The Middleware Pattern

In Beluga AI, middleware is typically implemented as a function that takes a `Runnable` and returns a `Runnable`.
```go
type Middleware func(core.Runnable) core.Runnable
```

## Step 2: Creating a Logging Middleware

Let's create middleware that logs execution time and results.
```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/core"
)

// LoggingMiddleware wraps a runnable with logging
func LoggingMiddleware(next core.Runnable) core.Runnable {
    return &loggingRunnable{next: next}
}

type loggingRunnable struct {
    next core.Runnable
}

func (l *loggingRunnable) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
    start := time.Now()
    fmt.Printf("[START] Input: %v\n", input)
    
    output, err := l.next.Invoke(ctx, input, opts...)
    
    duration := time.Since(start)
    if err != nil {
        fmt.Printf("[ERROR] %v (took %v)\n", err, duration)
        return nil, err
    }
    
    fmt.Printf("[SUCCESS] Output: %v (took %v)\n", output, duration)
    return output, nil
}

// Implement Stream and Batch similarly...
func (l *loggingRunnable) Stream(ctx context.Context, input any, opts ...core.Option) (<-chan any, error) {
    fmt.Println("[STREAM START]")
    return l.next.Stream(ctx, input, opts...)
}

func (l *loggingRunnable) Batch(ctx context.Context, inputs []any, opts ...core.Option) ([]any, error) {
    fmt.Printf("[BATCH START] %d items\n", len(inputs))
    return l.next.Batch(ctx, inputs, opts...)
}
```

## Step 3: Creating Validation Middleware

Middleware that validates input before passing it on.
```go
func ValidationMiddleware(next core.Runnable) core.Runnable {
    return &validationRunnable{next: next}
}

type validationRunnable struct {
    next core.Runnable
}

func (v *validationRunnable) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
    // Check if input is a string and not empty
    if str, ok := input.(string); ok {
        if str == "" {
            return nil, fmt.Errorf("validation error: input cannot be empty")
        }
    }

    
    return v.next.Invoke(ctx, input, opts...)
}
// Implement Stream/Batch...
```

## Step 4: Creating Error Handling Middleware

Middleware that recovers from panics or transforms errors.
```go
func ErrorHandlingMiddleware(next core.Runnable) core.Runnable {
    return &errorRunnable{next: next}
}

type errorRunnable struct {
    next core.Runnable
}

func (e *errorRunnable) Invoke(ctx context.Context, input any, opts ...core.Option) (output any, err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("panic recovered: %v", r)
        }
    }()

    
    return e.next.Invoke(ctx, input, opts...)
}
// Implement Stream/Batch...
```

## Step 5: Applying Middleware

Now let's apply these middleware to a simple runnable.
// Simple runnable for demonstration
```go
type EchoRunnable struct{}

func (e *EchoRunnable) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
    return fmt.Sprintf("Echo: %v", input), nil
}
// ... Stream/Batch stubs ...

func main() {
    base := &EchoRunnable{}
    
    // Apply middleware (outer wraps inner)
    // Pipeline: Error -> Logging -> Validation -> Base
    runner := ErrorHandlingMiddleware(
        LoggingMiddleware(
            ValidationMiddleware(base),
        ),
    )
    
    ctx := context.Background()
    
    // Test successful execution
    runner.Invoke(ctx, "Hello World")
    
    // Test validation error
    runner.Invoke(ctx, "")
}
```

## Step 6: Using `ApplyMiddleware` Helper

You can create a helper to apply multiple middleware cleanly.
```go
func ApplyMiddleware(base core.Runnable, middleware ...func(core.Runnable) core.Runnable) core.Runnable {
    wrapped := base
    // Apply in reverse order to preserve expected wrapping layers
    for i := len(middleware) - 1; i >= 0; i-- {
        wrapped = middleware[i](wrapped)
    }
    return wrapped
}

// Usage
runner := ApplyMiddleware(base,
    ValidationMiddleware,
    LoggingMiddleware,
    ErrorHandlingMiddleware,
)
```

## Verification

1. Run the code with valid input and check logs.
2. Run with invalid input and check validation error.
3. Add a panic in `EchoRunnable` and verify `ErrorHandlingMiddleware` catches it.

## Next Steps

- **[Building a Custom Runnable](./core-custom-runnable.md)** - Review runnable basics
- **[End-to-End Tracing with OpenTelemetry](../monitoring/monitoring-otel-tracing.md)** - Advanced observability
