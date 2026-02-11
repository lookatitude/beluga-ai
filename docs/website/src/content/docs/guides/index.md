---
title: Guides
description: "In-depth guides covering every major Beluga AI v2 capability, from building agents to deploying to production."
---

Beluga AI v2 is a Go-native framework for building agentic AI systems with streaming-first design, protocol interoperability, and pluggable providers. These guides cover everything from your first LLM interaction to production deployment with observability, safety, and resilience. Each guide is self-contained with complete code examples, but they are organized into a progressive learning path so that concepts build naturally from one to the next.

## How to Use These Guides

The guides are organized into three categories that form a recommended learning path:

1. **Foundations** — Start here. These guides introduce the core abstractions that every Beluga application uses: the `ChatModel` interface for LLM interaction, the `Agent` runtime for autonomous reasoning, the `Tool` system for extending agent capabilities, and the `PromptManager` for template management. Master these patterns first — registries, middleware, hooks, and `iter.Seq2` streaming — and the rest of the framework follows the same conventions.

2. **Capabilities** — Explore these as your needs grow. Each guide covers a major subsystem — retrieval-augmented generation, persistent memory, voice processing, or multimodal AI — that extends the foundation with domain-specific features. You can read them in any order based on what your application requires.

3. **Production** — Read these when preparing for real-world deployment. They cover orchestration patterns for coordinating multiple agents, safety pipelines for content filtering and PII protection, OpenTelemetry instrumentation for observability, and resilience patterns for fault-tolerant operation.

## Guide Categories

### [Foundations](./foundations/)

The building blocks every Beluga application uses. These guides establish the core patterns and abstractions that the rest of the framework builds on.

| Guide | What You'll Learn |
|-------|-------------------|
| [Building Your First Agent](./foundations/first-agent/) | Create a complete AI agent from scratch — wire up tools, stream responses with `iter.Seq2`, implement the ReAct reasoning loop, and hand off between specialized agents |
| [Working with LLMs](./foundations/working-with-llms/) | Configure any language model through the unified `ChatModel` interface — set up providers, compose middleware for logging and retries, attach hooks for lifecycle events, and route requests across multiple models |
| [Structured Output](./foundations/structured-output/) | Extract typed Go structs from LLM responses — generate JSON schemas automatically, validate and retry on parse failures, and build classification pipelines for routing and labeling |
| [Prompt Engineering](./foundations/prompt-engineering/) | Manage prompts as versioned, testable assets — use `PromptManager` for template resolution, `Builder` for cache-optimal token ordering, few-shot example selection, and A/B testing across prompt variants |

### [Capabilities](./capabilities/)

Domain-specific subsystems that extend the foundation. Each capability follows the same extensibility model — small interfaces, registry-based providers, and composable middleware — so patterns you learn in one transfer directly to the others.

| Guide | What You'll Learn |
|-------|-------------------|
| [RAG Pipeline](./capabilities/rag-pipeline/) | Give agents access to your data — build retrieval pipelines with embeddings, vector stores, and advanced strategies like hybrid search (BM25 + vector + RRF fusion), CRAG for self-correcting retrieval, and HyDE for hypothetical document generation |
| [Document Processing](./capabilities/document-processing/) | Prepare data for retrieval — load documents from files, URLs, and databases, split them into semantically meaningful chunks, and ingest them into vector stores for downstream search |
| [Memory System](./capabilities/memory-system/) | Give agents persistent memory across conversations — implement the MemGPT-inspired 3-tier model with Core memory (always in context), Recall memory (searchable conversation history), and Archival memory (vector-searchable long-term storage) |
| [Tools & MCP](./capabilities/tools-and-mcp/) | Extend what agents can do — create typed Go functions as tools, organize them in registries with middleware, and connect to remote MCP servers for runtime tool discovery and cross-framework interoperability |
| [Voice AI Pipeline](./capabilities/voice-ai/) | Build real-time voice applications — process audio through a frame-based pipeline with STT, TTS, and speech-to-speech models, handle voice activity detection, and stream over WebSocket or WebRTC transports |
| [Multimodal AI](./capabilities/multimodal/) | Process images, audio, and video alongside text — send mixed-content messages to multimodal models for document intelligence, visual question answering, audio transcription, and content analysis |

### [Production](./production/)

Patterns and practices for operating Beluga applications under real-world demands — coordinating agent teams, enforcing safety constraints, instrumenting for observability, and deploying with resilience.

| Guide | What You'll Learn |
|-------|-------------------|
| [Orchestration & Workflows](./production/orchestration/) | Coordinate complex agent pipelines — use Sequential, Parallel, and Loop workflow agents, build DAG execution graphs, and implement durable workflows that survive process restarts |
| [Multi-Agent Systems](./production/multi-agent-systems/) | Design systems where specialized agents collaborate — implement handoffs for agent-to-agent transfers, supervisor patterns for centralized coordination, and event-driven communication for decoupled architectures |
| [Safety & Guards](./production/safety-and-guards/) | Protect your application and users — implement the three-stage guard pipeline (input, output, tool) with PII redaction, content filtering, prompt injection detection, and human-in-the-loop approval workflows |
| [Observability](./production/observability/) | Understand what your agents are doing — instrument with OpenTelemetry using GenAI semantic conventions, collect metrics on token usage and latency, stream structured logs, and integrate health checks |
| [Deploying to Production](./production/deployment/) | Ship with confidence — serve agents as REST APIs using HTTP framework adapters (Gin, Fiber, Echo, Chi), apply circuit breakers and rate limiters for resilience, and configure container orchestration for scaling |

## Where to Go Next

- **[Tutorials](/tutorials/)** — Step-by-step walkthroughs that build complete, working applications from start to finish. Good for hands-on learning when you want to see all the pieces come together.
- **[Cookbook](/cookbook/)** — Focused recipes that solve specific problems in isolation. Use these when you know what you need to accomplish and want a concise, copy-paste-ready solution.
- **[API Reference](/api-reference/)** — Complete interface documentation for every exported type, function, and constant. The definitive reference when you need exact method signatures, option fields, or error codes.
