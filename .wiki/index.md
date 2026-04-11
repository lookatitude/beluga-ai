# Beluga AI v2 Architecture Wiki

**Last Scanned:** 2026-04-11

This wiki documents canonical architectural patterns, package structure, and invariants across the Beluga AI v2 Go codebase.

## Patterns

8 core patterns extracted with real `file:line` references, code snippets, variations, anti-patterns, and invariants:

1. **[Provider Registration](./patterns/provider-registration.md)** — Registry-based global provider discovery with sync.RWMutex
2. **[Middleware](./patterns/middleware.md)** — Function-based middleware composition applied outside-in
3. **[Hooks](./patterns/hooks.md)** — Optional func field hooks with nil-safe ComposeHooks composition
4. **[Streaming](./patterns/streaming.md)** — Go 1.23 iter.Seq2 range-over-func producers with internal channels
5. **[Testing](./patterns/testing.md)** — Table-driven tests with context cancellation and error code validation
6. **[OTel Instrumentation](./patterns/otel-instrumentation.md)** — GenAI semantic conventions (v1.37+) with gen_ai.* attributes
7. **[Error Handling](./patterns/error-handling.md)** — Structured errors with ErrorCode enums and IsRetryable classification
8. **[Security Guards](./patterns/security-guards.md)** — Three-stage guard pipeline (Input → Output → Tool) with Decision enum

## Architecture

- **[Package Map](./architecture/package-map.md)** — 7 core packages with purpose, key types, registry, dependencies, test coverage
- **[Invariants](./architecture/invariants.md)** — 10 architectural invariants with file:line references and violation symptoms

## Retrieval Protocol

To validate findings or add new patterns:

1. **Index Lookup** — Check `.wiki/index.md` (this file) for pattern or invariant name
2. **Pattern/Invariant Query** — Read `.wiki/patterns/*.md` or `.wiki/architecture/*.md` files directly
3. **Source Validation** — Cross-reference file:line citations (e.g., `core/errors.go:8-39`) against actual codebase

## Scan Artifacts

Raw scan output: `raw/research/wiki-scan-2026-04-11.md`

