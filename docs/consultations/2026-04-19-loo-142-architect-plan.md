---
agent: architect (framework-layer)
feature: DX-1 S1 — CLI Foundation (cobra + beluga version + beluga providers)
linear: LOO-142 (framework sub-issue of LOO-141)
brief: research/briefs/2026-04-19-dx1-s1-cli-foundation.md
branch: feat/loo-142-cli-foundation-cobra-version-providers
date: 2026-04-19
consults:
  - research/briefs/2026-04-19-dx1-s1-cli-foundation/specialist-systems-architect.md
  - research/briefs/2026-04-19-dx1-s1-cli-foundation/specialist-devops-expert.md
  - research/briefs/2026-04-19-dx1-s1-cli-foundation/specialist-observability-expert.md
---

## Design: DX-1 S1 CLI Foundation

The approved brief is exhaustive and is the source of truth. This consultation converts its 17-file plan into a Red/Green TDD-friendly task sequence, pins down the Go signatures that the brief describes in prose, maps each of the 10 success criteria to concrete tests, and flags layering / security surfaces for the downstream reviewers. No new architectural decisions are introduced beyond those already locked in by the brief.

---

### Decisions

No new architectural decisions beyond the brief. Four framework-local execution choices that are NOT in the brief and that the developer-go agent must follow:

1. **Go toolchain target: `go 1.25.9`.** The highest published 1.25.x patch as of 2026-04-19 per the Go module proxy toolchain index (`curl -sL https://proxy.golang.org/golang.org/toolchain/@v/list | grep -oE 'go1\.25\.[0-9]+' | sort -uV | tail`). `go.mod` and `.github/workflows/release.yml` `env.GO_VERSION` must be updated in **one commit** (T1 below) — they are two halves of the same truth and a split commit leaves a window where local and CI use different toolchains. Cobra v1.8+ requires Go 1.21 minimum — 1.25.9 is comfortably above floor.
2. **RunE error convention: `fmt.Errorf(... %w, err)` for CLI-local errors; `core.Error` only when wrapping framework calls that already return `core.Error`.** The brief does not specify. `core.Error` belongs to the capability layer (LLM/tool/retry decisions). CLI-local errors (flag parse, path traversal, unknown target) are operator-facing strings — `fmt.Errorf` with `%w` preserves the chain without dragging the Layer 1 error taxonomy into Layer 7. Existing `cmd/beluga/*.go` files already follow this pattern (see `init.go:15`, `test.go:54`); keep it.
3. **Cobra `Args` policy.** `version` and `providers` take zero positional args — use `cobra.NoArgs`. `init`/`dev`/`test`/`deploy` preserve their existing "any extra args are flag parse errors" semantics — use `cobra.NoArgs` for them too (none of them accept positional arguments today). This was not explicit in the brief; stating it here so tests can assert it.
4. **Shared `newRootCmd()` constructor, no package-level `rootCmd` global.** Tests need a fresh command tree per invocation so they can capture output and re-parse args without state leaking between table entries. The `main()` function is a one-liner that does `os.Exit(cli.Execute())` where `cli.Execute()` calls `newRootCmd().Execute()`. This preserves the existing testability of `run()` (see `main_test.go:189-297`) after the migration.

No ADR is written. The brief itself is the external design artefact (per agent definition: "every binding decision gets an ADR" — all binding decisions here are already recorded in the brief). The four items above are execution conventions, not architectural commitments; they are documented here and do not rise to ADR status. If during implementation a genuinely new architectural question emerges (e.g., a cobra version upgrade breaks a rule), developer-go should bounce it back to the architect before an ADR is retrofitted.

---

### Interface Definitions

#### 1. `cmd/beluga/internal/version/version.go`

```go
// Package version exposes the framework version baked into the beluga CLI.
//
// Precedence: ldflags -X > build info (go install @vX.Y.Z) > "(devel)".
package version

import "runtime/debug"

// Version is overridden at link time by goreleaser:
//
//	-ldflags "-X github.com/lookatitude/beluga-ai/cmd/beluga/internal/version.Version=v1.2.3"
//
// When not set (local go build, ad-hoc compiles), Get falls back to the
// module version from runtime/debug.ReadBuildInfo, which is populated by
// `go install github.com/lookatitude/beluga-ai/cmd/beluga@vX.Y.Z`. When
// neither is set (ephemeral `go run` or a cold compile in the framework
// repo), Get returns "(devel)".
var (
	Version = ""
	Commit  = ""
	Date    = ""
)

// Get returns the resolved version string. See package doc for precedence.
func Get() string {
	if Version != "" {
		return Version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return "(devel)"
}
```

No exported constructor, no interface — this is a leaf package. `internal/` placement keeps it out of the framework's public surface.

#### 2. `o11y/bootstrap.go` (skeleton, not called in S1)

```go
package o11y

import (
	"context"
	"os"
)

// BootstrapFromEnv configures the global OTel tracer provider from the
// standard OTEL_* environment variables and returns a shutdown function.
//
// S1 ships this as a skeleton: no S1 subcommand invokes it. S3+ subcommands
// (beluga run, beluga dev, beluga eval) will call it from their RunE. It
// lives in o11y/ (not cmd/) so application developers writing their own
// main.go can reuse the same one-call bootstrap.
//
// Behavior:
//   - If OTEL_EXPORTER_OTLP_ENDPOINT is set, the SDK's auto-configuration
//     dials an OTLP exporter.
//   - If BELUGA_OTEL_STDOUT=1 is set and no OTLP endpoint, fall back to a
//     stdout JSON exporter (useful for local debugging).
//   - Otherwise, no exporter is attached; spans become no-ops silently.
//
// The returned shutdown function is always safe to call (nil-safe). A
// non-nil error means SDK initialisation failed — callers should log and
// continue rather than hard-fail.
//
// Additional TracerOption values override the env-derived configuration
// and compose with the existing WithSpanExporter / WithSampler / WithSyncExport
// options in tracer.go.
func BootstrapFromEnv(ctx context.Context, serviceName string, opts ...TracerOption) (shutdown func(), err error) {
	_ = ctx
	_ = serviceName

	// S1 skeleton: read env to validate the contract compiles; do not attach
	// an exporter. S3+ will wire the real OTLP exporter behind these env vars.
	_ = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	_ = os.Getenv("BELUGA_OTEL_STDOUT")

	_ = opts // reserved; composes with existing tracerConfig in S3+.

	// Nil-safe no-op. Callers always `defer shutdown()`.
	return func() {}, nil
}
```

Nil-safe Shutdown is a hard requirement (success criterion 9): the returned closure must never panic even if called twice. The skeleton above satisfies this trivially.

#### 3. `cmd/beluga/root.go` (cobra root)

```go
package main

import (
	"io"

	"github.com/spf13/cobra"
)

// newRootCmd builds the beluga root command tree. Constructed per-call so
// tests can capture output and parse fresh args without state leakage.
func newRootCmd() *cobra.Command {
	var (
		logLevel string // persistent flag; S1 recognised but not wired to slog
		output   string // persistent flag; consumed by `providers` in S1
	)

	root := &cobra.Command{
		Use:           "beluga",
		Short:         "Beluga AI CLI — scaffold, run, and operate agents",
		Long:          "beluga is the official command-line tool for the Beluga AI framework.",
		SilenceUsage:  true, // RunE errors print once via the top-level handler
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&logLevel, "log-level", "info",
		"log level (debug, info, warn, error) — written to stderr")
	root.PersistentFlags().StringVarP(&output, "output", "o", "",
		`output format for machine-readable commands (e.g. "json")`)

	root.AddCommand(
		newVersionCmd(),
		newProvidersCmd(),
		newInitCmd(),
		newDevCmd(),
		newTestCmd(),
		newDeployCmd(),
	)
	return root
}

// Execute is the entry point called by main(). It returns an exit code so
// tests can exercise error paths without os.Exit.
func Execute(stdout, stderr io.Writer) int {
	cmd := newRootCmd()
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	if err := cmd.Execute(); err != nil {
		// cobra's SilenceErrors means we own the stderr write.
		_, _ = stderr.Write([]byte("error: " + err.Error() + "\n"))
		return 1
	}
	return 0
}
```

`main.go` reduces to:

```go
package main

import (
	"os"

	_ "github.com/lookatitude/beluga-ai/cmd/beluga/providers" // trigger provider init()
)

func main() { os.Exit(Execute(os.Stdout, os.Stderr)) }
```

#### 4. `cmd/beluga/version.go` (cobra subcommand)

```go
package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/lookatitude/beluga-ai/cmd/beluga/internal/version"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/memory"
	"github.com/lookatitude/beluga-ai/rag/embedding"
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "version",
		Short:         "Print framework version, Go runtime, and provider counts",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "beluga %s\n", version.Get())
			fmt.Fprintf(w, "go %s\n", runtime.Version()[2:]) // strip "go" prefix — keep one line with "go1." visible
			fmt.Fprintf(w, "providers: llm=%d embedding=%d vectorstore=%d memory=%d\n",
				len(llm.List()), len(embedding.List()),
				len(vectorstore.List()), len(memory.List()))
			return nil
		},
	}
}
```

The brief requires the output to contain a framework version string, a `go1.` prefix, and provider category counts (AC2). The format above puts `beluga <ver>` on line 1, `go <ver>` on line 2 (still contains `go1.` after the first line's version number), and a single `providers:` line with four named counts. `runtime.Version()` returns e.g. `"go1.25.9"` — the call `runtime.Version()[2:]` is cosmetic; if the test needs the literal `go1.` prefix visible, keep `runtime.Version()` unmodified and print `go%s` without the slice. **Developer decision:** pick the unmodified form (`fmt.Fprintf(w, "%s\n", runtime.Version())`) to satisfy AC2 unambiguously. The example above shows the slice only to illustrate the choice.

#### 5. `cmd/beluga/providers.go` (cobra subcommand) — JSON schema

```go
package main

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/memory"
	"github.com/lookatitude/beluga-ai/rag/embedding"
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
)

// providerCategory is the JSON element emitted by `beluga providers --output json`.
type providerCategory struct {
	Category  string   `json:"category"`
	Providers []string `json:"providers"`
}

func newProvidersCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "providers",
		Short:         "List providers compiled into this binary",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cats := []providerCategory{
				{"llm", llm.List()},
				{"embedding", embedding.List()},
				{"vectorstore", vectorstore.List()},
				{"memory", memory.List()},
			}

			format, _ := cmd.Flags().GetString("output")
			w := cmd.OutOrStdout()

			switch format {
			case "json":
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(cats)
			case "", "text":
				tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
				for _, c := range cats {
					for _, p := range c.Providers {
						fmt.Fprintf(tw, "%s\t%s\n", c.Category, p)
					}
				}
				return tw.Flush()
			default:
				return fmt.Errorf("unsupported output format: %q (supported: text, json)", format)
			}
		},
	}
}
```

**Stable JSON schema** (this is the contract for the `--output json` path that CI consumers parse):

```json
[
  {"category": "llm",         "providers": ["anthropic", "ollama", "openai"]},
  {"category": "embedding",   "providers": ["ollama", "openai"]},
  {"category": "vectorstore", "providers": ["inmemory"]},
  {"category": "memory",      "providers": ["inmemory"]}
]
```

- Top-level: JSON array (not an object) so consumers can do `jq '.[] | select(.category=="llm") | .providers'`.
- Each element: `{ "category": <string>, "providers": <string[]> }`.
- `providers` is sorted alphabetically (inherited from each registry's `sort.Strings` in `List()` — verified in `llm/registry.go:49`, `rag/embedding/registry.go:49`, `rag/vectorstore/registry.go:49`, `memory/memory.go:71`).
- Stderr is empty on the success path; errors (unsupported format) go to stderr with exit 1.

#### 6. `cmd/beluga/providers/providers.go` (blank imports, exact verbatim from brief)

```go
// Package providers is a side-effect-only package that triggers init()
// registration for the curated set of providers shipped with the beluga CLI.
//
// The providers listed here MUST be CGo-free. CGO_ENABLED=0 is set in the
// goreleaser build; any CGo dependency will silently break cross-compilation
// on the CI runner. Each addition to this list requires an explicit audit —
// check the provider's imports (and its transitive SDK imports) for `import
// "C"` before adding it here. See docs/consultations/2026-04-19-loo-142-architect-plan.md
// (risks for reviewer-security).
package providers

import (
	_ "github.com/lookatitude/beluga-ai/llm/providers/anthropic"
	_ "github.com/lookatitude/beluga-ai/llm/providers/ollama"
	_ "github.com/lookatitude/beluga-ai/llm/providers/openai"
	_ "github.com/lookatitude/beluga-ai/memory/stores/inmemory"
	_ "github.com/lookatitude/beluga-ai/rag/embedding/providers/ollama"
	_ "github.com/lookatitude/beluga-ai/rag/embedding/providers/openai"
	_ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/inmemory"
)
```

The seven blank imports are exactly those listed in the brief. The comment block is new and makes the CGo-free invariant enforceable in code review.

---

### Implementation Plan

Tasks are ordered for Red/Green TDD — each "new behavior" task adds tests before the code that makes them pass. The cobra migration of existing subcommands (T3) is the highest-risk step because it touches 6 files with existing test coverage; it is broken out from T2 so the cobra root lands green against the existing dispatch stubs before the RunE rewrites begin.

Each task lists: description, files touched, deps, and acceptance criteria mapped to the brief's AC indices (AC1–AC10, in brief section order).

#### T1: Bump Go toolchain to 1.25.9 in lockstep

- **Description:** Update `go.mod` (`go 1.25.9`) and `.github/workflows/release.yml` (`env.GO_VERSION: "1.25.9"`) in one commit. Run the full pre-commit gate (`go build ./...`, `go vet ./...`, `go test -race ./...`, `go mod tidy`, `golangci-lint`, `gosec`, `govulncheck`) after the bump to confirm no downstream break.
- **Files:** `framework/go.mod`, `framework/.github/workflows/release.yml`.
- **Deps:** none.
- **AC:** None of the brief's AC rely on the bump directly, but AC6 (pre-commit gates pass on feature branch) requires the chosen toolchain to be installable on the CI runner.

#### T2: Add cobra dependency and thin cobra root

- **Description (test-first):** Add a test `TestRoot_Help` and `TestRoot_UnknownCommand` that invoke `Execute(stdout, stderr)` and assert help text on `[]string{}` and non-zero exit on `beluga bogus`. Preserve the existing `TestRun_*` tests by keeping `run()` as a thin adapter that shells into `Execute()` — or replace `run()` with `Execute()` signature-compatibly. **Green:** `go get github.com/spf13/cobra`, create `root.go` per the interface definition, rewrite `main.go` to a one-liner calling `Execute`. Existing subcommand dispatch remains through `cmdInit`/`cmdDev`/`cmdTest`/`cmdDeploy` stub funcs wired into cobra `RunE` via thin adapters (e.g., `RunE: func(cmd, args) error { return cmdInit(args) }`). This keeps T3's migration isolated.
- **Files (modify):** `framework/go.mod`, `framework/go.sum`, `framework/cmd/beluga/main.go`, `framework/cmd/beluga/main_test.go`.
- **Files (create):** `framework/cmd/beluga/root.go`.
- **Deps:** T1.
- **AC mapping:**
  - AC5 (existing subcommands continue to pass): green on `go test ./cmd/beluga/... -race` — adapter still calls the old `cmdInit`/`cmdDev`/`cmdTest`/`cmdDeploy`.
  - AC6 (pre-commit gates): `go mod tidy` must produce a clean diff (i.e., cobra + its transitive deps must end up in `go.mod`/`go.sum`).

#### T3: Migrate init/dev/test/deploy to native cobra RunE

- **Description (test-first):** Update `main_test.go` to exercise commands via `rootCmd.SetArgs([]string{"init", "-name", ...})` + `rootCmd.Execute()`. Preserve every existing assertion on behavior (path traversal rejection, flag parsing errors, success output strings). **Green:** convert each of `init.go`/`dev.go`/`test.go`/`deploy.go` from `flag.FlagSet` to cobra `*cobra.Command` with `RunE`. Flag names MUST be preserved exactly (`-name`, `-dir`, `-port`, `-config`, `-target`, `-output`, `-pkg`, `-v`, `-race`) to satisfy AC5's "no flags or behaviors regressed". The pre-existing `execCommand`/`lookPath` test stubs must still work — do not inline them.
- **Files (modify):** `framework/cmd/beluga/init.go`, `framework/cmd/beluga/dev.go`, `framework/cmd/beluga/test.go`, `framework/cmd/beluga/deploy.go`, `framework/cmd/beluga/main_test.go`.
- **Deps:** T2.
- **AC mapping:**
  - **AC5 (all four existing subcommands pass, no flag/behavior regression):** every pre-existing test case (`TestCmdInit`, `TestCmdInit_DefaultName`, `TestCmdInit_PathTraversal`, `TestCmdInit_RelativeTraversal`, `TestCmdDev`, `TestCmdDeploy`, `TestCmdTest_InvalidPkgPattern`, `TestCmdTest_ParseError`, `TestCmdTest_LookPathFailure`, `TestCmdTest_Success`, `TestCmdTest_RunFailure`, `TestRun_Init`, `TestRun_Dev`, `TestRun_Deploy`, `TestRun_DeployError`, `TestRun_Test`) must still pass after migration.
  - AC6.
- **Risk watch:** cobra's default flag-parse error output goes to stderr with usage text; existing tests (e.g., `TestCmdTest_ParseError`) assert on an error return, not output. `SilenceUsage: true` on each subcommand prevents usage spam on error and keeps existing stderr assertions valid.

#### T4: Framework version package + `beluga version` subcommand

- **Description (test-first):** Write `internal/version/version_test.go` covering: (a) `Version=""` + no build info → returns `"(devel)"`; (b) `Version="v1.2.3"` → returns `"v1.2.3"`; (c) precedence of ldflags over build info. Write a `TestVersionCommand` in `main_test.go` that invokes `beluga version` and asserts stdout contains `"beluga "`, `"go1."`, and `"providers:"`. **Green:** create `internal/version/version.go` per the interface definition; create `version.go` per the interface definition; wire it into `newRootCmd()`.
- **Files (create):** `framework/cmd/beluga/internal/version/version.go`, `framework/cmd/beluga/internal/version/version_test.go`, `framework/cmd/beluga/version.go`.
- **Files (modify):** `framework/cmd/beluga/root.go` (register subcommand), `framework/cmd/beluga/main_test.go` (extend).
- **Deps:** T2. Does NOT depend on T5/T6; the version subcommand can print counts by calling `llm.List()` etc. — those return `[]string{}` if no providers are registered yet. That is acceptable for T4 in isolation, but the eventual counts in the shipped binary require T5 to land first.
- **AC mapping:**
  - **AC2 (`beluga version` prints framework version, `go1.` prefix, provider counts; exit 0):** `TestVersionCommand`.
  - AC6.

#### T5: Curated provider blank imports

- **Description (test-first):** Write a test at `cmd/beluga/providers/providers_test.go` that imports the package (transitively via a test in the same package — the blank imports don't need a consumer) and asserts `llm.List()` contains `"anthropic"`, `"ollama"`, `"openai"`, and `embedding.List()` contains `"ollama"`, `"openai"`, and `vectorstore.List()` contains `"inmemory"`, and `memory.List()` contains `"inmemory"`. **Green:** create `cmd/beluga/providers/providers.go` with the exact 7 blank imports (verbatim from brief Architecture section) plus the CGo-free invariant comment.
- **Files (create):** `framework/cmd/beluga/providers/providers.go`, `framework/cmd/beluga/providers/providers_test.go`.
- **Files (modify):** `framework/cmd/beluga/main.go` (blank import of the providers package — already in T2's stub; confirm).
- **Deps:** T2.
- **AC mapping:**
  - **AC3 (`beluga providers` lists the 7 curated providers):** combined with T6.
  - AC6.

#### T6: `beluga providers` subcommand with `--output json`

- **Description (test-first):** Add three tests:
  1. `TestProvidersCommand_Human` — invoke `beluga providers`, assert stdout contains all 7 provider names in 4 category rows, stderr empty, exit 0.
  2. `TestProvidersCommand_JSON` — invoke `beluga providers --output json`, decode stdout via `json.Unmarshal` into `[]providerCategory`, assert 4 entries in canonical order (`llm, embedding, vectorstore, memory`), assert slice contents match registries, assert stderr empty, exit 0.
  3. `TestProvidersCommand_UnsupportedFormat` — invoke `beluga providers --output yaml`, assert error return with "unsupported output format" substring, exit 1.
  **Green:** create `providers.go` per the interface definition; wire into `newRootCmd()`.
- **Files (create):** `framework/cmd/beluga/providers.go`.
- **Files (modify):** `framework/cmd/beluga/root.go` (register subcommand), `framework/cmd/beluga/main_test.go` (extend).
- **Deps:** T5 (registry content), T2 (root + `--output` persistent flag).
- **AC mapping:**
  - **AC3 (human-readable lists 7 providers, stderr empty):** `TestProvidersCommand_Human`.
  - **AC4 (`--output json` emits valid JSON to stdout parseable by jq; stderr empty; exit 0):** `TestProvidersCommand_JSON`.
  - AC6.

#### T7: `o11y.BootstrapFromEnv` skeleton

- **Description (test-first):** Write `o11y/bootstrap_test.go` with three cases: (a) env clean → returns non-nil shutdown, nil error, shutdown is safe to call twice; (b) `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318` set → returns non-nil shutdown, nil error (SDK dial-on-send means no failure in S1's skeleton); (c) optional `TracerOption` args don't panic when provided. **Green:** create `o11y/bootstrap.go` per the interface definition.
- **Files (create):** `framework/o11y/bootstrap.go`, `framework/o11y/bootstrap_test.go`.
- **Deps:** none (layer-1 file, no CLI coupling).
- **AC mapping:**
  - **AC9 (`o11y/bootstrap.go` compiles, `BootstrapFromEnv` exported, not called by any S1 CLI subcommand):** `go build ./o11y/...` + `go vet ./o11y/...` + absence of `BootstrapFromEnv(` in `cmd/beluga/` (grep assertion in PR description, not a code test).
  - AC6.

#### T8: Goreleaser config — builds + archives + checksum

- **Description:** Replace the `builds: [{skip: true}]` entry in `.goreleaser.yml` with the 5-target build block from the brief Architecture section. Add `archives` (zip for windows, tar.gz otherwise) and `checksum` (sha256) blocks. Preserve the existing `release` and `changelog` blocks. Validate locally with `goreleaser check` if installed, else rely on CI.
- **Files (modify):** `framework/.goreleaser.yml`.
- **Deps:** T4 (ldflags `-X` references `.../internal/version.Version` etc., which must exist).
- **AC mapping:**
  - **AC7 (5 cross-platform binaries + `checksums.txt` on tag push):** verified only at release time; `goreleaser check` locally is the dry-run gate. The PR can verify syntactic validity but not full build — CI tag-push is the integration test.
  - AC6 (yaml lint as applicable).
- **No test:** goreleaser config is not exercised by `go test`. The validation is `goreleaser check` (if available locally) plus the first tag push post-merge.

#### T9: CI smoke-install job

- **Description:** Add a new job `smoke-install` to `.github/workflows/release.yml` after the `release` job and before `docs-bundle`. Configure `needs: [release]`, `if: needs.release.result == 'success'`, `continue-on-error: true`, include a `sleep 30` to absorb proxy indexing lag, then `go install github.com/lookatitude/beluga-ai/cmd/beluga@${RELEASE_TAG}`, then `beluga version` and grep for the tag string (minus the leading `v`). Use `actions/setup-go@v6` with `${{ env.GO_VERSION }}`.
- **Files (modify):** `framework/.github/workflows/release.yml`.
- **Deps:** T4 (version output must contain the tag), T8 (goreleaser must successfully publish binaries on tag push — although `go install` itself does not depend on goreleaser, the assertion that `beluga version` outputs the tag only works if T4 landed).
- **AC mapping:**
  - **AC1 (`go install ...@tag` succeeds on clean ubuntu-latest):** smoke-install job exercises this on every tag; `continue-on-error` makes it informational in S1.
  - **AC8 (smoke-install executes after release, asserts tag string):** the job itself.
  - AC6 (yaml lint).

#### T10: Docs — CLI reference and architecture overview

- **Description:** Add a small section to `docs/architecture/01-overview.md` under Layer 7 mentioning the `beluga` CLI with the `go install` path and listing S1 subcommands. Optionally create `docs/reference/cli.md` with a more detailed subcommand reference (the brief allows either — pick one). Update `cmd/beluga/doc.go` to list the new subcommands (`version`, `providers`) alongside the existing four.
- **Files (modify):** `framework/docs/architecture/01-overview.md`, `framework/cmd/beluga/doc.go`.
- **Files (create, optional):** `framework/docs/reference/cli.md`.
- **Deps:** T4, T6 (subcommands must exist to be documented accurately).
- **AC mapping:**
  - **AC10 (docs reference the `beluga` CLI with install path and S1 subcommands):** content of the modified docs files.
- **No test:** docs changes are verified by review, not by `go test`.

---

### Success-criteria matrix

| AC  | Success criterion (paraphrased)                                                                       | Verifying task(s)   | Test / verification command                                                                                                   |
|-----|--------------------------------------------------------------------------------------------------------|---------------------|-------------------------------------------------------------------------------------------------------------------------------|
| AC1 | `go install .../cmd/beluga@tag` succeeds on clean ubuntu-latest                                       | T9                  | `smoke-install` job (release.yml) runs `go install` on a fresh runner — informational in S1, gated after 2 clean releases.     |
| AC2 | `beluga version` prints framework version, `go1.` prefix, provider counts; exit 0                      | T4                  | `TestVersionCommand` (cmd/beluga/main_test.go): `strings.Contains(stdout, "beluga ")`, `"go1."`, `"providers:"`, `exitCode==0`. |
| AC3 | `beluga providers` lists 7 providers human-readable; exit 0; stderr empty                              | T5 + T6             | `TestProvidersCommand_Human`: iterate expected provider names, `strings.Contains(stdout, name)`; `stderr.Len()==0`.             |
| AC4 | `beluga providers --output json` emits valid JSON to stdout; stderr empty; exit 0                      | T6                  | `TestProvidersCommand_JSON`: `json.Unmarshal([]byte(stdout), &cats)` succeeds, `len(cats)==4`, `stderr.Len()==0`.                |
| AC5 | Existing `init`/`dev`/`test`/`deploy` tests pass after cobra migration; no flag/behavior regression    | T2 + T3             | Pre-existing `TestCmdInit*`, `TestCmdDev`, `TestCmdDeploy*`, `TestCmdTest*`, `TestRun_*` tests — all must still pass.           |
| AC6 | All pre-commit gates pass on the feature branch                                                        | all tasks           | `go build ./...`, `go vet ./...`, `go test -race ./...`, `go mod tidy` (clean diff), `gofmt -l`, `golangci-lint`, `gosec`, `govulncheck`. |
| AC7 | Goreleaser produces 5 cross-platform binaries + `checksums.txt` on tag push                            | T8                  | Tag push triggers `release` job; artifacts visible on GitHub Release page — verifiable only post-merge.                         |
| AC8 | `smoke-install` CI job runs after `release` and asserts `beluga version` contains tag                  | T9                  | Job definition in `release.yml`; observable run on first tag push after merge.                                                  |
| AC9 | `o11y/bootstrap.go` compiles with `BootstrapFromEnv` exported, not called by S1 CLI                    | T7                  | `go build ./o11y/...`; `grep -r "BootstrapFromEnv(" cmd/beluga/` returns zero matches (PR-body assertion).                      |
| AC10 | Docs reference the `beluga` CLI with `go install` path and S1 subcommands                              | T10                 | Review-time check of `docs/architecture/01-overview.md` (and `docs/reference/cli.md` if created).                               |

---

### Risks / Notes for developer-go

1. **Go toolchain: commit `go.mod` and `release.yml` in one commit (T1).** Splitting them leaves a window where CI resolves a different toolchain than local `go build`. Use `go 1.25.9`.
2. **RunE error style:** plain `fmt.Errorf("...: %w", err)` for CLI-local errors; do NOT introduce `core.Error` in `cmd/beluga/` — it belongs to the capability layer and is already applied by registry `New()` calls. Preserve the existing `cmdInit`/`cmdDev`/`cmdTest`/`cmdDeploy` error shape when migrating to cobra.
3. **Preserve `execCommand` and `lookPath` test indirection in `test.go`.** The current tests stub those package-level vars. Keep them as package-level vars even after converting `cmdTest` to a cobra `RunE` — otherwise the `TestCmdTest_Success`, `TestCmdTest_RunFailure`, `TestCmdTest_LookPathFailure` tests break (AC5 regression).
4. **Cobra `SilenceUsage: true` is mandatory.** Without it, every `RunE` error dumps the full usage text to stderr, which breaks the existing `TestCmdTest_ParseError` and `TestRun_DeployError` assertions that check for an `"error:"` prefix and nothing else.
5. **`TestRun_Test` currently passes `./...`** — after migration it routes through cobra. Confirm that the stubbed `execCommand` is still invoked (the stub swap must happen in the test _after_ `newRootCmd()` constructs the `cmd/beluga/test` command, but _before_ `Execute()` — because the var is package-level, the stub swap before `Execute()` suffices).
6. **`version` output must literally contain `go1.`** (AC2). Use `runtime.Version()` unmodified: it returns `"go1.25.9"`. Do NOT strip the `go` prefix.
7. **`providers --output json` contract.** Emit a JSON array, not a wrapping object. Indent 2 spaces (cosmetic; `jq` parses either). Array order is fixed: `llm`, `embedding`, `vectorstore`, `memory`. Provider names within each category come from `List()` and are already alphabetically sorted.
8. **Blank import for providers in `main.go`.** The line `_ "github.com/lookatitude/beluga-ai/cmd/beluga/providers"` MUST be in `main.go` (not `root.go`, not `providers.go`). If it's in a test-only file, `go test` passes but `go install` produces a binary with empty registries — AC3/AC4 would fail at release time only.
9. **Binary size audit:** after `go build -ldflags="-s -w" ./cmd/beluga` locally, `ls -lh` the binary. Expected 30–40 MB. If >60 MB, CGo has leaked in — stop and audit the 7 provider imports. This is a manual check; record the number in the PR body.
10. **Pre-existing DOC-18 layering violation — `eval → agent`.** Do NOT touch this in the DX-1 S1 PR. The PR body must explicitly note: "Pre-existing `eval → agent` Layer 3→6 upward import (DOC-18) is unchanged in this PR — to be addressed before S4 `beluga eval` ships."
11. **`o11y.BootstrapFromEnv` is not invoked in S1.** Do not add `_ = o11y.BootstrapFromEnv` or equivalent reference from `cmd/beluga/` — AC9 explicitly requires it to be unused in S1 CLI code.
12. **Cobra persistent flag `--output, -o`.** This flag is on the root command so `beluga providers --output json` and `beluga --output json providers` both work. The existing `deploy -output` flag conflicts with the short form `-o` — use `StringVarP(..., "output", "o", ...)` on root only, and on `deploy`, use `--output` as a local flag with no short form (or rename to `--output-dir`). Pre-existing tests reference `-output` via `deploy` — verify none use `-o`. (Checked: `main_test.go` uses only `-target`; no `-o`. Safe to add the root-level `-o`.)

### Risks / Notes for reviewer-security

The CLI surface in S1 is intentionally small. Enumerate the threat surfaces below and verify each:

1. **No new auth boundary.** `beluga version` and `beluga providers` do not authenticate, authorise, or call out to any network service. `version` reads `runtime/debug.ReadBuildInfo()` (stdlib, local). `providers` calls `llm.List()` / `embedding.List()` / `vectorstore.List()` / `memory.List()` — all read-only in-memory maps.
2. **No new file I/O beyond pre-existing `init`.** `init.go`'s path-traversal defence (lines 22–36) is preserved under the cobra migration; the existing `TestCmdInit_PathTraversal` and `TestCmdInit_RelativeTraversal` tests guard it. Verify no regression in the diff.
3. **No network calls in S1 subcommands.** `version` and `providers` are offline. `BootstrapFromEnv` reads `OTEL_EXPORTER_OTLP_ENDPOINT` but in S1 does not dial; even when it does (S3+), the SDK dial is opt-in via the env var. Confirm no `http.Get`, `net.Dial`, or SDK constructor call in S1 code.
4. **Env-var reads.** `BootstrapFromEnv` reads `OTEL_EXPORTER_OTLP_ENDPOINT` and `BELUGA_OTEL_STDOUT` only. Neither is a secret and neither is logged. Confirm no `slog.Info("env", ...)` leaks env contents.
5. **gosec G204 surface unchanged.** `test.go:23` already carries a `#nosec G204` with a justification comment — preserved by the cobra migration. `validPkgPattern` regex at `test.go:14` still guards `-pkg` values. Verify the regex and the `#nosec` annotation both survive T3.
6. **Supply-chain: cobra is a new direct dep.** `github.com/spf13/cobra` (and its transitive `github.com/spf13/pflag`, `github.com/inconshreveable/mousetrap`) — all well-known, maintained by Kubernetes SIGs and pflag's author respectively. Confirm `govulncheck ./...` returns clean post-T2.
7. **Provider blank imports are side-effect imports only.** No provider code executes at `cmd/beluga/providers/providers.go` init time beyond `Register()` calls into the framework's own registries. Confirm none of the 7 providers have `init()` side effects beyond `Register` (a grep for `init()` in each provider file during review is sufficient).
8. **JSON output is not user-controlled.** `beluga providers --output json` emits framework-constant registry names. No user-controlled input reaches the JSON encoder. No injection surface.
9. **Exit-code discipline.** `Execute` returns 0 on success, 1 on error. No `os.Exit` calls inside cobra `RunE` functions (cobra handles them). Confirm via grep of `cmd/beluga/*.go` (excluding `main.go`'s single `os.Exit(Execute(...))`).
10. **`smoke-install` job uses pinned `setup-go@v6` and pinned `GO_VERSION`.** No `@latest` on action pins. Confirm.

No new attack surface is introduced by S1 compared to the existing stdlib-flag CLI. The `init` path-traversal defence is the only non-trivial security check and is preserved.

---

### Specialist citations

- Systems-architect: `research/briefs/2026-04-19-dx1-s1-cli-foundation/specialist-systems-architect.md` — consulted for Layer 7 import clearance, cobra module boundary (Option A), and the `version.Get()` precedence chain.
- DevOps-expert: `research/briefs/2026-04-19-dx1-s1-cli-foundation/specialist-devops-expert.md` — consulted for the 5-target goreleaser build, `smoke-install` job shape, and `CGO_ENABLED=0` discipline.
- Observability-expert: `research/briefs/2026-04-19-dx1-s1-cli-foundation/specialist-observability-expert.md` — consulted for the stdout/stderr split, no-OTel-at-startup decision, and `o11y.BootstrapFromEnv` home (framework/o11y/, not cmd/).

No additional `/consult` bounces were needed during design. The brief and specialist inputs fully cover the S1 surface.
