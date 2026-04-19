# Reference: CLI

The `beluga` command-line tool is the reference Layer 7 application shipped with the framework. It is a cobra-based CLI with six subcommands: `version`, `providers`, `init`, `dev`, `test`, and `deploy`.

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

### `beluga dev`

Start the development server (playground + hot reload). S1 prints the configuration; the full server lands in S3.

| Flag | Default | Purpose |
|---|---|---|
| `--port` | `8080` | Server port. |
| `--config` | `config/agent.json` | Agent config file. |

### `beluga test`

Run agent tests via `go test`. Resolves the toolchain via `exec.LookPath("go")` so a mutated `PATH` cannot redirect the binary.

| Flag | Shorthand | Default | Purpose |
|---|---|---|---|
| `--verbose` | `-v` | `false` | Verbose test output. |
| `--race` | — | `false` | Enable race detector. |
| `--pkg` | — | `./...` | Package pattern. Restricted by regex to block flag smuggling. |

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

- [Providers catalog](./providers.md) — every registered provider across categories.
- [Architecture Overview — Layer 7](../architecture/01-overview.md#layer-7--application) — where the CLI fits in the stack.
- [Goreleaser config](../../.goreleaser.yml) — the release build matrix.
