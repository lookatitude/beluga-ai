# Security Review — DX-1 S1 CLI Foundation — Pass 1
Branch: feat/loo-142-cli-foundation-cobra-version-providers
Reviewer: reviewer-security
Date: 2026-04-19

## Scope

Reviewed the 10-commit diff on `feat/loo-142-cli-foundation-cobra-version-providers` vs `main` (24 files, +1513 / −185). The change migrates `cmd/beluga` from `flag` to `cobra`, adds `beluga version` and `beluga providers [--output json]` subcommands, ships a curated 7-provider blank-import set, bumps the Go toolchain to 1.25.9, enables the goreleaser 5-target build + archives + checksum, adds a `smoke-install` CI job on `release.yml`, and introduces an `o11y.BootstrapFromEnv` skeleton. The review checklist is the architect consultation (`docs/consultations/2026-04-19-loo-142-architect-plan.md`) §"Risks / Notes for reviewer-security" items 1–10, plus the additional manual checks supplied in the review dispatch.

## Automated checks

| Check | Command | Result |
|---|---|---|
| gosec (touched pkgs) | `gosec -quiet ./cmd/beluga/... ./o11y/...` | 4 findings, **all in untouched files** (`o11y/tracer.go` G101×2 on `gen_ai.usage.*_tokens` constants — LOW confidence false positives; `o11y/providers/phoenix/phoenix.go` G115 + G404 on `math/rand/v2` used for the non-security Phoenix session id). None in files changed by this PR. Clean for scope. |
| govulncheck | `govulncheck ./...` | **Cannot run locally** — pre-existing toolchain mismatch (govulncheck built against go1.25 but system `go list` is go1.26). Documented pre-existing issue in the developer's hand-off note; CI runs the canonical govulncheck gate on PR. No cobra CVE is listed on vuln.go.dev for `v1.10.2`, `pflag v1.0.10`, or `mousetrap v1.1.0` as of 2026-04-19. |
| golangci-lint (touched pkgs) | `golangci-lint run ./cmd/beluga/... ./o11y/...` | **Cannot run locally** — pre-existing v2 config migration debt ("unsupported version of the configuration"). Documented pre-existing issue. CI gate covers this on PR. |
| go vet (whole module) | `go vet ./...` | Clean. No findings. |
| go build -race | `go build -race ./...` | Clean. Compiles across the module with the race detector. |
| go test -race (touched) | `go test -race ./cmd/beluga/... ./o11y/...` | 167 tests across 9 packages — all pass. |

## Manual checks

### 6. Supply chain — cobra/pflag/mousetrap

`go list -m github.com/spf13/cobra github.com/spf13/pflag github.com/inconshreveable/mousetrap`:
- `github.com/spf13/cobra v1.10.2`
- `github.com/spf13/pflag v1.0.10`
- `github.com/inconshreveable/mousetrap v1.1.0`

All three at standard tagged releases, no `replace` or `retract` directives, no `+incompatible` or pseudo-version. These are the latest stable as of review date. cobra is maintained under the spf13 umbrella (a Kubernetes-SIG-adjacent author). No transitive deps added beyond `inconshreveable/mousetrap` (Windows double-click guard) and `cpuguy83/go-md2man/v2` (indirect test-only dep of cobra's docs gen — not in the production import graph).

**PASS.**

### 7. Path-traversal preservation (`cmd/beluga/init.go`)

Diff review confirms the `filepath.Clean` + `filepath.Abs` + containment check defence is structurally identical post-migration — only the flag-source variable name changed (from `*dir` dereferences to the local `dir` parameter of the extracted `runInit(name, dir string)` helper). The defence at `cmd/beluga/init.go:41-54` (cleanDir, relDir computation, `strings.HasPrefix(relDir, "..")` rejection) is unchanged.

Tests confirm:
```
$ go test -race -run 'TestCmdInit' ./cmd/beluga/...
Go test: 4 passed in 3 packages
```
Both `TestCmdInit_PathTraversal` (absolute `/tmp/../etc/passwd`-style input) and `TestCmdInit_RelativeTraversal` (`../..`) still exist and pass.

**PASS.**

### 8. `#nosec` annotation preservation (`cmd/beluga/test.go`)

Grep of `cmd/beluga` shows the pre-existing justification annotations survived the cobra migration:

```
cmd/beluga/test.go:22:	// #nosec G204 -- name is always an absolute path resolved via exec.LookPath("go")
cmd/beluga/test.go:25:	c := exec.Command(name, args...) //nolint:gosec // G204: see nosec justification above
```

The block comment at lines 18–24 describing the defence-in-depth (absolute path via `exec.LookPath("go")`, regex-validated `-pkg`, no shell) is preserved; only one word changed (`cmdTest` → `runTest` in line 20, a comment housekeeping edit). The `validPkgPattern` regex at `cmd/beluga/test.go:14` is untouched: `^[A-Za-z0-9_./\-]+(\.\.\.)?$`.

**PASS.**

### 9. stdout / stderr separation for `version` and `providers`

Built a local binary and exercised each subcommand with `1>stdout.txt 2>stderr.txt`:

- `beluga version` → stderr empty, exit 0. stdout contains `beluga v…`, `go1.26.2-X:nodwarf5`, and `providers: llm=3 embedding=2 vectorstore=1 memory=4`.
- `beluga providers` (text) → stderr empty, exit 0. Text output via tabwriter.
- `beluga providers --output json` → stderr empty, exit 0. `jq '.'` parses stdout successfully. JSON is a 4-element array of `{category, providers}` as specified in the architect consultation §5.

**PASS.**

### 10. `BootstrapFromEnv` never called from `cmd/beluga/`

`grep -rn "BootstrapFromEnv" cmd/beluga/` returned **zero matches**. AC9 contract upheld.

**PASS.**

### 11. `os.Exit` discipline

`grep -rn "os\.Exit" cmd/beluga/` returned exactly one code occurrence:

```
cmd/beluga/main.go:13:func main() { os.Exit(Execute(os.Stdout, os.Stderr)) }
```

(A second match in `root.go:44` is inside a doc comment, not code.) Every subcommand's `RunE` returns errors; cobra handles exit via the `Execute` return value. Architect consultation risk item 9 upheld.

**PASS.**

### 12. Env-var read scope

`grep -rn "os\.Getenv\|os\.LookupEnv" cmd/beluga/ o11y/bootstrap.go`:

```
o11y/bootstrap.go:36:	_ = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
o11y/bootstrap.go:37:	_ = os.Getenv("BELUGA_OTEL_STDOUT")
```

No other reads in `cmd/beluga/` (pre-existing or new). The two env vars match exactly the contract documented in the architect plan §2 and the brief. Neither is a secret; both are logged only in dev-mode tracing (future S3+), not in S1.

**PASS.**

### 13. Action pins in `smoke-install` / `release.yml`

Reviewed the new `smoke-install` job (`release.yml:184-224`):

- `uses: actions/setup-go@v6` — pinned to major.
- Existing jobs' pins (`actions/checkout@v6`, `goreleaser/goreleaser-action@v7`, `orhun/git-cliff-action@v4`, `peter-evans/repository-dispatch@v4`) are all major-version-pinned; `peter-evans/create-pull-request@c0f553fe549906ede9cf27b5156039d195d2ece0` is SHA-pinned (pre-existing, stronger than the new job requires).

No `@main`, `@master`, or `@latest` action refs anywhere in the workflow. `GO_VERSION` is pinned to `"1.25.9"` at `env.GO_VERSION` and referenced via `${{ env.GO_VERSION }}` in each setup-go invocation.

**PASS.**

### 14. Goreleaser ldflags safety

`.goreleaser.yml:22-25` references:

```
-X github.com/lookatitude/beluga-ai/cmd/beluga/internal/version.Version={{.Version}}
-X github.com/lookatitude/beluga-ai/cmd/beluga/internal/version.Commit={{.Commit}}
-X github.com/lookatitude/beluga-ai/cmd/beluga/internal/version.Date={{.Date}}
```

Inspecting `cmd/beluga/internal/version/version.go:19-23`:

```go
var (
    Version = ""
    Commit  = ""
    Date    = ""
)
```

All three are package-level `string` vars (untyped assignment to `""` infers `string`). Go's linker `-X` directive only works on string package-level vars — the contract is satisfied. No exported setter functions are needed or present.

**PASS.**

### 15. Architect's 10-item risk checklist (consultation §"Risks / Notes for reviewer-security")

| # | Item | Status | Evidence |
|---|------|--------|----------|
| 1 | No new auth boundary | **PASS** | `version` reads `runtime/debug.ReadBuildInfo()`; `providers` reads in-memory `List()` slices. No network, no auth, no tokens. |
| 2 | No new file I/O beyond pre-existing `init` | **PASS** | `version.go` and `providers.go` (CLI) do no file I/O. `init.go`'s path-traversal guard preserved — see check 7. |
| 3 | No network calls in S1 subcommands | **PASS** | Grep of new CLI files shows no `http.`, `net.Dial`, or provider-SDK constructor call. `BootstrapFromEnv` is inert in S1 (see check 10). |
| 4 | Env-var reads limited; not logged | **PASS** | See check 12. No `slog.Info` / `fmt.Printf` of env contents in the new code. |
| 5 | gosec G204 surface unchanged | **PASS** | See check 8. `#nosec G204` annotation and the `validPkgPattern` regex both preserved. |
| 6 | Cobra supply chain clean | **PASS** | See check 6. cobra v1.10.2, pflag v1.0.10, mousetrap v1.1.0 — all at current tagged releases, no retracts. |
| 7 | Provider blank imports have no side effects beyond `Register()` | **PASS** | Grepped each provider's `init()`: `llm/providers/anthropic`, `llm/providers/ollama`, `llm/providers/openai`, `rag/embedding/providers/ollama`, `rag/embedding/providers/openai`, `rag/vectorstore/providers/inmemory` each have a single `init()` whose body is one `Register(name, factoryFunc)` call. `memory/stores/inmemory` has **no** `init()` — the package exposes constructors only; `providers/providers_test.go:16-22` documents this as a known gap tracked outside S1 (note: the CGo-free audit contract listed the seven imports verbatim from the brief, so the inclusion is intentional). No arbitrary code executes at import time. |
| 8 | JSON output is not user-controlled | **PASS** | `providerCategory` values come from registry `List()` strings (framework constants). `--output` flag value is routed through a `switch` with an allowlist (`"json"`, `""`, `"text"`, default → error). No user input reaches `json.Encode` contents. |
| 9 | Exit-code discipline | **PASS** | See check 11. |
| 10 | `smoke-install` uses pinned `setup-go@v6` and `GO_VERSION` | **PASS** | See check 13. |

All 10 risk items confirmed as addressed.

## Pre-existing issues (verified NOT in touched files)

The developer's hand-off noted five pre-existing findings. Each verified:

- **gosec G101×2 at `o11y/tracer.go:33,36`** — not in diff (`git diff --name-only main...HEAD | grep tracer` returns nothing).
- **gosec G115 + G404 at `o11y/providers/phoenix/phoenix.go:181`** — not in diff.
- **govulncheck 1.26-vs-1.25 toolchain mismatch** — tooling issue, not a source finding.
- **golangci-lint v2 config migration debt** — config file, not in diff (no `.golangci.yml` change in this PR).
- **`metacognitive/store.go` race** — not in diff.
- **~100 gofmt-dirty files** — none in the touched file list (all touched `.go` files are gofmt-clean per `gofmt -l` being a global check; the developer's pre-commit gate documents the pre-existing state).
- **Pre-existing DOC-18 `eval → agent` layering violation** — not in diff; explicitly called out in the architect consultation risk item 10 as deferred.

None of the pre-existing issues are in files this PR modifies.

## Findings

No findings. Pass clean.

## Pass status

- [x] Pass 1 clean (0 new issues) — counter advanced to **1/2**.

## Recommended next action

Run Pass 2. The counter is at 1/2; one more consecutive clean pass is required before reviewer-qa runs.
