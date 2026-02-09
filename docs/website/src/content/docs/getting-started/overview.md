---
title: Overview
description: Beluga AI v2 is a production-ready Go framework for building agentic AI systems with streaming-first design, 22+ LLM providers, and enterprise-grade infrastructure.
---

Beluga AI v2 is a Go-native framework for building production agentic AI systems. It provides a unified architecture for LLM orchestration, agent reasoning, RAG pipelines, voice AI, and multi-agent collaboration — all built on Go's strengths in concurrency, type safety, and operational reliability.

The framework synthesizes production patterns from Google ADK, OpenAI Agents SDK, LangGraph, ByteDance Eino, and LiveKit into a single, coherent system with 157 packages and 2,885 tests.

```go
// Create an agent with tools in a few lines
a := agent.New("assistant",
    agent.WithLLM(model),
    agent.WithTools(tools),
    agent.WithPersona(agent.Persona{Role: "helpful assistant"}),
)

result, err := a.Invoke(ctx, "What are the latest GPU prices?")
```

## Why Beluga AI?

**Go-native.** Built from the ground up in Go, not a port from Python. Uses `iter.Seq2[T, error]` for streaming, `context.Context` for cancellation and tracing, and functional options for configuration. Every interface is designed for Go developers.

**Production-ready.** Circuit breakers, hedged requests, rate limiting, graceful shutdown, multi-tenancy, capability-based security, and OpenTelemetry instrumentation are built into every layer — not bolted on after the fact.

**Pluggable everything.** Every package exposes extension interfaces, a registry, lifecycle hooks, and middleware. Add a custom LLM provider, retrieval strategy, or reasoning planner without touching framework code.

## Feature Highlights

### Streaming-First Design

Every component produces typed event streams using Go 1.23+ `iter.Seq2[T, error]`. Request/response is a degenerate case of streaming, not the other way around. Backpressure and flow control are built into the stream abstraction.

```go
for event, err := range agent.Stream(ctx, "Research GPU pricing") {
    if err != nil { break }
    switch event.Type {
    case agent.EventText:
        fmt.Print(event.Text)
    case agent.EventToolCall:
        fmt.Printf("Calling: %s\n", event.ToolCall.Name)
    }
}
```

### 22+ LLM Providers

Connect to OpenAI, Anthropic, Google, Ollama, AWS Bedrock, Groq, Mistral, DeepSeek, xAI, Cohere, Together, Fireworks, Azure OpenAI, OpenRouter, Perplexity, HuggingFace, Cerebras, SambaNova, LiteLLM, Llama.cpp, Qwen, and Bifrost. Each provider registers via `init()` — import the package and it's available.

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/openai"

model, err := llm.New("openai", config.ProviderConfig{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "gpt-4o",
})
```

The **LLM Router** routes across multiple backends using pluggable strategies: round-robin, lowest latency, cost-optimized, capability-based, failover chain, or learned routing.

### Agent Framework

Build agents with pluggable reasoning strategies on a cost-quality spectrum:

| Strategy | Best For |
|----------|----------|
| **ReAct** | General-purpose tasks (default) |
| **Reflexion** | Quality-sensitive tasks with self-correction |
| **Self-Discover** | Cost-sensitive deployments (10-40x fewer calls) |
| **Tree-of-Thought** | Combinatorial problems |
| **Graph-of-Thought** | Non-linear reasoning |
| **LATS** | Complex reasoning where quality justifies cost |
| **Mixture of Agents** | Ensemble quality across multiple models |

Agents support **handoffs-as-tools** (the OpenAI pattern), where agent-to-agent transfers appear as callable tools in the LLM's tool list.

### RAG Pipeline

Default **hybrid search** combining dense vector retrieval, BM25 sparse search, and RRF fusion. Advanced strategies include Corrective RAG (CRAG), Adaptive RAG, HyDE, SEAL-RAG, and GraphRAG. Supports 11+ vector stores including pgvector, Qdrant, Pinecone, Weaviate, Milvus, and Redis.

### Voice AI

Frame-based voice pipeline inspired by Pipecat. Three composable modes:

- **Cascading**: STT → LLM → TTS
- **Speech-to-Speech**: Native audio-in/audio-out (OpenAI Realtime, Gemini Live)
- **Hybrid**: S2S for conversation, cascade for tool-heavy turns

Includes Silero VAD, semantic turn detection, and transport adapters for LiveKit, Daily, and WebSocket.

### Tool System with MCP

Type-safe `FuncTool` wraps any Go function as a tool with auto-generated JSON Schema. **MCP (Model Context Protocol)** support discovers and wraps remote tool servers using the Streamable HTTP transport. MCP registry discovery finds servers from public registries.

### MemGPT 3-Tier Memory

Four memory tiers following the MemGPT/Letta model:

- **Core**: Always-in-context persona and human blocks, self-editable by the agent
- **Recall**: Searchable conversation history across sessions
- **Archival**: Vector-based long-term storage with embedding retrieval
- **Graph**: Entity-relationship storage for structured knowledge

### Safety & Guard Pipelines

Three-stage defense-in-depth: input guards (prompt injection detection, spotlighting), output guards (content moderation, PII redaction), and tool guards (capability checks, input validation).

### Observability

OpenTelemetry GenAI semantic conventions baked into every boundary. Six metric categories: latency, token usage, cost, error rates, tool success rates, and quality scores. Adapters for Langfuse and Arize Phoenix.

### Resilience Patterns

Circuit breakers, hedged requests, adaptive retry with jitter, and provider-aware rate limiting (RPM, TPM, concurrent). Applied as middleware — wrap any `ChatModel` without changing code.

### Auth & Security

RBAC, ABAC, and capability-based security. Agents operate with explicit, minimal permissions. Default-deny for network access. Open Policy Agent integration.

### Orchestration & Workflows

Five orchestration patterns (supervisor, hierarchical, scatter-gather, router, blackboard) plus a built-in durable execution engine that survives crashes, rate limits, and human-in-the-loop pauses.

### Protocol Interoperability

First-class **MCP** (Streamable HTTP) for tool/resource/prompt access and **A2A** (Agent-to-Agent protocol) for cross-system agent collaboration. Expose any agent as an A2A server or consume remote A2A agents as sub-agents.

## Who Is It For?

Beluga AI is built for **Go developers building production AI systems**:

- Teams that need AI agents running in Go services alongside existing infrastructure
- Organizations requiring enterprise-grade observability, security, and resilience
- Developers building multi-agent systems that need protocol interoperability
- Teams building voice AI applications with sub-second latency requirements

## Architecture

The framework is organized in seven layers. Data flows downward through typed event streams; each layer only depends on the layers below it:

1. **Application Layer** — Your code, CLI tools, API servers
2. **Agent Runtime** — Persona engine, pluggable reasoning loop, executor, handoffs
3. **Protocol Gateway** — MCP, A2A, REST/gRPC/WebSocket/SSE
4. **Orchestration** — Chain, Graph, Durable Workflow, Supervisor, Router
5. **Capability Layer** — LLM, Tools, Memory, RAG, Voice, Guard
6. **Cross-Cutting** — Resilience, Cache, Auth, HITL, Evaluation
7. **Foundation** — Schema types, Stream primitives, Config, Observability

The framework is a **host** — everything else is a **plugin**. Core defines contracts; providers, reasoning strategies, tools, and agent types are added from application code with zero framework changes.

## Next Steps

- [Installation](/getting-started/installation/) — Set up Beluga AI in your project
- [Quick Start](/getting-started/quick-start/) — Build your first agent in 5 minutes
- [Building Your First Agent](/guides/first-agent/) — In-depth agent tutorial
- [Architecture](/architecture/concepts/) — Design decisions and rationale
