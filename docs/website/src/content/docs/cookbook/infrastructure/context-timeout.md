---
title: "Advanced Context Timeout Management"
description: "Implement fine-grained per-operation timeouts with graceful handling and deadline propagation through Runnable chains."
---

## Problem

You need fine-grained control over timeouts for different operations in a `Runnable` chain, with the ability to set per-operation timeouts, cascade timeouts through nested operations, and handle timeout errors gracefully without losing partial work.

## Solution

Implement a timeout manager that creates operation-specific contexts with appropriate deadlines, tracks timeout hierarchies, and provides timeout-aware error handling. This works because Go's `context` package supports deadline propagation, allowing you to set different timeouts at different levels of your call stack.

## Code Example

```go
package main

import (
    "context"
    "fmt"
    "time"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"

    "github.com/lookatitude/beluga-ai/core"
)

var tracer = otel.Tracer("beluga.core.timeout")

// TimeoutConfig defines timeout settings for an operation
type TimeoutConfig struct {
    OperationTimeout time.Duration
    GracePeriod      time.Duration // Time to allow cleanup after timeout
    PropagateTimeout bool          // Whether to propagate timeout to nested operations
}

// TimeoutManager manages timeouts for Runnable operations
type TimeoutManager struct {
    defaultTimeout    time.Duration
    operationTimeouts map[string]time.Duration
}

// NewTimeoutManager creates a new timeout manager
func NewTimeoutManager(defaultTimeout time.Duration) *TimeoutManager {
    return &TimeoutManager{
        defaultTimeout:    defaultTimeout,
        operationTimeouts: make(map[string]time.Duration),
    }
}

// SetOperationTimeout sets a specific timeout for an operation
func (tm *TimeoutManager) SetOperationTimeout(operation string, timeout time.Duration) {
    tm.operationTimeouts[operation] = timeout
}

// GetTimeout returns the timeout for an operation
func (tm *TimeoutManager) GetTimeout(operation string) time.Duration {
    if timeout, exists := tm.operationTimeouts[operation]; exists {
        return timeout
    }
    return tm.defaultTimeout
}

// TimeoutRunnable wraps a Runnable with timeout management
type TimeoutRunnable struct {
    runnable      core.Runnable
    manager       *TimeoutManager
    operationName string
}

// NewTimeoutRunnable creates a new timeout wrapper
func NewTimeoutRunnable(runnable core.Runnable, manager *TimeoutManager, operationName string) *TimeoutRunnable {
    return &TimeoutRunnable{
        runnable:      runnable,
        manager:       manager,
        operationName: operationName,
    }
}

// Invoke executes with timeout management
func (tr *TimeoutRunnable) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
    ctx, span := tracer.Start(ctx, "timeout.invoke")
    defer span.End()

    timeout := tr.manager.GetTimeout(tr.operationName)
    span.SetAttributes(
        attribute.String("timeout.operation", tr.operationName),
        attribute.String("timeout.duration", timeout.String()),
    )

    // Create timeout context
    timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    // Execute with timeout
    type result struct {
        value any
    }
    resultCh := make(chan result, 1)
    errCh := make(chan error, 1)

    go func() {
        res, err := tr.runnable.Invoke(timeoutCtx, input, options...)
        if err != nil {
            errCh <- err
            return
        }
        resultCh <- result{value: res}
    }()

    select {
    case res := <-resultCh:
        span.SetStatus(trace.StatusOK, "operation completed")
        return res.value, nil

    case err := <-errCh:
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return nil, err

    case <-timeoutCtx.Done():
        timeoutErr := fmt.Errorf("operation %s timed out after %v", tr.operationName, timeout)
        span.RecordError(timeoutErr)
        span.SetStatus(trace.StatusError, "timeout exceeded")

        // Allow grace period for cleanup
        if tr.manager.defaultTimeout > 0 {
            graceCtx, graceCancel := context.WithTimeout(context.Background(), tr.manager.defaultTimeout)
            defer graceCancel()

            select {
            case <-graceCtx.Done():
            case <-time.After(100 * time.Millisecond):
            }
        }

        return nil, timeoutErr
    }
}

// Batch executes with per-item timeout management
func (tr *TimeoutRunnable) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
    ctx, span := tracer.Start(ctx, "timeout.batch")
    defer span.End()

    span.SetAttributes(attribute.Int("batch.size", len(inputs)))

    totalTimeout := tr.manager.GetTimeout(tr.operationName)
    perItemTimeout := totalTimeout / time.Duration(len(inputs))
    if perItemTimeout < 100*time.Millisecond {
        perItemTimeout = 100 * time.Millisecond
    }

    results := make([]any, len(inputs))
    errors := make([]error, len(inputs))

    for i, input := range inputs {
        itemCtx, cancel := context.WithTimeout(ctx, perItemTimeout)
        result, err := tr.runnable.Invoke(itemCtx, input, options...)
        cancel()

        results[i] = result
        errors[i] = err
    }

    for _, err := range errors {
        if err != nil {
            span.RecordError(err)
            span.SetStatus(trace.StatusError, "batch operation failed")
            return results, fmt.Errorf("batch operation failed: %w", err)
        }
    }

    span.SetStatus(trace.StatusOK, "batch completed")
    return results, nil
}

// Stream manages timeout for streaming operations
func (tr *TimeoutRunnable) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
    ctx, span := tracer.Start(ctx, "timeout.stream")
    defer span.End()

    timeout := tr.manager.GetTimeout(tr.operationName)
    timeoutCtx, cancel := context.WithTimeout(ctx, timeout)

    outputCh := make(chan any)

    go func() {
        defer close(outputCh)
        defer cancel()

        streamCh, err := tr.runnable.Stream(timeoutCtx, input, options...)
        if err != nil {
            outputCh <- err
            return
        }

        for {
            select {
            case item, ok := <-streamCh:
                if !ok {
                    span.SetStatus(trace.StatusOK, "stream completed")
                    return
                }
                select {
                case outputCh <- item:
                case <-timeoutCtx.Done():
                    span.RecordError(timeoutCtx.Err())
                    return
                }
            case <-timeoutCtx.Done():
                span.RecordError(timeoutCtx.Err())
                span.SetStatus(trace.StatusError, "stream timeout")
                return
            }
        }
    }()

    return outputCh, nil
}

func main() {
    manager := NewTimeoutManager(5 * time.Second)
    manager.SetOperationTimeout("llm_call", 10*time.Second)
    manager.SetOperationTimeout("embedding", 2*time.Second)

    // Create timeout-aware runnable
    // timeoutRunnable := NewTimeoutRunnable(runnable, manager, "llm_call")
    // result, err := timeoutRunnable.Invoke(ctx, input)
    fmt.Println("Timeout manager created successfully")
}
```

## Explanation

1. **Per-operation timeouts** — Different timeouts are allowed for different operations. An LLM call might need 10 seconds, while an embedding operation only needs 2 seconds. This prevents slow operations from blocking fast ones.

2. **Graceful timeout handling** — When a timeout occurs, a grace period is provided for cleanup. This is important because abruptly canceling operations can leave resources in an inconsistent state.

3. **Batch timeout distribution** — For batch operations, the total timeout is divided across items. This ensures the entire batch completes within the timeout while giving each item a fair share.

Always propagate context timeouts through your call stack. If a parent operation times out, child operations should also respect that timeout to avoid wasted work.

## Testing

```go
func TestTimeoutRunnable_TimeoutExceeded(t *testing.T) {
    ctx := context.Background()

    slowRunnable := &MockRunnable{
        invokeFunc: func(ctx context.Context, input any, opts ...core.Option) (any, error) {
            time.Sleep(2 * time.Second)
            return "result", nil
        },
    }

    manager := NewTimeoutManager(1 * time.Second)
    timeoutRunnable := NewTimeoutRunnable(slowRunnable, manager, "test")

    _, err := timeoutRunnable.Invoke(ctx, "input")

    if err == nil {
        t.Error("Expected timeout error")
    }
}
```

## Variations

### Hierarchical Timeout Propagation

Propagate remaining time to nested operations:

```go
func (tr *TimeoutRunnable) InvokeWithRemainingTime(ctx context.Context, input any, options ...core.Option) (any, error) {
    deadline, ok := ctx.Deadline()
    if !ok {
        return tr.Invoke(ctx, input, options...)
    }

    remaining := time.Until(deadline)
    if remaining < tr.manager.GetTimeout(tr.operationName) {
        timeoutCtx, cancel := context.WithTimeout(ctx, remaining)
        defer cancel()
        return tr.runnable.Invoke(timeoutCtx, input, options...)
    }

    return tr.Invoke(ctx, input, options...)
}
```

### Adaptive Timeouts

Adjust timeouts based on operation history:

```go
type AdaptiveTimeoutManager struct {
    baseTimeout time.Duration
    history     []time.Duration
}
```

## Related Recipes

- **[Global Retry Wrappers](./global-retry)** — Combine retry with timeout management
- **[LLM Error Handling](./llm-error-handling)** — Timeout handling for LLM operations
