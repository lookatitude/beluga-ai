---
name: streaming-patterns
description: Go 1.23 iter.Seq2 streaming and backpressure patterns for Beluga AI v2. Use when implementing streaming in any package, creating stream transformers, handling backpressure, or working with the Event[T] type system.
---

# Streaming Patterns for Beluga AI v2

## Primary Primitive: iter.Seq2[T, error]

Beluga uses Go 1.23+ range-over-func iterators, NOT channels, for public API streaming.

```go
// Producing a stream
func (m *Model) Stream(ctx context.Context, msgs []schema.Message) iter.Seq2[schema.StreamChunk, error] {
    return func(yield func(schema.StreamChunk, error) bool) {
        // Setup
        stream, err := m.client.Stream(ctx, msgs)
        if err != nil {
            yield(schema.StreamChunk{}, err)
            return
        }
        defer stream.Close()

        // Produce events
        for {
            select {
            case <-ctx.Done():
                yield(schema.StreamChunk{}, ctx.Err())
                return
            default:
            }

            chunk, err := stream.Recv()
            if err == io.EOF {
                return // stream complete
            }
            if err != nil {
                yield(schema.StreamChunk{}, err)
                return
            }
            if !yield(convertChunk(chunk), nil) {
                return // consumer stopped iteration
            }
        }
    }
}

// Consuming a stream
for chunk, err := range model.Stream(ctx, msgs) {
    if err != nil {
        log.Error("stream error", "err", err)
        break
    }
    fmt.Print(chunk.Text)
}
```

## Event[T] Type

```go
type Event[T any] struct {
    Type    EventType
    Payload T
    Err     error
    Meta    map[string]any
}

type EventType string
const (
    EventData       EventType = "data"
    EventToolCall   EventType = "tool_call"
    EventToolResult EventType = "tool_result"
    EventHandoff    EventType = "handoff"
    EventDone       EventType = "done"
    EventError      EventType = "error"
)
```

## Stream Composition

### Pipe (sequential)
```go
func Pipe[A, B any](
    first iter.Seq2[A, error],
    transform func(A) (B, error),
) iter.Seq2[B, error] {
    return func(yield func(B, error) bool) {
        for a, err := range first {
            if err != nil {
                var zero B
                yield(zero, err)
                return
            }
            b, err := transform(a)
            if !yield(b, err) {
                return
            }
        }
    }
}
```

### Fan-out (parallel)
```go
func FanOut[T any](stream iter.Seq2[T, error], n int) []iter.Seq2[T, error] {
    // Use iter.Pull to get next/stop, then broadcast to n consumers
    next, stop := iter.Pull2(stream)
    // ... create n output iterators that share the source
}
```

### Collect (stream to slice)
```go
func Collect[T any](stream iter.Seq2[T, error]) ([]T, error) {
    var results []T
    for item, err := range stream {
        if err != nil {
            return results, err
        }
        results = append(results, item)
    }
    return results, nil
}
```

## Invoke from Stream

`Invoke()` is always implemented as "stream, collect, return last":
```go
func (a *Agent) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
    var last any
    for event, err := range a.Stream(ctx, input, opts...) {
        if err != nil {
            return nil, err
        }
        last = event
    }
    return last, nil
}
```

## Backpressure with BufferedStream

```go
type BufferedStream[T any] struct {
    source iter.Seq2[T, error]
    buffer chan T
    size   int
}

func NewBufferedStream[T any](source iter.Seq2[T, error], bufferSize int) *BufferedStream[T] {
    return &BufferedStream[T]{source: source, size: bufferSize}
}
```

## Context Cancellation

ALWAYS check context in stream producers:
```go
select {
case <-ctx.Done():
    yield(zero, ctx.Err())
    return
default:
}
```

## iter.Pull for Pull Semantics

When a consumer needs explicit next/stop control:
```go
next, stop := iter.Pull2(stream)
defer stop()

val, err, ok := next()
if !ok { /* stream exhausted */ }
```

## Rules
1. Public API streaming: `iter.Seq2[T, error]` — NEVER `<-chan`
2. Internal goroutine communication: channels are fine
3. Always handle context cancellation in producers
4. `yield` returning false means consumer stopped — respect it immediately
5. Use `iter.Pull2` only when pull semantics are genuinely needed
6. Collect/Invoke are convenience wrappers around streaming
