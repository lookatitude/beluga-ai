---
title: Building a Custom Runnable
description: "Implement the core Runnable interface in Go to create reusable, composable pipeline components that integrate with Beluga AI's Pipe, Parallel, and streaming."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go, tutorial, Runnable interface, composable, pipeline, streaming, custom component"
---

The `core.Runnable` interface is the universal execution abstraction in Beluga AI. Every component that processes input — LLMs, tools, agents, pipelines — implements `Runnable`. This uniformity exists because composition depends on a shared contract: `Pipe` can chain any two runnables, `Parallel` can fan out to any set of runnables, and middleware can wrap any runnable. Without a common interface, each composition pattern would need special-case handling for every component type.

By building custom runnables, you create first-class components that plug into composition, streaming, and observability without additional glue code.

## What You Will Build

A deterministic keyword-based sentiment analyzer that implements `core.Runnable`. This component integrates seamlessly with Beluga AI's `Pipe`, `Parallel`, and streaming infrastructure.

## Prerequisites

- Go 1.23+ (for `iter.Seq2` streaming support)
- Familiarity with `context.Context` and Go interfaces

## The Runnable Interface

The interface defines two methods — synchronous execution and streaming:

```go
package core

type Runnable interface {
    // Invoke executes synchronously and returns a single result.
    Invoke(ctx context.Context, input any, opts ...Option) (any, error)

    // Stream returns an iterator of intermediate results.
    Stream(ctx context.Context, input any, opts ...Option) iter.Seq2[any, error]
}
```

Beluga AI v2 uses `iter.Seq2[T, error]` (Go 1.23+) for streaming rather than channels. This design choice avoids the resource leak risks inherent in channel-based APIs (forgotten closes, goroutine leaks) and supports cooperative cancellation through the `yield` return value. Consumers iterate with a standard `for range` loop, which aligns with idiomatic Go patterns.

## Step 1: Define the Component

Create a struct that holds any configuration or state your component requires. Following Beluga AI's convention, the constructor uses the `New` prefix and accepts configuration as parameters.

```go
package main

import (
    "context"
    "fmt"
    "iter"
    "strings"

    "github.com/lookatitude/beluga-ai/core"
)

// KeywordSentiment analyzes text for positive or negative keywords.
type KeywordSentiment struct {
    DefaultSentiment string
}

// NewKeywordSentiment creates a new analyzer with the given default sentiment.
func NewKeywordSentiment(defaultSentiment string) *KeywordSentiment {
    return &KeywordSentiment{DefaultSentiment: defaultSentiment}
}
```

## Step 2: Implement Invoke

`Invoke` handles synchronous, single-input execution. Since the `Runnable` interface uses `any` for input and output types (to enable heterogeneous composition), you must validate the input type at the entry point. Returning a descriptive error for type mismatches is important because these errors surface at composition time, where the developer needs to understand which component received unexpected input.

```go
func (k *KeywordSentiment) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
    text, ok := input.(string)
    if !ok {
        return nil, fmt.Errorf("KeywordSentiment: expected string input, got %T", input)
    }

    lower := strings.ToLower(text)
    if strings.Contains(lower, "good") || strings.Contains(lower, "great") {
        return "POSITIVE", nil
    }
    if strings.Contains(lower, "bad") || strings.Contains(lower, "terrible") {
        return "NEGATIVE", nil
    }

    return k.DefaultSentiment, nil
}
```

## Step 3: Implement Stream

`Stream` returns an `iter.Seq2[any, error]` — a pull-based iterator. For a component that produces a single result, emit the result once and return. For components that produce multiple values (like an LLM generating tokens), yield each value in sequence. The common pattern for single-result components is to delegate to `Invoke` and yield the result, which keeps the synchronous and streaming paths consistent.

```go
func (k *KeywordSentiment) Stream(ctx context.Context, input any, opts ...core.Option) iter.Seq2[any, error] {
    return func(yield func(any, error) bool) {
        result, err := k.Invoke(ctx, input, opts...)
        if err != nil {
            yield(nil, err)
            return
        }
        yield(result, nil)
    }
}
```

There are no channels or goroutines needed. The `iter.Seq2` pattern is simpler, avoids resource leaks, and supports cooperative cancellation through the `yield` return value.

## Step 4: Use the Component

```go
func main() {
    analyzer := NewKeywordSentiment("NEUTRAL")
    ctx := context.Background()

    // Single invocation
    result, err := analyzer.Invoke(ctx, "This is a good outcome")
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    fmt.Printf("Result: %s\n", result) // Output: Result: POSITIVE

    // Streaming
    for val, err := range analyzer.Stream(ctx, "That was terrible") {
        if err != nil {
            fmt.Printf("Stream error: %v\n", err)
            break
        }
        fmt.Printf("Stream value: %s\n", val)
    }
}
```

## Composition with Pipe

The primary benefit of implementing `Runnable` is composition. `core.Pipe` chains runnables sequentially, passing the output of one as the input to the next. This works because all runnables share the same `(any, error)` signature — the composition infrastructure does not need to know the concrete types flowing through the pipeline.

```go
// Pipe the sentiment analyzer's output into another runnable
pipeline := core.Pipe(analyzer, &SentimentReporter{})

result, err := pipeline.Invoke(ctx, "This is great news")
if err != nil {
    fmt.Printf("Pipeline error: %v\n", err)
    return
}
fmt.Println(result)
```

Use `core.Parallel` to fan out to multiple runnables concurrently. Parallel execution is safe because each runnable receives its own copy of the input and produces independent output — there is no shared mutable state between branches.

```go
// Run multiple analyzers in parallel
parallel := core.Parallel(
    NewKeywordSentiment("NEUTRAL"),
    NewKeywordSentiment("UNKNOWN"),
)

results, err := parallel.Invoke(ctx, "This is good")
if err != nil {
    fmt.Printf("Error: %v\n", err)
    return
}
fmt.Println(results) // []any{"POSITIVE", "POSITIVE"}
```

## Stream Utilities

Beluga AI provides utility functions for working with streams. These utilities compose with `iter.Seq2` iterators, enabling functional-style data processing without allocating intermediate collections.

```go
// Collect all stream values into a slice
stream := analyzer.Stream(ctx, "good day")
events, err := core.CollectStream(stream)

// Transform stream values
mapped := core.MapStream(stream, func(e core.Event[any]) (core.Event[string], error) {
    return core.Event[string]{Payload: fmt.Sprint(e.Payload)}, nil
})

// Filter stream values
filtered := core.FilterStream(stream, func(e core.Event[any]) bool {
    return e.Payload != nil
})
```

## Troubleshooting

**Stream consumer hangs indefinitely**: Ensure your `Stream` implementation always returns from the iterator function. With `iter.Seq2`, there are no channels to forget to close — the iterator function simply returns when done.

**Type assertion panics**: Always validate input types at the beginning of `Invoke` and return descriptive errors instead of panicking. This is especially important when your runnable is part of a composed pipeline where the upstream output type may vary.

## Next Steps

- [Middleware Implementation](/docs/tutorials/foundation/middleware-implementation) — Wrap runnables with cross-cutting behavior
- [Multi-turn Conversations](/docs/tutorials/foundation/multiturn-conversations) — Model structured conversations with the schema package
