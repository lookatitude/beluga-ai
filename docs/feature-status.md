# Feature Status

Single source of truth for what ships today vs what is on the roadmap.
Every capability claimed in the README or docs must have an entry here.

## Status definitions

| Status | Meaning |
|---|---|
| **Stable** | In `main`, tested, API considered frozen |
| **Beta** | In `main`, tested, API may change |
| **Experimental** | In `main`, may lack full test coverage or be subject to redesign |
| **Planned** | Not yet in `main` — link to tracking PR |

---

## Stable

These packages exist on `main` with test coverage and a frozen public API.

### Foundation (Layer 1)

| Package | Path | Notes |
|---|---|---|
| Core primitives | `core/` | `Stream[T]`, `Event[T]`, `Runnable`, `core.Error`, context helpers |
| Wire types | `schema/` | `Message`, `ContentPart`, `Tool`, `Document`, `Session` |
| Configuration | `config/` | Generic `Load[T]`, validation, hot-reload |
| Observability | `o11y/` | OTel GenAI conventions, slog adapters, `WithTracing()` |

### Cross-cutting (Layer 2)

| Package | Path | Notes |
|---|---|---|
| Resilience | `resilience/` | Circuit breaker, hedging, adaptive retry, rate limiting |
| Audit | `audit/` | Audit log store and registry |
| Cost accounting | `cost/` | Per-request cost tracking and budget enforcement |
| Agent state | `state/` | Shared agent state with Watch; inmemory provider |
| Durable workflow | `workflow/` | Built-in engine + Temporal, NATS, Dapr, Inngest, Kafka providers |
| Auth | `auth/` | JWT, OAuth2, API-key providers; capability-based access control |

### Capability (Layer 3)

| Package | Path | Notes |
|---|---|---|
| LLM abstraction | `llm/` | 22 providers: OpenAI, Anthropic, Google, Bedrock, Azure, Ollama, Groq, Mistral, DeepSeek, xAI, Cohere, Together, Fireworks, OpenRouter, Perplexity, Qwen, Cerebras, SambaNova, HuggingFace, LiteLLM, Llama, Bifrost |
| Tool system | `tool/` | `FuncTool`, MCP client/registry, middleware, hooks, DAG execution |
| Memory | `memory/` | 3-tier MemGPT model; 9 store providers (inmemory, Redis, Postgres, SQLite, Neo4j, Memgraph, MongoDB, Dragonfly) |
| RAG — embedding | `rag/embedding/` | 9 providers: OpenAI, Cohere, Google, Voyage, Jina, Mistral, Ollama, SentenceTransformers, inmemory |
| RAG — vector store | `rag/vectorstore/` | 13 providers: pgvector, Pinecone, Weaviate, Qdrant, Milvus, Chroma, Redis, MongoDB, Elasticsearch, SQLiteVec, Turbopuffer, Vespa, inmemory |
| RAG — retriever | `rag/retriever/` | Hybrid (BM25+vector+RRF), CRAG, HyDE, Adaptive, ensemble, multi-query, sub-question, reranking |
| RAG — loaders | `rag/loader/` | 8 providers: Firecrawl, Docling, Unstructured, Confluence, Notion, GitHub, GDrive, Cloud Storage |
| RAG — splitter | `rag/splitter/` | Recursive, markdown, token splitters |
| Voice STT | `voice/stt/` | 6 providers: Deepgram, AssemblyAI, ElevenLabs, Gladia, Groq, Whisper |
| Voice TTS | `voice/tts/` | 7 providers: ElevenLabs, Cartesia, OpenAI TTS, PlayHT, Groq, Fish, LMNT |
| Voice S2S | `voice/s2s/` | 3 providers: OpenAI Realtime, Gemini Live, Amazon Nova |
| Voice transport | `voice/transport/` | 3 providers: LiveKit, Daily, Pipecat |
| Guard pipeline | `guard/` | 3-stage (Input → Output → Tool); Lakera, NeMo, LLM Guard, Guardrails AI, Azure AI Content Safety |
| Prompt management | `prompt/` | Versioning, cache-optimised building; file provider |
| Cache | `cache/` | Exact cache; inmemory provider |
| Evaluation | `eval/` | Metrics (faithfulness, relevance, hallucination, toxicity, latency, cost), dataset runner; Braintrust, DeepEval, Ragas providers |
| HITL | `hitl/` | Confidence-based approval gates, notifier hooks |

### Protocol (Layer 4)

| Package | Path | Notes |
|---|---|---|
| MCP | `protocol/mcp/` | Server and client, Streamable HTTP; Composio provider |
| A2A | `protocol/a2a/` | Agent-to-Agent, `AgentCard` at `/.well-known/agent.json` |
| Server adapters | `server/` | Gin, Fiber, Echo, Chi, gRPC, Connect-Go |

### Orchestration (Layer 5)

| Package | Path | Notes |
|---|---|---|
| Multi-agent patterns | `orchestration/` | Supervisor, Handoff, Scatter-Gather, Pipeline, Blackboard, Router |

### Agent runtime (Layer 6)

| Package | Path | Notes |
|---|---|---|
| Agent framework | `agent/` | `BaseAgent`, `Executor`, `Runner`; planners: ReAct, Reflexion, Self-Discover, ToT, GoT, LATS, MoA, MindMap |
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
| Optimize (cost) | `optimize/` | Cost optimization strategies; early-stage |

---

## Planned

These capabilities are advertised or commonly requested but are **not yet merged to `main`**. Do not write code that depends on them — link to the PR to follow progress.

| Capability | Tracking | Description |
|---|---|---|
| `beluga` CLI scaffolding | [PR #234](https://github.com/lookatitude/beluga-ai/pull/234) | `beluga new`, `beluga test`, `beluga run` — project scaffolding and provider smoke-tests |
| Agent Playground UI | [PR #232](https://github.com/lookatitude/beluga-ai/pull/232) | Browser-based chat UI for inspecting tool calls, planner traces, and memory state |
| Code-as-Action (CodeAct) | [PR #243](https://github.com/lookatitude/beluga-ai/pull/243) | Agents that generate and execute sandboxed code as their primary action (`agent/codeact/`) |
| Computer Use / browser tools | [PR #218](https://github.com/lookatitude/beluga-ai/pull/218) | Native click/type/scroll/screenshot tools for browser-driving agents (`tool/computeruse/`) |
| LLM-as-Judge framework | [PR #228](https://github.com/lookatitude/beluga-ai/pull/228) | Rubric-based scoring, batch evaluation, consistency checks as a first-class `eval/judge/` sub-package |

---

## How to read status in the README

Features marked `<sup>[planned](docs/feature-status.md)</sup>` in the README are tracked here under **Planned**. Features marked `<sup>[experimental](docs/feature-status.md)</sup>` are tracked here under **Experimental**.

If you see a capability in the README without a status badge, it is **Stable**.
