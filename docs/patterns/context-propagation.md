# Pattern: Context Propagation

## What it is

`context.Context` is the first parameter of every public function and carries **cancellation**, the **tenant ID**, the **session ID**, the **request ID**, and the **OTel trace/span**. Typed helper functions (`core.WithTenant`/`GetTenant`, `core.WithSessionID`/`GetSessionID`, `core.WithRequestID`/`GetRequestID`) prevent key collisions between packages. Agent credentials travel the same way via `auth/credential.WithCredential`.

## Why we use it

Multi-tenancy, session isolation, tracing, and cancellation are all **per-request** concerns that need to reach every layer of the stack. You could pass them as explicit function arguments, but that pollutes every signature and breaks every time a new piece of context is added. Go's `context.Context` is the idiomatic carrier.

**Alternatives considered:**
- **Explicit struct per call.** Works but every signature grows every time you add a field.
- **Global variables.** Breaks multi-tenancy catastrophically.
- **Thread-locals (not in Go).** Not possible without goroutine-local storage hacks. Even if it were, per-goroutine state is fragile.
- **Function-colored arguments.** Some languages use function coloring (`async`, `pure`). Go doesn't have that.

`context.Context` is the unique answer. It propagates through every call without changing any signature, supports cancellation trees, and carries request-scoped metadata.

## How it works

From `core/tenant.go` and `core/context.go`:

```go
// core/tenant.go (excerpt)
type tenantKey struct{}

// TenantID uniquely identifies a tenant in a multi-tenant deployment.
type TenantID string

func WithTenant(ctx context.Context, id TenantID) context.Context {
    return context.WithValue(ctx, tenantKey{}, id)
}

func GetTenant(ctx context.Context) TenantID {
    id, _ := ctx.Value(tenantKey{}).(TenantID)
    return id
}

// core/context.go: WithSessionID/GetSessionID and WithRequestID/GetRequestID
// follow the same shape, keyed by an unexported contextKey type.
```

Key collision prevention: the context key is an unexported type (`tenantKey{}`). No other package can forge a colliding key without importing the unexported type.

Usage across a call chain:

```go
// handler.go
func handleChat(a agent.Agent) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        ctx = core.WithTenant(ctx, core.TenantID(r.Header.Get("X-Tenant-ID")))
        ctx = core.WithSessionID(ctx, r.Header.Get("X-Session-ID"))

        // every downstream call sees the tenant and session
        answer, err := a.Invoke(ctx, r.FormValue("q"))
        // ...
    }
}

// a ChatModel implementation
func (m *Model) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
    tenant := core.GetTenant(ctx)
    if tenant == "" {
        return nil, core.Errorf(core.ErrAuth, "myprovider: no tenant in context")
    }
    // use tenant for per-tenant rate limit, cost tracking, logging
    // ...
}

// a store
func (s *Store) Append(ctx context.Context, msg schema.Message) error {
    key := fmt.Sprintf("%s:messages:%s", core.GetTenant(ctx), core.GetSessionID(ctx))
    return s.client.RPush(ctx, key, serialize(msg)).Err()
}
```

The store scopes by tenant because it reads the tenant from the context. Adding a new store? Read `core.GetTenant(ctx)`.

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
func (m *Model) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
    ctx, span := tracer.Start(ctx, "llm.generate",
        trace.WithAttributes(
            attribute.String("gen_ai.system", "myprovider"),
            attribute.String("gen_ai.request.model", m.model),
        ))
    defer span.End()

    resp, err := m.callAPI(ctx, msgs) // downstream calls see the span via ctx
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
    }
    return resp, err
}
```

The `ctx` returned by `tracer.Start` has the new span attached. Pass it downstream. Downstream `tracer.Start` calls create child spans automatically. In practice, `llm.WithTracing()` adds this span for you. See [DOC-14](../architecture/14-observability.md).

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

`core` already ships `WithRequestID`/`GetRequestID`. Here is the same shape for a value of your own, a feature-flag set:

```go
package flags

import "context"

type flagsKey struct{}

// Set is an immutable set of enabled feature flags.
type Set map[string]bool

// With attaches a flag set to the context.
func With(ctx context.Context, s Set) context.Context {
    return context.WithValue(ctx, flagsKey{}, s)
}

// Enabled reports whether flag is on in ctx's flag set.
func Enabled(ctx context.Context, flag string) bool {
    s, _ := ctx.Value(flagsKey{}).(Set)
    return s[flag]
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
        ctx := core.WithRequestID(r.Context(), id)
        ctx = flags.With(ctx, flags.Set{"new-planner": r.Header.Get("X-Beta") == "1"})
        w.Header().Set("X-Request-ID", id)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// anywhere downstream
func (t *myTool) Execute(ctx context.Context, in map[string]any) (*tool.Result, error) {
    slog.InfoContext(ctx, "tool start",
        slog.String("request_id", core.GetRequestID(ctx)),
        slog.Bool("new_planner", flags.Enabled(ctx, "new-planner")))
    // ...
}
```

Every layer in the stack sees the request ID and the flags through the same context. No signature changes, no global state.

## Related

- [02 — Core Primitives](../architecture/02-core-primitives.md#context--the-invisible-argument)
- [13 — Security Model](../architecture/13-security-model.md#multi-tenancy-isolation)
- [14 — Observability](../architecture/14-observability.md) — how OTel uses the context.
- [Go blog: Context](https://go.dev/blog/context) — the canonical reference.
