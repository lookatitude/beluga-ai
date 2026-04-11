# Pattern: Streaming with iter.Seq2

**Status:** stub — populate with `/wiki-learn`

## Contract

Public streaming APIs return `iter.Seq2[T, error]`. Consumers use `for event, err := range stream { if err != nil { break } }`. Producers respect `context.Context` cancellation.

```go
func (c *Client) Stream(ctx context.Context, req Request) iter.Seq2[Event, error] {
    return func(yield func(Event, error) bool) {
        for {
            select {
            case <-ctx.Done():
                yield(Event{}, ctx.Err())
                return
            default:
            }
            ev, err := c.next()
            if !yield(ev, err) { return }
            if err != nil { return }
        }
    }
}
```

## Canonical example

(populate via `/wiki-learn` — scan for `iter.Seq2` in `core/stream.go`)

## Anti-patterns

- Channels in public APIs.
- Producers that ignore `ctx.Done()`.
- Yielding zero values without paired errors on termination.

## Related

- `architecture/invariants.md#1-streaming-uses-iterseq2ttypeerror--never-channels`
- `patterns/testing.md` (stream testing)
