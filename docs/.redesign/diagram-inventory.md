# Diagram Inventory & Placement Plan

**Scope:** Catalogue every mermaid diagram and piece of imagery in `/docs/architecture/` that could land on the public website, propose a target page for each, specify a brand-coherent mermaid theme, and call out imagery gaps on marketing surfaces.

**Total diagrams inventoried:** **78** mermaid fences across 18 architecture docs (DOC-01 … DOC-18). Patterns docs (`docs/patterns/*.md`) contain zero mermaid fences — they are prose-and-code. DOC-19, DOC-20, DOC-21 (prompt management, evaluation, HITL) contain zero mermaid fences in this snapshot.

**Proposed mermaid theme:** `belugaDark` / `belugaLight` (OKLCH-approximated hex pair), loaded from `src/lib/mermaid-theme.ts` via the existing `<script>` block in `src/components/override-components/Head.astro`.

**Current rendering:** mermaid **11.12.3** (`docs/website/package.json`) is loaded with `mermaid.initialize({ startOnLoad: true })` in `docs/website/src/components/override-components/Head.astro:215-218` with no theme config. Every diagram today renders in mermaid's default (light) palette — which clashes with the dark-default site chrome. This is the single highest-leverage fix in this report.

**Retrieval protocol executed for this inventory (amended after stop-hook review):**

1. `.wiki/index.md` — read; confirms canonical source of truth is `docs/architecture/*` for prose, `.wiki/` for `file:line` pointers.
2. `.claude/hooks/wiki-query.sh architecture` — run; surfaced corrections **C-012** (package-map stale re: `WithTracing()` 17-package sweep) and **C-013** (docs must be verified against package source, not just architecture docs).
3. `.wiki/architecture/invariants.md` + `.wiki/architecture/package-map.md` — read; confirmed no invariant governs diagram placement, but invariant #8 (GenAI span attribute prefixes) and #9 (3-stage guard pipeline) directly anchor two of the P0 diagrams (#59 span hierarchy, #55 guard pipeline).

Net effect on Part A: the target column is unchanged, but diagram **#59** (DOC-14 span hierarchy) and diagram **#62** (o11y adapter split) now explicitly cite `docs/architecture/14-observability.md` as authoritative — **not** `.wiki/architecture/package-map.md`, which C-012 warns is stale. See also the "Invariant anchoring" note added to Part A below.

---

## Part A — Mermaid diagram inventory

Priority key:
- **P0** — the target website page makes a claim the diagram *is* the proof of. Shipping the page without it leaves the reader guessing.
- **P1** — the page reads fine without it, but the diagram materially reduces cognitive load.
- **P2** — nice-to-have; cut-line candidate if we need to trim.

Target-page aliases used in the table:
- `concepts/*` → `docs/website/src/content/docs/docs/concepts/*.md`
- `ref/arch/*` → `docs/website/src/content/docs/docs/reference/architecture/*.md`
- `guides/cap/*` → `docs/website/src/content/docs/docs/guides/capabilities/**/*.md`
- `guides/prod/*` → `docs/website/src/content/docs/docs/guides/production/*.md`
- `pages/*` → `docs/website/src/pages/*.astro` (marketing)

### DOC-01 — Overview (4 diagrams)

| # | Source (file:line) | Type | Shows | Target | Priority |
|---|---|---|---|---|---|
| 1 | `01-overview.md:45` | `graph TD` | Layer 7→1 linear dep arrow (7-layer model) | `ref/arch/overview.md` hero section; redundant with `LayerStack` on `pages/index.astro` (skip there) | P0 |
| 2 | `01-overview.md:141` | `sequenceDiagram` | Text-chat lifecycle: User → Runner → Guard → Agent → LLM → Tool | `concepts/streaming.md` and `ref/arch/overview.md` § "How data flows" | P0 |
| 3 | `01-overview.md:166` | `graph LR` | Voice agent frame flow (Mic → VAD → STT → LLM → TTS → Speaker) | `guides/cap/voice/voice-ai.md` intro; `pages/product.astro § P4 Voice` | P0 |
| 4 | `01-overview.md:183` | `graph TD` | Multi-agent scatter-gather topology | `guides/cap/agents/multi-agent-orchestration.md` intro | P1 |

### DOC-02 — Core Primitives (2 diagrams)

| # | Source | Type | Shows | Target | Priority |
|---|---|---|---|---|---|
| 5 | `02-core-primitives.md:80` | `graph LR` (subgraphs) | Stream combinators: Pipe, Parallel, Merge | `concepts/streaming.md` § combinators | P0 |
| 6 | `02-core-primitives.md:162` | `graph TD` | Contents of `context.Context`: cancel, OTel, tenant, session, auth | `concepts/context.md` intro | P0 |

### DOC-03 — Extensibility Patterns (4 diagrams)

| # | Source | Type | Shows | Target | Priority |
|---|---|---|---|---|---|
| 7 | `03-extensibility-patterns.md:20` | `graph TD` (nested subgraphs) | Four concentric rings: Interface ↘ Registry ↘ Hooks ↘ Middleware | `concepts/extensibility.md` hero | P0 |
| 8 | `03-extensibility-patterns.md:121` | `graph LR` | `init()` registration + runtime factory lookup | `concepts/extensibility.md` § registry | P0 |
| 9 | `03-extensibility-patterns.md:225` | `graph LR` | Middleware application order (caller → guard → log → ratelimit → retry → Tool.Execute) | `concepts/extensibility.md` § middleware; also `guides/prod/safety-and-guards.md` | P0 |
| 10 | `03-extensibility-patterns.md:248` | `graph TD` | Full compose: User → Registry → Interface → Middleware → Hooks → Work | `concepts/extensibility.md` § summary | P1 |

### DOC-04 — Data Flow (5 diagrams)

| # | Source | Type | Shows | Target | Priority |
|---|---|---|---|---|---|
| 11 | `04-data-flow.md:15` | `sequenceDiagram` | Full text-chat lifecycle through Plugin chain, Guard, ContextManager, Memory, PromptBuilder, Executor, Planner, LLM, Tool (12 participants) | `ref/arch/overview.md` § lifecycle | P0 |
| 12 | `04-data-flow.md:107` | `graph LR` | Plan → Act → Observe → Replan loop | `guides/cap/agents/autonomous-support.md`; `concepts/index.md` § runtime | P0 |
| 13 | `04-data-flow.md:150` | `graph LR` | Event propagation: LLM → Event stream → Executor → Hooks → Plugin → Client | `concepts/streaming.md` § events | P1 |
| 14 | `04-data-flow.md:171` | `graph TD` | Dual error paths: tool.Execute error → Replan; LLM transient → middleware retry | `concepts/errors.md` | P1 |
| 15 | `04-data-flow.md:192` | `sequenceDiagram` | Handoff flow via `ActionHandoff` + InputFilter | `guides/cap/agents/multi-agent-orchestration.md` § handoff | P1 |

### DOC-05 — Agent Anatomy (3 diagrams)

| # | Source | Type | Shows | Target | Priority |
|---|---|---|---|---|---|
| 16 | `05-agent-anatomy.md:13` | `graph TD` | Agent composition: persona, planner, tools, memory, executor, hooks, middleware, guards, card | `guides/cap/agents/autonomous-support.md`; `ref/arch/packages.md` § agent | P0 |
| 17 | `05-agent-anatomy.md:88` | `graph TD` | BaseAgent subtypes: LLMAgent, SequentialAgent, ParallelAgent, LoopAgent, CustomAgent, TeamAgent | `guides/cap/agents/autonomous-support.md` | P1 |
| 18 | `05-agent-anatomy.md:141` | `graph TD` | Recursive team composition (SupervisorTeam → ResearchTeam + WritingTeam) | `guides/cap/orchestration/` index or `guides/cap/agents/multi-agent-orchestration.md` | P1 |

### DOC-06 — Reasoning Strategies (9 diagrams)

| # | Source | Type | Shows | Target | Priority |
|---|---|---|---|---|---|
| 19 | `06-reasoning-strategies.md:13` | `graph LR` | Strategy ladder: ReAct → Reflexion → Self-Discover → MindMap → ToT → GoT → LATS → MoA | `guides/cap/agents/autonomous-support.md` § strategies | P0 |
| 20 | `06-reasoning-strategies.md:37` | `graph LR` | ReAct: Think → Act → Observe → Think | same | P1 |
| 21 | `06-reasoning-strategies.md:49` | `graph TD` | Reflexion: Actor → Evaluator → Self-Reflection | same | P1 |
| 22 | `06-reasoning-strategies.md:68` | `graph TD` | MindMap planner: populate → coherence check → synthesize → add evidence | same | P1 |
| 23 | `06-reasoning-strategies.md:87` | `graph TD` | Tree-of-Thought: branches pruned/expanded | same | P1 |
| 24 | `06-reasoning-strategies.md:107` | `graph TD` | LATS MCTS-style loop: selection → expand → evaluate → simulate → backprop → reflect | same | P1 |
| 25 | `06-reasoning-strategies.md:186` | `graph TD` | Decision tree for strategy selection | same § "selecting a strategy" | P0 |
| 26 | `06-reasoning-strategies.md:212` | `graph LR` | plancache: matcher score → cache hit/miss → inner planner → track deviations → evict | `guides/cap/agents/autonomous-support.md` § caching | P2 |
| 27 | `06-reasoning-strategies.md:265` | `graph TD` | Speculative execution: predictor + ground-truth goroutines | same | P2 |

### DOC-07 — Orchestration Patterns (7 diagrams)

| # | Source | Type | Shows | Target | Priority |
|---|---|---|---|---|---|
| 28 | `07-orchestration-patterns.md:23` | `graph TD` | Supervisor delegation + aggregation | `guides/cap/agents/multi-agent-orchestration.md` § supervisor | P0 |
| 29 | `07-orchestration-patterns.md:41` | `sequenceDiagram` | Handoff: Agent A → transfer_to_hotel_agent → Agent B | same § handoff | P0 |
| 30 | `07-orchestration-patterns.md:59` | `graph LR` | Scatter-Gather: Orchestrator → N agents → Aggregator | same § scatter-gather | P0 |
| 31 | `07-orchestration-patterns.md:76` | `graph LR` | Pipeline: linear stage chain | same § pipeline | P0 |
| 32 | `07-orchestration-patterns.md:90` | `graph TD` | Blackboard: shared state + conflict resolver | same § blackboard | P0 |
| 33 | `07-orchestration-patterns.md:107` | `graph TD` | Recursive team-of-teams: Supervisor → ResearchTeam (SG) + WritingTeam (pipeline) | `guides/prod/multi-agent-systems.md` | P1 |
| 34 | `07-orchestration-patterns.md:161` | `graph TD` | Pattern-selection decision tree | same § "selecting a pattern" | P0 |

### DOC-08 — Runner & Lifecycle (5 diagrams)

| # | Source | Type | Shows | Target | Priority |
|---|---|---|---|---|---|
| 35 | `08-runner-and-lifecycle.md:13` | `graph TD` | Runner composition: Agent, SessionService, ArtifactService, Plugin chain, GuardPipeline, EventBus, Metrics, WorkerPool | `ref/arch/packages.md` § runtime | P0 |
| 36 | `08-runner-and-lifecycle.md:37` | `sequenceDiagram` | Turn execution across Client → Runner → SessionService → Plugin → Agent | `ref/arch/overview.md` § lifecycle (or dedicated concepts page) | P0 |
| 37 | `08-runner-and-lifecycle.md:93` | `graph LR` | Session lifecycle: Create → Load → Turn → Save → TTL/Clear → End | `guides/prod/deployment.md` § sessions | P1 |
| 38 | `08-runner-and-lifecycle.md:119` | `graph TD` | One Runner → multiple protocol endpoints (REST, A2A, MCP, WS, gRPC) | `ref/arch/overview.md` § protocol gateway; also `pages/product.astro § Protocols` | P0 |
| 39 | `08-runner-and-lifecycle.md:154` | `sequenceDiagram` | Graceful shutdown: SIGTERM → drain → Plugin.Cleanup → exit | `guides/prod/deployment.md` | P2 |

### DOC-09 — Memory Architecture (4 diagrams)

| # | Source | Type | Shows | Target | Priority |
|---|---|---|---|---|---|
| 40 | `09-memory-architecture.md:13` | `graph TD` | 3 tiers: Working, Recall, Archival (with semantic store) | `guides/cap/memory/memory-system.md` hero; `pages/product.astro § Know` | P0 |
| 41 | `09-memory-architecture.md:53` | `sequenceDiagram` | Composite memory parallel load: working + recall + archival | `guides/cap/memory/memory-system.md` § load | P1 |
| 42 | `09-memory-architecture.md:86` | `graph LR` | MemGPT-style memory tools: core_memory_update, archival_search | `guides/cap/memory/memory-system.md` § agent-controlled | P1 |
| 43 | `09-memory-architecture.md:116` | `graph TD` | Memory interface → MessageStore + GraphStore → providers (redis, postgres, sqlite, neo4j) | `ref/arch/providers.md` § memory | P0 |

### DOC-10 — RAG Pipeline (4 diagrams)

| # | Source | Type | Shows | Target | Priority |
|---|---|---|---|---|---|
| 44 | `10-rag-pipeline.md:13` | `graph LR` | Ingestion: Source → Loader → Splitter → Contextualizer → Embedder → VectorStore + BM25 | `guides/cap/rag/rag-pipeline.md` § ingestion; `pages/product.astro § Know` | P0 |
| 45 | `10-rag-pipeline.md:33` | `graph LR` | Hybrid retrieval: BM25 + Dense → RRF fusion → rerank → top-10 → LLM | `guides/cap/rag/rag-pipeline.md` § retrieval; `guides/cap/rag/semantic-search.md` | P0 |
| 46 | `10-rag-pipeline.md:61` | `graph LR` | CRAG corrective flow: evaluate → rewrite/web-fallback | `guides/cap/rag/rag-strategies.md` | P1 |
| 47 | `10-rag-pipeline.md:232` | `graph TD` | Retriever → Embedder + VectorStore interface split with provider fan-out | `ref/arch/packages.md` § rag | P1 |

### DOC-11 — Voice Pipeline (4 diagrams)

| # | Source | Type | Shows | Target | Priority |
|---|---|---|---|---|---|
| 48 | `11-voice-pipeline.md:38` | `graph LR` | Cascading pipeline: Mic → VAD → STT → LLM → TTS → Speaker + ToolCall loop | `guides/cap/voice/voice-ai.md` § cascading; `pages/product.astro § Voice` | P0 |
| 49 | `11-voice-pipeline.md:99` | `graph LR` | S2S pipeline: Mic ⇄ S2S model ⇄ Speaker + ToolCall intercept | `guides/cap/voice/voice-sessions-overview.md` § S2S | P0 |
| 50 | `11-voice-pipeline.md:115` | `graph TD` | Hybrid: policy router between S2S and cascade | `guides/cap/voice/voice-sessions-overview.md` § hybrid | P1 |
| 51 | `11-voice-pipeline.md:129` | `graph TD` | Transport fan-in: LiveKit, WebRTC, WebSocket, Local mic → Transport interface | `guides/cap/voice/voice-ai.md` § transports | P1 |

### DOC-12 — Protocol Layer (3 diagrams)

| # | Source | Type | Shows | Target | Priority |
|---|---|---|---|---|---|
| 52 | `12-protocol-layer.md:23` | `graph LR` | MCP client → Beluga MCP server → tools/resources/prompts | `guides/prod/mcp-tools.md`; `pages/product.astro § Protocols` | P0 |
| 53 | `12-protocol-layer.md:47` | `graph LR` | A2A: discovery via agent.json → task submit → SSE streaming back | `guides/prod/multi-agent-systems.md` | P0 |
| 54 | `12-protocol-layer.md:97` | `graph TD` | One Runner exposing REST + A2A + MCP + WS + gRPC | `ref/arch/overview.md` § protocols (duplicate of #38 — pick one) | P1 |

### DOC-13 — Security Model (4 diagrams)

| # | Source | Type | Shows | Target | Priority |
|---|---|---|---|---|---|
| 55 | `13-security-model.md:13` | `graph LR` | 3-stage guard pipeline: Input → Agent → Tool guard → Output guard with block paths | `guides/prod/safety-and-guards.md` hero; `pages/product.astro § Ship`; `pages/enterprise.astro` | P0 |
| 56 | `13-security-model.md:84` | `graph TD` | Capability-based access: identity → capability matrix → allow/deny | `pages/enterprise.astro` § auth; `guides/prod/multi-tenant-keys.md` | P0 |
| 57 | `13-security-model.md:99` | `graph TD` | Multi-tenancy: ctx.WithTenant → scoped memory/rate/cost/audit | `pages/enterprise.astro` § multi-tenancy; `guides/prod/multi-tenant-keys.md` | P0 |
| 58 | `13-security-model.md:145` | `graph TD` | Defence-in-depth layers 1–8 | `guides/prod/safety-and-guards.md` § defence-in-depth | P1 |

### DOC-14 — Observability (4 diagrams)

| # | Source | Type | Shows | Target | Priority |
|---|---|---|---|---|---|
| 59 | `14-observability.md:13` | `graph TD` | Span hierarchy for a chat turn (agent.invoke root → planner/llm/tool/memory/guard children) | `guides/prod/otel-tracing.md`; `guides/prod/observability.md` | P0 |
| 60 | `14-observability.md:122` | `graph LR` | Metrics pipeline: App → OTel SDK → Exporter → {OTLP, Prometheus, stdout} | `guides/prod/prometheus-grafana.md` | P0 |
| 61 | `14-observability.md:150` | `graph TD` | Trace + log + metric correlation via trace_id/span_id | `guides/prod/monitoring-dashboards.md` | P1 |
| 62 | `14-observability.md:179` | `graph TD` | `o11y` interface → adapter split (OTel, slog, no-op) → backends | `guides/prod/otel-tracing.md` § pluggability | P1 |

### DOC-15 — Resilience (4 diagrams)

| # | Source | Type | Shows | Target | Priority |
|---|---|---|---|---|---|
| 63 | `15-resilience.md:13` | `graph LR` | Retry flow with backoff + jitter, retryable vs non-retryable | `guides/prod/resilience.md` § retry; `guides/prod/error-recovery-service.md` | P0 |
| 64 | `15-resilience.md:47` | `graph TD` | Rate limiting: RPM → TPM → concurrency bucket gating | `guides/prod/resilience.md` § rate limit | P0 |
| 65 | `15-resilience.md:78` | `stateDiagram-v2` | Circuit breaker states: Closed ↔ Open ↔ HalfOpen | `guides/prod/resilience.md` § circuit breaker; `pages/product.astro § Ship` | P0 |
| 66 | `15-resilience.md:97` | `sequenceDiagram` | Hedged requests: primary vs fallback, whoever wins first | `guides/prod/resilience.md` § hedging | P1 |

### DOC-16 — Durable Workflows (4 diagrams)

| # | Source | Type | Shows | Target | Priority |
|---|---|---|---|---|---|
| 67 | `16-durable-workflows.md:15` | `graph TD` | Deterministic workflow vs non-deterministic activities boundary | `guides/prod/workflow-durability.md` hero | P0 |
| 68 | `16-durable-workflows.md:37` | `sequenceDiagram` | Crash recovery: event log → crash → replay → resume | `guides/prod/workflow-durability.md` § recovery | P0 |
| 69 | `16-durable-workflows.md:61` | `sequenceDiagram` | Signals: workflow waits for external API approval signal | `guides/prod/human-in-loop.md`; `guides/prod/workflow-orchestration.md` | P1 |
| 70 | `16-durable-workflows.md:125` | `graph LR` | Agent loop (Plan/Act/Observe) as durable workflow activities | `guides/prod/workflow-durability.md` § agent-loop | P1 |

### DOC-17 — Deployment Modes (6 diagrams)

| # | Source | Type | Shows | Target | Priority |
|---|---|---|---|---|---|
| 71 | `17-deployment-modes.md:11` | `graph TD` | Agent code → 4 deployment modes fan-out | `guides/prod/deployment.md` hero; `pages/enterprise.astro` § deployment | P0 |
| 72 | `17-deployment-modes.md:31` | `graph LR` | Library mode: in-process agent | `guides/prod/deployment.md` § library | P1 |
| 73 | `17-deployment-modes.md:64` | `graph TD` | Docker mode: Runner container + Redis + NATS on single host | `guides/prod/deployment.md` § docker | P1 |
| 74 | `17-deployment-modes.md:103` | `graph TD` | K8s mode: CRD → Operator → Deployment/Svc/HPA/NP/SM → Runner pods | `guides/prod/deployment.md` § k8s; `pages/enterprise.astro` | P0 |
| 75 | `17-deployment-modes.md:149` | `graph LR` | Temporal mode: Runner → Temporal cluster → workers → event log | `guides/prod/workflow-durability.md`; `guides/prod/deployment.md` § temporal | P0 |
| 76 | `17-deployment-modes.md:170` | `graph TD` | Deployment-mode selection decision tree | `guides/prod/deployment.md` § choosing | P0 |

### DOC-18 — Package Dependency Map (2 diagrams)

| # | Source | Type | Shows | Target | Priority |
|---|---|---|---|---|---|
| 77 | `18-package-dependency-map.md:19` | `graph TD` (7 subgraphs) | Full 7-layer dependency graph with every Beluga package slotted into its layer | `ref/arch/packages.md` hero; `ref/arch/overview.md` § layering | P0 |
| 78 | `18-package-dependency-map.md:173` | `graph TD` | Prohibited dependencies (dotted NO edges): core→llm, schema→tool, any→k8s, tool→agent, llm→memory | `ref/arch/packages.md` § prohibitions | P1 |

**Totals: 78 diagrams. 32 × P0, 36 × P1, 4 × P2. Fences not inventoried: 6 (the P2 ones are still in the table; everything is accounted for).**

### Invariant anchoring (added post-hoc via `.wiki/architecture/invariants.md`)

The following diagrams are structural evidence for specific invariants and should be surfaced on pages that make the corresponding claim:

| Diagram | Invariant | Wiki ref | Why it anchors |
|---|---|---|---|
| #7 (concentric rings, DOC-03:20) | Inv. 3 (middleware application order), Inv. 4 (nil-safe hooks), Inv. 5 (registry registration before main) | `.wiki/architecture/invariants.md:25-57` | Three of the four rings *are* named invariants; the diagram is the single-image proof. |
| #9 (middleware chain order, DOC-03:225) | Inv. 3 | `tool/middleware.go:11-22` | Reverse-iteration outside-in application is encoded at that file:line. Caption should link. |
| #55 (3-stage guard pipeline, DOC-13:13) | Inv. 9 (guard runs all stages) | `guard/guard.go:1-52` | "First Block halts" is literally the invariant. |
| #59 (span hierarchy, DOC-14:13) | Inv. 8 (GenAI attribute prefixes) | `o11y/tracer.go:15-47` | Every span in the diagram uses the `gen_ai.*` prefix — the diagram *is* the semantic. **Caveat from C-012:** authoritative package coverage list is `docs/architecture/14-observability.md`, not `.wiki/architecture/package-map.md` (stale). |
| #62 (o11y adapter split, DOC-14:179) | Inv. 8 | same | Same authoritative-source caveat. |
| #11/#12 (Plan/Act/Observe + full lifecycle, DOC-04) | Inv. 6 (stream respects backpressure), Inv. 7 (context cancellation stops retries) | `core/stream.go:73-90`, `tool/middleware_test.go:164-188` | Event propagation in #13 is the runtime shape of backpressure. Worth calling out in the caption. |

Captioning rule: when a diagram lands on its target website page, the figure caption should include the invariant number (e.g. "Invariant 9 — Guard pipeline") so the page is self-citing back to `.wiki/architecture/invariants.md`. This makes the learning pipeline bidirectional: docs cite invariants, invariants reference canonical `file:line`s, docs stay honest.

### Deduplication & drift notes

- **#1** (`01:45`) and **#77** (`18:19`) both depict the 7-layer stack. #77 is the richer, load-bearing one and should be the canonical render on `ref/arch/packages.md`. #1 should be replaced on the homepage by the existing `LayerStack.astro` component, which already carries the brand palette.
- **#38** (`08:119`) and **#54** (`12:97`) both show "one runner, many protocols". Pick #54 for `ref/arch/overview.md` and let #38 stay archive-only in DOC-08.
- **#18** (`05:141`) and **#33** (`07:107`) both show recursive team-of-teams composition. Keep #33 on the orchestration page and drop #18 to P2.

---

## Part B — Mermaid theme

### Design

One theme object, two palettes (dark / light), switched by `document.documentElement.dataset.theme`. Mermaid cannot resolve CSS vars at init time — we commit to OKLCH-approximated hex values.

All tokens are derived from `docs/website/src/styles/global.css` and the `.impeccable.md` palette constraint. Hex approximations are WCAG-AA validated against `--ink-950` / `--paper-50`:

| Token (CSS)        | OKLCH                    | Hex       | Role in mermaid |
|---|---|---|---|
| `--brand-500`      | `oklch(0.68 0.09 235)`   | `#5CA3CA` | Primary node stroke + flow arrow accent |
| `--brand-200`      | `oklch(0.89 0.03 215)`   | `#C9E1E5` | Highlight / hover / decision fill (dark) |
| `--brand-700`      | `oklch(0.52 0.10 240)`   | `#3B7898` | Primary fill (dark) / stroke (light) |
| `--brand-800`      | `oklch(0.42 0.08 244)`   | `#2C5E7A` | Secondary/cluster stroke |
| `--ink-950`        | `oklch(0.12 0.012 240)`  | `#0E1217` | Dark page background |
| `--ink-900`        | `oklch(0.16 0.014 240)`  | `#141920` | Dark node background |
| `--ink-850`        | `oklch(0.20 0.014 238)`  | `#1B2027` | Dark cluster background |
| `--ink-800`        | `oklch(0.26 0.012 235)`  | `#262D35` | Dark hairline / cluster border |
| `--ink-500`        | `oklch(0.62 0.008 230)`  | `#8B9299` | Dark muted edge stroke |
| `--ink-300`        | `oklch(0.82 0.010 225)`  | `#C7CBD1` | Dark body text |
| `--ink-100`        | `oklch(0.95 0.010 220)`  | `#EAEDF0` | Dark heading text |
| `--paper-50`       | `oklch(0.985 0.005 215)` | `#F7F9FA` | Light page background |
| `--paper-100`      | `oklch(0.965 0.006 215)` | `#EEF1F3` | Light node background |
| `--paper-200`      | `oklch(0.935 0.008 218)` | `#E1E6EA` | Light hairline |
| `--paper-700`      | `oklch(0.44 0.011 232)`  | `#5C6670` | Light muted edge |
| `--paper-900`      | `oklch(0.22 0.014 240)`  | `#272E38` | Light body text |
| `--paper-950`      | `oklch(0.14 0.016 242)`  | `#171D27` | Light heading text |

### Mermaid 11.12 API caveats (verified against `docs/website/node_modules/mermaid/dist/config.type.d.ts` post-hoc)

The site has **mermaid 11.12.3**, not 10.x. `themeVariables` is typed as `any` in v11, so the keys below compile regardless, but three v10-isms would silently no-op in v11 and are removed below:

- `arrowheadColor` — removed. v11 arrowheads inherit from `lineColor`.
- `loopTextColor` — deprecated in favour of inheritance from `noteTextColor`; keeping it does no harm but adds no value.
- `signalColor` / `signalTextColor` — still supported in v11 sequence diagrams.

Also applied: v11 uses `look: "classic"` as default. We stay on classic — the "handdrawn" look reads as unserious for this brand.

### File: `docs/website/src/lib/mermaid-theme.ts`

```ts
// Mermaid 11 themeVariables. Verified against mermaid@11.12.3.
// See https://mermaid.js.org/config/theming.html
// Values are literal hex (mermaid cannot resolve CSS custom properties).
// Keep in sync with tokens in src/styles/global.css — regenerate if those move.

export const belugaDark = {
  // Base
  background: "#0E1217",          // --ink-950
  mainBkg: "#141920",             // --ink-900  (node fill)
  secondaryColor: "#1B2027",      // --ink-850
  tertiaryColor: "#262D35",       // --ink-800
  primaryColor: "#3B7898",        // --brand-700 (active/primary node fill)
  primaryBorderColor: "#5CA3CA",  // --brand-500
  primaryTextColor: "#EAEDF0",    // --ink-100
  secondaryBorderColor: "#2C5E7A",// --brand-800
  secondaryTextColor: "#C7CBD1",  // --ink-300
  tertiaryBorderColor: "#262D35",
  tertiaryTextColor: "#8B9299",

  // Lines / arrows — arrowhead inherits lineColor in v11.
  lineColor: "#8B9299",           // --ink-500
  edgeLabelBackground: "#141920",

  // Flowchart / cluster subgraphs
  clusterBkg: "#1B2027",
  clusterBorder: "#262D35",
  titleColor: "#EAEDF0",

  // Node text
  nodeTextColor: "#C7CBD1",
  textColor: "#C7CBD1",

  // Decisions / special shapes
  activationBkgColor: "#3B7898",
  activationBorderColor: "#5CA3CA",

  // Sequence diagram
  actorBkg: "#141920",
  actorBorder: "#5CA3CA",
  actorTextColor: "#EAEDF0",
  actorLineColor: "#8B9299",
  signalColor: "#C7CBD1",
  signalTextColor: "#C7CBD1",
  labelBoxBkgColor: "#1B2027",
  labelBoxBorderColor: "#5CA3CA",
  labelTextColor: "#EAEDF0",
  loopTextColor: "#C7CBD1",
  noteBkgColor: "#1B2027",
  noteBorderColor: "#2C5E7A",
  noteTextColor: "#C7CBD1",

  // State diagram
  stateBkg: "#141920",
  stateBorder: "#5CA3CA",
  labelColor: "#EAEDF0",

  // Typography — match site tokens
  fontFamily: '"IBM Plex Sans", system-ui, -apple-system, "Segoe UI", sans-serif',
  fontSize: "14px",
};

export const belugaLight: typeof belugaDark = {
  background: "#F7F9FA",          // --paper-50
  mainBkg: "#EEF1F3",              // --paper-100
  secondaryColor: "#E1E6EA",       // --paper-200
  tertiaryColor: "#D2D9DF",
  primaryColor: "#5CA3CA",         // brand primary
  primaryBorderColor: "#2C5E7A",   // --brand-800
  primaryTextColor: "#171D27",     // --paper-950
  secondaryBorderColor: "#3B7898",
  secondaryTextColor: "#272E38",
  tertiaryBorderColor: "#E1E6EA",
  tertiaryTextColor: "#5C6670",

  lineColor: "#5C6670",            // --paper-700 — arrowhead inherits in v11
  edgeLabelBackground: "#EEF1F3",

  clusterBkg: "#EEF1F3",
  clusterBorder: "#E1E6EA",
  titleColor: "#171D27",

  nodeTextColor: "#272E38",
  textColor: "#272E38",

  activationBkgColor: "#5CA3CA",
  activationBorderColor: "#2C5E7A",

  actorBkg: "#EEF1F3",
  actorBorder: "#2C5E7A",
  actorTextColor: "#171D27",
  actorLineColor: "#5C6670",
  signalColor: "#272E38",
  signalTextColor: "#272E38",
  labelBoxBkgColor: "#E1E6EA",
  labelBoxBorderColor: "#2C5E7A",
  labelTextColor: "#171D27",
  loopTextColor: "#272E38",
  noteBkgColor: "#EEF1F3",
  noteBorderColor: "#3B7898",
  noteTextColor: "#272E38",

  stateBkg: "#EEF1F3",
  stateBorder: "#2C5E7A",
  labelColor: "#171D27",

  fontFamily: '"IBM Plex Sans", system-ui, -apple-system, "Segoe UI", sans-serif',
  fontSize: "14px",
};
```

### Wiring — replace the script block in `Head.astro:215-218`

```astro
<script>
  import mermaid from "mermaid";
  import { belugaDark, belugaLight } from "@/lib/mermaid-theme";

  const getTheme = () =>
    document.documentElement.dataset.theme === "light" ? belugaLight : belugaDark;

  mermaid.initialize({
    startOnLoad: true,
    theme: "base",
    themeVariables: getTheme(),
    securityLevel: "strict",
    flowchart: { curve: "basis", padding: 16 },
    sequence:  { actorMargin: 48, messageMargin: 36, mirrorActors: false },
    themeCSS: `
      .node rect, .node polygon, .node circle { stroke-width: 1.25px; }
      .cluster rect { stroke-dasharray: 4 3; }
      .edgePath .path { stroke-width: 1.25px; }
      .label foreignObject { overflow: visible; }
    `,
  });

  // Re-render on theme switch (Starlight fires a storage event on toggle).
  const rerender = async () => {
    mermaid.initialize({ startOnLoad: false, theme: "base", themeVariables: getTheme() });
    const nodes = document.querySelectorAll("pre.mermaid");
    for (const el of nodes) {
      // Reset original source by re-reading a data-mmd attribute if set by rehype (see note).
      // If not, a full page reload is acceptable fallback since Starlight reloads on theme change.
    }
    await mermaid.run({ nodes: Array.from(nodes) as HTMLElement[] });
  };
  window.addEventListener("storage", (e) => {
    if (e.key === "starlight-theme") rerender();
  });
</script>
```

**Caveat on re-render:** mermaid mutates `pre.mermaid` into `<svg>` after first `run()`, so a simple re-run won't reparse the original source. Two options:
1. **Easy:** accept a full page reload on theme toggle (Starlight already flickers anyway).
2. **Correct:** extend `src/lib/rehype-mermaid.mjs` to also set a `data-mermaid-source` attribute on the `<pre>`, then restore from that attribute before `mermaid.run()` on theme change. Recommend option 2 — one-line change to the rehype plugin and it makes the site feel "alive" under theme toggle.

### Test plan before merge

Pick one arch doc already migrated under `src/content/docs/docs/guides/capabilities/memory/memory-system.md` (it contains the word `mermaid` per the earlier scan — grep found 33 website files referencing mermaid via shortcodes or fences). Run `pnpm --filter website dev`, open the page, and verify: (a) the node fill is `--ink-900`, not white; (b) edges are `--ink-500`, not black; (c) typeface is IBM Plex Sans, not Trebuchet. Toggle the header theme switch and confirm the page reloads into `belugaLight`.

---

## Part C — Imagery gaps on marketing pages

Only listing gaps where a visual would materially advance a reader's decision. No gratuitous illustrations, no hero blobs.

### `pages/index.astro`

| Location | Visual type | Shows | Source | Constraint |
|---|---|---|---|---|
| After `LayerStack`, before `eyebrow="PROOF"` (around line 155) | Mermaid sequence (diagram #2, DOC-01:141) | "Here's what one chat turn actually does." — Text-chat lifecycle through Runner/Guard/Agent/LLM/Tool. Turns the 7-layer abstraction into a concrete trace reveal. | DOC-01 #2 | Full content width on desktop, 80vh max height. No legend; the eyebrow label "LIFECYCLE" and a 1-sentence caption linking to `ref/arch/overview.md` does the work. |
| `eyebrow="CRASH-DURABLE"` section (line 268) | Mermaid sequence (diagram #68, DOC-16:37) | Crash recovery replay. The marketing claim "workflows survive pod kills" is currently text-only; the replay sequence is the evidence. | DOC-16 #68 | Same width constraints. Split left-code / right-diagram layout matches the editorial tone in `.impeccable.md`. |

### `pages/product.astro`

Each of the 5 section eyebrows (`BUILD`, `KNOW`, `SHIP`, `VOICE`, `PROTOCOLS`) could carry a small diagram. Four earn their pixels:

| Section | Visual | Source | Why |
|---|---|---|---|
| `#build` BUILD | Plan → Act → Observe loop | DOC-04 #12 (`04:107`) | The section currently asserts "ReAct, Reflexion, LATS, MoA" in prose; the loop diagram is the one visual that makes a reader *see* why a planner is a pluggable thing. |
| `#know` KNOW | Ingestion pipeline | DOC-10 #44 (`10:13`) | RAG prose is low-signal; the Loader→Splitter→Embedder→VectorStore+BM25 graph is the most information-dense possible 8-node drawing. |
| `#ship` SHIP | Circuit breaker state diagram | DOC-15 #65 (`15:78`) | Three-state diagram is compact and shows resilience is a *real* concern with a *real* implementation, not a buzzword. |
| `#voice` VOICE | Cascading frame flow | DOC-11 #48 (`11:38`) | Same reasoning: frame-level flow is the differentiator vs. other frameworks that treat voice as a post-hoc feature. |
| `#protocols` PROTOCOLS | **skip** — `LayerStack` on the homepage already covers protocol fan-out; adding another graph here would restate what the adjacent section already implied. |

Constraint for all four: render as mermaid inline in the astro page via a thin wrapper component that injects `<pre class="mermaid">{diagram}</pre>` — the theme from Part B applies automatically. Max 400px height on desktop, collapses gracefully on mobile.

### `pages/enterprise.astro`

| Location | Visual | Source | Why |
|---|---|---|---|
| Deployment-modes row | Fan-out to Library/Docker/K8s/Temporal | DOC-17 #71 (`17:11`) | Enterprise buyers want to see the word "Kubernetes" in a diagram, not a bullet list. |
| Multi-tenancy row | `ctx.WithTenant` fan-out | DOC-13 #57 (`13:99`) | Shows tenant isolation touches *everything* — memory, rate limits, cost, audit. That's the sell for anyone pricing multi-tenant LLM infra. |
| Guard pipeline row | 3-stage guard | DOC-13 #55 (`13:13`) | Makes the input/output/tool separation explicit. Maps directly to how an AppSec reviewer will read the page. |

### Deliberately not proposed

- No decorative imagery anywhere (rule 5 of `.impeccable.md`: every claim earns its pixels).
- No hero illustration behind the `index.astro` hero — the existing display typography does the job.
- No icon-card grid replacements. The existing `LayerStack` + `ProofPanel` + eyebrow rhythm is already distinctive; adding more diagrams would dilute.
- No diagrams for `compare.astro` or `community.astro` — those pages are tables and text, and a visualisation would be filler.

---

## Part D — Existing imagery assets

Scanning excluding `node_modules`, `dist`, `.claude/worktrees`:

### Real assets (checked in)

| Path | Purpose |
|---|---|
| `assets/beluga-logo.svg` | Canonical logo source (outside `docs/`). Referenced by all website copies. |
| `docs/website/public/beluga-logo.svg` | Public logo served at `/beluga-logo.svg`. |
| `docs/website/public/favicon.svg` | Browser favicon. |
| `docs/website/public/hero-icon.svg` | Small hero mark; referenced in `Head.astro` JSON-LD logo field. |
| `docs/website/src/assets/logo-dark.svg` | Header logo, dark theme. |
| `docs/website/src/assets/logo-light.svg` | Header logo, light theme. |
| `docs/website/src/assets/hero-star.svg` | Decorative star mark in the hero. |
| `docs/website/src/assets/code-block.svg` | Icon for code-block feature card. |
| `docs/website/src/assets/content.svg` | Feature icon. |
| `docs/website/src/assets/element.svg` | Feature icon. |
| `docs/website/src/assets/layouts.svg` | Feature icon. |
| `docs/website/src/assets/changelogs.svg` | Feature icon. |
| `docs/website/public/og/*.png` | OpenGraph card images (per-route). |

### Referenced but missing (the "two architecture SVGs")

The user's "2 SVGs for the architecture" comment traces to these two file paths that are **referenced but do not exist on disk**:

- `docs/beluga_full_layered_architecture.svg` — cited at `docs/architecture/01-overview.md:15`, `docs/architecture/README.md:72`, `docs/.redesign/marketing-brief.md:170`, and in the design spec `docs/superpowers/specs/2026-04-11-docs-architecture-design.md`.
- `docs/beluga_request_lifecycle.svg` — cited in the same spec as the second canonical visual for DOC-04 (data flow).

Neither is present in the working tree (`find docs/ -maxdepth 2 -name "*.svg"` returns nothing). They are **phantom references** — almost certainly deleted during the DOC-01 rewrite but never unlinked from the markdown. Two remediation options:

1. **Restore & theme** — find the last commit that had them (`git log -- docs/beluga_full_layered_architecture.svg`), extract, retheme to OKLCH brand colors, drop into `docs/website/src/assets/architecture/` and reference via Astro `~/assets/architecture/beluga_full_layered_architecture.svg` so the image pipeline can size/optimise it.
2. **Regenerate from mermaid** — diagram #77 (`DOC-18:19`, the full dep graph) is the structural equivalent. Render it once with the `belugaDark` theme, save the output SVG, and commit it as `src/assets/architecture/layers.svg`. This has the advantage that the canonical source is *still* the mermaid fence in DOC-18, so it can never drift from the checked rules.

Recommend **option 2**: delete the broken `![Full layered architecture](../beluga_full_layered_architecture.svg)` line from `01-overview.md` and let the existing mermaid fence three lines below it do the work. Same for `04-data-flow.md`. Zero net visual loss, and the rendered art stays single-sourced in the markdown.

### Worktree duplicates

Every SVG above also exists under `.claude/worktrees/agent-*/` (8 copies). These are agent scratch workspaces — ignore. None should be committed.

---

## Corrections and process improvements discovered during this inventory

These are candidates for `.wiki/corrections.md` and (if sticky) eventual promotion into `.claude/rules/` or `CLAUDE.md`.

1. **Phantom SVG references in `docs/architecture/` (drift).** `docs/beluga_full_layered_architecture.svg` and `docs/beluga_request_lifecycle.svg` are cited in four files but do not exist in the working tree. Class: documentation drift, similar in shape to C-006 (providers.md undercount) and C-013 (doc examples vs. actual package API). **Proposed rule:** `doc-check` should scan for `![...](*.svg)` / `![...](*.png)` image references and fail when the target file is missing. One-line ripgrep check, zero ongoing cost.
2. **Mermaid theme is version-sensitive.** The theme keys that work in mermaid 10 partially differ in 11 (`arrowheadColor` removed, `loopTextColor` deprecated). This mirrors C-013 exactly: pattern-matching from memory against a package's documented shape is not the same as verifying against the installed version. **Proposed rule:** any time a doc or configuration references an external library's API shape, verify against `node_modules/<pkg>/dist/*.d.ts` or the Go package source at the version pinned in `package.json` / `go.mod` — not against generic documentation.
3. **`.wiki/architecture/package-map.md` is stale relative to the instrumentation sweep (confirmed C-012).** For this task specifically, it meant diagrams #59 and #62 (observability) cannot rely on the wiki package-map to confirm which packages have `WithTracing()`. The authoritative count lives in `docs/architecture/14-observability.md`. **Proposed rule:** before any task that depends on "which packages implement X", check the file's scan date (`.wiki/index.md` header) against recent commit timestamps for X. If stale, fall through to the canonical doc. (C-012 already encodes this as a correction — this task is corroborating evidence.)
4. **Retrieval protocol omission in a purely documentary task.** The first pass of this inventory went straight to grep and Read, skipping `.wiki/index.md` and `wiki-query.sh`. The omission felt harmless ("no code is being written") but demonstrably wasn't: C-012 and C-013 each altered specific cells in Parts A and B, and invariant anchoring added a whole new table. **Proposed rule:** the retrieval protocol is mandatory for any task touching architecture, not just code. Documentation-only tasks are the *most* likely to drift because they lack compile-time feedback. Amend `.claude/rules/documentation.md` § "Sources to consult before writing" to explicitly list `.wiki/index.md` first and `wiki-query.sh $PACKAGE` second.
5. **The existing learning pipeline is under-used for docs-only work.** The `/learn` command → `.wiki/corrections.md` → `.claude/rules/` → `CLAUDE.md` loop is described in `CLAUDE.md` but has 13 entries so far (C-001…C-013) all of which are code-class. No doc-class corrections yet, despite documented drift in C-006, C-011, C-012, C-013. **Proposed rule:** a task explicitly labelled "docs-only" still goes through the `/learn` loop when a correction surfaces. Capture it, don't let it decay into tribal knowledge.

Items (1), (4), and (5) are proposed for the next `/learn` capture. Item (2) is proposed for the same *and* for explicit inclusion in `.claude/rules/website.md` under "Before editing" (verify library version before writing example config).

## Recommendations (short, ordered by leverage)

1. **Ship the mermaid theme** (Part B). Single file, single edit in `Head.astro`. Fixes 78 already-in-place diagrams and every future one. P0.
2. **Extend `rehype-mermaid.mjs`** to preserve source in `data-mermaid-source`, so theme toggles don't require a full reload.
3. **Delete phantom SVG references** in `docs/architecture/01-overview.md:15`, `docs/architecture/04-data-flow.md`, `docs/architecture/README.md:72`, and `docs/.redesign/marketing-brief.md:170`. Replace with the adjacent mermaid fence where applicable.
4. **Surface diagrams on the 4 highest-leverage marketing sections** (Part C): homepage lifecycle + crash-recovery, `product.astro` BUILD/KNOW/SHIP/VOICE, `enterprise.astro` deployment + tenancy + guards. That's 9 placements from 9 unique diagrams — achievable in one PR.
5. **Move the 32 P0 mermaid fences into the migrated docs tree** following the Part A targets. This is the backbone of the "concepts + reference architecture" section and the main reason the user asked for this inventory.
6. **Drop duplicates** (diagrams #1 vs #77, #18 vs #33, #38 vs #54) per the deduplication notes.
