---
title: Getting Started with Beluga AI
description: "Learn what Beluga AI offers — a Go-native agentic AI framework with 22+ LLM providers, streaming-first design, and production-grade infrastructure."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, getting started, Go AI framework, agentic AI, LLM framework Go, production AI, AI agents Go"
---

Beluga AI v2 is a Go-native framework for building production agentic AI systems. It provides a unified architecture for LLM orchestration, agent reasoning, RAG pipelines, voice AI, and multi-agent collaboration — all built on Go's strengths in concurrency, type safety, and operational reliability.

Most agentic AI frameworks are written in Python and designed for prototyping. Beluga takes a different approach: it starts from production requirements — observability, resilience, type safety, and operational control — and builds the developer ergonomics on top. The result is a framework where building a quick prototype and running it in production use the same code paths, the same error handling, and the same observability.

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

**Go-native.** Built from the ground up in Go, not a port from Python. Uses `iter.Seq2[T, error]` for streaming, `context.Context` for cancellation and tracing, and functional options for configuration. Every interface is designed for Go developers. This matters because Go's type system catches integration errors at compile time that would only surface at runtime in dynamically typed languages — and in agentic systems where LLMs call tools and hand off between agents, those integration boundaries are where bugs hide.

**Production-ready.** Circuit breakers, hedged requests, rate limiting, graceful shutdown, multi-tenancy, capability-based security, and OpenTelemetry instrumentation are built into every layer — not bolted on after the fact. These capabilities exist as middleware that wraps the same interfaces you already use, so adding resilience to an existing agent requires changing configuration, not rewriting code.

**Pluggable everything.** Every package exposes extension interfaces, a registry, lifecycle hooks, and middleware. Add a custom LLM provider, retrieval strategy, or reasoning planner without touching framework code. This extensibility follows the same `Register()` + `New()` + `List()` pattern in all 19 registries, so learning one package teaches you the pattern for all of them.

## Feature Highlights

### Streaming-First Design

Every component produces typed event streams using Go 1.23+ `iter.Seq2[T, error]`. Request/response is a degenerate case of streaming, not the other way around. Backpressure and flow control are built into the stream abstraction.

This design choice means that the synchronous `Invoke()` method is implemented by collecting a stream — not the other way around. When you need to show users real-time progress during a multi-step agent execution, the streaming path is already there. The `yield` function's boolean return value provides natural backpressure: if a consumer stops reading, the producer stops producing, with no goroutine leaks or channel cleanup required.

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

Providers use the same registration mechanism as Go's `database/sql` and `image` packages: a blank import triggers `init()`, which registers a factory function. Your application code calls `llm.New("openai", cfg)` without importing the provider package directly, making it trivial to swap providers in tests or across environments.

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/openai"

model, err := llm.New("openai", config.ProviderConfig{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "gpt-4o",
})
```

The **LLM Router** routes across multiple backends using pluggable strategies: round-robin, lowest latency, cost-optimized, capability-based, failover chain, or learned routing.

### Agent Framework

Agents combine an LLM, a persona, tools, and a reasoning strategy into an autonomous loop that decides what to do, executes actions, observes results, and iterates. The framework separates the reasoning strategy (the *planner*) from the execution engine (the *executor*), so you can swap reasoning approaches without changing how tools are called or how events are streamed.

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

Pure vector search misses keyword-specific queries; pure BM25 misses semantic similarity. Beluga defaults to **hybrid search** combining dense vector retrieval, BM25 sparse search, and RRF (Reciprocal Rank Fusion), which merges rankings without requiring score normalization. Advanced strategies include Corrective RAG (CRAG), Adaptive RAG, HyDE, SEAL-RAG, and GraphRAG. Supports 11+ vector stores including pgvector, Qdrant, Pinecone, Weaviate, Milvus, and Redis.

### Voice AI

Voice AI requires sub-second latency, which means the pipeline must be composable at the frame level — individual audio frames flow through processing stages without waiting for complete utterances. Beluga's frame-based voice pipeline, inspired by Pipecat, achieves this by making each stage (VAD, STT, LLM, TTS) an independent `FrameProcessor` connected by channels. Three composable modes:

- **Cascading**: STT → LLM → TTS
- **Speech-to-Speech**: Native audio-in/audio-out (OpenAI Realtime, Gemini Live)
- **Hybrid**: S2S for conversation, cascade for tool-heavy turns

Includes Silero VAD, semantic turn detection, and transport adapters for LiveKit, Daily, and WebSocket.

### Tool System with MCP

Tools are how agents interact with the outside world. Type-safe `FuncTool` wraps any Go function as a tool with auto-generated JSON Schema — you define a struct for the input, add struct tags for descriptions and constraints, and the framework generates the schema the LLM needs to call it correctly. **MCP (Model Context Protocol)** support discovers and wraps remote tool servers using the Streamable HTTP transport, so tools running on external servers appear identically to local Go functions. MCP registry discovery finds servers from public registries.

### MemGPT 3-Tier Memory

LLMs have finite context windows, so agents need a memory system that balances what's always available (low latency, small capacity) against what can be retrieved on demand (higher latency, larger capacity). Following the MemGPT/Letta model, Beluga organizes memory into four tiers:

- **Core**: Always-in-context persona and human blocks, self-editable by the agent
- **Recall**: Searchable conversation history across sessions
- **Archival**: Vector-based long-term storage with embedding retrieval
- **Graph**: Entity-relationship storage for structured knowledge

### Safety & Guard Pipelines

AI systems face threats at multiple points in the processing chain: malicious inputs before the LLM, harmful outputs after the LLM, and dangerous tool calls before side effects. Beluga addresses this with three-stage defense-in-depth: input guards (prompt injection detection, spotlighting), output guards (content moderation, PII redaction), and tool guards (capability checks, input validation). Each stage runs independently, so you can add safety providers from different vendors at each point.

### Observability

When an agent makes an unexpected decision, you need to trace the full chain: what the LLM saw, what it decided, which tool it called, and what the tool returned. Beluga integrates OpenTelemetry GenAI semantic conventions at every boundary, so every LLM call, tool execution, and agent event automatically emits spans and metrics. Six metric categories are tracked: latency, token usage, cost, error rates, tool success rates, and quality scores. Adapters for Langfuse and Arize Phoenix let you visualize agent behavior in specialized AI observability platforms.

### Resilience Patterns

LLM providers are external services that throttle requests, go down temporarily, and exhibit variable latency. Beluga provides circuit breakers, hedged requests, adaptive retry with jitter, and provider-aware rate limiting (RPM, TPM, concurrent) as middleware. Because middleware uses the `func(ChatModel) ChatModel` signature, you wrap any `ChatModel` without changing application code — resilience is additive, not invasive.

### Auth & Security

RBAC, ABAC, and capability-based security. Agents operate with explicit, minimal permissions. Default-deny for network access. Open Policy Agent integration.

### Orchestration & Workflows

Complex AI applications need coordination patterns beyond single-agent execution. Beluga provides five orchestration patterns (supervisor, hierarchical, scatter-gather, router, blackboard) plus a built-in durable execution engine that survives crashes, rate limits, and human-in-the-loop pauses. The durable engine is included so you can get started without external infrastructure; for production deployments with higher durability requirements, Temporal, NATS, Kafka, and Dapr are available as provider options behind the same `DurableExecutor` interface.

### Protocol Interoperability

Agents in production rarely operate in isolation — they need to consume external tools and collaborate with agents running in other systems. Beluga provides first-class **MCP** (Streamable HTTP) for tool/resource/prompt access and **A2A** (Agent-to-Agent protocol) for cross-system agent collaboration. Expose any agent as an A2A server or consume remote A2A agents as sub-agents, without writing transport or serialization code.

## Who Is It For?

Beluga AI is built for **Go developers building production AI systems**:

- Teams that need AI agents running in Go services alongside existing infrastructure
- Organizations requiring enterprise-grade observability, security, and resilience
- Developers building multi-agent systems that need protocol interoperability
- Teams building voice AI applications with sub-second latency requirements

## Architecture

The framework is organized in seven layers with strict dependency rules. Data flows downward through typed event streams; each layer only depends on the layers below it. This layering ensures that foundation types like `schema.Message` and `core.Error` have zero external dependencies, so they compile fast and never introduce transitive dependency conflicts. Upper layers add capabilities without polluting the types that flow through the entire system.

1. **Application Layer** — Your code, CLI tools, API servers
2. **Agent Runtime** — Persona engine, pluggable reasoning loop, executor, handoffs
3. **Protocol Gateway** — MCP, A2A, REST/gRPC/WebSocket/SSE
4. **Orchestration** — Chain, Graph, Durable Workflow, Supervisor, Router
5. **Capability Layer** — LLM, Tools, Memory, RAG, Voice, Guard
6. **Cross-Cutting** — Resilience, Cache, Auth, HITL, Evaluation
7. **Foundation** — Schema types, Stream primitives, Config, Observability

The framework is a **host** — everything else is a **plugin**. Core defines contracts; providers, reasoning strategies, tools, and agent types are added from application code with zero framework changes.

## Next Steps

- [Installation](/docs/getting-started/installation/) — Set up Beluga AI in your project
- [Quick Start](/docs/getting-started/quick-start/) — Build your first agent in 5 minutes
- [Building Your First Agent](/docs/guides/first-agent/) — In-depth agent tutorial
- [Architecture](/docs/architecture/concepts/) — Design decisions and rationale
