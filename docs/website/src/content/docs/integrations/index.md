---
title: Integrations
description: "60 integration guides for connecting Beluga AI v2 with external services, platforms, and infrastructure."
---

Detailed integration guides showing how to connect Beluga AI with external services. Each guide covers configuration, authentication, usage patterns, and troubleshooting.

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

Most integrations follow the same three steps:

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
