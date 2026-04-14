---
title: Core Primitives
description: "Stream[T], Event[T], and Runnable — the three types every Beluga AI component is built from. Canonical definitions with source references."
---

Three types underpin every package in Beluga: `Event[T]`, `Stream[T]`, and
`Runnable`. Every LLM call, tool execution, memory load, and agent turn
produces or consumes these types. Understand them and you can read any package
in the framework without a guide.

All three are defined in `core/`, which has zero external dependencies beyond
the Go standard library and OpenTelemetry.

## Event\[T\] — the unit of data flow

`Event[T]` is the single type that flows through every stream. It carries a
typed payload, an optional error, and a metadata map.

Source: [`core/stream.go:35-47`](https://github.com/lookatitude/beluga-ai/blob/main/core/stream.go#L35-L47)

```go
// core/stream.go:35-47
type Event[T any] struct {
    Type    EventType       // data, tool_call, tool_result, handoff, done, error
    Payload T               // concrete type depends on the producer
    Err     error           // non-nil only for EventError events
    Meta    map[string]any  // trace IDs, token counts, provider annotations
}
```

Six event types are defined at `core/stream.go:12-30`:

| `EventType` | Meaning |
|---|---|
| `EventData` | A payload chunk — text token, audio frame, document |
| `EventToolCall` | The LLM is requesting a tool invocation |
| `EventToolResult` | A tool returned its result |
| `EventHandoff` | An agent-to-agent transfer |
| `EventDone` | The stream terminated normally |
| `EventError` | The stream terminated with an error |

**Why one type for the whole stream.** You could use separate channel types
for each signal (`TextChunkChan`, `ToolCallChan`), but that forces every
consumer to spin up parallel goroutines and join them. A single `Event[T]`
keeps iteration linear: one `for event, err := range stream` handles
everything.

**Why `Meta map[string]any`.** The metadata map carries per-event observational
data — OTel trace IDs, token counts, cost hints — that no critical-path code
depends on. The payload stays strictly typed; the metadata stays flexible. See
[DOC-14 — Observability](/docs/reference/architecture/overview/observability).

## Stream\[T\] — the transport

`Stream[T]` is a type alias over Go 1.23's `iter.Seq2`:

Source: [`core/stream.go:56`](https://github.com/lookatitude/beluga-ai/blob/main/core/stream.go#L56)

```go
// core/stream.go:56
type Stream[T any] = iter.Seq2[Event[T], error]
```

That is the complete definition. `Stream[T]` is not a struct — it is a
function type that takes a `yield` callback. The `for … range` syntax in Go
1.23+ calls the function and drives the yield protocol automatically.

### Consumer pattern

```go
import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/core"
)

func consumeStream[T any](ctx context.Context, stream core.Stream[T]) error {
    for event, err := range stream {
        if err != nil {
            return fmt.Errorf("stream error: %w", err)
        }
        if event.Type == core.EventDone {
            break
        }
        // process event.Payload
    }
    return nil
}
```

The `yield` protocol provides backpressure for free: if the consumer returns
from the loop early (`break` or `return`), the Go runtime stops calling the
producer's next iteration. No goroutine leaks, no channel draining.

### Stream combinators

Four functions in `core/stream.go` compose streams without introducing
intermediate goroutines:

| Function | Signature | What it does |
|---|---|---|
| `CollectStream` | `Stream[T] → ([]Event[T], error)` | Drains to a slice |
| `MapStream` | `Stream[T] × func → Stream[U]` | Transforms each event |
| `FilterStream` | `Stream[T] × predicate → Stream[T]` | Drops non-matching events |
| `MergeStreams` | `...Stream[T] → Stream[T]` | Interleaves N streams |
| `FanOut` | `Stream[T] × n → []Stream[T]` | Broadcasts to N consumers |

`MergeStreams` (`core/stream.go:113-130`) uses internal goroutines for the
multi-producer case. All other combinators are pure closures.

### Backpressure and `BufferedStream`

When a producer is faster than a consumer, `NewBufferedStream`
(`core/stream.go:230-255`) absorbs bursts via a bounded internal channel.
The buffer capacity is explicit — there is no unbounded growth:

```go
import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/core"
)

func bufferExample[T any](ctx context.Context, src core.Stream[T]) error {
    bs := core.NewBufferedStream(ctx, src, 64) // 64-event buffer
    for event, err := range bs.Iter() {
        if err != nil {
            return fmt.Errorf("buffered stream: %w", err)
        }
        _ = event
    }
    return nil
}
```

`FlowController` (`core/stream.go:285-295`) provides explicit
acquire/release semaphore-style backpressure for producers that need to
pause, not buffer.

## Runnable — the composable unit

`Runnable` is the interface every component that processes data implements.

Source: [`core/runnable.go:15-22`](https://github.com/lookatitude/beluga-ai/blob/main/core/runnable.go#L15-L22)

```go
// core/runnable.go:15-22
type Runnable interface {
    Invoke(ctx context.Context, input any, opts ...Option) (any, error)
    Stream(ctx context.Context, input any, opts ...Option) iter.Seq2[any, error]
}
```

`LLM`, `Tool`, `Agent`, `Retriever`, `PromptTemplate` — all implement
`Runnable` so they can be composed:

```go
import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/core"
)

func pipeExample(ctx context.Context, retriever, promptBuild, llm core.Runnable) error {
    chain := core.Pipe(retriever, core.Pipe(promptBuild, llm))
    for val, err := range chain.Stream(ctx, "what is the refund policy?") {
        if err != nil {
            return fmt.Errorf("chain: %w", err)
        }
        _ = val
    }
    return nil
}
```

`core.Pipe` (`core/runnable.go:26-28`) is the sequential combinator. It
invokes `a`, passes the result to `b`, and streams `b`'s output.
`core.Parallel` (`core/runnable.go:64-66`) fans out to N runnables
concurrently and collects results.

### `Invoke` vs `Stream`

`Invoke` is a convenience wrapper — it calls `Stream` internally and
returns the last non-error value. Middleware and hooks attach to `Stream`;
`Invoke` delegates. **Never implement `Invoke` without implementing `Stream`.**

## Common mistakes

- **Storing `ctx` in a struct field.** Pass it per call. See [Context](/docs/concepts/context).
- **Returning early from `range` without a `break`.** Go handles the teardown, but returning an error from inside the loop while ignoring the remaining events can mask the stream's terminal error. Read the loop to completion when you need the full result.
- **Using channels in a public return type.** Return `core.Stream[T]`. Channels belong inside your implementation as synchronization primitives, not in the API. See C-004 and C-005 in `.wiki/corrections.md`.
- **Implementing `Invoke` without `Stream`.** Middleware attaches to `Stream`; calling `Invoke` on a type that fakes it breaks middleware and hook chains.

## Related reading

- [Streaming](/docs/concepts/streaming) — `iter.Seq2` in depth: producers, consumers, backpressure.
- [Extensibility](/docs/concepts/extensibility) — how middleware and hooks layer on top of `Runnable`.
- [DOC-02 — Core Primitives](/docs/reference/architecture/overview/core-primitives) — the full architecture doc with more detail on context helpers.
- [`core/stream.go`](https://github.com/lookatitude/beluga-ai/blob/main/core/stream.go) — canonical source.
