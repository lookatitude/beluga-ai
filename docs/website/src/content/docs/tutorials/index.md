---
title: Tutorials
description: "55 step-by-step tutorials across 12 categories, from foundation concepts to advanced voice AI pipelines."
---

Hands-on tutorials with complete, runnable code examples. Each tutorial builds on foundation concepts and walks through a specific feature or integration from start to finish. Tutorials are organized in a progressive learning path: start with Foundation to understand the core abstractions, move through Providers and Agents to build intelligence, and progress to production topics like Voice, Orchestration, and Deployment.

## Tutorial Categories

### Core Concepts

| Category | Tutorials | Topics |
|----------|-----------|--------|
| [Foundation](./foundation/) | 9 | Streams, Runnable, errors, config, lifecycle, context, batch processing. Start here to understand the building blocks that every other package depends on: `iter.Seq2` streaming, functional options, the registry pattern, and error handling with `core.Error`. |
| [Providers](./providers/) | 8 | LLM setup, switching providers, middleware chains, custom providers. Learn how to configure and swap between LLM providers using the registry pattern, build middleware chains that add cross-cutting concerns like logging and retry, and create your own provider implementations. |

### Building Intelligence

| Category | Tutorials | Topics |
|----------|-----------|--------|
| [Agents](./agents/) | 5 | Agent creation, planners, handoffs, multi-agent patterns. Build autonomous agents that reason, use tools, and delegate to other agents. Covers the `BaseAgent` abstraction, tool binding, planner-executor loops, and handoff-based multi-agent systems. |
| [Memory](./memory/) | 2 | 3-tier memory setup, conversation persistence. Implement the MemGPT-inspired memory model with Core (always-in-context), Recall (searchable history), and Archival (vector + graph) tiers for long-running conversational agents. |
| [RAG & Retrieval](./rag/) | 2 | Embedding pipelines, hybrid search strategies. Build retrieval-augmented generation pipelines that combine vector search with BM25 keyword search using Reciprocal Rank Fusion for more accurate document retrieval. |

### Production Capabilities

| Category | Tutorials | Topics |
|----------|-----------|--------|
| [Orchestration](./orchestration/) | 3 | Chains, graphs, parallel execution. Compose agents and tools into complex workflows using the orchestration package. Build sequential chains, directed acyclic graphs (DAGs), and parallel execution patterns with the supervisor agent. |
| [Safety](./safety/) | 2 | Guard pipelines, PII redaction. Implement the three-stage guard pipeline (input, output, tool) to enforce content policies, redact personally identifiable information, and prevent prompt injection before it reaches your agents. |
| [Server & Deployment](./server/) | 2 | HTTP adapters, production configuration. Expose agents as REST endpoints using Beluga's HTTP framework adapters (Gin, Fiber, Echo, Chi) and configure production settings for graceful shutdown, health checks, and TLS. |

### Specialized Domains

| Category | Tutorials | Topics |
|----------|-----------|--------|
| [Messaging](./messaging/) | 2 | Platform integrations, webhook handling. Connect agents to messaging platforms like WhatsApp and build omnichannel gateways that route conversations across multiple channels while maintaining session state. |
| [Multimodal](./multimodal/) | 2 | Image inputs, mixed content. Process images, audio, and mixed content types through LLM providers that support multimodal inputs, with automatic content type detection and fallback handling. |
| [Documents](./documents/) | 4 | Loading, splitting, ingestion pipelines. Build document processing pipelines that load files from various sources (PDF, Markdown, HTML), split them into semantically meaningful chunks, and prepare them for embedding and retrieval. |
| [Voice](./voice/) | 14 | STT, TTS, S2S, VAD, transport, real-time pipelines. Build real-time voice AI applications using Beluga's frame-based voice pipeline. Covers speech-to-text streaming, text-to-speech synthesis, native speech-to-speech models, voice activity detection, turn-taking, interruption handling, and production deployment with LiveKit and Vapi. |

## Suggested Learning Path

1. **Foundation** -- Understand streams, errors, and configuration before building anything else.
2. **Providers** -- Learn to configure LLM providers and chain middleware.
3. **Agents** -- Build your first agent with tool use and planner-executor loops.
4. **Memory + RAG** -- Add long-term memory and document retrieval to your agents.
5. **Orchestration + Safety** -- Compose multi-step workflows and enforce safety policies.
6. **Voice / Messaging / Documents** -- Specialize in your domain of interest.
7. **Server & Deployment** -- Package everything for production.

## Prerequisites

All tutorials assume you have:
- Go 1.23+ installed
- A working `go.mod` with `github.com/lookatitude/beluga-ai` as a dependency
- An API key for at least one LLM provider (OpenAI, Anthropic, etc.)

Start with the [Foundation tutorials](./foundation/) if you are new to the framework.
