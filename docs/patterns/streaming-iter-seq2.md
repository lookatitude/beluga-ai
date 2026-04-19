# Pattern: Streaming with iter.Seq2

## What it is

Beluga streams use Go 1.23's `iter.Seq2[T, error]` — a pull-based iterator that integrates with the `for … range` syntax and propagates backpressure. Every streaming component returns a `*core.Stream[Event[T]]` that wraps an `iter.Seq2`. Channels never appear in public API boundaries.

## Why we use it

LLM tokens, tool observations, voice frames, retrieval results — all of these are naturally iterative streams. You need backpressure, cancellation, and composition. Channels give you this too, but at the cost of goroutine management and a heavier consumer syntax.

**Alternatives considered:**
- **Channels.** Require a goroutine per stage, leak goroutines if the consumer returns early, and need `select` for multi-stream coordination. Rejected for public APIs.
- **Callback/visitor pattern.** Inverts control — the producer drives. Consumers lose the ability to pause or cancel naturally. Rejected.
- **Buffered slices.** Trivial but loses streaming (the consumer waits for the whole result before seeing any of it). Rejected for anything interactive.

`iter.Seq2` is pull-based (the consumer drives), type-safe (generic), cancellation-aware (backpressure via the `yield` protocol), and uses a syntax every Go programmer already knows. It's the right primitive.

## How it works

Canonical code from `core/stream.go:49-56` (see [`.wiki/patterns/streaming.md`](../../.wiki/patterns/streaming.md)):

```go
// core/stream.go
package core

import "iter"

type Stream[T any] struct {
    name   string
    chunks iter.Seq2[int, T]
}

func (s *Stream[T]) Range(yield func(int, T) bool) {
    for idx, chunk := range s.chunks {
        if !yield(idx, chunk) {
            break
        }
    }
}
```

Consumer:

```go
for idx, chunk := range stream.Range {
    // do something with chunk
    if somethingBad {
        break // producer sees yield==false next iteration and cleans up
    }
}
```

Producer (an LLM wrapper):

```go
func (c *Client) Stream(ctx context.Context, req Request) *core.Stream[Event[Token]] {
    return core.NewStream("openai.stream", func(yield func(int, Event[Token]) bool) {
        resp, err := c.client.ChatStream(ctx, req)
        if err != nil {
            yield(0, Event[Token]{Err: err, Type: EventError})
            return
        }
        defer resp.Close()
        idx := 0
        for {
            select {
            case <-ctx.Done():
                yield(idx, Event[Token]{Err: ctx.Err(), Type: EventError})
                return
            default:
            }
            tok, err := resp.Next()
            if err == io.EOF {
                yield(idx, Event[Token]{Type: EventDone})
                return
            }
            if err != nil {
                yield(idx, Event[Token]{Err: err, Type: EventError})
                return
            }
            if !yield(idx, Event[Token]{Payload: tok, Type: EventData}) {
                return // consumer quit early; clean up
            }
            idx++
        }
    })
}
```

The `yield` return value is the backpressure signal: if it returns `false`, the consumer is done and the producer must exit cleanly.

## Stream operations

Four core compositions:

### Pipe — transform

```go
// conceptual
func Pipe[A, B any](in *Stream[A], fn func(A) B) *Stream[B]
```

Applies `fn` to every event of `in`. If the source ends, the pipe ends.

### Parallel — fan-out

```go
func Parallel[A, B any](in *Stream[A], fn func(context.Context, A) (B, error), n int) *Stream[B]
```

Runs `fn` on each event in parallel across `n` workers. Results are emitted in completion order (not input order) — if you need ordered output, use `Pipe` with a sequential function.

### Merge — fan-in

```go
func Merge[T any](streams ...*Stream[T]) *Stream[T]
```

Merges multiple streams into one. Used by scatter-gather orchestration.

### Copy — broadcast

```go
func Copy[T any](in *Stream[T], n int) []*Stream[T]
```

Broadcasts one stream to `n` consumers. Each consumer gets its own view; internally this buffers as much as needed to serve the slowest consumer.

## Where it's used

- `core` — `Stream[T]`, base operations.
- `llm` — `ChatModel.Stream` returns a token stream.
- `agent` — `Agent.Stream` returns an event stream.
- `voice` — `FrameProcessor` pipelines use channel-backed streams internally but expose `*Stream` at boundaries.
- `tool` — streaming tools return `*Stream[Chunk]`.
- `rag/retriever` — some retrievers (streaming retrieval, HyDE) return streams.

## Common mistakes

- **Returning channels in public APIs.** Don't. Use `*Stream[T]`. Channels belong inside your implementation, not on the boundary.
- **Ignoring the `yield` return.** A producer that keeps yielding after `yield(…) == false` leaks work. Always check and return.
- **Not respecting `ctx.Done()`.** A producer that doesn't check the context hangs forever on cancellation. Every blocking call must be context-aware.
- **Yielding partial state as final.** `EventDone` means the stream is over. Don't use it for checkpoints — use a custom `EventType` for mid-stream notifications.
- **Mutating chunks after yielding.** The consumer may still be reading. If you need to reuse a buffer, copy before yielding.

## Example: implementing your own

A paginated HTTP fetcher that streams results page by page:

```go
package paginated

import (
    "context"
    "io"
    "github.com/lookatitude/beluga-ai/v2/core"
)

type Page struct {
    Items []Item
    Total int
    Next  string
}

func Fetch(ctx context.Context, client *http.Client, startURL string) *core.Stream[core.Event[Page]] {
    return core.NewStream("paginated.fetch", func(yield func(int, core.Event[Page]) bool) {
        url := startURL
        idx := 0
        for url != "" {
            select {
            case <-ctx.Done():
                yield(idx, core.Event[Page]{Err: ctx.Err(), Type: core.EventError})
                return
            default:
            }
            req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
            resp, err := client.Do(req)
            if err != nil {
                yield(idx, core.Event[Page]{Err: err, Type: core.EventError})
                return
            }
            var page Page
            _ = json.NewDecoder(resp.Body).Decode(&page)
            _ = resp.Body.Close()
            if !yield(idx, core.Event[Page]{Payload: page, Type: core.EventData}) {
                return
            }
            idx++
            url = page.Next
        }
        yield(idx, core.Event[Page]{Type: core.EventDone})
    })
}
```

Consumer:

```go
for _, ev := range paginated.Fetch(ctx, client, url).Range {
    if ev.Err != nil {
        return ev.Err
    }
    if ev.Type == core.EventDone {
        break
    }
    for _, item := range ev.Payload.Items {
        // process
    }
}
```

## Related

- [02 — Core Primitives](../architecture/02-core-primitives.md) — `Event[T]` and `Stream[T]` in detail.
- [`.wiki/patterns/streaming.md`](../../.wiki/patterns/streaming.md) — canonical code references.
- [11 — Voice Pipeline](../architecture/11-voice-pipeline.md) — streaming at audio rate.
