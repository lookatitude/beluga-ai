# Pattern: Streaming with iter.Seq2

## What it is

Beluga streams use `iter.Seq2[T, error]` (added in Go 1.23), a pull-based iterator that integrates with the `for … range` syntax and propagates backpressure. Public streaming APIs return a plain `iter.Seq2`: `llm.ChatModel.Stream` returns `iter.Seq2[schema.StreamChunk, error]`, `agent.Agent.Stream` returns `iter.Seq2[agent.Event, error]`. `core.Stream[T]` is a type alias for `iter.Seq2[core.Event[T], error]` used by the generic stream helpers. Channels never appear in public API boundaries.

## Why we use it

LLM tokens, tool observations, voice frames, retrieval results — all of these are naturally iterative streams. You need backpressure, cancellation, and composition. Channels give you this too, but at the cost of goroutine management and a heavier consumer syntax.

**Alternatives considered:**
- **Channels.** Require a goroutine per stage, leak goroutines if the consumer returns early, and need `select` for multi-stream coordination. Rejected for public APIs.
- **Callback/visitor pattern.** Inverts control — the producer drives. Consumers lose the ability to pause or cancel naturally. Rejected.
- **Buffered slices.** Trivial but loses streaming (the consumer waits for the whole result before seeing any of it). Rejected for anything interactive.

`iter.Seq2` is pull-based (the consumer drives), type-safe (generic), cancellation-aware (backpressure via the `yield` protocol), and uses a syntax every Go programmer already knows. It's the right primitive.

## How it works

From `core/stream.go`:

```go
// core/stream.go (excerpt)
type Event[T any] struct {
    Type    EventType // EventData, EventToolCall, EventToolResult, EventHandoff, EventDone, EventError
    Payload T
    Err     error
    Meta    map[string]any
}

// Stream is a pull-based event iterator built on iter.Seq2.
type Stream[T any] = iter.Seq2[Event[T], error]
```

Consumer:

```go
for chunk, err := range model.Stream(ctx, msgs) {
    if err != nil {
        return err // the producer yields at most one error, then stops
    }
    fmt.Print(chunk.Delta)
    if somethingBad {
        break // yield returns false; the producer must stop and clean up
    }
}
```

Producer (an `llm.ChatModel.Stream` over a vendor SDK stream):

```go
func (m *Model) Stream(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
    return func(yield func(schema.StreamChunk, error) bool) {
        stream, err := m.client.ChatStream(ctx, toRequest(msgs, opts))
        if err != nil {
            yield(schema.StreamChunk{}, core.Errorf(core.ErrProviderDown, "myprovider: %w", err))
            return
        }
        defer stream.Close()
        for stream.Next() {
            if !yield(schema.StreamChunk{Delta: stream.Current().Text}, nil) {
                return // consumer quit early; clean up
            }
        }
        if err := stream.Err(); err != nil {
            yield(schema.StreamChunk{}, err)
        }
    }
}
```

The `yield` return value is the backpressure signal: if it returns `false`, the consumer is done and the producer must exit cleanly.

## Stream operations

`core` ships these helpers for `core.Stream[T]`:

| Function | Purpose |
|---|---|
| `core.CollectStream(s)` | Drain into a `[]Event[T]`, returning the first error. |
| `core.MapStream(s, fn)` | Transform each event (`func(Event[T]) (Event[U], error)`). |
| `core.FilterStream(s, keep)` | Drop events for which `keep` returns false. |
| `core.MergeStreams(ctx, s...)` | Fan-in: interleave several streams in arrival order. |
| `core.FanOut(ctx, s, n)` | Broadcast one stream to `n` independent consumers. |
| `core.NewBufferedStream(ctx, s, size)` | Decouple producer and consumer speed with a bounded buffer. |

For composing `core.Runnable` values (not streams), use `core.Pipe(a, b)` and `core.Parallel(rs...)`.

## Where it's used

- `core` — `Event[T]`, `Stream[T]` and the helpers above.
- `llm` — `ChatModel.Stream` returns `iter.Seq2[schema.StreamChunk, error]`.
- `agent` — `Agent.Stream` returns `iter.Seq2[agent.Event, error]`.
- `voice` — `FrameProcessor.Process` takes and returns `iter.Seq2[voice.Frame, error]`; channels are used only inside implementations.
- `workflow` — `WorkflowContext.ReceiveSignal` returns `iter.Seq2[any, error]`.
- `core.Runnable.Stream` returns `iter.Seq2[any, error]`.

## Common mistakes

- **Returning channels in public APIs.** Don't. Return an `iter.Seq2`. Channels belong inside your implementation, not on the boundary.
- **Ignoring the `yield` return.** A producer that keeps yielding after `yield(…) == false` violates the iterator contract (the runtime panics) and leaks work. Always check and return.
- **Not respecting `ctx.Done()`.** A producer that doesn't check the context hangs forever on cancellation. Every blocking call must be context-aware.
- **Yielding after an error.** Yield the error once and return. Consumers stop at the first non-nil error.
- **Mutating chunks after yielding.** The consumer may still be reading. If you need to reuse a buffer, copy before yielding.

## Example: implementing your own

A paginated HTTP fetcher that streams results page by page:

```go
package paginated

import (
    "context"
    "encoding/json"
    "iter"
    "net/http"

    "github.com/lookatitude/beluga-ai/v2/core"
)

// Item is one result row.
type Item struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

// Page is one page of results.
type Page struct {
    Items []Item `json:"items"`
    Total int    `json:"total"`
    Next  string `json:"next"`
}

// Fetch follows Next links and yields each page.
func Fetch(ctx context.Context, client *http.Client, startURL string) iter.Seq2[Page, error] {
    return func(yield func(Page, error) bool) {
        url := startURL
        for url != "" {
            req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
            if err != nil {
                yield(Page{}, core.Errorf(core.ErrInvalidInput, "paginated: %w", err))
                return
            }
            resp, err := client.Do(req)
            if err != nil {
                yield(Page{}, core.Errorf(core.ErrProviderDown, "paginated: %w", err))
                return
            }
            var page Page
            err = json.NewDecoder(resp.Body).Decode(&page)
            _ = resp.Body.Close()
            if err != nil {
                yield(Page{}, core.Errorf(core.ErrProviderDown, "paginated: decode: %w", err))
                return
            }
            if !yield(page, nil) {
                return
            }
            url = page.Next
        }
    }
}
```

Consumer:

```go
for page, err := range paginated.Fetch(ctx, client, url) {
    if err != nil {
        return err
    }
    for _, item := range page.Items {
        process(item)
    }
}
```

`http.NewRequestWithContext` makes each request respect cancellation, so the loop needs no separate `ctx.Done()` check.

## Related

- [02 — Core Primitives](../architecture/02-core-primitives.md) — `Event[T]` and `Stream[T]` in detail.
- [`.guild/wiki/patterns/streaming.md`](../../.guild/wiki/patterns/streaming.md) — canonical code references.
- [11 — Voice Pipeline](../architecture/11-voice-pipeline.md) — streaming at audio rate.
