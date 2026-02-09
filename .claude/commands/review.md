---
name: review
description: Review code against Beluga AI v2 architecture docs. Checks for compliance with design decisions, Go idioms, and testing coverage.
---

Review the specified code (or recent changes) against Beluga AI v2 architecture.

## Steps

1. Read `docs/concepts.md` for design decisions
2. Identify which packages are affected
3. Check each file against:

### Architecture Compliance
- iter.Seq2 for streaming (not channels)
- Registry pattern (Register/New/List)
- Middleware pattern (func(T) T)
- Hooks pattern (optional, composable)
- Small interfaces (≤4 methods)
- context.Context first parameter
- Functional options WithX()
- Zero external deps in core/schema
- No circular imports
- Typed errors with ErrorCode

### Go Idioms
- Error wrapping with %w
- Goroutine leak prevention
- Race condition safety
- Resource cleanup with defer
- Doc comments on exports
- Compile-time interface checks

### Testing
- Tests exist for all exports
- Error paths tested
- Context cancellation tested for streams
- Mocks in internal/testutil/

Report findings as: **Critical** / **Warning** / **Suggestion**
