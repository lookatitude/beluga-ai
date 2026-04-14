---
title: Config Schema
description: Every WithX() option across the framework, grouped by package, with one-line descriptions.
---

# Config Schema

Beluga uses functional options for all configurable types. Every constructor accepts a variadic `...Option`, and every option has a `WithX()` builder. This page is the per-package lookup. For full per-option type signatures and constraints, run `go doc` against the source — every option carries a doc comment with its default and any constraint.

## How to read this page

- Options here are extracted from the public source files at the time of the last build (`2026-04-13`).
- Per-call options (like LLM `WithTemperature`) configure a single `Generate` / `Stream` invocation.
- Constructor middleware options (like `llm.WithTracing`, `tool.WithRetry`) wrap the registered instance via `ApplyMiddleware`.
- Hooks options (`WithHooks`) take a `Hooks` struct with optional function fields — see [Extensibility · Ring 3](/docs/concepts/extensibility/).

## llm

Per-call options (`GenerateOption`, used in `Generate()` / `Stream()`):

| Option | Purpose |
|---|---|
| `WithTemperature(t float64)` | Sampling temperature |
| `WithMaxTokens(n int)` | Hard cap on completion tokens |
| `WithTopP(p float64)` | Nucleus sampling cutoff |
| `WithStopSequences(seqs ...string)` | Stop generation on any of these substrings |
| `WithResponseFormat(format ResponseFormat)` | Structured-output schema (JSON / JSON-schema) |
| `WithToolChoice(choice ToolChoice)` | Force / disable / auto tool selection |
| `WithSpecificTool(name string)` | Force a specific tool by name |
| `WithReasoning(cfg ReasoningConfig)` | Configure extended thinking / reasoning mode |
| `WithReasoningEffort(effort ReasoningEffort)` | Reasoning effort tier (`low` / `medium` / `high`) |
| `WithReasoningBudget(tokens int)` | Token budget for reasoning |
| `WithMetadata(kv map[string]any)` | Free-form metadata attached to the request span |

Middleware options (wrap a `ChatModel` via `llm.ApplyMiddleware`):

| Option | Purpose |
|---|---|
| `WithTracing()` | Emit `gen_ai.*` OTel spans at every boundary |
| `WithHooks(hooks Hooks)` | Lifecycle hooks (BeforeGenerate / AfterGenerate / OnError) |
| `WithLogging(logger *slog.Logger)` | Structured request/response logging |
| `WithFallback(fallback ChatModel)` | Failover to a secondary model on retryable error |
| `WithProviderLimits(limits ProviderLimits)` | Per-provider rate limit (rpm / tpm / max body size) |
| `WithMaxRetries(n int)` | Retry count for retryable errors |
| `WithContextStrategy(s ContextStrategy)` | History compaction policy |
| `WithKeepSystemMessages(b bool)` | Pin system messages through compaction |
| `WithModels(...string)` | Routing pool for the multi-model strategy |
| `WithStrategy(name string)` | Named routing strategy (`round-robin`, `cheapest`, `least-loaded`) |
| `WithTokenizer(t Tokenizer)` | Tokenizer override for context-window accounting |

## tool

| Option | Purpose |
|---|---|
| `WithTracing()` | Emit `gen_ai.tool.*` spans on `Execute` |
| `WithHooks(hooks Hooks)` | BeforeExecute / AfterExecute / OnError lifecycle |
| `WithRetry(max int)` | Retry retryable failures up to `max` times |
| `WithTimeout(d time.Duration)` | Per-execute timeout |
| `WithHTTPClient(c *http.Client)` | Custom HTTP client (used by HTTP-backed tools) |
| `WithSessionID(id string)` | Pin a tool execution to a session for stateful tools |
| `WithMCPHeaders(h http.Header)` | Custom headers for MCP-backed tools |

## memory

| Option | Purpose |
|---|---|
| `WithCore(s Store)` | Working-memory store (volatile, fast) |
| `WithRecall(s RecallStore)` | Recall-memory store (semantic search) |
| `WithArchival(s ArchivalStore)` | Archival-memory store (long-term, RAG-backed) |
| `WithGraph(s GraphStore)` | Graph backend (Neo4j, Memgraph) for relational memory |
| `WithProcedural(s ProceduralStore)` | Procedural memory for learned skills |
| `WithQueryLimit(n int)` | Cap on results returned from a recall query |
| `WithHooks(hooks Hooks)` | Lifecycle hooks |
| `WithTracing()` | Emit `gen_ai.memory.*` spans |

## rag/retriever

Composition + tuning for hybrid retrieval pipelines.

| Option | Purpose |
|---|---|
| `WithRetrievers(r ...Retriever)` | Retrievers to fuse |
| `WithHybridRRFK(k int)` | RRF constant for hybrid fusion |
| `WithTopK(k int)` | Top-K results returned |
| `WithThreshold(t float64)` | Minimum similarity score |
| `WithRelevanceThreshold(t float64)` | CRAG relevance gate |
| `WithCRAGMaxAttempts(n int)` | CRAG retry budget |
| `WithCRAGThreshold(t float64)` | CRAG escalation threshold |
| `WithRerankTopN(n int)` | Documents passed to the reranker |
| `WithMaxRewrites(n int)` | Cap on query-rewrite iterations |
| `WithMultiQueryCount(n int)` | Number of paraphrased queries |
| `WithMaxSubQuestions(n int)` | Decomposition cap |
| `WithDecomposer(d Decomposer)` | Sub-question planner |
| `WithRewriteModel(m ChatModel)` | LLM used for query rewrites |
| `WithHyDEPrompt(p string)` | Hypothetical-document-embedding prompt |
| `WithToolTopK(k int)` | Top-K when used as an agent tool |
| `WithMetadata(kv map[string]any)` | Filter metadata applied to every retrieval |
| `WithTracing()` | Emit `gen_ai.retriever.*` spans |

Per-strategy hook options: `WithCRAGHooks`, `WithAdaptiveHooks`, `WithHybridHooks`, `WithEnsembleHooks`, `WithHyDEHooks`, `WithMultiQueryHooks`, `WithRerankHooks`, `WithRewriteHooks`, `WithSubQuestionHooks`, `WithVectorStoreHooks`.

## rag/vectorstore

| Option | Purpose |
|---|---|
| `WithFilter(f Filter)` | Default metadata filter applied to every query |
| `WithStrategy(name string)` | Provider-specific search strategy |
| `WithThreshold(t float64)` | Minimum cosine similarity |
| `WithHooks(hooks Hooks)` | Lifecycle hooks |
| `WithTracing()` | Emit `gen_ai.vectorstore.*` spans |

## voice

| Option | Purpose |
|---|---|
| `WithVAD(v VAD)` | Voice-activity detector instance |
| `WithSTT(s STT)` | Speech-to-text instance |
| `WithLLM(m ChatModel)` | LLM in the cascaded pipeline |
| `WithTTS(t TTS)` | Text-to-speech instance |
| `WithTransport(t Transport)` | Audio transport (LiveKit, Daily, Pipecat) |
| `WithCascade()` | Use the cascaded STT → LLM → TTS pipeline |
| `WithHybridSession(opts ...HybridOption)` | Hybrid cascade + S2S session config |
| `WithSession(s Session)` | Pre-built session instance |
| `WithSwitchPolicy(p SwitchPolicy)` | When to switch between cascade and S2S |
| `WithChannelBufferSize(n int)` | Audio frame buffer depth |
| `WithHooks(hooks Hooks)` | Lifecycle hooks |

## guard

Pattern / signal options used by built-in guards.

| Option | Purpose |
|---|---|
| `WithKeywords(words ...string)` | Keyword list for the keyword guard |
| `WithPattern(p string)` | Regex pattern for the regex guard |
| `WithThreshold(t float64)` | Score threshold for ML-backed guards |

## workflow

| Option | Purpose |
|---|---|
| `WithStore(s WorkflowStore)` | Backing store (`temporal` / `inngest` / `dapr` / `nats` / `kafka` / `inmemory`) |
| `WithActivityRetry(p RetryPolicy)` | Default retry policy for activities |
| `WithActivityTimeout(d time.Duration)` | Default per-activity timeout |
| `WithExecutorHooks(h Hooks)` | Hooks fired around each activity execution |
| `WithHooks(h Hooks)` | Workflow-level lifecycle hooks |
| `WithTracing()` | Emit `gen_ai.workflow.*` spans |

## agent

The richest options surface — agents compose persona, planner, tools, memory, and orchestration policy.

| Option | Purpose |
|---|---|
| `WithLLM(m ChatModel)` | The agent's primary model |
| `WithPersona(s string)` | Persona / system-prompt seed |
| `WithTools(...Tool)` | Tools the agent can call |
| `WithMemory(m Memory)` | Memory backend |
| `WithPlanner(p Planner)` | Planner instance |
| `WithPlannerName(name string)` | Planner by registered name (`react`, `lats`, `tot`, `moa`, …) |
| `WithExecutorPlanner(p Planner)` | Executor's per-step planner |
| `WithExecutorMaxIterations(n int)` | Cap on Plan→Act→Observe loops |
| `WithExecutorTimeout(d time.Duration)` | Per-step timeout |
| `WithExecutorHooks(h Hooks)` | Per-step lifecycle hooks |
| `WithHooks(h Hooks)` | Top-level agent lifecycle hooks |
| `WithMetadata(kv map[string]any)` | Free-form metadata on every span |
| `WithTimeout(d time.Duration)` | End-to-end agent timeout |
| `WithMaxIterations(n int)` | Hard cap on total iterations |
| `WithMaxConcurrency(n int)` | Bounded parallelism for fan-out planners |
| `WithThreshold(t float64)` | Score threshold for graded planners |
| `WithCoherenceThreshold(t float64)` | Mind-Map coherence floor |
| `WithDependencyDetection(b bool)` | Dependency-aware planning |
| `WithReasoningModules(m ...Module)` | Self-Discover module set |
| `WithBranchFactor(n int)` | Tree-of-Thought branch width |
| `WithExpansionWidth(n int)` | Graph-of-Thought expansion width |
| `WithExplorationConstant(c float64)` | LATS exploration constant |
| `WithLATSMaxDepth(n int)` | LATS depth cap |
| `WithMaxDepth(n int)` | Generic depth cap |
| `WithMaxNodes(n int)` | Graph-of-Thought node cap |
| `WithMaxOperations(n int)` | Per-iteration operation cap |
| `WithMaxReflections(n int)` | Reflexion reflection cap |
| `WithSearchStrategy(name string)` | LATS search strategy |
| `WithGenerateCount(n int)` | Mixture-of-Agents proposer count |
| `WithMergeEnabled(b bool)` | Mixture-of-Agents response merging |
| `WithEvaluator(e Evaluator)` | Reflexion / LATS evaluator |
| `WithAggregator(a Aggregator)` | Mixture-of-Agents aggregator |
| `WithController(c Controller)` | Workflow controller |
| `WithControllerMaxIterations(n int)` | Controller iteration cap |
| `WithChildren(...Agent)` | Sub-agents for hierarchical agents |
| `WithLayers(...Layer)` | Mixture-of-Agents layer stack |
| `WithHandoffs(...Handoff)` | Handoff destinations |
| `WithHandoffContext(c HandoffContext)` | Context passed across handoffs |
| `WithContract(c Contract)` | Pre/post contract for HITL gates |
| `WithTracing()` | Emit `gen_ai.agent.*` spans |

## o11y

| Option | Purpose |
|---|---|
| `WithLogger(l *slog.Logger)` | Replace the default logger |
| `WithLogLevel(level slog.Level)` | Minimum log level |
| `WithJSON(b bool)` | Switch to JSON-formatted log output |
| `WithSpanExporter(e SpanExporter)` | OTel span exporter (Langfuse, Phoenix, Jaeger, ...) |
| `WithSampler(s Sampler)` | OTel sampler |
| `WithSyncExport(b bool)` | Synchronous span export (tests only) |

## Conventions for new options

When adding a new `WithX()`:

1. Name follows `WithX()` where `X` describes the property — never `SetX`, never `OptX`.
2. Returns the typed `Option` for the package, never a generic `interface{}`.
3. Validates input at construction; invalid options return an error from the constructor.
4. Has a sensible default. `pkg.New()` with no options must produce a working instance.
5. Doc comment includes the default value and any constraint, so `go doc` is the canonical reference.

## Related

- [Extensibility](/docs/concepts/extensibility/) — why functional options are the chosen pattern
- [API](/docs/reference/api/) — generated package-level reference
- [Resilience](/docs/guides/production/resilience/) — composition example using `WithRetry`, `WithRateLimit`, `WithCircuitBreaker`
