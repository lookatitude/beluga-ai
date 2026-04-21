# Dev Loop — `beluga run`, `beluga dev`, and `beluga test`

**You will build:** a scaffolded Beluga project, run it once with `beluga run`, hot-rebuild it on every save with `beluga dev`, and drive its tests with `beluga test` — no API key required.
**Prerequisites:** Go 1.25+, a working `beluga` CLI on `$PATH` (`go install github.com/lookatitude/beluga-ai/v2/cmd/beluga@latest`).
**Related:** [First Agent](./first-agent.md), [CLI reference](../reference/cli.md), [Architecture Overview — Layer 7](../architecture/01-overview.md#layer-7--application).

The three subcommands in this guide share one supervisor — [`cmd/beluga/devloop/`](../../cmd/beluga/devloop/) — and one deterministic LLM backend — [`llm/providers/mock`](../../llm/providers/mock/). Learn them once; they behave the same across Linux, macOS, and Windows.

## 1. Scaffold a project

```bash
beluga init hello-beluga && cd hello-beluga
```

`beluga init` generates `main.go`, `go.mod`, `config/agent.json`, an `.env.example`, and a `Makefile`. The scaffolded `main.go` follows the [canonical consumer shape](../architecture/01-overview.md#canonical-consumer-shape): it calls `o11y.BootstrapFromEnv(ctx, "hello-beluga")` at startup and wraps its agent in `agent.ApplyMiddleware(..., agent.WithTracing())` so every agent invocation emits OTel GenAI spans.

## 2. Run it once — `beluga run`

```bash
cp .env.example .env
beluga run
```

`beluga run` does three things:

1. Reads `.env` — each `KEY=value` line is layered onto `os.Environ()` in file order. Malformed lines (`FOO=bar;SOMETHING ELSE`) produce a named `.env:<line>: expected KEY=value` error rather than silently skipping. A `.env` symlinked outside the project root is rejected with `.env escapes project root`.
2. Runs `go build -o <tmp> .` rooted at `--project-root` (default `.`). Build output streams to the parent's stderr.
3. Execs the compiled binary with the merged environment. Parent `SIGINT`/`SIGTERM` translate into a graceful child shutdown (SIGTERM → 3s grace → SIGKILL). The parent exits with the child's exit code.

Arguments after `--` are forwarded to the child as `argv[1:]`:

```bash
beluga run -- --agent-id=worker-1 --input="hello"
```

`beluga run` **never logs env values**. Keys go into the child's environment; the only observable side effect in the parent's stdout/stderr is whatever the child itself prints.

## 3. Edit → save → see it restart — `beluga dev`

In one terminal:

```bash
beluga dev
# beluga dev: playground at http://127.0.0.1:8089
# ...FIXTURE_RAN: Always return the canned mock response.
```

In a second terminal, edit `main.go` — add a `fmt.Println("tick")` anywhere in `main`. Save. `beluga dev` observes the write, debounces for 500ms, terminates the previous child, rebuilds, and starts a fresh child. The playground stderr tail marks the boundary with `--- restart #N ---`.

### What's running

- **fsnotify watcher** — rooted at `--project-root`, recursively registering every directory except `vendor/`, `node_modules/`, and any hidden directory (`.git/`, `.idea/`, etc.). See [`supervisor.go:242`](../../cmd/beluga/devloop/supervisor.go).
- **`GoSourceFilter`** — accepts `*.go` writes, ignores everything else. Edit a README and the loop does not fire.
- **Debounce** — 500ms quiet period after the *last* accepted event. Saving five files in a single `:wqa` burst triggers one rebuild, not five.
- **Playground** — loopback-only HTTP server at `http://127.0.0.1:<port>`. Exact-origin CORS, same-origin POSTs, and a `127.0.0.1` bind: connections from any other address are rejected at the listener, not the handler.

### Playground flag

| Value | Meaning |
|---|---|
| `8089` (default) | Bind `127.0.0.1:8089`. |
| `0` | Ephemeral port — `127.0.0.1:<auto>` — useful when running multiple projects on the same machine. |
| `off` | Do not start the playground. `beluga dev` still rebuilds and restarts; stderr/stdout go directly to the parent. |
| any integer `1-65535` | Bind that specific port on `127.0.0.1`. |

The child receives `BELUGA_PLAYGROUND_URL=http://127.0.0.1:<port>` in its environment, so an agent can print a link back to its own trace view.

### Crash recovery

A child that exits non-zero during `beluga dev` is **not fatal**. The exit is logged (`devloop: child exited: exit status 1`) and the watcher keeps running. Saving the file that fixes the bug recovers the session — you never need to restart the supervisor itself.

## 4. Drive tests with the mock provider — `beluga test`

```bash
beluga test -v
```

`beluga test` is `go test` with three environment variables appended:

| Key | Value | Effect |
|---|---|---|
| `BELUGA_ENV` | `test` | Selects the test profile in `config.Load`; disables production-only middleware. |
| `BELUGA_LLM_PROVIDER` | `mock` | Routes `llm.New` to the [mock provider](../../llm/providers/mock/) — no API key, no network, no flakes. |
| `OTEL_SDK_DISABLED` | `true` | `o11y.BootstrapFromEnv` returns the no-op shutdown so `go test -v` output is not polluted with span JSON. |

Source: `canonicalTestEnv` in [`cmd/beluga/test.go`](../../cmd/beluga/test.go).

### Writing a mock-backed test

The mock provider implements `llm.ChatModel` with a FIFO fixture queue — each call to `Generate` pops one `Fixture` off the queue, and exhaustion returns a deterministic `"mock: fixture queue exhausted"` final answer that the ReAct planner interprets as an `ActionFinish`, so the agent exits cleanly instead of deadlocking.

```go
// agent_test.go
package main

import (
    "context"
    "strings"
    "testing"

    "github.com/lookatitude/beluga-ai/v2/agent"
    "github.com/lookatitude/beluga-ai/v2/config"
    "github.com/lookatitude/beluga-ai/v2/llm"

    _ "github.com/lookatitude/beluga-ai/v2/llm/providers/mock"
)

func TestAgentAnswersHello(t *testing.T) {
    model, err := llm.New("mock", config.ProviderConfig{
        Provider: "mock",
        Model:    "mock-test",
    })
    if err != nil {
        t.Fatalf("llm.New: %v", err)
    }

    a := agent.New("hello-agent",
        agent.WithLLM(model),
        agent.WithPersona(agent.Persona{Role: "fixture"}),
    )

    out, err := a.Invoke(context.Background(), "hello")
    if err != nil {
        t.Fatalf("Invoke: %v", err)
    }
    if !strings.Contains(out, "mock") {
        t.Fatalf("expected mock response, got %q", out)
    }
}
```

Run it:

```bash
beluga test -v --race
```

The `Makefile` generated by `beluga init` defines a `test-ci` target that pre-sets the same three env vars alongside `go test -race ./...`, so CI jobs can invoke `make test-ci` directly without calling the `beluga` binary.

## 5. Observability

Both `beluga run` and `beluga dev` invoke the child with the same environment contract, so the same variables control exporter selection for both:

| Variable | Effect |
|---|---|
| `OTEL_SDK_DISABLED=true` | `o11y.BootstrapFromEnv` returns the no-op shutdown. No exporter attached. |
| `OTEL_EXPORTER_OTLP_ENDPOINT=<url>` | OTLP/HTTP exporter. The SDK reads `OTEL_EXPORTER_OTLP_HEADERS`, `OTEL_EXPORTER_OTLP_TIMEOUT`, etc. directly. |
| `BELUGA_OTEL_STDOUT=1` | Pretty-printed stdout JSON exporter. Intended for local `beluga dev` debugging — **never** for production. |
| (none of the above) | No exporter attached. Spans are silent no-ops. |

Resolution order: explicit `WithSpanExporter` in code > `OTEL_SDK_DISABLED` > `OTEL_EXPORTER_OTLP_ENDPOINT` > `BELUGA_OTEL_STDOUT` > no-op. Source: [`o11y/bootstrap.go`](../../o11y/bootstrap.go).

The scaffolded `.env.example` ships with `BELUGA_OTEL_STDOUT=1` uncommented so first-run tracing works without configuration. Disable it for production builds — stdout span emission is a debugging convenience, not a production export path.

## 6. `.env` conventions

`beluga run` and `beluga dev` read `.env` from `--project-root` and layer it on top of `os.Environ()`:

- `KEY=value` per line. Blank lines and `# comment` lines are ignored.
- Values may be single- or double-quoted to preserve surrounding whitespace. Inside double quotes, `\n`, `\r`, `\t`, `\"`, and `\\` escape sequences are expanded.
- Trailing inline ` # comment` is stripped from unquoted values.
- Keys must match `[A-Za-z_][A-Za-z0-9_]*` — shell metacharacters and leading digits are rejected.
- Malformed lines return a `.env:<line>: expected KEY=value` error; lines are not silently skipped.
- `.env` is resolved via `filepath.EvalSymlinks`; a symlinked `.env` whose target falls outside the project root is rejected with `.env escapes project root: <path>`.

**`.env.local` is not auto-loaded.** Unlike Next.js or Vite, Beluga deliberately loads only `.env` — no `.env.local`, no `.env.production`, no implicit merge. The rationale is prevent-the-footgun: production credentials should never leak into `beluga dev` via an accidentally-committed `.env.local`. Use `--env <path>` (deferred to a future release) or source-switch by checkout if you need per-profile overrides.

## 7. Next steps

- Read the [CLI reference](../reference/cli.md) for every flag and its source file.
- Read [First Agent](./first-agent.md) to plug in a real LLM provider once you're past the dev-loop skeleton.
- Read the [Architecture Overview — Layer 7](../architecture/01-overview.md#layer-7--application) to see how the CLI composes Layers 1–6.

## Common mistakes

- **`touch main.go` does not trigger a rebuild.** On some Linux kernels, fsnotify does not report mtime-only changes as `Write` events. Use a content-modifying edit — `echo "// tick" >> main.go` — or save from your editor, which always writes.
- **`beluga dev --playground=8089` on a busy port.** The bind fails with `listen tcp 127.0.0.1:8089: address already in use`. Use `--playground=0` (ephemeral) or pick another port. `--playground=off` disables the UI entirely.
- **Expecting the playground on `localhost` or `0.0.0.0`.** The server binds `127.0.0.1` exclusively. A request to `http://localhost:8089` works on most systems because the resolver prefers IPv4 loopback, but a request to `http://0.0.0.0:8089` does not — the listener refuses it by design.
- **Setting `OPENAI_API_KEY` in `.env` for `beluga test`.** `beluga test` pins `BELUGA_LLM_PROVIDER=mock`, which routes `llm.New` through the mock provider before any API-key check. Real providers do not run under `beluga test` unless you override `BELUGA_LLM_PROVIDER` in the child explicitly.
- **Relying on the mock fixture queue for production tests.** The queue is FIFO with a deterministic `ActionFinish` fallback on exhaustion. Fine for "does this agent wire up correctly?" tests — misleading for "does this agent reason correctly?" evaluation, which belongs in `eval/`.
