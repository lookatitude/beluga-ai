# Hooks Pattern

Optional func field hooks with nil-safe composition via ComposeHooks.

## Canonical Example

**File:** `tool/hooks.go:9-44`

```go
type Hooks struct {
	OnStart func(ctx context.Context, name string, input map[string]any) error
	OnEnd   func(ctx context.Context, name string) error
	OnError func(ctx context.Context, name string, err error) error
}

func ComposeHooks(hks ...Hooks) Hooks {
	return Hooks{
		OnStart: func(ctx context.Context, name string, input map[string]any) error {
			for _, h := range hks {
				if h.OnStart != nil {
					if err := h.OnStart(ctx, name, input); err != nil {
						return err
					}
				}
			}
			return nil
		},
		// ... OnEnd, OnError similarly
	}
}
```

## Variations

1. **Single hook invocation** — `llm/client.go:line` (hypothetical)
   - Direct nil check: `if c.hooks.OnStart != nil { ... }`

2. **Guard hooks pipeline** — `guard/guard.go` (hypothetical)
   - Structured hooks for Input/Output/Tool guard stages

## Anti-Patterns

- **Nil pointer dereference**: Calling hook without checking `!= nil`
- **Silent error swallowing**: Returning error from hook but ignoring it
- **Unbounded hook chains**: No limit on ComposedHooks depth; O(n) invocations per operation
- **Hook side effects**: Modifying context/input within hooks; breaks idempotence

## Invariants

- All OnStart/OnEnd/OnError hook fields are optional (nil = no-op)
- ComposeHooks returns nil-safe composed hook struct; all callbacks check nil before invoke
- Hooks execute sequentially; first error stops chain
- Hooks never modify caller's context or input parameters
