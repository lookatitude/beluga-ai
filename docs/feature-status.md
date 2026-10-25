# Feature Status

Single source of truth for what ships today vs what is on the roadmap.
Every capability claimed in the README or docs must have an entry here.

## Status definitions

| Status | Meaning |
|---|---|
| **Stable** | In `main`, tested, API considered frozen |
| **Beta** | In `main`, tested, API may change |
| **Experimental** | In `main`, may lack full test coverage or be subject to redesign |
| **Planned** | Not yet in `main` |

---

## Stable

These packages exist on `main` with test coverage and a frozen public API.

### Foundation (Layer 1)

| Package | Path | Notes |
|---|---|---|
| Core primitives | `core/` | `Stream[T]`, `Event[T]`, `Runnable`, `core.Error`, context helpers |
| Wire types | `schema/` | `Message`, `ContentPart`, `Tool`, `Document`, `Session` |
| Configuration | `config/` | Generic `Load[T]` (JSON files), `LoadFromEnv`, struct-tag validation, polling `FileWatcher` for hot-reload |
| Observability | `o11y/` | OTel GenAI conventions, slog adapters, `WithTracing()` |

### Cross-cutting (Layer 2)

| Package | Path | Notes |
|---|---|---|
| Resilience | `resilience/` | Circuit breaker, hedging, adaptive retry, rate limiting |
| Audit | `audit/` | Audit log store and registry |
| Cost accounting | `cost/` | Per-request cost tracking and budget enforcement |
| Agent state | `state/` | Shared agent state with Watch; inmemory provider |
| Durable workflow | `workflow/` | Built-in engine + Temporal, NATS, Dapr, Inngest, Kafka providers |
| Auth | `auth/` | RBAC, ABAC, OPA policy, composite policies, capability-based access control |

### Capability (Layer 3)

| Package | Path | Notes |
|---|---|---|
| LLM abstraction | `llm/` | 22 providers: OpenAI, Anthropic, Google, Bedrock, Azure, Ollama, Groq, Mistral, DeepSeek, xAI, Cohere, Together, Fireworks, OpenRouter, Perplexity, Qwen, Cerebras, SambaNova, HuggingFace, LiteLLM, Llama, Bifrost; plus a `mock` fixture provider for offline tests |
| Tool system | `tool/` | `FuncTool`, MCP client/registry, middleware, hooks, DAG execution |
| Memory | `memory/` | 3-tier MemGPT model; 8 store providers (inmemory, Redis, Postgres, SQLite, Neo4j, Memgraph, MongoDB, Dragonfly) |
| RAG — embedding | `rag/embedding/` | 9 providers: OpenAI, Cohere, Google, Voyage, Jina, Mistral, Ollama, SentenceTransformers, inmemory |
| RAG — vector store | `rag/vectorstore/` | 13 providers: pgvector, Pinecone, Weaviate, Qdrant, Milvus, Chroma, Redis, MongoDB, Elasticsearch, SQLiteVec, Turbopuffer, Vespa, inmemory |
| RAG — retriever | `rag/retriever/` | Hybrid (BM25+vector+RRF), CRAG, HyDE, Adaptive, ensemble, multi-query, sub-question, reranking |
| RAG — loaders | `rag/loader/` | 8 providers: Firecrawl, Docling, Unstructured, Confluence, Notion, GitHub, GDrive, Cloud Storage |
| RAG — splitter | `rag/splitter/` | Recursive, markdown, token splitters |
| Voice STT | `voice/stt/` | 6 providers: Deepgram, AssemblyAI, ElevenLabs, Gladia, Groq, Whisper |
| Voice TTS | `voice/tts/` | 7 providers: ElevenLabs, Cartesia, PlayHT, Groq, Fish, LMNT, Smallest |
| Voice S2S | `voice/s2s/` | 3 providers: OpenAI Realtime, Gemini Live, Amazon Nova |
| Voice transport | `voice/transport/` | Built-in WebSocket transport; 3 providers: LiveKit, Daily, Pipecat |
| Voice VAD | `voice/vad/` | 2 providers: Silero, WebRTC |
| Guard pipeline | `guard/` | 3-stage (Input → Output → Tool); Lakera, NeMo, LLM Guard, Guardrails AI, Azure AI Content Safety |
| Prompt management | `prompt/` | Versioning, cache-optimised building; file provider |
| Cache | `cache/` | Exact and semantic caches; inmemory provider |
| Evaluation | `eval/` | Metrics (faithfulness, relevance, hallucination, toxicity, latency, cost), dataset runner; Braintrust, DeepEval, Ragas providers |
| HITL | `hitl/` | Confidence-based approval gates; log and webhook notifiers |

### Protocol (Layer 4)

| Package | Path | Notes |
|---|---|---|
| MCP | `protocol/mcp/` | Server and client, Streamable HTTP; Composio provider |
| A2A | `protocol/a2a/` | Agent-to-Agent, `AgentCard` at `/.well-known/agent.json` |
| Server adapters | `server/` | Gin, Fiber, Echo, Chi, gRPC, Connect-Go, Huma |

### Orchestration (Layer 5)

| Package | Path | Notes |
|---|---|---|
| Multi-agent patterns | `orchestration/` | Supervisor, Handoff, Scatter-Gather, Pipeline, Blackboard, Router |

### Agent runtime (Layer 6)

| Package | Path | Notes |
|---|---|---|
| Agent framework | `agent/` | `BaseAgent`, `Executor`; planners: ReAct, Reflexion, Self-Discover, ToT, GoT, LATS, MoA, MindMap |
| Agent runtime | `runtime/` | `Runner` (session, plugins, worker pool, graceful drain), `Team` |
| Agent workflow | `agent/workflow/` | `SequentialAgent`, `ParallelAgent`, `LoopAgent` |
| Plan cache | `agent/plancache/` | Plan-level caching for repeated reasoning paths |

### Observability providers

| Package | Path | Notes |
|---|---|---|
| Langfuse | `o11y/providers/langfuse/` | OTel trace export to Langfuse |
| LangSmith | `o11y/providers/langsmith/` | OTel trace export to LangSmith |
| Opik | `o11y/providers/opik/` | OTel trace export to Opik |
| Arize Phoenix | `o11y/providers/phoenix/` | OTel trace export to Arize Phoenix |

---

## Experimental

These packages are in `main` and have tests, but their public API may change significantly or test coverage is partial. Use in production at your own risk.

| Capability | Path | Notes |
|---|---|---|
| Cognitive architectures | `agent/cognitive/` | Heuristic and LLM-based cognitive scorers |
| Metacognitive planners | `agent/metacognitive/` | Self-reflective monitoring and plugin execution |
| Evolving agents | `agent/evolving/` | Self-modifying agent patterns |
| Speculative execution | `agent/speculative/` | Speculative decoding for agents; predictor/validator |
| Red-team evaluation | `eval/redteam/` | Adversarial evaluation; API may change |
| Trajectory evaluation | `eval/trajectory/` | Multi-step trajectory metrics |
| Simulation evaluation | `eval/simulation/` | Simulated user interaction runner |

---

## Beta

These capabilities ship on `main` with tests. Their public API may still change.

| Capability | Path | Notes |
|---|---|---|
| `beluga` CLI | `cmd/beluga/` | `init`, `new` (agent, tool, planner), `run`, `dev`, `test`, `eval`, `providers`, `version`. `deploy` is a stub that prints what it would write. See [CLI reference](reference/cli.md). |
| Agent Playground | `server/playground/`, `cmd/beluga/playground/` | Embedded chat UI over SSE; `beluga dev` serves a loopback playground with spans, stderr and token/cost panels |
| Code-as-Action (CodeAct) | `agent/codeact/` | Agents that generate code and run it through a pluggable `CodeExecutor`; registered as the `codeact` planner |
| Computer Use / browser tools | `tool/computeruse/` | Screenshot, click, type and scroll actions behind URL allowlists and action rate limits |
| LLM-as-Judge | `eval/judge/` | `JudgeMetric` with rubrics, `BatchJudge`, `ConsistencyChecker` |

## Experimental sub-packages

These sub-packages are in `main` with tests but are newer and less exercised than their parent packages.

| Capability | Path |
|---|---|
| Model routing | `llm/routing/` |
| Context engineering pipeline | `llm/context/` |
| Associative, procedural, temporal, shared, RL-optimised memory; consolidation | `memory/associative/`, `memory/procedural/`, `memory/temporal/`, `memory/shared/`, `memory/rl/`, `memory/consolidation/` |
| ColBERT, RAPTOR and Text2SQL/Text2Cypher retrievers | `rag/retriever/colbert/`, `rag/retriever/raptor/`, `rag/retriever/structured/` |
| OWASP agentic guards, autonomy degradation, memory-poisoning guards | `guard/agentic/`, `guard/degradation/`, `guard/memory/` |
| Speaker diarization, SIP/PSTN telephony | `voice/diarization/`, `voice/telephony/` |
| Multi-agent debate | `orchestration/debate/` |
| Blackboard shared state | `state/blackboard/` |
| Runtime plugins, replay, sleep-time compute, dynamic team building | `runtime/plugins/`, `runtime/replay/`, `runtime/sleeptime/`, `runtime/teambuilder/` |
| AG-UI, OpenAI Agents compatibility, REST/SSE server, MCP registry | `protocol/agui/`, `protocol/openai_agents/`, `protocol/rest/`, `protocol/mcp/registry/` |
| Agent File (`.af`) format, declarative agent definitions | `config/agentfile/`, `config/declarative/` |
| Execution visualisation | `o11y/viz/` |
| Conversation clustering, cost-aware evaluation | `eval/clustering/`, `eval/cost/` |
| Semantic contracts | `agent/contract/` |
| Temporal least-privilege credentials | `auth/credential/` |
| Tool learning, sandboxed code execution | `tool/learning/`, `tool/sandbox/` |
| Deployment artifact helpers | `deploy/` |
| Kubernetes operator and admission webhooks (optional overlay, never imported by the library) | `k8s/operator/`, `k8s/webhooks/` |

## Planned

Nothing is currently tracked as planned in this file. Roadmap items live in the workspace roadmap, not here. A capability is added to this section only when the README or docs mention it before its code is on `main`.

The README and docs used to describe these as shipped; they are **not** in the tree:

| Capability | Status |
|---|---|
| Built-in tool library (`tool/builtin/`: HTTP, SQL, shell, filesystem, GitHub, arXiv) | Not present. Tools come from `tool.NewFuncTool` or MCP servers. |
| Config hot-reload from Consul, etcd or Kubernetes | Not present. Only the polling `config.FileWatcher` ships. |
| HITL Slack or email notifiers | Not present. `hitl.LogNotifier` and `hitl.WebhookNotifier` ship. |
| SEAL-RAG and GraphRAG retrievers | Not present. |

---

## How to read status in the README

Features marked `<sup>[beta](docs/feature-status.md)</sup>` in the README are tracked here under **Beta**. Features marked `<sup>[experimental](docs/feature-status.md)</sup>` are tracked under **Experimental**. If a capability in the README has no status badge, it is **Stable**.

To check a claim against the tree: every path in this file must exist (`ls <path>`), and every provider count must match [reference/providers.md](reference/providers.md), which is generated from the filesystem.
