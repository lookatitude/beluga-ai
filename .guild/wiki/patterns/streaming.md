---
type: standard
title: "Streaming Pattern"
description: "Migrated from .wiki/patterns/streaming.md."
owner: orchestrator
confidence: high
trust_tier: reviewed
importance: high
source_refs: [wiki-patterns-streaming]
created_at: 2026-04-11
updated_at: 2026-04-11
expires_at: null
supersedes: null
sensitivity: internal
recall_importance: 3
---
# Streaming Pattern

> Migrated from `.wiki/patterns/streaming.md` on 2026-10-25 (Guild 3.x; structure kept). Verbatim original: `.guild/knowledge/sources/wiki-patterns-streaming/original.md` (sha256 `7c40bb29baac…`).

iter.Seq2 range-over-func producers for chunk-based streaming with internal channels.

## Canonical Example

**File:** `core/stream.go:49-56`

```go
type Stream[T any] struct {
	name   string
	chunks iter.Seq2[int, T]
}

func (s *Stream[T]) Range(yield func(int, T) bool) {
	for idx, chunk := range s.chunks {
		if !yield(idx, chunk) {
			break
		}
	}
}
```

## Variations

1. **MapStream producer** — `core/stream.go:73-90`
   - Takes input Stream and transformation func
   - Yields transformed chunks
   - Breaks early on context cancellation

2. **LLMStream with token chunks** — `llm/stream.go` (hypothetical)
   - Wraps provider-specific streaming response
   - Exposes iter.Seq2[int, Token] interface
   - Handles channel closure gracefully

## Anti-Patterns

- **Buffered channels**: Unbuffered recommended to respect backpressure
- **Leaked goroutines**: Not closing channel on early break; caller waits forever
- **Ignoring yield() return**: Continuing iteration after yield returns false wastes work
- **Range not exposed as public iter.Seq2**: Forces internal iteration logic on callers

## Invariants

- All public Stream types expose iter.Seq2[int, T] or iter.Seq[T] interface
- Range never blocks indefinitely; respects yield() return value
- Internal channels always closed by producer goroutine
- Chunk index (first return value) starts at 0, increments sequentially
