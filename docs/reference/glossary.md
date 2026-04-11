# Reference: Glossary

Key terms used across Beluga AI v2 documentation. Terms are grouped by topic and cross-linked to the deeper docs.

## Core concepts

**Agent.** The atomic unit of behaviour. A persona + tools + planner + memory + hooks. Implements the `core.Runnable` interface. See [DOC-05](../architecture/05-agent-anatomy.md).

**Event[T].** A typed value emitted from a stream. Carries payload, error, type, and metadata. See [DOC-02](../architecture/02-core-primitives.md).

**Executor.** The engine inside an agent that runs the Plan → Act → Observe → Replan loop. See [DOC-04](../architecture/04-data-flow.md).

**Handoff.** Transfer of control from one agent to another via an auto-generated `transfer_to_{name}` tool. See [DOC-05](../architecture/05-agent-anatomy.md#handoffs-are-tools).

**Invariant.** One of ten non-negotiable design rules. Violations are architectural defects. See [`.wiki/architecture/invariants.md`](../../.wiki/architecture/invariants.md).

**Persona.** Role + goal + backstory + traits, rendered into the system prompt. See [DOC-05](../architecture/05-agent-anatomy.md#persona-model).

**Planner.** Decides the next action for an agent. Seven strategies ship. See [DOC-06](../architecture/06-reasoning-strategies.md).

**Runnable.** The base interface every composable component implements. `Invoke` + `Stream`. See [DOC-02](../architecture/02-core-primitives.md).

**Runner.** The deployment boundary. Hosts an agent, manages sessions/plugins/guards/network. See [DOC-08](../architecture/08-runner-and-lifecycle.md).

**Session.** Per-request state that survives across turns. Stored by a `SessionService`. See [DOC-08](../architecture/08-runner-and-lifecycle.md#session-lifecycle).

**Stream[T].** A typed, pull-based streaming value wrapping `iter.Seq2[int, T]`. See [Streaming pattern](../patterns/streaming-iter-seq2.md).

**Team.** An agent whose `Stream` method delegates to an `OrchestrationPattern` over its members. See [DOC-07](../architecture/07-orchestration-patterns.md).

**Tool.** A named capability the agent can invoke. Native, MCP-backed, or remote via handoff. See [DOC-05](../architecture/05-agent-anatomy.md).

**Turn.** One round-trip through the runner: input → agent execution → events out.

## Extension mechanisms

**Factory.** `func(cfg Config) (Interface, error)`. Used to instantiate providers by name. See [Registry + Factory pattern](../patterns/registry-factory.md).

**Hook.** An optional function field that fires at a specific lifecycle point. See [Hooks Lifecycle pattern](../patterns/hooks-lifecycle.md).

**Middleware.** A `func(T) T` that wraps an interface with augmented behaviour. Applied outside-in. See [Middleware Chain pattern](../patterns/middleware-chain.md).

**Plugin.** Runner-level cross-cutting concern with `BeforeTurn`/`AfterTurn` hooks. See [DOC-08](../architecture/08-runner-and-lifecycle.md#plugins-vs-hooks).

**Provider.** A concrete implementation of a capability interface (LLM, embedding, store, etc.), registered in its package's registry. See [Provider Template pattern](../patterns/provider-template.md).

**Registry.** A name → factory map with `Register` / `New` / `List`. Every extensible package has one. See [Registry + Factory pattern](../patterns/registry-factory.md).

## Memory terms

**Archival memory.** Permanent, vector-indexed semantic store. Highest latency, highest capacity. See [DOC-09](../architecture/09-memory-architecture.md).

**Composite memory.** Wrapper that queries all three tiers + graph on load and returns a merged context. See [DOC-09](../architecture/09-memory-architecture.md#composite-memory-on-load).

**Graph memory.** Entity-relation-entity store, separate from the three tiers. See [DOC-09](../architecture/09-memory-architecture.md#graph-memory).

**Recall memory.** Cross-session medium-latency store of summaries and entities. See [DOC-09](../architecture/09-memory-architecture.md).

**Self-editable memory.** The MemGPT pattern: agent has tools to modify its own memory. See [DOC-09](../architecture/09-memory-architecture.md#self-editable-memory-the-memgpt-pattern).

**Working memory.** Session-scoped buffer of recent messages. Lowest latency, smallest capacity. See [DOC-09](../architecture/09-memory-architecture.md).

## RAG terms

**BM25.** Keyword-based ranking algorithm. Combined with vector search for hybrid retrieval.

**Contextual retrieval.** Prepending context summary to chunks before embedding. Ingestion-time optimisation. See [DOC-10](../architecture/10-rag-pipeline.md#contextual-retrieval).

**CRAG.** Corrective RAG. Evaluates retrieval quality and rewrites queries or falls back to web. See [DOC-10](../architecture/10-rag-pipeline.md#crag--corrective-retrieval).

**HyDE.** Hypothetical Document Embedding. Generate an answer, embed *that*, retrieve similar real chunks.

**Hybrid search.** BM25 + dense vectors fused with RRF. The default in Beluga. See [DOC-10](../architecture/10-rag-pipeline.md).

**Reranker.** Cross-encoder that re-scores retrieved candidates with a joint query-document model. See [DOC-10](../architecture/10-rag-pipeline.md#retrieval).

**RRF.** Reciprocal Rank Fusion. `sum(1 / (k + rank_i))` over rankings from multiple retrievers. Fuses results without score calibration.

## Planner strategies

**ReAct.** Reason-Act-Observe loop. Default planner. 1 LLM call per iteration.

**Reflexion.** Actor + Evaluator + Self-Reflection. Iterative quality improvement.

**Self-Discover.** Compose task-specific plan first, then execute it.

**Tree-of-Thought (ToT).** Expand branches, evaluate, prune, continue.

**Graph-of-Thought (GoT).** Like ToT but reasoning graph can cycle and merge.

**LATS.** Language Agent Tree Search. Monte Carlo tree search for agents.

**Mixture-of-Agents (MoA).** Run N diverse agents in parallel, aggregate.

See [DOC-06](../architecture/06-reasoning-strategies.md).

## Orchestration patterns

**Blackboard.** Agents communicate only through shared state. Conflict resolver arbitrates.

**Handoff.** Agent A transfers control to agent B via auto-generated tool.

**Pipeline.** Linear sequence. Stage N's output is stage N+1's input.

**Scatter-Gather.** Fan-out to N agents in parallel, aggregate results.

**Supervisor.** Central coordinator delegates to specialists, validates, aggregates.

See [DOC-07](../architecture/07-orchestration-patterns.md).

## Protocol terms

**A2A.** Agent-to-Agent protocol. Agents discover each other via `/.well-known/agent.json`. See [DOC-12](../architecture/12-protocol-layer.md).

**AgentCard.** JSON description of an agent's capabilities, served at `/.well-known/agent.json`. See [DOC-05](../architecture/05-agent-anatomy.md#the-a2a-agentcard).

**MCP.** Model Context Protocol. Exposes tools/resources/prompts to MCP clients (Claude Desktop, Cursor, etc.). See [DOC-12](../architecture/12-protocol-layer.md).

**SSE.** Server-Sent Events. HTTP streaming for browser consumers.

## Security terms

**Capability.** A named permission (e.g., `tool.filesystem.read`) that a tool requires and a tenant may or may not have. See [DOC-13](../architecture/13-security-model.md).

**Guard.** A check run at input, tool, or output stage. See [DOC-13](../architecture/13-security-model.md#the-three-stage-guard-pipeline).

**HITL.** Human-in-the-Loop. A workflow paused for human approval before proceeding.

**Spotlighting.** Delimiting untrusted input so the LLM treats it as data, not instructions.

**Tenant.** Isolation boundary for multi-tenant deployments. Lives in `context.Context`. See [DOC-13](../architecture/13-security-model.md#multi-tenancy-isolation).

## Resilience terms

**Circuit breaker.** Fail fast when a downstream service is visibly broken. States: Closed / Open / Half-Open.

**Hedged request.** Fire a fallback request after a delay; first response wins.

**Rate limit.** RPM / TPM / MaxConcurrent buckets per provider.

**Retry.** Automatic re-attempt on retryable errors (`ErrRateLimit`, `ErrTimeout`, `ErrProviderDown`).

See [DOC-15](../architecture/15-resilience.md).

## Observability terms

**gen_ai.* attributes.** OpenTelemetry GenAI semantic conventions (v1.37+). See [DOC-14](../architecture/14-observability.md).

**Span.** A timed, attributed unit of work in a distributed trace.

**Trace.** A directed acyclic graph of spans for one request.

## Durable workflow terms

**Activity.** A non-deterministic unit of work (LLM call, tool call, network request). Recorded in the event log.

**Event log.** Append-only log of activity schedules and results. Used for replay. See [DOC-16](../architecture/16-durable-workflows.md).

**Replay.** Re-running a workflow's orchestration code, reading activity results from the event log.

**Signal.** An external input that wakes a paused workflow (e.g., HITL approval).

**Workflow.** Deterministic orchestration code that dispatches activities. Must be replay-safe.

## Deployment terms

**Deployment mode.** Library, Docker, Kubernetes, or Temporal. See [DOC-17](../architecture/17-deployment-modes.md).

**Operator.** Kubernetes controller that reconciles `Agent` custom resources into Deployment/Service/HPA/NetworkPolicy/ServiceMonitor.

## Abbreviations

- **ADR** — Architecture Decision Record
- **BM25** — Best Match 25 (ranking algorithm)
- **CRAG** — Corrective Retrieval-Augmented Generation
- **HITL** — Human-in-the-Loop
- **HyDE** — Hypothetical Document Embeddings
- **LATS** — Language Agent Tree Search
- **MoA** — Mixture of Agents
- **MCP** — Model Context Protocol
- **OTel** — OpenTelemetry
- **RAG** — Retrieval-Augmented Generation
- **RRF** — Reciprocal Rank Fusion
- **S2S** — Speech-to-Speech
- **STT** — Speech-to-Text
- **ToT** — Tree-of-Thought
- **TPM** — Tokens Per Minute
- **TTS** — Text-to-Speech

## Related

- [01 — Overview](../architecture/01-overview.md)
- [`.wiki/architecture/invariants.md`](../../.wiki/architecture/invariants.md)
- [`./interfaces.md`](./interfaces.md)
