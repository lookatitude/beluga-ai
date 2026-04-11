# Beluga AI v2

Go-native agentic AI framework. `github.com/lookatitude/beluga-ai`. Go 1.23+. Streaming via `iter.Seq2[T, error]`.

## Critical rules

1. Streaming uses `iter.Seq2[T, error]` in public APIs — never channels. `Invoke()` = stream + collect.
2. Every extensible package: Interface → Registry (`Register`/`New`/`List`) → Hooks → Middleware (`func T → T`).
3. Providers auto-register in `init()` + `Register()`. No config files to edit.
4. Library never imports `k8s/`. Kubernetes is an optional overlay.
5. Errors: `core.Error` with typed `ErrorCode`. Check `IsRetryable()` before retry.
6. OTel GenAI spans (`gen_ai.*`) at every package boundary.
7. Tests: table-driven, `-race`, `*_test.go` alongside source. Red/Green TDD.
8. No `interface{}` in public APIs — use generics. No global mutable state outside registries.
9. Interfaces have ≤4 methods. `context.Context` is the first parameter of every public function.
10. Zero external deps in `core/` and `schema/` beyond stdlib + otel. No circular imports.

## Before writing code

1. File-scoped rules in `.claude/rules/` auto-load for the files you touch.
2. Run `.claude/hooks/wiki-query.sh <package>` — returns relevant wiki index entries, corrections, and pattern files in one call.
3. Read `.wiki/index.md` retrieval routing table; read the targeted files for your task type.
4. Read existing code in the target package — match style exactly.
5. Write a failing test first, then make it pass (Red/Green TDD).

## Commands (all independently triggerable)

| Command | Purpose |
|---|---|
| `/plan $FEATURE` | Architect + Researcher design loop → implementation plan with acceptance criteria |
| `/develop $TASK` | Developer-go Red/Green TDD → QA review → fix loop |
| `/security-review $PATH` | 2 consecutive clean passes required |
| `/qa-review $PATH` | Standalone QA review |
| `/doc-check $PATH` | Verify examples compile and docs match current API |
| `/document $TARGET` | Write package docs, tutorials, API reference |
| `/promote $FEATURE` | Blog + social + release note |
| `/blog $TOPIC` | Technical blog post |
| `/dependency-audit` | gosec + govulncheck + update safe deps |
| `/new-feature $DESC` | Composite pipeline: plan → develop → security-review → document → promote |
| `/learn $DESCRIPTION` | Capture a correction into `.wiki/corrections.md` |
| `/wiki-learn [$PATH\|all]` | Extract patterns and architecture from the codebase into `.wiki/` |
| `/arch-validate $PACKAGE` | Validate code against architecture invariants |
| `/arch-update $CHANGE` | Update architecture docs + ADRs after significant changes |
| `/notion-sync` | Mirror docs/ to Notion + update tracking dashboard |
| `/status` | Package health snapshot |

## Architecture references

@docs/concepts.md
@docs/packages.md
@docs/architecture.md
@docs/providers.md

## Agent team

- **coordinator** — orchestrator, breaks down work, captures learnings
- **architect** — designs interfaces, writes ADRs, validates invariants
- **researcher** — evidence gathering, never implements
- **developer-go** — Red/Green TDD Go implementation
- **developer-web** — Astro/Starlight website
- **reviewer-qa** — validates acceptance criteria, read-only
- **reviewer-security** — 2 clean passes required, read-only
- **docs-writer** — package docs, tutorials, API reference
- **marketeer** — blog, release notes, social content
- **notion-syncer** — Notion sync and tracking dashboard

## Learning pipeline

Per-agent `.claude/agents/<name>/rules/` (automatic, fast) → `.wiki/corrections.md` (curated) → `.claude/rules/<file>.md` (enforced) → `CLAUDE.md` (always-loaded, human-approved).
