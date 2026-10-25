---
type: standard
title: "Provider Registration Pattern"
description: "Migrated from .wiki/patterns/provider-registration.md."
owner: orchestrator
confidence: high
trust_tier: reviewed
importance: high
source_refs: [wiki-patterns-provider-registration]
created_at: 2026-04-11
updated_at: 2026-04-11
expires_at: null
supersedes: null
sensitivity: internal
recall_importance: 3
---
# Provider Registration Pattern

> Migrated from `.wiki/patterns/provider-registration.md` on 2026-10-25 (Guild 3.x; structure kept). Verbatim original: `.guild/knowledge/sources/wiki-patterns-provider-registration/original.md` (sha256 `60c64e31ac2a…`).

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
