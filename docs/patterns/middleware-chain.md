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

Canonical code from `tool/middleware.go:11-22` (see [`.wiki/patterns/middleware.md`](../../.wiki/patterns/middleware.md)):

```go
// tool/middleware.go
package tool

type Middleware func(Tool) Tool

func ApplyMiddleware(t Tool, mws ...Middleware) Tool {
    result := t
    for i := len(mws) - 1; i >= 0; i-- {
        result = mws[i](result)
    }
    return result
}
```

Applied in reverse so the **first** middleware in the slice wraps the outermost layer. Reading left to right gives you the call order:

```go
wrapped := tool.ApplyMiddleware(base,
    tool.WithGuardrails(guards),   // runs first
    tool.WithLogging(logger),      // runs second
    tool.WithRateLimit(rl),        // runs third
    tool.WithRetry(3),             // runs fourth
)
// base.Execute() runs last
```

A concrete middleware (retry):

```go
// tool/middleware.go — conceptual
func WithRetry(maxAttempts int) Middleware {
    return func(inner Tool) Tool {
        return &retryTool{inner: inner, maxAttempts: maxAttempts}
    }
}

type retryTool struct {
    inner       Tool
    maxAttempts int
}

func (r *retryTool) Name() string             { return r.inner.Name() }
func (r *retryTool) Description() string      { return r.inner.Description() }
func (r *retryTool) InputSchema() map[string]any { return r.inner.InputSchema() }

func (r *retryTool) Execute(ctx context.Context, input map[string]any) (*Result, error) {
    var lastErr error
    for attempt := 0; attempt < r.maxAttempts; attempt++ {
        if err := ctx.Err(); err != nil {
            return nil, err
        }
        out, err := r.inner.Execute(ctx, input)
        if err == nil || !core.IsRetryable(err) {
            return out, err
        }
        lastErr = err
        backoffWithJitter(attempt)
    }
    return nil, lastErr
}
```

## Where it's used

| Package | Middleware types |
|---|---|
| `llm` | retry, rate limit, circuit breaker, logging, metrics, guardrails, cache |
| `tool` | retry, timeout, logging, guardrails, HITL |
| `memory` | encryption, compression, caching |
| `rag/retriever` | reranking, filtering, caching |
| `agent` | session-scoping, plugin hooks |

## Common mistakes

- **Applying middleware in the wrong order.** Outside-in means the first argument runs first. Reading the slice and picturing the onion layers helps.
- **Losing metadata in wrappers.** A middleware must delegate every non-augmented method (`Name`, `Description`, `InputSchema`) to the inner implementation. Forgetting to do this breaks introspection.
- **Non-idempotent middleware on retries.** If your middleware has side effects (write a row, send an email), and retry is above it, the side effect fires once per attempt. Put non-idempotent logic below retry, or make it idempotent.
- **Blocking in middleware without checking `ctx.Done()`.** A rate-limited call can wait forever if the context is cancelled. Always respect cancellation.
- **Swapping middleware at runtime.** Middleware is a build-time composition. If you need dynamic behaviour, that's a hook, not middleware.

## Example: implementing your own

A simple timing middleware that logs how long each `Tool.Execute` call takes:

```go
// tool/middleware_timing.go
package tool

import (
    "context"
    "log/slog"
    "time"
)

func WithTiming(logger *slog.Logger) Middleware {
    return func(inner Tool) Tool {
        return &timingTool{inner: inner, logger: logger}
    }
}

type timingTool struct {
    inner  Tool
    logger *slog.Logger
}

func (t *timingTool) Name() string             { return t.inner.Name() }
func (t *timingTool) Description() string      { return t.inner.Description() }
func (t *timingTool) InputSchema() map[string]any { return t.inner.InputSchema() }

func (t *timingTool) Execute(ctx context.Context, input map[string]any) (*Result, error) {
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
    tool.WithGuardrails(guards),
    tool.WithTiming(logger),     // new middleware
    tool.WithRetry(3),
)
```

Note the delegation: `Name()`, `Description()`, and `InputSchema()` must pass through untouched. Only `Execute` is augmented.

## Related

- [03 — Extensibility Patterns](../architecture/03-extensibility-patterns.md#ring-4--middleware)
- [`patterns/hooks-lifecycle.md`](./hooks-lifecycle.md) — when to use hooks instead.
- [`.wiki/patterns/middleware.md`](../../.wiki/patterns/middleware.md) — canonical code references.
- [15 — Resilience](../architecture/15-resilience.md) — retry, rate limit, circuit breaker details.
