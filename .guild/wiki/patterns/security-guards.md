---
type: standard
title: "Security Guards Pattern"
description: "Migrated from .wiki/patterns/security-guards.md."
owner: orchestrator
confidence: high
trust_tier: reviewed
importance: high
source_refs: [wiki-patterns-security-guards]
created_at: 2026-04-11
updated_at: 2026-04-11
expires_at: null
supersedes: null
sensitivity: internal
recall_importance: 3
---
# Security Guards Pattern

> Migrated from `.wiki/patterns/security-guards.md` on 2026-10-25 (Guild 3.x; structure kept). Verbatim original: `.guild/knowledge/sources/wiki-patterns-security-guards/original.md` (sha256 `00d73d82a0ec…`).

Three-stage guard pipeline (Input → Output → Tool) with guard.Guard interface and GuardResult decision codes.

## Canonical Example

**File:** `guard/guard.go:1-52`

```go
type Guard interface {
	InspectInput(ctx context.Context, input GuardInput) (GuardResult, error)
	InspectOutput(ctx context.Context, output GuardOutput) (GuardResult, error)
	InspectTool(ctx context.Context, tool GuardTool) (GuardResult, error)
}

type GuardResult struct {
	Decision Decision // Allow, Review, Block
	Reason   string
}

type Decision string

const (
	Allow   Decision = "allow"
	Review  Decision = "review"
	Block   Decision = "block"
)
```

## Variations

1. **Guard composability** — Multiple guards chained via pipeline
   - First Block decision stops processing
   - Review decisions aggregate for human review

2. **Custom guard implementations** — Domain-specific validation
   - Token limit guards (budget checking)
   - Content policy guards (harmful content detection)
   - Tool access guards (allowlist-based)

## Anti-Patterns

- **Skipping guard stages**: Applying Input guard but not Tool guard; incomplete coverage
- **Ignoring Review decisions**: Treating Review as Allow; bypasses compliance
- **Unbounded guard chain**: No limit on guard count; O(n) decision latency
- **Silent failures**: Guard error not propagated; request proceeds unsafely

## Invariants

- All three guard stages (Input, Output, Tool) must be invoked in order
- First Block decision stops processing; Request rejected immediately
- Review decisions logged for audit trail
- Guard errors always propagated (never silently ignored)
- All guard implementations implement Guard interface (compile-time check: var _ Guard = (*impl)(nil))
