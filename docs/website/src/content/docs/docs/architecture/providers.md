---
title: "Provider Integration — Beluga AI"
description: "100+ pluggable providers across 16 categories for Beluga AI. LLMs, vector stores, voice, workflow engines, and more with universal registry pattern."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI providers, LLM providers Go, vector store providers, embedding providers, voice AI providers, provider registry"
---

Beluga follows a pluggable provider architecture. Every extensible package (LLM, embedding, vector store, voice, etc.) uses the same registry pattern, making it straightforward to add new integrations. This document describes the provider categories, how to discover available providers, and how to add new ones.

The provider system is designed around a core principle: application code should never depend directly on provider implementations. Instead, providers register themselves via `init()` and are consumed through abstract interfaces. This means swapping from OpenAI to Anthropic, or from pgvector to Pinecone, requires changing an import and a string — not rewriting application logic.

## Provider Categories

Each category maps to a package with a well-defined interface. Providers register via `init()` and are discovered through the registry's `List()` function. The categories are organized by the capability they provide, and each has its own independent registry — this means importing an LLM provider does not pull in vector store or voice dependencies.

### LLM Providers
**Package**: `llm/providers/`
**Interface**: `llm.ChatModel` — `Generate`, `Stream`, `BindTools`, `ModelID`
**Purpose**: Large language model inference. Supports text generation, streaming, tool calling, and structured output.
**Discovery**: `llm.List()` or `ls llm/providers/`

### Embedding Providers
**Package**: `rag/embedding/providers/`
**Interface**: `embedding.Embedder` — `Embed`, `EmbedBatch`
**Purpose**: Convert text to dense vector representations for similarity search.
**Discovery**: `embedding.List()` or `ls rag/embedding/providers/`

### Vector Store Providers
**Package**: `rag/vectorstore/providers/`
**Interface**: `vectorstore.VectorStore` — `Add`, `Search`, `Delete`
**Purpose**: Store and retrieve vectors with metadata filtering. Used by retrieval strategies.
**Discovery**: `vectorstore.List()` or `ls rag/vectorstore/providers/`

### Voice: Speech-to-Text (STT)
**Package**: `voice/stt/providers/`
**Interface**: `stt.STT` — streaming audio transcription
**Purpose**: Convert spoken audio into text with real-time streaming support.
**Discovery**: `stt.List()` or `ls voice/stt/providers/`

### Voice: Text-to-Speech (TTS)
**Package**: `voice/tts/providers/`
**Interface**: `tts.TTS` — streaming audio synthesis
**Purpose**: Convert text responses into natural-sounding speech with streaming output.
**Discovery**: `tts.List()` or `ls voice/tts/providers/`

### Voice: Speech-to-Speech (S2S)
**Package**: `voice/s2s/providers/`
**Interface**: `s2s.S2S` — bidirectional voice sessions
**Purpose**: Native audio-in/audio-out models (e.g., OpenAI Realtime, Gemini Live).
**Discovery**: `s2s.List()` or `ls voice/s2s/providers/`

### Voice: VAD
**Package**: `voice/vad/providers/`
**Interface**: VAD — voice activity detection
**Purpose**: Detect speech boundaries for turn-taking in voice pipelines.
**Discovery**: `ls voice/vad/providers/`

### Voice: Transport
**Package**: `voice/transport/providers/`
**Interface**: `transport.AudioTransport` — real-time audio transport
**Purpose**: WebRTC/WebSocket transport for audio streaming (LiveKit, Daily, etc.).
**Discovery**: `ls voice/transport/providers/`

### Memory Store Backends
**Package**: `memory/stores/`
**Interface**: `memory.MessageStore`, `memory.GraphStore`
**Purpose**: Persistent storage for conversation history and entity relationships.
**Discovery**: `ls memory/stores/`

### Document Loaders
**Package**: `rag/loader/providers/`
**Interface**: `loader.DocumentLoader` — `Load`
**Purpose**: Ingest documents from various sources (web, cloud storage, APIs).
**Discovery**: `ls rag/loader/providers/`

### Guardrails & Safety
**Package**: `guard/providers/`
**Interface**: `guard.Guard` — `Validate`
**Purpose**: External safety validation (content moderation, PII detection, prompt injection).
**Discovery**: `ls guard/providers/`

### Evaluation & Observability
**Package**: `eval/providers/`, `o11y/providers/`
**Interface**: Various — metrics export, tracing, evaluation
**Purpose**: LLM observability platforms and evaluation framework integrations.
**Discovery**: `ls eval/providers/` or `ls o11y/providers/`

### Workflow Engines
**Package**: `workflow/providers/`
**Interface**: `workflow.DurableExecutor`
**Purpose**: External durable execution engines (Temporal, NATS, Kafka, etc.).
**Discovery**: `workflow.List()` or `ls workflow/providers/`

### HTTP Framework Adapters
**Package**: `server/adapters/`
**Interface**: `server.ServerAdapter`
**Purpose**: Integrate Beluga agents into existing HTTP frameworks (Gin, Fiber, Echo, Chi, gRPC, etc.).
**Discovery**: `server.List()` or `ls server/adapters/`

### Protocol Integrations
**Package**: `protocol/mcp/providers/`, `protocol/a2a/`
**Purpose**: MCP and A2A protocol SDK integrations and registry discovery.
**Discovery**: `ls protocol/mcp/providers/`

### Infrastructure
**Package**: Various
**Purpose**: LLM gateways (LiteLLM, Bifrost) and infrastructure connectors.

---

## How to Add a Provider

Every provider follows the same pattern regardless of category. Whether you're adding an LLM, embedding model, vector store, voice transport, or workflow engine, the steps are identical: implement the interface, register via `init()`, map errors, and write tests. This uniformity is deliberate — it means that experience adding one type of provider directly transfers to any other category. See the [Full Architecture](/architecture/architecture/) for detailed templates.

### Step 1: Create the provider package

```
<parent>/providers/<name>/
├── <name>.go          # Implementation + New() + init()
├── options.go         # Provider-specific options (if needed)
├── <name>_test.go     # Unit tests
└── testdata/          # Recorded responses for tests
```

### Step 2: Implement the interface

```go
package myprovider

import (
    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/llm"
)

var _ llm.ChatModel = (*Model)(nil) // compile-time check

type Model struct { /* ... */ }

func New(cfg config.ProviderConfig) (*Model, error) {
    // Validate config, create client
    return &Model{/* ... */}, nil
}
```

### Step 3: Register via init()

```go
func init() {
    llm.Register("myprovider", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
        return New(cfg)
    })
}
```

### Step 4: Map errors to core.Error

Error mapping is critical: it translates provider-specific error types into the framework's unified error model. This enables generic retry middleware to work across all providers — it checks `IsRetryable()` on `core.Error` without knowing which provider produced the error.

```go
func (m *Model) mapError(op string, err error) error {
    code := core.ErrProviderDown
    // Map HTTP status codes or provider errors to ErrorCode
    return core.NewError(op, code, "provider error", err)
}
```

### Step 5: Test with recorded responses

```go
func TestGenerate(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        data, _ := os.ReadFile("testdata/response.json")
        w.Write(data)
    }))
    defer server.Close()

    model, err := New(config.ProviderConfig{APIKey: "test", BaseURL: server.URL})
    require.NoError(t, err)
    // ... test the provider
}
```

### Provider Checklist

- [ ] Implements the full interface
- [ ] Registers via `init()` with the parent package's `Register()` function
- [ ] Maps provider-specific errors to `core.Error` with correct `ErrorCode`
- [ ] Supports context cancellation
- [ ] Has a `New(cfg Config) (Interface, error)` constructor
- [ ] Has a compile-time interface check: `var _ Interface = (*Impl)(nil)`
- [ ] Has unit tests with mocked HTTP responses
- [ ] Includes token/usage metrics where applicable

---

## Discovering Available Providers

The registry pattern supports runtime discovery through the `List()` function and filesystem discovery through the conventional directory structure. This is useful for building admin UIs, health dashboards, and configuration validation that needs to verify which providers are available in a given deployment.

### From code

```go
import "github.com/lookatitude/beluga-ai/llm"

// List all registered LLM providers
for _, name := range llm.List() {
    fmt.Println(name)
}
```

### From the filesystem

```bash
# List LLM providers
ls llm/providers/

# List embedding providers
ls rag/embedding/providers/

# List all provider directories across the project
find . -path '*/providers/*' -type d -maxdepth 4
```

### Importing providers

Providers auto-register when imported with a blank identifier:

```go
import (
    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
    _ "github.com/lookatitude/beluga-ai/llm/providers/anthropic"
)
```

---

## Provider Summary

| Category | Package | Interface | Count |
|----------|---------|-----------|-------|
| LLM | `llm/providers/` | `ChatModel` | 23 |
| Embedding | `rag/embedding/providers/` | `Embedder` | 9 |
| Vector Store | `rag/vectorstore/providers/` | `VectorStore` | 13 |
| Voice STT | `voice/stt/providers/` | `STT` | 6 |
| Voice TTS | `voice/tts/providers/` | `TTS` | 7 |
| Voice S2S | `voice/s2s/providers/` | `S2S` | 3 |
| Voice VAD | `voice/vad/providers/` | `VAD` | 2 |
| Voice Transport | `voice/transport/providers/` | `AudioTransport` | 3 |
| Memory Stores | `memory/stores/` | `Memory` | 8 |
| Document Loaders | `rag/loader/providers/` | `DocumentLoader` | 12 |
| Guardrails | `guard/providers/` | `Guard` | 5 |
| Eval | `eval/providers/` | `Metric` | 3 |
| Observability | `o11y/providers/` | `TraceExporter` | 4 |
| Workflow | `workflow/providers/` | `DurableExecutor` | 6 |
| HTTP Adapters | `server/adapters/` | `ServerAdapter` | 7 |
| Protocol MCP | `protocol/mcp/providers/` | Various | 1 |

Use `ls <package>/providers/` or the registry `List()` function to see the current set of available providers.
