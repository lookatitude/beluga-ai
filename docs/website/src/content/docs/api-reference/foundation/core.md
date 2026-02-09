---
title: "Core Package"
description: "Foundation primitives: streams, Runnable, events, errors, lifecycle, multi-tenancy"
---

```go
import "github.com/lookatitude/beluga-ai/core"
```

Package core provides the foundational primitives for the Beluga AI framework.

It defines the universal execution model, typed event streaming, batch
processing, lifecycle management, multi-tenancy, context propagation,
functional options, and structured error handling that all other packages
build upon.

## Runnable Interface

Runnable is the universal execution interface. Every component that processes
input — LLMs, tools, agents, pipelines — implements Runnable. It supports
both synchronous invocation and streaming:

```go
type Runnable interface {
    Invoke(ctx context.Context, input any, opts ...Option) (any, error)
    Stream(ctx context.Context, input any, opts ...Option) iter.Seq2[any, error]
}
```

Runnables compose via `Pipe` (sequential) and `Parallel` (concurrent):

```go
pipeline := core.Pipe(tokenizer, llm)
result, err := pipeline.Invoke(ctx, "Hello, world!")

parallel := core.Parallel(agent1, agent2, agent3)
results, err := parallel.Invoke(ctx, input)
```

## Event Streaming

`Stream` is a pull-based event iterator built on Go 1.23+ [iter.Seq2].
Events are generic via `Event`, carrying a typed payload, event type,
optional error, and metadata:

```go
for event, err := range stream {
    if err != nil { break }
    switch event.Type {
    case core.EventData:
        fmt.Print(event.Payload)
    case core.EventToolCall:
        // handle tool invocation
    }
}
```

Stream utilities include `CollectStream`, `MapStream`, `FilterStream`,
`MergeStreams`, and `FanOut` for transforming and combining streams.
`BufferedStream` adds backpressure control between fast producers and
slow consumers, and `FlowController` provides semaphore-based
concurrency limiting.

## Batch Processing

`BatchInvoke` executes a function over multiple inputs concurrently with
configurable concurrency limits, per-item timeouts, and retry policies:

```go
results := core.BatchInvoke(ctx, embedFn, documents, core.BatchOptions{
    MaxConcurrency: 10,
    Timeout:        5 * time.Second,
})
```

## Lifecycle Management

The `Lifecycle` interface provides Start/Stop/Health semantics for
components that require explicit initialization and graceful shutdown.
`App` manages a set of Lifecycle components, starting them in registration
order and stopping them in reverse:

```go
app := core.NewApp()
app.Register(dbPool, cacheLayer, httpServer)
if err := app.Start(ctx); err != nil {
    log.Fatal(err)
}
defer app.Shutdown(ctx)
```

## Multi-Tenancy

`WithTenant` and `GetTenant` store and retrieve a `TenantID` from context,
enabling tenant-scoped data isolation across all framework operations.

## Context Helpers

`WithSessionID`, `GetSessionID`, `WithRequestID`, and `GetRequestID`
propagate session and request identifiers through context for correlation
across distributed traces and logs.

## Structured Errors

`Error` carries an operation name, `ErrorCode`, human-readable message,
and optional wrapped cause. Error codes like `ErrRateLimit`, `ErrTimeout`,
and `ErrProviderDown` enable programmatic retry decisions via `IsRetryable`:

```go
if core.IsRetryable(err) {
    // safe to retry
}
```

## Functional Options

The `Option` interface and `OptionFunc` adapter implement the functional
options pattern used throughout the framework for configuration.
