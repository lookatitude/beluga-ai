# Reference: CLI

The `beluga` command-line tool is the reference Layer 7 application shipped with the framework. It is a cobra-based CLI with seven subcommands: `version`, `providers`, `init`, `run`, `dev`, `test`, and `deploy`.

Source: [`cmd/beluga/`](../../cmd/beluga/).

## Installation

```bash
go install github.com/lookatitude/beluga-ai/v2/cmd/beluga@latest
```

Pre-built binaries for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, and windows/amd64 are attached to each GitHub release along with a `checksums.txt` (sha256). The build matrix is defined in [`.goreleaser.yml`](../../.goreleaser.yml).

## Root flags

Available on every subcommand.

| Flag | Shorthand | Default | Purpose |
|---|---|---|---|
| `--log-level` | — | `info` | Log level (`debug`, `info`, `warn`, `error`) written to stderr. S1 recognises but does not wire to slog. |
| `--output` | `-o` | `""` | Output format for machine-readable commands. Consumed by `providers` in S1. |

## Subcommands

### `beluga version`

Print the framework version, the Go runtime version, and the compiled-in provider counts.

```
beluga (devel)
go1.25.9
providers: llm=3 embedding=2 vectorstore=1 memory=4
```

Version resolution precedence: goreleaser ldflags > `runtime/debug.ReadBuildInfo` (populated by `go install ...@vX.Y.Z`) > `"(devel)"`. Implementation: [`cmd/beluga/internal/version/version.go`](../../cmd/beluga/internal/version/version.go).

### `beluga providers`

List the providers compiled into this binary. By default, emits tabwriter-aligned rows grouped by category (`llm`, `embedding`, `vectorstore`, `memory`).

```bash
beluga providers
```

With `--output json`, emits a stable JSON array filterable with `jq`:

```bash
beluga providers --output json | jq '.[] | select(.category=="llm") | .providers'
```

```json
[
  {"category": "llm",         "providers": ["anthropic", "ollama", "openai"]},
  {"category": "embedding",   "providers": ["ollama", "openai"]},
  {"category": "vectorstore", "providers": ["inmemory"]},
  {"category": "memory",      "providers": ["archival", "composite", "core", "recall"]}
]
```

Unsupported formats return exit 1 with `"unsupported output format"` on stderr.

### `beluga init`

Scaffold a new Beluga agent project.

| Flag | Default | Purpose |
|---|---|---|
| `--name` | derived from `--dir` | Project name (used to generate the agent ID). |
| `--dir` | `.` | Output directory. Rejects path traversal (`..`) and absolute paths outside the CWD. |

Creates `agents/`, `tools/`, `config/agent.json`, and `main.go`.

### `beluga run`

Build the scaffolded project at `--project-root` and exec the resulting binary once. The child inherits the current environment plus any `KEY=value` entries from `<root>/.env`. `beluga run` exits with the child's exit code, so shell `&&` chains and CI gates behave as they would with a hand-rolled `go run .`.

| Flag | Default | Purpose |
|---|---|---|
| `--project-root` | `.` | Directory containing `go.mod` + `.beluga/project.yaml`. Resolved via `filepath.Abs`. |

Arguments after `--` are forwarded to the child binary as `argv[1:]`:

```bash
beluga run --project-root . -- --agent-id=worker-1 --input="hello"
```

**Environment contract.** The child's environment is layered deterministically: `os.Environ()` → `.env` entries (in file order) → `Config.ExtraEnv` overlays. Later sources win for duplicate keys. Keys must match the POSIX identifier grammar (`[A-Za-z_][A-Za-z0-9_]*`); malformed lines return a `.env:<line>: expected KEY=value` error rather than silently skipping. Values may be single- or double-quoted; inside double quotes `\n`, `\r`, `\t`, and `\"` escape sequences are expanded. A `.env` symlinked outside the project root returns a `.env escapes project root` error and aborts the run.

`beluga run` never logs env values. The `.env` file is read once per invocation and its contents flow directly into `exec.Cmd.Env`.

**Signal handling.** `SIGINT` and `SIGTERM` received by the parent are translated into a graceful shutdown on the child via `terminateGracefully`: the child gets `SIGTERM`, waits up to `GraceTimeout` (default 3s), then escalates to `SIGKILL`. On Windows the equivalent ctrl-event is sent via the platform-specific `exec_windows.go` path.

**Exit code forwarding.** `devloop.ExitCode` extracts `*exec.ExitError.ExitCode()` when the child exited non-zero and returns it unchanged; build/start errors return `1`. Source: [`cmd/beluga/devloop/supervisor.go`](../../cmd/beluga/devloop/supervisor.go).

### `beluga dev`

Watch the scaffolded project and restart the child on Go source changes. `beluga dev` is the same binary-supervision path as `beluga run` plus an fsnotify watcher rooted at `--project-root`: events that pass the `GoSourceFilter` (accepts `*.go` writes, ignores `vendor/`, `node_modules/`, hidden directories) arm a 500ms debounce timer; on fire, the prior child is terminated (SIGTERM → 3s grace → SIGKILL), the binary is rebuilt, and a fresh child is started.

| Flag | Default | Purpose |
|---|---|---|
| `--project-root` | `.` | Directory containing `go.mod` + `.beluga/project.yaml`. |
| `--playground` | `8089` | Playground port. `off` disables the dev UI entirely; `0` picks an ephemeral port; any valid integer `0-65535` binds explicitly. |

Arguments after `--` are forwarded to every spawned child, identical to `beluga run`.

**Playground.** When `--playground` is not `off`, `beluga dev` starts a loopback-only (`127.0.0.1:<port>`) HTTP server that surfaces three panels: OTel spans (populated when the child runs `o11y.BootstrapFromEnv` and emits `gen_ai.*` spans — see the [Canonical consumer shape](../architecture/01-overview.md#canonical-consumer-shape)), a stderr tail, and a token/cost counter. The server enforces an exact same-origin check on POSTs and an exact-origin CORS policy; requests from any address other than `127.0.0.1` are rejected. The URL is exported to the child as `BELUGA_PLAYGROUND_URL=http://127.0.0.1:<port>` so an agent can link back to its own spans.

**Restart semantics.** Child exits are not fatal during `beluga dev` — a crash is logged and the watcher keeps running, so saving the file that fixes the bug recovers the session without restarting the supervisor. The restart sequence number flows through `OnRestart(seq)` and becomes the first line of each playground stderr frame (`--- restart #N ---`), so you can correlate an event with the exact child that produced it.

**Platform coverage.** Linux and macOS are covered by the `beluga run` matrix plus a dedicated `beluga-dev-smoke` CI job that starts `beluga dev --playground=off`, appends a line to `main.go`, and asserts the child runs to completion at least twice before the supervisor is torn down with `SIGTERM`. Windows receives the `beluga run` signal-handling path via `exec_windows.go`; a Windows-native `beluga dev` smoke is tracked as an S3.5 follow-up.

### `beluga test`

Run agent tests via `go test`. Resolves the toolchain via `exec.LookPath("go")` so a mutated `PATH` cannot redirect the binary.

| Flag | Shorthand | Default | Purpose |
|---|---|---|---|
| `--verbose` | `-v` | `false` | Verbose test output. |
| `--race` | — | `false` | Enable race detector. |
| `--pkg` | — | `./...` | Package pattern. Restricted by regex (`^[A-Za-z0-9_./\-]+(\.\.\.)?$`) to block flag smuggling. |

**Canonical test env.** Every `beluga test` invocation appends the following entries to `os.Environ()` before handing them to the `go test` child — the scaffolded project is expected to honour them:

| Key | Value | Purpose |
|---|---|---|
| `BELUGA_ENV` | `test` | Selects the test profile in `config.Load` and disables production-only middleware. |
| `BELUGA_LLM_PROVIDER` | `mock` | Routes `llm.New` to the deterministic [mock provider](../../llm/providers/mock/) so tests run offline, no API key required. |
| `OTEL_SDK_DISABLED` | `true` | `o11y.BootstrapFromEnv` skips exporter initialisation and returns a no-op shutdown, so `go test -v` output stays clean. |

Source: `canonicalTestEnv` in [`cmd/beluga/test.go`](../../cmd/beluga/test.go). Tests can assert the exact values via `os.Environ()` if they need to distinguish a `beluga test` run from a bare `go test`.

### `beluga deploy`

Generate deployment artifacts. S1 is a stub — prints what would be written without creating files.

| Flag | Default | Purpose |
|---|---|---|
| `--target` | `docker` | Target (`docker`, `compose`, `k8s`). Unknown targets error. |
| `--config` | `config/agent.json` | Agent config file. |
| `--output` | `.` | Output directory for artifacts. Shadows the root `--output` inside this subcommand. |

## Bundled providers

The binary ships with a curated, CGo-free provider set: [`cmd/beluga/providers/providers.go`](../../cmd/beluga/providers/providers.go). Every addition to this list requires a CGo-free audit — see the package-level comment for details.

To build a binary with a different provider set, write your own `main` package and blank-import the provider packages you need.

## Related

- [Dev-loop guide](../guides/dev-loop.md) — task-oriented walkthrough of `beluga run` + `beluga dev` + `beluga test`.
- [Providers catalog](./providers.md) — every registered provider across categories.
- [Architecture Overview — Layer 7](../architecture/01-overview.md#layer-7--application) — where the CLI fits in the stack.
- [Goreleaser config](../../.goreleaser.yml) — the release build matrix.
