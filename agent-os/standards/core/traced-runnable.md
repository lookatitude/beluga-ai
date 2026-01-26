# TracedRunnable Wrapper

Decorator that adds OTEL tracing and metrics to any Runnable.

```go
traced := core.NewTracedRunnable(runnable, tracer, metrics, "llm", "gpt-4")
traced := core.RunnableWithTracing(runnable, tracer, metrics, "llm") // Without name
```

## Span Naming
```go
span name = componentType + "." + operation
// Examples: "llm.invoke", "retriever.batch", "chain.stream"
```

## NoOp Fallbacks
- `tracer == nil` → uses `noop.NewTracerProvider().Tracer("")`
- `metrics == nil` → uses `NoOpMetrics()`

## Stream Wrapper
- Creates goroutine to count chunks and measure total duration
- Overhead may be significant; make configurable for performance-critical paths (not yet implemented)

## Metrics Recorded
- `RecordRunnableInvoke(ctx, componentType, duration, err)`
- `RecordRunnableBatch(ctx, componentType, batchSize, duration, err)`
- `RecordRunnableStream(ctx, componentType, duration, chunkCount, err)`
