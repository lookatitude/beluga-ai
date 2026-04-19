# Security Review — DX-1 S1 CLI Foundation — Pass 2
Branch: feat/loo-142-cli-foundation-cobra-version-providers
Reviewer: reviewer-security
Date: 2026-04-19

## Scope

Second independent review of the 10-commit diff on `feat/loo-142-cli-foundation-cobra-version-providers` vs `main` (24 files, +1513 / −185 LOC). The goal of Pass 2 is to re-check the change with a fresh heuristic — NOT to rubber-stamp Pass 1. I re-ran the automated gate, re-read every diff hunk, and explicitly exercised the ten Pass-2-specific angles in the dispatch note: hunk-level diff review, new exported-symbol audit, error surface scan, concurrency re-run, CI pipeline diff, goreleaser diff, binary sanity, deferred-scope leak check, AC-coverage verification, and "what did Pass 1 under-weight". Files in scope are only those in `git diff --name-only main...HEAD`; pre-existing findings outside that set are out of scope unless they now co-locate with a touched file.

## Automated checks

| Check | Command | Result |
|---|---|---|
| gosec (touched pkgs) | `gosec -quiet ./cmd/beluga/... ./o11y/...` | 4 findings (`o11y/tracer.go` G101×2 on `gen_ai.usage.*_tokens` constants; `o11y/providers/phoenix/phoenix.go` G115 + G404 on Phoenix session-id). **All four in untouched files.** None in files changed by this PR. `cmd/beluga/**` and `o11y/bootstrap.go` are clean. |
| go vet (touched pkgs) | `go vet ./cmd/beluga/... ./o11y/...` | Clean. |
| go test -race (touched pkgs) | `go test -race ./cmd/beluga/... ./o11y/...` | 167 passed in 9 packages. No new race detector hits. Metacognitive pre-existing race is not in scope (files untouched). |
| golangci-lint | `golangci-lint run ./cmd/beluga/... ./o11y/...` | Cannot run locally — documented pre-existing v2-config migration debt (`unsupported version of the configuration: ""`). Not introduced by this PR. CI canonical gate covers. |
| govulncheck | `govulncheck ./...` | Cannot run locally — documented pre-existing toolchain mismatch (govulncheck built against go1.25 but system `go list` is go1.26). Independently verified no cobra v1.10.2 / pflag v1.0.10 / mousetrap v1.1.0 CVEs on vuln.go.dev as of 2026-04-19. |
| go build -ldflags="-s -w" | `go build -ldflags="-s -w" -o /tmp/beluga-pass2 ./cmd/beluga` | Clean. 9.3 MB ELF, stripped (`file` reports "stripped"), confirms `-s -w` applied. |

Automated gate: clean for the scope of this PR.

## Manual checks — Pass-2-specific angles

### 1. Hunk-level diff review (not file-level)

Read every hunk in `git diff main...HEAD` for each of the 24 files. Attack-surface questions asked per hunk:

- **`cmd/beluga/main.go`** — collapsed from 60 LOC to 5. Only remaining attack surface is `os.Exit(Execute(os.Stdout, os.Stderr))` — entry-point delegating to cobra. Strictly smaller surface than pre-cobra.
- **`cmd/beluga/root.go`** (new) — constructs cobra tree per call; no global mutable state, no init-time registration from user input. `SilenceUsage`/`SilenceErrors` is deliberate (stderr write owned by `Execute`). The `%s` in `fmt.Fprintf(stderr, "error: %s\n", err.Error())` operates on a Go string — no format-string or terminal-escape vulnerability in Go's `fmt`.
- **`cmd/beluga/version.go`** (new) — reads `runtime.Version()` (host OS data) and four registry `List()` calls (in-memory slices). No file I/O, no network, no env reads. Safe.
- **`cmd/beluga/providers.go`** (new) — `--output` routed via an allowlist switch (`json`/`""`/`text` → default error). JSON encoder uses `encoding/json` on framework-controlled category strings and registry-controlled provider name strings. No user content reaches the encoder. `text` path uses `tabwriter` — also framework-controlled values.
- **`cmd/beluga/init.go`** — path-traversal defence at lines 41–55 (cleanDir, relDir, `strings.HasPrefix(relDir, "..")` rejection) byte-identical to pre-cobra; only flag-source variable renamed from `*dir` to `dir` (moved to `runInit(name, dir string)` helper).
- **`cmd/beluga/test.go`** — `#nosec G204` justification block and `validPkgPattern` regex preserved. One comment-only word change (`cmdTest` → `runTest`).
- **`cmd/beluga/dev.go`, `deploy.go`** — stubs; no file writes, no network, no exec. The `deploy --target` user input is quoted via `%s` inside `fmt.Errorf`; Go `fmt` neither interprets escapes nor opens a subprocess. Non-issue.
- **`cmd/beluga/doc.go`** — doc comment only.
- **`o11y/bootstrap.go`** (new) — skeleton. Two `os.Getenv` reads to validate the contract compiles; values are assigned to `_` and never logged, stored, or propagated. `_ = opts` explicitly discards options in S1. `shutdown` returned as `func(){}` — nil-safe, idempotent, test-verified.
- **`go.mod` / `go.sum`** — adds cobra v1.10.2, pflag v1.0.10, mousetrap v1.1.0, transitive test-only `cpuguy83/go-md2man/v2` and `russross/blackfriday/v2`. Verified all at tagged releases, no `replace`, no pseudo-versions, no retract directives. (`go-md2man` and `blackfriday` are cobra docs-gen deps; they land in `go.sum` but not in the production import graph — neither appears in `go list -deps ./cmd/beluga`.)

No hunk introduces a new attack surface beyond what the brief contemplates.

### 2. Symbol-level review of new exports

`grep '^func [A-Z]|^type [A-Z]|^var [A-Z]'` over touched files:

- **`cmd/beluga/root.go:45 — Execute(stdout, stderr io.Writer) int`** — public contract: documented in godoc comment (lines 42–44), returns exit code so tests avoid `os.Exit`. No interface to assert against.
- **`cmd/beluga/internal/version/version.go:26 — Get() string`** and the three package-level string vars `Version`, `Commit`, `Date` at lines 19–23. All documented in package godoc and the var block comment. `internal/` path makes this non-public to other modules. No compile-time interface assertion needed (pure value type).
- **`o11y/bootstrap.go:30 — BootstrapFromEnv(ctx context.Context, serviceName string, opts ...TracerOption) (shutdown func(), err error)`** — **signature exactly matches the brief**: first param `context.Context`, `serviceName string`, variadic `TracerOption`, return `(shutdown func(), err error)`. Documented in a 19-line godoc block covering behaviour (three exporter modes), shutdown semantics (nil-safe, idempotent), and composition rules with existing `TracerOption` values. No interface to implement — it is itself a top-level function.
- All other new identifiers in `cmd/beluga/` are lowercase unexported constructors (`newRootCmd`, `newVersionCmd`, etc.) or test functions.

No new exported symbol is undocumented. No new exported symbol implements an existing interface without a compile-time assertion — none of them implement Layer 3 interfaces.

### 3. Error surface

Full set of error producers in touched files:

```
cmd/beluga/init.go         7 × fmt.Errorf (all wrap with %w except "path traversal not allowed: %q <dir>")
cmd/beluga/providers.go    1 × fmt.Errorf ("unsupported output format: %q ...")
cmd/beluga/test.go         2 × fmt.Errorf ("invalid package pattern: %q <pkg>", "locate go toolchain: %w <err>")
cmd/beluga/deploy.go       1 × fmt.Errorf ("unknown deployment target: %s (supported: docker, compose, k8s)")
o11y/bootstrap.go          0 × (returns nil error only)
```

Checks:
- **Secret leak:** None. No error message references env-var contents, tokens, API keys, file contents, or stack frames.
- **Path leak:** `init.go:55` returns `"path traversal not allowed: %q <dir>"` where `<dir>` is user-supplied. `%q` safely quotes it. This is the error the caller INTENDED — echoing back the rejected input is the usability point. Not a leak of internal state.
- **`%w` vs raw:** Every `fmt.Errorf` that wraps an underlying error uses `%w` (the error-chain-preserving verb). Verified by eye across all 11 Errorf sites. None wrap with `%v` or `%s`.
- **Duplication with framework error types:** The touched files do not use `core.Error` — they are Layer 7 (application) CLI code. `core.Error` is intended for Layer 1–6 inter-package contracts (see `.claude/rules/security.md` §"Error Handling" and CLAUDE.md rule 5). CLI-level `fmt.Errorf` is the accepted pattern. No duplication.

### 4. Concurrency surface

Re-ran `go test -race ./cmd/beluga/... ./o11y/...` → 167 passed, 0 race detector hits. Touched code introduces no new goroutine, no shared mutable state, no unprotected map/struct access. Cobra is a per-call state container (each `executeArgs` builds a fresh tree via `newRootCmd()`), which makes test parallelism safe. The `metacognitive/store.go` race is pre-existing and in files this PR does not touch.

### 5. CI pipeline diff — `smoke-install` job

Re-read `.github/workflows/release.yml:176-224`:

- `needs: [release]` — correct. The job consumes `github.ref_name` (or `inputs.tag`) which is only valid once `release` has pushed the tag.
- `if: always() && needs.release.result == 'success'` — standard guard.
- `continue-on-error: true` — matches brief AC8 ("smoke-install is informational in S1; failures do not block release"). The architect consultation explicitly defers a hard gate to S2+ per devops specialist Q3.
- Tokens — no new token. The job does `go install` from the public module path `github.com/lookatitude/beluga-ai/cmd/beluga@<tag>`; no `GITHUB_TOKEN` required on a public repo. Verified no `secrets.*` references in the smoke-install step.
- **Shell injection check on `${RELEASE_TAG#v}`:** `RELEASE_TAG` is sourced from `${{ github.event_name == 'workflow_dispatch' && inputs.tag || github.ref_name }}`. Both are GitHub-controlled ref names — `github.ref_name` comes from the triggering ref, and `inputs.tag` is bound to the workflow_dispatch input schema. Neither is user-free-text without GitHub validation. The `${RELEASE_TAG#v}` parameter expansion and subsequent `grep -Fq "$EXPECTED"` are executed inside a `set -euo pipefail` bash step. `grep -F` treats the pattern as literal (no regex metacharacters), so even if a tag somehow contained `$`, `*`, or backticks, it cannot execute. Confirmed safe.
- Action pins: `actions/setup-go@v6`, `actions/checkout@v6`, `goreleaser/goreleaser-action@v7`, `orhun/git-cliff-action@v4`, `peter-evans/repository-dispatch@v4` all major-version-pinned. `peter-evans/create-pull-request@c0f553fe549906ede9cf27b5156039d195d2ece0` is SHA-pinned (pre-existing). No `@main`/`@master`/`@latest`.

All new CI surface is aligned with the brief and introduces no new secret or injection vector.

### 6. Goreleaser diff

Re-read `.goreleaser.yml` against the brief's 5-target build list:

- `CGO_ENABLED=0` set via `env:` — confirmed at top of `beluga-cli` build.
- `ldflags: -s -w` present — strips DWARF + symbol table. Binary inspection confirmed `file` reports "stripped".
- `-X github.com/lookatitude/beluga-ai/cmd/beluga/internal/version.{Version,Commit,Date}` — three `-X` flags target string vars in `cmd/beluga/internal/version/version.go:19-23`. Go's linker only accepts `-X` on package-level string vars; the contract holds.
- `ignore: - goos: windows, goarch: arm64` — present, matching the brief's explicit 5-target list (linux/{amd64,arm64}, darwin/{amd64,arm64}, windows/amd64).
- Archives block (`format_overrides: windows → zip`; `files: [LICENSE, README.md]`) — present.
- Checksum block (`algorithm: sha256`, `name_template: "checksums.txt"`) — present.
- **No cosign / signing configuration** — correctly deferred per brief §"Out of scope (deferred to S2+)".

Goreleaser surface is exactly what the brief specified, nothing more, nothing less.

### 7. Binary sanity check

Built locally with `go build -ldflags="-s -w" -o /tmp/beluga-pass2 ./cmd/beluga` and exercised:

```
$ /tmp/beluga-pass2 --help              → exit 0, full usage printed
$ /tmp/beluga-pass2 version             → exit 0, stderr empty, stdout contains "beluga ", "go1.", "providers: llm=3 embedding=2 vectorstore=1 memory=4"
$ /tmp/beluga-pass2 providers           → exit 0, stderr empty, 10 tabwriter rows (3 llm + 2 embedding + 1 vectorstore + 4 memory)
$ /tmp/beluga-pass2 providers --output json | python3 -c "..."  → parses as JSON, length 4
$ /tmp/beluga-pass2 providers --output yaml → exit 1, stderr = `error: unsupported output format: "yaml" (supported: text, json)`
$ /tmp/beluga-pass2 bogus               → exit 1, stderr = `error: unknown command "bogus" for "beluga"`
```

All seven behaviours match the brief's AC1–AC8 contract. Binary size 9.3 MB — reasonable for a cobra-based CLI with 7 provider packages statically linked.

### 8. Deferred-to-S2+ leak check

Brief §"Out of scope (deferred to S2+)" lists: `init` template expansion, `run`/`dev` fsnotify, `test` new functionality, `eval`, provider expansion beyond 7, OTel init-at-startup, cosign, windows/arm64, Homebrew. Verified:

- `git diff --name-only main...HEAD | grep -iE 'fsnotify|cosign|eval/|windows.*arm64'` → **zero matches**.
- `grep 'fsnotify' go.sum` → **zero matches** (no new `fsnotify` dep introduced).
- No `cmd/beluga/eval.go` or `cmd/beluga/run.go`.
- No `cosign` or `sigstore` in `.github/` or `.goreleaser.yml`.
- `cmd/beluga/providers/providers.go` — exactly 7 blank imports (llm: anthropic, ollama, openai; embedding: ollama, openai; vectorstore: inmemory; memory/stores: inmemory). Matches brief §3.1 verbatim.
- No `BootstrapFromEnv` call anywhere in `cmd/beluga/` — confirmed via grep.

No deferred scope leaked into this PR.

### 9. AC coverage — security-slanted

- **AC1 (`go install @latest` works)** — proxy indexing verified by `smoke-install` job post-tag (informational in S1 per AC8).
- **AC2 (version prints three substrings)** — `TestVersionCommand` (`main_test.go:229-244`) actively asserts `"beluga "`, `"go1."`, `"providers:"` substrings present in stdout, exit 0.
- **AC3/AC4 (providers text / JSON empty-stderr contract)** — `TestProvidersCommand_Human` (`main_test.go:251-253`) and `TestProvidersCommand_JSON` (`main_test.go:276-278`) both ACTIVELY ASSERT `errBuf != ""` → test failure. Not just documented; mechanically gated.
- **AC5 (flag regression)** — `TestCmdInit`, `TestCmdDev`, `TestCmdDeploy`, `TestCmdTest_*` cover every preserved flag.
- **AC6 (path traversal)** — `TestCmdInit_PathTraversal` and `TestCmdInit_RelativeTraversal` exercise the defence. Both pass.
- **AC7 (unsupported format error)** — `TestProvidersCommand_UnsupportedFormat` (`main_test.go:312-320`) asserts exit 1 + `"unsupported output format"` substring.
- **AC8 (smoke-install informational)** — `continue-on-error: true` in the workflow.
- **AC9 (BootstrapFromEnv never called in S1 CLI)** — `grep` on `cmd/beluga/` returned zero matches.
- **AC10 (tests are table-driven and `-race`-clean)** — `go test -race ./cmd/beluga/...` 167 passed.

Every AC is security-regression-free.

### 10. Angles Pass 1 under-weighted (now checked)

Pass 2 specifically emphasised these angles. Record here regardless of whether they turned up anything:

- **Symbol-level new-exports audit.** Checked every new `func A-Z`, `type A-Z`, `var A-Z` in touched files against the architect's brief. Only two public functions added — `Execute` and `BootstrapFromEnv`. Both documented. `BootstrapFromEnv` signature **byte-for-byte matches** the brief (first param `context.Context`, variadic `TracerOption`, return `(shutdown func(), err error)`). No drift from the contract.
- **Root-level `--output` persistent flag vs `deploy`'s local `--output` flag.** Verified shadowing behaviour: running `/tmp/beluga --output json deploy --target docker` passes `"json"` as `deploy`'s output directory (local flag wins over persistent when names collide, per cobra documented behaviour). This is a **UX quirk**, not a security flaw — `deploy` is a stub that never writes files in S1. The local `--output` flag has no short alias, so the root's `-o` short form is never ambiguous. Noting here for S2+ reviewer awareness: when `deploy` becomes functional in a later slice, this shadowing should be documented or the local flag renamed. **Not a Pass 2 finding.**
- **Terminal-escape injection via `%s` on user-controlled strings.** Confirmed Go's `fmt` package does not interpret control characters. Even if a caller passes `$'\x1b[2J'` as a `--target` value, the error message echoes it verbatim and the terminal renders whatever the terminal renders — this is identical to `echo "$VAR"` in bash. Not a framework concern. Not a Pass 2 finding.
- **Transitive test-only deps (`cpuguy83/go-md2man/v2`, `russross/blackfriday/v2`, `go.yaml.in/yaml/v3`) in `go.sum`.** Verified via `go list -deps ./cmd/beluga` that none of these appear in the production import graph — they are only pulled by `cobra/cobra`'s `docs` sub-package which is not imported. They land in `go.sum` by virtue of cobra's `go.mod` but do not ship in the binary.
- **`internal/version` package placement.** Confirmed the package lives under `cmd/beluga/internal/`, which per Go's import rules prevents it from being imported outside `cmd/beluga`. This enforces the architect's decision that version data is a leaf, not a reusable framework primitive.

None of these angles uncovered a new finding; all five resolved PASS.

## Verification of Pass 1's categorisation

Every pre-existing finding Pass 1 documented was independently re-checked for in-scope status:

| Pass 1 categorised as pre-existing | Pass 2 re-verification |
|---|---|
| gosec G101×2 on `o11y/tracer.go:33,36` | `tracer.go` not in `git diff --name-only main...HEAD`. Still pre-existing. |
| gosec G115+G404 on `o11y/providers/phoenix/phoenix.go:181` | `phoenix.go` not in diff. Still pre-existing. |
| govulncheck 1.26-vs-1.25 toolchain mismatch | Tooling issue, not a source finding. CI gate covers. |
| golangci-lint v2 config migration debt | `.golangci.yml` not in diff. Still pre-existing. |
| metacognitive/store.go race | `store.go` not in diff. Still pre-existing. |
| ~100 gofmt-dirty files | None of the 24 touched files are gofmt-dirty (verified: `gofmt -l` on the touched-file set returns empty). Pre-existing to those files only. |
| DOC-18 `eval → agent` layering violation | `eval/` not in diff. Still pre-existing. |

All Pass 1 categorisations hold. No Pass-1-classified pre-existing item is now in-scope.

## Findings

**No findings.** Pass 2 is clean.

Pass 2 also confirms Pass 1 did not miss anything — every angle re-checked independently returned the same PASS result, and the five Pass-2-specific angles (symbol-level exports audit, cobra flag-shadowing, terminal-escape `%s` analysis, transitive test-only deps, `internal/` package placement) all resolved PASS without finding anything Pass 1 overlooked.

## Pass status

- [x] Pass 2 clean (0 new issues).
- [x] All Pass 1 findings correctly categorised (pre-existing vs in-scope).
- [x] Consecutive-pass counter: **2/2 — security review complete.**

## Recommended next action

Advance to **reviewer-qa**. Two consecutive clean security passes achieved; the PR is unblocked from the security gate per `workflow.md` §"Security Review (2 clean passes required)".
