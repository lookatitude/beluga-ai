# Wiki Index

On-demand knowledge base for Beluga AI v2. Agents read this file first to find what's relevant to their task. Populate by running `/wiki-learn all`.

**Last scanned:** never (run `/wiki-learn all` to populate)

## Retrieval routing

Agents: match your task type against this table, then read only the listed files.

| Task type                               | Read these files |
|-----------------------------------------|------------------|
| Implement provider in `llm/providers/*` | `patterns/provider-registration.md`, `patterns/streaming.md`, `patterns/otel-instrumentation.md`, `architecture/package-map.md#llm` |
| Add streaming API                       | `patterns/streaming.md`, `patterns/testing.md` |
| Security-sensitive edit                 | `patterns/security.md`, `corrections.md` (grep for package name) |
| Refactor interface                      | `architecture/invariants.md`, `architecture/decisions.md` |
| New package                             | `architecture/invariants.md`, `architecture/package-map.md`, `patterns/provider-registration.md` |
| Bug fix in tests                        | `patterns/testing.md`, `corrections.md` (grep: test) |
| Add middleware                          | `patterns/middleware.md`, `architecture/invariants.md` |
| Add hooks                               | `patterns/hooks.md` |
| Error handling                          | `patterns/error-handling.md`, `corrections.md` (grep: error) |
| Documentation update                    | `architecture/package-map.md`, latest entries in `log.md` |

## Files

### Core

- `index.md` — this file
- `log.md` — append-only chronological record of workflow runs
- `corrections.md` — C-NNN formatted correction log (curated)

### Patterns — canonical implementation snippets

- `patterns/provider-registration.md` — Register/New/List + init()
- `patterns/middleware.md` — `func(T) T` signature, outside-in application
- `patterns/hooks.md` — optional function fields, ComposeHooks
- `patterns/streaming.md` — iter.Seq2[T, error], backpressure, cancellation
- `patterns/testing.md` — table-driven, stream testing, testutil mocks
- `patterns/otel-instrumentation.md` — gen_ai.* spans at boundaries
- `patterns/error-handling.md` — core.Error + ErrorCode + IsRetryable
- `patterns/security.md` — input validation, guard pipeline, injection prevention

### Architecture

- `architecture/invariants.md` — the 10 design invariants with WHY
- `architecture/decisions.md` — ADR log
- `architecture/package-map.md` — generated package → purpose → key types

### Competitors

- `competitors/adk-go.md` — Google ADK Go
- `competitors/eino.md` — ByteDance Eino
- `competitors/langchaingo.md` — LangChainGo

### Releases

- `releases/drafts/` — in-progress release notes

## Maintenance

- `/wiki-learn` regenerates patterns, package-map, invariants from current code.
- `/learn` appends a correction.
- Coordinator promotes mature learnings from per-agent rules to `corrections.md`.
- Periodic wiki lint checks: no orphans, no stale references, all corrections have prevention rules.
