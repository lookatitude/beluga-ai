# Middleware Pattern

Function-based middleware composition applied outside-in (outermost middleware executes first).

## Canonical Example

**File:** `tool/middleware.go:11-22`

```go
func ApplyMiddleware(tool Tool, mws ...Middleware) Tool {
	result := tool
	for i := len(mws) - 1; i >= 0; i-- {
		result = mws[i](result)
	}
	return result
}

type Middleware func(Tool) Tool
```

## Variations

1. **WithTimeout concrete middleware** — `tool/middleware.go:27-56`
   - Wraps Tool.Execute with context.WithTimeout
   - Propagates metadata (Name, Description, InputSchema)

2. **WithRetry middleware** — `tool/middleware.go:58-85`
   - Inspects core.IsRetryable to decide retry
   - Respects context cancellation
   - Exponential backoff with jitter

## Anti-Patterns

- **Inside-out application**: Reversing middleware order breaks intended semantics
- **Metadata loss**: Not delegating Name/Description/InputSchema to wrapped tool
- **Non-idempotent operations**: Retrying side-effecting tools without guards
- **Unbounded retries**: No max attempt limit; infinite loop on transient errors

## Invariants

- Middleware applied via reverse iteration: mws[n-1](mws[n-2](...(tool)))
- Outermost middleware (first in slice) executes first
- Metadata always read from underlying tool, not wrapped version
- Context cancellation short-circuits retry loops
