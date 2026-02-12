---
title: Integrations Overview
description: "Connect Beluga AI with 60+ services including LLM providers, vector databases, voice engines, and cloud infrastructure in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI integrations, Go AI framework integrations, LLM provider integration, vector database connector, voice AI integration, cloud infrastructure AI"
---

Beluga AI is designed to work with the systems you already use. These integration guides cover connecting the framework to LLM providers, vector databases, cloud infrastructure, observability platforms, and communication channels. Each guide includes configuration, authentication, working code examples, and troubleshooting for the specific service.

The guides range from drop-in provider registrations (one import line) to custom loaders and retrievers that you build to match your infrastructure.

## Integration Categories

| Category | Guides |
|----------|--------|
| [Agents & Tools](./agents/) | Tool registry, MCP integration, API bridges, testing |
| [LLM Providers](./llm/) | OpenAI, Anthropic, Google, Bedrock, and 18 more |
| [Embeddings](./embeddings/) | OpenAI, Cohere, Ollama, Jina, Voyage |
| [Data & Storage](./data/) | Vector stores, document loaders, S3, MongoDB, Redis, Elasticsearch |
| [Voice](./voice/) | STT, TTS, S2S, VAD, transport, session management |
| [Infrastructure](./infrastructure/) | Kubernetes, HashiCorp Vault, NATS, Auth0 |
| [Observability](./observability/) | Langfuse, LangSmith, Datadog, Phoenix, Opik |
| [Messaging](./messaging/) | Slack, Twilio, webhooks |
| [Prompts & Schema](./prompts/) | LangChain Hub, filesystem templates, JSON Schema |
| [Safety & Compliance](./safety/) | JSON reporting, ethical API filters |

## Integration Pattern

Beluga AI uses a registry-based architecture where providers auto-register via Go's `init()` mechanism. This means most integrations follow the same three steps -- import, configure, create -- regardless of the underlying service:

```go
// 1. Import the provider (auto-registers via init())
import _ "github.com/lookatitude/beluga-ai/llm/providers/openai"

// 2. Configure
cfg := config.ProviderConfig{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "gpt-4o",
}

// 3. Create and use
model, err := llm.New("openai", cfg)
```
