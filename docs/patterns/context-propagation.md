# Pattern: Context Propagation

## What it is

`context.Context` is the first parameter of every public function and carries five things: **cancellation**, **tenant ID**, **session ID**, **auth claims**, and **OTel trace/span**. Typed helper functions (`WithTenant`, `GetTenant`, …) prevent key collisions between packages.

## Why we use it

Multi-tenancy, session isolation, tracing, and cancellation are all **per-request** concerns that need to reach every layer of the stack. You could pass them as explicit function arguments, but that pollutes every signature and breaks every time a new piece of context is added. Go's `context.Context` is the idiomatic carrier.

**Alternatives considered:**
- **Explicit struct per call.** Works but every signature grows every time you add a field.
- **Global variables.** Breaks multi-tenancy catastrophically.
- **Thread-locals (not in Go).** Not possible without goroutine-local storage hacks. Even if it were, per-goroutine state is fragile.
- **Function-colored arguments.** Some languages use function coloring (`async`, `pure`). Go doesn't have that.

`context.Context` is the unique answer. It propagates through every call without changing any signature, supports cancellation trees, and carries request-scoped metadata.

## How it works

```go
// core/context.go — conceptual
package core

import "context"

type tenantKey struct{}
type sessionKey struct{}
type authKey struct{}

// WithTenant attaches a tenant ID to the context.
func WithTenant(ctx context.Context, tenant string) context.Context {
    return context.WithValue(ctx, tenantKey{}, tenant)
}

// GetTenant returns the tenant ID from context, or "" if absent.
func GetTenant(ctx context.Context) string {
    if v, ok := ctx.Value(tenantKey{}).(string); ok {
        return v
    }
    return ""
}

// WithSession, GetSession, WithAuth, GetAuth follow the same shape.
```

Key collision prevention: the context key is an unexported struct type (`tenantKey{}`). No other package can forge a colliding key without importing the unexported type.

Usage across a call chain:

```go
// handler.go
func handleChat(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    ctx = core.WithTenant(ctx, extractTenant(r))
    ctx = core.WithSession(ctx, extractSession(r))
    ctx = core.WithAuth(ctx, extractAuth(r))

    // every downstream call sees the tenant/session/auth
    resp, err := agent.Invoke(ctx, r.Body)
    // ...
}

// llm/providers/openai.go
func (p *Provider) Generate(ctx context.Context, req Request) (*Response, error) {
    tenant := core.GetTenant(ctx)
    if tenant == "" {
        return nil, core.Errorf(core.ErrAuth, "openai: no tenant in context")
    }
    // use tenant for per-tenant rate limit, cost tracking, logging
    ...
}

// memory/stores/redis.go
func (s *Store) Save(ctx context.Context, msg schema.Message) error {
    tenant := core.GetTenant(ctx)
    key := fmt.Sprintf("%s:messages:%s", tenant, core.GetSession(ctx))
    return s.client.RPush(ctx, key, serialize(msg)).Err()
}
```

The Redis store automatically scopes by tenant because it reads the tenant from the context. Adding a new store? Just read `core.GetTenant(ctx)`.

## Cancellation

`context.Context` is also how cancellation propagates. If the caller's context is cancelled, every downstream function sees `ctx.Done()` fire:

```go
func (t *httpTool) Execute(ctx context.Context, input map[string]any) (*Result, error) {
    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil) // cancellation-aware
    resp, err := t.client.Do(req)
    if err != nil {
        if ctx.Err() != nil {
            return nil, ctx.Err() // cancellation beats the generic error
        }
        return nil, core.Errorf(core.ErrToolFailed, "http: %w", err)
    }
    // ...
}
```

`http.NewRequestWithContext` is the correct pattern. `http.NewRequest` without a context is legacy and doesn't propagate cancellation.

## OTel trace/span

OpenTelemetry uses `context.Context` to thread trace and span IDs:

```go
func (p *Provider) Generate(ctx context.Context, req Request) (*Response, error) {
    ctx, span := tracer.Start(ctx, "llm.generate",
        trace.WithAttributes(
            attribute.String("gen_ai.system", "openai"),
            attribute.String("gen_ai.request.model", req.Model),
        ))
    defer span.End()

    resp, err := p.callAPI(ctx, req) // downstream calls see the span via ctx
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
    }
    return resp, err
}
```

The `ctx` returned by `tracer.Start` has the new span attached. Pass it downstream. Downstream `tracer.Start` calls create child spans automatically. See [DOC-14](../architecture/14-observability.md).

## Where it's used

Every public function in Beluga. It's a hard rule:

- `context.Context` is the first parameter.
- It is never stored in a struct field.
- It is never passed as nil (use `context.Background()` in tests).

## Common mistakes

- **Storing ctx in a struct.** Breaks cancellation and tenant isolation. Pass it per call.
- **Using `context.TODO()` in production code.** `TODO` is for migration — code that hasn't been plumbed with a real context yet. Replace before shipping.
- **Not reading the tenant in new stores.** Every new store must honour `core.GetTenant(ctx)` or you have cross-tenant data leakage.
- **Passing `nil` as ctx.** Always use `context.Background()` or derive from an explicit parent. `nil` causes panics in half the stdlib.
- **Creating keys as exported strings.** `ctx.Value("tenant")` can be forged by any other package. Use unexported struct keys.
- **Forgetting to propagate ctx to a goroutine.** If you `go doWork()`, the new goroutine has no context. Pass it: `go doWork(ctx)`.

## Example: implementing your own

Adding a `RequestID` to the context that propagates through all layers:

```go
package mycore

import "context"

type requestIDKey struct{}

// WithRequestID attaches a request ID to the context.
func WithRequestID(ctx context.Context, id string) context.Context {
    return context.WithValue(ctx, requestIDKey{}, id)
}

// GetRequestID returns the request ID from the context, or "" if absent.
func GetRequestID(ctx context.Context) string {
    if v, ok := ctx.Value(requestIDKey{}).(string); ok {
        return v
    }
    return ""
}
```

Usage:

```go
// middleware.go
func requestIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        id := r.Header.Get("X-Request-ID")
        if id == "" {
            id = uuid.NewString()
        }
        ctx := mycore.WithRequestID(r.Context(), id)
        w.Header().Set("X-Request-ID", id)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// anywhere downstream
func (t *tool) Execute(ctx context.Context, in map[string]any) (*Result, error) {
    rid := mycore.GetRequestID(ctx)
    slog.InfoContext(ctx, "tool start", slog.String("request_id", rid))
    // ...
}
```

Every layer in the stack sees the request ID through the same context. No signature changes, no global state.

## Related

- [02 — Core Primitives](../architecture/02-core-primitives.md#context--the-invisible-argument)
- [13 — Security Model](../architecture/13-security-model.md#multi-tenancy-isolation)
- [14 — Observability](../architecture/14-observability.md) — how OTel uses the context.
- [Go blog: Context](https://go.dev/blog/context) — the canonical reference.
