---
name: streaming-patterns
description: iter.Seq2 streaming patterns for Beluga AI v2 (Go 1.26+). Use when implementing streaming, transforms, or backpressure.
---

# Streaming Patterns

## Primary Primitive: iter.Seq2[T, error]

```go
func (m *Model) Stream(ctx context.Context, msgs []schema.Message) iter.Seq2[schema.StreamChunk, error] {
    return func(yield func(schema.StreamChunk, error) bool) {
        stream, err := m.client.Stream(ctx, msgs)
        if err != nil { yield(schema.StreamChunk{}, err); return }
        defer stream.Close()
        for {
            select {
            case <-ctx.Done(): yield(schema.StreamChunk{}, ctx.Err()); return
            default:
            }
            chunk, err := stream.Recv()
            if err == io.EOF { return }
            if err != nil { yield(schema.StreamChunk{}, err); return }
            if !yield(convertChunk(chunk), nil) { return } // consumer stopped
        }
    }
}
```

## Composition

There are no generic `Pipe`/`Collect` helpers over plain `iter.Seq2[T, error]`. What exists:

- **`core.Stream[T]`** = `iter.Seq2[core.Event[T], error]`, with `core.CollectStream`, `core.MapStream`, `core.FilterStream`, `core.MergeStreams(ctx, ...)`, `core.FanOut(ctx, s, n)` and `core.NewBufferedStream(ctx, s, size)` (`core/stream.go`).
- **`core.Pipe(a, b core.Runnable)`** and **`core.Parallel(rs ...core.Runnable)`** compose `Runnable`s, not streams (`core/runnable.go`).
- **Invoke from Stream**: range the stream, accumulate, return the result (see `agent.BaseAgent.Invoke`).
- **Fan-out**: `core.FanOut`; use `iter.Pull2()` only when you need pull semantics.

## Rules

1. Public API: `iter.Seq2[T, error]` — never `<-chan`.
2. Internal goroutine communication: channels are fine.
3. Always check context cancellation in producers.
4. `yield` returning false = consumer stopped — respect immediately.
5. Use `iter.Pull2` only when pull semantics are genuinely needed.
