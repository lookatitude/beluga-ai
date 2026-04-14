---
title: "Context Timeout Management"
description: "Recipe for implementing per-operation timeouts in Go agent workflows with graceful handling and deadline propagation through Runnable chains."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, context timeout, Go deadline propagation, per-operation timeout, graceful cancellation, Runnable chains, timeout recipe"
---

## Problem

Different operations in an agent workflow have vastly different time requirements. An LLM generation call might need 10 seconds, an embedding operation 2 seconds, and a vector search 500 milliseconds. Applying a single timeout to the entire workflow leads to two problems.

First, a short timeout causes fast operations to fail when slow operations time out, losing partial work. Second, a long timeout allows slow operations to block indefinitely. The challenge is providing per-operation timeouts while propagating parent timeouts to child operations.

## Solution

Wrap any `core.Runnable` in a `TimeoutRunnable` that applies an operation-specific timeout on `Invoke`. The `TimeoutManager` maps operation names to configured durations so each operation type gets exactly the time it needs.

## Why This Matters

Per-operation timeouts mean that a hung embedding call fails fast at 2 seconds rather than blocking the entire workflow for 30 seconds. Deadline propagation ensures that if a parent context has 3 seconds remaining, child operations inherit that limit even if their own configured timeout is 10 seconds.

## Code Example

```go
package main

import (
	"context"
	"fmt"
	"iter"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/lookatitude/beluga-ai/core"
)

var tracer = otel.Tracer("beluga.core.timeout")

// TimeoutManager holds per-operation timeout configuration.
type TimeoutManager struct {
	defaultTimeout    time.Duration
	operationTimeouts map[string]time.Duration
}

// NewTimeoutManager creates a TimeoutManager with the given default timeout.
func NewTimeoutManager(defaultTimeout time.Duration) *TimeoutManager {
	return &TimeoutManager{
		defaultTimeout:    defaultTimeout,
		operationTimeouts: make(map[string]time.Duration),
	}
}

// SetOperationTimeout assigns a specific timeout for an operation name.
func (tm *TimeoutManager) SetOperationTimeout(operation string, timeout time.Duration) {
	tm.operationTimeouts[operation] = timeout
}

// GetTimeout returns the effective timeout for an operation.
// If no per-operation timeout is set, the default is returned.
func (tm *TimeoutManager) GetTimeout(operation string) time.Duration {
	if timeout, ok := tm.operationTimeouts[operation]; ok {
		return timeout
	}
	return tm.defaultTimeout
}

// TimeoutRunnable wraps a core.Runnable with per-operation timeout management.
// It implements core.Runnable so it is transparent to callers.
type TimeoutRunnable struct {
	runnable      core.Runnable
	manager       *TimeoutManager
	operationName string
}

var _ core.Runnable = (*TimeoutRunnable)(nil)

// NewTimeoutRunnable creates a timeout wrapper for the given runnable.
func NewTimeoutRunnable(runnable core.Runnable, manager *TimeoutManager, operationName string) *TimeoutRunnable {
	return &TimeoutRunnable{
		runnable:      runnable,
		manager:       manager,
		operationName: operationName,
	}
}

// Invoke executes the inner runnable with a deadline derived from the operation timeout.
// If the parent context has a deadline that expires sooner, the parent deadline wins.
func (tr *TimeoutRunnable) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	ctx, span := tracer.Start(ctx, "timeout.invoke")
	defer span.End()

	timeout := tr.effectiveTimeout(ctx)
	span.SetAttributes(
		attribute.String("timeout.operation", tr.operationName),
		attribute.String("timeout.duration", timeout.String()),
	)

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result, err := tr.runnable.Invoke(timeoutCtx, input, options...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(trace.StatusError, err.Error())
		if timeoutCtx.Err() != nil {
			return nil, fmt.Errorf("operation %s timed out after %v: %w", tr.operationName, timeout, timeoutCtx.Err())
		}
		return nil, err
	}

	span.SetStatus(trace.StatusOK, "operation completed")
	return result, nil
}

// Stream delegates streaming to the inner runnable inside a timeout context.
func (tr *TimeoutRunnable) Stream(ctx context.Context, input any, options ...core.Option) iter.Seq2[any, error] {
	timeout := tr.effectiveTimeout(ctx)
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)

	return func(yield func(any, error) bool) {
		defer cancel()
		for item, err := range tr.runnable.Stream(timeoutCtx, input, options...) {
			if !yield(item, err) {
				return
			}
			if err != nil {
				return
			}
		}
	}
}

// effectiveTimeout returns the smaller of the operation-specific timeout and
// the remaining time on the parent context deadline.
func (tr *TimeoutRunnable) effectiveTimeout(ctx context.Context) time.Duration {
	operationTimeout := tr.manager.GetTimeout(tr.operationName)
	deadline, ok := ctx.Deadline()
	if !ok {
		return operationTimeout
	}
	remaining := time.Until(deadline)
	if remaining < operationTimeout {
		return remaining
	}
	return operationTimeout
}

// --- Example usage ---

// slowRunnable simulates an operation that takes longer than its timeout.
type slowRunnable struct {
	delay time.Duration
}

func (r *slowRunnable) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	select {
	case <-time.After(r.delay):
		return "done", nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (r *slowRunnable) Stream(ctx context.Context, input any, opts ...core.Option) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		result, err := r.Invoke(ctx, input, opts...)
		yield(result, err)
	}
}

func main() {
	manager := NewTimeoutManager(5 * time.Second)
	manager.SetOperationTimeout("llm_call", 10*time.Second)
	manager.SetOperationTimeout("embedding", 2*time.Second)

	// Wrap a slow runnable with the embedding timeout (2s).
	inner := &slowRunnable{delay: 500 * time.Millisecond}
	wrapped := NewTimeoutRunnable(inner, manager, "embedding")

	result, err := wrapped.Invoke(context.Background(), "input")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("result:", result)
}
```

## Explanation

1. **`effectiveTimeout`** — Compares the configured operation timeout against the remaining time on the parent context deadline. This ensures that when a parent operation times out, child operations also terminate rather than running until their own longer timeout expires.

2. **`context.WithTimeout` on `Invoke`** — Creates a deadline-bounded context for each call. The `defer cancel()` pattern ensures the context is released immediately when the call returns, preventing goroutine leaks from the context's internal timer.

3. **`Stream` timeout propagation** — The timeout context is passed to the inner runnable's `Stream`. The producer sees context cancellation and stops yielding, which terminates the `iter.Seq2` iterator cleanly.

4. **Transparency** — `TimeoutRunnable` implements `core.Runnable` so it is substitutable anywhere a `core.Runnable` is expected. Timeout behavior is added through construction, not code changes.

## Testing

```go
func TestTimeoutRunnable_TimesOut(t *testing.T) {
	manager := NewTimeoutManager(100 * time.Millisecond)
	inner := &slowRunnable{delay: 2 * time.Second}
	wrapped := NewTimeoutRunnable(inner, manager, "test_op")

	_, err := wrapped.Invoke(context.Background(), "input")
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestTimeoutRunnable_RespectsParentDeadline(t *testing.T) {
	manager := NewTimeoutManager(10 * time.Second)
	inner := &slowRunnable{delay: 2 * time.Second}
	wrapped := NewTimeoutRunnable(inner, manager, "test_op")

	// Parent gives 100ms — shorter than the 10s operation timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := wrapped.Invoke(ctx, "input")
	if err == nil {
		t.Fatal("expected deadline exceeded, got nil")
	}
}
```

## Variations

### Adaptive Timeouts

Adjust timeouts based on observed p99 latency from a metrics store:

```go
type AdaptiveTimeoutManager struct {
	base    *TimeoutManager
	metrics LatencyMetrics
}

func (a *AdaptiveTimeoutManager) GetTimeout(operation string) time.Duration {
	p99 := a.metrics.P99(operation)
	if p99 == 0 {
		return a.base.GetTimeout(operation)
	}
	// Add 20% headroom above observed p99.
	return time.Duration(float64(p99) * 1.2)
}
```

## Related Recipes

- **[Global Retry Wrappers](/docs/recipes/llm/global-retry)** — Combine retry with timeout management
- **[LLM Error Handling](/docs/recipes/llm/llm-error-handling)** — Timeout handling for LLM operations
