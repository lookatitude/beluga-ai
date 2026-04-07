# Beluga AI v2 — Architecture

This document describes the architecture of the framework: how the components fit together, how data flows through the system, and how to extend each layer. For design decisions (the "why"), see `docs/concepts.md`. For the package API surface (the "what"), see `docs/packages.md`.

---

## Package Layering

Dependencies flow strictly downward. No package may create a circular import.

```
internal/
   └── core/
         └── schema/
               ├── config/
               ├── o11y/
               ├── tool/
               ├── llm/
               ├── memory/
               ├── rag/
               ├── guard/
               ├── resilience/
               └── cache/
                     └── agent/
                           ├── orchestration/
                           └── workflow/
                                 └── runtime/
                                       ├── cost/
                                       ├── audit/
                                       ├── protocol/
                                       ├── hitl/
                                       ├── auth/
                                       ├── eval/
                                       ├── state/
                                       ├── prompt/
                                       ├── server/
                                       ├── voice/
                                       ├── deploy/
                                       └── k8s/
```

`core/` and `schema/` have zero external dependencies beyond stdlib and OpenTelemetry. `runtime/` may import `cost/`, `audit/`, and `agent/`. `deploy/` and `k8s/` sit at the top and may import anything below.

---

## Streaming Architecture

Every streaming API produces `iter.Seq2[T, error]`. This is a pull-based iterator from the Go standard library (`iter` package, Go 1.23+).

```
Producer                           Consumer
────────                           ────────
func(yield func(T, error) bool) {  for value, err := range stream {
    // context-aware loop              if err != nil { break }
    for each event {                   // use value
        if !yield(event, nil) {    }
            return // consumer done
        }
    }
}
```

Key properties:

- **Pull-based**: the consumer drives the pace. No buffering by default.
- **Cancellable**: producers check `ctx.Err()` at every iteration step.
- **Composable**: `iter.Seq2` values can be wrapped, filtered, and transformed by returning a new `iter.Seq2`.
- **No goroutine required**: the producer function runs in the consumer's goroutine unless the implementation explicitly launches one.

Channels are not used in public streaming APIs. The Runner uses an internal channel to bridge the worker pool goroutine and the `iter.Seq2` returned to the caller, but this is an implementation detail hidden behind the interface.

---

## Runtime Layer

The runtime layer (`runtime/`) is where agent definitions become running processes.

### Runner

The Runner is the lifecycle manager for a single agent. It owns:

- A `WorkerPool` for bounded concurrency.
- A `SessionService` for conversation state.
- A `PluginChain` for cross-cutting concerns.

Turn execution flow:

```
caller
  │
  ▼
Runner.Run(ctx, sessionID, input)
  │
  ├── [shutdown check] → error if shut down
  │
  ├── WorkerPool.Submit(ctx, fn)
  │     │
  │     └── executeTurn(ctx, sessionID, input)
  │           │
  │           ├── SessionService.Get or Create
  │           ├── PluginChain.RunBeforeTurn → (modified input)
  │           ├── agent.Stream → collect []Event
  │           ├── PluginChain.RunAfterTurn → (modified events)
  │           └── SessionService.Update
  │
  └── yield events to caller (iter.Seq2)
```

If any step errors, the `PluginChain.RunOnError` hook fires before the error reaches the caller.

### Team

A Team wraps multiple agents behind the `agent.Agent` interface. Because Teams implement the same interface, they compose recursively: a Team can be a member of another Team, and a Runner hosts either an Agent or a Team without knowing which.

```
Runner
  └── Team (root)
        ├── AgentA
        ├── Team (nested)
        │     ├── AgentB
        │     └── AgentC
        └── AgentD
```

The `OrchestrationPattern` is the only behavioral variable:

```go
type OrchestrationPattern interface {
    Execute(ctx context.Context, agents []agent.Agent, input string) iter.Seq2[agent.Event, error]
}
```

Built-in patterns:

| Pattern | Concurrency | Output |
|---------|-------------|--------|
| `PipelinePattern` | sequential | output of last agent |
| `SupervisorPattern(coordinator)` | sequential (coordinator decides) | coordinator's output |
| `ScatterGatherPattern(aggregator)` | parallel | aggregator's synthesis |

### Plugin System

Plugins intercept every turn at three hook points:

```
BeforeTurn → [agent execution] → AfterTurn
                     ↓ (on error)
                  OnError
```

The `PluginChain` threads the (potentially modified) input message through all `BeforeTurn` implementations in registration order, and threads events through all `AfterTurn` implementations in the same order. Errors short-circuit the chain.

Because plugins compose via `PluginChain`, you can assemble arbitrary combinations:

```go
import (
    "github.com/lookatitude/beluga-ai/runtime"
    "github.com/lookatitude/beluga-ai/runtime/plugins"
)

chain := runtime.NewPluginChain(
    plugins.NewRateLimit(120),       // 120 RPM — runs first
    plugins.NewAuditPlugin(store),   // logs every turn
    plugins.NewCostTracking(tracker, budget), // records usage
)
runner := runtime.NewRunner(myAgent, runtime.WithPlugins(
    plugins.NewRateLimit(120),
    plugins.NewAuditPlugin(store),
    plugins.NewCostTracking(tracker, budget),
))
```

### WorkerPool

The `WorkerPool` limits concurrent agent executions with a semaphore:

```
Submit(ctx, fn)
  │
  ├── [drained?] → error
  ├── [acquire semaphore slot, blocking until available or ctx done]
  ├── wg.Add(1)
  └── go func() {
        defer wg.Done()
        defer release semaphore slot
        fn(ctx)
      }()
```

`Drain` sets the drained flag and calls `wg.Wait()` with an optional deadline. After drain, `Submit` always returns an error.

Default pool size is 10. Override with `WithWorkerPoolSize(n)`.

---

## Cost and Audit Layer

### Cost Tracking

`cost.Tracker` is a write-then-query interface. Implementations store `Usage` records and return `Summary` aggregates. The `InMemoryBudgetChecker` queries the tracker for rolling-window totals before approving each request.

```
Request
  │
  ▼
BudgetChecker.Check(budget, estimated)
  │
  ├── tracker.Query(hourFilter) → hourly totals
  ├── [projected tokens > MaxTokensPerHour?] → BudgetDecision{Allowed: false}
  ├── tracker.Query(dayFilter) → daily totals
  └── [projected cost > MaxCostPerDay?] → BudgetDecision{Allowed: false}
  │
  ▼
BudgetDecision{Allowed: true}
```

The `CostTracking` plugin records usage after `AfterTurn` completes. Tracker `Record` errors are ignored so storage failures never block agent execution.

### Audit Logging

`audit.Store` extends `audit.Logger` with `Query`. The `InMemoryStore` bounds memory growth by evicting the oldest entries when `maxEntries` is exceeded.

The `AuditPlugin` writes three event types per turn:
- `"agent.turn.start"` in `BeforeTurn` (before any modification)
- `"agent.turn.end"` in `AfterTurn` (after all modifications)
- `"agent.turn.error"` in `OnError`

Entry IDs are generated with `crypto/rand` — never `math/rand`.

---

## Deployment Architecture

Four deployment modes use the same framework code. The agent implementation does not change.

### Mode 1: Library

Construct agents and runners in Go code. Invoke `runner.Run()` directly or attach to `http.ServeMux`.

```go
import (
    "net/http"

    "github.com/lookatitude/beluga-ai/runtime"
    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

func main() {
    runner := runtime.NewRunner(myAgent)

    http.HandleFunc("/run", func(w http.ResponseWriter, r *http.Request) {
        // parse input, stream runner.Run() as SSE
    })
    http.ListenAndServe(":8080", nil)
}
```

### Mode 2: Docker

`deploy.GenerateDockerfile` produces a multi-stage build:

```
Stage 1: golang:1.23 (builder)
  COPY go.mod go.sum → go mod download
  COPY . .
  RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /agent ./cmd/agent

Stage 2: gcr.io/distroless/static-debian12 (runtime)
  COPY --from=builder /agent /app/agent
  COPY config/agent.yaml /config/
  USER nonroot:nonroot
  EXPOSE 8080
  ENTRYPOINT ["/app/agent"]
```

`deploy.GenerateCompose` wires multiple agent containers with shared networking and dependency ordering. All string inputs are validated against safe character sets before being written to generated files.

`deploy.NewHealthEndpoint` exposes `/healthz` (liveness) and `/readyz` (readiness) for orchestrators. Readiness checks run with a 5-second per-check deadline. Error details are suppressed in the HTTP response.

### Mode 3: Kubernetes

The `k8s/operator/` package defines CRD types and a `Reconciler` interface with no Kubernetes library dependencies. A controller watches `Agent` and `Team` CRDs and calls the reconciler to produce desired-state structs:

```
CRD change detected
  │
  ▼
Reconciler.ReconcileAgent(ctx, agentResource)
  │
  ├── derive DeploymentSpec (image, replicas, env, resources, ports)
  ├── derive ServiceSpec (ClusterIP, port mapping)
  └── [scaling.Enabled?] derive HPASpec (minReplicas, maxReplicas, targetCPU)
  │
  ▼
ReconcileResult → apply via Kubernetes client
```

Admission webhooks (`k8s/webhooks/`) validate and mutate resources before persistence:

- **Validation**: rejects resources with missing required fields or unrecognized planner/pattern values.
- **Mutation**: sets defaults (`MaxIterations=10`) and applies standard labels.

Both webhook functions operate on plain Go structs — no HTTP servers, no Kubernetes libraries required in the package itself.

### Mode 4: Temporal

Wrap `runner.Run()` in a Temporal workflow activity. The `workflow/` package provides a built-in durable engine with Temporal as an optional backend.

```
Temporal Worker
  └── Activity: RunAgentTurn(ctx, sessionID, input)
        └── runner.Run(ctx, sessionID, input)
```

Temporal handles retries, checkpointing, and human-in-the-loop signaling. The agent implementation is unaware of the durability layer.

---

## Performance Architecture

### Event Pool

Agent events (`agent.Event`) are frequently allocated during streaming. The `internal/syncutil` package provides a `sync.Pool` for `Event` values to reduce GC pressure on hot paths.

### Worker Pool Semaphore

The `WorkerPool` uses a buffered channel as a counting semaphore. Slot acquisition is a channel send (`p.sem <- struct{}{}`), which parks the goroutine until capacity is available. Release is a channel receive (`<-p.sem`) in a deferred function.

This avoids a `sync.Mutex` and allows the semaphore to participate in `select` statements for context cancellation.

### Tool DAG

When an agent has multiple tools, the `orchestration/` package can represent tool dependencies as a directed acyclic graph. Independent tools execute in parallel; dependent tools wait for their prerequisites. The scheduler uses the `WorkerPool` for bounded concurrency within the DAG.

---

## Extension Points

Every layer has a defined extension contract.

### Adding an LLM Provider

1. Create a package under `llm/providers/yourprovider/`.
2. Implement `llm.ChatModel`.
3. In `init()`, call `llm.Register("yourprovider", factory)`.
4. Callers import with `_ "github.com/lookatitude/beluga-ai/llm/providers/yourprovider"`.

### Adding a Cost Tracker Backend

1. Implement `cost.Tracker`.
2. In `init()`, call `cost.Register("yourbackend", factory)`.

### Adding an Audit Store Backend

1. Implement `audit.Store`.
2. In `init()`, call `audit.Register("yourbackend", factory)`.

Note: `audit.Register` panics on duplicate registration. Each backend must be registered exactly once.

### Adding a Plugin

Implement `runtime.Plugin` and pass the instance to `runtime.WithPlugins()`.

### Adding an Orchestration Pattern

Implement `runtime.OrchestrationPattern` and pass the instance to `runtime.WithPattern()`.

### Adding a Kubernetes CRD Reconciler

Implement `operator.Reconciler` and wire it to your controller-manager. The built-in `operator.NewDefaultReconciler()` can be used as a starting point or baseline.

---

## Security Architecture

The security model is layered:

```
External input
      │
      ▼
[Plugin.BeforeTurn — rate limit, auth check]
      │
      ▼
[Guard.InputGuard — injection detection, spotlighting]
      │
      ▼
[Agent.Stream — LLM call]
      │
      ▼
[Guard.OutputGuard — content filtering]
      │
      ▼
[Guard.ToolGuard — tool call validation before execution]
      │
      ▼
[Plugin.AfterTurn — audit, cost recording]
      │
      ▼
Caller
```

Key invariants:

- `context.Context` carries tenant identity. Every data access scopes by tenant.
- Secrets travel through Kubernetes Secret references (`APIKeyRef`), not config fields.
- `audit.Entry.Input` and `audit.Entry.Output` must be redacted by the caller before logging.
- Health endpoint responses suppress check error details to prevent information disclosure.
- Generated Dockerfile and Compose YAML inputs are validated against safe character sets before rendering.
- `crypto/rand` is used for all ID generation. `math/rand` is never used for security purposes.

---

## Testing Conventions

- Unit tests live alongside source in `*_test.go` files.
- Integration tests use `//go:build integration`.
- Every public interface has a mock in `internal/testutil/`.
- Table-driven tests cover: happy path, error paths, context cancellation, concurrent access.
- Benchmarks exist for hot paths: worker pool submission, plugin chain execution, event streaming.

Run the test suite:

```bash
go test ./...                          # unit tests
go test -tags integration ./...        # include integration tests
go test -race ./...                    # race detector
go test -bench=. ./runtime/...         # benchmarks
```
