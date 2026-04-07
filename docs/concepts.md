# Beluga AI v2 — Core Concepts

This document explains the design decisions and mental models behind the framework. Read this before reading `docs/packages.md` (the "what") or `docs/architecture.md` (the "how").

---

## Agent

An Agent is the atomic unit of reasoning. It encapsulates a language model, a set of tools, a memory backend, a planner strategy, and optional hooks and middleware. Agents implement a single interface:

```go
type Agent interface {
    ID() string
    Persona() Persona
    Tools() []tool.Tool
    Invoke(ctx context.Context, input string, opts ...Option) (string, error)
    Stream(ctx context.Context, input string, opts ...Option) iter.Seq2[Event, error]
}
```

`Invoke` collects the full response before returning. `Stream` yields typed `Event` values as they arrive. Both methods respect `context.Context` cancellation.

Every event has an `EventType`: `EventText` for content chunks, `EventToolCall` and `EventToolResult` for tool interactions, `EventDone` for completion signals, and `EventError` for recoverable errors.

---

## Runner

A Runner is the lifecycle manager for a single agent. It takes an `agent.Agent` and hosts it as a service: managing sessions, applying plugins, enforcing concurrency limits, and providing a graceful shutdown path.

The Runner does not contain business logic — it is pure infrastructure. The agent does the reasoning; the Runner handles everything around it.

```go
import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/runtime"
    "github.com/lookatitude/beluga-ai/schema"
)

runner := runtime.NewRunner(myAgent,
    runtime.WithWorkerPoolSize(10),
    runtime.WithPlugins(auditPlugin, costPlugin),
)
defer func() {
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    if err := runner.Shutdown(shutdownCtx); err != nil {
        fmt.Println("shutdown error:", err)
    }
}()

for evt, err := range runner.Run(ctx, "session-1", schema.NewHumanMessage("summarize this")) {
    if err != nil {
        return err
    }
    fmt.Print(evt.Text)
}
```

The Runner's execution flow for each turn is:

1. Load or create the session identified by `sessionID`.
2. Run the `Plugin` chain's `BeforeTurn` hooks (may modify the input message).
3. Submit the agent invocation to the `WorkerPool`.
4. Stream agent events and collect them.
5. Run the `Plugin` chain's `AfterTurn` hooks (may modify the events).
6. Persist the updated session.
7. Yield the events to the caller.

`Runner.Shutdown` sets a shutdown flag (new calls to `Run` return an error immediately) and drains the worker pool, waiting for all in-flight turns to finish.

**See**: `runtime/` package, `runtime/plugins/`.

---

## Team

A Team is a group of agents coordinated by an `OrchestrationPattern`. Teams implement `agent.Agent`, so a Team can be hosted by a Runner or nested inside another Team.

```go
import "github.com/lookatitude/beluga-ai/runtime"

team := runtime.NewTeam(
    runtime.WithTeamID("research-team"),
    runtime.WithAgents(analyst, summarizer, factChecker),
    runtime.WithPattern(runtime.ScatterGatherPattern(aggregatorAgent)),
)

// Teams are Agents — use a Runner to host them.
runner := runtime.NewRunner(team, runtime.WithWorkerPoolSize(5))
```

Three orchestration patterns are built in:

**PipelinePattern**: Agents execute sequentially. The text output of each agent becomes the input for the next. Use this when each stage refines or extends the previous stage's work.

```go
pipeline := runtime.NewTeam(
    runtime.WithAgents(drafterAgent, editorAgent, reviewerAgent),
    runtime.WithPattern(runtime.PipelinePattern()),
)
```

**SupervisorPattern**: A coordinator agent receives the original input plus a description of available agents. The coordinator decides how to respond. Use this for dynamic task delegation where the routing logic is itself language-model-driven.

```go
supervised := runtime.NewTeam(
    runtime.WithAgents(codeAgent, docsAgent, testAgent),
    runtime.WithPattern(runtime.SupervisorPattern(coordinatorAgent)),
)
```

**ScatterGatherPattern**: All agents run in parallel on the same input. Their outputs are concatenated and passed to an aggregator agent. Use this for tasks where multiple independent analyses should be synthesized.

```go
parallel := runtime.NewTeam(
    runtime.WithAgents(legalAgent, technicalAgent, financialAgent),
    runtime.WithPattern(runtime.ScatterGatherPattern(synthesizerAgent)),
)
```

Custom patterns implement `OrchestrationPattern`:

```go
type OrchestrationPattern interface {
    Execute(ctx context.Context, agents []agent.Agent, input string) iter.Seq2[agent.Event, error]
}
```

**See**: `runtime/` package.

---

## Plugin

A Plugin intercepts agent execution at the Runner level. Every turn calls three methods in order: `BeforeTurn`, `AfterTurn`, and — if an error occurs — `OnError`.

```go
type Plugin interface {
    Name() string
    BeforeTurn(ctx context.Context, session *Session, input schema.Message) (schema.Message, error)
    AfterTurn(ctx context.Context, session *Session, events []agent.Event) ([]agent.Event, error)
    OnError(ctx context.Context, err error) error
}
```

Plugins are stateful, composable, and ordered. The `PluginChain` executes them in registration order for `BeforeTurn` and `AfterTurn`, passing the (potentially modified) message or event slice from one plugin to the next. `OnError` also chains, with each plugin receiving the (potentially replaced) error from the previous.

Built-in plugins cover the most common cross-cutting concerns:

- **AuditPlugin**: writes structured log entries to an `audit.Store` at turn start, turn end, and on error.
- **CostTracking**: records a `cost.Usage` entry after every successful turn.
- **RateLimit**: rejects turns that exceed a requests-per-minute threshold.
- **RetryAndReflect**: retries turns that fail with a retryable error.

To implement a custom plugin:

```go
import (
    "context"

    "github.com/lookatitude/beluga-ai/agent"
    "github.com/lookatitude/beluga-ai/runtime"
    "github.com/lookatitude/beluga-ai/schema"
)

type loggingPlugin struct{}

func (p *loggingPlugin) Name() string { return "logging" }

func (p *loggingPlugin) BeforeTurn(ctx context.Context, session *runtime.Session, input schema.Message) (schema.Message, error) {
    slog.InfoContext(ctx, "turn start", "session", session.ID)
    return input, nil
}

func (p *loggingPlugin) AfterTurn(ctx context.Context, session *runtime.Session, events []agent.Event) ([]agent.Event, error) {
    slog.InfoContext(ctx, "turn end", "session", session.ID, "events", len(events))
    return events, nil
}

func (p *loggingPlugin) OnError(ctx context.Context, err error) error {
    slog.ErrorContext(ctx, "turn error", "error", err)
    return err // return nil to suppress the error
}
```

**See**: `runtime/plugins/`, `audit/`, `cost/`.

---

## Session

A Session holds the full conversation state for one agent interaction: ordered turn history, arbitrary key-value state, and lifecycle timestamps. Sessions are created and managed by a `SessionService`.

```go
type Session struct {
    ID        string
    AgentID   string
    TenantID  string
    State     map[string]any
    Turns     []schema.Turn
    CreatedAt time.Time
    UpdatedAt time.Time
    ExpiresAt time.Time
}
```

The `Runner` automatically creates a new session when `sessionID` is empty, or loads an existing one when a known ID is provided. Sessions are updated after every turn.

The built-in `InMemorySessionService` is suitable for development and single-instance deployments. Implement `SessionService` for distributed or persistent session storage:

```go
type SessionService interface {
    Create(ctx context.Context, agentID string) (*Session, error)
    Get(ctx context.Context, sessionID string) (*Session, error)
    Update(ctx context.Context, session *Session) error
    Delete(ctx context.Context, sessionID string) error
}
```

---

## WorkerPool

A `WorkerPool` bounds the number of agent executions that can run concurrently inside a Runner. It prevents resource exhaustion when many requests arrive simultaneously.

```go
import "github.com/lookatitude/beluga-ai/runtime"

pool := runtime.NewWorkerPool(8)

for _, req := range requests {
    req := req
    if err := pool.Submit(ctx, func(ctx context.Context) {
        if err := processRequest(ctx, req); err != nil {
            slog.ErrorContext(ctx, "request failed", "error", err)
        }
    }); err != nil {
        // ctx cancelled while waiting for a slot
        break
    }
}

pool.Wait() // wait for all submitted work to complete
```

`Submit` blocks until a slot is available or the context is cancelled. `Drain` stops accepting new work and waits for in-flight work to finish, respecting a deadline.

---

## Streaming

All streaming APIs in the framework use `iter.Seq2[T, error]` from the Go standard library. This is a pull-based iterator — the caller controls consumption rate. The convention is:

```go
for value, err := range stream {
    if err != nil {
        // handle error and stop
        break
    }
    // use value
}
```

Producers respect `context.Context` cancellation: if the context is done, the stream yields the context error and stops. Consumers should not call `break` unless they intend to stop consuming — doing so signals the producer to stop as well (via the `yield` bool return).

Channels are never used in public streaming APIs.

**See**: `core/` for stream type definitions.

---

## Registry Pattern

Every extensible package (llm, tool, memory, cost, audit, etc.) uses the same registry pattern. This enables provider plug-ins without modifying core code.

```go
// 1. Register a factory in init():
func init() {
    cost.Register("postgres", func(cfg cost.Config) (cost.Tracker, error) {
        return newPostgresTracker(cfg)
    })
}

// 2. Import for side-effects:
import _ "myorg/beluga-plugins/cost/postgres"

// 3. Construct by name:
tracker, err := cost.New("postgres", cost.Config{})
```

`List()` returns all registered names, sorted, for discovery and debugging.

---

## Errors

Errors in the framework are typed `*core.Error` values with an `ErrorCode` field:

```go
type Error struct {
    Op      string    // operation that failed, e.g. "runtime.runner.run"
    Code    ErrorCode // machine-readable code, e.g. ErrNotFound
    Message string    // human-readable description
    Cause   error     // underlying error, if any
}
```

Use `errors.As` to inspect error codes:

```go
var coreErr *core.Error
if errors.As(err, &coreErr) && coreErr.Code == core.ErrNotFound {
    // handle not found
}
```

Use `core.IsRetryable(err)` before retrying:

```go
if core.IsRetryable(err) {
    // safe to retry with backoff
}
```

Never expose `*core.Error.Cause` in responses to external callers — it may contain internal details.

---

## Deployment Modes

The same agent code runs in four deployment modes. The framework code does not change — only the hosting wrapper changes.

### Library

Import Beluga into a Go program. Construct agents and runners in code. Invoke directly or expose via an HTTP server.

```go
import (
    "github.com/lookatitude/beluga-ai/runtime"
    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

func main() {
    runner := runtime.NewRunner(myAgent)
    // use runner directly or attach to http.ServeMux
}
```

Best for: scripts, CLI tools, single-agent services, development.

### Docker

Generate a multi-stage Dockerfile with `deploy.GenerateDockerfile()`. Each agent runs as a separate container. Wire them with `deploy.GenerateCompose()`.

```go
import "github.com/lookatitude/beluga-ai/deploy"

dockerfile, err := deploy.GenerateDockerfile(deploy.DockerfileConfig{
    AgentConfig: "config/agent.yaml",
    Port:        8080,
})
```

Best for: teams of agents with independent scaling, local orchestration, CI/CD pipelines.

### Kubernetes

Define agents and teams as CRDs (`beluga.ai/v1 Agent` and `beluga.ai/v1 Team`). The operator in `k8s/operator/` reconciles CRDs into Deployments, Services, and HPAs. Validation and mutation webhooks (`k8s/webhooks/`) run before resources are persisted.

```yaml
apiVersion: beluga.ai/v1
kind: Agent
metadata:
  name: planner
  namespace: agents
spec:
  persona:
    role: Planner
  planner: react
  maxIterations: 10
  modelRef: gpt4o-secret
  replicas: 2
  scaling:
    enabled: true
    minReplicas: 1
    maxReplicas: 10
    targetCPUUtilization: 70
```

The `k8s/operator/` reconciler is library-only — it produces desired-state structs that you apply with your Kubernetes client of choice.

Best for: production multi-agent systems, autoscaling, GitOps workflows.

### Temporal

Wrap agent execution in a durable workflow via the `workflow/` package. Handles retries, checkpointing, and resumption across process restarts. Temporal is the cloud provider option; the framework includes a built-in durable engine as well.

Best for: long-running tasks, multi-step pipelines that must survive failures, tasks requiring human-in-the-loop approvals.

---

## Multi-Tenancy

All public functions accept `context.Context` as the first parameter. Tenant identity travels through the context:

```go
ctx = core.WithTenant(ctx, "tenant-abc")
```

Every data type that stores user data (`Session`, `cost.Usage`, `audit.Entry`) has a `TenantID` field. Implementations that query or store data must scope by tenant. The `guard/` pipeline validates tenant identity before any agent execution.

---

## Observability

The framework emits OpenTelemetry spans and metrics using the `gen_ai.*` semantic conventions. The `o11y/` package provides adapters that translate framework events into OTel signals.

No observability code is in `core/` or `schema/` — those packages have zero external dependencies.

Structured logging uses `slog` from the Go standard library. Log entries are contextual: they include span IDs, tenant IDs, agent IDs, and session IDs where available.
