---
title: "Runtime API — Runner, Team, Plugins, Sessions"
description: "Runtime package API reference for Beluga AI. Agent lifecycle management, multi-agent team orchestration, plugin pipeline, session service, and bounded worker pool."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "runtime API, Runner, Team, Plugin, Session, WorkerPool, OrchestrationPattern, PipelinePattern, SupervisorPattern, ScatterGatherPattern, Beluga AI, Go, reference"
---

```go
import "github.com/lookatitude/beluga-ai/runtime"
```

Package runtime provides the lifecycle management layer for individual agents
and coordinated multi-agent teams. A Runner hosts a single agent, managing
sessions, plugins, and bounded concurrency. A Team groups multiple agents and
coordinates them with an OrchestrationPattern.

## Runner

Runner is the production host for a single agent.Agent. It handles session
lookup or creation, routes the input through the plugin chain, dispatches work
to a bounded WorkerPool, and streams agent.Event values back to the caller
as an iter.Seq2 iterator.

```go
import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/runtime"
    "github.com/lookatitude/beluga-ai/schema"
)

runner := runtime.NewRunner(myAgent,
    runtime.WithWorkerPoolSize(20),
    runtime.WithRunnerConfig(runtime.RunnerConfig{
        SessionTTL:              30 * time.Minute,
        GracefulShutdownTimeout: 15 * time.Second,
        StreamingMode:           runtime.StreamingNone,
    }),
)

ctx := context.Background()
for evt, err := range runner.Run(ctx, "session-abc", schema.NewHumanMessage("Hello")) {
    if err != nil {
        fmt.Println("error:", err)
        break
    }
    fmt.Print(evt.Text)
}

if err := runner.Shutdown(ctx); err != nil {
    fmt.Println("shutdown error:", err)
}
```

### Runner Functional Options

| Option | Default | Description |
|--------|---------|-------------|
| `WithPlugins(plugins ...Plugin)` | none | Registers plugins in execution order. |
| `WithSessionService(s SessionService)` | `InMemorySessionService` | Overrides the session backend. |
| `WithRunnerConfig(cfg RunnerConfig)` | see RunnerConfig defaults | Sets the full config struct. |
| `WithWorkerPoolSize(size int)` | 10 | Number of concurrent worker goroutines. Values < 1 are normalized to 1. |

### RunnerConfig Fields

| Field | Default | Description |
|-------|---------|-------------|
| `SessionTTL` | 0 (no expiry) | TTL for sessions created by this runner. |
| `StreamingMode` | `StreamingNone` | Transport hint for callers: `StreamingNone`, `StreamingSSE`, `StreamingWebSocket`. |
| `WorkerPoolSize` | 10 | Concurrent worker slots. |
| `GracefulShutdownTimeout` | 30s | Max wait for in-flight sessions during `Shutdown`. |

### Execution Flow

```
Run(ctx, sessionID, input)
  │
  ├─ 1. Load or create Session
  ├─ 2. PluginChain.RunBeforeTurn  → may modify input
  ├─ 3. agent.Stream               → collect events
  ├─ 4. PluginChain.RunAfterTurn   → may modify events
  ├─ 5. Append turn to session, persist
  └─ 6. Yield events to caller
```

Run returns an error event (not a Go error) if the runner has been shut down
or the context is cancelled before a worker slot is acquired.

## Plugin Interface

Plugin intercepts agent execution at the Runner level. Every method is called
on every turn. Implementations must be lightweight and side-effect-free where
possible.

```go
type Plugin interface {
    Name() string
    BeforeTurn(ctx context.Context, session *Session, input schema.Message) (schema.Message, error)
    AfterTurn(ctx context.Context, session *Session, events []agent.Event) ([]agent.Event, error)
    OnError(ctx context.Context, err error) error
}
```

Implement Plugin to add cross-cutting concerns such as rate limiting, PII
redaction, or audit logging to every agent turn:

```go
import (
    "context"
    "log/slog"

    "github.com/lookatitude/beluga-ai/agent"
    "github.com/lookatitude/beluga-ai/runtime"
    "github.com/lookatitude/beluga-ai/schema"
)

type LoggingPlugin struct{}

func (p *LoggingPlugin) Name() string { return "logging" }

func (p *LoggingPlugin) BeforeTurn(ctx context.Context, s *runtime.Session, input schema.Message) (schema.Message, error) {
    slog.InfoContext(ctx, "before turn", "session", s.ID)
    return input, nil
}

func (p *LoggingPlugin) AfterTurn(ctx context.Context, s *runtime.Session, events []agent.Event) ([]agent.Event, error) {
    slog.InfoContext(ctx, "after turn", "session", s.ID, "events", len(events))
    return events, nil
}

func (p *LoggingPlugin) OnError(ctx context.Context, err error) error {
    slog.ErrorContext(ctx, "agent error", "error", err)
    return err
}
```

### PluginChain

PluginChain manages an ordered sequence of plugins. NewPluginChain copies its
arguments so the original slice cannot be mutated after construction.

- `RunBeforeTurn` — passes the message through each plugin's BeforeTurn in
  registration order. Stops on the first error.
- `RunAfterTurn` — passes the event slice through each plugin's AfterTurn in
  registration order. Stops on the first error.
- `RunOnError` — passes the error through each plugin's OnError in registration
  order. A plugin may return nil to suppress the error for subsequent plugins.

## Session and SessionService

Session holds the full conversation state for one agent instance, including
the ordered turn history and arbitrary key-value data.

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

SessionService manages session lifecycle. Provide a custom implementation to
back sessions with a database or cache:

```go
type SessionService interface {
    Create(ctx context.Context, agentID string) (*Session, error)
    Get(ctx context.Context, sessionID string) (*Session, error)
    Update(ctx context.Context, session *Session) error
    Delete(ctx context.Context, sessionID string) error
}
```

Get returns a `core.Error` with code `ErrNotFound` when the session does not
exist, allowing callers to distinguish not-found from other errors.

The built-in `InMemorySessionService` is created automatically when no
`WithSessionService` option is provided:

```go
svc := runtime.NewInMemorySessionService(
    runtime.WithSessionTTL(1 * time.Hour),
)
```

## Team

Team implements agent.Agent, enabling recursive composition — a Team can
contain other Teams as members. The OrchestrationPattern determines how
members are coordinated.

```go
import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/runtime"
)

team := runtime.NewTeam(
    runtime.WithTeamID("analysis-team"),
    runtime.WithAgents(researchAgent, writerAgent, reviewerAgent),
    runtime.WithPattern(runtime.PipelinePattern()),
)

result, err := team.Invoke(context.Background(), "Summarise Q1 earnings.")
if err != nil {
    fmt.Println("error:", err)
} else {
    fmt.Println(result)
}
```

### Team Functional Options

| Option | Default | Description |
|--------|---------|-------------|
| `WithTeamID(id string)` | `"team"` | Unique identifier for the team. |
| `WithTeamPersona(p agent.Persona)` | zero value | Persona presented when the team is used as a sub-agent. |
| `WithAgents(agents ...agent.Agent)` | none | Member agents in execution order. |
| `WithPattern(p OrchestrationPattern)` | `PipelinePattern()` | Coordination strategy. |
| `WithTeamTools(tools ...tool.Tool)` | none | Additional tools available at the team level. |

## OrchestrationPattern

OrchestrationPattern defines how a Team coordinates its members.

```go
type OrchestrationPattern interface {
    Execute(ctx context.Context, agents []agent.Agent, input string) iter.Seq2[agent.Event, error]
}
```

Three patterns ship in the package:

### PipelinePattern

Executes agents sequentially. The text output of each agent becomes the input
for the next. Use this for multi-step refinement workflows.

```go
pattern := runtime.PipelinePattern()
```

### SupervisorPattern

A coordinator agent receives the original input along with a description of
available agents and delegates work. The coordinator's output is the team's
output.

```go
pattern := runtime.SupervisorPattern(coordinatorAgent)
```

### ScatterGatherPattern

Executes all member agents concurrently with the same input, collects their
outputs, and passes the combined results to an aggregator agent for synthesis.

```go
pattern := runtime.ScatterGatherPattern(aggregatorAgent)
```

## WorkerPool

WorkerPool provides bounded concurrency. It uses a semaphore channel to limit
simultaneous goroutines and a WaitGroup to support graceful drain.

```go
pool := runtime.NewWorkerPool(50) // up to 50 concurrent workers

err := pool.Submit(ctx, func(ctx context.Context) {
    // work that respects ctx cancellation
})
if err != nil {
    // ctx cancelled or pool drained
}

// Drain waits for all in-flight work, respecting ctx deadline.
if err := pool.Drain(ctx); err != nil {
    fmt.Println("drain timed out:", err)
}
```

After `Drain` returns, `Submit` returns an error immediately. `Wait` blocks
until all submitted work finishes without preventing new submissions.

## Related Packages

- `agent` — Agent interface, Event types, Persona, Option.
- `schema` — Message, Turn, ContentPart types.
- `core` — Error types and error codes used by Runner and SessionService.
- `tool` — Tool interface used by Team.
- `docs/concepts.md` — Design decisions, including session model and worker pool rationale.
