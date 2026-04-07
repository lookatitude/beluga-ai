# Beluga AI v2 Migration Plan

**Generated**: 2026-04-07
**Source**: `docs/beluga-ai-v2-comprehensive-architecture.md` Section 8 (Complete Package Layout)
**Baseline**: Current codebase on branch `fix/release-workflow-trigger` (commit d04250f1)

---

## 1. Gap Analysis Summary

| Package | Status | Key Gaps |
|---|---|---|
| `core/` | EXISTS_PARTIAL | Missing `event_pool.go` (sync.Pool for zero-alloc streaming) |
| `schema/` | EXISTS_PARTIAL | Missing `frame.go` (Frame type for voice; currently in `voice/frame.go`) |
| `config/` | EXISTS_COMPLETE | All files present with tests |
| `o11y/` | EXISTS_COMPLETE | All files present with tests; adapters exist as providers |
| `llm/` | EXISTS_COMPLETE | All files present with tests and 20+ providers |
| `tool/` | EXISTS_COMPLETE | All files present with tests |
| `memory/` | EXISTS_COMPLETE | All files present with 9 store providers |
| `rag/` | EXISTS_COMPLETE | All subpackages present (embedding, vectorstore, retriever, loader, splitter) |
| `agent/` | EXISTS_PARTIAL | Missing `tool_dag.go` (parallel tool execution DAG). Has all 7 planners, workflow subpkg |
| `orchestration/` | EXISTS_PARTIAL | Missing `handoff.go` (peer-to-peer), `pipeline.go` (sequential chain). Missing `OrchestrationPattern` interface. Has supervisor, scatter, blackboard, router, chain, graph |
| `voice/` | EXISTS_COMPLETE | All files present including s2s, stt, tts, transport, vad subdirs |
| `workflow/` | EXISTS_PARTIAL | Missing `signal.go` (HITL signals), `patterns/` subdir. Has executor, activity, state, store, providers |
| `protocol/` | EXISTS_COMPLETE | Has a2a, mcp, rest, openai_agents subdirs |
| `guard/` | EXISTS_COMPLETE | All files present with 5 guard providers |
| `auth/` | EXISTS_PARTIAL | Missing `opa.go` (Open Policy Agent). Has rbac, abac, composite |
| `resilience/` | EXISTS_COMPLETE | All files present |
| `cache/` | EXISTS_COMPLETE | Has cache, semantic, providers/inmemory |
| `hitl/` | EXISTS_COMPLETE | All files present |
| `eval/` | EXISTS_COMPLETE | Has metrics, providers (braintrust, deepeval, ragas) |
| `state/` | EXISTS_COMPLETE | Has providers/inmemory |
| `prompt/` | EXISTS_COMPLETE | Has builder.go with cache-optimal ordering, manager.go, template.go |
| `server/` | EXISTS_COMPLETE | Has 7 adapters (gin, fiber, echo, chi, grpc, connect, huma) |
| `internal/` | EXISTS_COMPLETE | Has syncutil, jsonutil, testutil, hookutil, httpclient, httputil, openaicompat |
| `runtime/` | **NEW** | Entire package must be created (runner, team, plugin, session, worker_pool) |
| `cost/` | **NEW** | Top-level cost tracking package (currently empty `optimize/cost/`) |
| `audit/` | **NEW** | Top-level audit logging package |
| `k8s/` | **NEW** | Kubernetes operator, CRDs, webhooks, helm chart |
| `deploy/` | **NEW** | Deployment utilities (container, compose, healthz) |

### Existing Package Health

All existing packages compile (`go build ./...` succeeds) and have test files alongside source. The registry pattern (Register/New/List) is present in: `llm/`, `agent/` (RegisterPlanner/NewPlanner/ListPlanners), `tool/`, `memory/`, `guard/`, `rag/embedding/`, `rag/vectorstore/`, `rag/retriever/`. Hooks and middleware are present in: `llm/`, `tool/`, `memory/`, `agent/`, `orchestration/`, `voice/`, `workflow/`, `hitl/`, `state/`, `server/`.

---

## 2. Implementation Batches

### Batch 1 -- Foundation (No dependencies on other new code)

These packages depend only on existing `core/`, `schema/`, and stdlib. They can be built in parallel.

#### Task 1.1: `core/event_pool.go` -- Zero-Allocation Event Pool

- **Classification**: ENHANCEMENT (add file to existing package)
- **Files to create**: `core/event_pool.go`, `core/event_pool_test.go`
- **Interface definitions**:

```go
// core/event_pool.go
package core

import "sync"

// Event wraps a typed payload for zero-allocation streaming on the hot path.
type Event[T any] struct {
    Type    string
    Payload T
    SeqNum  int64
}

// Reset clears all fields so the Event can be returned to the pool.
func (e *Event[T]) Reset() { /* zero all fields */ }

// EventPool provides sync.Pool-backed allocation for streaming events.
// Use AcquireEvent/ReleaseEvent for the hot path (LLM token -> consumer).
var eventPool = sync.Pool{
    New: func() any { return &Event[any]{} },
}

// AcquireEvent retrieves a pre-allocated Event from the pool.
func AcquireEvent[T any]() *Event[T]

// ReleaseEvent returns an Event to the pool after Reset().
func ReleaseEvent[T any](e *Event[T])
```

- **Acceptance criteria**:
  - `go build ./core/...` passes
  - `go test ./core/...` passes
  - `go test -bench=BenchmarkEventPool ./core/...` shows zero allocations per op in steady state
  - No external dependencies added to core/
- **Dependencies**: None

#### Task 1.2: `cost/` -- Cost Tracking and Budgets

- **Classification**: NEW
- **Files to create**: `cost/cost.go`, `cost/budget.go`, `cost/tracker.go`, `cost/registry.go`, `cost/cost_test.go`, `cost/doc.go`
- **Interface definitions**:

```go
// cost/cost.go
package cost

import (
    "context"
    "time"
)

// Usage represents token and cost usage for a single operation.
type Usage struct {
    InputTokens   int
    OutputTokens  int
    CachedTokens  int
    TotalTokens   int
    Cost          float64 // in USD
    Model         string
    Provider      string
    Timestamp     time.Time
}

// Tracker records and queries cost/token usage.
type Tracker interface {
    // Record stores a usage entry scoped to the context (tenant, session).
    Record(ctx context.Context, usage Usage) error
    // Query returns aggregated usage matching the filter.
    Query(ctx context.Context, filter Filter) (*Summary, error)
}

// Filter selects usage records for aggregation.
type Filter struct {
    TenantID  string
    Model     string
    Provider  string
    Since     time.Time
    Until     time.Time
}

// Summary is an aggregated view of usage.
type Summary struct {
    TotalInputTokens  int64
    TotalOutputTokens int64
    TotalCost         float64
    EntryCount        int64
}

// Budget defines spending limits with enforcement actions.
type Budget struct {
    MaxTokensPerHour int64
    MaxCostPerDay    float64
    AlertThreshold   float64 // 0.0-1.0, e.g. 0.8 = alert at 80%
    Action           BudgetAction
}

// BudgetAction determines what happens when a budget threshold is crossed.
type BudgetAction string

const (
    BudgetActionThrottle BudgetAction = "throttle"
    BudgetActionReject   BudgetAction = "reject"
    BudgetActionAlert    BudgetAction = "alert"
)

// BudgetChecker evaluates whether a new operation would exceed the budget.
type BudgetChecker interface {
    Check(ctx context.Context, budget Budget, estimated Usage) (BudgetDecision, error)
}

// BudgetDecision is the result of a budget check.
type BudgetDecision struct {
    Allowed    bool
    Reason     string
    UsageRatio float64 // current usage / limit
}
```

- **Acceptance criteria**:
  - `go build ./cost/...` passes
  - `go test ./cost/...` passes with table-driven tests for Tracker and BudgetChecker
  - In-memory Tracker implementation included
  - No circular dependencies (cost/ depends only on core/, stdlib)
  - Registry pattern: `Register("inmemory", ...)` / `New("inmemory", ...)` / `List()`
- **Dependencies**: None

#### Task 1.3: `audit/` -- Audit Logging

- **Classification**: NEW
- **Files to create**: `audit/audit.go`, `audit/store.go`, `audit/registry.go`, `audit/audit_test.go`, `audit/doc.go`
- **Interface definitions**:

```go
// audit/audit.go
package audit

import (
    "context"
    "time"
)

// Entry represents a single audit log record.
type Entry struct {
    ID        string
    Timestamp time.Time
    TenantID  string
    AgentID   string
    SessionID string
    Action    string    // e.g. "agent.invoke", "tool.execute", "handoff"
    Input     any       // redacted input summary
    Output    any       // redacted output summary
    Metadata  map[string]string
    Error     string    // empty if no error
    Duration  time.Duration
}

// Logger writes audit entries. Implementations may be synchronous or async.
type Logger interface {
    // Log records an audit entry.
    Log(ctx context.Context, entry Entry) error
}

// Store persists and queries audit entries.
type Store interface {
    Logger
    // Query returns entries matching the filter.
    Query(ctx context.Context, filter Filter) ([]Entry, error)
}

// Filter selects audit entries.
type Filter struct {
    TenantID  string
    AgentID   string
    SessionID string
    Action    string
    Since     time.Time
    Until     time.Time
    Limit     int
}
```

- **Acceptance criteria**:
  - `go build ./audit/...` passes
  - `go test ./audit/...` passes
  - In-memory Store implementation included
  - Registry pattern: `Register()` / `New()` / `List()`
  - No circular dependencies (audit/ depends only on core/, stdlib)
- **Dependencies**: None

#### Task 1.4: `runtime/worker_pool.go` -- Bounded Concurrency

- **Classification**: NEW (first file in new runtime/ package)
- **Files to create**: `runtime/worker_pool.go`, `runtime/worker_pool_test.go`, `runtime/doc.go`
- **Interface definitions**:

```go
// runtime/worker_pool.go
package runtime

import (
    "context"
    "sync"
)

// WorkerPool provides bounded concurrency for agent execution.
// It limits the number of concurrent goroutines via a semaphore.
type WorkerPool struct {
    sem     chan struct{}
    wg      sync.WaitGroup
}

// NewWorkerPool creates a pool with the given concurrency limit.
func NewWorkerPool(size int) *WorkerPool

// Submit schedules fn for execution. Blocks if the pool is full.
// Returns ctx.Err() if the context is cancelled while waiting.
func (p *WorkerPool) Submit(ctx context.Context, fn func(context.Context)) error

// Wait blocks until all submitted work completes.
func (p *WorkerPool) Wait()

// Drain stops accepting new work and waits for in-flight work to finish.
func (p *WorkerPool) Drain(ctx context.Context) error
```

- **Acceptance criteria**:
  - `go build ./runtime/...` passes
  - `go test ./runtime/...` passes
  - Tests verify: bounded concurrency (no more than N concurrent), context cancellation while queued, Drain behavior
  - No external dependencies
- **Dependencies**: None

#### Task 1.5: `runtime/session.go` -- Session Service Interface

- **Classification**: NEW
- **Files to create**: `runtime/session.go`, `runtime/session_memory.go`, `runtime/session_test.go`
- **Interface definitions**:

```go
// runtime/session.go
package runtime

import (
    "context"
    "time"

    "github.com/lookatitude/beluga-ai/schema"
)

// Session holds the state for a single agent conversation.
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

// SessionService manages session lifecycle.
type SessionService interface {
    // Create initializes a new session.
    Create(ctx context.Context, agentID string) (*Session, error)
    // Get retrieves an existing session.
    Get(ctx context.Context, sessionID string) (*Session, error)
    // Update persists session state changes.
    Update(ctx context.Context, session *Session) error
    // Delete removes a session.
    Delete(ctx context.Context, sessionID string) error
}
```

- **Acceptance criteria**:
  - `go build ./runtime/...` passes
  - `go test ./runtime/...` passes
  - In-memory SessionService implementation included (`session_memory.go`)
  - Tests cover: CRUD operations, concurrent access safety, expiration
  - SessionService interface has exactly 4 methods
- **Dependencies**: Task 1.4 (shares runtime/ package)

---

### Batch 2 -- Runtime Core (Depends on Batch 1)

#### Task 2.1: `runtime/plugin.go` -- Plugin Interface

- **Classification**: NEW
- **Files to create**: `runtime/plugin.go`, `runtime/plugin_test.go`
- **Interface definitions**:

```go
// runtime/plugin.go
package runtime

import (
    "context"

    "github.com/lookatitude/beluga-ai/schema"
)

// Plugin intercepts agent execution at the Runner level.
// All methods are called for every turn. Implementations should be lightweight.
type Plugin interface {
    // Name returns a unique identifier for this plugin.
    Name() string
    // BeforeTurn runs before each agent invocation. It may modify the input.
    BeforeTurn(ctx context.Context, session *Session, input schema.Message) (schema.Message, error)
    // AfterTurn runs after each agent invocation. It may modify events.
    AfterTurn(ctx context.Context, session *Session, events []schema.Event) ([]schema.Event, error)
    // OnError runs when an error occurs during agent execution.
    OnError(ctx context.Context, err error) error
}
```

- **Acceptance criteria**:
  - `go build ./runtime/...` passes
  - `go test ./runtime/...` passes
  - Plugin chain executes in order (first registered runs first)
  - Tests verify: BeforeTurn modifies input, AfterTurn modifies events, OnError propagation
  - Plugin interface has exactly 4 methods
- **Dependencies**: Task 1.4, Task 1.5

#### Task 2.2: `runtime/runner.go` -- Agent Lifecycle Runner

- **Classification**: NEW
- **Files to create**: `runtime/runner.go`, `runtime/runner_test.go`, `runtime/config.go`
- **Interface definitions**:

```go
// runtime/runner.go
package runtime

import (
    "context"
    "iter"
    "time"

    "github.com/lookatitude/beluga-ai/agent"
    "github.com/lookatitude/beluga-ai/guard"
    "github.com/lookatitude/beluga-ai/schema"
)

// StreamingMode determines how the Runner streams events to callers.
type StreamingMode int

const (
    StreamingNone      StreamingMode = iota
    StreamingSSE
    StreamingWebSocket
)

// RunnerConfig configures a Runner instance.
type RunnerConfig struct {
    MaxConcurrentSessions   int
    SessionTTL              time.Duration
    StreamingMode           StreamingMode
    WorkerPoolSize          int
    GracefulShutdownTimeout time.Duration
}

// Runner is the lifecycle manager for a single agent.
// It handles session management, plugin execution, guard enforcement,
// event streaming, and graceful shutdown.
type Runner struct { /* unexported fields */ }

// RunnerOption configures a Runner.
type RunnerOption func(*Runner)

// NewRunner creates a Runner for the given agent.
func NewRunner(a agent.Agent, opts ...RunnerOption) *Runner

// Run executes a single turn, returning a stream of events.
func (r *Runner) Run(ctx context.Context, sessionID string, input schema.Message) iter.Seq2[schema.Event, error]

// Serve starts an HTTP server exposing the agent via configured protocols.
func (r *Runner) Serve(ctx context.Context, addr string) error

// Shutdown drains in-flight requests and stops the Runner.
func (r *Runner) Shutdown(ctx context.Context) error

// Functional options
func WithPlugins(plugins ...Plugin) RunnerOption
func WithGuardPipeline(p guard.Pipeline) RunnerOption
func WithSessionService(s SessionService) RunnerOption
func WithRunnerConfig(cfg RunnerConfig) RunnerOption
```

- **Acceptance criteria**:
  - `go build ./runtime/...` passes
  - `go test ./runtime/...` passes
  - Run() returns `iter.Seq2[schema.Event, error]` (streaming-first)
  - Plugin chain fires in order on every Run() call
  - Guard pipeline enforced (input guards before agent, output guards after)
  - Session created/loaded automatically per sessionID
  - Worker pool limits concurrency
  - Context cancellation stops execution
  - Graceful shutdown drains in-flight sessions
- **Dependencies**: Task 1.4, Task 1.5, Task 2.1

#### Task 2.3: `runtime/team.go` -- Multi-Agent Composition

- **Classification**: NEW
- **Files to create**: `runtime/team.go`, `runtime/team_test.go`, `runtime/pattern.go`
- **Interface definitions**:

```go
// runtime/pattern.go
package runtime

import (
    "context"
    "iter"

    "github.com/lookatitude/beluga-ai/agent"
    "github.com/lookatitude/beluga-ai/schema"
)

// OrchestrationPattern defines how a Team coordinates its agents.
type OrchestrationPattern interface {
    // Execute runs the pattern with the given agents and input.
    Execute(ctx context.Context, agents []agent.Agent, input any) iter.Seq2[schema.Event, error]
}

// runtime/team.go

// Team is a group of agents with an orchestration pattern.
// Teams implement the Agent interface, enabling recursive composition.
type Team struct {
    agent.BaseAgent
    agents  []agent.Agent
    pattern OrchestrationPattern
}

// TeamOption configures a Team.
type TeamOption func(*Team)

// NewTeam creates a Team with the given options.
func NewTeam(opts ...TeamOption) *Team

func WithAgents(agents ...agent.Agent) TeamOption
func WithPattern(p OrchestrationPattern) TeamOption
func WithTeamID(id string) TeamOption

// Built-in pattern constructors
func SupervisorPattern(coordinatorModel interface{}) OrchestrationPattern
func HandoffPattern() OrchestrationPattern
func ScatterGatherPattern(aggregator agent.Agent) OrchestrationPattern
func PipelinePattern() OrchestrationPattern
func BlackboardPattern(resolver func(context.Context, map[string]any) (any, error)) OrchestrationPattern
```

- **Acceptance criteria**:
  - `go build ./runtime/...` passes
  - `go test ./runtime/...` passes
  - Team implements `agent.Agent` interface (compile-time check: `var _ agent.Agent = (*Team)(nil)`)
  - Teams compose recursively (Team of Teams)
  - All 5 built-in patterns have constructors
  - Pattern.Execute returns `iter.Seq2[schema.Event, error]`
  - Tests verify: supervisor delegation, scatter-gather parallelism, pipeline sequencing
- **Dependencies**: Task 2.2

#### Task 2.4: `runtime/plugins/` -- Built-in Plugins

- **Classification**: NEW
- **Files to create**: `runtime/plugins/retry_reflect.go`, `runtime/plugins/audit.go`, `runtime/plugins/cost.go`, `runtime/plugins/ratelimit.go`, `runtime/plugins/plugins_test.go`
- **Interface definitions**:

```go
// runtime/plugins/retry_reflect.go
package plugins

import "github.com/lookatitude/beluga-ai/runtime"

// NewRetryAndReflect creates a plugin that retries failed agent turns
// with LLM-generated reflection on the error.
func NewRetryAndReflect(maxRetries int) runtime.Plugin

// runtime/plugins/audit.go
import "github.com/lookatitude/beluga-ai/audit"

// NewAuditPlugin creates a plugin that logs every turn to the audit store.
func NewAuditPlugin(store audit.Store) runtime.Plugin

// runtime/plugins/cost.go
import "github.com/lookatitude/beluga-ai/cost"

// NewCostTracking creates a plugin that tracks token usage and enforces budgets.
func NewCostTracking(budget cost.Budget) runtime.Plugin

// runtime/plugins/ratelimit.go

// NewRateLimit creates a plugin that enforces provider-aware rate limits.
func NewRateLimit(opts ...RateLimitOption) runtime.Plugin
```

- **Acceptance criteria**:
  - `go build ./runtime/...` passes
  - `go test ./runtime/plugins/...` passes
  - Each plugin implements `runtime.Plugin` interface (compile-time check)
  - Cost plugin calls `cost.Tracker.Record()` after every turn
  - Audit plugin calls `audit.Logger.Log()` for every turn
  - RetryAndReflect retries up to maxRetries on retryable errors
- **Dependencies**: Task 1.2 (cost/), Task 1.3 (audit/), Task 2.1 (Plugin interface)

---

### Batch 3 -- Enhanced Capabilities (Depends on Batch 2)

#### Task 3.1: `agent/tool_dag.go` -- Parallel Tool Execution DAG

- **Classification**: ENHANCEMENT (add file to existing package)
- **Files to create**: `agent/tool_dag.go`, `agent/tool_dag_test.go`
- **Interface definitions**:

```go
// agent/tool_dag.go
package agent

import (
    "context"

    "github.com/lookatitude/beluga-ai/schema"
    "github.com/lookatitude/beluga-ai/tool"
)

// ToolDAGExecutor executes tool calls with dependency-aware parallelism.
// Independent tools run concurrently; dependent tools run sequentially.
type ToolDAGExecutor struct { /* unexported fields */ }

// ToolDAGOption configures a ToolDAGExecutor.
type ToolDAGOption func(*ToolDAGExecutor)

// NewToolDAGExecutor creates a DAG executor.
func NewToolDAGExecutor(opts ...ToolDAGOption) *ToolDAGExecutor

// WithMaxConcurrency limits the number of concurrent tool executions.
func WithMaxConcurrency(n int) ToolDAGOption

// WithDependencyDetection enables automatic input/output dependency analysis.
func WithDependencyDetection(enabled bool) ToolDAGOption

// Execute runs tool calls respecting dependency ordering and concurrency limits.
func (e *ToolDAGExecutor) Execute(ctx context.Context, calls []schema.ToolCall, registry *tool.Registry) []tool.ToolResult
```

- **Acceptance criteria**:
  - `go build ./agent/...` passes
  - `go test ./agent/...` passes
  - Independent tool calls execute in parallel (verified by timing test)
  - Dependent tool calls execute sequentially
  - MaxConcurrency is respected (verified by concurrency counter)
  - Context cancellation stops pending tool calls
  - Benchmark: `BenchmarkToolDAGExecutor` included
- **Dependencies**: Batch 1 complete (for worker pool pattern reference)

#### Task 3.2: `orchestration/handoff.go` and `orchestration/pipeline.go` -- Missing Patterns

- **Classification**: ENHANCEMENT (add files to existing package)
- **Files to create**: `orchestration/handoff.go`, `orchestration/handoff_test.go`, `orchestration/pipeline.go`, `orchestration/pipeline_test.go`, `orchestration/pattern.go`
- **Interface definitions**:

```go
// orchestration/pattern.go
package orchestration

import (
    "context"
    "iter"

    "github.com/lookatitude/beluga-ai/agent"
    "github.com/lookatitude/beluga-ai/schema"
)

// OrchestrationPattern is the shared interface for all orchestration strategies.
type OrchestrationPattern interface {
    Execute(ctx context.Context, agents []agent.Agent, input any) iter.Seq2[schema.Event, error]
}

// orchestration/handoff.go

// Handoff implements peer-to-peer agent transfers using handoff-as-tools.
// When an agent calls transfer_to_{name}, control passes to the target agent.
type Handoff struct { /* unexported fields */ }

func NewHandoff(agents ...agent.Agent) *Handoff
func (h *Handoff) Execute(ctx context.Context, agents []agent.Agent, input any) iter.Seq2[schema.Event, error]

// orchestration/pipeline.go

// Pipeline executes agents sequentially, passing each agent's output as
// the next agent's input. Implements the chain-of-responsibility pattern.
type Pipeline struct { /* unexported fields */ }

func NewPipeline(agents ...agent.Agent) *Pipeline
func (p *Pipeline) Execute(ctx context.Context, agents []agent.Agent, input any) iter.Seq2[schema.Event, error]
```

- **Acceptance criteria**:
  - `go build ./orchestration/...` passes
  - `go test ./orchestration/...` passes
  - Handoff transfers control when `transfer_to_{name}` tool is called
  - Pipeline passes output of agent N as input to agent N+1
  - Both implement `OrchestrationPattern` interface (compile-time check)
  - Streaming: both return `iter.Seq2[schema.Event, error]`
  - Context cancellation respected in both patterns
- **Dependencies**: None (existing package, can start in Batch 2 but logically Batch 3)

#### Task 3.3: `workflow/signal.go` -- HITL Signals

- **Classification**: ENHANCEMENT (add file to existing package)
- **Files to create**: `workflow/signal.go`, `workflow/signal_test.go`
- **Interface definitions**:

```go
// workflow/signal.go
package workflow

import (
    "context"
    "time"
)

// Signal represents a human-in-the-loop signal sent to a running workflow.
type Signal struct {
    Name      string
    Payload   any
    SentAt    time.Time
    SenderID  string
}

// SignalChannel allows workflows to wait for and receive signals.
type SignalChannel interface {
    // Send delivers a signal to a workflow.
    Send(ctx context.Context, workflowID string, signal Signal) error
    // Receive blocks until a signal with the given name arrives or ctx is cancelled.
    Receive(ctx context.Context, workflowID string, signalName string) (*Signal, error)
}
```

- **Acceptance criteria**:
  - `go build ./workflow/...` passes
  - `go test ./workflow/...` passes
  - In-memory SignalChannel implementation included
  - Tests: send/receive, timeout via context, multiple signals
  - SignalChannel interface has exactly 2 methods
- **Dependencies**: None (existing package)

#### Task 3.4: `deploy/` -- Deployment Utilities

- **Classification**: NEW
- **Files to create**: `deploy/container.go`, `deploy/compose.go`, `deploy/healthz.go`, `deploy/deploy_test.go`, `deploy/doc.go`
- **Interface definitions**:

```go
// deploy/container.go
package deploy

// DockerfileConfig describes how to generate a Dockerfile for an agent.
type DockerfileConfig struct {
    BaseImage   string
    GoVersion   string
    AgentConfig string // path to agent YAML
    Port        int
}

// GenerateDockerfile returns Dockerfile contents for the given config.
func GenerateDockerfile(cfg DockerfileConfig) (string, error)

// deploy/compose.go

// ComposeConfig describes a multi-agent Docker Compose deployment.
type ComposeConfig struct {
    Agents []AgentDeployment
}

// AgentDeployment describes a single agent in a Compose file.
type AgentDeployment struct {
    Name        string
    ConfigPath  string
    Port        int
    DependsOn   []string
    Environment map[string]string
}

// GenerateCompose returns docker-compose.yaml contents.
func GenerateCompose(cfg ComposeConfig) (string, error)

// deploy/healthz.go

// HealthEndpoint provides standard /healthz and /readyz HTTP handlers.
type HealthEndpoint struct { /* unexported fields */ }

func NewHealthEndpoint() *HealthEndpoint
func (h *HealthEndpoint) Healthz() http.HandlerFunc
func (h *HealthEndpoint) Readyz() http.HandlerFunc
func (h *HealthEndpoint) AddCheck(name string, check func(context.Context) error)
```

- **Acceptance criteria**:
  - `go build ./deploy/...` passes
  - `go test ./deploy/...` passes
  - GenerateDockerfile produces valid Dockerfile syntax
  - GenerateCompose produces valid YAML
  - Health endpoints return 200 when healthy, 503 when unhealthy
  - No dependency on k8s/ or runtime/
- **Dependencies**: None

#### Task 3.5: `schema/frame.go` -- Frame Type for Voice Pipeline

- **Classification**: ENHANCEMENT (potentially move/alias from voice/frame.go)
- **Files to create or modify**: `schema/frame.go`, `schema/frame_test.go`
- **Design note**: The v2 architecture specifies Frame in `schema/`. Currently `voice/frame.go` defines Frame. The implementation should define the core Frame type in `schema/` and have `voice/` use it, OR create a type alias in `schema/` that re-exports from `voice/`. The preferred approach is to move the type definition to `schema/` and update `voice/` imports.
- **Acceptance criteria**:
  - `go build ./...` passes (no broken imports)
  - `go test ./schema/...` passes
  - `schema.Frame` type exists and is importable
  - `voice/` package uses `schema.Frame` or is updated accordingly
  - No circular dependencies introduced
- **Dependencies**: None

#### Task 3.6: `auth/opa.go` -- Open Policy Agent Integration

- **Classification**: ENHANCEMENT (add file to existing package)
- **Files to create**: `auth/opa.go`, `auth/opa_test.go`
- **Interface definitions**:

```go
// auth/opa.go
package auth

import "context"

// OPAPolicy evaluates authorization decisions against an Open Policy Agent server.
type OPAPolicy struct { /* unexported fields */ }

// OPAOption configures an OPA policy evaluator.
type OPAOption func(*OPAPolicy)

// NewOPAPolicy creates an OPA-backed policy evaluator.
func NewOPAPolicy(endpoint string, opts ...OPAOption) *OPAPolicy

// Evaluate checks whether the given request is authorized.
func (p *OPAPolicy) Evaluate(ctx context.Context, req AuthRequest) (*AuthDecision, error)
```

- **Acceptance criteria**:
  - `go build ./auth/...` passes
  - `go test ./auth/...` passes
  - Implements existing `Policy` interface (compile-time check)
  - Tests use mock HTTP server for OPA endpoint
- **Dependencies**: None

---

### Batch 4 -- Kubernetes (Optional, depends on Batch 2)

This batch is entirely optional. `k8s/` is never imported by any library package.

#### Task 4.1: `k8s/crds/` -- CRD YAML Definitions

- **Classification**: NEW
- **Files to create**:
  - `k8s/crds/agent.yaml`
  - `k8s/crds/team.yaml`
  - `k8s/crds/modelconfig.yaml`
  - `k8s/crds/toolserver.yaml`
  - `k8s/crds/guardpolicy.yaml`
  - `k8s/crds/memorystore.yaml`
- **Acceptance criteria**:
  - All CRDs have `apiVersion: beluga.ai/v1alpha1`
  - CRDs include OpenAPI v3 schema validation
  - `kubectl apply --dry-run=client -f k8s/crds/` succeeds (with a K8s context)
  - CRD fields match the spec in architecture doc Section 3.3
- **Dependencies**: None

#### Task 4.2: `k8s/operator/` -- Controller Implementation

- **Classification**: NEW
- **Files to create**: `k8s/operator/controller.go`, `k8s/operator/reconciler.go`, `k8s/operator/types.go`, `k8s/operator/operator_test.go`
- **Acceptance criteria**:
  - `go build ./k8s/...` passes
  - `go test ./k8s/...` passes
  - Reconciler handles Agent CR create/update/delete
  - Creates Deployment, Service, HPA from Agent spec
  - Uses controller-runtime library (kubebuilder pattern)
- **Dependencies**: Task 4.1

#### Task 4.3: `k8s/webhooks/` -- Admission Webhooks

- **Classification**: NEW
- **Files to create**: `k8s/webhooks/validate.go`, `k8s/webhooks/mutate.go`, `k8s/webhooks/webhooks_test.go`
- **Acceptance criteria**:
  - Validates Agent spec fields (modelRef exists, toolRefs valid, etc.)
  - Mutates defaults (add standard labels, set default resources)
  - Tests use envtest or mock webhook server
- **Dependencies**: Task 4.1, Task 4.2

#### Task 4.4: `k8s/helm/` -- Helm Chart

- **Classification**: NEW
- **Files to create**: `k8s/helm/Chart.yaml`, `k8s/helm/values.yaml`, `k8s/helm/templates/`
- **Acceptance criteria**:
  - `helm lint k8s/helm/` passes
  - `helm template beluga k8s/helm/` produces valid K8s manifests
  - Includes: operator deployment, CRDs, RBAC, ServiceAccount
- **Dependencies**: Task 4.1, Task 4.2, Task 4.3

---

### Batch 5 -- Documentation and Website (Depends on all above)

#### Task 5.1: Update `docs/packages.md` and `docs/architecture.md`

- **Classification**: ENHANCEMENT
- **Files to modify**: `docs/packages.md`, `docs/architecture.md`
- **Acceptance criteria**:
  - All new packages (runtime/, cost/, audit/, deploy/) documented
  - All new interfaces documented with usage examples
  - Dependency graph updated
- **Dependencies**: Batches 1-3 complete

#### Task 5.2: Package-level Documentation

- **Classification**: ENHANCEMENT
- **Files to create/update**: `runtime/doc.go`, `cost/doc.go`, `audit/doc.go`, `deploy/doc.go`
- **Acceptance criteria**:
  - Every new package has a `doc.go` with package overview
  - `go doc ./runtime`, `go doc ./cost`, etc. produce meaningful output
- **Dependencies**: Batches 1-3 complete

#### Task 5.3: Website Updates

- **Classification**: ENHANCEMENT
- **Acceptance criteria**:
  - Docs website includes pages for runtime, cost, audit, deploy packages
  - Architecture diagram updated to show Runner/Team/Plugin relationships
- **Dependencies**: Task 5.1, Task 5.2

---

## 3. Dependency Graph

```
Batch 1 (parallel):
  Task 1.1 (core/event_pool.go)      ──┐
  Task 1.2 (cost/)                    ──┤
  Task 1.3 (audit/)                   ──┤──→ Batch 2
  Task 1.4 (runtime/worker_pool.go)   ──┤
  Task 1.5 (runtime/session.go)       ──┘
                                         │
Batch 2 (sequential within):             │
  Task 2.1 (runtime/plugin.go)      ←───┘──┐
  Task 2.2 (runtime/runner.go)      ←──────┤──→ Batch 3
  Task 2.3 (runtime/team.go)        ←──────┤
  Task 2.4 (runtime/plugins/)       ←──────┘
                                         │
Batch 3 (parallel):                      │
  Task 3.1 (agent/tool_dag.go)      ←───┘
  Task 3.2 (orchestration/handoff+pipeline)
  Task 3.3 (workflow/signal.go)
  Task 3.4 (deploy/)
  Task 3.5 (schema/frame.go)
  Task 3.6 (auth/opa.go)

Batch 4 (optional, parallel after Batch 2):
  Task 4.1 → Task 4.2 → Task 4.3 → Task 4.4

Batch 5 (after all):
  Task 5.1 → Task 5.2 → Task 5.3
```

## 4. Circular Dependency Risks

| Risk | Packages | Mitigation |
|---|---|---|
| runtime/ ↔ agent/ | Runner imports agent.Agent; Team embeds agent.BaseAgent | One-way dependency: runtime/ → agent/. Agent must NEVER import runtime/. |
| runtime/plugins/ → cost/ → runtime/ | Cost plugin uses cost.Tracker; cost tracker might need runtime session | cost/ must not import runtime/. Pass session info via context (core.WithTenant). |
| runtime/plugins/ → audit/ → runtime/ | Same pattern as cost | audit/ must not import runtime/. Audit entry gets agentID/sessionID from context. |
| orchestration/ ↔ runtime/ | Both define OrchestrationPattern | Define interface in runtime/. orchestration/ provides implementations that satisfy it. Or define in a shared location. Preferred: runtime/ defines the interface, orchestration/ implements it. |
| schema/ ↔ voice/ | Moving Frame to schema/ requires voice/ to import schema/ (already does) | Safe. voice/ already imports schema/. No reverse dependency. |

## 5. Open Questions for Architect Decision

1. **OrchestrationPattern interface location**: Should it live in `runtime/` (as the v2 doc implies) or `orchestration/` (where implementations live)? Recommendation: `orchestration/` defines the interface, `runtime/team.go` imports it. This avoids orchestration/ depending on runtime/.

2. **optimize/ package**: The current codebase has an `optimize/` directory with `cost/` and `examples/` subdirs (both empty). The v2 spec defines a top-level `cost/` package. Recommendation: Create top-level `cost/` as specified and remove empty `optimize/`.

3. **schema.Frame relocation**: Moving the Frame type from `voice/frame.go` to `schema/frame.go` will break existing imports. Recommendation: Define canonical type in `schema/`, add type alias in `voice/` for backward compatibility during transition.
