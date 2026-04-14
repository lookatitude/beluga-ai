---
title: "Global Retry Wrappers"
description: "Recipe for adding configurable retry logic with exponential backoff to any Go Runnable — LLMs, agents, or retrievers — without modifying components."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go retry wrapper, exponential backoff, Runnable middleware, transient error handling, resilience recipe, production patterns"
---

## Problem

You need to add retry logic with exponential backoff to any `core.Runnable` component (LLMs, agents, retrievers) without modifying each component individually. This is especially important for handling transient network errors, rate limits, and temporary service unavailability.

## Solution

Create a `RetryRunnable` wrapper that implements `core.Runnable` and delegates to an inner `Runnable` with configurable retry logic on `Invoke`. This follows Beluga AI's middleware pattern: compose behavior around interfaces without touching implementations.

## Why This Matters

Individual retry logic per component leads to code duplication and inconsistent behavior. A database retriever, an LLM call, and a tool execution all need retry handling, but the retry mechanics (backoff, jitter, error classification) are identical. The `RetryRunnable` wrapper extracts this cross-cutting concern into a single, testable component.

The wrapper implements `core.Runnable` so it is transparent to callers. You can add or remove retry behavior through construction rather than code changes.

## Code Example

```go
package main

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
	"math/rand"
	"time"

	"github.com/lookatitude/beluga-ai/core"
)

// RetryConfig configures retry behavior.
type RetryConfig struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	BackoffFactor  float64
	Jitter         float64 // Fraction of backoff added as random jitter (0.0 to 1.0).
}

// DefaultRetryConfig provides sensible defaults for LLM API calls.
var DefaultRetryConfig = RetryConfig{
	MaxRetries:     3,
	InitialBackoff: 1 * time.Second,
	MaxBackoff:     30 * time.Second,
	BackoffFactor:  2.0,
	Jitter:         0.1,
}

// RetryRunnable wraps a core.Runnable with retry logic.
// It implements core.Runnable so it is transparent to callers.
type RetryRunnable struct {
	inner  core.Runnable
	config RetryConfig
}

// NewRetryRunnable creates a retry wrapper around any core.Runnable.
func NewRetryRunnable(inner core.Runnable, config RetryConfig) *RetryRunnable {
	return &RetryRunnable{inner: inner, config: config}
}

// Invoke executes the inner runnable with exponential backoff retry.
func (rr *RetryRunnable) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	backoff := rr.config.InitialBackoff
	var lastErr error

	for attempt := 0; attempt <= rr.config.MaxRetries; attempt++ {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("context cancelled before attempt %d: %w", attempt, ctx.Err())
		}

		result, err := rr.inner.Invoke(ctx, input, opts...)
		if err == nil {
			return result, nil
		}
		lastErr = err

		if !core.IsRetryable(err) {
			return nil, fmt.Errorf("permanent error (not retrying): %w", err)
		}

		if attempt < rr.config.MaxRetries {
			wait := rr.jittered(backoff)
			slog.Warn("retrying after error",
				"attempt", attempt+1,
				"max", rr.config.MaxRetries,
				"wait", wait,
				"error", err,
			)
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled during backoff: %w", ctx.Err())
			case <-time.After(wait):
			}
			backoff = min(time.Duration(float64(backoff)*rr.config.BackoffFactor), rr.config.MaxBackoff)
		}
	}

	return nil, fmt.Errorf("max retries (%d) exceeded: %w", rr.config.MaxRetries, lastErr)
}

// Stream delegates to the inner runnable's Stream. Retry on stream creation
// errors is not supported because streaming responses are not idempotent.
func (rr *RetryRunnable) Stream(ctx context.Context, input any, opts ...core.Option) iter.Seq2[any, error] {
	return rr.inner.Stream(ctx, input, opts...)
}

// jittered adds random jitter to a backoff duration.
func (rr *RetryRunnable) jittered(d time.Duration) time.Duration {
	if rr.config.Jitter == 0 {
		return d
	}
	delta := float64(d) * rr.config.Jitter * (rand.Float64()*2 - 1)
	return time.Duration(float64(d) + delta)
}

// min returns the smaller of two durations.
func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

// --- Example usage ---

// exampleRunnable is a simple core.Runnable for demonstration.
type exampleRunnable struct {
	failN int // Fail the first N invocations.
	calls int
}

func (r *exampleRunnable) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	r.calls++
	if r.calls <= r.failN {
		return nil, core.NewError("example.invoke", core.ErrProviderDown,
			fmt.Sprintf("simulated failure %d", r.calls), nil)
	}
	return fmt.Sprintf("success on attempt %d", r.calls), nil
}

func (r *exampleRunnable) Stream(ctx context.Context, input any, opts ...core.Option) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		result, err := r.Invoke(ctx, input, opts...)
		yield(result, err)
	}
}

func main() {
	ctx := context.Background()

	// Wrap a runnable that fails twice before succeeding.
	inner := &exampleRunnable{failN: 2}
	wrapped := NewRetryRunnable(inner, DefaultRetryConfig)

	result, err := wrapped.Invoke(ctx, "hello")
	if err != nil {
		slog.Error("all retries exhausted", "error", err)
		return
	}
	fmt.Println("Result:", result)
}
```

## Explanation

1. **`core.IsRetryable(err)`** -- Uses Beluga AI's error classification to distinguish transient errors (`ErrRateLimit`, `ErrTimeout`, `ErrProviderDown`) from permanent ones (`ErrAuth`, `ErrInvalidInput`). Only transient errors are retried.

2. **Exponential backoff with jitter** -- The wait doubles each attempt (`BackoffFactor: 2.0`) and is capped at `MaxBackoff`. Random jitter (±10% of the backoff) prevents multiple clients from retrying simultaneously and triggering another rate limit.

3. **Context respect** -- The context is checked before each attempt and during the backoff `select`. This ensures the wrapper honours timeouts and cancellations from the calling code.

4. **Stream delegation** -- Streaming responses are not retried because re-issuing a streaming request may produce different output than the original. If you need retry on stream creation failures, wrap `Invoke` and buffer the complete response, then return it as a single-item stream.

## Testing

```go
func TestRetryRunnable_RetriesOnRetryableError(t *testing.T) {
	inner := &exampleRunnable{failN: 2}
	wrapped := NewRetryRunnable(inner, RetryConfig{
		MaxRetries:     3,
		InitialBackoff: time.Millisecond, // Fast for tests.
		BackoffFactor:  2.0,
	})

	result, err := wrapped.Invoke(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inner.calls != 3 {
		t.Errorf("expected 3 calls, got %d", inner.calls)
	}
	_ = result
}
```

## Related Recipes

- **[LLM Error Handling](/docs/recipes/llm/llm-error-handling)** -- Classification and handling of LLM API errors
- **[Circuit Breaker](/docs/recipes/infrastructure/#circuit-breaker-for-provider-outages)** -- Short-circuit repeated failures
