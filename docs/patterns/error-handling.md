# Pattern: Error Handling

## What it is

Beluga errors use a typed `core.Error` value with an `ErrorCode` enum. `core.IsRetryable(err)` returns true only for errors whose code is in the retryable set (`ErrRateLimit`, `ErrTimeout`, `ErrProviderDown`). Middleware reads the code to decide whether to retry, circuit-break, or fail fast.

## Why we use it

You need programmatic distinctions between error kinds: a rate limit should be retried, an auth failure should not. If errors are `errors.New("...")`, the only way to check is string matching — fragile, untested, and locale-dependent.

**Alternatives considered:**
- **Sentinel errors.** `var ErrRateLimit = errors.New("rate limit")`. Works for a few kinds but scales poorly. No structured fields.
- **Panicking.** Forces callers to `recover`. Not Go-idiomatic for recoverable errors.
- **Error codes as integers.** Less readable than enums. Harder to search the codebase for all uses.

Typed errors with an `ErrorCode` enum are structured, programmatic, and align with how Google's error packages and the `errors.Is/As` machinery work.

## How it works

Canonical code from `core/errors.go` (doc comments trimmed) (see [`.guild/wiki/patterns/error-handling.md`](../../.guild/wiki/patterns/error-handling.md)):

```go
// core/errors.go
package core

import (
    "errors"
    "fmt"
)

type ErrorCode string

const (
    ErrRateLimit       ErrorCode = "rate_limit"
    ErrAuth            ErrorCode = "auth_error"
    ErrTimeout         ErrorCode = "timeout"
    ErrInvalidInput    ErrorCode = "invalid_input"
    ErrToolFailed      ErrorCode = "tool_failed"
    ErrProviderDown    ErrorCode = "provider_unavailable"
    ErrGuardBlocked    ErrorCode = "guard_blocked"
    ErrBudgetExhausted ErrorCode = "budget_exhausted"
    ErrNotFound        ErrorCode = "not_found"
)

var retryableCodes = map[ErrorCode]bool{
    ErrRateLimit:    true,
    ErrTimeout:      true,
    ErrProviderDown: true,
}

type Error struct {
    Op      string
    Code    ErrorCode
    Message string
    Err     error
}

func (e *Error) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("%s [%s]: %s: %v", e.Op, e.Code, e.Message, e.Err)
    }
    return fmt.Sprintf("%s [%s]: %s", e.Op, e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.Err }

func (e *Error) Is(target error) bool {
    var t *Error
    if errors.As(target, &t) {
        return e.Code == t.Code
    }
    return false
}

// IsRetryable returns true if err is a core.Error with a retryable code.
func IsRetryable(err error) bool {
    var e *Error
    if errors.As(err, &e) {
        return retryableCodes[e.Code]
    }
    return false
}

// NewError constructs a core.Error.
func NewError(op string, code ErrorCode, msg string, cause error) *Error {
    return &Error{Op: op, Code: code, Message: msg, Err: cause}
}

// Errorf is the fmt.Errorf equivalent for core.Error. A %w argument is kept
// as the cause, so errors.Is, errors.As and IsRetryable see through it.
func Errorf(code ErrorCode, format string, args ...any) *Error {
    inner := fmt.Errorf(format, args...)
    return &Error{Code: code, Message: inner.Error(), Err: errors.Unwrap(inner)}
}
```

Usage in a provider (simplified from an HTTP-based `llm.ChatModel`):

```go
func (m *Model) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
    resp, err := m.do(ctx, msgs, opts...)
    if err != nil {
        return nil, core.Errorf(core.ErrProviderDown, "myprovider: %w", err)
    }
    defer resp.Body.Close()

    // translate provider errors to core codes
    switch {
    case resp.StatusCode == http.StatusTooManyRequests:
        return nil, core.Errorf(core.ErrRateLimit, "myprovider: rate limited")
    case resp.StatusCode == http.StatusUnauthorized:
        return nil, core.Errorf(core.ErrAuth, "myprovider: invalid API key")
    case resp.StatusCode == http.StatusBadRequest:
        return nil, core.Errorf(core.ErrInvalidInput, "myprovider: bad request")
    case resp.StatusCode >= 500:
        return nil, core.Errorf(core.ErrProviderDown, "myprovider: status %d", resp.StatusCode)
    }
    return decodeResponse(resp.Body)
}
```

Consumer: `resilience.Retry` retries only while `core.IsRetryable` says so.

```go
resp, err := resilience.Retry(ctx, resilience.DefaultRetryPolicy(),
    func(ctx context.Context) (*schema.AIMessage, error) {
        return model.Generate(ctx, msgs)
    })
// ErrRateLimit, ErrTimeout, ErrProviderDown: retried with backoff.
// Anything else: returned after the first attempt.
```

## Where it's used

Every provider, every middleware, every layer that can fail. In particular:

- **LLM providers** — translate API errors to codes.
- **Tool implementations** — `ErrToolFailed` for failures that should feed back to the planner, `ErrTimeout` for retry-worthy.
- **Memory stores** — `ErrNotFound`, `ErrProviderDown`.
- **Guard pipeline** — `ErrGuardBlocked` (non-retryable — reviews and denials should not be retried).
- **Cost plugin** — `ErrBudgetExhausted` (non-retryable — wait for budget refresh, not retry).

## Which errors are retryable

Only three codes:

- `ErrRateLimit` — the rate window will eventually reopen.
- `ErrTimeout` — the next attempt might be faster.
- `ErrProviderDown` — the provider might come back.

Everything else is **not retryable**: auth won't self-fix, invalid input won't self-correct, a blocked guard won't unblock, a budget won't refill. Retrying these wastes time and money.

This list is deliberately small — adding more retryable codes increases the risk of looping on permanent failures.

## Common mistakes

- **Retrying on `ErrAuth`.** Auth failures don't fix themselves. Check `core.IsRetryable` and respect it.
- **Using `errors.New` in new code.** Produces an untyped error — middleware can't decide what to do with it. Use `core.Errorf` or `core.NewError`.
- **String-matching error messages.** Non-deterministic (localisation, API changes). Use `errors.Is` or `errors.As` with the typed error.
- **Swallowing errors silently.** At minimum, log them. Better: propagate with context via `%w` wrapping.
- **Leaking internal details in errors.** Error messages returned to external callers should not contain stack traces, file paths, or SQL. Strip before returning from handlers.

## Example: implementing your own

Wrapping a third-party library's errors in a new tool. `Client` and `APIError` stand in for the vendor SDK:

```go
package weather

import (
    "context"
    "errors"

    "github.com/lookatitude/beluga-ai/v2/core"
    "github.com/lookatitude/beluga-ai/v2/tool"
)

// Kind classifies vendor API failures.
type Kind int

const (
    KindOther Kind = iota
    KindRateLimited
    KindUnauthorized
    KindNotFound
)

// APIError is the error type returned by the vendor SDK.
type APIError struct{ Kind Kind }

func (e *APIError) Error() string { return "weather api error" }

// Client is the subset of the vendor SDK the tool needs.
type Client interface {
    Forecast(ctx context.Context, city string) (string, error)
}

// Tool exposes the forecast API to agents.
type Tool struct{ api Client }

var _ tool.Tool = (*Tool)(nil)

func (t *Tool) Name() string        { return "weather" }
func (t *Tool) Description() string { return "Get the weather forecast for a city" }
func (t *Tool) InputSchema() map[string]any {
    return map[string]any{
        "type":       "object",
        "properties": map[string]any{"city": map[string]any{"type": "string"}},
        "required":   []string{"city"},
    }
}

func (t *Tool) Execute(ctx context.Context, input map[string]any) (*tool.Result, error) {
    city, _ := input["city"].(string)
    if city == "" {
        return nil, core.Errorf(core.ErrInvalidInput, "weather: city is required")
    }

    summary, err := t.api.Forecast(ctx, city)
    if err != nil {
        var apiErr *APIError
        if errors.As(err, &apiErr) {
            switch apiErr.Kind {
            case KindRateLimited:
                return nil, core.Errorf(core.ErrRateLimit, "weather: rate limit for %s", city)
            case KindUnauthorized:
                return nil, core.Errorf(core.ErrAuth, "weather: api key invalid")
            case KindNotFound:
                return nil, core.Errorf(core.ErrNotFound, "weather: city %q not found", city)
            }
        }
        return nil, core.Errorf(core.ErrToolFailed, "weather: %w", err)
    }
    return tool.TextResult(summary), nil
}
```

When `tool.WithRetry` or `resilience.Retry` sees `ErrRateLimit`, it retries. When it sees `ErrAuth`, it fails fast. When it sees `ErrNotFound`, it fails fast (the city doesn't exist; retrying won't help). When it sees `ErrToolFailed`, it fails fast (the planner decides whether to try a different tool).

## Related

- [`patterns/middleware-chain.md`](./middleware-chain.md) — how retry middleware consumes these.
- [15 — Resilience](../architecture/15-resilience.md) — retry, circuit breaker, rate limiter.
- [`.guild/wiki/patterns/error-handling.md`](../../.guild/wiki/patterns/error-handling.md) — canonical code references.
- [`.guild/wiki/architecture/invariants.md`](../../.guild/wiki/architecture/invariants.md) — the retry invariant.
