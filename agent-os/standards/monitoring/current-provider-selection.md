# Current Provider Selection

Require explicit provider selection before creation.

```go
type ProviderRegistry struct {
    loggers map[string]LoggerProvider
    tracers map[string]TracerProvider
    metrics map[string]MetricsProvider
    current map[string]string  // "logger" -> "otel", "tracer" -> "jaeger"
}

// Must call SetCurrent* before Create*
registry.RegisterLoggerProvider(otelProvider)
registry.RegisterLoggerProvider(stdoutProvider)
registry.SetCurrentLogger("otel")  // Required!

logger, err := registry.CreateLogger("my-service", config)
// Error if SetCurrentLogger not called: "no current logger provider set"
```

## Why Not Auto-Select First?
- App may have multiple providers active simultaneously
- Logger→stdout, Tracer→Jaeger, Metrics→Prometheus
- Different components may use different providers

## Pattern
1. Register all providers at startup
2. Set current provider for each type based on config
3. Create instances using current provider

## Error Behavior
```go
if !exists {
    return nil, errors.New("no current logger provider set")
}
```
Fails loudly if provider not explicitly selected.
