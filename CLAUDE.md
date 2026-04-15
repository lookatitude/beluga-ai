# Beluga AI v2

Go-native agentic AI framework. `github.com/lookatitude/beluga-ai`. Go 1.23+. Streaming via `iter.Seq2[T, error]`.

## Project scope

This repo owns the **engineering layer** of the Beluga project: code, tech docs, and releases. It does **not** own website content, marketing, blog posts, runnable example programs, or cross-repo strategy — those live in sibling repos (`beluga-website`, `beluga-examples`) and in the private workspace repo (`beluga.git`).

When working on cross-repo concerns (release flow fan-out, website dispatch, shared incident learnings, feature-brief coordination), consult the **workspace wiki** at `../.wiki/index.md` and the workspace `CLAUDE.md` at `../CLAUDE.md` — they're only visible when this repo is checked out inside the multi-repo workspace. Anything code-grounded or framework-specific stays here, in this repo's `.wiki/` and `docs/`.

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

## Branch + PR discipline (MANDATORY)

1. **Never commit to `main`.** Every change — code, docs, tests — starts with `git checkout -b <type>/<short-desc>` where `<type>` is `fix`, `feat`, `refactor`, `docs`, or `chore`.
2. Every branch ends with `gh pr create`. CI runs gosec, golangci-lint, govulncheck, SonarCloud, Snyk, Trivy, CodeQL, unit + integration tests. Wait for green before merge.
3. Verify `git branch --show-current` ≠ `main` before any `git commit`.

## Pre-commit verification gate (MANDATORY)

Before ANY `git commit` on a Go change, run and pass:

```bash
go build ./...
go vet ./...
go test -race ./...
go mod tidy && git diff --exit-code go.mod go.sum
gofmt -l . | grep -v ".claude/worktrees"
golangci-lint run ./...
gosec -quiet ./...
govulncheck ./...
```

Pre-existing findings in files you did NOT change: document in the commit
message, don't block on them. New findings in files you DID change: fix
before commit. See `.claude/rules/go-packages.md` for gosec focus areas.

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
| `/dependency-audit` | gosec + govulncheck + update safe deps |
| `/new-feature $DESC` | Composite pipeline: plan → develop → security-review → document |
| `/learn $DESCRIPTION` | Capture a correction into `.wiki/corrections.md` |
| `/wiki-learn [$PATH\|all]` | Extract patterns and architecture from the codebase into `.wiki/` |
| `/arch-validate $PACKAGE` | Validate code against architecture invariants |
| `/arch-update $CHANGE` | Update architecture docs + ADRs after significant changes |
| `/status` | Package health snapshot |

Marketing and content commands (`/promote`, `/blog`) and `/notion-sync` live in the website and workspace repos respectively — see `beluga-website/.claude/commands/` and `beluga.git/.claude/commands/`. Framework-side `/new-feature` no longer chains into promotion; promotion happens independently in the website repo when content is ready.

## Architecture references

@docs/README.md
@docs/architecture/01-overview.md
@docs/architecture/03-extensibility-patterns.md
@docs/architecture/18-package-dependency-map.md
@docs/reference/providers.md

## Agent team

- **coordinator** — orchestrator, breaks down work, captures learnings
- **architect** — designs interfaces, writes ADRs, validates invariants
- **researcher** — evidence gathering, never implements
- **developer-go** — Red/Green TDD Go implementation
- **reviewer-qa** — validates acceptance criteria, read-only
- **reviewer-security** — 2 clean passes required, read-only
- **docs-writer** — package docs, tutorials, API reference

Website (`developer-web`, `marketeer`) and Notion (`notion-publisher` in website, `notion-syncer` in workspace) agents live in their respective sibling repos. They are reachable when this repo is checked out inside the multi-repo Beluga workspace, but not invoked by framework commands.

## Learning pipeline

Per-agent `.claude/agents/<name>/rules/` (automatic, fast) → `.wiki/corrections.md` (curated) → `.claude/rules/<file>.md` (enforced) → `CLAUDE.md` (always-loaded, human-approved).
