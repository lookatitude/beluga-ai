---
agent: architect (framework-layer)
feature: DX-1 S2 — CLI Scaffolding (beluga init + beluga new agent/tool/planner)
linear: LOO-149 (framework sub-issue of LOO-148)
brief: research/briefs/2026-04-19-dx1-s2-cli-scaffolding.md
branch: feat/loo-149-cli-scaffolding
date: 2026-04-20
consults:
  - research/briefs/2026-04-19-dx1-s2-cli-scaffolding/specialist-systems-architect.md
  - research/briefs/2026-04-19-dx1-s2-cli-scaffolding/specialist-security-architect.md
  - research/briefs/2026-04-19-dx1-s2-cli-scaffolding/specialist-devops-expert.md
  - research/briefs/2026-04-19-dx1-s2-cli-scaffolding/specialist-ai-ml-expert.md
  - research/briefs/2026-04-19-dx1-s2-cli-scaffolding/specialist-observability-expert.md
---

## Design: LOO-149 — CLI scaffolding (beluga init + beluga new)

S2 ships `beluga init <project-name>` and `beluga new agent|tool|planner <Name>` by introducing a self-contained `cmd/beluga/scaffold/` subpackage with a named-template registry (`Registry{map[string]fs.FS}`), a stdlib-only renderer that uses `strings.ReplaceAll` + `__BELUGA_*__` sentinels gated by `go/format.Source`, and one populated template (`basic`) structurally isomorphic to `examples/first-agent/main.go` — OpenAI + `gpt-4o-mini`, echo tool, implicit ReAct planner, no OTel wiring. The surface is 19 files: 5 Go source in `scaffold/` + 9 `.tmpl` files under `templates/basic/` + 1 golden directory + 4 Go source in `cmd/beluga/` (new `new*.go` + rewritten `init.go`) + 1 CI workflow + 1 doc subsection. The single highest-risk change is **deleting the 93-line `runInit` body in `cmd/beluga/init.go:37-130` in its entirety**: the pre-existing stub generates `main.go` missing `llm.New`, the provider blank import, and `config.ProviderConfig` — it will not compile against the canonical Layer 7 shape. There is no incremental migration path; it is a delete-and-replace (brief Risk #7, systems-architect §Risk #1).

---

### Decisions

No new architectural decisions beyond the brief. The following 19 bullets restate the brief's binding Decision-summary list so developer-go has them at hand; each cites the brief and specialist source. If any of these decisions turns out to be wrong during implementation, bounce it back — do not silently deviate.

1. **Template engine:** `strings.ReplaceAll` + `__BELUGA_<FIELD>__` sentinels + mandatory `go/format.Source` gate on every generated `.go` file. `text/template` deferred to S3+. (brief Decision #1; specialist-security-architect §Q3.)
2. **Scaffolder placement:** `framework/cmd/beluga/scaffold/` subpackage; `//go:embed templates` directive lives in `renderer.go` (not in any cobra command file); cobra command calls `scaffold.Scaffold(ctx, opts)` only. Layer 7, stdlib-only, zero new external deps. (brief Decision #2; specialist-systems-architect §Q1.)
3. **Named-template registry from day one:** `Registry{templates map[string]fs.FS}` with `Register` / `Get` / `Names`. `basic` is the sole populated entry in S2. Adding S3+ templates is a file-drop under `templates/<name>/` + one `Register` call; zero renderer or command changes. This `Registry` is a scaffold-internal type (package `scaffold`), **not** a framework-wide extension point — it does not participate in the Layer 3 registry invariant. (brief Decision #3; unanimous across all five specialists.)
4. **Default LLM provider:** OpenAI `gpt-4o-mini`. Generated `main.go` performs a startup check for `OPENAI_API_KEY == ""` and `log.Fatal`s with a human-readable message including `https://platform.openai.com/api-keys` and a swap-provider comment. (brief Decision #4; specialist-ai-ml-expert §Q3.)
5. **Default tool:** `echo` — `EchoInput{Message string}`; ~10 lines; demonstrates registration + dispatch + round-trip. Persona nudges the LLM to call it. (brief Decision #5; specialist-ai-ml-expert §Q2.)
6. **Default planner:** omit `WithPlannerName` entirely; framework default (`react`, set in `agent/option.go:42-48`) applies implicitly. Single in-line comment linking to DOC-06 with an example upgrade (`// agent.WithPlannerName("reflexion")` commented). (brief Decision #6; specialist-ai-ml-expert §Q1.)
7. **Project-name validation:** strict allowlist regex `^[a-z][a-z0-9-]{0,62}[a-z0-9]$|^[a-z][a-z0-9]$` enforced at `cobra.RunE` entry before any filesystem or template op. Reject on mismatch — do **not** sanitise. Pre-regex checks: empty string rejected first; length > 64 bytes rejected second (anti-ReDoS). Windows reserved-name blocklist (`CON`, `PRN`, `AUX`, `NUL`, `COM[1-9]`, `LPT[1-9]`) as an explicit secondary check by case-insensitive compare. (brief Decision #7; specialist-security-architect §Q1.)
8. **Target overwrite:** three-state policy. Non-existent → create + proceed. Empty → warn to stderr + proceed. Non-empty → exit non-zero unless `--force`. `--force` overwrites **individual files**, never calls `os.RemoveAll`. `--force` not implied by any other flag. (brief Decision #8; specialist-security-architect §Q2.)
9. **Module path:** `example.com/<project-name>` default; `--module` override. The `--module` value is **not** subject to the project-name regex — it has its own Go module-path grammar (dots, slashes, mixed case valid). Validate against `golang.org/x/mod/module.CheckPath` before write. If `golang.org/x/mod` is not already a framework dep, fall back to an inline regex matching Go's module grammar (see task T1 below). (brief Decision #9; specialist-systems-architect §Q2; specialist-security-architect §Risk #1.)
10. **`.beluga/project.yaml` schema:** exactly five fields — `schema-version: 1`, `name`, `template`, `beluga-version`, `scaffolded-at` (RFC 3339). No hooks, no plugin config, no provider overrides. (brief Decision #10; specialist-systems-architect §Q4.)
11. **`beluga new` project detection:** ancestor walk from CWD. At each level, require **both** `.beluga/project.yaml` exists **and** `go.mod` contains a line matching `require github.com/lookatitude/beluga-ai/v2`. Stop at filesystem root. If not found, exit non-zero: `not inside a Beluga project (no .beluga/ directory found — run beluga init first)`. (brief Decision #11; specialist-systems-architect §Q4.)
12. **Framework version pin:** use `version.Get()` from `framework/cmd/beluga/internal/version`. If `Get() == "(devel)"`, write a `replace github.com/lookatitude/beluga-ai/v2 => <workspace-root>` directive in the generated `go.mod` instead of a bogus `require v(devel)`. Workspace root resolved by ancestor-walking for `go.mod` containing `module github.com/lookatitude/beluga-ai/v2`. (brief Decision #12; specialist-devops-expert §Q3 + §Risk #1.)
13. **Dockerfile:** multi-stage, `golang:1.25-alpine` builder, `gcr.io/distroless/static-debian12` runtime, `CGO_ENABLED=0`, `-ldflags="-s -w"`, OCI `LABEL org.opencontainers.image.source` + `version` placeholders, **mandatory** comment: `# NOTE: this image requires CGO_ENABLED=0. If you add a package with CGo requirements, switch to gcr.io/distroless/base-debian12 or a glibc-based runtime image.` (brief Decision #13 + Risk #10; specialist-devops-expert §Q1.)
14. **Makefile + GHA workflow both.** `Makefile` has `build`, `test`, `lint`, `check` phony targets plus a **commented** `security:` target showing `gosec` + `govulncheck`. `.github/workflows/ci.yml` uses `actions/checkout@v4`, `actions/setup-go@v5` with `go-version-file: go.mod`, `golangci-lint-action@v6` (install-only), then calls `make check`. (brief Decision #14; specialist-devops-expert §Q2.)
15. **Zero OTel wiring in S2 templates.** No `BootstrapFromEnv` call, no `o11y` import (commented or otherwise), no `OTEL_*` or `BELUGA_OTEL_*` vars in `.env.example`. Miguel override of observability-expert §Q1 option (b) — a commented stub referencing a no-op skeleton creates worse DX than omission. Whole OTel scaffold lands in S3 when `BootstrapFromEnv` is real. (brief Decision #15 + Risk #8.)
16. **`.env.example`:** exactly one non-comment line — `OPENAI_API_KEY=YOUR_OPENAI_API_KEY_HERE`. `YOUR_*_HERE` sentinel form is the pattern gitleaks v8 and truffleHog v3 explicitly exclude from high-confidence detection. Header comment lines reference `https://platform.openai.com/api-keys` and "copy to `.env`". (brief Decision #16; specialist-security-architect §Q4.)
17. **`init.go` full REPLACE.** Delete `runInit` (lines 37-130 of current `init.go`) and the `--name`/`--dir` flag wiring in `newInitCmd`; rewrite `newInitCmd` for positional `<project-name>` argument + `--template` / `--module` / `--force` flags; `RunE` calls `scaffold.Scaffold(ctx, opts)`. No incremental migration. (brief Decision #17 + Risk #7; specialist-systems-architect §Risk #1.)
18. **`beluga new` stub error semantics.** Planner (and agent / tool) stub bodies use `core.Errorf(core.ErrNotFound, "<Type>.<Method> not implemented")` — **not** `panic`. Safer accidental-wiring behaviour; idiomatic Beluga. Tests use `t.Skip("remove when <Type>.<Method> is implemented")`. Commented `init()` registration block included only for the planner stub (matches specialist-ai-ml-expert §Q4). (brief Decision #18; specialist-ai-ml-expert §Q4 + §Risk #4.)
19. **Architecture doc update.** Append a subsection to `framework/docs/architecture/01-overview.md` Layer 7 section titled "Canonical consumer shape" — 2-3 paragraphs enumerating mandatory elements (`package main`, exactly one provider blank import, `llm.New(name, config.ProviderConfig{...})`, `agent.New(id, WithLLM, WithPersona)` + at least one of `Invoke` / `Stream`) + optional elements (tools, OTel bootstrap, `WithPlannerName`) + import discipline (blank import, never `&openai.Provider{}`). Link to `docs/guides/first-agent.md`; do **not** repeat code inline. (brief Decision #19; specialist-systems-architect §Q3.)

**Two execution conventions** (not in the brief, stated here so developer-go follows them uniformly):

A. **`context.Context` first.** Every exported function in `scaffold/` takes `ctx context.Context` as the first parameter — Invariant #9 + framework CLAUDE.md rule 9. `Scaffold(ctx, opts)`, not `Scaffold(opts)`. The context is plumbed so `go/format.Source` and file walks respect cancellation via `ctx.Err()` checks at each file boundary (cheap, no goroutines introduced).

B. **No channels, no `iter.Seq2`.** The scaffolder is synchronous by nature — one filesystem walk, one write per file. The framework's streaming-via-`iter.Seq2` invariant (Invariant #6, CLAUDE.md rule 1) does **not** apply: there is no streaming data. State this in the QA notes so reviewer-qa's invariant check confirms "N/A — scaffolder is pure stdlib synchronous" rather than flagging absence as a regression.

No ADR is written. The brief itself is the external design artefact. The two items above are execution conventions, not architectural commitments.

---

### Interface definitions

Compile-checked in my head against `framework/core/errors.go`, `framework/cmd/beluga/internal/version/version.go`, `framework/tool/functool.go`, and Go 1.25 stdlib. All types are ≤4 methods per Invariant #9.

#### 1. `cmd/beluga/scaffold/scaffold.go`

```go
// Package scaffold generates a Beluga project from a named template. It is
// Layer 7, stdlib-only, and exposes one entry point to the cobra init command.
package scaffold

import (
    "context"
    "io/fs"
    "time"
)

// Options controls a single Scaffold run. All fields are required unless noted.
// The caller (cobra RunE in cmd/beluga/init.go) is responsible for resolving
// defaults (TargetDir = filepath.Join(cwd, ProjectName), ModulePath =
// "example.com/"+ProjectName), capturing the framework version via
// version.Get(), and stamping ScaffoldedAt via time.Now() — these are
// explicit on Options so tests can drive them with fixed values.
type Options struct {
    ProjectName   string    // validated against the project-name allowlist before Scaffold is called
    Template      string    // name registered in the built-in Registry; defaults to "basic" at caller
    ModulePath    string    // validated against Go module-path grammar before Scaffold is called
    TargetDir     string    // absolute path; three-state overwrite policy applies
    Force         bool      // true → overwrite individual files in a non-empty target
    BelugaVersion string    // from version.Get(); scaffolder detects "(devel)" and emits replace directive
    ScaffoldedAt  time.Time // caller sets; tests fix this to a known value for golden-file stability
}

// Scaffold renders the template identified by opts.Template into opts.TargetDir
// using opts as substitution inputs. Returns core.Error with:
//   - core.ErrInvalidInput if the template name is not registered or any
//     substitution variable fails renderer-side sanity checks.
//   - core.ErrConflict if TargetDir is non-empty and Force is false.
//   - core.ErrInternal if go/format.Source fails on any generated .go file
//     (indicates a scaffolder bug, not user input error).
//
// Scaffold does not itself validate ProjectName or ModulePath — those are
// rejected at the cobra RunE entry point before Scaffold is called, per
// specialist-security-architect §Q1. Scaffold does validate the target
// directory state and the template registration.
func Scaffold(ctx context.Context, opts Options) error

// validateProjectName enforces the allowlist regex + Windows-reserved-name
// blocklist. Exported for cobra RunE to call before constructing Options.
//
// Returns core.ErrInvalidInput with a descriptive message naming the rule;
// never sanitises.
func ValidateProjectName(name string) error

// ValidateModulePath applies the Go module-path grammar to a --module override.
// Returns core.ErrInvalidInput with a descriptive message on rejection.
func ValidateModulePath(path string) error

// DefaultRegistry returns the process-wide registry populated at package init()
// by templates_builtin.go. Exported so the cobra command can use
// DefaultRegistry().Names() for --template help text.
func DefaultRegistry() *Registry
```

#### 2. `cmd/beluga/scaffold/template.go`

```go
package scaffold

import (
    "io/fs"
    "sync"
)

// ScaffoldVars is the fixed substitution set. Every __BELUGA_<FIELD>__ sentinel
// in a .tmpl file maps to exactly one field here. Substitution is deterministic:
// strings.ReplaceAll is called for each field in alphabetical order of the
// sentinel name (AgentName, BelugaVersion, ModelName, ModulePath, ProjectName,
// ProviderImport, ProviderName, ScaffoldedAt) to keep golden-file tests stable.
type ScaffoldVars struct {
    AgentName      string // derived from ProjectName (e.g., "myproject-agent")
    BelugaVersion  string // Options.BelugaVersion or "(devel)"
    ModelName      string // "gpt-4o-mini" for basic
    ModulePath     string // Options.ModulePath
    ProjectName    string // Options.ProjectName
    ProviderImport string // "github.com/lookatitude/beluga-ai/v2/llm/providers/openai" for basic
    ProviderName   string // "openai" for basic
    ScaffoldedAt   string // Options.ScaffoldedAt.UTC().Format(time.RFC3339)
}

// Registry is a scaffold-internal named-template registry. It is NOT a
// framework extensibility point (not exposed to consumers of beluga-ai/v2),
// so it does not participate in the Layer 3 registry invariant.
//
// Zero value is NOT usable — construct via NewRegistry.
type Registry struct {
    mu        sync.RWMutex
    templates map[string]fs.FS
}

func NewRegistry() *Registry

// Register associates a template name with a read-only embedded filesystem.
// Returns core.ErrInvalidInput if name is empty or already registered.
// Called from templates_builtin.go's init().
func (r *Registry) Register(name string, fsys fs.FS) error

// Get returns the filesystem for a template. ok==false if the name is not
// registered.
func (r *Registry) Get(name string) (fsys fs.FS, ok bool)

// Names returns the registered template names in sorted order. Used by the
// --template flag's help text at startup.
func (r *Registry) Names() []string
```

Four methods on `Registry` (including `NewRegistry`): `NewRegistry`, `Register`, `Get`, `Names`. Borderline on Invariant #9; `NewRegistry` is a constructor, not a method on the type, so the method count is 3 — within the limit.

#### 3. `cmd/beluga/scaffold/renderer.go`

```go
package scaffold

import (
    "context"
    "io/fs"
)

//go:embed templates
var builtinTemplatesFS embed.FS

// applyTemplate substitutes every __BELUGA_<FIELD>__ sentinel in src with the
// corresponding field of vars. Sentinels are replaced in a fixed order (see
// ScaffoldVars doc) so output is deterministic across runs. Unknown sentinels
// are left in place — applyTemplate does not error on them; the caller (tests)
// must assert the output contains no remaining __BELUGA_ substring.
func applyTemplate(src string, vars ScaffoldVars) string

// renderFS walks fsys and writes every file under it to targetDir, with per-file
// behaviour:
//   - Regular text replacement via applyTemplate on byte contents.
//   - Files whose rendered path ends in ".go" additionally pass through
//     go/format.Source. Format failure returns a core.Error wrapping the
//     parse error with the preamble: "beluga: generated source has a syntax
//     error — this is a bug in the scaffolder, please report it at
//     github.com/lookatitude/beluga-ai/issues (details: <parse error>)"
//     (brief Risk #9).
//   - Files with names ending in ".tmpl" have that suffix stripped on write
//     (main.go.tmpl → main.go, Dockerfile.tmpl → Dockerfile).
//   - Paths under ".beluga/" are created with 0750 perms; all other dirs 0755;
//     files 0644 (0600 for .env.example? — no: .env.example is a committable
//     sample, 0644 is correct; specialist-security-architect §Q4 keeps it
//     committable).
//
// When force is false and targetDir is non-empty, renderFS returns
// core.Errorf(core.ErrConflict, ...) without writing any files. When force is
// true, each file is opened with os.O_CREATE|os.O_TRUNC (O_TRUNC overwrites
// existing files; O_EXCL is NOT used with force, by design).
//
// ctx.Err() is checked between files so large templates remain cancellable.
func renderFS(ctx context.Context, fsys fs.FS, targetDir string, vars ScaffoldVars, force bool) error

// detectProjectRoot ancestor-walks from startDir upward looking for a directory
// containing BOTH .beluga/project.yaml AND go.mod with a
// "github.com/lookatitude/beluga-ai/v2" require line. Returns the root
// directory's absolute path.
//
// Returns core.Errorf(core.ErrNotFound, "not inside a Beluga project...") if
// no ancestor qualifies. Used by beluga new agent/tool/planner.
func detectProjectRoot(startDir string) (projectRoot string, err error)

// detectWorkspaceRoot ancestor-walks from startDir upward looking for a go.mod
// whose "module" directive equals "github.com/lookatitude/beluga-ai/v2". Used
// only when version.Get() == "(devel)" to emit a replace directive in the
// generated go.mod.
//
// Returns core.ErrNotFound if no such ancestor exists; the caller falls back
// to writing "v0.0.0-unknown" and printing a stderr warning.
func detectWorkspaceRoot(startDir string) (workspaceRoot string, err error)
```

Error-code choices verified against `framework/core/errors.go`:

- `core.ErrInvalidInput` (defined line 23) — used for project-name, module-path, and template-name validation failures.
- `core.ErrNotFound` (defined line 38) — used for "project root not found" and for `beluga new` stub bodies.
- **`core.ErrConflict` does not exist** in the current `core/errors.go` error-code set (only the 9 codes at lines 14-38 plus `ErrToolFailed` etc. — no conflict code). **Developer-go decision:** use `core.ErrInvalidInput` with a message that names the non-empty target directory + `--force` flag, OR add `ErrConflict` as a new code in `core/errors.go`. The minimal-risk path is using `ErrInvalidInput` with a clear message; adding a new error code touches Layer 1 and expands blast radius. **Default: `core.ErrInvalidInput` with message `"target directory %q is not empty; use --force to overwrite"`**. Flag as an open question (§7) if developer-go disagrees.
- **`core.ErrInternal` does not exist either.** For `go/format.Source` failures (which indicate a scaffolder bug, not user input), use `core.Errorf(core.ErrInvalidInput, "beluga: generated source has a syntax error — this is a bug...")` with the understanding that this is a scaffolder-side "invalid input" from the template's perspective. Alternatively treat as a plain `fmt.Errorf` (matching the S1 architect-plan's execution-convention #2: `fmt.Errorf` for CLI-local errors). **Default: `fmt.Errorf`** because the error never crosses the capability-layer boundary and the wrap already includes the parse error via `%w`. This is consistent with S1's pattern; flag if developer-go prefers `core.Error`.

#### 4. `cmd/beluga/new.go` and `new_agent.go` / `new_tool.go` / `new_planner.go` — stub shapes

```go
// cmd/beluga/new.go
func newNewCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:           "new",
        Short:         "Scaffold a new component (agent, tool, planner) inside a Beluga project",
        SilenceUsage:  true,
        SilenceErrors: true,
    }
    cmd.AddCommand(newNewAgentCmd(), newNewToolCmd(), newNewPlannerCmd())
    return cmd
}
```

Each `new_<kind>_cmd.go` takes a single positional argument `<Name>` (PascalCase; validated against `^[A-Z][A-Za-z0-9]*$`), calls `scaffold.detectProjectRoot(cwd)`, then writes exactly two files:

- `<snake_name>.go` — one exported type (e.g., `MyAgent`, `MyTool`, `MyPlanner`) with a `var _ <interface> = (*T)(nil)` compile-time check and method bodies returning `core.Errorf(core.ErrNotFound, "<Type>.<Method> not implemented")`. (The `panic`-based approach from specialist-ai-ml-expert §Q4 is overridden by Decision #18 / specialist-ai-ml-expert §Risk #4.) Tool stub uses `tool.NewFuncTool` per `framework/tool/functool.go:39-41`. Planner stub additionally includes a commented `init()` block showing `agent.RegisterPlanner` as specified in specialist-ai-ml-expert §Q4.
- `<snake_name>_test.go` — `package <projectpkg>_test` + `t.Skip("remove when <Type> is implemented")` guard + one table-driven test case. Ensures `go test ./...` is green out of the box.

Generated agent stub (shape only):

```go
// myagent.go — generated by `beluga new agent MyAgent`.
package <projectpkg>

import (
    "context"

    "github.com/lookatitude/beluga-ai/v2/agent"
    "github.com/lookatitude/beluga-ai/v2/core"
)

// MyAgent is a user-defined agent.
// Replace the bodies below with your implementation.
var _ agent.Agent = (*MyAgent)(nil)

type MyAgent struct{ /* TODO: fields */ }

func NewMyAgent() *MyAgent { return &MyAgent{} }

// Invoke runs the agent for a single request. TODO: implement.
func (a *MyAgent) Invoke(ctx context.Context, input string) (string, error) {
    return "", core.Errorf(core.ErrNotFound, "MyAgent.Invoke not implemented")
}
```

Generated tool stub:

```go
// mytool.go — generated by `beluga new tool MyTool`.
package <projectpkg>

import (
    "context"

    "github.com/lookatitude/beluga-ai/v2/core"
    "github.com/lookatitude/beluga-ai/v2/tool"
)

// MyToolInput is the typed input for MyTool. Tags drive the JSON schema.
type MyToolInput struct {
    // TODO: add fields with `description:` and `required:"true"` tags.
}

func NewMyTool() tool.Tool {
    return tool.NewFuncTool(
        "mytool",
        "TODO: one-sentence description of MyTool.",
        func(ctx context.Context, in MyToolInput) (*tool.Result, error) {
            return nil, core.Errorf(core.ErrNotFound, "MyTool not implemented")
        },
    )
}
```

Generated planner stub mirrors specialist-ai-ml-expert §Q4 with the `panic` lines replaced by `return nil, core.Errorf(core.ErrNotFound, ...)`. The commented `init()` block is included verbatim from that specialist input (lines 240-247 of specialist-ai-ml-expert.md).

**Detection-failure contract (for all three `new_<kind>` commands):** when `detectProjectRoot` fails, print the returned `core.Error.Message` to stderr and return exit code 1. Do not create any files.

**Version-skew warning contract:** after `detectProjectRoot` succeeds, read the version pin from `<projectRoot>/go.mod` (simple regex on the `require github.com/lookatitude/beluga-ai/v2` line — do not import `golang.org/x/mod` just for this). If pin differs from `version.Get()`, print the warning specified in brief §Architecture ("beluga new version-skew warning"). Does not block execution.

---

### Implementation plan

Twelve tasks, dependency-ordered. Red/Green TDD: every task with production code lists its tests first. Tasks 1-6 build the scaffolder bottom-up; task 7 is the load-bearing `init.go` replacement; tasks 8-12 wire the CLI surface and ship the non-code files.

#### T1: Scaffold package — Options, validators, Scaffold dispatch

- **Description:** Create `cmd/beluga/scaffold/scaffold.go` with the public surface per interface definition #1. Implement `ValidateProjectName` (regex + length + Windows-reserved-name blocklist; empty-check and >64-byte check BEFORE regex) and `ValidateModulePath` (inline regex matching Go module-path grammar — avoid `golang.org/x/mod` dep; confirm Go module proxy rules in code comments). `Scaffold` body delegates to `renderFS` (task T3). Write tests first: `scaffold_validate_test.go` table-driven: empty string, `"My Project"` (spaces), `"../evil"` (path traversal), `"-badstart"` (leading hyphen), `"con"` (Windows reserved), 64-char all-lowercase (accepted), 65-char (rejected), `"ab"` (accepted, minimum), `"a"` (rejected, too short). Separate table for `ValidateModulePath`: `"example.com/foo"` (accepted), `"github.com/org/repo"` (accepted), `"UPPER/foo"` (accepted — module paths allow mixed case), `"foo bar"` (rejected — space), `"foo;rm -rf /"` (rejected — metachar), `""` (rejected).
- **Files (create):** `framework/cmd/beluga/scaffold/scaffold.go`, `framework/cmd/beluga/scaffold/scaffold_validate_test.go`.
- **Dependencies:** none.
- **Cites:** brief Decision #7, #9; specialist-security-architect §Q1, §Risk #1.
- **Acceptance criteria:**
  - `go build ./cmd/beluga/scaffold/...` compiles.
  - `go test -race ./cmd/beluga/scaffold/...` green; table-driven tests cover at least the 12 cases above.
  - `ValidateProjectName("../evil")` returns `*core.Error` with `Code == core.ErrInvalidInput` and message containing the phrase `"allowed pattern"` or similar naming the rule.
  - `ValidateProjectName("CON")` rejected (Windows reserved), message mentions `"reserved"`.
  - `ValidateModulePath("github.com/myorg/myproject")` returns nil.
  - `Scaffold` with an unregistered template name returns `core.ErrInvalidInput` naming the unknown template and listing `DefaultRegistry().Names()`.

#### T2: Scaffold package — Registry + ScaffoldVars

- **Description:** Create `cmd/beluga/scaffold/template.go` per interface definition #2. Tests first: `template_test.go` covers `NewRegistry` (empty), `Register` returns error on duplicate and on empty name, `Get` ok/not-ok, `Names` returns sorted slice. Mutex concurrency test: spawn 8 goroutines all calling `Register`/`Get` — race detector must be clean.
- **Files (create):** `framework/cmd/beluga/scaffold/template.go`, `framework/cmd/beluga/scaffold/template_test.go`.
- **Dependencies:** T1 (for `core.Error` import + consistent error taxonomy).
- **Cites:** brief Decision #3; specialist-systems-architect §Q1 + "Template scope (Risk #5)".
- **Acceptance criteria:**
  - `go test -race ./cmd/beluga/scaffold/...` green including concurrency test.
  - `NewRegistry().Names()` returns `[]string{}` (nil-safe, not nil).
  - `Register("basic", testFS); Register("basic", testFS)` second call returns `core.ErrInvalidInput`.
  - `Register("", testFS)` returns `core.ErrInvalidInput`.
  - Compile-time assertion: no exported field on `Registry` (all internal).

#### T3: Scaffold package — renderer + go/format.Source gate + detection helpers

- **Description:** Create `cmd/beluga/scaffold/renderer.go` per interface definition #3. Tests first: `renderer_test.go`:
  - `applyTemplate` — known input with every sentinel → expected output; unknown `__BELUGA_MISSING__` left in place.
  - `renderFS` happy path with a small in-memory `fstest.MapFS`: 3 files (main.go.tmpl, go.mod.tmpl, .env.example.tmpl), check (a) written paths lose `.tmpl` suffix, (b) `main.go` content passes `go/format.Source` round-trip identity.
  - `renderFS` rejects non-empty target without `--force` with `core.ErrInvalidInput` (see interface-def §3 rationale for code choice).
  - `renderFS` overwrites existing files when `--force=true`.
  - `renderFS` returns a wrapped parse error with the "this is a bug" preamble when a template deliberately produces malformed Go (e.g., `func main() { syntax error }`).
  - `renderFS` respects `ctx.Err()` — cancelled context before first write returns `ctx.Err()`.
  - `detectProjectRoot` — construct tempdir with `.beluga/project.yaml` + `go.mod` containing the require line; assert walk finds it from a subdirectory.
  - `detectProjectRoot` — tempdir with only one of the two signals; walk fails with `core.ErrNotFound`.
  - `detectWorkspaceRoot` — similar tests using `module github.com/lookatitude/beluga-ai/v2` line.
- **Files (create):** `framework/cmd/beluga/scaffold/renderer.go`, `framework/cmd/beluga/scaffold/renderer_test.go`.
- **Dependencies:** T1, T2.
- **Cites:** brief Decision #1, #11, #12 + Risk #9, #12; specialist-security-architect §Q3 "one firm rule"; specialist-systems-architect §Q4.
- **Acceptance criteria:**
  - `go test -race ./cmd/beluga/scaffold/...` green.
  - `applyTemplate` output is deterministic across runs (same input → same bytes).
  - `renderFS` never writes a `.go` file that fails `go/format.Source` — test asserts error is returned and no `.go` file is on disk.
  - `renderFS` with a cancelled context returns `ctx.Err()` and writes zero files.
  - `detectProjectRoot` ancestor-walks at least 3 levels deep without allocation surprises.

#### T4: Built-in templates — `templates/basic/*.tmpl` (9 files)

- **Description:** Create the embed subtree under `framework/cmd/beluga/scaffold/templates/basic/` matching the brief's File-level plan. Every file uses `__BELUGA_<FIELD>__` sentinels only from the `ScaffoldVars` set (no other names). The 9 files:
  1. `main.go.tmpl` — structurally isomorphic to `examples/first-agent/main.go`: imports `context fmt log os time` + `agent config core llm tool` + blank `_ "__BELUGA_PROVIDER_IMPORT__"`. Startup check: `if apiKey == "" { log.Fatal("OPENAI_API_KEY is not set...\n  Get a key at https://platform.openai.com/api-keys\n  Or swap providers: see comments at the top of this file.") }`. Calls `llm.New("__BELUGA_PROVIDER_NAME__", config.ProviderConfig{Provider: "__BELUGA_PROVIDER_NAME__", APIKey: apiKey, Model: "__BELUGA_MODEL_NAME__"})`. Builds `agent.New("__BELUGA_AGENT_NAME__", WithLLM, WithPersona{Role: "helpful assistant", Goal: "answer questions accurately", Backstory: "You are concise and direct. When asked to echo something, always use the echo tool.", Traits: []string{"concise","accurate"}}, WithTools([]tool.Tool{newEchoTool()}))`. Single commented line: `// agent.WithPlannerName("reflexion"), // see: docs/architecture/06-reasoning-strategies.md`. Calls `a.Invoke(ctx, "Please echo: hello world")`. `newEchoTool()` defined per specialist-ai-ml-expert §Q2 code sketch. **Zero OTel wiring.**
  2. `go.mod.tmpl` — `module __BELUGA_MODULE_PATH__` + `go 1.25` + `require github.com/lookatitude/beluga-ai/v2 __BELUGA_VERSION__`. The `(devel)` handling is done by the renderer post-processing: when `vars.BelugaVersion == "(devel)"`, strip the `require` line's version + append a `replace github.com/lookatitude/beluga-ai/v2 => <workspaceRoot>` line. Implement this branch in `renderer.go` (not in the template) — template stays simple.
  3. `.env.example.tmpl` — 4 lines only: comment header (2 lines: copy-to-`.env` + do-not-commit) + blank + `OPENAI_API_KEY=YOUR_OPENAI_API_KEY_HERE` + optional `# BELUGA_MODEL=gpt-4o-mini` commented. **No OTEL_* vars.**
  4. `.gitignore.tmpl` — `.env`, `.env.local`, `*.env`, binary name `__BELUGA_PROJECT_NAME__`, `/tmp/`. Explicitly does **not** exclude `.env.example`.
  5. `.beluga/project.yaml.tmpl` — exactly 5 fields per Decision #10. `schema-version: 1`, `name: __BELUGA_PROJECT_NAME__`, `template: basic`, `beluga-version: __BELUGA_VERSION__`, `scaffolded-at: __BELUGA_SCAFFOLDED_AT__`.
  6. `Dockerfile.tmpl` — multi-stage per Decision #13. Mandatory CGo-incompat comment block at the top of the runtime stage.
  7. `Makefile.tmpl` — 4 phony targets (`build test lint check`) + commented `security:` target. Comment referencing goreleaser as "copy from framework's own .goreleaser.yml when you need releases".
  8. `.github/workflows/ci.yml.tmpl` — single job `check` per specialist-devops-expert §Q2 sketch. `golangci-lint-action@v6` in install-only mode + `make check`.
  9. **(no 9th file)** — the File-level-plan table also lists `go.sum` placeholder, but an empty `go.sum.tmpl` is neither useful nor required: `go mod tidy` on first run writes it. **Developer-go decision:** omit `go.sum.tmpl`; document this divergence from the brief's table (which says "Create" for go.sum placeholder) as a minor simplification — an empty go.sum file may cause checksum-mismatch anxiety and adds zero value. Flag in §7 if Miguel wants it retained.
- **Files (create):** the 8 `.tmpl` files above plus their parent directories under `framework/cmd/beluga/scaffold/templates/basic/`.
- **Dependencies:** T3 (to validate they render cleanly via `applyTemplate` + `go/format.Source`).
- **Cites:** brief File-level plan lines "basic/main.go.tmpl" through "basic/.github/workflows/ci.yml.tmpl"; specialist-ai-ml-expert §Q1/Q2/Q3; specialist-security-architect §Q4; specialist-devops-expert §Q1/Q2; brief Decision #10/#13/#14/#15/#16.
- **Acceptance criteria:**
  - Every `.tmpl` file uses only sentinels from the `ScaffoldVars` set — verify with `grep -r __BELUGA_ framework/cmd/beluga/scaffold/templates/basic/ | grep -v -E '(AGENT_NAME|BELUGA_VERSION|MODEL_NAME|MODULE_PATH|PROJECT_NAME|PROVIDER_IMPORT|PROVIDER_NAME|SCAFFOLDED_AT)'` returns empty.
  - `main.go.tmpl` after substitution with `{ProjectName: "test", ModulePath: "example.com/test", ProviderName: "openai", ProviderImport: "github.com/lookatitude/beluga-ai/v2/llm/providers/openai", ModelName: "gpt-4o-mini", AgentName: "test-agent", BelugaVersion: "v2.10.1", ScaffoldedAt: "2026-04-20T00:00:00Z"}` passes `go/format.Source` with byte-identity round trip.
  - `main.go.tmpl` contains **none** of: `o11y`, `BootstrapFromEnv`, `OTEL_`, `BELUGA_OTEL`.
  - `.env.example.tmpl` contains exactly one `OPENAI_API_KEY=...HERE` line matching the gitleaks-safe pattern.
  - `Dockerfile.tmpl` contains the `CGO_ENABLED=0` line AND the CGo-incompatibility NOTE comment.
  - `Makefile.tmpl` contains a commented `security:` target referencing both `gosec` and `govulncheck`.

#### T5: Template registration

- **Description:** Create `cmd/beluga/scaffold/templates_builtin.go` with `func init()` that calls `DefaultRegistry().Register("basic", fsSubtree)` where `fsSubtree` is obtained via `fs.Sub(builtinTemplatesFS, "templates/basic")`. Panic if registration fails (init-time failure is a compile-equivalent error — same pattern as `llm/providers/*/init()`). Add a compile-time assertion and a test that `DefaultRegistry().Get("basic")` returns ok=true at init time.
- **Files (create):** `framework/cmd/beluga/scaffold/templates_builtin.go`, `framework/cmd/beluga/scaffold/templates_builtin_test.go`.
- **Dependencies:** T2, T4.
- **Cites:** brief File-level plan "templates_builtin.go"; specialist-systems-architect "Template scope (Risk #5)".
- **Acceptance criteria:**
  - `DefaultRegistry().Names()` returns `[]string{"basic"}` after package init.
  - `DefaultRegistry().Get("basic")` returns non-nil `fs.FS` with at least 8 files under it (verify by counting `fs.WalkDir`).
  - `go vet ./cmd/beluga/scaffold/...` clean.

#### T6: Scaffold golden tests

- **Description:** Create `cmd/beluga/scaffold/scaffold_test.go` and commit golden output under `cmd/beluga/scaffold/testdata/golden/basic/`. Test fixes `ScaffoldVars` to the stable values from T4's AC, runs `Scaffold` into a `t.TempDir()`, then diffs every produced file against its golden counterpart. On mismatch, emit a unified diff and fail. Provide a `-update` flag convention (`go test -run TestScaffoldBasic_Golden -update`) to regenerate goldens during template edits.
- **Files (create):** `framework/cmd/beluga/scaffold/scaffold_test.go`, `framework/cmd/beluga/scaffold/testdata/golden/basic/` (tree of committed files).
- **Dependencies:** T3, T4, T5.
- **Cites:** brief File-level plan "scaffold_test.go" + "testdata/golden/basic/"; specialist-devops-expert §Q4 "Golden file tests".
- **Acceptance criteria:**
  - `go test ./cmd/beluga/scaffold/...` green.
  - `testdata/golden/basic/` contains exactly the 8 files listed in T4 (no `go.sum`).
  - Intentional template mutation → test fails with a readable diff (verify by flipping a byte and running; revert before commit).
  - Golden `main.go` passes `gofmt -l` with empty output (already go-formatted by the scaffolder).

#### T7: REPLACE `cmd/beluga/init.go` — delete 37-130, rewrite newInitCmd

- **Description:** **This is the highest-risk task of LOO-149.** Delete the entire `runInit` function (lines 37-130 of current `init.go`) and rewrite `newInitCmd` as follows:
  - Positional arg: `<project-name>` required (cobra `Args: cobra.ExactArgs(1)`).
  - Flags: `--template string` (default `"basic"`, help text built from `scaffold.DefaultRegistry().Names()`), `--module string` (default `""`, empty means use `example.com/<project-name>`), `--force bool` (default `false`).
  - `RunE`: (1) `scaffold.ValidateProjectName(args[0])`; (2) if `--module != ""`, `scaffold.ValidateModulePath(--module)`; (3) resolve `cwd`, build `Options{ProjectName, Template, ModulePath, TargetDir: filepath.Join(cwd, ProjectName), Force, BelugaVersion: version.Get(), ScaffoldedAt: time.Now().UTC()}`; (4) call `scaffold.Scaffold(cmd.Context(), opts)`; (5) on success print a 2-line summary to `cmd.OutOrStdout()`: `Initialized Beluga AI project %q in %s` + `Next: cd %s && export OPENAI_API_KEY=... && go run .`.
  - No other logic. No path-traversal code (the allowlist regex eliminates traversal by excluding `/`, `\`, `.`, `..`).
  - `SilenceUsage: true`, `SilenceErrors: true` preserved from existing `newInitCmd`.
- **Files (modify):** `framework/cmd/beluga/init.go` (full rewrite of the two functions; the existing 131-line file becomes ~40 lines).
- **Dependencies:** T1, T3, T5 (all scaffold surface must be ready).
- **Cites:** brief Decision #17 + Risk #7; specialist-systems-architect §Risk #1.
- **Acceptance criteria:**
  - `git diff framework/cmd/beluga/init.go` shows a near-complete file rewrite — old `runInit` gone, new `newInitCmd` uses positional arg and dispatches to `scaffold.Scaffold`.
  - `grep -c "runInit" framework/cmd/beluga/init.go` returns 0.
  - `grep -c "scaffold.Scaffold" framework/cmd/beluga/init.go` returns 1.
  - Manual smoke test (performed by developer, not automated): from a clean `t.TempDir()`, run `beluga init smoke-test` → exits 0 → `smoke-test/go.mod` exists → `smoke-test/main.go` contains `llm.New("openai"` AND blank-imports `/llm/providers/openai`.

#### T8: Rewrite `cmd/beluga/main_test.go` init tests against new dispatch

- **Description:** The four existing `TestCmdInit*` tests (lines 47-128 of current `main_test.go`) exercise `--name`/`--dir` flags and a `main.go`+`config/agent.json` directory shape that no longer exists post-T7. Rewrite them:
  - `TestCmdInit_PositionalName` — `executeSubcommand(newInitCmd(), []string{"my-project"})` from a `t.TempDir()` chdir; assert `my-project/go.mod`, `my-project/main.go`, `my-project/.beluga/project.yaml` exist; assert `main.go` contains `llm.New` and the openai blank import.
  - `TestCmdInit_RejectBadName` — `"../evil"`, `"-badstart"`, `"My Project"`, `"con"`: each must return `error != nil` with message containing the validation-rule phrase from `ValidateProjectName`.
  - `TestCmdInit_ForceOverwrite` — create a non-empty target; assert without `--force` returns error (message contains `"--force"`); with `--force` succeeds.
  - `TestCmdInit_DevelVersion` — run with `version.Version == ""` (default); assert generated `go.mod` contains a `replace` directive (since CLI version is `(devel)` in test context and the framework workspace root is detectable from the test's CWD inside `framework/cmd/beluga`).
  - `TestCmdInit_PathTraversal` + `TestCmdInit_RelativeTraversal`: remove — the `--dir` flag no longer exists, and the allowlist regex eliminates the traversal class by character-class exclusion. QA will verify the regex tests in T1 cover this.
  - `TestCmdInit_DefaultName`: remove — the positional arg is now required.
  - `TestRoot_Init` at line 355 must be updated to pass the positional name: `executeArgs([]string{"init", "runtest"})`.
- **Files (modify):** `framework/cmd/beluga/main_test.go`.
- **Dependencies:** T7.
- **Cites:** brief Success-criterion 7 ("non-empty target exits non-zero"), 8 (name rejection), 9 (project.yaml 5 fields).
- **Acceptance criteria:**
  - `go test -race ./cmd/beluga/...` green, including the 4 new init tests.
  - No test in `main_test.go` references `--name` or `--dir` flags.
  - `TestCmdInit_ForceOverwrite` asserts the error message contains the literal `"--force"` substring (Success criterion 7).
  - `TestCmdInit_RejectBadName` asserts at least one error message contains `"allowed pattern"` or similar (Success criterion 8).

#### T9: `beluga new agent/tool/planner` cobra commands + tests

- **Description:** Create `cmd/beluga/new.go` (parent) + `new_agent.go`, `new_tool.go`, `new_planner.go` (each implements its positional-arg subcommand per interface def #4) + companion `_test.go` files.
  - `new_agent.go` / `new_tool.go` / `new_planner.go`: each generates exactly 2 files (`<snake_name>.go` + `<snake_name>_test.go`) into the detected project root. Stub bodies return `core.Errorf(core.ErrNotFound, ...)` per Decision #18.
  - Name validation: `<Name>` must match `^[A-Z][A-Za-z0-9]*$` (PascalCase, no hyphens). Reject otherwise with `core.ErrInvalidInput`.
  - Project detection: call `scaffold.detectProjectRoot(cwd)`; on failure, print returned error message and exit 1.
  - Version-skew warning: after detection, read `go.mod` pin; compare to `version.Get()`; print stderr warning if they differ.
  - Tests: one per kind, table-driven, using `executeSubcommand`. Each test sets up a scaffolded project via a small in-test helper that writes `.beluga/project.yaml` + a stub `go.mod`, then runs the new command, then asserts the two expected files exist and compile (`go/parser.ParseFile` check; full `go build` happens in T11).
- **Files (create):** `framework/cmd/beluga/new.go`, `framework/cmd/beluga/new_agent.go`, `framework/cmd/beluga/new_tool.go`, `framework/cmd/beluga/new_planner.go`, and their `*_test.go` companions.
- **Dependencies:** T3 (for `detectProjectRoot`), T7 (for `init.go` to create a project to write into during tests — tests can also construct the minimum project shape by hand).
- **Cites:** brief §Scope bullet 2; specialist-ai-ml-expert §Q4 + §Risk #4; brief Decision #11, #18.
- **Acceptance criteria:**
  - `go test -race ./cmd/beluga/...` green.
  - `beluga new agent MyAgent` from a scaffolded project writes `my_agent.go` + `my_agent_test.go` (snake-case filenames derived from PascalCase argument).
  - Generated `my_agent.go` uses `core.Errorf(core.ErrNotFound, ...)` — not `panic`.
  - Generated `my_agent_test.go` contains exactly one `t.Skip("remove when MyAgent is implemented")` call.
  - Running `beluga new agent MyAgent` outside a Beluga project exits non-zero with a stderr message containing `"not inside a Beluga project"`.
  - Generated planner file contains a commented `init() { agent.RegisterPlanner(...) }` block (specialist-ai-ml-expert §Q4 lines 240-247).

#### T10: Wire new commands into `root.go`

- **Description:** Add `newNewCmd()` to the `root.AddCommand` list in `framework/cmd/beluga/root.go` alongside the existing six. No other changes. Single-line edit per line of code.
- **Files (modify):** `framework/cmd/beluga/root.go`.
- **Dependencies:** T9.
- **Cites:** brief §Scope bullet 2.
- **Acceptance criteria:**
  - `beluga --help` lists `new` as a subcommand alongside `version`, `providers`, `init`, `dev`, `test`, `deploy`.
  - `beluga new --help` lists `agent`, `tool`, `planner` as nested subcommands.
  - A new test `TestRoot_New_Help` asserts `executeArgs([]string{"new", "--help"})` exits 0 and output contains all three kind names.

#### T11: CI — scaffold-integration workflow

- **Description:** Create `framework/.github/workflows/scaffold-integration.yml` per specialist-devops-expert §Q4 sketch. Parallel to existing `_ci-checks.yml`. Steps: checkout → setup-go (pin from framework's own `go.mod`) → `actions/cache` for `~/.cache/go/pkg` → `go build -o /tmp/beluga ./cmd/beluga` → `mkdir /tmp/scaffold-test && cd /tmp/scaffold-test && /tmp/beluga init testproject` → `cd testproject && go mod edit -replace github.com/lookatitude/beluga-ai/v2=<GITHUB_WORKSPACE>` (only needed if `version.Get()` didn't already emit the `replace` directive — the scaffolder's `(devel)` handling covers this; the redundant edit is a safety net for tagged-release CI runs) → `go mod tidy` → `go build ./...` → `go vet ./...`. Matrix: `[basic]` to keep future template additions easy.
- **Files (create):** `framework/.github/workflows/scaffold-integration.yml`.
- **Dependencies:** T7 (working `beluga init`).
- **Cites:** brief §Scope bullet 6 + Success criterion "scaffold-integration CI job completes in under 90 seconds on cold cache"; specialist-devops-expert §Q4, §Risk #1.
- **Acceptance criteria:**
  - YAML lints clean (GitHub Actions workflow validator or `yamllint` if available).
  - On a push to the feature branch, the job runs and exits 0 within 90 seconds (cold cache) / 30 seconds (warm cache).
  - The job runs `gitleaks detect --source /tmp/scaffold-test/testproject` as a final step and reports zero high-confidence findings (Success criterion 10).

#### T12: Architecture doc update — DOC-01 Layer 7 canonical consumer shape

- **Description:** Append a 2-3 paragraph subsection to `framework/docs/architecture/01-overview.md` at the end of the Layer 7 section (the existing Layer 7 paragraph at DOC-01 already mentions the `beluga` CLI — the new subsection follows). Content per Decision #19 and specialist-systems-architect §Q3 content outline lines 77-82:
  1. **Mandatory elements** paragraph — `package main`; exactly one LLM provider blank import; `llm.New(name, config.ProviderConfig{Provider, APIKey, Model})`; `agent.New(id, agent.WithLLM(model), agent.WithPersona(agent.Persona{...}))`; `a.Invoke(ctx, input)` or `a.Stream(ctx, input)`.
  2. **Optional elements** paragraph — `agent.WithTools([]tool.Tool{...})`; `o11y.BootstrapFromEnv` (deferred to S3); `agent.WithPlannerName(...)` to override the default `react` planner; `agent.WithTracing()` to emit OTel spans at the agent boundary.
  3. **Import discipline** paragraph — providers registered via blank `_ "..."` imports; never construct `&openai.Provider{}` directly; `llm.New(name, cfg)` is the only correct constructor because middleware and hooks attach there. Link `[First Agent guide](../guides/first-agent.md)` as the canonical worked example — do **not** repeat code inline.
- **Files (modify):** `framework/docs/architecture/01-overview.md` (append only; do not touch existing paragraphs).
- **Dependencies:** none (can land in parallel with code tasks; no code dependency).
- **Cites:** brief Decision #19; specialist-systems-architect §Q3.
- **Acceptance criteria:**
  - New subsection titled `### Canonical consumer shape` (or equivalent heading) appended inside Layer 7 section.
  - Subsection contains exactly the three paragraphs above.
  - Subsection does **not** repeat code from `first-agent/main.go`; it references the guide by link.
  - `framework/docs/architecture/01-overview.md` still renders valid Markdown (no table/list breakage).

---

### Acceptance criteria — LOO-149 rollup

The 14 Success criteria from the brief, verbatim, mapped to task(s):

| # | Success criterion | Task(s) |
|---|---|---|
| 1 | `beluga init test-project && cd test-project && go build ./...` exits 0 on a clean `ubuntu-latest` machine with no prior framework dependencies (module proxy must resolve) | T7 + T11 |
| 2 | `go vet ./...` passes on the generated `test-project` without any reported issues | T11 |
| 3 | `golangci-lint run ./...` passes on the generated `test-project` (if installed in drift-check env) | T11 (advisory — job installs golangci-lint; optional in drift-check) |
| 4 | `beluga new agent MyAgent` creates `myagent.go` and `myagent_test.go` inside a scaffolded project; `go build ./...` on the project succeeds after the addition | T9 (file creation) + T11 (build verification extendable) |
| 5 | `beluga new tool MyTool` creates the analogous files; `go build ./...` succeeds | T9 + T11 |
| 6 | `beluga new planner MyPlanner` creates the analogous files; `go build ./...` and `go test ./...` (with `t.Skip` guard firing) both succeed | T9 (`t.Skip` guard) + T11 |
| 7 | `beluga init` with an existing non-empty target directory exits non-zero with a message containing `"--force"` before writing any files | T3 (renderFS conflict branch) + T8 (`TestCmdInit_ForceOverwrite`) |
| 8 | `beluga init` rejects project names not matching the allowlist with an error that names the validation rule; exits non-zero | T1 (`ValidateProjectName`) + T8 (`TestCmdInit_RejectBadName`) |
| 9 | Generated `.beluga/project.yaml` contains exactly the five fields: `schema-version`, `name`, `template`, `beluga-version`, `scaffolded-at` | T4 (template content) + T6 (golden) |
| 10 | `gitleaks detect --source ./test-project` passes with zero high-confidence findings | T4 (`.env.example` `YOUR_*_HERE` pattern) + T11 (gitleaks step in workflow) |
| 11 | `docker build -t scaffold-smoke .` inside the generated project exits 0 (optional; may be skipped in cost-constrained drift-check) | T4 (Dockerfile content) — verified outside CI unless Miguel opts in |
| 12 | `scaffold-integration` CI job completes in under 90 seconds on cold cache | T11 |
| 13 | Framework pre-commit gate passes on the feature branch: `go build ./...`, `go vet ./...`, `go test -race ./...`, `go mod tidy`, `gofmt`, `golangci-lint run ./...`, `gosec -quiet ./...`, `govulncheck ./...` | All tasks — enforced before each commit per `framework/.claude/rules/go-packages.md` |
| 14 | `framework/docs/architecture/01-overview.md` Layer 7 section contains a subsection naming mandatory elements of a canonical consumer application (blank provider import, `llm.New`, `agent.New` with `WithLLM` + `WithPersona`, `Invoke`) | T12 |

---

### Risks surfaced for developer-go attention

1. **Brief Risk #7 — `init.go` full REPLACE (highest risk of LOO-149).** Delete lines 37-130 of `framework/cmd/beluga/init.go` (the `runInit` function) and rewrite lines 14-32 (the `newInitCmd` function) for the positional-arg surface. No incremental migration. Landing place: T7. The existing path-traversal checks are replaced by the project-name allowlist regex in `scaffold.ValidateProjectName` (T1) — traversal attempts like `"../evil"` never reach any filesystem call.

2. **Brief Risk #9 — `go/format.Source` UX wrap.** When substitution produces malformed Go (indicative of a scaffolder bug, not user input), the error emitted to the user must be wrapped with the preamble `"beluga: generated source has a syntax error — this is a bug in the scaffolder, please report it at github.com/lookatitude/beluga-ai/issues (details: <parse error>)"`. Never surface raw `go/format` compiler output. Landing place: T3 `renderFS` error branch.

3. **Brief Risk #10 — Dockerfile CGo-incompat comment is mandatory.** `gcr.io/distroless/static-debian12` silently runtime-fails on any binary built with `CGO_ENABLED=1`. The generated Dockerfile must include the comment block at the top of the runtime stage: `# NOTE: this image requires CGO_ENABLED=0. If you add a package with CGo requirements, switch to gcr.io/distroless/base-debian12 or a glibc-based runtime image.` Landing place: T4 (`Dockerfile.tmpl`).

4. **Brief Risk #12 — `(devel)` version requires `replace` directive.** When `version.Get() == "(devel)"`, writing `require github.com/lookatitude/beluga-ai/v2 v(devel)` into `go.mod` is invalid semver and fails `go mod tidy`. The scaffolder must detect this case in `renderFS` (not in the template) and emit a `replace github.com/lookatitude/beluga-ai/v2 => <workspaceRoot>` directive instead. Workspace root is found via `detectWorkspaceRoot` (T3). If workspace root cannot be found (the CLI was `go install`-ed without a tagged version AND is running outside the framework repo), fall back to writing `v0.0.0-unknown` with a stderr warning — the generated project will fail `go build` until the user sets `GOFLAGS="-mod=mod"` or the user's env provides a working module proxy path. Landing place: T3 `renderFS` + T4 `go.mod.tmpl`.

---

### Open questions for workspace-coordinator

Two minor implementation questions surfaced during planning. Both have defensible defaults stated here; flagging them so developer-go doesn't silently pick differently.

1. **`core.ErrConflict` error code does not exist.** `framework/core/errors.go` defines 9 error codes (line 14-38): none of them is "conflict". The plan defaults to using `core.ErrInvalidInput` with a message containing `"target directory %q is not empty; use --force to overwrite"` for the non-empty-target case. An alternative is to add `ErrConflict` to `core/errors.go` — but that expands Layer 1 surface for a Layer 7 concern. Default is `ErrInvalidInput`; developer-go should follow that unless a reviewer objects.

2. **Empty `go.sum` placeholder.** The brief's File-level plan table lists `go.sum placeholder` under "Create". The plan omits `go.sum.tmpl` on the grounds that an empty `go.sum` communicates nothing and `go mod tidy` writes the real file on first `go build`. If Miguel wants the placeholder retained (for parity with some other frameworks' scaffolders that pre-commit empty sum files), add one `go.sum.tmpl` with exactly no content and include it in the golden tree. Default: omit.

Both are low-stakes. No `/consult <specialist>` bounce needed — the brief and specialist inputs fully cover LOO-149's substantive surface.
