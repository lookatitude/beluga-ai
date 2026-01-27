# Convenience Packages

The convenience packages provide simplified APIs for common tasks in the Beluga AI Framework. They reduce boilerplate and make it easier to get started with common patterns.

## Available Packages

| Package | Description | Status |
|---------|-------------|--------|
| [mock](./mock/) | Centralized mock implementations for testing | Stable |
| [context](./context/) | RAG context building utilities | Stable |
| [provider](./provider/) | Unified provider discovery | Stable |
| [agent](./agent/) | Simplified agent creation | WIP |
| [rag](./rag/) | RAG pipeline builder | WIP |
| [voiceagent](./voiceagent/) | Voice agent builder | WIP |

## Quick Start

### Mock Package - Testing Utilities

```go
import "github.com/lookatitude/beluga-ai/pkg/convenience/mock"

// Create mock LLM for testing
llm := mock.NewLLM(mock.WithResponse("test response"))
result, _ := llm.Invoke(ctx, "input")

// Create mock embedder
embedder := mock.NewEmbedder(mock.WithDimension(1536))
embeddings, _ := embedder.EmbedDocuments(ctx, texts)
```

### Context Package - RAG Context Building

```go
import "github.com/lookatitude/beluga-ai/pkg/convenience/context"

// Build RAG context
ctx := context.NewBuilder().
    AddDocuments(docs, scores).
    AddHistory(messages).
    WithSystemPrompt("You are helpful").
    SortByScore().
    BuildForQuestion("What is AI?")
```

### Provider Package - Discovery

```go
import "github.com/lookatitude/beluga-ai/pkg/convenience/provider"

// List available providers
llmProviders := provider.ListLLMs()
sttProviders := provider.ListSTTs()
ttsProviders := provider.ListTTSs()

// Get all providers
allProviders := provider.GetAllProviders()
```

### Agent Package (WIP)

```go
import "github.com/lookatitude/beluga-ai/pkg/convenience/agent"

// Configure agent builder
builder := agent.NewBuilder().
    WithSystemPrompt("You are helpful").
    WithName("assistant").
    WithMaxTurns(10)
```

### RAG Package (WIP)

```go
import "github.com/lookatitude/beluga-ai/pkg/convenience/rag"

// Configure RAG pipeline builder
builder := rag.NewBuilder().
    WithDocumentSource("./docs", "md", "txt").
    WithTopK(10).
    WithChunkSize(500)
```

### Voice Agent Package (WIP)

```go
import "github.com/lookatitude/beluga-ai/pkg/convenience/voiceagent"

// Configure voice agent builder
builder := voiceagent.NewBuilder().
    WithSTT("deepgram").
    WithTTS("elevenlabs").
    WithVAD("silero").
    WithLLM("openai").
    WithMemory(true)
```

## Package Status

### Stable Packages

These packages are production-ready:

- **mock**: Complete mock implementations for LLM, Embedder, STT, TTS, and Tool
- **context**: Full-featured RAG context builder with templates
- **provider**: Provider discovery and listing utilities

### Work in Progress

These packages provide builder patterns but don't yet create full instances:

- **agent**: Agent builder pattern (use `pkg/agents` for production)
- **rag**: RAG pipeline builder (use individual packages for production)
- **voiceagent**: Voice agent builder (use individual packages for production)

## Design Principles

1. **Simplicity**: Reduce boilerplate for common tasks
2. **Fluent API**: Chain methods for readable configuration
3. **Composability**: Work with existing framework packages
4. **Progressive Complexity**: Start simple, customize as needed

## Production Usage

For production applications, consider using the full packages directly:

```go
// Full packages offer more control and features
import (
    "github.com/lookatitude/beluga-ai/pkg/agents"
    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/embeddings"
    "github.com/lookatitude/beluga-ai/pkg/vectorstores"
    "github.com/lookatitude/beluga-ai/pkg/stt"
    "github.com/lookatitude/beluga-ai/pkg/tts"
)
```
