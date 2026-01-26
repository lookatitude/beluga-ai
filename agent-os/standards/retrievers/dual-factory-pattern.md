# Dual Factory Pattern

Support both functional options and config struct factories.

```go
// Functional options pattern (runtime flexibility)
func NewVectorStoreRetriever(vectorStore VectorStore, options ...Option) (*VectorStoreRetriever, error) {
    opts := &RetrieverOptions{
        DefaultK:       4,
        ScoreThreshold: 0.0,
        MaxRetries:     3,
    }
    for _, option := range options {
        option(opts)
    }
    // validate and create...
}

// Config struct pattern (declarative setup)
func NewVectorStoreRetrieverFromConfig(vectorStore VectorStore, config Config) (*VectorStoreRetriever, error) {
    config.ApplyDefaults()
    if err := config.Validate(); err != nil {
        return nil, err
    }
    // create from config...
}
```

## When to Use Each

| Factory | Use Case |
|---------|----------|
| Functional options | Runtime flexibility, programmatic configuration |
| Config struct | File-based config, declarative setup, serialization |

## Guidelines
- Both factories populate the same internal options struct
- Config struct must have `ApplyDefaults()` and `Validate()` methods
- Functional options apply defaults before processing options
- Keep default values consistent between both factories
