# Provider Registration Pattern

Registry-based global provider discovery with sync.RWMutex protection.

## Canonical Example

**File:** `llm/registry.go:19-27`

```go
func Register(name string, factory Factory) error {
	mu.Lock()
	defer mu.Unlock()
	
	if _, exists := providers[name]; exists {
		return fmt.Errorf("provider %q already registered", name)
	}
	providers[name] = factory
	return nil
}
```

## Variations

1. **Registration via init()** — `llm/providers/anthropic/anthropic.go:19-23`
   ```go
   func init() {
   	if err := llm.Register("anthropic", newFactory()); err != nil {
   		panic(err)
   	}
   }
   ```

2. **Instance-based registry with Add/Get** — `tool/registry.go:17-35`
   - Not global; passed explicitly to callers
   - Same sync.RWMutex protection
   - Error on duplicate registration

## Anti-Patterns

- **Missing duplicate check**: Silently overwrites existing provider
- **Unprotected map access**: Race conditions on concurrent Register + Get
- **panic() in init()**: Prevents graceful degradation; use error return instead
- **No cleanup function**: Registered providers persist for application lifetime

## Invariants

- All providers registered before main() executes via init()
- Registry lookup is O(1) map access
- Zero providers can be registered (valid empty state)
