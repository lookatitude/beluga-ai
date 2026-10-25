---
type: standard
title: "Middleware Pattern"
description: "Migrated from .wiki/patterns/middleware.md."
owner: orchestrator
confidence: high
trust_tier: reviewed
importance: high
source_refs: [wiki-patterns-middleware]
created_at: 2026-04-11
updated_at: 2026-04-11
expires_at: null
supersedes: null
sensitivity: internal
recall_importance: 3
---
# Middleware Pattern

> Migrated from `.wiki/patterns/middleware.md` on 2026-10-25 (Guild 3.x; structure kept). Verbatim original: `.guild/knowledge/sources/wiki-patterns-middleware/original.md` (sha256 `3df02c38c832…`).

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
