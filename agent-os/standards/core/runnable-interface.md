# Runnable Interface

Central abstraction for composable components.

```go
type Runnable interface {
    Invoke(ctx context.Context, input any, options ...Option) (any, error)
    Batch(ctx context.Context, inputs []any, options ...Option) ([]any, error)
    Stream(ctx context.Context, input any, options ...Option) (<-chan any, error)
}
```

## Input/Output Typing
- Uses `any` for composability (chains, graphs)
- Type checking deferred to runtime
- Implementations type-assert internally

## Option Pattern
```go
type Option interface {
    Apply(config *map[string]any)
}

// Create ad-hoc options
core.WithOption("temperature", 0.7)
```
- Options modify a shared config map
- Applied before execution, not stored

## Stream Contract
- Channel closed when complete
- Errors sent as last item before close (check type)
- Context cancellation stops stream

## Implementing Runnable
- LLMs, Retrievers, Agents, Chains, Graphs all implement Runnable
- Embed in custom types for composition
