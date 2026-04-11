# Pattern: Error Handling

**Status:** stub — populate with `/wiki-learn`

## Contract

- Return `(T, error)`. Never panic for recoverable errors.
- Typed errors via `core.Error` with `ErrorCode`.
- Wrap with `%w` to preserve the chain.
- Check `IsRetryable()` before retrying LLM or tool errors.
- Never expose internal details to external callers.

```go
return core.Errorf(core.ErrCodeRateLimit, "provider %s rate limited: %w", name, err)
```

## Canonical example

(populate via `/wiki-learn` — scan for `core.Errorf` in `core/errors.go`)

## Anti-patterns

- Raw `errors.New("...")` for conditions that need a code.
- Swallowing errors silently (`_ = doThing()`).
- Exposing stack traces, SQL, or file paths in user-facing errors.

## Related

- `patterns/security.md#error-handling-information-disclosure`
