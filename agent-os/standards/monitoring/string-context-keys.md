# String Context Keys for Tracing

Use plain string keys for trace context propagation.

```go
// In tracer: store with string keys
ctx = context.WithValue(ctx, "trace_id", traceID)
ctx = context.WithValue(ctx, "span_id", spanID)
ctx = context.WithValue(ctx, "current_span", span)

// In logger: extract without importing tracer
if traceID, ok := ctx.Value("trace_id").(string); ok {
    entry.TraceID = traceID
}
if spanID, ok := ctx.Value("span_id").(string); ok {
    entry.SpanID = spanID
}
```

## Why Not OTEL Context Carriers?
- Logger can extract trace IDs without importing tracer package
- Avoids circular dependencies between monitoring subpackages
- Works alongside OTEL (parallel instrumentation)

## Standard Keys
| Key | Type | Purpose |
|-----|------|---------|
| `"trace_id"` | string | Distributed trace identifier |
| `"span_id"` | string | Current span identifier |
| `"current_span"` | *spanImpl | Full span for advanced access |

## Extraction Helpers
```go
func TraceIDFromContext(ctx context.Context) string {
    if traceID, ok := ctx.Value("trace_id").(string); ok {
        return traceID
    }
    return ""
}
```
