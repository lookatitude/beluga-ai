# Pattern: Middleware

**Status:** stub — populate with `/wiki-learn`

## Contract

Middleware signature: `func(T) T` where T is the interface being wrapped. Applied outside-in: the last middleware added runs first.

```go
type Middleware func(Chat) Chat

func Chain(base Chat, mws ...Middleware) Chat {
    for i := len(mws) - 1; i >= 0; i-- {
        base = mws[i](base)
    }
    return base
}
```

## Canonical example

(populate via `/wiki-learn`)

## Anti-patterns

- Middleware that breaks the interface contract (e.g., swallows errors silently).
- Middleware that captures context in a closure and outlives the request.
- Stateful middleware without documented concurrency guarantees.

## Related

- `patterns/hooks.md` (hooks observe; middleware transforms)
- `patterns/provider-registration.md`
