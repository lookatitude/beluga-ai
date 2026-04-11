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

Canonical code from `core/errors.go:8-110` (see [`.wiki/patterns/error-handling.md`](../../.wiki/patterns/error-handling.md)):

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
    t, ok := target.(*Error)
    if !ok {
        return false
    }
    return e.Code == t.Code
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
func NewError(op string, code ErrorCode, message string, cause error) *Error {
    return &Error{Op: op, Code: code, Message: message, Err: cause}
}

// Errorf is the fmt.Errorf equivalent for core.Error.
func Errorf(code ErrorCode, format string, args ...any) *Error {
    msg := fmt.Sprintf(format, args...)
    return &Error{Code: code, Message: msg}
}
```

Usage in a provider:

```go
func (p *openaiProvider) Generate(ctx context.Context, req llm.Request) (*llm.Response, error) {
    resp, err := p.client.Chat(ctx, req)
    if err != nil {
        // translate provider errors to core codes
        if apiErr, ok := err.(*openai.APIError); ok {
            switch apiErr.Code {
            case "rate_limit_exceeded":
                return nil, core.Errorf(core.ErrRateLimit, "openai: %w", err)
            case "invalid_api_key":
                return nil, core.Errorf(core.ErrAuth, "openai: %w", err)
            case "context_length_exceeded":
                return nil, core.Errorf(core.ErrInvalidInput, "openai: %w", err)
            default:
                return nil, core.Errorf(core.ErrProviderDown, "openai: %w", err)
            }
        }
        return nil, core.Errorf(core.ErrProviderDown, "openai: %w", err)
    }
    return translateResponse(resp), nil
}
```

Middleware consumer:

```go
func (r *retryLLM) Generate(ctx context.Context, req llm.Request) (*llm.Response, error) {
    for attempt := 0; attempt < r.maxAttempts; attempt++ {
        out, err := r.inner.Generate(ctx, req)
        if err == nil || !core.IsRetryable(err) {
            return out, err // success or non-retryable failure
        }
        // retry with backoff
    }
    return nil, lastErr
}
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

Wrapping a third-party library's errors in a new tool:

```go
package weather

import (
    "context"
    "errors"

    "github.com/lookatitude/beluga-ai/core"
    "github.com/lookatitude/beluga-ai/tool"
)

func (t *weatherTool) Execute(ctx context.Context, input map[string]any) (*tool.Result, error) {
    city, _ := input["city"].(string)
    if city == "" {
        return nil, core.Errorf(core.ErrInvalidInput, "weather: city is required")
    }

    data, err := t.api.Forecast(ctx, city)
    if err != nil {
        var apiErr *weatherapi.Error
        if errors.As(err, &apiErr) {
            switch apiErr.Kind {
            case weatherapi.KindRateLimited:
                return nil, core.Errorf(core.ErrRateLimit, "weather: rate limit for %s", city)
            case weatherapi.KindUnauthorized:
                return nil, core.Errorf(core.ErrAuth, "weather: api key invalid")
            case weatherapi.KindNotFound:
                return nil, core.Errorf(core.ErrNotFound, "weather: city %q not found", city)
            }
        }
        return nil, core.Errorf(core.ErrToolFailed, "weather: %w", err)
    }
    return tool.TextResult(data.Summary()), nil
}
```

When the retry middleware sees `ErrRateLimit`, it retries. When it sees `ErrAuth`, it fails fast. When it sees `ErrNotFound`, it fails fast (the city doesn't exist; retrying won't help). When it sees `ErrToolFailed`, it fails fast (the planner decides whether to try a different tool).

## Related

- [`patterns/middleware-chain.md`](./middleware-chain.md) — how retry middleware consumes these.
- [15 — Resilience](../architecture/15-resilience.md) — retry, circuit breaker, rate limiter.
- [`.wiki/patterns/error-handling.md`](../../.wiki/patterns/error-handling.md) — canonical code references.
- [`.wiki/architecture/invariants.md`](../../.wiki/architecture/invariants.md) — the retry invariant.
