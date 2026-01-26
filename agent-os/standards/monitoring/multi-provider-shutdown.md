# Multi-Provider Shutdown

Shutdown all providers, collecting errors instead of failing fast.

```go
func (pr *ProviderRegistry) Shutdown() error {
    pr.mu.Lock()
    defer pr.mu.Unlock()

    var errors []error

    // Shutdown ALL providers, don't stop on first error
    for _, provider := range pr.loggers {
        if err := provider.Shutdown(); err != nil {
            errors = append(errors, fmt.Errorf("failed to shutdown logger %s: %w", provider.Name(), err))
        }
    }
    for _, provider := range pr.tracers {
        if err := provider.Shutdown(); err != nil {
            errors = append(errors, fmt.Errorf("failed to shutdown tracer %s: %w", provider.Name(), err))
        }
    }
    for _, provider := range pr.metrics {
        if err := provider.Shutdown(); err != nil {
            errors = append(errors, fmt.Errorf("failed to shutdown metrics %s: %w", provider.Name(), err))
        }
    }

    if len(errors) > 0 {
        return fmt.Errorf("provider shutdown errors: %v", errors)
    }
    return nil
}
```

## Why Collect All Errors?
- **Best-effort cleanup**: Shutdown as many providers as possible
- **Debugging visibility**: See all failures at once

## Shutdown Order
1. Loggers first (may still be needed by tracers/metrics)
2. Tracers second
3. Metrics last

## Usage
```go
defer func() {
    if err := monitoring.GetGlobalRegistry().Shutdown(); err != nil {
        log.Printf("Shutdown errors: %v", err)
    }
}()
```
