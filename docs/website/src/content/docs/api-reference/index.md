---
title: API Reference
description: Complete API documentation for all Beluga AI v2 packages.
---

This section provides comprehensive API documentation for all exported types, functions, and interfaces in Beluga AI v2.

## Core Packages

### Foundation

- **[Core](./core.md)** — Typed event streams, Runnable execution interface, batch processing, context helpers, multi-tenancy, lifecycle management, and typed errors
- **[Schema](./schema.md)** — Shared message types, multimodal content parts, tool definitions, documents, events, and session types
- **[Config](./config.md)** — Configuration loading, validation, environment variable merging, and hot-reload watchers

### LLM & Agents

- **[LLM](./llm.md)** — ChatModel interface, provider registry, middleware, hooks, structured output, context management, tokenization, and routing
- **[Agent](./agent.md)** — Agent runtime, BaseAgent, Executor, Planner implementations (ReAct, Reflexion, ToT, GoT, LATS, MoA, Self-Discover), handoffs, and workflows
- **[Tool](./tool.md)** — Tool interface, FuncTool, registry, MCP client integration, and middleware

### Memory & RAG

- **[Memory](./memory.md)** — MemGPT-inspired 3-tier memory system (Core, Recall, Archival), graph memory, and composite memory
- **[RAG](./rag.md)** — Embedder interface, VectorStore interface, Retriever implementations (Vector, Hybrid, HyDE, CRAG, Multi-Query, Ensemble, Rerank, Adaptive)

### Voice & Safety

- **[Voice](./voice.md)** — Frame-based voice pipeline, VAD, STT/TTS/S2S processors, transport layer, hybrid cascade/S2S switching
- **[Guard](./guard.md)** — Three-stage safety pipeline (input, output, tool), prompt injection detection, PII redaction, content filtering, spotlighting

## Package Design Patterns

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

## Import Paths

All packages use the module path `github.com/lookatitude/beluga-ai`:

```go
import (
    "github.com/lookatitude/beluga-ai/core"
    "github.com/lookatitude/beluga-ai/schema"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/agent"
    "github.com/lookatitude/beluga-ai/tool"
    "github.com/lookatitude/beluga-ai/memory"
    "github.com/lookatitude/beluga-ai/rag/embedding"
    "github.com/lookatitude/beluga-ai/rag/vectorstore"
    "github.com/lookatitude/beluga-ai/rag/retriever"
    "github.com/lookatitude/beluga-ai/voice"
    "github.com/lookatitude/beluga-ai/guard"
    "github.com/lookatitude/beluga-ai/config"
)
```

## Provider Registration

Import providers with the blank identifier to auto-register:

```go
import (
    "github.com/lookatitude/beluga-ai/llm"
    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
    _ "github.com/lookatitude/beluga-ai/llm/providers/anthropic"
)

model, err := llm.New("openai", config.ProviderConfig{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "gpt-4o",
})
```

## Error Handling

All errors follow the typed error pattern from `core.Error`:

```go
_, err := model.Generate(ctx, msgs)
if err != nil {
    var coreErr *core.Error
    if errors.As(err, &coreErr) {
        if coreErr.Code == core.ErrRateLimit {
            // handle rate limit
        }
        if core.IsRetryable(err) {
            // retry the operation
        }
    }
}
```

## Context Propagation

Every public function accepts `context.Context` as its first parameter:

```go
ctx := context.Background()
ctx = core.WithTenant(ctx, "tenant-123")
ctx = core.WithSessionID(ctx, "session-456")
ctx = core.WithRequestID(ctx, "req-789")

result, err := agent.Invoke(ctx, input)
```

## Next Steps

- Read the [Core Package Reference](./core.md) for foundational types
- Explore [LLM Package Reference](./llm.md) for model abstraction
- Review [Agent Package Reference](./agent.md) for agentic patterns
- Check [RAG Package Reference](./rag.md) for retrieval-augmented generation
