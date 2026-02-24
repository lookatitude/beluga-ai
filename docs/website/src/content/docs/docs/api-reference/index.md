---
title: "API Reference — Beluga AI v2"
description: "Complete API documentation for all Beluga AI v2 Go packages. Reference for core, LLM, agent, RAG, voice, and infrastructure APIs."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "API reference, documentation, Beluga AI, Go, godoc, packages, framework, agentic AI"
---

Complete API reference for all Beluga AI v2 packages. This documentation is generated from the Go source code doc comments.

## Foundation

| Package | Description |
|---------|-------------|
| [Core Package](./core/) | Foundation primitives: streams, Runnable, events, errors, lifecycle, multi-tenancy |
| [Schema Package](./schema/) | Shared types: messages, content parts, tool definitions, documents, events, sessions |
| [Config Package](./config/) | Configuration loading, validation, environment variables, and hot-reload |

## LLM & Agents

| Package | Description |
|---------|-------------|
| [LLM Package](./llm/) | ChatModel interface, provider registry, middleware, hooks, structured output, routing |
| [LLM Providers](./llm-providers/) | All LLM provider implementations: OpenAI, Anthropic, Google, Ollama, Bedrock, and more |
| [Agent Package](./agent/) | Agent runtime, BaseAgent, Executor, Planner strategies, handoffs, and event bus |
| [Agent Workflows](./agent-workflow/) | Sequential, Parallel, and Loop workflow agents for multi-agent orchestration |
| [Tool Package](./tool/) | Tool interface, FuncTool, registry, MCP client integration, and middleware |

## Memory & RAG

| Package | Description |
|---------|-------------|
| [Memory Package](./memory/) | MemGPT-inspired 3-tier memory: Core, Recall, Archival, graph memory, composite |
| [Memory Store Providers](./memory-stores/) | Memory store implementations: in-memory, Redis, PostgreSQL, SQLite, MongoDB, Neo4j, Memgraph, Dragonfly |
| [RAG Embedding](./rag-embedding/) | Embedder interface for converting text to vector embeddings |
| [Embedding Providers](./rag-embedding-providers/) | Embedding provider implementations: OpenAI, Cohere, Google, Jina, Mistral, Ollama, Voyage, and more |
| [RAG Vector Store](./rag-vectorstore/) | VectorStore interface for similarity search over document embeddings |
| [Vector Store Providers](./rag-vectorstore-providers/) | Vector store implementations: pgvector, Pinecone, Qdrant, Weaviate, Milvus, Elasticsearch, and more |
| [RAG Retriever](./rag-retriever/) | Retriever strategies: Vector, Hybrid, HyDE, CRAG, Multi-Query, Ensemble, Rerank, Adaptive |
| [RAG Document Loaders](./rag-loader/) | Document loaders for files, cloud storage, APIs, and web content |
| [RAG Text Splitters](./rag-splitter/) | Text splitting strategies for chunking documents |

## Voice

| Package | Description |
|---------|-------------|
| [Voice Package](./voice/) | Frame-based voice pipeline, VAD, hybrid cascade/S2S switching |
| [Voice STT](./voice-stt/) | Speech-to-text interface and providers: Deepgram, AssemblyAI, Whisper, Groq, ElevenLabs, Gladia |
| [Voice TTS](./voice-tts/) | Text-to-speech interface and providers: ElevenLabs, Cartesia, PlayHT, Fish, Groq, LMNT, Smallest |
| [Voice S2S](./voice-s2s/) | Speech-to-speech interface and providers: OpenAI Realtime, Gemini Live, Nova S2S |
| [Voice Transport](./voice-transport/) | Transport layer for voice sessions: WebSocket, LiveKit, Daily, Pipecat |
| [Voice VAD](./voice-vad/) | Voice activity detection providers: Silero, WebRTC |

## Infrastructure

| Package | Description |
|---------|-------------|
| [Guard Package](./guard/) | Three-stage safety pipeline: input, output, tool guards with built-in and external providers |
| [Resilience Package](./resilience/) | Circuit breaker, hedge, retry, and rate limiting patterns |
| [Cache Package](./cache/) | Exact, semantic, and prompt caching with pluggable backends |
| [HITL Package](./hitl/) | Human-in-the-loop: confidence-based approval, escalation policies |
| [Auth Package](./auth/) | RBAC, ABAC, and capability-based security |
| [Eval Package](./eval/) | Evaluation framework: metrics, runners, and provider integrations |
| [State Package](./state/) | Shared agent state with watch and notify |
| [Prompt Package](./prompt/) | Prompt management, templating, and versioning |
| [Orchestration Package](./orchestration/) | Chain, Graph, Router, Parallel, and Supervisor orchestration patterns |
| [Workflow Package](./workflow/) | Durable execution engine with provider integrations |

## Protocol & Server

| Package | Description |
|---------|-------------|
| [Protocol Package](./protocol/) | Protocol abstractions for MCP, A2A, REST, and OpenAI Agents compatibility |
| [MCP Protocol](./protocol-mcp/) | Model Context Protocol server/client, SDK, registry, and Composio integration |
| [A2A Protocol](./protocol-a2a/) | Agent-to-Agent protocol types and SDK implementation |
| [REST & OpenAI Agents](./protocol-rest/) | REST/SSE API server and OpenAI Agents protocol compatibility |
| [Server Adapters](./server/) | HTTP framework adapters: Gin, Fiber, Echo, Chi, gRPC, Connect, Huma |
| [Observability Package](./o11y/) | OpenTelemetry GenAI conventions, tracing, and provider integrations |

## Design Patterns

All extensible packages in Beluga AI v2 follow consistent patterns:

### Registry Pattern

Every extensible package provides:
- `Register(name, factory)` — register providers in `init()`
- `New(name, config)` — instantiate providers by name
- `List()` — discover available providers

### Middleware Pattern

Wrap interfaces to add cross-cutting behavior:
```go
model = llm.ApplyMiddleware(model,
    llm.WithLogging(logger),
    llm.WithRetry(3),
)
```

### Hooks Pattern

Inject lifecycle callbacks without middleware:
```go
hooks := llm.Hooks{
    BeforeGenerate: func(ctx, msgs) error { ... },
    AfterGenerate:  func(ctx, resp, err) { ... },
}
model = llm.WithHooks(model, hooks)
```

### Streaming Pattern

All streaming uses Go 1.23+ `iter.Seq2`:
```go
for chunk, err := range model.Stream(ctx, msgs) {
    if err != nil { break }
    fmt.Print(chunk.Delta)
}
```
