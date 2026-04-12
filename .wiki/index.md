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

## Documentation Conventions (for docs-writer tasks)

Key corrections for anyone writing or auditing documentation:

- **C-006** — `docs/` has drifted from codebase: providers.md undercounts, first-agent.md uses non-existent APIs, README has unshipped features. See `.wiki/corrections.md`.
- **C-007** — `gh pr view --json state` returning `MERGED` does NOT confirm code is in `main`. Always verify with `ls <path>` in the worktree. See `.wiki/corrections.md`.
- **C-009** — This repo uses a multi-branch merge workflow. PRs merge into staging/integration before `main`. Canonical presence check: `git ls-files origin/main -- <path>`. See `.wiki/corrections.md`.
- **feature-status**: Provider counts verified by `ls <pkg>/providers/ | wc -l`. As of 2026-04-12: 22 LLM, 9 embedding, 13 vectorstore, 9 memory, 6 STT, 7 TTS, 3 S2S, 3 transport providers.
- **readme-drift**: README and `docs/reference/providers.md` must match `ls */providers/` output. Both files are hand-curated and drift within one release cycle.

## Project Documentation Cross-Reference

The authoritative human-facing docs now live under `docs/`:

- **`docs/README.md`** — top-level entry point
- **`docs/architecture/`** — 18 architecture docs (overview, primitives, extensibility, data flow, agent anatomy, reasoning, orchestration, runner, memory, RAG, voice, protocol, security, observability, resilience, workflows, deployment, package map)
- **`docs/patterns/`** — 8 pattern docs mirroring `.wiki/patterns/` but with full prose and rationale
- **`docs/guides/`** — 7 how-to guides (first agent, custom provider, custom planner, multi-agent team, deploy k8s/temporal/docker)
- **`docs/reference/`** — interfaces, configuration, glossary, providers

When answering user questions about architecture, prefer reading `docs/architecture/*` first (prose with rationale) and fall back to `.wiki/` for canonical `file:line` pointers.

