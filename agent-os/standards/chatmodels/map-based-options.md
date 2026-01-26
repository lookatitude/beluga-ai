# Map-Based Option Application

Options use Apply(&map) pattern to avoid import cycles.

```go
// In iface/ - no dependency on implementation types
type Option interface {
    Apply(config *map[string]any)
}

// In chatmodels/ - applies options via type assertions
for _, opt := range opts {
    configMap := make(map[string]any)
    opt.Apply(&configMap)

    if temp, ok := configMap["temperature"].(float32); ok {
        options.Temperature = temp
    }
    if maxTokens, ok := configMap["max_tokens"].(int); ok {
        options.MaxTokens = maxTokens
    }
    // ... more type assertions
}
```

## Why Not Functional Options?
- Options interface defined in `iface/`
- Concrete option functions in `chatmodels/`
- Functional options would require iface to import implementation types → cycle

## Option Definition
```go
func WithTemperature(temp float32) iface.Option {
    return iface.OptionFunc(func(config *map[string]any) {
        (*config)["temperature"] = temp
    })
}
```

## Trade-off
- Pro: Avoids import cycles
- Con: Runtime type assertions instead of compile-time safety
- Con: Implicit coupling between key names and types
