# Pattern: Middleware Chain

## What it is

A middleware is a function `func(T) T` that takes an implementation of interface `T` and returns a wrapped version with the same interface but augmented behaviour. `ApplyMiddleware(impl, mw1, mw2, mw3)` composes multiple middlewares, applying them **outside-in** — the first middleware in the list runs first.

## Why we use it

Cross-cutting concerns — retry, rate limit, logging, guardrails, caching, metrics — need to wrap the *entire* call without modifying the wrapped type. The `func(T) T` signature is the simplest possible composition primitive: function application. It has no dependency injection framework, no builder pattern, no extra indirection.

**Alternatives considered:**
- **Decorator base class.** Requires inheritance and per-interface boilerplate. Go idiomatic is embedding, but that makes composition verbose.
- **Interceptor chains (gRPC style).** Works but invents a new type (`Interceptor`) per interface. `func(T) T` is the minimum.
- **AOP-style weaving.** Not possible in Go without code generation. Rejected.

## How it works

Canonical code from `tool/middleware.go` (see [`.guild/wiki/patterns/middleware.md`](../../.guild/wiki/patterns/middleware.md)):

```go
// tool/middleware.go (excerpt)
type Middleware func(Tool) Tool

func ApplyMiddleware(t Tool, mws ...Middleware) Tool {
    for i := len(mws) - 1; i >= 0; i-- {
        t = mws[i](t)
    }
    return t
}
```

Applied in reverse so the **first** middleware in the slice wraps the outermost layer. Reading left to right gives you the call order:

```go
wrapped := tool.ApplyMiddleware(base,
    tool.WithTracing(),               // runs first
    tool.WithTimeout(30*time.Second), // runs second
    tool.WithRetry(3),                // runs third
)
// base.Execute() runs last
```

A concrete middleware (retry), from `tool/middleware.go`:

```go
// tool/middleware.go (excerpt)
func WithRetry(maxAttempts int) Middleware {
    return func(t Tool) Tool {
        return &retryTool{tool: t, maxAttempts: maxAttempts}
    }
}

type retryTool struct {
    tool        Tool
    maxAttempts int
}

func (r *retryTool) Name() string                { return r.tool.Name() }
func (r *retryTool) Description() string         { return r.tool.Description() }
func (r *retryTool) InputSchema() map[string]any { return r.tool.InputSchema() }

func (r *retryTool) Execute(ctx context.Context, input map[string]any) (*Result, error) {
    var lastErr error
    for attempt := 0; attempt < r.maxAttempts; attempt++ {
        result, err := r.tool.Execute(ctx, input)
        if err == nil {
            return result, nil
        }
        lastErr = err
        if !core.IsRetryable(err) {
            return nil, err
        }
        if ctx.Err() != nil {
            return nil, ctx.Err()
        }
    }
    return nil, core.Errorf(core.ErrToolFailed, "tool %s failed after %d attempts: %w", r.tool.Name(), r.maxAttempts, lastErr)
}
```

This retry has no backoff between attempts. For backoff with jitter, wrap the call in `resilience.Retry` instead.

## Where it's used

| Package | Middleware shipped |
|---|---|
| `llm` | `WithTracing`, `WithHooks`, `WithLogging`, `WithFallback`, `WithProviderLimits` |
| `tool` | `WithTracing`, `WithTimeout`, `WithRetry` |
| `memory` | `WithTracing`, `WithHooks` |
| `rag/retriever` | `WithTracing`, `WithHooks` |
| `agent` | `WithTracing` |

Every extensible package ships `WithTracing()`. Retry, circuit breaking and hedging for `llm` calls live in `resilience/` as generic helpers rather than as middleware.

## Common mistakes

- **Applying middleware in the wrong order.** Outside-in means the first argument runs first. Reading the slice and picturing the onion layers helps.
- **Losing metadata in wrappers.** A middleware must delegate every non-augmented method (`Name`, `Description`, `InputSchema`) to the inner implementation. Forgetting to do this breaks introspection.
- **Non-idempotent middleware on retries.** If your middleware has side effects (write a row, send an email), and retry is above it, the side effect fires once per attempt. Put non-idempotent logic below retry, or make it idempotent.
- **Blocking in middleware without checking `ctx.Done()`.** A rate-limited call can wait forever if the context is cancelled. Always respect cancellation.
- **Swapping middleware at runtime.** Middleware is a build-time composition. If you need dynamic behaviour, that's a hook, not middleware.

## Example: implementing your own

A simple timing middleware that logs how long each `Tool.Execute` call takes, written in your own package:

```go
// mymw/timing.go
package mymw

import (
    "context"
    "log/slog"
    "time"

    "github.com/lookatitude/beluga-ai/v2/tool"
)

// WithTiming logs the duration of every Execute call.
func WithTiming(logger *slog.Logger) tool.Middleware {
    return func(inner tool.Tool) tool.Tool {
        return &timingTool{inner: inner, logger: logger}
    }
}

type timingTool struct {
    inner  tool.Tool
    logger *slog.Logger
}

func (t *timingTool) Name() string                { return t.inner.Name() }
func (t *timingTool) Description() string         { return t.inner.Description() }
func (t *timingTool) InputSchema() map[string]any { return t.inner.InputSchema() }

func (t *timingTool) Execute(ctx context.Context, input map[string]any) (*tool.Result, error) {
    start := time.Now()
    out, err := t.inner.Execute(ctx, input)
    t.logger.InfoContext(ctx, "tool executed",
        slog.String("tool", t.inner.Name()),
        slog.Duration("elapsed", time.Since(start)),
        slog.Bool("success", err == nil))
    return out, err
}
```

Usage:

```go
wrapped := tool.ApplyMiddleware(base,
    tool.WithTracing(),
    mymw.WithTiming(logger), // new middleware
    tool.WithRetry(3),
)
```

Note the delegation: `Name()`, `Description()`, and `InputSchema()` must pass through untouched. Only `Execute` is augmented.

## Related

- [03 — Extensibility Patterns](../architecture/03-extensibility-patterns.md#ring-4--middleware)
- [`patterns/hooks-lifecycle.md`](./hooks-lifecycle.md) — when to use hooks instead.
- [`.guild/wiki/patterns/middleware.md`](../../.guild/wiki/patterns/middleware.md) — canonical code references.
- [15 — Resilience](../architecture/15-resilience.md) — retry, rate limit, circuit breaker details.
