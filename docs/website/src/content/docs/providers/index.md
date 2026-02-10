---
title: Providers
description: "108 provider integrations across 15 categories: LLMs, embeddings, vector stores, voice, document loaders, and more."
---

Beluga AI v2 ships with 108 provider implementations across 15 categories. Every provider follows the same registry pattern: import the package, and it auto-registers via `init()`. Switch between providers by changing an import path and configuration — no code changes required.

## Provider Categories

| Category | Providers | Interface | Description |
|----------|-----------|-----------|-------------|
| [LLM](./llm/) | 22 | `llm.ChatModel` | OpenAI, Anthropic, Google, Ollama, Bedrock, Groq, and 16 more |
| [Embedding](./embedding/) | 9 | `embedding.Embedder` | OpenAI, Cohere, Google, Jina, Mistral, Ollama, Voyage, Sentence Transformers |
| [Vector Store](./vectorstore/) | 13 | `vectorstore.VectorStore` | pgvector, Pinecone, Qdrant, Weaviate, Milvus, Elasticsearch, Redis, and more |
| [Voice (STT/TTS)](./voice/) | 14 | `stt.STT` / `tts.TTS` | Deepgram, ElevenLabs, AssemblyAI, Cartesia, Whisper, Groq, and more |
| [Document Loader](./loader/) | 8 | `loader.DocumentLoader` | Cloud storage, Confluence, Firecrawl, Google Drive, GitHub, Notion |
| [Guard](./guard/) | 5 | `guard.Guard` | Azure Safety, Guardrails AI, Lakera, LLM Guard, NeMo |
| [Evaluation](./eval/) | 3 | `eval.Metric` | Braintrust, DeepEval, RAGAS |
| [Observability](./observability/) | 4 | `o11y.TracerProvider` | Langfuse, LangSmith, Opik, Phoenix |
| [Workflow](./workflow/) | 6 | `workflow.Engine` | Dapr, In-memory, Inngest, Kafka, NATS, Temporal |
| [Transport](./transport/) | 3 | `transport.Transport` | Daily, LiveKit, Pipecat |
| [VAD](./vad/) | 2 | `vad.Detector` | Silero, WebRTC |
| [Cache](./cache/) | 1 | `cache.Store` | In-memory |
| [State](./state/) | 1 | `state.Store` | In-memory |
| [Prompt](./prompt/) | 1 | `prompt.Loader` | File-based |
| [MCP](./mcp/) | 1 | MCP integration | Composio |

## Registry Pattern

Every provider category uses the same pattern:

```go
import (
    "github.com/lookatitude/beluga-ai/llm"
    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"     // auto-registers
    _ "github.com/lookatitude/beluga-ai/llm/providers/anthropic"  // auto-registers
)

// Create by name
model, err := llm.New("openai", cfg)

// Discover available providers
names := llm.List() // ["anthropic", "openai", ...]
```

## Adding Custom Providers

Any provider category can be extended by implementing the interface and calling `Register()` in `init()`:

```go
func init() {
    llm.Register("my-provider", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
        return &myModel{apiKey: cfg.APIKey}, nil
    })
}
```

See the [API Reference](/api-reference/) for complete interface definitions, or browse individual provider pages for configuration details and usage examples.
