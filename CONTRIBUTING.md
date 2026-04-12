# Contributing to Beluga AI

Thank you for your interest in contributing to Beluga AI. Every contribution — bug report, feature request, documentation improvement, or code change — helps make this framework better for everyone.

This guide gets you from a fresh clone to an open PR. The fastest path is under an hour.

## Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. Report unacceptable behavior to conduct@lookatitude.com.

---

## Your first contribution in under an hour

The fastest path: add a provider for a service you already know.

1. **Pick a category** — LLM, embedding, vector store, STT, TTS, guard, memory store, etc.
2. **Read [Provider Template](docs/patterns/provider-template.md)** — the 5-part scaffold every provider follows.
3. **Copy an existing provider** from the same category as your starting point (e.g. `llm/providers/openai/` for LLM).
4. **Implement the interface** (≤ 4 methods) and register in `init()`.
5. **Write table-driven tests** covering the happy path, error paths, and registration.
6. **Run the pre-commit gate** (see below).
7. **Open a PR**.

Other great first contributions:

- **Write a cookbook** — realistic end-to-end examples in `docs/guides/`
- **Improve docs** — if something confused you, fix the wording and open a PR
- **Add eval metrics** — new metrics go in `eval/metrics/`
- **Add a built-in tool** — new tools go in `tool/<name>/`
- **Fix a bug** — search [open issues](https://github.com/lookatitude/beluga-ai/issues) labelled `good first issue`

---

## Development setup

### Prerequisites

- **Go 1.23+** — uses `iter.Seq2` for streaming
- **Git**
- **Make** (optional but recommended)
- **golangci-lint** — `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`
- **gosec** — `go install github.com/securego/gosec/v2/cmd/gosec@latest`
- **govulncheck** — `go install golang.org/x/vuln/cmd/govulncheck@latest`

### Clone and build

```bash
git clone https://github.com/lookatitude/beluga-ai.git
cd beluga-ai
go mod download
go build ./...
go test -race ./...
```

### Common Make targets

| Target                   | Description                                  |
|--------------------------|----------------------------------------------|
| `make build`             | Build all packages                           |
| `make test`              | Run unit tests with race detection           |
| `make lint`              | Run go vet and golangci-lint                 |
| `make integration-test`  | Run integration tests                        |
| `make coverage`          | Generate HTML coverage report                |
| `make fuzz`              | Run fuzz tests (30 s each)                   |
| `make bench`             | Run benchmarks                               |
| `make check`             | Full pre-commit check (lint + test)          |
| `make fmt`               | Format code with gofmt + goimports           |
| `make tidy`              | Run go mod tidy and verify clean             |

---

## The extension model (learn in 2 minutes)

Every extensible package in Beluga follows the same 4-ring contract. Learn it once, apply it to any of the 13 pluggable packages.

**Ring 1 — Interface** (≤ 4 methods, compile-time check required):

```go
type Tool interface {
    Name()        string
    Description() string
    InputSchema() map[string]any
    Execute(ctx context.Context, input map[string]any) (*Result, error)
}

// Every implementation must include this:
var _ Tool = (*MyTool)(nil)
```

**Ring 2 — Registry** (`Register` / `New` / `List`):

```go
// Providers self-register in init():
func init() {
    if err := tool.Register("my_tool", newFactory()); err != nil {
        panic(err) // duplicate registration = programming error
    }
}

// Users construct via the registry, never directly:
t, err := tool.New("my_tool", tool.Config{})
```

**Ring 3 — Hooks** (nil-safe, composable):

```go
// Hooks fire at lifecycle points, e.g. before/after tool execution.
// All fields are optional — nil means skip.
hooks := tool.ComposeHooks(auditHooks, metricsHooks)
```

**Ring 4 — Middleware** (`func(T) T`, applied outside-in):

```go
wrapped := tool.ApplyMiddleware(base,
    tool.WithRetry(3),
    tool.WithRateLimit(100),
    tool.WithTracing(),
)
```

Full details: [DOC-03 Extensibility Patterns](docs/architecture/03-extensibility-patterns.md).

---

## The layering rule

Layer N imports only from Layers 1 … N−1. Never upward.

```
Layer 7 — Application
Layer 6 — Agent runtime     (agent, runtime)
Layer 5 — Orchestration     (orchestration/*)
Layer 4 — Protocol gateway  (protocol, server)
Layer 3 — Capability        (llm, tool, memory, rag, voice, guard, prompt, cache, eval, hitl)
Layer 2 — Cross-cutting     (resilience, auth, audit, cost, state, workflow)
Layer 1 — Foundation        (core, schema, config, o11y)
```

`go vet` catches import cycles. `arch-validate` enforces the full rule. Never add an upward import — refactor the caller instead.

Full details: [DOC-18 Package Dependency Map](docs/architecture/18-package-dependency-map.md).

---

## Pre-commit gate (MANDATORY before every commit)

Run all of the following and fix any new findings before committing. Pre-existing findings in files you did **not** change do not block your commit, but must be noted in the commit message.

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

`make check` runs the first five commands. Run `gosec` and `govulncheck` separately.

---

## Branch discipline

Never commit directly to `main`. Every change — code, docs, tests — lives on a branch and merges via PR.

```bash
# Create a branch:
git checkout -b feat/my-feature

# Commit and push:
git add <files>
git commit -m "feat(llm): add streaming support for echo provider"
git push -u origin feat/my-feature

# Open a PR:
gh pr create
```

Branch naming:

| Prefix        | Use for                    |
|---------------|----------------------------|
| `feat/`       | New features               |
| `fix/`        | Bug fixes                  |
| `docs/`       | Documentation changes      |
| `refactor/`   | Code refactoring           |
| `test/`       | Test additions or fixes    |
| `chore/`      | Tooling, CI, dependencies  |

---

## Code style

- **TDD** — write the failing test first, then make it pass.
- **`context.Context` is always the first parameter** of every public function, no exceptions.
- **No `interface{}` in public APIs** — use generics.
- **Errors use `core.Error`** with a typed `ErrorCode`; wrap with `%w` to preserve the chain.
- **Streaming uses `iter.Seq2[T, error]`** — never channels in public APIs.
- **Interfaces have ≤ 4 methods** — split larger surfaces into composed interfaces.
- **Every exported symbol has a godoc comment.**

---

## Commit message format

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

**Types:** `feat`, `fix`, `docs`, `chore`, `test`, `refactor`, `perf`, `ci`

**Scopes** (optional): `llm`, `agent`, `tool`, `rag`, `voice`, `memory`, `core`, `schema`, `guard`, `protocol`, `cache`, `auth`, `eval`, `config`, `o11y`

**Breaking changes:** Append `!` after the type/scope (e.g. `feat(llm)!: remove deprecated Generate API`) or include a `BREAKING CHANGE:` line in the commit body. This prevents automatic release — the team publishes major versions manually.

Examples:

```
feat(llm): add streaming support for Anthropic provider
fix(agent): prevent nil pointer in handoff execution
docs(rag): add hybrid search tutorial
test(tool): add FuncTool edge case tests
```

---

## CI pipeline

Every PR runs the full automated suite.

**Quality and testing:**

| Check              | Description                                                       |
|--------------------|-------------------------------------------------------------------|
| golangci-lint      | 13 linters (gosec, staticcheck, errcheck, revive, etc.)           |
| go vet             | Official Go static analysis                                       |
| Unit tests         | `go test -race` with coverage reporting                           |
| Integration tests  | `go test -race -tags integration`                                 |
| SonarCloud         | Code quality, duplication, and maintainability analysis           |

**Security scanning:**

| Scanner      | Purpose                                                                          |
|--------------|----------------------------------------------------------------------------------|
| gosec        | Static analysis for Go security issues (SARIF upload to GitHub Security tab)    |
| govulncheck  | Symbol-level reachability analysis against the Go vulnerability database         |
| Snyk         | Dependency vulnerability scanning with severity thresholds                       |
| Trivy        | Filesystem and dependency scanning (SARIF upload to GitHub Security tab)         |
| CodeQL       | Deep semantic static analysis (weekly + push/PR)                                 |
| Gitleaks     | Secret detection across git history                                               |
| go-licenses  | Dependency license compliance (MIT, Apache-2.0, BSD-2/3-Clause, ISC, MPL-2.0)  |

Security scans also run on a weekly schedule (Monday 4 am UTC) to catch newly disclosed CVEs.

**Code review:**

| Tool      | Description                                                                              |
|-----------|------------------------------------------------------------------------------------------|
| Greptile  | AI-powered code review on every PR — provides contextual feedback from full codebase    |

---

## Releases

### Automatic releases (patch and minor)

When CI passes, a job inspects all commits since the last tag:

| Commits contain                              | Bump    | Example                 |
|----------------------------------------------|---------|-------------------------|
| Any `feat:` (no breaking)                    | Minor   | `v1.2.3` → `v1.3.0`   |
| Only `fix:`, `docs:`, `chore:`, etc.         | Patch   | `v1.2.3` → `v1.2.4`   |
| Any breaking change (`feat!:`, `BREAKING CHANGE`) | Skipped | No tag created     |

### Major releases (manual only)

Major versions are never created automatically. Go to **Actions → Release → Run workflow** and specify the desired tag (e.g. `v2.0.0`).

You can also push a version tag directly: `git tag v1.2.3 && git push origin v1.2.3`. The Release workflow triggers on any `v*.*.*` tag.

---

## Reporting bugs and suggesting features

- **Bug report:** open a [bug report](https://github.com/lookatitude/beluga-ai/issues/new?template=bug_report.yml) with a clear description, reproduction steps, and expected vs actual behavior.
- **Feature request:** open a [feature request](https://github.com/lookatitude/beluga-ai/issues/new?template=feature_request.yml) describing the problem, proposed solution, and use case.

## Security vulnerabilities

If you discover a security vulnerability, **do not** open a public issue. Report it responsibly by emailing **security@lookatitude.com**.

---

## Where to get help

- [GitHub Discussions](https://github.com/lookatitude/beluga-ai/discussions) — questions, ideas, design discussion
- [GitHub Issues](https://github.com/lookatitude/beluga-ai/issues) — bugs and tracked feature work
- [Full docs](https://beluga-ai.org/docs/) — architecture, guides, API reference

---

## License

By contributing to Beluga AI, you agree that your contributions will be licensed under the [Apache License 2.0](LICENSE).
