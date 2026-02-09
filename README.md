<p align="center">
  <img src="assets/beluga-logo.svg" width="200" alt="Beluga AI">
</p>

<h1 align="center">Beluga AI</h1>

<p align="center">
  <strong>Build production-ready AI agents in Go. Fast.</strong>
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/lookatitude/beluga-ai"><img src="https://pkg.go.dev/badge/github.com/lookatitude/beluga-ai.svg" alt="Go Reference"></a>
  <a href="https://goreportcard.com/report/github.com/lookatitude/beluga-ai"><img src="https://goreportcard.com/badge/github.com/lookatitude/beluga-ai" alt="Go Report Card"></a>
  <img src="https://img.shields.io/badge/go-%3E%3D1.23-blue" alt="Go Version">
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-green" alt="License: MIT"></a>
  <img src="https://img.shields.io/badge/tests-2%2C885_passed-brightgreen" alt="Tests">
  <img src="https://img.shields.io/badge/packages-157-blue" alt="Packages">
</p>

---

## What is Beluga AI?

Beluga AI is a Go-native framework for building agentic AI systems. It provides everything you need to create, deploy, and operate AI agents in production: from LLM orchestration and tool calling to voice pipelines, RAG, multi-agent collaboration, and durable workflows.

The framework combines production patterns from Google ADK, OpenAI Agents SDK, LangGraph, ByteDance Eino, and LiveKit into a unified, idiomatic Go framework. Streaming uses Go 1.23+ `iter.Seq2[T, error]` — no channels, no callbacks. Every component is extensible through a consistent interface + registry + middleware + hooks pattern.

Beluga ships 157 packages, 2,885 tests (all race-free), and 20+ LLM providers out of the box. It's built for teams that need enterprise-grade reliability without leaving the Go ecosystem.

## Key Features

**LLM Abstraction** — Unified `ChatModel` interface across 20+ providers (OpenAI, Anthropic, Google, Ollama, Bedrock, Groq, Mistral, DeepSeek, xAI, Cohere, Together, Fireworks, and more). Intelligent routing, structured output, context window management, and prompt cache optimization.

**Agent Framework** — Pluggable reasoning strategies via the `Planner` interface: ReAct, Reflexion, Self-Discover, Tree-of-Thought, Graph-of-Thought, LATS, and Mixture of Agents. Bring your own strategy with zero framework changes.

**Tool System** — Wrap any Go function as a tool with automatic JSON Schema generation. First-class MCP client with registry-based discovery. Parallel DAG execution for independent tool calls.

**Multi-Agent Orchestration** — Handoffs-as-tools (OpenAI pattern), supervisor delegation, scatter-gather, DAG workflows, conditional routing, and blackboard patterns. Agents collaborate through typed event streams.

**RAG Pipeline** — Hybrid search (vector + BM25 + RRF fusion) by default. Advanced retrieval strategies: CRAG, Adaptive RAG, HyDE, SEAL-RAG, and GraphRAG. 12+ vector store backends, 8+ embedding providers, contextual retrieval ingestion.

**Voice AI** — Frame-based pipeline architecture: cascading (STT → LLM → TTS), speech-to-speech (OpenAI Realtime, Gemini Live), and hybrid modes. Silero VAD, semantic turn detection. Target <800ms end-to-end latency.

**Memory** — MemGPT three-tier model: Core (always in context, self-editable), Recall (searchable conversation history), and Archival (vector + graph long-term knowledge). Hybrid store for 90% token savings.

**Protocol Interop** — MCP server and client (Streamable HTTP). A2A server and client (protobuf + gRPC). REST/SSE/WebSocket/gRPC API adapters for Gin, Fiber, Echo, Chi, and Connect-Go.

**Safety** — Three-stage guard pipeline (input → output → tool). Prompt injection detection, PII redaction, content filtering, Spotlighting for untrusted input isolation. Capability-based agent sandboxing with default-deny.

**Durable Execution** — Built-in workflow engine that separates deterministic orchestration from non-deterministic activities (LLM calls, tool invocations). Survives crashes, rate limits, and human-in-the-loop pauses.

**Resilience** — Circuit breakers, hedged requests, adaptive retry with jitter, provider-aware rate limiting (RPM, TPM, concurrent). Middleware-composable on any `ChatModel`.

**Observability** — OpenTelemetry GenAI semantic conventions baked into every boundary. Traces, metrics, and structured logging. Adapter support for Langfuse and Arize Phoenix.

**Human-in-the-Loop** — Confidence-based approval policies per tool. Configurable risk levels, auto-approve thresholds, and notification dispatch (Slack, email, webhook).

**Auth & Multi-Tenancy** — RBAC, ABAC, and capability-based security. Tenant isolation via `context.Context`. OPA integration.

**Configuration** — Generics-based `config.Load[T]()` with struct tags, environment variable overrides, validation, and hot-reload via file, Consul, etcd, or Kubernetes.

**Evaluation** — CI/CD-integrated quality gates. Built-in metrics: faithfulness, relevance, hallucination, toxicity, latency, and cost. Parallel evaluation runner.

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/lookatitude/beluga-ai/agent"
    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/tool"

    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

func main() {
    ctx := context.Background()

    // Create an LLM
    model, err := llm.New("openai", config.ProviderConfig{
        Provider: "openai",
        APIKey:   os.Getenv("OPENAI_API_KEY"),
        Model:    "gpt-4o",
    })
    if err != nil {
        panic(err)
    }

    // Define a tool
    type WeatherInput struct {
        City string `json:"city" description:"City name" required:"true"`
    }
    weatherTool := tool.NewFuncTool("get_weather", "Get current weather",
        func(ctx context.Context, input WeatherInput) (*tool.Result, error) {
            return tool.TextResult(fmt.Sprintf("72°F and sunny in %s", input.City)), nil
        },
    )

    // Create an agent
    a := agent.New("assistant",
        agent.WithLLM(model),
        agent.WithTools([]tool.Tool{weatherTool}),
        agent.WithPersona(agent.Persona{
            Role: "Helpful weather assistant",
            Goal: "Provide accurate weather information",
        }),
    )

    // Run synchronously
    result, err := a.Invoke(ctx, "What's the weather in San Francisco?")
    if err != nil {
        panic(err)
    }
    fmt.Println(result)

    // Or stream events
    for event, err := range a.Stream(ctx, "What's the weather in Tokyo?") {
        if err != nil {
            fmt.Fprintf(os.Stderr, "error: %v\n", err)
            break
        }
        switch event.Type {
        case agent.EventText:
            fmt.Print(event.Text)
        case agent.EventToolCall:
            fmt.Printf("\n[calling %s]\n", event.ToolCall.Name)
        }
    }
    fmt.Println()
}
```

## Installation

```bash
go get github.com/lookatitude/beluga-ai@latest
```

Import the providers you need:

```go
import (
    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
    _ "github.com/lookatitude/beluga-ai/llm/providers/anthropic"
    _ "github.com/lookatitude/beluga-ai/llm/providers/google"
    _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/openai"
    _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/pgvector"
)
```

Providers self-register via `init()`. Import a provider package and it becomes available through the registry:

```go
model, err := llm.New("openai", cfg)
embedder, err := embedding.New("openai", cfg)
store, err := vectorstore.New("pgvector", cfg)
```

## Architecture

Beluga is organized in seven layers. Data flows downward through typed event streams; each layer depends only on the layers below it.

```
Application Layer        → User code, CLI tools, API servers
Agent Runtime            → Persona, pluggable planners, executor, handoffs
Protocol Gateway         → MCP, A2A, REST/gRPC/WebSocket/SSE
Pipeline / Orchestration → Chain, Graph, Workflow, Supervisor, Router, Scatter-Gather
Capability Layer         → LLM, Tools, Memory, RAG, Voice, Guard
Cross-Cutting            → Resilience, Cache, Auth, HITL, Eval
Foundation               → Schema, Stream, Config, Observability, Transport
```

### Package Overview

```
beluga-ai/
├── core/             Foundation: Stream, Runnable, Lifecycle, Errors, Tenant
├── schema/           Shared types: Message, ContentPart, Tool, Document, Event
├── config/           Configuration: Load[T], Validate, hot-reload
├── o11y/             Observability: OTel GenAI conventions, slog, adapters
├── llm/              LLM: ChatModel, Router, Structured Output, Context Manager
│   └── providers/    openai, anthropic, google, ollama, bedrock, groq, mistral,
│                     deepseek, xai, cohere, together, fireworks, and more
├── tool/             Tools: Tool, FuncTool, MCP client, registry
├── memory/           Memory: Core, Recall, Archival, Graph (MemGPT model)
│   └── stores/       inmemory, redis, postgres, sqlite, neo4j, memgraph
├── rag/              RAG pipeline
│   ├── embedding/    Embedder interface + providers
│   ├── vectorstore/  VectorStore interface + 12 providers
│   ├── retriever/    Hybrid, CRAG, HyDE, Adaptive, SEAL-RAG, ensemble
│   ├── loader/       PDF, HTML, web, CSV, JSON, docx, Confluence, Notion
│   └── splitter/     Recursive, markdown text splitters
├── agent/            Agent: BaseAgent, Planner, Executor, Handoffs
│   └── workflow/     SequentialAgent, ParallelAgent, LoopAgent
├── voice/            Voice: Frame-based pipeline, STT/TTS/S2S
│   ├── stt/          Deepgram, AssemblyAI, Whisper, ElevenLabs, Groq, Gladia
│   ├── tts/          ElevenLabs, Cartesia, OpenAI, PlayHT, Groq, Fish, LMNT
│   ├── s2s/          OpenAI Realtime, Gemini Live, Amazon Nova
│   └── transport/    WebSocket, LiveKit, Daily
├── orchestration/    Chain, Graph, Router, Parallel, Supervisor
├── workflow/         Durable execution engine + Temporal, NATS providers
├── protocol/         MCP server/client, A2A server/client, REST
├── guard/            Safety: input → output → tool guard pipeline
├── resilience/       Circuit breaker, hedge, retry, rate limit
├── cache/            Exact + semantic + prompt cache
├── hitl/             Human-in-the-loop: confidence-based approval
├── auth/             RBAC, ABAC, capability-based security
├── eval/             Evaluation: faithfulness, relevance, cost metrics
├── state/            Shared agent state with Watch
├── prompt/           Prompt management and versioning
├── server/           HTTP adapters: Gin, Fiber, Echo, Chi, gRPC, Connect-Go
└── internal/         syncutil, jsonutil, testutil (mocks for all interfaces)
```

### Extension Pattern

Every package follows the same extension contract. Learn it once, apply it everywhere:

```go
// 1. Interface — small, focused (1-4 methods)
type ChatModel interface { ... }

// 2. Registry — Register() + New() + List()
llm.Register("openai", factory)
model, err := llm.New("openai", cfg)
providers := llm.List()

// 3. Middleware — func(T) T, composable
model = llm.ApplyMiddleware(model, withRetry, withCache, withLogging)

// 4. Hooks — fine-grained lifecycle callbacks
agent.ComposeHooks(auditHook, costHook, guardrailHook)
```

## Documentation

Full documentation, tutorials, and API reference are available at the [documentation site](https://lookatitude.github.io/beluga-ai/).

Key resources:
- **Getting Started** — Installation, quick start, first agent
- **Guides** — LLM providers, agents, tools, RAG, voice, memory, orchestration
- **Cookbooks** — Multi-agent customer service, RAG pipeline, voice assistant
- **Architecture** — Design decisions, extensibility patterns, data flows

## Contributing

Contributions are welcome. Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Write tests for new functionality
4. Ensure `go test ./...` passes with `-race`
5. Run `go vet ./...` and `staticcheck ./...`
6. Submit a pull request

All code must follow the conventions in [CLAUDE.md](CLAUDE.md).

## License

[MIT](LICENSE)
