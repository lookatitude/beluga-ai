# How Beluga Compares

This page exists because "pick a framework" is a real decision.
We want you to make it well — even if the answer is "not Beluga."
Every claim below is cited against codebase paths or competitor source URLs.

---

## Quick comparison table

| Framework | Language | Reasoning strategies | Durable execution | Voice pipeline | Guard pipeline | Streaming model |
|---|---|---|---|---|---|---|
| **Beluga AI v2** | Go | 7 (ReAct, Reflexion, Self-Discover, ToT, GoT, LATS, MoA) | Built-in (`workflow.DurableExecutor`) | Full STT/TTS/S2S + transport | 3-stage OWASP-mapped pipeline | `iter.Seq2[Event, error]` |
| LangGraph | Python/JS | Custom graph nodes (roll-your-own) | LangGraph Platform (SaaS) | None | NeMo/Lakera integrations (not typed) | Callbacks / async iterators |
| Google ADK | Python/Go | 1 (SequentialAgent + tool loop) | None (Cloud Run restart) | None | None | Various |
| ByteDance Eino | Go | 3 (ReAct, Plan-Execute-Replan, Supervisor) | None | None | None | Callback/channel hybrid |
| OpenAI Agents SDK | Python/JS | 1 (implicit tool-calling loop) | None (Agentspan.ai third-party) | Python-only via Realtime API | Input guardrails only | Async iterators |
| LiveKit Agents | Python/Node.js | None (voice-only) | None | Best-in-class transport | None | Callbacks |
| Semantic Kernel | .NET/Python/Java | 3 built-in planners | None | None | Azure AI Content Safety | Various |

Sources: `agent/react.go`, `agent/reflection.go`, `agent/selfdiscover.go`, `agent/tot.go`, `agent/got.go`, `agent/lats.go`, `agent/moa.go` confirm 7 strategies. `workflow/workflow.go:52-65` confirms `DurableExecutor`. `voice/s2s/s2s.go` confirms S2S pipeline. `guard/agentic/pipeline.go:124-170` confirms OWASP mapping.

---

## Beluga's differentiators

### 1. The only Go-native framework with a complete production stack

Every other Go-ecosystem option (ADK-Go, Eino, LangChainGo, Genkit-Go) is either a thin wrapper, an LLM-call library, or a port of a Python framework. Beluga ships everything a production agent needs — LLM routing, tool sandboxing, 3-tier memory, hybrid RAG, voice pipeline, guards, durable workflows, HITL, RBAC/ABAC, cost tracking, audit, multi-tenant isolation — with zero Python or JavaScript in the hot path.

```go
import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/agent"
    "github.com/lookatitude/beluga-ai/llm"
    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

func main() {
    model, err := llm.New("openai", llm.Config{Model: "gpt-4o"})
    if err != nil {
        panic(err)
    }
    a := agent.New(
        agent.WithPersona(agent.Persona{Role: "assistant"}),
        agent.WithModel(model),
    )
    result, err := a.Invoke(context.Background(), "What is the capital of France?")
    if err != nil {
        panic(err)
    }
    fmt.Println(result)
}
```

Evidence: `docs/architecture/18-package-dependency-map.md` — `core/` and `schema/` depend only on stdlib + OpenTelemetry. `docs/architecture/17-deployment-modes.md` — 4 deployment modes from the same agent codebase.

Honest qualification: ADK-Go v1.0 (released 2026) also targets Go natively with OTel, HITL, and YAML-driven agents. Its breadth is narrower — roughly Beluga Layers 1–3 plus a thin runner. ADK-Go is Google-Cloud-first; Beluga is provider-agnostic.

---

### 2. Seven swappable reasoning strategies on one interface

No other Go framework exposes more than 3 reasoning strategies. Beluga ships: ReAct, Reflexion, Self-Discover, Tree-of-Thought, Graph-of-Thought, LATS, Mixture-of-Agents. Swapping is a one-line change because every strategy implements the same `Planner` interface (2 methods, `agent/planner.go:14-21`).

```go
// agent/planner.go:14-21
type Planner interface {
    Plan(ctx context.Context, state PlannerState) ([]Action, error)
    Replan(ctx context.Context, state PlannerState) ([]Action, error)
}

// Swap the reasoning strategy without changing the agent:
a := agent.New(
    agent.WithPlanner(agent.NewLATSPlanner(agent.WithMaxDepth(5))),
    // or: agent.NewToTPlanner(...), agent.NewMoAPlanner(...)
)
```

Evidence: `agent/react.go`, `agent/reflection.go`, `agent/selfdiscover.go`, `agent/tot.go`, `agent/got.go`, `agent/lats.go`, `agent/moa.go` — all present in the codebase. Full strategy cost table: `docs/architecture/06-reasoning-strategies.md`.

Competitor comparison: ADK-Go — 1 documented strategy (source: `github.com/google/adk-go`). Eino — 3 strategies (source: CloudWeGo docs). OpenAI Agents SDK — 1 implicit strategy (source: `openai.github.io/openai-agents-python`).

---

### 3. Built-in durable execution — no Go agentic framework ships this natively

Durable workflows with crash recovery, signal/query, activity retry, and HITL pause are first-class features, not add-ons. The `DurableExecutor` interface (`workflow/workflow.go:52-65`) ships with providers for in-memory (dev), NATS, Kafka, Temporal, Dapr, and Inngest. Agents do not change code when the executor backend changes.

```go
import "github.com/lookatitude/beluga-ai/workflow"

// workflow/workflow.go:52-65 — the full interface:
// type DurableExecutor interface {
//     Execute(ctx context.Context, fn WorkflowFunc, opts WorkflowOptions) (WorkflowHandle, error)
//     Signal(ctx context.Context, workflowID string, signal Signal) error
//     Query(ctx context.Context, workflowID string, queryType string) (any, error)
//     Cancel(ctx context.Context, workflowID string) error
// }

exec, err := workflow.NewExecutor(workflow.Config{Backend: "temporal", HostPort: "localhost:7233"})
if err != nil {
    panic(err)
}
handle, err := exec.Execute(ctx, myAgentWorkflow, workflow.WorkflowOptions{ID: "order-42"})
if err != nil {
    panic(err)
}
result, err := handle.Result(ctx)
if err != nil {
    panic(err)
}
```

Evidence: `workflow/workflow.go:52-65`, `workflow/retry.go`, `workflow/activity.go`, `workflow/tracing.go`.

Competitor comparison: ADK-Go — no built-in durable execution as of v1.0. Eino — interrupt/resume via checkpointing, but not event-log-replay durability. LangGraph — persistence plugin (LangGraph Platform), Python-only, SaaS-first. OpenAI Agents SDK — no durability; Agentspan.ai is a third-party paid service.

---

### 4. Frame-based voice pipeline — no Go competitor has this

Beluga's `voice/` layer implements a full real-time voice pipeline with VAD, cascaded (STT + LLM + TTS) and speech-to-speech (S2S) modes. Providers: OpenAI Realtime, Gemini Live (S2S); Deepgram, AssemblyAI, Whisper (STT); ElevenLabs, Cartesia, OpenAI TTS, Azure Speech (TTS); LiveKit, Daily, WebSocket (transport). Source: `docs/reference/providers.md` — Voice providers section.

```go
import (
    "context"

    "github.com/lookatitude/beluga-ai/voice/s2s"
    _ "github.com/lookatitude/beluga-ai/voice/s2s/providers/openai"
)

func streamVoice(ctx context.Context, transport AudioTransport) error {
    session, err := s2s.New("openai", s2s.Config{Model: "gpt-4o-realtime-preview"})
    if err != nil {
        return err
    }
    for event, err := range session.Stream(ctx) {
        if err != nil {
            return err
        }
        if event.Type == s2s.EventAudioOutput {
            if err := transport.Send(event.Audio); err != nil {
                return err
            }
        }
    }
    return nil
}
```

Evidence: `voice/s2s/s2s.go`, `voice/s2s/registry.go`. Provider lists: `docs/reference/providers.md`.

Competitor comparison: LiveKit Agents — Python + Node.js only; Beluga integrates *with* LiveKit transport while adding agent reasoning, memory, and guards. ADK-Go — no voice pipeline. Eino — no voice pipeline. OpenAI Agents SDK — voice support is Python-only.

---

### 5. OWASP-mapped agentic guard pipeline

Beyond generic content moderation, `guard/agentic/` maps to the OWASP Top 10 for LLM Applications: tool misuse, privilege escalation, data exfiltration, cascade failure, prompt injection, memory poisoning. These compose into an `AgenticPipeline` and run at tool-execution time, not only at input/output boundaries.

Evidence: `guard/agentic/pipeline.go:124-170` — `guardNameToRisk` + `riskSeverity` (critical/high/medium per risk type). `guard/memory/guard.go`, `guard/memory/signed.go`. External providers: Lakera, NeMo Guardrails, LLM Guard, Guardrails AI, Azure AI Content Safety in `guard/providers/`.

Competitor comparison: LangChain — NeMo/Lakera integrations exist but not composed into a typed pipeline with tool-time enforcement. ADK-Go — no dedicated guard layer. Eino — no guard layer documented. OpenAI Agents SDK — `guardrails` field covers input only. Semantic Kernel — Azure AI Content Safety; no multi-stage agentic guard pipeline.

---

### 6. `iter.Seq2` streaming — composable, no goroutine leaks

Every streaming API uses `iter.Seq2[T, error]` (Go 1.23 range-over-func). Cancellation propagates through `context.Context`. Composing two streams is function application. There are no background goroutines to leak.

```go
// agent/agent.go:36 — Stream returns iter.Seq2[Event, error]
for event, err := range a.Stream(ctx, "summarise this document") {
    if err != nil {
        return err
    }
    fmt.Print(event.Text)
}
```

Evidence: `agent/agent.go:36` — `Stream(...) iter.Seq2[Event, error]`. Streaming pattern: `.wiki/patterns/streaming.md`.

Competitor comparison: LangChainGo — uses channels for streaming (leak risk, awkward composition). Eino — callback/channel hybrid. ADK-Go — not documented as `iter.Seq2`.

---

### 7. Uniform four-ring extension model across all 30+ packages

Every extensible package follows the same contract: Interface (≤4 methods) → Registry (`Register`/`New`/`List`) → Middleware (`func T→T`) → Hooks (nil-safe, composable). Learning one package transfers immediately to all others. This is enforced by `/arch-validate` and the pre-commit gate.

```go
import (
    "github.com/lookatitude/beluga-ai/llm"
    _ "github.com/lookatitude/beluga-ai/llm/providers/anthropic"
)

// The same four-ring shape applies to llm, tool, memory, rag, voice, guard, workflow...
base, err := llm.New("anthropic", llm.Config{Model: "claude-opus-4-5"})
if err != nil {
    panic(err)
}
wrapped := llm.ApplyMiddleware(base,
    llm.WithRateLimit(rpm, tpm),
    llm.WithRetry(3),
)
wrapped.SetHooks(llm.ComposeHooks(auditHooks, costHooks))
```

Evidence: `docs/architecture/03-extensibility-patterns.md`. `WithTracing()` in 17 packages: `agent/middleware.go`, `llm/`, `tool/`, `memory/tracing.go`, `workflow/tracing.go`, and 12 others.

Competitor comparison: ADK-Go — modular but no single enforced extension contract across all capability packages. Eino — component abstractions exist but no middleware/hooks layer. LangGraph — graph-node extension model, less uniform across capability types.

---

## Detailed comparisons

### vs LangChain / LangGraph

**What they do better:**
- Ecosystem size: vastly more community integrations, cookbooks, and tutorials.
- LangGraph Platform: managed deployment, persistence, human-in-the-loop UI — production SaaS.
- LangSmith: first-class observability and evaluation platform.
- Graph-based control flow is the most expressive for complex, non-linear workflows.
- No Go requirement — teams not using Go cannot adopt Beluga.

**What Beluga does better:**
- Go-native: single binary, no Python runtime, no GIL, predictable memory profile under load.
- `iter.Seq2` streaming vs channel-based streaming (no leak risk, composable by function application).
- Production signals built in — circuit breakers, rate limits, multi-tenancy — vs ecosystem add-ons.
- 7 reasoning strategies vs custom graph roll-your-own.
- No vendor lock-in to a SaaS platform for durability.
- `core/` compiles with zero cloud SDK dependencies.

**Honest verdict:** LangGraph is the dominant ecosystem choice for Python teams. For Go shops that don't want a Python runtime in their critical path, Beluga is the most complete alternative. The surface areas are not identical — LangGraph's graph-node model and LangSmith are genuinely better tooled for teams building novel orchestration topologies.

---

### vs Google ADK (Python + Go v1.0)

**What they do better:**
- Deep Vertex AI / Google Cloud integration (Workload Identity, VPC Service Controls, Cloud Logging).
- YAML-driven agent definitions — lower barrier for non-Go engineers.
- Cross-language (Python, TypeScript, Go, Java) A2A interop via ADK Core Specification.
- Google-backed release cadence and enterprise support SLA.
- Genkit-based plugin ecosystem.

**What Beluga does better:**
- 7 reasoning strategies vs ADK-Go's 1 (SequentialAgent + tool loop). Source: `github.com/google/adk-go`.
- Built-in durable execution (not in ADK-Go; Cloud Run restart is not event-log replay).
- Frame-based voice pipeline — ADK has no STT/TTS/S2S pipeline.
- Provider-agnostic: not biased toward any cloud vendor.
- Richer guard pipeline (OWASP agentic risk mapping, memory guard, degradation policy).
- 4 deployment modes; ADK-Go is primarily Cloud Run / Vertex Agent Engine.

**Honest verdict:** If your team is deep in Google Cloud, ADK-Go is the natural choice for straightforward agents. Beluga wins on reasoning depth, voice, durability, and infrastructure neutrality.

---

### vs ByteDance Eino

**What they do better:**
- Battle-tested at ByteDance scale (Doubao, TikTok).
- Component abstractions (ChatModel, Tool, Retriever) are idiomatic Go.
- Strong graph/DAG workflow for composing LLM pipelines.
- Backed by ByteDance engineering team with fast release cadence.

**What Beluga does better:**
- 7 reasoning strategies vs Eino's 3 (ReAct, Plan-Execute-Replan, Supervisor). Source: CloudWeGo docs (`cloudwego.io/docs/eino`).
- Built-in durable execution (not in Eino).
- Frame-based voice pipeline (not in Eino).
- OWASP agentic guard pipeline (not in Eino).
- `iter.Seq2` streaming vs callback/channel hybrid.
- 4 deployment modes from one codebase.
- HITL with Slack/email/webhook dispatch (`hitl/hitl.go`, `hitl/notifier.go`).
- RBAC/ABAC + OPA integration (`auth/rbac.go`, `auth/abac.go`, `auth/opa.go`).
- Eval framework with trajectory, simulation, red team — `eval/trajectory/`, `eval/redteam/`, `eval/simulation/`.

**Honest verdict:** Eino is the closest Go-native competitor. It has production pedigree and clean abstractions. Beluga's advantage is depth: more reasoning strategies, durable execution, voice, guards, and eval — the full stack for production agents, not just LLM pipeline orchestration.

---

### vs OpenAI Agents SDK

**What they do better:**
- First-mover advantage; tightest integration with OpenAI's own models and Responses API.
- Simplest possible API for single-model agents.
- OpenAI-managed tracing in the platform dashboard.
- Handoff pattern is clean and well-documented.

**What Beluga does better:**
- Go-native binary (no Python runtime).
- Multi-provider (20+ LLMs, not just OpenAI). Source: `docs/reference/providers.md`.
- Built-in durable execution (OpenAI SDK has none — Agentspan.ai is third-party paid).
- 7 reasoning strategies vs 1.
- Frame-based voice pipeline in Go.
- RBAC/ABAC, cost tracking, audit log, multi-tenancy (`auth/rbac.go`, `eval/cost/`, `audit/`, `core.WithTenant(ctx)`).

**Honest verdict:** OpenAI Agents SDK is the fastest path to a working OpenAI agent in Python. Beluga is the Go production system when you need provider independence and enterprise controls backed by specific shipped features.

---

### vs CrewAI

**What they do better:**
- Fastest time-to-demo for role-based multi-agent systems.
- Lowest learning curve — opinionated role/goal/backstory pattern makes simple crews trivial.
- Large community of pre-built crews and role templates.

**What Beluga does better:**
- Go-native.
- Production durability, observability, multi-tenancy — CrewAI's monitoring tooling is less mature.
- Type-safe `iter.Seq2` streaming; no channels.
- Guard pipeline, HITL, RBAC/ABAC.
- 7 reasoning strategies vs CrewAI's default task delegation.

**Honest verdict:** CrewAI is a prototyping accelerator. Beluga is a production framework. Different jobs.

---

### vs LlamaIndex

**What they do better:**
- The deepest RAG feature set in any framework: document parsing, chunking, indexing, reranking, evaluation — exceptionally mature.
- 160+ data source integrations.
- LlamaCloud for managed RAG pipelines.

**What Beluga does better:**
- Go-native (Python is a hard requirement for LlamaIndex).
- Full agent stack beyond RAG: voice, durable workflows, multi-agent orchestration.
- Beluga's RAG covers hybrid BM25+vector+RRF, CRAG, HyDE, Adaptive retrieval, GraphRAG, RAPTOR, ColBERT (`rag/retriever/`) — production-capable, though not as deep as LlamaIndex's specialist feature set.

**Honest verdict:** For Python teams whose primary use case is RAG, LlamaIndex is the specialist. Beluga's RAG is production-capable but not the reason to choose it — choose it when you need the full agent stack in Go.

---

### vs LiveKit Agents

**What they do better:**
- Best-in-class realtime transport layer (WebRTC, TURN, SFU).
- Managed LiveKit Cloud for turn-key voice agent hosting.
- Large ecosystem of realtime SDKs (mobile, web, embedded).
- `VoicePipelineAgent` is simple and well-documented.

**What Beluga does better:**
- Go-native (LiveKit Agents is Python + Node.js only).
- Beluga integrates LiveKit as one transport option (`voice/transport/providers/livekit`) while adding agent reasoning, memory, guards, and durable workflows.
- Full agent stack — LiveKit Agents is voice-only.

**Honest verdict:** LiveKit is the transport layer. Beluga can sit on top of it. They are not mutually exclusive. For Go teams building voice agents, Beluga + LiveKit transport is the natural stack.

---

### vs Semantic Kernel

**What they do better:**
- Best Azure integration story (Azure AI Content Safety, Copilot Studio, Azure Cognitive Services).
- .NET-first — natural choice for .NET shops.
- Built-in planners (Sequential, Stepwise, Action) and semantic memory with Azure AI Search.
- Microsoft enterprise support and compliance certifications.

**What Beluga does better:**
- Go-native (Semantic Kernel has no Go SDK).
- 7 reasoning strategies vs 3 built-in planners.
- Built-in durable execution (`workflow/workflow.go`).
- Frame-based voice pipeline.
- OWASP-mapped agentic guard pipeline (`guard/agentic/pipeline.go`).
- Provider neutrality — not Azure-biased.

**Honest verdict:** Semantic Kernel is the obvious choice for .NET/Azure shops. Beluga is the obvious choice for Go/cloud-neutral shops.

---

## Enterprise and production signals

These are concrete, shipped features — not roadmap items.

| Feature | Code path |
|---|---|
| OTel GenAI spans at every package boundary | `WithTracing()` in 17 packages: `agent/middleware.go`, `llm/`, `tool/`, `memory/tracing.go`, `workflow/tracing.go`, `auth/tracing.go`, `hitl/tracing.go`, `rag/retriever/tracing.go`, `voice/s2s/` + 9 more |
| Durable execution with crash recovery | `workflow/workflow.go` — `DurableExecutor` interface; providers: inmemory, NATS, Kafka, Temporal, Dapr, Inngest. Activity retry: `workflow/retry.go` |
| HITL approval gates | `hitl/hitl.go`, `hitl/manager.go`, `hitl/notifier.go` — confidence-based routing; Slack/email/webhook dispatch |
| RBAC + ABAC + capability-based auth | `auth/rbac.go`, `auth/abac.go`, `auth/auth.go`. OPA integration: `auth/opa.go`. Capability constants: `auth/auth.go:39-51` |
| Multi-tenant isolation | `core.WithTenant(ctx)` pattern; `auth/credential/context.go`; per-tenant namespacing in memory and state stores |
| 3-stage guard pipeline (Input → Output → Tool) | `guard/pipeline.go`; agentic risks: `guard/agentic/pipeline.go`. External providers: Lakera, NeMo, LLM Guard, Guardrails AI, Azure Safety |
| Circuit breakers + hedging + retry + rate limits | `resilience/` — composable as middleware on any `ChatModel`, `Tool`, or `Retriever` |
| Cost tracking | `eval/cost/` — `cost.go`, `budget.go`, `pareto.go`, `report.go`; cost middleware on LLM calls |
| Audit log | `audit/` package |
| Prompt injection detection | `guard/injection.go`; Spotlighting: `guard/spotlighting.go` |
| PII detection + redaction | `guard/pii.go` |
| Memory guard (poisoning + signed memory) | `guard/memory/guard.go`, `guard/memory/signed.go`, `guard/memory/circuit.go` |
| Sandboxed tool execution | `tool/sandbox/sandbox.go` — `Sandbox` interface; `tool/sandbox/process.go`; `tool/sandbox/pool.go` |
| Eval with trajectory, red team, simulation | `eval/trajectory/`, `eval/redteam/`, `eval/simulation/`. Providers: Ragas, Braintrust, DeepEval |
| Hot-reload config (file, Consul, etcd, K8s) | `config/` — `Load[T]`, validation, hot-reload |
| 4 deployment modes from same codebase | Library, Docker, Kubernetes (CRD + operator), Temporal — `docs/architecture/17-deployment-modes.md` |
| A2A + MCP protocol | `protocol/` — MCP server/client (Streamable HTTP); A2A server/client with AgentCard at `/.well-known/agent.json` |
| Speculative execution | `agent/speculative/executor.go`, `agent/speculative/predictor.go`, `agent/speculative/validator.go` |
| Plan caching | `agent/plancache/` — keyword matcher, store, integration |
| Metacognitive monitoring | `agent/metacognitive/` — monitor, extractor, plugin, store |

---

## When NOT to use Beluga

- **Your team does not use Go.** Beluga is Go-native and idiomatic. If your production stack is Python, TypeScript, or .NET, the ecosystem friction is real. LangGraph (Python), Pydantic AI (Python), and Semantic Kernel (.NET) are better fits.
- **You need the deepest possible RAG pipeline.** LlamaIndex's RAG feature set is more mature — richer document parsing, more chunking strategies, broader data source support. Beluga's RAG is production-capable but not the specialist tool.
- **You're deep in Google Cloud.** ADK-Go's Vertex AI and Cloud Run integrations, Workload Identity support, and Google enterprise SLA are genuine advantages for Google Cloud shops. Beluga's provider neutrality is a cost if you're not paying it.
- **You're deep in Azure.** Semantic Kernel's Azure AI, Copilot Studio, and Azure Cognitive Services integrations are first-class. Beluga treats Azure as one of many providers.
- **You need a managed SaaS platform today.** LangGraph Platform provides managed persistence, deployment, and a human-in-the-loop UI without self-hosting. Beluga ships the components to build this yourself — that's an upside for operators who want control, and a cost for teams that want managed infrastructure out of the box.
- **You're prototyping and speed-to-demo is the only metric.** CrewAI's role/goal/backstory pattern gets a multi-agent demo running in minutes. Beluga's composable architecture has more upfront surface area; the payoff is at production scale, not at demo.
