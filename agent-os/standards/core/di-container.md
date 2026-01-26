# DI Container

Reflection-based dependency injection with auto-resolution.

## Registration
```go
container.Register(func() MyService { return &myServiceImpl{} })
container.Register(func(dep Dependency) MyService { return &myServiceImpl{dep} })
```
- Factory first return type becomes the registration key
- Factory parameters are auto-resolved recursively
- Last return value checked for error interface

## Resolution
```go
var svc MyService
container.Resolve(&svc) // Target must be pointer
```
- Results cached automatically (singleton behavior)
- Recursive resolution for factory dependencies
- No explicit cycle detection (relies on stack overflow)

## OTEL Context in Logs
```go
spanCtx := trace.SpanContextFromContext(ctx)
fields = append([]any{"trace_id", spanCtx.TraceID(), "span_id", spanCtx.SpanID()}, fields...)
```
- All DI operations extract and prepend trace context to logs

## NoOp Defaults
- `NewContainer()` uses noop.TracerProvider and noOpLogger
- `NewContainerWithOptions(WithLogger(...), WithTracerProvider(...))` for real observability
