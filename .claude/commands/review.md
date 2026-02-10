---
name: review
description: Review code against Beluga AI v2 architecture and Go idioms.
---

Review the specified code (or recent changes) against Beluga AI v2 architecture.

## Steps

1. Read `docs/concepts.md` for the relevant package.
2. Check against:

### Architecture

- iter.Seq2 for streaming (not channels)
- Registry pattern (Register/New/List)
- Middleware `func(T) T`
- Hooks (optional, composable)
- Small interfaces (<= 4 methods)
- context.Context first, functional options WithX()
- Zero external deps in core/schema, no circular imports
- Typed errors with ErrorCode

### Go Idioms

- Error wrapping %w, goroutine leak prevention, race safety
- Resource cleanup with defer, doc comments on exports
- Compile-time interface checks

### Testing

- Tests for all exports, error paths, context cancellation for streams

3. Report: **Critical** / **Warning** / **Suggestion**.
