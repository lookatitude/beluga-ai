---
title: Core Package API
description: API documentation for the core package providing foundational primitives.
---

```go
import "github.com/lookatitude/beluga-ai/core"
```

Package core provides the foundational primitives for the Beluga AI framework: typed event streams, the Runnable execution interface, batch processing, context helpers, multi-tenancy, lifecycle management, and typed errors.

## Key Types

### Stream

`Stream[T]` is a pull-based event iterator built on Go 1.23+ `iter.Seq2`:

```go
type Stream[T any] = iter.Seq2[Event[T], error]
```

Consumers use range to iterate:

```go
for event, err := range stream {
    if err != nil { break }
    // handle event
}
```

### Event

`Event[T]` is the unit of data flowing through the system:

```go
type Event[T any] struct {
    Type    EventType      // "data", "tool_call", "tool_result", "handoff", "done", "error"
    Payload T              // Typed event data
    Err     error          // Error for "error" events
    Meta    map[string]any // Trace IDs, latency, token counts, etc.
}
```

### Runnable

`Runnable` is the universal execution interface implemented by LLMs, tools, agents, and pipelines:

```go
type Runnable interface {
    Invoke(ctx context.Context, input any, opts ...Option) (any, error)
    Stream(ctx context.Context, input any, opts ...Option) iter.Seq2[any, error]
}
```

### Error

`Error` is a structured error with operation, code, message, and cause:

```go
type Error struct {
    Op      string    // "llm.generate", "tool.execute", etc.
    Code    ErrorCode // "rate_limit", "timeout", "auth_error", etc.
    Message string    // Human-readable description
    Err     error     // Underlying cause
}
```

Error codes:

```go
const (
    ErrRateLimit       ErrorCode = "rate_limit"
    ErrAuth            ErrorCode = "auth_error"
    ErrTimeout         ErrorCode = "timeout"
    ErrInvalidInput    ErrorCode = "invalid_input"
    ErrToolFailed      ErrorCode = "tool_failed"
    ErrProviderDown    ErrorCode = "provider_unavailable"
    ErrGuardBlocked    ErrorCode = "guard_blocked"
    ErrBudgetExhausted ErrorCode = "budget_exhausted"
)
```

Check if an error is retryable:

```go
if core.IsRetryable(err) {
    // retry the operation
}
```

## Context Helpers

### Multi-Tenancy

```go
ctx = core.WithTenant(ctx, "tenant-123")
tenantID := core.GetTenant(ctx) // returns TenantID
```

### Session & Request IDs

```go
ctx = core.WithSessionID(ctx, "session-456")
ctx = core.WithRequestID(ctx, "req-789")

sessionID := core.GetSessionID(ctx)
requestID := core.GetRequestID(ctx)
```

## Batch Processing

Process multiple inputs concurrently with configurable limits:

```go
results := core.BatchInvoke(ctx, fn, inputs, core.BatchOptions{
    MaxConcurrency: 10,
    Timeout:        5 * time.Second,
    RetryPolicy:    &core.RetryPolicy{MaxAttempts: 3},
})

for i, result := range results {
    if result.Err != nil {
        log.Printf("Item %d failed: %v", i, result.Err)
    } else {
        log.Printf("Item %d: %v", i, result.Value)
    }
}
```

## Stream Utilities

### Collect

Drain a stream into a slice:

```go
events, err := core.CollectStream(stream)
```

### Map

Transform stream events:

```go
mapped := core.MapStream(src, func(event core.Event[T]) (core.Event[U], error) {
    // transform event
    return newEvent, nil
})
```

### Filter

Keep only matching events:

```go
filtered := core.FilterStream(src, func(event core.Event[T]) bool {
    return event.Type == core.EventData
})
```

### Merge

Combine multiple streams:

```go
merged := core.MergeStreams(ctx, stream1, stream2, stream3)
```

### Fan-Out

Duplicate a stream to multiple consumers:

```go
streams := core.FanOut(ctx, source, 3) // 3 independent consumers
```

### Buffered Stream

Add backpressure buffering:

```go
buffered := core.NewBufferedStream(ctx, source, 100) // 100-event buffer
for event, err := range buffered.Iter() {
    // consume at your own pace
}
```

## Composition

### Pipe

Chain two runnables:

```go
composed := core.Pipe(retriever, llm)
result, err := composed.Invoke(ctx, "query")
```

### Parallel

Fan-out to multiple runnables:

```go
parallel := core.Parallel(model1, model2, model3)
results, err := parallel.Invoke(ctx, input) // []any with 3 results
```

## Lifecycle Management

### App

Manage component lifecycles:

```go
type Lifecycle interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Health() HealthStatus
}

app := core.NewApp()
app.Register(dbConn, cacheClient, apiServer)

// Start all in order
if err := app.Start(ctx); err != nil {
    log.Fatal(err)
}

// Stop all in reverse order
defer app.Shutdown(context.Background())

// Health checks
statuses := app.HealthCheck()
```

## Functional Options

Apply options to any configurable type:

```go
type options struct {
    timeout time.Duration
}

core.ApplyOptions(&opts,
    WithTimeout(30*time.Second),
    WithRetry(3),
)
```

## Flow Control

Manage backpressure:

```go
fc := core.NewFlowController(10) // max 10 concurrent

// Blocking acquire
if err := fc.Acquire(ctx); err != nil {
    return err
}
defer fc.Release()

// Non-blocking try
if fc.TryAcquire() {
    defer fc.Release()
    // do work
}
```

## See Also

- [Schema Package](./schema.md) for message and content types
- [LLM Package](./llm.md) for LLM abstraction
- [Agent Package](./agent.md) for agent runtime
