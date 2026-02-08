# Beluga AI v2 â€” Integrations, Tools & Providers

Based on comprehensive market research across the AI ecosystem as of early 2026, the following integrations position Beluga v2 as the most complete Go AI framework available. Items are organized by package and priority-ranked:

- **P1** = Critical â€” required for launch, already planned in architecture
- **P2** = Important â€” significant competitive advantage, implement in Phase 2-3
- **P3** = Nice-to-have â€” ecosystem completeness, implement in Phase 4 or community-driven

**Total: 107 integrations across 12 categories.**

---

## 1. LLM Providers

Maps to: `llm/providers/`

| # | Provider | Priority | Notes |
|---|----------|----------|-------|
| 1 | **OpenAI** (GPT-4o, GPT-4.1, o1, o3) | P1 | Already planned. Include structured output, vision, and audio modes. |
| 2 | **Anthropic** (Claude Opus 4, Sonnet 4.5, Haiku 4.5) | P1 | Already planned. Include extended thinking, tool use, and prompt caching. |
| 3 | **Google Gemini** (2.5 Pro, 2.5 Flash, Gemini 3) | P1 | Already planned. Include 1M+ context, multimodal, and code execution. |
| 4 | **AWS Bedrock** (multi-model gateway) | P1 | Already planned. Unified access to Claude, Llama, Mistral, Titan on AWS. |
| 5 | **Ollama** (local models) | P1 | Already planned. Essential for local development and privacy-sensitive deployments. |
| 6 | **Groq** (ultra-fast inference) | P1 | Already planned. 500+ tokens/sec on Llama/Mixtral â€” critical for latency-sensitive agents. |
| 7 | **Mistral AI** (Large, Medium, Codestral) | P2 | Apache 2.0 models. Strong coding + multilingual. $0.40/M input â€” excellent cost/performance. |
| 8 | **DeepSeek** (V3, R1, Coder V2) | P2 | Chinese open-source leader. R1 rivals GPT-4 in reasoning at fraction of cost ($0.27/M). |
| 9 | **xAI Grok** (Grok 3, 3.5) | P2 | Real-time data access. Apache 2.0 314B base model for fine-tuning. |
| 10 | **Cohere** (Command R+, Embed V4) | P2 | Enterprise RAG specialist. Built-in citations, 10-language support. Non-commercial default license â€” needs commercial arrangement. |
| 11 | **Together AI** (200+ models) | P2 | Meta-provider: Llama 4, DeepSeek, Qwen, Gemma. Pay-per-token. Fireworks-like but wider catalog. |
| 12 | **Fireworks AI** (FireAttention engine) | P2 | Up to 12x accelerated inference. SOC 2 + HIPAA. Excellent for multimodal. |
| 13 | **Azure OpenAI** (enterprise wrapper) | P2 | Enterprise compliance (HIPAA, SOC 2, GDPR). Private endpoints. Regional deployment. |
| 14 | **Alibaba Qwen** (Qwen 3, QwQ-32B) | P3 | Leading Asian open-source LLM. Apache 2.0. Strong multilingual + coding. |
| 15 | **Perplexity pplx-API** (search-grounded LLM) | P3 | LLM with built-in web search. Unique for research agents. |
| 16 | **SambaNova** (ultra-fast Llama serving) | P3 | Custom silicon. Fastest Llama inference. |
| 17 | **Cerebras** (wafer-scale inference) | P3 | ~1800 tok/s for Llama 70B. Fastest available inference hardware. |
| 18 | **OpenRouter** (meta-gateway) | P3 | Access 100+ models through one API. Automatic fallbacks. Useful for rapid prototyping. |
| 19 | **Hugging Face Inference** (60,000+ models) | P3 | Access any HF model via unified API. Essential for research/fine-tuning workflows. |
| 20 | **Meta Llama** (Llama 4 Scout/Maverick) | P3 | Via Ollama/Together/Fireworks. 400B parameters, 1M context. Apache 2.0. |

---

## 2. Embedding Providers

Maps to: `rag/embedding/providers/`

| # | Provider | Priority | Notes |
|---|----------|----------|-------|
| 21 | **OpenAI Embeddings** (text-embedding-3-large) | P1 | Already planned. 3072 dimensions. $0.13/M tokens. Industry standard. |
| 22 | **Google Embedding** (text-embedding-005) | P1 | Already planned. Tight Vertex AI integration. |
| 23 | **Ollama Embeddings** (nomic-embed, mxbai) | P1 | Already planned. Local embedding for privacy. |
| 24 | **Cohere Embed** (Embed V4, V3) | P1 | Already planned. Best multilingual embeddings. Built-in compression. |
| 25 | **Voyage AI** (voyage-3-large) | P2 | Top-ranked on MTEB benchmark. Optimized for code and legal text. |
| 26 | **Jina Embeddings** (jina-embeddings-v3) | P2 | 8192 token context. Excellent for long documents. Late-interaction model. |
| 27 | **Mistral Embed** | P3 | Aligned with Mistral LLMs for consistent semantic space. |
| 28 | **Sentence Transformers** (via Ollama/HF) | P3 | Open-source. 100+ specialized models on HuggingFace. |

---

## 3. Vector Store Providers

Maps to: `rag/vectorstore/providers/`

| # | Provider | Priority | Notes |
|---|----------|----------|-------|
| 29 | **pgvector** (PostgreSQL) | P1 | Already planned. Best for teams already using Postgres. 471 QPS at 99% recall on 50M vectors. |
| 30 | **Qdrant** | P1 | Already planned. Rust-based. Best real-time performance. Rich payload filtering. 9k+ GitHub stars. |
| 31 | **Pinecone** | P1 | Already planned. Fully managed. Easiest setup. Best for teams wanting zero ops. |
| 32 | **ChromaDB** | P1 | Already planned. Developer-friendly. Best for prototyping and local development. |
| 33 | **Weaviate** | P2 | GraphQL API. Best hybrid search (vector + BM25 + metadata in single query). Built-in vectorization modules. |
| 34 | **Milvus / Zilliz** | P2 | Enterprise scale. GPU-accelerated. Handles billions of vectors. Kubernetes-native. 35k+ GitHub stars. |
| 35 | **Turbopuffer** | P2 | S3-based serverless. Used by Cursor, Notion, Anthropic. 10x cheaper than alternatives. Go SDK available. |
| 36 | **Redis (RediSearch)** | P2 | In-memory vector search. Ultra-low latency. Good for cache + vector hybrid use cases. |
| 37 | **Elasticsearch / OpenSearch** | P2 | Combines mature text search with vector fields. Best for existing ES/OS deployments. |
| 38 | **MongoDB Atlas Vector Search** | P3 | For teams already on MongoDB. Integrated vector + document store. |
| 39 | **Vespa** | P3 | Full-featured search platform. Billion-scale. Best for recommendation engines. |
| 40 | **SQLite-vec** | P3 | Embedded vector search. Perfect for edge/mobile/CLI tools. Zero infrastructure. |

---

## 4. Voice Providers

Maps to: `voice/stt/providers/`, `voice/tts/providers/`, `voice/s2s/providers/`

### 4.1 Speech-to-Text (STT)

| # | Provider | Priority | Notes |
|---|----------|----------|-------|
| 41 | **Deepgram** (Nova-3, Nova-3 Medical, Flux) | P1 | Already planned. Leading voice agent STT. Sub-300ms. 36 languages. Medical variant. |
| 42 | **ElevenLabs** (Scribe v2) | P1 | Highest accuracy STT (14.5% WER). |
| 44 | **OpenAI Whisper / GPT-4o Transcribe** | P1 | Already planned. Whisper for batch, GPT-4o Transcribe for streaming. |
| 47 | **AssemblyAI** (Slam-1 SLM) | P2 | Already planned. Speech Language Model â€” prompt-based customization for domain terms. |
| 49 | **Groq STT** (Whisper on LPU, distil-whisper) | P2 | Blazing fast Whisper inference. 200x realtime speed. |
| 53 | **Gladia** (Solaria-1) | P3 | 100 languages including 42 underserved languages. ~270ms latency. |

### 4.2 Text-to-Speech (TTS)

| # | Provider | Priority | Notes |
|---|----------|----------|-------|
| 42 | **ElevenLabs** (TTS) | P1 | Already planned. Highest quality TTS. |
| 43 | **Cartesia** (Sonic TTS) | P1 | Already planned. Ultra-low latency streaming TTS for voice agents. |
| 48 | **PlayHT** (PlayDialog, Play3.0-mini) | P2 | Conversational TTS. Partners with Groq + LiveKit. Voice cloning in 300ms. |
| 49 | **Groq TTS** (PlayAI TTS on LPU) | P2 | Blazing fast TTS on custom hardware. |
| 50 | **Fish Audio** (Fish-Speech TTS) | P3 | Multilingual TTS. Strong in Asian languages. Open-source model available. |
| 51 | **LMNT** | P3 | Low-latency conversational TTS. Developer-focused. Simple API. |
| 52 | **Smallest.ai** (Lightning) | P3 | Ultra-low cost TTS. Best voice cloning in class. |

### 4.3 Speech-to-Speech (S2S)

| # | Provider | Priority | Notes |
|---|----------|----------|-------|
| 45 | **OpenAI Realtime API** | P1 | Already planned. Native voice-to-voice. GPT-4o multimodal. |
| 46 | **Gemini Live** | P1 | Already planned. Google's bidirectional voice. Tight Vertex integration. |
| 56 | **Amazon Nova S2S** | P2 | Already planned. AWS native speech-to-speech. Bedrock integration. |

### 4.4 Voice Activity Detection (VAD)

| # | Provider | Priority | Notes |
|---|----------|----------|-------|
| 54 | **Silero VAD** (open-source) | P2 | Already planned. Best open-source VAD. Runs on CPU. Essential for voice pipelines. |
| 55 | **WebRTC VAD** | P3 | Browser-native VAD. For web-based voice agents. |

### 4.5 Voice Frameworks

| # | Provider | Priority | Notes |
|---|----------|----------|-------|
| 57 | **Pipecat** (open-source voice framework) | P3 | Transport adapter. Compatible audio pipeline framework by Daily.co. |

---

## 5. Memory Store Backends

Maps to: `memory/stores/`

| # | Provider | Priority | Notes |
|---|----------|----------|-------|
| 58 | **Redis** | P1 | Already planned. Best for session memory with TTL. Pub/sub for event bus. |
| 59 | **PostgreSQL** | P1 | Already planned. Relational memory with full SQL query support. |
| 60 | **SQLite** | P1 | Already planned. Embedded. Zero-setup. Best for CLI tools and local dev. |
| 61 | **Neo4j** | P2 | Graph memory backend for entity-relationship memory. Cypher queries. |
| 62 | **DragonflyDB** | P3 | Redis-compatible but 25x faster. Drop-in replacement for high-throughput. |
| 63 | **Memgraph** | P3 | In-memory graph database. Faster than Neo4j for real-time traversals. |
| 64 | **MongoDB** | P3 | Document store for flexible conversation history schemas. |

---

## 6. Document Loaders

Maps to: `rag/loader/`

| # | Provider | Priority | Notes |
|---|----------|----------|-------|
| 65 | **Firecrawl** | P2 | Web crawling + scraping as API. Handles JS-rendered pages. Used by LlamaIndex, LangChain. |
| 66 | **Unstructured.io** | P2 | Universal document parser. PDF, DOCX, PPTX, HTML, images with table extraction and OCR. |
| 67 | **Docling** (IBM) | P2 | Open-source document understanding. PDF/DOCX to structured output. Table extraction. |
| 68 | **Confluence Loader** | P3 | Enterprise wiki integration. Common in enterprise deployments. |
| 69 | **Notion Loader** | P3 | Notion database and page extraction. Popular in startups. |
| 70 | **GitHub Loader** | P3 | Repos, issues, PRs, wikis. Essential for code-aware agents. |
| 71 | **Google Drive Loader** | P3 | Docs, Sheets, Slides via Google API. |
| 72 | **S3 / GCS / Azure Blob Loader** | P3 | Cloud storage buckets. |

---

## 7. Guardrails & Safety Providers

Maps to: `guard/`, `guard/adapters/`

| # | Provider | Priority | Notes |
|---|----------|----------|-------|
| 73 | **NVIDIA NeMo Guardrails** | P2 | Colang-based programmable guardrails. GPU-accelerated. Integrates with jailbreak detection and fact-checking models. |
| 74 | **Guardrails AI** (guardrails-ai) | P2 | Validator Hub with 50+ pre-built validators. PII, toxicity, schema enforcement. Complements NeMo. |
| 75 | **LLM Guard** | P3 | Input/output scanners: anonymize PII, detect toxicity, check bias. Lightweight. |
| 76 | **Lakera** | P3 | Prompt injection detection API. Real-time protection. |
| 77 | **Azure AI Content Safety** | P3 | Microsoft's content moderation API. Enterprise-grade. |

---

## 8. Evaluation & Observability Tools

Maps to: `eval/`, `o11y/adapters/`

| # | Provider | Priority | Notes |
|---|----------|----------|-------|
| 78 | **Langfuse** | P2 | Open-source LLM observability. 19k+ GitHub stars. MIT license. Self-hostable. Traces, evals, prompt management. |
| 79 | **Arize Phoenix** | P2 | OTel-native LLM observability. Open-source. First-class LangChain/LlamaIndex support. |
| 80 | **RAGAS** | P2 | RAG evaluation metrics: faithfulness, relevance, context precision/recall. Industry standard. |
| 81 | **DeepEval** | P3 | LLM evaluation framework. Unit-test style. 14+ metrics. CI/CD integration. |
| 82 | **LangSmith** | P3 | LangChain's evaluation platform. Rich tracing and dataset management. |
| 83 | **Opik** (Comet) | P3 | Apache 2.0. Built-in guardrails + evaluation. 7-14x faster than Phoenix/Langfuse for eval. |
| 84 | **Braintrust** | P3 | Evaluation-focused. Strong A/B testing for prompts. |

---

## 9. Workflow / Orchestration Engines

Maps to: `workflow/providers/`

| # | Provider | Priority | Notes |
|---|----------|----------|-------|
| 85 | **Temporal** | P1 | Durable execution. Go SDK. Used by NVIDIA, Snap, Retool for AI agents. Signals for HITL. Full event history. |
| 86 | **NATS JetStream** | P2 | Lightweight message queue with persistence. Go-native. Good for event bus and async agent communication. |
| 87 | **Apache Kafka / Confluent** | P3 | Enterprise event streaming. MCP server available. Real-time data for agents. |
| 88 | **Dapr** | P3 | Go-based distributed application runtime. State management, pub/sub, actor model. Dapr Agents extension for multi-agent. |
| 89 | **Inngest** | P3 | Durable functions platform. Event-driven. Simpler than Temporal for basic workflows. |

---

## 10. Protocols & Interoperability

Maps to: `protocol/`, `tool/`

| # | Provider | Priority | Notes |
|---|----------|----------|-------|
| 90 | **MCP Go SDK** (official) | P1 | Already planned. Official Go SDK for Model Context Protocol. Server + client. |
| 91 | **A2A Go SDK** (official) | P1 | Already planned. Official Go SDK for Agent-to-Agent protocol. |
| 92 | **MCP Registry Client** | P2 | Discover MCP servers from the official MCP Registry (Go + PostgreSQL). |
| 93 | **Composio MCP** | P3 | 100+ pre-built MCP servers for popular tools (Slack, GitHub, Jira, Salesforce). |
| 94 | **OpenAI Agents SDK Compatibility** | P3 | Wire-compatible with OpenAI Agents SDK tool format. Enables interop with OpenAI agent ecosystem. |

---

## 11. HTTP Framework Adapters

Maps to: `server/adapters/`

| # | Provider | Priority | Notes |
|---|----------|----------|-------|
| 95 | **Gin** | P1 | Already planned. Most popular Go web framework. |
| 96 | **Fiber** | P1 | Already planned. Express-like. Fastest Go web framework. |
| 97 | **Echo** | P1 | Already planned. Minimalist with good middleware support. |
| 98 | **Chi** | P1 | Already planned. stdlib-compatible router. |
| 99 | **gRPC** | P1 | Already planned. Protocol Buffers. High-performance RPC. |
| 100 | **Connect-Go** | P2 | gRPC-compatible with HTTP/1.1 fallback. No proxy required. Better developer experience than raw gRPC. |
| 101 | **Huma** | P3 | API framework with auto-generated OpenAPI docs. Good for agent API servers. |

---

## 12. Additional Infrastructure

Maps to: various packages

| # | Provider | Priority | Notes |
|---|----------|----------|-------|
| 102 | **LiteLLM Gateway** | P2 | Python proxy supporting 100+ LLMs with unified API. Useful as a backend for the LLM Router. |
| 103 | **Bifrost Gateway** | P2 | Go-native LLM gateway. Sub-100Âµs overhead. OpenAI-compatible API for 15+ providers. |
| 104 | **Apigee (Google)** | P3 | Enterprise API management. Translates any API into MCP server. Governance + rate limiting. |
| 105 | **LiveKit** (transport) | P1 | Already planned. Real-time audio/video rooms. WebRTC. Essential for voice agents. |
| 106 | **Daily.co** (transport) | P3 | Alternative to LiveKit. Pipecat framework uses it. |
| 107 | **Kong AI Gateway** | P3 | Enterprise API gateway with AI traffic management. |

---

## Summary by Priority

| Priority | Count | Description |
|----------|-------|-------------|
| **P1** | 35 | Critical â€” required for launch |
| **P2** | 37 | Important â€” significant competitive advantage |
| **P3** | 35 | Nice-to-have â€” ecosystem completeness |

### Implementation Phase Mapping

| Phase | Providers |
|-------|-----------|
| **Phase 1** (Weeks 1-4) | OpenAI, Anthropic, Groq (LLM); OpenAI (embeddings); pgvector, in-memory (vector); Redis, in-memory (cache); Temporal (workflow) |
| **Phase 2** (Weeks 5-8) | Google Gemini, Ollama, Bedrock (LLM); Cohere, Voyage, Jina (embeddings); Qdrant, Weaviate, Milvus, Turbopuffer (vector); Deepgram, ElevenLabs, Cartesia (voice); OpenAI Realtime, Gemini Live (S2S); Langfuse, Phoenix (observability); RAGAS (eval) |
| **Phase 3** (Weeks 9-12) | MCP Go SDK, A2A Go SDK, MCP Registry (protocols); NeMo, Guardrails AI (safety); NATS JetStream (workflow); Gin, Fiber, Echo, Chi, gRPC (HTTP); Silero VAD; Neo4j (graph memory) |
| **Phase 4** (Weeks 13-16) | Mistral, DeepSeek, xAI, Together, Fireworks, Cohere (LLM); Pinecone, ChromaDB, Elasticsearch, SQLite-vec (vector); PlayHT, Fish Audio, LMNT, AssemblyAI (voice); DragonflyDB, Memgraph (memory); Connect-Go, Huma (HTTP); Bifrost, LiteLLM (infrastructure) |
