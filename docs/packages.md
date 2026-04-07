# Beluga AI v2 — Package Reference

This document describes every package in the framework. Each entry covers purpose, primary interfaces, and relationships with other packages. For architectural context and design decisions, see `docs/architecture.md` and `docs/concepts.md`.

---

## Core Layer

### `core/`

Foundation for the entire framework. Defines `Runnable`, stream types, lifecycle interfaces (`Start`/`Stop`/`Health`), typed errors with `ErrorCode`, and multi-tenancy utilities.

Zero external dependencies beyond stdlib and OpenTelemetry.

**Key types**: `Error`, `ErrorCode`, `IsRetryable()`, `WithTenant()`.

**Rule**: No package in the framework may import a package that imports `core/` back — dependency only flows downward.

---

### `schema/`

Shared data types used across all packages. Defines `Message`, `ContentPart`, `Tool`, `Document`, `Event`, `Session`, `Turn`, and related structs.

Zero external dependencies beyond stdlib and OpenTelemetry.

**Key types**: `Message`, `AIMessage`, `HumanMessage`, `SystemMessage`, `ContentPart`, `TextPart`, `ToolCall`, `Turn`.

---

### `config/`

Generic configuration loading with hot-reload. Provides `Load[T]()` for unmarshalling YAML/JSON into typed structs, `Validate()` for struct validation, and a file-watcher for live updates.

---

### `o11y/`

OpenTelemetry integration following GenAI semantic conventions (`gen_ai.*` namespace). Provides structured logging via `slog`, adapter interfaces for tracing, and metric helpers.

---

## Inference

### `llm/`

`ChatModel` interface, router, structured output, and context manager. Provider implementations live under `llm/providers/`.

**Key interfaces**: `ChatModel`, `Router`, `ContextManager`.

**Key options**: `WithTemperature()`, `WithMaxTokens()`, `WithSystemPrompt()`, `WithTools()`.

**Extension**: Implement `ChatModel` → call `llm.Register("name", factory)` in `init()` → import with `_`.

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/openai"

model, err := llm.New("openai", llm.ProviderConfig{Model: "gpt-4o"})
if err != nil {
    return err
}
```

---

### `tool/`

`Tool` interface, `FuncTool` adapter for wrapping plain Go functions, MCP client integration, and tool registry.

**Key types**: `Tool`, `FuncTool`, `Registry`.

**Key functions**: `Register()`, `New()`, `List()`.

---

## Agent

### `agent/`

`BaseAgent`, planner interfaces, executor, and handoff system. Handoffs are auto-generated tools named `transfer_to_{name}`.

**Key interfaces**: `Agent`, `Planner`, `Executor`.

**Key types**: `Persona`, `Option`, `Event`, `EventType`.

**Event types**: `EventText`, `EventToolCall`, `EventToolResult`, `EventDone`, `EventError`.

```go
a := agent.New(
    agent.WithPersona(agent.Persona{Role: "Assistant", Goal: "Help users"}),
    agent.WithLLM(model),
    agent.WithTools(searchTool, calcTool),
)
for evt, err := range a.Stream(ctx, "What is 2+2?") {
    if err != nil {
        return err
    }
    fmt.Print(evt.Text)
}
```

#### `agent/workflow/`

Sequential, parallel, and loop agent wrappers built on top of the core `Agent` interface.

---

## Runtime

### `runtime/`

Agent lifecycle management layer. Hosts a single agent (via `Runner`) or a group of agents (via `Team`). Applies `Plugin` chains, manages `Session` state, and enforces bounded concurrency with `WorkerPool`.

The `runtime/` package is the primary entry point for deploying agents as services.

**Key types**: `Runner`, `Team`, `Plugin`, `PluginChain`, `Session`, `SessionService`, `WorkerPool`, `OrchestrationPattern`.

**Key constructors**: `NewRunner()`, `NewTeam()`, `NewWorkerPool()`, `NewPluginChain()`, `NewInMemorySessionService()`.

**Functional options for Runner**:

| Option | Default | Description |
|--------|---------|-------------|
| `WithWorkerPoolSize(n)` | 10 | Maximum concurrent agent executions |
| `WithPlugins(p...)` | none | Plugin chain applied to every turn |
| `WithSessionService(s)` | in-memory | Session persistence backend |
| `WithRunnerConfig(cfg)` | defaults | Full config struct override |

**Quick start**:

```go
import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/runtime"
    "github.com/lookatitude/beluga-ai/schema"
)

runner := runtime.NewRunner(myAgent,
    runtime.WithWorkerPoolSize(20),
    runtime.WithPlugins(auditPlugin, costPlugin),
)
defer func() {
    if err := runner.Shutdown(context.Background()); err != nil {
        fmt.Println("shutdown error:", err)
    }
}()

for evt, err := range runner.Run(ctx, "session-abc", schema.NewHumanMessage("hello")) {
    if err != nil {
        return err
    }
    fmt.Print(evt.Text)
}
```

**Team composition**:

```go
import "github.com/lookatitude/beluga-ai/runtime"

team := runtime.NewTeam(
    runtime.WithTeamID("research-team"),
    runtime.WithAgents(analyst, summarizer),
    runtime.WithPattern(runtime.PipelinePattern()),
)
// Team implements agent.Agent — wrap it in a Runner like any other agent.
runner := runtime.NewRunner(team)
```

**Orchestration patterns**:

- `PipelinePattern()` — sequential; output of each agent feeds the next.
- `SupervisorPattern(coordinator)` — coordinator agent receives available-agent descriptions and delegates.
- `ScatterGatherPattern(aggregator)` — all agents run in parallel; aggregator synthesizes results.

**Related packages**: `runtime/plugins/`, `cost/`, `audit/`.

---

#### `runtime/plugins/`

Built-in `Plugin` implementations for use with `runtime.NewPluginChain`.

| Plugin constructor | Effect |
|--------------------|--------|
| `NewAuditPlugin(store)` | Logs `agent.turn.start`, `agent.turn.end`, and `agent.turn.error` entries to an `audit.Store`. |
| `NewCostTracking(tracker, budget)` | Records a `cost.Usage` entry via `cost.Tracker` after every completed turn. |
| `NewRateLimit(rpm)` | Rejects turns that exceed `rpm` requests per minute using a token bucket. Returns `core.ErrRateLimit`. |
| `NewRetryAndReflect(...)` | Retries retryable errors up to a configurable limit. |

```go
import (
    "github.com/lookatitude/beluga-ai/audit"
    "github.com/lookatitude/beluga-ai/cost"
    "github.com/lookatitude/beluga-ai/runtime"
    "github.com/lookatitude/beluga-ai/runtime/plugins"
)

auditStore := audit.NewInMemoryStore()
tracker := cost.NewInMemoryTracker()
budget := cost.Budget{MaxCostPerDay: 10.0, Action: cost.BudgetActionReject}

runner := runtime.NewRunner(myAgent,
    runtime.WithPlugins(
        plugins.NewAuditPlugin(auditStore),
        plugins.NewCostTracking(tracker, budget),
        plugins.NewRateLimit(60),
    ),
)
```

---

## Observability and Control

### `cost/`

Token usage tracking and budget enforcement. Defines `Tracker` for recording and querying `Usage` records, and `BudgetChecker` for enforcing hourly token and daily cost limits.

**Key interfaces**: `Tracker`, `BudgetChecker`.

**Key types**: `Usage`, `Filter`, `Summary`, `Budget`, `BudgetDecision`, `BudgetAction`.

**Built-in implementations**: `InMemoryTracker`, `InMemoryBudgetChecker`.

**Registry**: `cost.Register()`, `cost.New()`, `cost.List()`. The `"inmemory"` backend is registered automatically.

```go
import "github.com/lookatitude/beluga-ai/cost"

tracker, err := cost.New("inmemory", cost.Config{MaxEntries: 50000})
if err != nil {
    return err
}

err = tracker.Record(ctx, cost.Usage{
    InputTokens:  512,
    OutputTokens: 128,
    TotalTokens:  640,
    Cost:         0.0048,
    Model:        "gpt-4o",
    Provider:     "openai",
    TenantID:     "tenant-a",
})
if err != nil {
    return err
}

summary, err := tracker.Query(ctx, cost.Filter{
    TenantID: "tenant-a",
    Provider: "openai",
})
if err != nil {
    return err
}
fmt.Printf("total cost: $%.4f over %d calls\n", summary.TotalCost, summary.EntryCount)
```

**Budget enforcement**:

```go
checker := cost.NewInMemoryBudgetChecker(tracker)
decision, err := checker.Check(ctx, cost.Budget{
    MaxTokensPerHour: 100_000,
    MaxCostPerDay:    10.0,
    AlertThreshold:   0.8,
    Action:           cost.BudgetActionReject,
}, cost.Usage{TotalTokens: 700, Cost: 0.005, TenantID: "tenant-a"})
if err != nil {
    return err
}
if !decision.Allowed {
    return fmt.Errorf("budget exceeded: %s (usage ratio %.1f%%)", decision.Reason, decision.UsageRatio*100)
}
```

**Budget actions**: `BudgetActionThrottle`, `BudgetActionReject`, `BudgetActionAlert`.

**Extension**: Implement `Tracker` → call `cost.Register("name", factory)` in `init()`.

**Related packages**: `runtime/plugins/` (CostTracking plugin), `o11y/`.

---

### `audit/`

Structured audit logging. Records `Entry` values describing significant system actions with full traceability (tenant, agent, session, action, duration, error).

**Key interfaces**: `Logger` (write-only), `Store` (Logger + Query).

**Key types**: `Entry`, `Filter`.

**Built-in implementations**: `InMemoryStore`.

**Registry**: `audit.Register()`, `audit.New()`, `audit.List()`. The `"inmemory"` backend is registered automatically.

```go
import "github.com/lookatitude/beluga-ai/audit"

store, err := audit.New("inmemory", audit.Config{})
if err != nil {
    return err
}

err = store.Log(ctx, audit.Entry{
    TenantID:  "tenant-a",
    AgentID:   "planner",
    SessionID: "session-001",
    Action:    "tool.execute",
    Duration:  250 * time.Millisecond,
})
if err != nil {
    return err
}

entries, err := store.Query(ctx, audit.Filter{
    TenantID: "tenant-a",
    Since:    time.Now().Add(-24 * time.Hour),
    Limit:    100,
})
if err != nil {
    return err
}
```

**Security note**: The `Input` and `Output` fields of `Entry` accept `any`. Redact PII, API keys, secrets, and passwords before logging.

**Extension**: Implement `Store` → call `audit.Register("name", factory)` in `init()`.

**Related packages**: `runtime/plugins/` (AuditPlugin).

---

## Deployment

### `deploy/`

Utilities for generating deployment artifacts and exposing health-check endpoints. Targets Go developers deploying agents to Docker and Linux environments.

**Key functions**:

| Function | Description |
|----------|-------------|
| `GenerateDockerfile(cfg)` | Multi-stage Dockerfile: golang builder + distroless runtime image. |
| `GenerateCompose(cfg)` | Docker Compose YAML with one service per `AgentDeployment`. |
| `NewHealthEndpoint()` | HTTP handlers for `/healthz` (liveness) and `/readyz` (readiness). |

**Dockerfile generation**:

```go
import "github.com/lookatitude/beluga-ai/deploy"

dockerfile, err := deploy.GenerateDockerfile(deploy.DockerfileConfig{
    BaseImage:   "gcr.io/distroless/static-debian12",
    GoVersion:   "1.23",
    AgentConfig: "config/planner.yaml",
    Port:        8080,
})
if err != nil {
    return err
}
// Write dockerfile to disk or pass to a Docker client.
```

**Compose generation**:

```go
compose, err := deploy.GenerateCompose(deploy.ComposeConfig{
    Agents: []deploy.AgentDeployment{
        {
            Name:       "planner",
            ConfigPath: "config/planner.yaml",
            Port:       8081,
        },
        {
            Name:       "executor",
            ConfigPath: "config/executor.yaml",
            Port:       8082,
            DependsOn:  []string{"planner"},
            Environment: map[string]string{
                "LOG_LEVEL": "info",
            },
        },
    },
})
if err != nil {
    return err
}
```

**Health endpoints**:

```go
h := deploy.NewHealthEndpoint()
h.AddCheck("database", func(ctx context.Context) error {
    return db.PingContext(ctx)
})
h.AddCheck("llm-provider", func(ctx context.Context) error {
    return model.Health(ctx)
})

mux := http.NewServeMux()
mux.HandleFunc("/healthz", h.Healthz())
mux.HandleFunc("/readyz", h.Readyz())
```

The liveness endpoint (`/healthz`) always returns `200 OK`. The readiness endpoint (`/readyz`) returns `503` when any check fails. Check error details are suppressed in the HTTP response to prevent information disclosure — log them internally.

**Validation**: All string inputs (image names, config paths, environment keys) are validated against safe character sets to prevent injection into generated files.

**Related packages**: `k8s/`, `runtime/`.

---

### `k8s/`

Kubernetes operator types, reconciliation logic, and admission webhooks for deploying agents and teams as Kubernetes custom resources. Intentionally free of Kubernetes library dependencies — all types are plain Go structs.

#### `k8s/operator/`

CRD type definitions and the `Reconciler` interface for computing desired Kubernetes state from Agent and Team custom resources.

**Key CRD types**: `AgentResource`, `TeamResource`, `AgentSpec`, `TeamSpec`, `AgentStatus`, `TeamStatus`.

**Key desired-state types**: `DeploymentSpec`, `ServiceSpec`, `HPASpec`, `ReconcileResult`, `TeamReconcileResult`.

**Key interface**: `Reconciler` with `ReconcileAgent()` and `ReconcileTeam()`.

The `Reconciler` does not call any Kubernetes API. It produces `ReconcileResult` values that callers map to actual Kubernetes objects using whichever client library they choose (controller-runtime, client-go, etc.).

```go
import "github.com/lookatitude/beluga-ai/k8s/operator"

r := operator.NewDefaultReconciler()

agentRes := operator.AgentResource{
    APIVersion: "beluga.ai/v1",
    Kind:       "Agent",
    Meta:       operator.ObjectMeta{Name: "planner", Namespace: "agents"},
    Spec: operator.AgentSpec{
        Persona:       operator.Persona{Role: "Planner"},
        Planner:       "react",
        MaxIterations: 10,
        ModelRef:      "gpt4o-config",
        Replicas:      2,
        Scaling: operator.ScalingConfig{
            Enabled:              true,
            MinReplicas:          1,
            MaxReplicas:          5,
            TargetCPUUtilization: 70,
        },
    },
}

result, err := r.ReconcileAgent(ctx, agentRes)
if err != nil {
    return err
}
// result.Deployment, result.Service, result.HPA are ready to apply
// via your preferred Kubernetes client.
```

**AgentSpec fields of note**:

| Field | Description |
|-------|-------------|
| `ModelRef` | Name of a Secret or ConfigMap key holding the model config. Required. |
| `Planner` | Planner strategy: `"react"`, `"openai-functions"`, `"plan-and-execute"`. |
| `MaxIterations` | Maximum reasoning iterations per invocation. Must be > 0. |
| `Scaling.Enabled` | Activates HPA generation in `ReconcileResult`. |
| `APIKeyRef` | `"<secret-name>/<key>"` reference for the provider API key. |

#### `k8s/webhooks/`

Admission webhook handlers for validating and mutating Agent and Team resources before they are persisted.

**Validation** (`ValidateAgent`, `ValidateTeam`): checks required fields, recognized planner and pattern values, and absence of duplicate member names.

**Mutation** (`MutateAgent`, `MutateTeam`): applies defaults (e.g., `MaxIterations=10`) and standard labels (`beluga.ai/component=agent`).

```go
import "github.com/lookatitude/beluga-ai/k8s/webhooks"

result := webhooks.ValidateAgent(agentResource)
if !result.Allowed {
    return fmt.Errorf("invalid agent spec: %s", result.Reason)
}

agentResource = webhooks.MutateAgent(agentResource)
```

**Related packages**: `deploy/`, `runtime/`.

---

## Memory

### `memory/`

3-tier memory following the MemGPT model: Core (always in context), Recall (recent turns, searchable), Archival (long-term, vector-indexed). Graph memory overlay is optional.

**Key stores**: `memory/stores/` for pluggable backends.

---

## Retrieval

### `rag/`

Retrieval-Augmented Generation pipeline: `embedding/`, `vectorstore/`, `retriever/`, `loader/`, `splitter/`. Hybrid search combines vector similarity with BM25 via RRF fusion.

---

## Orchestration

### `orchestration/`

Higher-level coordination primitives: Chain, Graph (DAG), Router, Parallel execution, Supervisor pattern. For agent-level composition use `runtime/`; for step-level composition use `orchestration/`.

---

### `workflow/`

Durable execution engine for long-running agent tasks. Built-in implementation with Temporal as an optional provider. Handles retries, checkpointing, and resumption after failures.

---

## Protocols

### `protocol/`

MCP (Model Context Protocol) server and client, A2A (Agent-to-Agent) server and client, REST gateway. Exposes agents and tools to external consumers.

---

## Safety

### `guard/`

3-stage safety pipeline: Input guard → Output guard → Tool guard. Implements spotlighting (data delimiters) to separate instructions from user content. Guards are composable.

---

### `resilience/`

Circuit breaker, hedge, retry with backoff, and rate limiter. Applied outside-in as middleware. Integrates with `core.IsRetryable()` to avoid retrying non-recoverable errors.

---

### `hitl/`

Human-in-the-loop approval flow. Pauses agent execution pending external confirmation before tool calls or agent actions. Integrates with durable workflow for persistence across restarts.

---

## Infrastructure

### `cache/`

Exact match cache, semantic cache, and prompt cache optimization. Prompt cache puts static content (system prompts, examples) before dynamic content for maximum provider-side cache hits.

---

### `auth/`

RBAC, ABAC, and capability-based access control. Multi-tenancy scoped via `core.WithTenant()`. Every handler must explicitly grant access; deny is the default.

---

### `eval/`

Evaluation metrics and runner. Measures factuality, relevance, toxicity, and task completion. Pluggable metric backends.

---

### `state/`

Shared agent state with `Watch` for reactive updates. Used by orchestration patterns that need coordination signals between agents.

---

### `prompt/`

Prompt template management and versioning. Supports variable substitution, few-shot example management, and prompt-level A/B testing.

---

### `server/`

HTTP adapter layer for gin, fiber, echo, chi, and gRPC. Wraps `Runner.Run()` behind framework-specific handlers with request parsing and SSE streaming.

---

### `voice/`

Frame-based voice pipeline. `FrameProcessor` interface for STT, LLM, TTS stages. Supports Speech-to-Speech (S2S) and hybrid pipelines. Transport adapters for WebRTC and WebSocket.

---

## Utilities

### `internal/`

Private utilities: `syncutil` (pool helpers, mutex wrappers), `jsonutil` (marshal/unmarshal helpers), `testutil` (mock implementations of all public interfaces).

---

## Package Dependency Order

```
internal/  ←  core/  ←  schema/  ←  config/ o11y/
                ↓
         tool/ llm/ memory/ rag/ guard/ resilience/ cache/
                ↓
         agent/  orchestration/  workflow/
                ↓
         runtime/  protocol/  hitl/  auth/  eval/  state/  prompt/
                ↓
         cost/  audit/  deploy/  k8s/  server/  voice/
```

No package may import a package above it in this hierarchy.
