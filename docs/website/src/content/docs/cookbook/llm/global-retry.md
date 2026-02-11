---
title: "Global Retry Wrappers"
description: "Add configurable retry logic with exponential backoff to any Runnable component without modifying each component individually."
---

## Problem

You need to add retry logic with exponential backoff to any `Runnable` component (LLMs, agents, retrievers) without modifying each component individually. This is especially important for handling transient network errors, rate limits, and temporary service unavailability.

## Solution

Create a retry wrapper that implements the `Runnable` interface and wraps any other `Runnable` with configurable retry logic. This works because Beluga AI's `Runnable` interface allows composition -- you can wrap any runnable with additional behavior without changing the underlying implementation.

## Why This Matters

Individual retry logic per component leads to code duplication and inconsistent behavior. A database retriever, an LLM call, and a tool execution all need retry handling, but the retry mechanics (backoff, jitter, classification) are identical. The `RetryRunnable` wrapper extracts this cross-cutting concern into a single, testable component that can wrap any `Runnable` in the framework.

This follows Beluga AI's middleware pattern (`func(T) T`), but at the `Runnable` level rather than the `ChatModel` level. The wrapper implements all three `Runnable` methods (`Invoke`, `Batch`, `Stream`), so it's transparent to callers -- they don't know retry logic is applied. This transparency is important because it means you can add or remove retry behavior through configuration rather than code changes.

The OpenTelemetry instrumentation in this wrapper is particularly valuable in production: spans record each retry attempt, the backoff duration, and whether the error was classified as retryable. This data helps you tune retry parameters based on actual failure patterns rather than guesswork.

## Code Example

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "log"
    "math/rand"
    "time"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"

    "github.com/lookatitude/beluga-ai/core"
)

var tracer = otel.Tracer("beluga.core.retry")

// RetryConfig configures retry behavior
type RetryConfig struct {
    MaxRetries      int
    InitialBackoff  time.Duration
    MaxBackoff      time.Duration
    BackoffFactor   float64
    Jitter          float64
    RetryableErrors []error // Errors that should trigger retry
}

// DefaultRetryConfig provides sensible defaults
var DefaultRetryConfig = RetryConfig{
    MaxRetries:     3,
    InitialBackoff: 1 * time.Second,
    MaxBackoff:     30 * time.Second,
    BackoffFactor:  2.0,
    Jitter:         0.1,
}

// RetryRunnable wraps a Runnable with retry logic
type RetryRunnable struct {
    runnable core.Runnable
    config   RetryConfig
}

// NewRetryRunnable creates a new retry wrapper
func NewRetryRunnable(runnable core.Runnable, config RetryConfig) *RetryRunnable {
    return &RetryRunnable{
        runnable: runnable,
        config:   config,
    }
}

// Invoke executes the runnable with retry logic
func (rr *RetryRunnable) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
    ctx, span := tracer.Start(ctx, "retry.invoke")
    defer span.End()

    var lastErr error
    backoff := rr.config.InitialBackoff

    for attempt := 0; attempt <= rr.config.MaxRetries; attempt++ {
        // Check context before attempting
        if ctx.Err() != nil {
            span.RecordError(ctx.Err())
            span.SetStatus(trace.StatusError, "context cancelled")
            return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
        }

        span.SetAttributes(
            attribute.Int("retry.attempt", attempt),
            attribute.Int("retry.max_retries", rr.config.MaxRetries),
        )

        // Execute the runnable
        result, err := rr.runnable.Invoke(ctx, input, options...)
        if err == nil {
            span.SetAttributes(attribute.Bool("retry.success", true))
            span.SetStatus(trace.StatusOK, "operation succeeded")
            return result, nil
        }

        lastErr = err

        // Check if error is retryable
        if !rr.isRetryable(err) {
            span.SetAttributes(attribute.Bool("retry.retryable", false))
            span.RecordError(err)
            span.SetStatus(trace.StatusError, "non-retryable error")
            return nil, fmt.Errorf("non-retryable error: %w", err)
        }

        // Don't sleep after the last attempt
        if attempt < rr.config.MaxRetries {
            jitteredBackoff := rr.calculateBackoff(backoff)
            span.SetAttributes(
                attribute.String("retry.backoff", jitteredBackoff.String()),
                attribute.Bool("retry.retryable", true),
            )

            log.Printf("Retry attempt %d/%d failed: %v. Retrying in %v",
                attempt+1, rr.config.MaxRetries+1, err, jitteredBackoff)

            select {
            case <-ctx.Done():
                span.RecordError(ctx.Err())
                return nil, fmt.Errorf("context cancelled during backoff: %w", ctx.Err())
            case <-time.After(jitteredBackoff):
            }

            // Increase backoff for next attempt
            backoff = time.Duration(float64(backoff) * rr.config.BackoffFactor)
            if backoff > rr.config.MaxBackoff {
                backoff = rr.config.MaxBackoff
            }
        }
    }

    span.RecordError(lastErr)
    span.SetStatus(trace.StatusError, "max retries exceeded")
    return nil, fmt.Errorf("max retries (%d) exceeded: %w", rr.config.MaxRetries, lastErr)
}

// Batch executes batch operations with retry logic
func (rr *RetryRunnable) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
    ctx, span := tracer.Start(ctx, "retry.batch")
    defer span.End()

    span.SetAttributes(attribute.Int("batch.size", len(inputs)))

    return rr.Invoke(ctx, inputs, options...)
}

// Stream wraps streaming with retry (retries on stream errors)
func (rr *RetryRunnable) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
    ctx, span := tracer.Start(ctx, "retry.stream")
    defer span.End()

    outputCh := make(chan any)

    go func() {
        defer close(outputCh)

        var streamCh <-chan any
        var err error

        // Retry stream creation
        for attempt := 0; attempt <= rr.config.MaxRetries; attempt++ {
            streamCh, err = rr.runnable.Stream(ctx, input, options...)
            if err == nil {
                break
            }

            if !rr.isRetryable(err) || attempt == rr.config.MaxRetries {
                outputCh <- err
                return
            }

            backoff := rr.calculateBackoff(rr.config.InitialBackoff * time.Duration(attempt+1))
            time.Sleep(backoff)
        }

        // Forward stream data
        for item := range streamCh {
            select {
            case outputCh <- item:
            case <-ctx.Done():
                return
            }
        }
    }()

    return outputCh, nil
}

// isRetryable determines if an error should trigger a retry
func (rr *RetryRunnable) isRetryable(err error) bool {
    for _, retryableErr := range rr.config.RetryableErrors {
        if errors.Is(err, retryableErr) {
            return true
        }
    }

    if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
        return false
    }

    return true
}

// calculateBackoff adds jitter to prevent thundering herd
func (rr *RetryRunnable) calculateBackoff(base time.Duration) time.Duration {
    if rr.config.Jitter == 0 {
        return base
    }
    jitter := float64(base) * rr.config.Jitter * (rand.Float64()*2 - 1)
    return time.Duration(float64(base) + jitter)
}

func main() {
    // Create a runnable (e.g., an LLM)
    // runnable := yourLLMRunnable

    // Wrap with retry logic
    retryConfig := RetryConfig{
        MaxRetries:     5,
        InitialBackoff: 500 * time.Millisecond,
        MaxBackoff:     10 * time.Second,
        BackoffFactor:  2.0,
        Jitter:         0.2,
    }

    // retryRunnable := NewRetryRunnable(runnable, retryConfig)
    // result, err := retryRunnable.Invoke(ctx, input)
    fmt.Println("Retry wrapper created successfully")
}
```

## Explanation

1. **Runnable interface implementation** -- `RetryRunnable` implements all three `Runnable` methods (`Invoke`, `Batch`, `Stream`). This allows it to wrap any runnable transparently -- the caller doesn't need to know retry logic is applied. This is the decorator pattern applied to Go interfaces.

2. **Exponential backoff with jitter** -- The backoff doubles each attempt (1s, 2s, 4s) but never exceeds `MaxBackoff`. Jitter adds randomness (+-10% by default) to prevent multiple clients from retrying simultaneously, which would cause another rate limit cascade.

3. **Context awareness** -- `ctx.Err()` is checked before each attempt and during backoff via `select`. This ensures timeouts and cancellations from the calling code are respected, preventing wasted retries when the upstream has already given up.

4. **Stream retry** -- Stream creation is retried, but once the stream is established, data is forwarded without retry. This is intentional: retrying individual stream chunks would produce duplicate data, while retrying stream creation handles the common case of transient connection failures.

## Testing

```go
func TestRetryRunnable_SuccessAfterRetries(t *testing.T) {
    ctx := context.Background()
    attempts := 0

    mockRunnable := &MockRunnable{
        invokeFunc: func(ctx context.Context, input any, opts ...core.Option) (any, error) {
            attempts++
            if attempts < 3 {
                return nil, errors.New("temporary error")
            }
            return "success", nil
        },
    }

    retryRunnable := NewRetryRunnable(mockRunnable, RetryConfig{
        MaxRetries:     5,
        InitialBackoff: 1 * time.Millisecond,
        MaxBackoff:     10 * time.Millisecond,
        BackoffFactor:  2.0,
    })

    result, err := retryRunnable.Invoke(ctx, "test")

    if err != nil {
        t.Errorf("Expected success, got: %v", err)
    }
    if result != "success" {
        t.Errorf("Expected 'success', got: %v", result)
    }
    if attempts != 3 {
        t.Errorf("Expected 3 attempts, got %d", attempts)
    }
}
```

## Variations

### Per-Operation Retry Configuration

Allow different retry configs per operation type, useful when LLM calls need more retries than tool calls:

```go
type RetryRunnableWithOptions struct {
    runnable       core.Runnable
    defaultConfig  RetryConfig
    operationConfigs map[string]RetryConfig
}
```

### Circuit Breaker Integration

Combine with a circuit breaker to stop retrying entirely during sustained outages:

```go
type CircuitBreakerRunnable struct {
    runnable core.Runnable
    breaker  *CircuitBreaker
}
```

## Related Recipes

- **[Context Timeout Management](/cookbook/infrastructure/context-timeout)** -- Advanced timeout handling
- **[LLM Error Handling](/cookbook/llm/llm-error-handling)** -- LLM-specific error handling with retries
