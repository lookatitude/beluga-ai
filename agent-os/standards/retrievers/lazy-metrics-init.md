# Lazy Metrics Initialization

Only create metrics when both enabled and Meter is provided.

```go
// Create metrics if enabled AND meter provided
if opts.EnableMetrics && opts.Meter != nil {
    var err error
    opts.Metrics, err = NewMetrics(opts.Meter, opts.Tracer)
    if err != nil {
        return nil, NewRetrieverError("NewVectorStoreRetriever", err, ErrCodeInvalidConfig)
    }
}
```

## Why This Pattern
- **Optional observability**: Works without OTEL configured
- **Explicit opt-in**: `EnableMetrics` flag prevents accidental overhead
- **Graceful fallback**: nil Metrics means no-op in recording calls

## Options Struct Pattern
```go
type RetrieverOptions struct {
    Tracer         trace.Tracer   // Optional: injected or nil
    Meter          metric.Meter   // Optional: injected or nil
    Logger         *slog.Logger   // Optional: defaults to slog.Default()
    Metrics        *Metrics       // Created lazily if enabled
    EnableTracing  bool
    EnableMetrics  bool
}
```

## Recording with nil Check
```go
if m.enableMetrics && m.metrics != nil {
    m.metrics.RecordRetrieval(ctx, "multi_query", duration, len(docs), avgScore, nil)
}
```

## Default Logger Fallback
```go
if opts.Logger == nil {
    opts.Logger = slog.Default()
}
```
