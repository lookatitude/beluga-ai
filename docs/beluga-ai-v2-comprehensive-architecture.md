# Beluga AI v2 — Comprehensive Framework Architecture

## 1. Competitive Landscape Analysis

### 1.1 Go AI Framework Comparison Matrix

| Capability | Google ADK Go 1.0 | ByteDance Eino | LangChainGo | Beluga AI v2 |
|---|---|---|---|---|
| **Core Abstraction** | Agent interface + Runner | Component interfaces + Graph compose | Chain/Agent/Tool interfaces | Agent + Planner + Runnable |
| **Streaming** | Event-driven via Runner | StreamReader[T] + auto-copy/merge | Callback-based | iter.Seq2[T, error] + backpressure |
| **Agent Types** | LlmAgent, SequentialAgent, ParallelAgent, LoopAgent, CustomAgent | ChatModelAgent (ReAct), Supervisor, Deep Agent, Plan-Execute | ZeroShotReact, Conversational, MRKL | ReAct, Reflexion, Self-Discover, ToT, GoT, LATS, MoA + custom |
| **Multi-Agent** | Hierarchical parent/child + A2A remote | Supervisor, Host multi-agent | Single agent executor | Supervisor, Hierarchical, Scatter-Gather, Router, Blackboard, Handoffs-as-Tools |
| **Orchestration** | Sequential/Parallel/Loop workflow agents | Chain (DAG), Graph (directed), Workflow (field mapping) | Sequential chains, router chains | Chain, Graph (DAG), Durable Workflow, Scatter-Gather, Router, Blackboard |
| **Plugin System** | Runner plugin config (retry-and-reflect, logging) | AOP callbacks (OnStart/End/Error/StreamIn/StreamOut) | Callback handlers | 4-mechanism: Interface + Registry + Hooks + Middleware |
| **Protocols** | A2A native, MCP via mcptoolset | MCP via tools | Basic tool interface | MCP (Streamable HTTP) + A2A (protobuf) + REST/gRPC/SSE |
| **Memory** | Session service (in-memory, Firestore) | Custom via components | Buffer, summary memory | 3-tier (core/recall/archival) + graph + self-editable |
| **Voice** | Live streaming via Gemini Live | Not built-in | Not built-in | Frame-based pipeline (STT→LLM→TTS), S2S, Hybrid |
| **Observability** | OTel via telemetry package | APMPlus + Langfuse callbacks | Callback handlers | OTel GenAI conventions + adapter interface |
| **Durability** | None built-in | None built-in | None built-in | Built-in durable execution engine + Temporal provider |
| **Guard/Safety** | RequireConfirmation on tools (HITL) | Not built-in | Not built-in | 3-stage pipeline (input→output→tool) + Spotlighting |
| **Kubernetes** | Via Vertex AI Agent Engine (managed) | Via CloudWeGo ecosystem | None | CRD-native operator (planned) |
| **Deployment** | CLI, web UI, Cloud Run, Vertex AI | Library import, Docker | Library import | Library, Docker, K8s CRDs, Temporal, standalone |

### 1.2 Key Insights from Competitors

**Google ADK Go 1.0** (shipped early 2026): The Runner is the central orchestrator — it manages agent selection via `findAgentToRun()` based on session history, creates invocation contexts, and yields events. The plugin system is elegant: plugins are injected into the Runner config and intercept every turn. The `RequireConfirmation` pattern for HITL is minimal and effective. Weakness: no durable execution, no voice pipeline, memory is session-scoped only.

**ByteDance Eino** (8.2K+ stars): The strongest architectural insight is the stream processing model. Eino automatically handles stream concatenation (for non-streaming consumers), stream copying (for fan-out to multiple downstream nodes), and stream merging (for convergence). The AOP callback system with five hook points is well-designed. The graph orchestration compiles to a typed `Runnable` with compile-time checks. Weakness: no multi-agent beyond supervisor, no protocols, no durability.

**LangChainGo**: The broadest provider ecosystem (10+ LLM integrations). The `Agent` interface with `Plan()` returning `AgentAction`/`AgentFinish` is clean. The `Executor` pattern separates reasoning from execution. Weakness: no streaming-first design, limited agent types, no orchestration beyond chains.

### 1.3 Where Beluga v2 Wins

Beluga's architecture is the most comprehensive by a wide margin. The specific advantages over every competitor:

1. **7 reasoning strategies** vs ADK's 1 (ReAct) and Eino's 3 (ReAct, Supervisor, Plan-Execute)
2. **5 orchestration patterns** vs ADK's 3 (Sequential/Parallel/Loop) and Eino's 3 (Chain/Graph/Workflow)
3. **Durable execution engine** — no Go competitor has this
4. **Voice pipeline** — no Go competitor has this
5. **3-stage guard pipeline** — ADK only has per-tool confirmation
6. **Handoffs-as-tools** (from OpenAI pattern) — only Beluga in Go
7. **4-mechanism extensibility** (Interface + Registry + Hooks + Middleware) — most systematic

---

## 2. The Beluga Runtime Model

The framework is organized around a single unifying concept: **everything is a stream of typed events flowing through composable processors**. This applies whether you're running a single agent in a Go binary, a team of agents in Docker Compose, a fleet of agents on Kubernetes, or a durable workflow on Temporal.

### 2.1 The Agent as the Atomic Unit

An Agent is the smallest deployable unit. It encapsulates:

```
┌──────────────────────────────────────────────────────────┐
│                        Agent                              │
│                                                           │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌────────────┐ │
│  │ Persona │  │ Planner │  │  Tools  │  │   Memory   │ │
│  │ (RGB)   │  │ (ReAct, │  │ (native │  │ (3-tier +  │ │
│  │         │  │  LATS,  │  │  + MCP) │  │   graph)   │ │
│  │         │  │ custom) │  │         │  │            │ │
│  └─────────┘  └─────────┘  └─────────┘  └────────────┘ │
│                                                           │
│  ┌─────────────────────────────────────────────────────┐ │
│  │              Executor (reasoning loop)               │ │
│  │   Plan → Act → Observe → Replan → (finish/handoff)  │ │
│  └─────────────────────────────────────────────────────┘ │
│                                                           │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐               │
│  │  Hooks   │  │Middleware│  │  Guards  │               │
│  └──────────┘  └──────────┘  └──────────┘               │
│                                                           │
│  Implements: Agent interface (Invoke + Stream + ID + Card)│
│  Exposes via: A2A AgentCard, MCP tools, REST/gRPC/SSE    │
└──────────────────────────────────────────────────────────┘
```

### 2.2 The Runner: Agent Lifecycle Manager

Drawing from ADK Go 1.0's Runner pattern (which proved effective at Google scale), Beluga introduces a **Runner** as the execution host for agents. The Runner is what turns an agent definition into a running process.

```go
// runtime/runner.go

type Runner struct {
    agent          agent.Agent
    sessionService SessionService
    artifactService ArtifactService
    pluginChain    []Plugin
    guardPipeline  guard.Pipeline
    eventBus       EventBus
    metrics        o11y.Meter
    config         RunnerConfig
}

type RunnerConfig struct {
    MaxConcurrentSessions int
    SessionTTL            time.Duration
    StreamingMode         StreamingMode  // None, SSE, WebSocket
    WorkerPoolSize        int
    GracefulShutdownTimeout time.Duration
}

// Run executes a single turn of the agent
func (r *Runner) Run(ctx context.Context, sessionID string, input schema.Message) iter.Seq2[schema.Event, error]

// RunDurable wraps execution in the durable workflow engine
func (r *Runner) RunDurable(ctx context.Context, opts workflow.Options, input schema.Message) (*workflow.Handle, error)

// Serve exposes the agent via protocol gateway (REST/SSE/A2A/MCP)
func (r *Runner) Serve(ctx context.Context, addr string) error
```

The Runner handles:
- Session creation and management
- Plugin execution (before/after each turn)
- Guard pipeline enforcement (input → output → tool)
- Event emission and streaming
- Graceful shutdown and drain
- Health checks and readiness probes

### 2.3 Teams: Multi-Agent Composition

A Team is a group of agents with an orchestration pattern. Teams are themselves Agents (they implement the same interface), enabling recursive composition.

```go
// runtime/team.go

type Team struct {
    agent.BaseAgent
    agents  []agent.Agent
    pattern OrchestrationPattern
    config  TeamConfig
}

type OrchestrationPattern interface {
    Execute(ctx context.Context, agents []agent.Agent, input any) iter.Seq2[schema.Event, error]
}

// Built-in patterns
type SupervisorPattern struct { ... }     // Central LLM delegates
type HandoffPattern struct { ... }        // Peer-to-peer transfers
type ScatterGatherPattern struct { ... }  // Parallel + aggregate
type PipelinePattern struct { ... }       // Sequential chain
type BlackboardPattern struct { ... }     // Shared state + resolver

// Teams compose recursively
researchTeam := runtime.NewTeam(
    runtime.WithAgents(researcher1, researcher2, factChecker),
    runtime.WithPattern(runtime.ScatterGather(summarizerAgent)),
)

fullTeam := runtime.NewTeam(
    runtime.WithAgents(researchTeam, writerAgent, reviewerAgent),
    runtime.WithPattern(runtime.Supervisor(coordinatorLLM)),
)
```

---

## 3. Deployment Architecture: Four Modes

Beluga v2 supports four deployment modes from the same codebase. The framework code never changes — only the hosting wrapper does.

### 3.1 Mode 1: Library (Standalone Go App)

The simplest mode. Import Beluga as a Go library, construct agents in code, run them directly.

```go
package main

import (
    "github.com/lookatitude/beluga-ai/agent"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/runtime"
    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

func main() {
    model, _ := llm.New("openai", llm.ProviderConfig{Model: "gpt-4o"})
    
    myAgent := agent.New(
        agent.WithPersona(agent.Persona{Role: "Assistant"}),
        agent.WithLLM(model),
        agent.WithTools(myTools...),
    )

    runner := runtime.NewRunner(myAgent, runtime.RunnerConfig{
        WorkerPoolSize: 10,
    })

    // Option A: Direct invocation
    for event, err := range runner.Run(ctx, "session-1", userMsg) {
        fmt.Print(event.Text())
    }

    // Option B: Serve via HTTP
    runner.Serve(ctx, ":8080")
}
```

Performance characteristics: Single binary, ~15MB, cold start <100ms, zero external dependencies beyond the LLM provider.

### 3.2 Mode 2: Docker / Docker Compose

For multi-agent deployments without Kubernetes. Each agent (or team) runs in its own container, communicating via A2A or event bus.

```yaml
# docker-compose.yaml
services:
  research-agent:
    image: beluga-agent:latest
    environment:
      BELUGA_AGENT_CONFIG: /config/research-agent.yaml
      BELUGA_LLM_API_KEY: ${OPENAI_API_KEY}
    ports:
      - "8081:8080"
    volumes:
      - ./config:/config
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/healthz"]

  writer-agent:
    image: beluga-agent:latest
    environment:
      BELUGA_AGENT_CONFIG: /config/writer-agent.yaml
    ports:
      - "8082:8080"
    depends_on:
      research-agent:
        condition: service_healthy

  coordinator:
    image: beluga-agent:latest
    environment:
      BELUGA_AGENT_CONFIG: /config/coordinator.yaml
      BELUGA_TEAM_AGENTS: "research-agent:8080,writer-agent:8080"
    ports:
      - "8080:8080"

  redis:
    image: redis:7-alpine

  nats:
    image: nats:2-alpine
    command: ["--jetstream"]
```

The agent configuration YAML:

```yaml
# config/research-agent.yaml
agent:
  id: research-agent
  persona:
    role: "Senior Researcher"
    goal: "Find accurate information from multiple sources"
  planner: react
  maxIterations: 15

llm:
  provider: openai
  model: gpt-4o
  temperature: 0.7

tools:
  - name: web_search
    type: builtin
  - name: calculator
    type: builtin
  - name: company-data
    type: mcp
    url: "http://mcp-server:3000"

memory:
  type: composite
  working: { type: window, size: 20 }
  recall: { type: semantic, store: redis }

guard:
  input: [prompt_injection_detector]
  output: [pii_redactor]
  tool: [capability_check]

server:
  port: 8080
  streaming: sse
  protocols: [rest, a2a]

observability:
  tracing: { exporter: otlp, endpoint: "jaeger:4317" }
  metrics: { exporter: prometheus, port: 9090 }
```

### 3.3 Mode 3: Kubernetes (CRD + Operator)

For enterprise deployments. Agents are declared as Kubernetes Custom Resources and managed by the Beluga Operator.

```yaml
apiVersion: beluga.ai/v1alpha1
kind: ModelConfig
metadata:
  name: gpt-4o-config
  namespace: agents
spec:
  provider: openai
  model: gpt-4o
  temperature: 0.7
  apiKeyFrom:
    secretKeyRef: { name: openai-secret, key: api-key }
  rateLimits:
    rpm: 500
    tpm: 200000
    maxConcurrent: 50
  fallback:
    modelRef: groq-llama-config
---
apiVersion: beluga.ai/v1alpha1
kind: ToolServer
metadata:
  name: web-search
spec:
  type: builtin
  name: web_search
---
apiVersion: beluga.ai/v1alpha1
kind: ToolServer
metadata:
  name: company-mcp
spec:
  type: mcp
  url: "http://mcp-company-data.tools.svc:3000"
  transport: streamable-http
---
apiVersion: beluga.ai/v1alpha1
kind: GuardPolicy
metadata:
  name: standard-guard
spec:
  input:
    - type: prompt_injection
      provider: built-in
    - type: spotlighting
      delimiter: "<<<DATA>>>"
  output:
    - type: pii_redactor
      patterns: [email, phone, ssn]
    - type: content_filter
      threshold: 0.8
  tool:
    - type: capability_check
---
apiVersion: beluga.ai/v1alpha1
kind: Agent
metadata:
  name: research-agent
  namespace: agents
  labels:
    team: research-team
spec:
  persona:
    role: "Senior Researcher"
    goal: "Find and synthesize information from multiple sources"
    backstory: "Expert at finding reliable sources and cross-referencing facts"
  planner: react
  maxIterations: 15
  modelRef: gpt-4o-config
  toolRefs: [web-search, company-mcp]
  memoryRef: redis-composite
  guardRef: standard-guard
  
  handoffs:
    - targetRef: writer-agent
      description: "Transfer when research is complete and writing is needed"
    - targetRef: fact-checker
      description: "Transfer when claims need verification"
  
  resources:
    requests: { cpu: "250m", memory: "256Mi" }
    limits: { cpu: "1", memory: "1Gi" }
  
  scaling:
    minReplicas: 2
    maxReplicas: 20
    metrics:
      - type: custom
        name: beluga_agent_queue_depth
        targetValue: 10
  
  costPolicy:
    maxTokensPerHour: 500000
    maxCostPerDay: "$25.00"
    alertAt: 80%
    action: throttle
  
  probes:
    liveness: { path: /healthz, period: 10s }
    readiness: { path: /readyz, period: 5s }
  
  observability:
    tracing: true
    metrics: true
    auditLog: true
---
apiVersion: beluga.ai/v1alpha1
kind: Team
metadata:
  name: research-team
spec:
  pattern: supervisor
  coordinatorModelRef: gpt-4o-config
  agentRefs:
    - research-agent
    - writer-agent
    - fact-checker
  durable: true
  durableConfig:
    retryPolicy: { maxAttempts: 3, backoffFactor: 2.0 }
    executionTimeout: 30m
  expose:
    protocols: [rest, a2a]
    port: 8080
```

The Beluga Operator reconciliation loop:

```
Agent CR changed
    │
    ▼
Validate spec (webhook)
    │
    ▼
Resolve references (ModelConfig, ToolServer, GuardPolicy, MemoryStore)
    │
    ▼
Build agent runtime config
    │
    ▼
Create/Update Deployment (pods running beluga-runtime)
    │
    ▼
Create/Update Service + Ingress
    │
    ▼
Create/Update HPA (scaling config)
    │
    ▼
Create/Update NetworkPolicy (sandbox config)
    │
    ▼
Create/Update ServiceMonitor (observability)
    │
    ▼
Register A2A AgentCard at /.well-known/agent.json
    │
    ▼
Update status (ready, endpoints, version)
```

### 3.4 Mode 4: Temporal (Durable Workflows)

For long-running, crash-resistant agent workflows. Every LLM call and tool execution becomes a Temporal Activity with automatic retry and state persistence.

```go
// workflow/temporal/agent_workflow.go

// The Workflow is the deterministic orchestration layer.
// It calls Activities for all non-deterministic work (LLM, tools, APIs).

func AgentWorkflow(ctx workflow.Context, input AgentWorkflowInput) (*AgentWorkflowOutput, error) {
    state := NewPlannerState(input)
    
    for iteration := 0; iteration < input.MaxIterations; iteration++ {
        // Activity: call LLM to plan (non-deterministic, retryable)
        var actions []agent.Action
        err := workflow.ExecuteActivity(ctx, PlanActivity, state).Get(ctx, &actions)
        if err != nil { return nil, err }
        
        // Event log persisted automatically by Temporal ✓
        
        if hasFinishAction(actions) {
            return extractResult(actions), nil
        }
        
        for _, action := range actions {
            switch action.Type {
            case agent.ActionTypeTool:
                // Activity: execute tool (non-deterministic, retryable)
                var result tool.ToolResult
                err := workflow.ExecuteActivity(ctx, ToolActivity, action.ToolCall).Get(ctx, &result)
                state.Observations = append(state.Observations, agent.Observation{
                    Action: action, Result: &result, Error: err,
                })
                
            case agent.ActionTypeHandoff:
                // Child workflow: delegate to another agent
                child := workflow.ExecuteChildWorkflow(ctx, AgentWorkflow, childInput)
                var childResult AgentWorkflowOutput
                child.Get(ctx, &childResult)
                
            case agent.ActionTypeRespond:
                // Signal: emit partial response to caller
                workflow.SignalExternalWorkflow(ctx, input.CallerWorkflowID, "", 
                    "partial_response", action.Message)
            }
        }
        
        state.Iteration = iteration + 1
    }
    return nil, fmt.Errorf("max iterations exceeded")
}

// Activities run the non-deterministic work
func PlanActivity(ctx context.Context, state agent.PlannerState) ([]agent.Action, error) {
    planner := agent.NewPlanner(state.PlannerType, state.PlannerConfig)
    return planner.Plan(ctx, state)
}

func ToolActivity(ctx context.Context, call schema.ToolCall) (*tool.ToolResult, error) {
    t, ok := toolRegistry.Get(call.Name)
    if !ok { return nil, fmt.Errorf("unknown tool: %s", call.Name) }
    return t.Execute(ctx, call.Arguments)
}
```

Usage from application code:

```go
// Simple: no durability
result, _ := agent.Invoke(ctx, "Research competitor pricing")

// Durable: same agent, wrapped in Temporal workflow
handle, _ := runner.RunDurable(ctx, workflow.Options{
    ID: "research-" + uuid.New().String(),
    RetryPolicy: workflow.RetryPolicy{MaxAttempts: 3},
    ExecutionTimeout: 30 * time.Minute,
}, userMsg)

// Can wait for result
result, _ := handle.Result()

// Or receive streaming events
for event := range handle.Events() {
    fmt.Print(event.Text())
}

// Survives crashes, rate limits, human approval delays
```

---

## 4. Performance Architecture

### 4.1 Zero-Allocation Hot Path

The critical path (LLM token → event → consumer) must avoid heap allocations:

```go
// core/event_pool.go

var eventPool = sync.Pool{
    New: func() any { return &Event[schema.StreamChunk]{} },
}

func AcquireEvent() *Event[schema.StreamChunk] {
    return eventPool.Get().(*Event[schema.StreamChunk])
}

func ReleaseEvent(e *Event[schema.StreamChunk]) {
    e.Reset()
    eventPool.Put(e)
}
```

### 4.2 Connection Pooling

Each provider maintains a persistent HTTP/2 connection pool:

```go
// llm/providers/openai/transport.go

type transport struct {
    client *http.Client
    pool   *x509.CertPool
}

func newTransport(cfg ProviderConfig) *transport {
    return &transport{
        client: &http.Client{
            Transport: &http.Transport{
                MaxIdleConns:        100,
                MaxIdleConnsPerHost: 100,
                IdleConnTimeout:    90 * time.Second,
                ForceAttemptHTTP2:  true,
            },
            Timeout: cfg.Timeout,
        },
    }
}
```

### 4.3 Bounded Worker Pools

Agent pods use bounded concurrency, not unbounded goroutines:

```go
// runtime/worker_pool.go

type WorkerPool struct {
    sem     chan struct{}
    wg      sync.WaitGroup
    metrics o11y.Meter
}

func NewWorkerPool(size int) *WorkerPool {
    return &WorkerPool{sem: make(chan struct{}, size)}
}

func (p *WorkerPool) Submit(ctx context.Context, fn func(context.Context)) error {
    select {
    case p.sem <- struct{}{}:
        p.wg.Add(1)
        go func() {
            defer func() { <-p.sem; p.wg.Done() }()
            fn(ctx)
        }()
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

### 4.4 Parallel Tool Execution (DAG)

Inspired by LLMCompiler (1.8× latency improvement):

```go
// agent/tool_dag.go

type ToolDAGExecutor struct {
    maxConcurrency int
    depDetection   bool // analyze input/output dependencies
}

func (e *ToolDAGExecutor) Execute(ctx context.Context, calls []schema.ToolCall, registry *tool.Registry) []tool.ToolResult {
    if e.depDetection {
        // Build dependency graph from tool call arguments
        dag := buildDependencyDAG(calls)
        // Execute independent tools in parallel, dependent tools sequentially
        return executeDAG(ctx, dag, registry, e.maxConcurrency)
    }
    // Simple parallel execution
    return executeParallel(ctx, calls, registry, e.maxConcurrency)
}
```

### 4.5 Prompt Cache Optimization

Automatic ordering for maximum cache hit rates:

```go
// prompt/builder.go

type Builder struct {
    systemPrompt   string          // slot 1: static (highest cache value)
    toolDefs       []tool.Tool     // slot 2: semi-static
    staticContext  []schema.Message // slot 3: semi-static
    cacheBreak     bool            // explicit cache boundary
    dynamicContext []schema.Message // slot 4: dynamic
    userInput      schema.Message  // slot 5: always changes
}
```

### 4.6 Performance Targets

| Metric | Target | How |
|---|---|---|
| Cold start | <100ms | Single static binary, lazy provider init |
| Token-to-first-byte | <50ms overhead | Zero-alloc streaming, pre-warmed connections |
| Tool execution parallelism | 5 concurrent | DAG executor with bounded pool |
| Memory per agent pod | <256MB base | sync.Pool, no global state |
| Agent scaling | 0→10 pods in <30s | HPA on queue depth metric |
| Prompt cache hit rate | >80% | Deterministic message ordering |

---

## 5. Extensibility Architecture

### 5.1 The Four Mechanisms

Every extensible package follows the same structure. Learning one teaches all:

```
<package>/
├── <interface>.go      # 1. Extension contract (Go interface, 1-3 methods)
├── registry.go         # 2. Register() / New() / List()
├── hooks.go            # 3. Lifecycle callbacks
├── middleware.go        # 4. func(T) T decorators
└── providers/           # Built-in implementations
```

### 5.2 Complete Extension Point Map

| Package | Interface | What You Extend | Hooks |
|---|---|---|---|
| `llm/` | `ChatModel` | LLM inference | BeforeGenerate, AfterGenerate, OnStream, OnError |
| `tool/` | `Tool` | Executable capability | BeforeExecute, AfterExecute, OnError |
| `agent/` | `Planner` | Reasoning strategy | BeforePlan, AfterPlan, OnIteration, OnFinish |
| `agent/` | `Agent` (BaseAgent) | Custom agent logic | OnStart, OnTool, OnHandoff, OnError, OnEnd |
| `memory/` | `Memory` | Conversation memory | BeforeSave, AfterLoad |
| `memory/stores/` | `MessageStore` | Storage backend | — |
| `rag/embedding/` | `Embedder` | Text → vector | BeforeEmbed, AfterEmbed |
| `rag/vectorstore/` | `VectorStore` | Vector storage | BeforeAdd, AfterSearch |
| `rag/retriever/` | `Retriever` | Document retrieval | BeforeRetrieve, AfterRetrieve, OnRerank |
| `voice/` | `FrameProcessor` | Audio processing | OnFrame, OnSpeechStart, OnSpeechEnd |
| `guard/` | `Guard` | Safety validation | — |
| `workflow/` | `DurableExecutor` | Durable engine | BeforeActivity, AfterActivity, OnSignal |
| `orchestration/` | `OrchestrationPattern` | Team coordination | BeforeStep, AfterStep |
| `runtime/` | `Plugin` | Cross-cutting concern | BeforeTurn, AfterTurn, OnError |
| `server/` | `ServerAdapter` | HTTP framework | — |
| `cache/` | `Cache` | Caching backend | — |
| `auth/` | `Policy` | Authorization | — |
| `state/` | `Store` | Shared state | — |

### 5.3 The Plugin System (ADK-Inspired)

Drawing from ADK Go 1.0's plugin architecture, Beluga introduces Runner-level plugins for cross-cutting concerns:

```go
// runtime/plugin.go

type Plugin interface {
    Name() string
    // BeforeTurn runs before each agent invocation
    BeforeTurn(ctx context.Context, session *Session, input schema.Message) (schema.Message, error)
    // AfterTurn runs after each agent invocation
    AfterTurn(ctx context.Context, session *Session, events []schema.Event) ([]schema.Event, error)
    // OnError runs when an error occurs
    OnError(ctx context.Context, err error) error
}

// Built-in plugins
type RetryAndReflectPlugin struct { maxRetries int }   // Auto-retry with LLM reflection
type AuditPlugin struct { store audit.Store }           // Audit logging
type CostTrackingPlugin struct { budget *cost.Budget }  // Token/cost tracking
type RateLimitPlugin struct { limits ProviderLimits }   // Provider-aware rate limiting
type GuardPlugin struct { pipeline guard.Pipeline }     // Safety pipeline

// Usage
runner := runtime.NewRunner(myAgent, runtime.RunnerConfig{
    Plugins: []runtime.Plugin{
        runtime.NewRetryAndReflect(3),
        runtime.NewAuditPlugin(auditStore),
        runtime.NewCostTracking(cost.Budget{MaxPerDay: 50.00}),
    },
})
```

---

## 6. Security Architecture

### 6.1 Defense-in-Depth Model

```
User Input
    │
    ▼
┌─────────────────────────────────────┐
│  INPUT GUARDS                        │
│  • Prompt injection detection        │
│  • Spotlighting (data delimiters)    │
│  • Input validation                  │
│  • Rate limiting                     │
└─────────────────┬───────────────────┘
                  │
                  ▼
┌─────────────────────────────────────┐
│  AGENT EXECUTION                     │
│  (capability-scoped)                 │
│                                      │
│  ┌───────────────────────────────┐  │
│  │  TOOL GUARDS (per-tool)       │  │
│  │  • Capability check            │  │
│  │  • Input schema validation     │  │
│  │  • HITL approval (risk-based)  │  │
│  │  • Network policy enforcement  │  │
│  └───────────────────────────────┘  │
└─────────────────┬───────────────────┘
                  │
                  ▼
┌─────────────────────────────────────┐
│  OUTPUT GUARDS                       │
│  • Content moderation                │
│  • PII redaction                     │
│  • Hallucination detection           │
│  • Schema enforcement                │
└─────────────────────────────────────┘
```

### 6.2 Capability-Based Agent Sandboxing

```go
agent := agent.New(
    agent.WithCapabilities(
        auth.Grant(auth.CapToolExec, "web_search", "calculator"),
        auth.Grant(auth.CapMemoryRead),
        auth.Deny(auth.CapNetworkAccess),  // default deny
        auth.Deny(auth.CapCodeExec),
    ),
    agent.WithSandbox(sandbox.Config{
        NetworkPolicy: sandbox.DenyAll,
        AllowedHosts:  []string{"api.openai.com"},
        MaxMemory:     256 * 1024 * 1024,
        Timeout:       30 * time.Second,
    }),
)
```

### 6.3 Multi-Tenancy Isolation

```go
ctx = core.WithTenant(ctx, "customer-123")
// All downstream: separate memory namespace, rate limit bucket,
// cost tracking, audit log, model config overrides
```

### 6.4 Secret Management

- Library mode: environment variables or config file
- Docker mode: Docker secrets or .env files
- Kubernetes mode: K8s Secrets with `secretKeyRef` in CRDs
- All modes: never log API keys, auto-redact from traces

---

## 7. Observability Architecture

### 7.1 OTel GenAI Semantic Conventions

Every boundary emits standardized spans:

```go
ctx, span := o11y.StartSpan(ctx, "agent.invoke", o11y.Attrs{
    "gen_ai.agent.name":      a.ID(),
    "gen_ai.request.model":   a.llm.ModelID(),
    "gen_ai.operation.name":  "agent_invoke",
    "gen_ai.system":          "beluga",
})
defer span.End()
```

### 7.2 Six Metric Categories

| Category | Metrics | Where |
|---|---|---|
| Latency | Per-step, end-to-end, TTFB | Every boundary |
| Token usage | Input, output, cached, total | LLM middleware |
| Cost | Per-request, cumulative, per-tenant | Cost tracking plugin |
| Error rates | By type (rate_limit, timeout, tool_failed) | Error middleware |
| Tool success | Execution rate, latency, failure rate | Tool hooks |
| Quality scores | Faithfulness, relevance, hallucination | Eval framework |

### 7.3 Built-in Endpoints

Every agent process exposes:
- `GET /healthz` — liveness (is the process alive?)
- `GET /readyz` — readiness (is the agent ready to serve?)
- `GET /metrics` — Prometheus metrics
- `GET /.well-known/agent.json` — A2A AgentCard

---

## 8. Complete Package Layout

```
beluga-ai/
├── go.mod
│
├── core/                    # Foundation — zero external deps
│   ├── stream.go            # iter.Seq2[T, error] primitives
│   ├── runnable.go          # Runnable interface (Invoke, Stream)
│   ├── batch.go             # BatchInvoke with concurrency control
│   ├── context.go           # Session context, cancel propagation
│   ├── tenant.go            # Multi-tenancy primitives
│   ├── lifecycle.go         # Lifecycle interface, App struct
│   ├── errors.go            # Typed errors, IsRetryable()
│   └── option.go            # Functional options
│
├── schema/                  # Shared types — no business logic
│   ├── message.go           # Message, HumanMsg, AIMsg, SystemMsg, ToolMsg
│   ├── content.go           # ContentPart: Text, Image, Audio, Video, File
│   ├── tool.go              # ToolCall, ToolResult, ToolDefinition
│   ├── document.go          # Document with metadata
│   ├── event.go             # AgentEvent, StreamEvent, LifecycleEvent
│   ├── frame.go             # Frame (audio/text/control) for voice
│   └── session.go           # Session, Turn, ConversationState
│
├── config/                  # Configuration loading
│   ├── config.go            # Load[T], Validate, env + file + struct tags
│   ├── provider.go          # ProviderConfig base type
│   └── watch.go             # Hot-reload (fsnotify, K8s ConfigMap)
│
├── o11y/                    # Observability
│   ├── tracer.go            # OTel GenAI tracer
│   ├── meter.go             # OTel meter + Prometheus
│   ├── logger.go            # Structured logging (slog)
│   ├── health.go            # /healthz, /readyz, /metrics
│   ├── exporter.go          # LLM-specific trace exporter interface
│   └── adapters/            # Langfuse, Arize Phoenix
│
├── llm/                     # LLM abstraction
│   ├── llm.go               # ChatModel interface
│   ├── options.go           # GenerateOptions
│   ├── registry.go          # Register(), New(), List()
│   ├── hooks.go             # LLM lifecycle hooks
│   ├── middleware.go         # Retry, rate-limit, cache, fallback, guardrail
│   ├── router.go            # Multi-model routing (cost, latency, capability)
│   ├── structured.go        # StructuredOutput[T] with JSON Schema
│   ├── context.go           # Context window management (6 strategies)
│   ├── tokenizer.go         # Token counting
│   └── providers/           # OpenAI, Anthropic, Google, Ollama, Groq, etc.
│
├── tool/                    # Tool system
│   ├── tool.go              # Tool interface
│   ├── functool.go          # Wrap Go functions as Tools
│   ├── registry.go          # ToolRegistry
│   ├── hooks.go             # Tool hooks
│   ├── mcp.go               # MCP client (Streamable HTTP)
│   ├── mcp_registry.go      # MCP server discovery
│   ├── middleware.go         # Auth, rate-limit, timeout
│   └── builtin/             # Calculator, HTTP, Shell, Code execution
│
├── memory/                  # 3-tier + graph memory
│   ├── memory.go            # Memory interface
│   ├── registry.go          # Register(), New(), List()
│   ├── hooks.go / middleware.go
│   ├── buffer.go            # Full-history
│   ├── window.go            # Sliding window
│   ├── summary.go           # LLM-summarized
│   ├── entity.go            # Entity tracking
│   ├── semantic.go          # Vector-backed
│   ├── graph.go             # Knowledge graph
│   ├── composite.go         # Composite (working + recall + archival + graph)
│   └── stores/              # inmemory, redis, postgres, sqlite, neo4j
│
├── rag/                     # RAG pipeline
│   ├── embedding/           # Embedder interface + providers
│   ├── vectorstore/         # VectorStore interface + providers
│   ├── retriever/           # Retriever + hybrid/CRAG/HyDE/GraphRAG
│   ├── loader/              # DocumentLoader (PDF, HTML, web, code, etc.)
│   └── splitter/            # TextSplitter (recursive, markdown)
│
├── agent/                   # Agent runtime
│   ├── agent.go             # Agent interface
│   ├── base.go              # BaseAgent (embeddable)
│   ├── persona.go           # Role/Goal/Backstory
│   ├── executor.go          # Reasoning loop (delegates to Planner)
│   ├── planner.go           # Planner interface + PlannerState
│   ├── registry.go          # RegisterPlanner(), NewPlanner()
│   ├── hooks.go             # Complete lifecycle hooks
│   ├── middleware.go         # Agent middleware
│   ├── bus.go               # EventBus (in-memory, NATS, Redis)
│   ├── handoff.go           # Handoffs-as-tools (OpenAI pattern)
│   ├── card.go              # A2A AgentCard
│   ├── react.go             # ReAct planner
│   ├── reflexion.go         # Reflexion planner
│   ├── selfdiscover.go      # Self-Discover planner
│   ├── tot.go               # Tree-of-Thought planner
│   ├── got.go               # Graph-of-Thought planner
│   ├── lats.go              # LATS planner
│   ├── moa.go               # Mixture-of-Agents planner
│   └── workflow/            # SequentialAgent, ParallelAgent, LoopAgent
│
├── runtime/                 # NEW — Agent lifecycle management
│   ├── runner.go            # Runner: host for a single agent
│   ├── team.go              # Team: multi-agent composition
│   ├── plugin.go            # Plugin interface
│   ├── plugins/             # Built-in plugins
│   │   ├── retry_reflect.go # Auto-retry with LLM reflection
│   │   ├── audit.go         # Audit logging
│   │   ├── cost.go          # Cost tracking + budgets
│   │   └── ratelimit.go     # Provider-aware rate limiting
│   ├── session.go           # SessionService interface
│   ├── session_memory.go    # In-memory sessions
│   ├── session_redis.go     # Redis-backed sessions
│   └── worker_pool.go       # Bounded concurrency
│
├── orchestration/           # Orchestration patterns
│   ├── pattern.go           # OrchestrationPattern interface
│   ├── supervisor.go        # Central LLM delegates
│   ├── handoff.go           # Peer-to-peer transfers
│   ├── scatter_gather.go    # Parallel + aggregate
│   ├── pipeline.go          # Sequential chain
│   ├── blackboard.go        # Shared state + resolver
│   ├── router.go            # Conditional dispatch
│   └── hooks.go             # BeforeStep, AfterStep
│
├── voice/                   # Voice pipeline
│   ├── pipeline.go          # Frame-based cascading
│   ├── hybrid.go            # S2S + cascade switching
│   ├── session.go           # Voice session management
│   ├── vad.go               # VAD interface
│   ├── stt/ tts/ s2s/       # Provider interfaces + implementations
│   └── transport/           # WebSocket, LiveKit, Daily
│
├── workflow/                # Durable execution
│   ├── executor.go          # DurableExecutor interface
│   ├── activity.go          # LLM/Tool/Human activity wrappers
│   ├── state.go             # Checkpoint, metadata, history
│   ├── signal.go            # HITL signals
│   ├── patterns/            # Pre-built workflow patterns
│   └── providers/           # Temporal, in-memory, NATS
│
├── protocol/                # External protocols
│   ├── mcp/                 # MCP server + client
│   ├── a2a/                 # A2A server + client
│   └── rest/                # REST/SSE API
│
├── guard/                   # Safety pipeline
│   ├── guard.go             # Guard interface
│   ├── pipeline.go          # Input → Output → Tool pipeline
│   ├── injection.go         # Prompt injection detection
│   ├── spotlight.go         # Spotlighting
│   ├── pii.go               # PII redaction
│   ├── content.go           # Content moderation
│   └── adapters/            # NeMo, Guardrails AI, LLM Guard, Lakera
│
├── auth/                    # Authorization
│   ├── auth.go              # Capability, Policy interfaces
│   ├── rbac.go / abac.go    # RBAC, ABAC
│   └── opa.go               # Open Policy Agent
│
├── resilience/              # Production resilience
│   ├── circuitbreaker.go
│   ├── hedge.go
│   ├── retry.go
│   └── ratelimit.go
│
├── cache/                   # Caching (exact + semantic + prompt)
├── hitl/                    # Human-in-the-loop
├── eval/                    # Evaluation framework
├── state/                   # Shared agent state
├── prompt/                  # Prompt management + cache optimization
├── cost/                    # Cost tracking + budgets
├── audit/                   # Audit logging
│
├── server/                  # HTTP framework adapters
│   ├── handler.go           # Standard http.Handler
│   ├── sse.go               # SSE streaming
│   └── adapters/            # Gin, Fiber, Echo, Chi, gRPC, Connect-Go
│
├── k8s/                     # Kubernetes operator (optional, never imported by core)
│   ├── operator/            # Controllers
│   ├── crds/                # CRD YAML definitions
│   ├── webhooks/            # Admission webhooks
│   └── helm/                # Helm chart
│
├── deploy/                  # Deployment utilities
│   ├── container.go         # Dockerfile generation
│   ├── compose.go           # Docker Compose generation
│   └── healthz.go           # Standard health endpoints
│
└── internal/                # Shared utilities
    ├── syncutil/            # sync primitives, worker pools
    ├── jsonutil/            # JSON schema generation
    └── testutil/            # Mocks for every interface
```

---

## 9. Design Invariants

These are the rules that must never be violated:

1. **The library never imports `k8s/`.** Kubernetes is an optional overlay.
2. **Every interface has ≤4 methods.** Larger surfaces are composed.
3. **Every provider uses `init()` + `Register()`.** No config files to edit.
4. **Streaming is the primary path.** `Invoke()` is always "stream, collect, return last."
5. **`context.Context` carries everything.** Cancellation, tracing, tenant, auth.
6. **Middleware is always `func(T) T`.** Stack any number, apply outside-in.
7. **Hooks fire at specific lifecycle points.** They complement (not replace) middleware.
8. **Events flow down through layers.** Each layer depends only on layers below it.
9. **The Runner is the deployment boundary.** One Runner = one deployable unit.
10. **Teams are Agents.** Recursive composition, infinite depth.
