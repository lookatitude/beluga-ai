---
title: "Advanced Context Timeout Management"
description: "Implement fine-grained per-operation timeouts with graceful handling and deadline propagation through Runnable chains."
---

## Problem

Different operations in an agent workflow have vastly different time requirements. An LLM generation call might need 10 seconds, an embedding operation 2 seconds, and a vector search 500 milliseconds. Applying a single timeout to the entire workflow leads to two problems.

First, a short timeout causes fast operations to fail when slow operations timeout, losing partial work. If your workflow generates embeddings, performs retrieval, and then calls an LLM, and the LLM times out, you've wasted the embedding and retrieval work. Second, a long timeout allows slow operations to block indefinitely. If embedding hangs, the entire workflow waits for the long timeout instead of failing fast.

The challenge is providing per-operation timeouts while propagating parent timeouts to child operations. When a parent operation has 5 seconds remaining, child operations should respect that limit even if their configured timeout is 10 seconds. This requires timeout hierarchies and deadline propagation.

Additionally, abrupt timeout cancellation can leave resources in an inconsistent state. A database transaction might timeout mid-write, a file mid-upload, or an LLM call mid-generation. You need graceful timeout handling that allows cleanup before terminating operations.

## Solution

Per-operation timeout management solves this by assigning different timeouts based on operation type. The timeout manager maintains a registry of operation timeouts, allowing you to configure LLM calls, embeddings, retrievals, and tool calls independently. This configuration happens once at startup and applies uniformly across all agent executions.

The timeout wrapper creates operation-specific contexts with appropriate deadlines. When an operation starts, it checks the configured timeout for that operation type and creates a context with that deadline. The goroutine pattern allows the operation to run while the wrapper monitors for timeout, completion, or errors.

Graceful timeout handling adds a grace period after timeout. When an operation times out, instead of immediately returning the error, the wrapper waits briefly for cleanup to complete. This is not a timeout extension; the operation has already been canceled via context. The grace period simply allows goroutines time to notice the cancellation and clean up resources.

Deadline propagation ensures child operations respect parent timeouts. Before creating an operation-specific timeout, check if the parent context has a deadline. If the parent deadline is sooner than the operation timeout, use the parent deadline instead. This ensures that when a parent operation times out, all child operations also terminate.

Batch timeout distribution divides the total timeout across items in a batch. If you have 10 items and a 5-second timeout, each item gets 500 milliseconds. This ensures the entire batch completes within the timeout while giving each item a fair share. The minimum per-item timeout prevents division from creating impractically short timeouts for large batches.

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

1. **Per-operation timeouts optimize resource usage** — By assigning operation-specific timeouts, you allow fast operations to fail quickly while giving slow operations adequate time. This improves overall system responsiveness because failures propagate immediately for operations that should be fast, rather than waiting for a global timeout. Each operation gets exactly the time it needs, no more and no less.

2. **Graceful timeout handling prevents resource leaks** — The grace period after timeout allows goroutines time to notice context cancellation and clean up. Without this, you might leave database connections open, temporary files undeleted, or memory unreleased. The grace period is not for continuing work; the operation has already been canceled. It's purely for cleanup to complete safely.

3. **Batch timeout distribution ensures fairness** — Dividing timeout across batch items guarantees the entire batch completes within the timeout while giving each item equal opportunity. Without this, early items might consume most of the timeout, causing later items to fail immediately. The per-item minimum prevents impractically short timeouts that guarantee failure.

4. **Deadline propagation maintains timeout hierarchies** — When a parent operation times out, child operations must also terminate to avoid wasted work. By checking parent deadlines before setting operation-specific timeouts, you ensure that child operations respect parent limits. This prevents the situation where a parent times out but child operations continue executing, consuming resources for work that will be discarded.

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
