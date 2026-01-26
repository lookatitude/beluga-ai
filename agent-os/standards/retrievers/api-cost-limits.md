# API Cost Protection with Hardcoded Limits

Cap unbounded parameters to prevent runaway API costs.

```go
numQueries := opts.NumQueries
if numQueries <= 0 {
    numQueries = 3 // Default to 3 query variations
}
if numQueries > 10 {
    numQueries = 10 // Cap at 10 to avoid excessive API calls
}
```

## Pattern
1. **Default** when not set or invalid (`<= 0`)
2. **Cap** to prevent excessive resource usage
3. **Comment** explaining the reasoning

## Examples in Codebase
- `MultiQueryRetriever.NumQueries`: default 3, cap 10
- `VectorStoreRetriever.DefaultK`: must be 1-100

## When to Apply
- Parameters that control API calls (queries, tokens, retries)
- Parameters that control resource allocation (batch size, concurrency)
- NOT for user-facing limits (let caller decide)

## Validation Pattern
```go
if opts.DefaultK < 1 || opts.DefaultK > 100 {
    return nil, NewError("DefaultK must be between 1 and 100")
}
```
