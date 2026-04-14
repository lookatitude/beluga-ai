# Docs Migration Map — Beluga AI v2 Site Redesign

**Prepared:** 2026-04-12
**Scope:** `docs/website/src/content/docs/docs/` → 6-section IA
**Not in scope:** Marketing pages, component design, token changes.

---

## 1. Target IA Summary

```
docs/
├── start/
│   ├── index.md                          (overview — what Beluga is, 5-min path)
│   ├── installation.md
│   └── quick-start.md
│
├── concepts/
│   ├── index.md
│   ├── architecture.md                   (7-layer model)
│   ├── core-primitives.md                (Stream, Event, Runnable)
│   ├── extensibility.md                  (interface / registry / hooks / middleware)
│   ├── streaming.md                      (iter.Seq2, producers, consumers)
│   ├── errors.md                         (core.Error, ErrorCode, IsRetryable)
│   └── context.md                        (context.Context conventions)
│
├── guides/
│   ├── index.md
│   ├── foundations/
│   │   ├── index.md
│   │   ├── working-with-llms.md
│   │   ├── prompt-engineering.md
│   │   ├── streaming.md
│   │   ├── errors.md
│   │   ├── custom-runnable.md
│   │   ├── custom-message-types.md
│   │   ├── middleware-implementation.md
│   │   ├── multiturn-conversations.md
│   │   └── environment-secrets.md
│   ├── capabilities/
│   │   ├── index.md
│   │   ├── agents/
│   │   │   ├── first-agent.md
│   │   │   ├── research-agent.md
│   │   │   ├── multi-agent-orchestration.md
│   │   │   ├── multi-agent-support.md
│   │   │   ├── multi-provider.md
│   │   │   ├── model-switching.md
│   │   │   ├── tools-registry.md
│   │   │   ├── dynamic-tool-injection.md
│   │   │   ├── conversational-ai.md
│   │   │   ├── autonomous-support.md
│   │   │   ├── few-shot-sql.md
│   │   │   ├── code-review-system.md
│   │   │   └── memory-ide-extension.md
│   │   ├── rag/
│   │   │   ├── rag-pipeline.md
│   │   │   ├── hybrid-search.md
│   │   │   ├── multiquery-chains.md
│   │   │   ├── enterprise-rag.md
│   │   │   ├── knowledge-qa.md
│   │   │   ├── rag-strategies.md
│   │   │   ├── rag-large-repos.md
│   │   │   ├── semantic-search.md
│   │   │   ├── recommendation-engine.md
│   │   │   ├── cross-lingual-retrieval.md
│   │   │   ├── regulatory-search.md
│   │   │   ├── scientific-papers.md
│   │   │   ├── search-everything.md
│   │   │   ├── cloud-doc-sync.md
│   │   │   └── semantic-image-search.md
│   │   ├── memory/
│   │   │   ├── memory-system.md
│   │   │   ├── redis-persistence.md
│   │   │   └── summary-window.md
│   │   ├── voice/
│   │   │   ├── voice-ai.md
│   │   │   ├── scalable-backend.md
│   │   │   ├── preemptive-generation.md
│   │   │   ├── s2s-amazon-nova.md
│   │   │   ├── s2s-reasoning-modes.md
│   │   │   ├── stt-realtime-streaming.md
│   │   │   ├── sentence-boundary.md
│   │   │   ├── session-interruptions.md
│   │   │   ├── sensitivity-tuning.md
│   │   │   ├── ml-turn-prediction.md
│   │   │   ├── custom-silero-vad.md
│   │   │   ├── livekit-vapi.md
│   │   │   ├── elevenlabs-cloning.md
│   │   │   ├── whisper-finetuning.md
│   │   │   ├── ssml-tuning.md
│   │   │   ├── hotel-concierge.md
│   │   │   ├── bilingual-tutor.md
│   │   │   ├── elearning-voiceovers.md
│   │   │   ├── industrial-control.md
│   │   │   ├── interactive-audiobooks.md
│   │   │   ├── low-latency-prediction.md
│   │   │   ├── meeting-minutes.md
│   │   │   ├── multi-speaker-segmentation.md
│   │   │   ├── noise-resistant-vad.md
│   │   │   ├── outbound-calling.md
│   │   │   ├── barge-in-detection.md
│   │   │   ├── voice-applications.md
│   │   │   ├── voice-forms.md
│   │   │   ├── voice-ivr.md
│   │   │   └── voice-sessions-overview.md
│   │   ├── orchestration/
│   │   │   ├── dag-workflows.md
│   │   │   ├── message-bus.md
│   │   │   └── temporal-workflows.md
│   │   ├── documents/
│   │   │   ├── document-ingestion.md
│   │   │   ├── lazy-loading.md
│   │   │   ├── markdown-chunking.md
│   │   │   ├── pdf-scraper.md
│   │   │   ├── semantic-splitting.md
│   │   │   ├── automated-code-generation.md
│   │   │   ├── batch-processing.md
│   │   │   ├── invoice-processor.md
│   │   │   ├── legacy-archive.md
│   │   │   ├── multi-doc-summarizer.md
│   │   │   └── multi-stage-etl.md
│   │   ├── multimodal/
│   │   │   ├── audio-analysis.md
│   │   │   ├── visual-reasoning.md
│   │   │   ├── audio-visual-search.md
│   │   │   └── security-camera.md
│   │   └── messaging/
│   │       ├── omnichannel-gateway.md
│   │       ├── whatsapp-bot.md
│   │       ├── multi-channel-hub.md
│   │       ├── patient-history.md
│   │       ├── sms-reminders.md
│   │       └── support-gateway.md
│   └── production/
│       ├── index.md
│       ├── observability.md
│       ├── safety-and-guards.md
│       ├── resilience.md
│       ├── workflow-durability.md
│       ├── health-checks.md
│       ├── otel-tracing.md
│       ├── prometheus-grafana.md
│       ├── human-in-loop.md
│       ├── mcp-tools.md
│       ├── rest-deployment.md
│       ├── content-moderation.md
│       ├── cost-optimized-router.md
│       ├── model-ab-testing.md
│       ├── model-benchmarking.md
│       ├── monitoring-dashboards.md
│       ├── real-time-analysis.md
│       ├── token-cost-attribution.md
│       ├── dynamic-feature-flags.md
│       ├── children-stories-safety.md
│       ├── devsecops-auditor.md
│       ├── financial-compliance.md
│       ├── legal-entity-extraction.md
│       ├── medical-records.md
│       ├── pii-leakage-detection.md
│       ├── error-recovery-service.md
│       ├── llm-gateway.md
│       ├── multi-tenant-keys.md
│       ├── production-platform.md
│       ├── streaming-proxy.md
│       ├── workflow-orchestration.md
│       └── integrations/
│           ├── auth0-jwt.md
│           ├── vault-connector.md
│           ├── viper-environment.md
│           ├── kubernetes-helm.md
│           ├── kubernetes-scheduler.md
│           ├── nats-message-bus.md
│           ├── context-deep-dive.md
│           ├── anthropic-enterprise.md
│           ├── bedrock-integration.md
│           ├── pixtral-mistral.md
│           ├── vertex-ai-vision.md
│           ├── openai-assistants-bridge.md
│           ├── mock-ui-testing.md
│           ├── agents-mcp-integration.md
│           ├── agents-tools-registry.md
│           ├── datadog-dashboards.md
│           ├── langsmith-debugging.md
│           ├── zap-logrus.md
│           ├── document-loaders.md
│           ├── elasticsearch-search.md
│           ├── google-drive-scraper.md
│           ├── mongodb-persistence.md
│           ├── pinecone-serverless.md
│           ├── qdrant-cloud.md
│           ├── redis-locking.md
│           ├── s3-event-loader.md
│           ├── weaviate-rag.md
│           ├── cohere-multilingual.md
│           ├── ollama-local-embeddings.md
│           ├── spacy-tokenizer.md
│           ├── tiktoken-bpe.md
│           ├── azure-speech.md
│           ├── deepgram-streams.md
│           ├── elevenlabs-streaming.md
│           ├── livekit-webhooks.md
│           ├── noisy-turn-detection.md
│           ├── nova-bedrock-streaming.md
│           ├── onnx-edge-vad.md
│           ├── openai-realtime.md
│           ├── session-persistence.md
│           ├── session-routing.md
│           ├── transcribe-websockets.md
│           ├── turn-heuristic-tuning.md
│           ├── vapi-custom-tools.md
│           ├── webrtc-browser-vad.md
│           ├── ethical-api-filter.md
│           ├── safety-json-reporting.md
│           ├── filesystem-templates.md
│           ├── go-struct-bridge.md
│           ├── json-schema-validation.md
│           ├── langchain-hub.md
│           ├── slack-webhooks.md
│           └── twilio-conversations.md
│
├── recipes/
│   ├── index.md
│   ├── agents/
│   │   ├── agents-parallel-execution.md
│   │   ├── agents-tool-failures.md
│   │   ├── custom-agent-patterns.md
│   │   └── tool-recipes.md
│   ├── llm/
│   │   ├── global-retry.md
│   │   ├── history-trimming.md
│   │   ├── llm-error-handling.md
│   │   ├── reasoning-models.md
│   │   ├── streaming-metadata.md
│   │   ├── streaming-tool-calls.md
│   │   └── token-counting.md
│   ├── rag/
│   │   ├── code-splitting.md
│   │   ├── corrupt-doc-handling.md
│   │   ├── meta-filtering.md
│   │   └── sentence-splitting.md
│   ├── memory/
│   │   ├── conversation-expiry.md
│   │   ├── memory-context-recovery.md
│   │   └── memory-ttl-cleanup.md
│   ├── voice/
│   │   ├── background-noise.md
│   │   ├── glass-to-glass-latency.md
│   │   ├── jitter-buffer.md
│   │   ├── long-utterances.md
│   │   ├── ml-barge-in.md
│   │   ├── multi-speaker-synthesis.md
│   │   ├── s2s-voice-metrics.md
│   │   ├── sentence-boundary-turns.md
│   │   ├── speech-interruption.md
│   │   ├── ssml-tuning.md
│   │   ├── vad-sensitivity.md
│   │   ├── voice-backends.md
│   │   ├── voice-preemptive-gen.md
│   │   └── voice-stream-scaling.md
│   ├── multimodal/
│   │   ├── capability-fallbacks.md
│   │   ├── inbound-media.md
│   │   └── multiple-images.md
│   ├── prompts/
│   │   ├── dynamic-templates.md
│   │   └── partial-substitution.md
│   └── infrastructure/
│       ├── prompt-injection-detection.md
│       ├── rate-limiting.md
│       ├── request-id-correlation.md
│       └── workflow-checkpoints.md
│
├── reference/
│   ├── index.md
│   ├── providers/                        (current docs/providers/ tree — unchanged)
│   ├── api/                              (current docs/api-reference/ tree — unchanged)
│   ├── architecture/
│   │   ├── overview.md
│   │   ├── concepts.md
│   │   ├── packages.md
│   │   └── providers.md
│   └── config-schema.md                  (GAP — does not exist yet)
│
└── contributing/
    ├── index.md
    ├── development-setup.md
    ├── code-style.md
    ├── testing.md
    ├── pull-requests.md
    ├── releases.md
    └── project-reports/
        ├── changelog.md
        ├── security.md
        └── code-quality.md
```

---

## 2. File-by-File Migration Table

All paths are relative to `docs/website/src/content/docs/docs/`.
Target paths are relative to the same root under the new IA.

### Getting Started → Start

| Current path | Current section | Target section | Target path | Action | Notes |
|---|---|---|---|---|---|
| `getting-started/overview.md` | Getting Started | Start | `start/index.md` | rename | Becomes the Start landing page |
| `getting-started/installation.md` | Getting Started | Start | `start/installation.md` | move | |
| `getting-started/quick-start.md` | Getting Started | Start | `start/quick-start.md` | move | |

### Architecture → Concepts + Reference

| Current path | Current section | Target section | Target path | Action | Notes |
|---|---|---|---|---|---|
| `architecture/index.md` | Architecture | Reference | `reference/architecture/overview.md` | move | Explanation-heavy arch overview belongs in Reference |
| `architecture/concepts.md` | Architecture | Concepts | `concepts/index.md` | move | Design principles / "what and why" — pure Concepts material |
| `architecture/packages.md` | Architecture | Reference | `reference/architecture/packages.md` | move | Package dependency map — reference table |
| `architecture/architecture.md` | Architecture | Reference | `reference/architecture/overview.md` | merge → delete | Duplicate of `architecture/index.md`; merge content then delete |
| `architecture/providers.md` | Architecture | Reference | `reference/architecture/providers.md` | move | Provider integration model |

### Current Guides → Guides (Foundations)

| Current path | Current section | Target section | Target path | Action | Notes |
|---|---|---|---|---|---|
| `guides/index.md` | Guides | Guides | `guides/index.md` | move | Update internal links |
| `guides/foundations/index.md` | Guides | Guides/Foundations | `guides/foundations/index.md` | move | |
| `guides/foundations/prompt-engineering.md` | Guides | Guides/Foundations | `guides/foundations/prompt-engineering.md` | move | |
| `guides/capabilities/index.md` | Guides | Guides/Capabilities | `guides/capabilities/index.md` | move | |
| `guides/production/index.md` | Guides | Guides/Production | `guides/production/index.md` | move | |
| `guides/production/safety-and-guards.md` | Guides | Guides/Production | `guides/production/safety-and-guards.md` | move | |
| `guides/production/observability.md` | Guides | Guides/Production | `guides/production/observability.md` | move | |

### Tutorials/Foundation → Guides/Foundations

| Current path | Current section | Target section | Target path | Action | Notes |
|---|---|---|---|---|---|
| `tutorials/foundation/custom-message-types.md` | Tutorials | Guides/Foundations | `guides/foundations/custom-message-types.md` | move | |
| `tutorials/foundation/custom-runnable.md` | Tutorials | Guides/Foundations | `guides/foundations/custom-runnable.md` | move | |
| `tutorials/foundation/environment-secrets.md` | Tutorials | Guides/Foundations | `guides/foundations/environment-secrets.md` | move | |
| `tutorials/foundation/health-checks.md` | Tutorials | Guides/Foundations | `guides/foundations/health-checks.md` | move | |
| `tutorials/foundation/middleware-implementation.md` | Tutorials | Guides/Foundations | `guides/foundations/middleware-implementation.md` | move | |
| `tutorials/foundation/multiturn-conversations.md` | Tutorials | Guides/Foundations | `guides/foundations/multiturn-conversations.md` | move | |
| `tutorials/foundation/otel-tracing.md` | Tutorials | Guides/Production | `guides/production/otel-tracing.md` | move | OTel is a production concern, not a foundations API tutorial |
| `tutorials/foundation/prometheus-grafana.md` | Tutorials | Guides/Production | `guides/production/prometheus-grafana.md` | move | |
| `tutorials/foundation/vault-integration.md` | Tutorials | Guides/Production/integrations | `guides/production/integrations/vault-connector.md` | merge → delete | Same topic as `integrations/infrastructure/vault-connector.md`; merge, keep integration version |

### Tutorials/Agents → Guides/Capabilities/agents

| Current path | Current section | Target section | Target path | Action | Notes |
|---|---|---|---|---|---|
| `tutorials/agents/model-switching.md` | Tutorials | Guides/Capabilities/agents | `guides/capabilities/agents/model-switching.md` | move | |
| `tutorials/agents/multi-agent-orchestration.md` | Tutorials | Guides/Capabilities/agents | `guides/capabilities/agents/multi-agent-orchestration.md` | move | |
| `tutorials/agents/multi-provider.md` | Tutorials | Guides/Capabilities/agents | `guides/capabilities/agents/multi-provider.md` | move | |
| `tutorials/agents/research-agent.md` | Tutorials | Guides/Capabilities/agents | `guides/capabilities/agents/research-agent.md` | move | |
| `tutorials/agents/tools-registry.md` | Tutorials | Guides/Capabilities/agents | `guides/capabilities/agents/tools-registry.md` | move | |

### Tutorials/RAG → Guides/Capabilities/rag

| Current path | Current section | Target section | Target path | Action | Notes |
|---|---|---|---|---|---|
| `tutorials/rag/hybrid-search.md` | Tutorials | Guides/Capabilities/rag | `guides/capabilities/rag/hybrid-search.md` | move | |
| `tutorials/rag/multiquery-chains.md` | Tutorials | Guides/Capabilities/rag | `guides/capabilities/rag/multiquery-chains.md` | move | |

### Tutorials/Memory → Guides/Capabilities/memory

| Current path | Current section | Target section | Target path | Action | Notes |
|---|---|---|---|---|---|
| `tutorials/memory/redis-persistence.md` | Tutorials | Guides/Capabilities/memory | `guides/capabilities/memory/redis-persistence.md` | move | |
| `tutorials/memory/summary-window.md` | Tutorials | Guides/Capabilities/memory | `guides/capabilities/memory/summary-window.md` | move | |

### Tutorials/Voice → Guides/Capabilities/voice

| Current path | Current section | Target section | Target path | Action | Notes |
|---|---|---|---|---|---|
| `tutorials/voice/custom-silero-vad.md` | Tutorials | Guides/Capabilities/voice | `guides/capabilities/voice/custom-silero-vad.md` | move | |
| `tutorials/voice/elevenlabs-cloning.md` | Tutorials | Guides/Capabilities/voice | `guides/capabilities/voice/elevenlabs-cloning.md` | move | |
| `tutorials/voice/livekit-vapi.md` | Tutorials | Guides/Capabilities/voice | `guides/capabilities/voice/livekit-vapi.md` | move | |
| `tutorials/voice/ml-turn-prediction.md` | Tutorials | Guides/Capabilities/voice | `guides/capabilities/voice/ml-turn-prediction.md` | move | |
| `tutorials/voice/preemptive-generation.md` | Tutorials | Guides/Capabilities/voice | `guides/capabilities/voice/preemptive-generation.md` | move | |
| `tutorials/voice/s2s-amazon-nova.md` | Tutorials | Guides/Capabilities/voice | `guides/capabilities/voice/s2s-amazon-nova.md` | move | |
| `tutorials/voice/s2s-reasoning-modes.md` | Tutorials | Guides/Capabilities/voice | `guides/capabilities/voice/s2s-reasoning-modes.md` | move | |
| `tutorials/voice/scalable-backend.md` | Tutorials | Guides/Capabilities/voice | `guides/capabilities/voice/scalable-backend.md` | move | |
| `tutorials/voice/sensitivity-tuning.md` | Tutorials | Guides/Capabilities/voice | `guides/capabilities/voice/sensitivity-tuning.md` | move | |
| `tutorials/voice/sentence-boundary.md` | Tutorials | Guides/Capabilities/voice | `guides/capabilities/voice/sentence-boundary.md` | move | |
| `tutorials/voice/session-interruptions.md` | Tutorials | Guides/Capabilities/voice | `guides/capabilities/voice/session-interruptions.md` | move | |
| `tutorials/voice/ssml-tuning.md` | Tutorials | Guides/Capabilities/voice | `guides/capabilities/voice/ssml-tuning.md` | move | Duplicate slug warning — see Section 3 |
| `tutorials/voice/stt-realtime-streaming.md` | Tutorials | Guides/Capabilities/voice | `guides/capabilities/voice/stt-realtime-streaming.md` | move | |
| `tutorials/voice/whisper-finetuning.md` | Tutorials | Guides/Capabilities/voice | `guides/capabilities/voice/whisper-finetuning.md` | move | |

### Tutorials/Orchestration → Guides/Capabilities/orchestration

| Current path | Current section | Target section | Target path | Action | Notes |
|---|---|---|---|---|---|
| `tutorials/orchestration/dag-workflows.md` | Tutorials | Guides/Capabilities/orchestration | `guides/capabilities/orchestration/dag-workflows.md` | move | |
| `tutorials/orchestration/message-bus.md` | Tutorials | Guides/Capabilities/orchestration | `guides/capabilities/orchestration/message-bus.md` | move | |
| `tutorials/orchestration/temporal-workflows.md` | Tutorials | Guides/Capabilities/orchestration | `guides/capabilities/orchestration/temporal-workflows.md` | move | |

### Tutorials/Safety → Guides/Production

| Current path | Current section | Target section | Target path | Action | Notes |
|---|---|---|---|---|---|
| `tutorials/safety/content-moderation.md` | Tutorials | Guides/Production | `guides/production/content-moderation.md` | move | |
| `tutorials/safety/human-in-loop.md` | Tutorials | Guides/Production | `guides/production/human-in-loop.md` | move | |

### Tutorials/Server → Guides/Production

| Current path | Current section | Target section | Target path | Action | Notes |
|---|---|---|---|---|---|
| `tutorials/server/mcp-tools.md` | Tutorials | Guides/Production | `guides/production/mcp-tools.md` | move | |
| `tutorials/server/rest-deployment.md` | Tutorials | Guides/Production | `guides/production/rest-deployment.md` | move | |

### Tutorials/Documents → Guides/Capabilities/documents

| Current path | Current section | Target section | Target path | Action | Notes |
|---|---|---|---|---|---|
| `tutorials/documents/lazy-loading.md` | Tutorials | Guides/Capabilities/documents | `guides/capabilities/documents/lazy-loading.md` | move | |
| `tutorials/documents/markdown-chunking.md` | Tutorials | Guides/Capabilities/documents | `guides/capabilities/documents/markdown-chunking.md` | move | |
| `tutorials/documents/pdf-scraper.md` | Tutorials | Guides/Capabilities/documents | `guides/capabilities/documents/pdf-scraper.md` | move | |
| `tutorials/documents/semantic-splitting.md` | Tutorials | Guides/Capabilities/documents | `guides/capabilities/documents/semantic-splitting.md` | move | |

### Tutorials/Messaging → Guides/Capabilities/messaging

| Current path | Current section | Target section | Target path | Action | Notes |
|---|---|---|---|---|---|
| `tutorials/messaging/omnichannel-gateway.md` | Tutorials | Guides/Capabilities/messaging | `guides/capabilities/messaging/omnichannel-gateway.md` | move | |
| `tutorials/messaging/whatsapp-bot.md` | Tutorials | Guides/Capabilities/messaging | `guides/capabilities/messaging/whatsapp-bot.md` | move | |

### Tutorials/Multimodal → Guides/Capabilities/multimodal

| Current path | Current section | Target section | Target path | Action | Notes |
|---|---|---|---|---|---|
| `tutorials/multimodal/audio-analysis.md` | Tutorials | Guides/Capabilities/multimodal | `guides/capabilities/multimodal/audio-analysis.md` | move | |
| `tutorials/multimodal/visual-reasoning.md` | Tutorials | Guides/Capabilities/multimodal | `guides/capabilities/multimodal/visual-reasoning.md` | move | |

### Tutorials/Providers → Guides (split)

| Current path | Current section | Target section | Target path | Action | Notes |
|---|---|---|---|---|---|
| `tutorials/providers/advanced-inference.md` | Tutorials | Guides/Foundations | `guides/foundations/advanced-inference.md` | move | LLM inference technique — foundations |
| `tutorials/providers/finetuning-embeddings.md` | Tutorials | Guides/Capabilities/rag | `guides/capabilities/rag/finetuning-embeddings.md` | move | Embedding topic |
| `tutorials/providers/inmemory-vectorstore.md` | Tutorials | Guides/Capabilities/rag | `guides/capabilities/rag/inmemory-vectorstore.md` | move | |
| `tutorials/providers/message-templates.md` | Tutorials | Guides/Foundations | `guides/foundations/message-templates.md` | move | Schema/prompt foundations |
| `tutorials/providers/multimodal-embeddings.md` | Tutorials | Guides/Capabilities/multimodal | `guides/capabilities/multimodal/multimodal-embeddings.md` | move | |
| `tutorials/providers/new-llm-provider.md` | Tutorials | Guides/Foundations | `guides/foundations/new-llm-provider.md` | move | Custom provider implementation — extension developer path |
| `tutorials/providers/pgvector-sharding.md` | Tutorials | Guides/Capabilities/rag | `guides/capabilities/rag/pgvector-sharding.md` | move | |
| `tutorials/providers/reusable-system-prompts.md` | Tutorials | Guides/Foundations | `guides/foundations/reusable-system-prompts.md` | move | |

### Use Cases/Search → Guides/Capabilities/rag

| Current path | Current section | Target section | Target path | Action | Notes |
|---|---|---|---|---|---|
| `use-cases/search/cloud-doc-sync.md` | Use Cases | Guides/Capabilities/rag | `guides/capabilities/rag/cloud-doc-sync.md` | move | |
| `use-cases/search/cross-lingual-retrieval.md` | Use Cases | Guides/Capabilities/rag | `guides/capabilities/rag/cross-lingual-retrieval.md` | move | |
| `use-cases/search/enterprise-rag.md` | Use Cases | Guides/Capabilities/rag | `guides/capabilities/rag/enterprise-rag.md` | move | |
| `use-cases/search/knowledge-qa.md` | Use Cases | Guides/Capabilities/rag | `guides/capabilities/rag/knowledge-qa.md` | move | |
| `use-cases/search/rag-large-repos.md` | Use Cases | Guides/Capabilities/rag | `guides/capabilities/rag/rag-large-repos.md` | move | |
| `use-cases/search/rag-strategies.md` | Use Cases | Guides/Capabilities/rag | `guides/capabilities/rag/rag-strategies.md` | move | |
| `use-cases/search/recommendation-engine.md` | Use Cases | Guides/Capabilities/rag | `guides/capabilities/rag/recommendation-engine.md` | move | |
| `use-cases/search/regulatory-search.md` | Use Cases | Guides/Capabilities/rag | `guides/capabilities/rag/regulatory-search.md` | move | |
| `use-cases/search/scientific-papers.md` | Use Cases | Guides/Capabilities/rag | `guides/capabilities/rag/scientific-papers.md` | move | |
| `use-cases/search/search-everything.md` | Use Cases | Guides/Capabilities/rag | `guides/capabilities/rag/search-everything.md` | move | |
| `use-cases/search/semantic-image-search.md` | Use Cases | Guides/Capabilities/multimodal | `guides/capabilities/multimodal/semantic-image-search.md` | move | |
| `use-cases/search/semantic-search.md` | Use Cases | Guides/Capabilities/rag | `guides/capabilities/rag/semantic-search.md` | move | |
| `use-cases/search/index.md` | Use Cases | delete | — | delete | Section-level index, replaced by sub-category headings |

### Use Cases/Agents → Guides/Capabilities/agents

| Current path | Current section | Target section | Target path | Action | Notes |
|---|---|---|---|---|---|
| `use-cases/agents/autonomous-support.md` | Use Cases | Guides/Capabilities/agents | `guides/capabilities/agents/autonomous-support.md` | move | |
| `use-cases/agents/code-review-system.md` | Use Cases | Guides/Capabilities/agents | `guides/capabilities/agents/code-review-system.md` | move | |
| `use-cases/agents/conversational-ai.md` | Use Cases | Guides/Capabilities/agents | `guides/capabilities/agents/conversational-ai.md` | move | |
| `use-cases/agents/dynamic-tool-injection.md` | Use Cases | Guides/Capabilities/agents | `guides/capabilities/agents/dynamic-tool-injection.md` | move | |
| `use-cases/agents/few-shot-sql.md` | Use Cases | Guides/Capabilities/agents | `guides/capabilities/agents/few-shot-sql.md` | move | |
| `use-cases/agents/memory-ide-extension.md` | Use Cases | Guides/Capabilities/agents | `guides/capabilities/agents/memory-ide-extension.md` | move | |
| `use-cases/agents/multi-agent-support.md` | Use Cases | Guides/Capabilities/agents | `guides/capabilities/agents/multi-agent-support.md` | move | |
| `use-cases/agents/index.md` | Use Cases | delete | — | delete | Section-level index |

### Use Cases/Voice → Guides/Capabilities/voice

| Current path | Current section | Target section | Target path | Action | Notes |
|---|---|---|---|---|---|
| `use-cases/voice/barge-in-detection.md` | Use Cases | Guides/Capabilities/voice | `guides/capabilities/voice/barge-in-detection.md` | move | |
| `use-cases/voice/bilingual-tutor.md` | Use Cases | Guides/Capabilities/voice | `guides/capabilities/voice/bilingual-tutor.md` | move | |
| `use-cases/voice/elearning-voiceovers.md` | Use Cases | Guides/Capabilities/voice | `guides/capabilities/voice/elearning-voiceovers.md` | move | |
| `use-cases/voice/hotel-concierge.md` | Use Cases | Guides/Capabilities/voice | `guides/capabilities/voice/hotel-concierge.md` | move | |
| `use-cases/voice/industrial-control.md` | Use Cases | Guides/Capabilities/voice | `guides/capabilities/voice/industrial-control.md` | move | |
| `use-cases/voice/interactive-audiobooks.md` | Use Cases | Guides/Capabilities/voice | `guides/capabilities/voice/interactive-audiobooks.md` | move | |
| `use-cases/voice/low-latency-prediction.md` | Use Cases | Guides/Capabilities/voice | `guides/capabilities/voice/low-latency-prediction.md` | move | |
| `use-cases/voice/meeting-minutes.md` | Use Cases | Guides/Capabilities/voice | `guides/capabilities/voice/meeting-minutes.md` | move | |
| `use-cases/voice/multi-speaker-segmentation.md` | Use Cases | Guides/Capabilities/voice | `guides/capabilities/voice/multi-speaker-segmentation.md` | move | |
| `use-cases/voice/noise-resistant-vad.md` | Use Cases | Guides/Capabilities/voice | `guides/capabilities/voice/noise-resistant-vad.md` | move | |
| `use-cases/voice/outbound-calling.md` | Use Cases | Guides/Capabilities/voice | `guides/capabilities/voice/outbound-calling.md` | move | |
| `use-cases/voice/voice-applications.md` | Use Cases | Guides/Capabilities/voice | `guides/capabilities/voice/voice-applications.md` | move | |
| `use-cases/voice/voice-forms.md` | Use Cases | Guides/Capabilities/voice | `guides/capabilities/voice/voice-forms.md` | move | |
| `use-cases/voice/voice-ivr.md` | Use Cases | Guides/Capabilities/voice | `guides/capabilities/voice/voice-ivr.md` | move | |
| `use-cases/voice/voice-sessions-overview.md` | Use Cases | Guides/Capabilities/voice | `guides/capabilities/voice/voice-sessions-overview.md` | move | |
| `use-cases/voice/index.md` | Use Cases | delete | — | delete | Section-level index |

### Use Cases/Documents → Guides/Capabilities/documents

| Current path | Current section | Target section | Target path | Action | Notes |
|---|---|---|---|---|---|
| `use-cases/documents/automated-code-generation.md` | Use Cases | Guides/Capabilities/documents | `guides/capabilities/documents/automated-code-generation.md` | move | |
| `use-cases/documents/batch-processing.md` | Use Cases | Guides/Capabilities/documents | `guides/capabilities/documents/batch-processing.md` | move | |
| `use-cases/documents/document-processing.md` | Use Cases | Guides/Capabilities/documents | `guides/capabilities/documents/document-processing.md` | move | |
| `use-cases/documents/invoice-processor.md` | Use Cases | Guides/Capabilities/documents | `guides/capabilities/documents/invoice-processor.md` | move | |
| `use-cases/documents/legacy-archive.md` | Use Cases | Guides/Capabilities/documents | `guides/capabilities/documents/legacy-archive.md` | move | |
| `use-cases/documents/multi-doc-summarizer.md` | Use Cases | Guides/Capabilities/documents | `guides/capabilities/documents/multi-doc-summarizer.md` | move | |
| `use-cases/documents/multi-stage-etl.md` | Use Cases | Guides/Capabilities/documents | `guides/capabilities/documents/multi-stage-etl.md` | move | |
| `use-cases/documents/index.md` | Use Cases | delete | — | delete | Section-level index |

### Use Cases/Analytics → Guides/Production

| Current path | Current section | Target section | Target path | Action | Notes |
|---|---|---|---|---|---|
| `use-cases/analytics/audio-visual-search.md` | Use Cases | Guides/Capabilities/multimodal | `guides/capabilities/multimodal/audio-visual-search.md` | move | Multimodal topic, not analytics |
| `use-cases/analytics/cost-optimized-router.md` | Use Cases | Guides/Production | `guides/production/cost-optimized-router.md` | move | |
| `use-cases/analytics/dynamic-feature-flags.md` | Use Cases | Guides/Production | `guides/production/dynamic-feature-flags.md` | move | |
| `use-cases/analytics/model-ab-testing.md` | Use Cases | Guides/Production | `guides/production/model-ab-testing.md` | move | |
| `use-cases/analytics/model-benchmarking.md` | Use Cases | Guides/Production | `guides/production/model-benchmarking.md` | move | |
| `use-cases/analytics/monitoring-dashboards.md` | Use Cases | Guides/Production | `guides/production/monitoring-dashboards.md` | move | |
| `use-cases/analytics/real-time-analysis.md` | Use Cases | Guides/Production | `guides/production/real-time-analysis.md` | move | |
| `use-cases/analytics/security-camera.md` | Use Cases | Guides/Capabilities/multimodal | `guides/capabilities/multimodal/security-camera.md` | move | Vision use case |
| `use-cases/analytics/token-cost-attribution.md` | Use Cases | Guides/Production | `guides/production/token-cost-attribution.md` | move | |
| `use-cases/analytics/index.md` | Use Cases | delete | — | delete | Section-level index |

### Use Cases/Safety → Guides/Production

| Current path | Current section | Target section | Target path | Action | Notes |
|---|---|---|---|---|---|
| `use-cases/safety/children-stories-safety.md` | Use Cases | Guides/Production | `guides/production/children-stories-safety.md` | move | |
| `use-cases/safety/devsecops-auditor.md` | Use Cases | Guides/Production | `guides/production/devsecops-auditor.md` | move | |
| `use-cases/safety/financial-compliance.md` | Use Cases | Guides/Production | `guides/production/financial-compliance.md` | move | |
| `use-cases/safety/legal-entity-extraction.md` | Use Cases | Guides/Production | `guides/production/legal-entity-extraction.md` | move | |
| `use-cases/safety/medical-records.md` | Use Cases | Guides/Production | `guides/production/medical-records.md` | move | |
| `use-cases/safety/pii-leakage-detection.md` | Use Cases | Guides/Production | `guides/production/pii-leakage-detection.md` | move | |
| `use-cases/safety/index.md` | Use Cases | delete | — | delete | Section-level index |

### Use Cases/Infrastructure → Guides/Production

| Current path | Current section | Target section | Target path | Action | Notes |
|---|---|---|---|---|---|
| `use-cases/infrastructure/error-recovery-service.md` | Use Cases | Guides/Production | `guides/production/error-recovery-service.md` | move | |
| `use-cases/infrastructure/llm-gateway.md` | Use Cases | Guides/Production | `guides/production/llm-gateway.md` | move | |
| `use-cases/infrastructure/multi-tenant-keys.md` | Use Cases | Guides/Production | `guides/production/multi-tenant-keys.md` | move | |
| `use-cases/infrastructure/production-platform.md` | Use Cases | Guides/Production | `guides/production/production-platform.md` | move | |
| `use-cases/infrastructure/streaming-proxy.md` | Use Cases | Guides/Production | `guides/production/streaming-proxy.md` | move | |
| `use-cases/infrastructure/workflow-orchestration.md` | Use Cases | Guides/Production | `guides/production/workflow-orchestration.md` | move | |
| `use-cases/infrastructure/index.md` | Use Cases | delete | — | delete | Section-level index |

### Use Cases/Messaging → Guides/Capabilities/messaging

| Current path | Current section | Target section | Target path | Action | Notes |
|---|---|---|---|---|---|
| `use-cases/messaging/multi-channel-hub.md` | Use Cases | Guides/Capabilities/messaging | `guides/capabilities/messaging/multi-channel-hub.md` | move | |
| `use-cases/messaging/patient-history.md` | Use Cases | Guides/Capabilities/messaging | `guides/capabilities/messaging/patient-history.md` | move | |
| `use-cases/messaging/sms-reminders.md` | Use Cases | Guides/Capabilities/messaging | `guides/capabilities/messaging/sms-reminders.md` | move | |
| `use-cases/messaging/support-gateway.md` | Use Cases | Guides/Capabilities/messaging | `guides/capabilities/messaging/support-gateway.md` | move | |
| `use-cases/messaging/index.md` | Use Cases | delete | — | delete | Section-level index |

### Use Cases top-level index

| Current path | Current section | Target section | Target path | Action | Notes |
|---|---|---|---|---|---|
| `use-cases/index.md` | Use Cases | delete | — | delete | Entire section dissolved |

### Integrations → Guides/Production/integrations

| Current path | Current section | Target section | Target path | Action | Notes |
|---|---|---|---|---|---|
| `integrations/index.md` | Integrations | delete | — | delete | Section dissolved |
| `integrations/infrastructure/index.md` | Integrations | delete | — | delete | Section index |
| `integrations/infrastructure/auth0-jwt.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/auth0-jwt.md` | move | |
| `integrations/infrastructure/vault-connector.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/vault-connector.md` | move | Absorbs tutorials/foundation/vault-integration.md |
| `integrations/infrastructure/viper-environment.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/viper-environment.md` | move | |
| `integrations/infrastructure/kubernetes-helm.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/kubernetes-helm.md` | move | |
| `integrations/infrastructure/kubernetes-scheduler.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/kubernetes-scheduler.md` | move | |
| `integrations/infrastructure/nats-message-bus.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/nats-message-bus.md` | move | |
| `integrations/infrastructure/context-deep-dive.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/context-deep-dive.md` | move | |
| `integrations/llm/index.md` | Integrations | delete | — | delete | |
| `integrations/llm/anthropic-enterprise.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/anthropic-enterprise.md` | move | |
| `integrations/llm/bedrock-integration.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/bedrock-integration.md` | move | |
| `integrations/llm/pixtral-mistral.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/pixtral-mistral.md` | move | |
| `integrations/llm/vertex-ai-vision.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/vertex-ai-vision.md` | move | |
| `integrations/agents/index.md` | Integrations | delete | — | delete | |
| `integrations/agents/openai-assistants-bridge.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/openai-assistants-bridge.md` | move | |
| `integrations/agents/mock-ui-testing.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/mock-ui-testing.md` | move | |
| `integrations/agents/agents-mcp-integration.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/agents-mcp-integration.md` | move | |
| `integrations/agents/agents-tools-registry.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/agents-tools-registry.md` | move | |
| `integrations/observability/index.md` | Integrations | delete | — | delete | |
| `integrations/observability/datadog-dashboards.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/datadog-dashboards.md` | move | |
| `integrations/observability/langsmith-debugging.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/langsmith-debugging.md` | move | |
| `integrations/observability/zap-logrus.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/zap-logrus.md` | move | |
| `integrations/data/index.md` | Integrations | delete | — | delete | |
| `integrations/data/document-loaders.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/document-loaders.md` | move | |
| `integrations/data/elasticsearch-search.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/elasticsearch-search.md` | move | |
| `integrations/data/google-drive-scraper.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/google-drive-scraper.md` | move | |
| `integrations/data/mongodb-persistence.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/mongodb-persistence.md` | move | |
| `integrations/data/pinecone-serverless.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/pinecone-serverless.md` | move | |
| `integrations/data/qdrant-cloud.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/qdrant-cloud.md` | move | |
| `integrations/data/redis-locking.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/redis-locking.md` | move | |
| `integrations/data/s3-event-loader.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/s3-event-loader.md` | move | |
| `integrations/data/weaviate-rag.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/weaviate-rag.md` | move | |
| `integrations/embeddings/index.md` | Integrations | delete | — | delete | |
| `integrations/embeddings/cohere-multilingual.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/cohere-multilingual.md` | move | |
| `integrations/embeddings/ollama-local-embeddings.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/ollama-local-embeddings.md` | move | |
| `integrations/embeddings/spacy-tokenizer.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/spacy-tokenizer.md` | move | |
| `integrations/embeddings/tiktoken-bpe.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/tiktoken-bpe.md` | move | |
| `integrations/voice/index.md` | Integrations | delete | — | delete | |
| `integrations/voice/azure-speech.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/azure-speech.md` | move | |
| `integrations/voice/deepgram-streams.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/deepgram-streams.md` | move | |
| `integrations/voice/elevenlabs-streaming.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/elevenlabs-streaming.md` | move | |
| `integrations/voice/livekit-webhooks.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/livekit-webhooks.md` | move | |
| `integrations/voice/noisy-turn-detection.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/noisy-turn-detection.md` | move | |
| `integrations/voice/nova-bedrock-streaming.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/nova-bedrock-streaming.md` | move | |
| `integrations/voice/onnx-edge-vad.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/onnx-edge-vad.md` | move | |
| `integrations/voice/openai-realtime.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/openai-realtime.md` | move | |
| `integrations/voice/session-persistence.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/session-persistence.md` | move | |
| `integrations/voice/session-routing.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/session-routing.md` | move | |
| `integrations/voice/transcribe-websockets.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/transcribe-websockets.md` | move | |
| `integrations/voice/turn-heuristic-tuning.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/turn-heuristic-tuning.md` | move | |
| `integrations/voice/vapi-custom-tools.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/vapi-custom-tools.md` | move | |
| `integrations/voice/webrtc-browser-vad.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/webrtc-browser-vad.md` | move | |
| `integrations/safety/index.md` | Integrations | delete | — | delete | |
| `integrations/safety/ethical-api-filter.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/ethical-api-filter.md` | move | |
| `integrations/safety/safety-json-reporting.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/safety-json-reporting.md` | move | |
| `integrations/prompts/index.md` | Integrations | delete | — | delete | |
| `integrations/prompts/filesystem-templates.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/filesystem-templates.md` | move | |
| `integrations/prompts/go-struct-bridge.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/go-struct-bridge.md` | move | |
| `integrations/prompts/json-schema-validation.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/json-schema-validation.md` | move | |
| `integrations/prompts/langchain-hub.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/langchain-hub.md` | move | |
| `integrations/messaging/index.md` | Integrations | delete | — | delete | |
| `integrations/messaging/slack-webhooks.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/slack-webhooks.md` | move | |
| `integrations/messaging/twilio-conversations.md` | Integrations | Guides/Production/integrations | `guides/production/integrations/twilio-conversations.md` | move | |

### Cookbook → Recipes

All cookbook files are short, self-contained snippets — direct rename of section.

| Current path | Current section | Target section | Target path | Action | Notes |
|---|---|---|---|---|---|
| `cookbook/index.md` | Cookbook | Recipes | `recipes/index.md` | rename | Update title to "Recipes" |
| `cookbook/agents/index.md` | Cookbook | delete | — | delete | Sub-index, not needed |
| `cookbook/agents/agents-parallel-execution.md` | Cookbook | Recipes/agents | `recipes/agents/agents-parallel-execution.md` | move | |
| `cookbook/agents/agents-tool-failures.md` | Cookbook | Recipes/agents | `recipes/agents/agents-tool-failures.md` | move | |
| `cookbook/agents/custom-agent-patterns.md` | Cookbook | Recipes/agents | `recipes/agents/custom-agent-patterns.md` | move | |
| `cookbook/agents/tool-recipes.md` | Cookbook | Recipes/agents | `recipes/agents/tool-recipes.md` | move | |
| `cookbook/llm/index.md` | Cookbook | delete | — | delete | |
| `cookbook/llm/global-retry.md` | Cookbook | Recipes/llm | `recipes/llm/global-retry.md` | move | |
| `cookbook/llm/history-trimming.md` | Cookbook | Recipes/llm | `recipes/llm/history-trimming.md` | move | |
| `cookbook/llm/llm-error-handling.md` | Cookbook | Recipes/llm | `recipes/llm/llm-error-handling.md` | move | |
| `cookbook/llm/reasoning-models.md` | Cookbook | Recipes/llm | `recipes/llm/reasoning-models.md` | move | |
| `cookbook/llm/streaming-metadata.md` | Cookbook | Recipes/llm | `recipes/llm/streaming-metadata.md` | move | |
| `cookbook/llm/streaming-tool-calls.md` | Cookbook | Recipes/llm | `recipes/llm/streaming-tool-calls.md` | move | |
| `cookbook/llm/streaming-tool-calls.md` | Cookbook | Recipes/llm | `recipes/llm/token-counting.md` | move | |
| `cookbook/rag/code-splitting.md` | Cookbook | Recipes/rag | `recipes/rag/code-splitting.md` | move | |
| `cookbook/rag/corrupt-doc-handling.md` | Cookbook | Recipes/rag | `recipes/rag/corrupt-doc-handling.md` | move | |
| `cookbook/rag/meta-filtering.md` | Cookbook | Recipes/rag | `recipes/rag/meta-filtering.md` | move | |
| `cookbook/rag/sentence-splitting.md` | Cookbook | Recipes/rag | `recipes/rag/sentence-splitting.md` | move | |
| `cookbook/memory/conversation-expiry.md` | Cookbook | Recipes/memory | `recipes/memory/conversation-expiry.md` | move | |
| `cookbook/memory/memory-context-recovery.md` | Cookbook | Recipes/memory | `recipes/memory/memory-context-recovery.md` | move | |
| `cookbook/memory/memory-ttl-cleanup.md` | Cookbook | Recipes/memory | `recipes/memory/memory-ttl-cleanup.md` | move | |
| `cookbook/voice/index.md` | Cookbook | delete | — | delete | |
| `cookbook/voice/background-noise.md` | Cookbook | Recipes/voice | `recipes/voice/background-noise.md` | move | |
| `cookbook/voice/glass-to-glass-latency.md` | Cookbook | Recipes/voice | `recipes/voice/glass-to-glass-latency.md` | move | |
| `cookbook/voice/jitter-buffer.md` | Cookbook | Recipes/voice | `recipes/voice/jitter-buffer.md` | move | |
| `cookbook/voice/long-utterances.md` | Cookbook | Recipes/voice | `recipes/voice/long-utterances.md` | move | |
| `cookbook/voice/ml-barge-in.md` | Cookbook | Recipes/voice | `recipes/voice/ml-barge-in.md` | move | |
| `cookbook/voice/multi-speaker-synthesis.md` | Cookbook | Recipes/voice | `recipes/voice/multi-speaker-synthesis.md` | move | |
| `cookbook/voice/s2s-voice-metrics.md` | Cookbook | Recipes/voice | `recipes/voice/s2s-voice-metrics.md` | move | |
| `cookbook/voice/sentence-boundary-turns.md` | Cookbook | Recipes/voice | `recipes/voice/sentence-boundary-turns.md` | move | |
| `cookbook/voice/speech-interruption.md` | Cookbook | Recipes/voice | `recipes/voice/speech-interruption.md` | move | |
| `cookbook/voice/ssml-tuning.md` | Cookbook | Recipes/voice | `recipes/voice/ssml-tuning.md` | move | Duplicate slug with tutorials/voice/ssml-tuning — see Section 3 |
| `cookbook/voice/vad-sensitivity.md` | Cookbook | Recipes/voice | `recipes/voice/vad-sensitivity.md` | move | |
| `cookbook/voice/voice-backends.md` | Cookbook | Recipes/voice | `recipes/voice/voice-backends.md` | move | |
| `cookbook/voice/voice-preemptive-gen.md` | Cookbook | Recipes/voice | `recipes/voice/voice-preemptive-gen.md` | move | |
| `cookbook/voice/voice-stream-scaling.md` | Cookbook | Recipes/voice | `recipes/voice/voice-stream-scaling.md` | move | |
| `cookbook/multimodal/capability-fallbacks.md` | Cookbook | Recipes/multimodal | `recipes/multimodal/capability-fallbacks.md` | move | |
| `cookbook/multimodal/inbound-media.md` | Cookbook | Recipes/multimodal | `recipes/multimodal/inbound-media.md` | move | |
| `cookbook/multimodal/multiple-images.md` | Cookbook | Recipes/multimodal | `recipes/multimodal/multiple-images.md` | move | |
| `cookbook/prompts/dynamic-templates.md` | Cookbook | Recipes/prompts | `recipes/prompts/dynamic-templates.md` | move | |
| `cookbook/prompts/partial-substitution.md` | Cookbook | Recipes/prompts | `recipes/prompts/partial-substitution.md` | move | |
| `cookbook/infrastructure/prompt-injection-detection.md` | Cookbook | Recipes/infrastructure | `recipes/infrastructure/prompt-injection-detection.md` | move | |
| `cookbook/infrastructure/rate-limiting.md` | Cookbook | Recipes/infrastructure | `recipes/infrastructure/rate-limiting.md` | move | |
| `cookbook/infrastructure/request-id-correlation.md` | Cookbook | Recipes/infrastructure | `recipes/infrastructure/request-id-correlation.md` | move | |
| `cookbook/infrastructure/workflow-checkpoints.md` | Cookbook | Recipes/infrastructure | `recipes/infrastructure/workflow-checkpoints.md` | move | |

### Providers → Reference/providers

All 110 provider pages move as a block — no structural change, only parent path update.

| Current path | Current section | Target section | Target path | Action | Notes |
|---|---|---|---|---|---|
| `providers/index.md` | Providers | Reference | `reference/providers/index.md` | move | |
| `providers/llm/**` | Providers | Reference | `reference/providers/llm/**` | move (block) | 23 files, no changes |
| `providers/embedding/**` | Providers | Reference | `reference/providers/embedding/**` | move (block) | 9 files |
| `providers/vectorstore/**` | Providers | Reference | `reference/providers/vectorstore/**` | move (block) | 13 files |
| `providers/voice/**` | Providers | Reference | `reference/providers/voice/**` | move (block) | 14 files |
| `providers/loader/**` | Providers | Reference | `reference/providers/loader/**` | move (block) | 8 files |
| `providers/guard/**` | Providers | Reference | `reference/providers/guard/**` | move (block) | 6 files |
| `providers/eval/**` | Providers | Reference | `reference/providers/eval/**` | move (block) | 4 files |
| `providers/observability/**` | Providers | Reference | `reference/providers/observability/**` | move (block) | 5 files |
| `providers/workflow/**` | Providers | Reference | `reference/providers/workflow/**` | move (block) | 7 files |
| `providers/vad/**` | Providers | Reference | `reference/providers/vad/**` | move (block) | 3 files |
| `providers/transport/**` | Providers | Reference | `reference/providers/transport/**` | move (block) | 4 files |
| `providers/cache/**` | Providers | Reference | `reference/providers/cache/**` | move (block) | 2 files |
| `providers/state/**` | Providers | Reference | `reference/providers/state/**` | move (block) | 2 files |
| `providers/prompt/**` | Providers | Reference | `reference/providers/prompt/**` | move (block) | 2 files |
| `providers/mcp/**` | Providers | Reference | `reference/providers/mcp/**` | move (block) | 2 files |

### API Reference → Reference/api

All 29 api-reference pages move as a block.

| Current path | Current section | Target section | Target path | Action | Notes |
|---|---|---|---|---|---|
| `api-reference/**` | API Reference | Reference | `reference/api/**` | move (block) | 29 files, no structural change |

### Reports → Contributing/project-reports

| Current path | Current section | Target section | Target path | Action | Notes |
|---|---|---|---|---|---|
| `reports/index.md` | Reports | delete | — | delete | Absorbed into Contributing landing |
| `reports/changelog.md` | Reports | Contributing | `contributing/project-reports/changelog.md` | move | |
| `reports/security.md` | Reports | Contributing | `contributing/project-reports/security.md` | move | |
| `reports/code-quality.md` | Reports | Contributing | `contributing/project-reports/code-quality.md` | move | |

### Contributing → Contributing (unchanged structure)

| Current path | Current section | Target section | Target path | Action | Notes |
|---|---|---|---|---|---|
| `contributing/index.md` | Contributing | Contributing | `contributing/index.md` | move | Update to add Project Reports subsection link |
| `contributing/development-setup.md` | Contributing | Contributing | `contributing/development-setup.md` | move | |
| `contributing/code-style.md` | Contributing | Contributing | `contributing/code-style.md` | move | |
| `contributing/testing.md` | Contributing | Contributing | `contributing/testing.md` | move | |
| `contributing/pull-requests.md` | Contributing | Contributing | `contributing/pull-requests.md` | move | |
| `contributing/releases.md` | Contributing | Contributing | `contributing/releases.md` | move | |

---

## 3. Merge Proposals

| Source files | Target file | Justification |
|---|---|---|
| `architecture/architecture.md` + `architecture/index.md` | `reference/architecture/overview.md` | Both cover the same 7-layer architecture narrative; `index.md` is the more complete version |
| `tutorials/foundation/vault-integration.md` + `integrations/infrastructure/vault-connector.md` | `guides/production/integrations/vault-connector.md` | Same topic (Vault secrets with Beluga); integration version is more complete, tutorial is a step-subset |
| `tutorials/voice/ssml-tuning.md` + `cookbook/voice/ssml-tuning.md` | `guides/capabilities/voice/ssml-tuning.md` | Identical slug, likely duplicate content; the tutorial (step-by-step) is the authoritative version; the recipe snippet should be extracted as a separate named recipe if distinct |
| `use-cases/search/semantic-search.md` + `tutorials/agents/multi-provider.md` (if multi-provider covers semantic search routing) | — | Verify before merging — flag for content review pass |

---

## 4. Delete Proposals

| File | Reason |
|---|---|
| `architecture/architecture.md` | Duplicate of `architecture/index.md`; merge content into `reference/architecture/overview.md` then delete |
| `use-cases/index.md` | Section dissolved; no standalone content |
| `use-cases/search/index.md` | Section-level index only |
| `use-cases/agents/index.md` | Section-level index only |
| `use-cases/voice/index.md` | Section-level index only |
| `use-cases/documents/index.md` | Section-level index only |
| `use-cases/analytics/index.md` | Section-level index only |
| `use-cases/safety/index.md` | Section-level index only |
| `use-cases/infrastructure/index.md` | Section-level index only |
| `use-cases/messaging/index.md` | Section-level index only |
| `integrations/index.md` | Section dissolved |
| `integrations/agents/index.md` | Section index only |
| `integrations/data/index.md` | Section index only |
| `integrations/embeddings/index.md` | Section index only |
| `integrations/infrastructure/index.md` | Section index only |
| `integrations/llm/index.md` | Section index only |
| `integrations/messaging/index.md` | Section index only |
| `integrations/observability/index.md` | Section index only |
| `integrations/prompts/index.md` | Section index only |
| `integrations/safety/index.md` | Section index only |
| `integrations/voice/index.md` | Section index only |
| `reports/index.md` | Absorbed into Contributing landing |
| `cookbook/agents/index.md` | Sub-index, not needed in Recipes |
| `cookbook/llm/index.md` | Sub-index |
| `cookbook/voice/index.md` | Sub-index |
| `tutorials/foundation/vault-integration.md` | Merged into `guides/production/integrations/vault-connector.md` |

Total delete candidates: **25 files**

---

## 5. Gap Analysis

Pages the new IA needs that do not currently exist in the tree:

| Target path | Description | Priority | Notes |
|---|---|---|---|
| `concepts/index.md` | Concepts section landing — what this section covers and when to read it | P0 | Blocks launch; `architecture/concepts.md` becomes the body but needs a landing |
| `concepts/core-primitives.md` | `Stream[T]`, `Event[T]`, `Runnable` explanation with code | P0 | `architecture/index.md` touches this but no dedicated Concepts page exists |
| `concepts/extensibility.md` | Interface / registry / hooks / middleware — the four rings explained conceptually | P0 | `architecture/concepts.md` partially covers this; a clean Concepts-section version is needed |
| `concepts/streaming.md` | `iter.Seq2[T, error]` explained — why not channels, consumer pattern | P0 | No dedicated page; scattered across tutorials |
| `concepts/errors.md` | `core.Error`, `ErrorCode`, `IsRetryable()` — the error model | P1 | Referenced everywhere, never explained as a concept |
| `concepts/context.md` | `context.Context` conventions — cancellation, tenant ID, auth, tracing | P1 | `integrations/infrastructure/context-deep-dive.md` covers this for a specific integration; a canonical Concepts page is missing |
| `start/index.md` | Start section landing (5-minute path overview, links to install + quickstart) | P0 | `getting-started/overview.md` becomes this but may need content updates |
| `guides/capabilities/agents/first-agent.md` | The canonical "build your first agent" guide (5-min) | P0 | Not present in website content tree; referenced in `docs/README.md` as `docs/guides/first-agent.md` but absent from website |
| `guides/capabilities/memory/memory-system.md` | How to configure 3-tier memory (working / recall / archival) | P1 | No current how-to for the memory system as a whole |
| `guides/capabilities/documents/document-ingestion.md` | End-to-end document ingestion guide | P1 | Fragmented across loader provider pages |
| `reference/index.md` | Reference section landing | P0 | |
| `reference/config-schema.md` | Config schema reference — all `WithX()` options across packages | P1 | No such page exists anywhere |
| `recipes/index.md` | Recipes section landing (rename of `cookbook/index.md`) | P0 | Exists as `cookbook/index.md`; just rename |
| `guides/production/resilience.md` | How to apply circuit breakers, retry, rate limits in production | P0 | `api-reference/infrastructure/resilience.md` is a ref page; a guide does not exist |
| `guides/production/workflow-durability.md` | Durable workflows with Temporal / Dapr / NATS — production guide | P1 | `tutorials/orchestration/temporal-workflows.md` covers one provider; a unified guide is missing |

---

## 6. New sidebar.json (Draft)

```json
{
  "main": [
    {
      "label": "Start",
      "items": [
        { "label": "Overview", "slug": "docs/start" },
        { "label": "Installation", "slug": "docs/start/installation" },
        { "label": "Quick Start", "slug": "docs/start/quick-start" }
      ]
    },
    {
      "label": "Concepts",
      "collapsed": true,
      "items": [
        { "label": "Overview", "slug": "docs/concepts" },
        { "label": "Architecture", "slug": "docs/concepts/architecture" },
        { "label": "Core Primitives", "slug": "docs/concepts/core-primitives" },
        { "label": "Extensibility", "slug": "docs/concepts/extensibility" },
        { "label": "Streaming", "slug": "docs/concepts/streaming" },
        { "label": "Errors", "slug": "docs/concepts/errors" },
        { "label": "Context", "slug": "docs/concepts/context" }
      ]
    },
    {
      "label": "Guides",
      "items": [
        { "label": "Overview", "slug": "docs/guides" },
        {
          "label": "Foundations",
          "collapsed": true,
          "autogenerate": { "directory": "docs/guides/foundations" }
        },
        {
          "label": "Capabilities",
          "collapsed": true,
          "items": [
            { "label": "Overview", "slug": "docs/guides/capabilities" },
            {
              "label": "Agents",
              "collapsed": true,
              "autogenerate": { "directory": "docs/guides/capabilities/agents" }
            },
            {
              "label": "RAG",
              "collapsed": true,
              "autogenerate": { "directory": "docs/guides/capabilities/rag" }
            },
            {
              "label": "Memory",
              "collapsed": true,
              "autogenerate": { "directory": "docs/guides/capabilities/memory" }
            },
            {
              "label": "Voice",
              "collapsed": true,
              "autogenerate": { "directory": "docs/guides/capabilities/voice" }
            },
            {
              "label": "Orchestration",
              "collapsed": true,
              "autogenerate": { "directory": "docs/guides/capabilities/orchestration" }
            },
            {
              "label": "Documents",
              "collapsed": true,
              "autogenerate": { "directory": "docs/guides/capabilities/documents" }
            },
            {
              "label": "Multimodal",
              "collapsed": true,
              "autogenerate": { "directory": "docs/guides/capabilities/multimodal" }
            },
            {
              "label": "Messaging",
              "collapsed": true,
              "autogenerate": { "directory": "docs/guides/capabilities/messaging" }
            }
          ]
        },
        {
          "label": "Production",
          "collapsed": true,
          "items": [
            { "label": "Overview", "slug": "docs/guides/production" },
            {
              "label": "Observability",
              "slug": "docs/guides/production/observability"
            },
            {
              "label": "Safety and Guards",
              "slug": "docs/guides/production/safety-and-guards"
            },
            {
              "label": "Resilience",
              "slug": "docs/guides/production/resilience"
            },
            {
              "label": "Workflow Durability",
              "slug": "docs/guides/production/workflow-durability"
            },
            {
              "label": "More",
              "collapsed": true,
              "autogenerate": { "directory": "docs/guides/production" }
            },
            {
              "label": "Integrations",
              "collapsed": true,
              "autogenerate": { "directory": "docs/guides/production/integrations" }
            }
          ]
        }
      ]
    },
    {
      "label": "Recipes",
      "collapsed": true,
      "items": [
        { "label": "Overview", "slug": "docs/recipes" },
        {
          "label": "Agents",
          "collapsed": true,
          "autogenerate": { "directory": "docs/recipes/agents" }
        },
        {
          "label": "LLM",
          "collapsed": true,
          "autogenerate": { "directory": "docs/recipes/llm" }
        },
        {
          "label": "RAG",
          "collapsed": true,
          "autogenerate": { "directory": "docs/recipes/rag" }
        },
        {
          "label": "Memory",
          "collapsed": true,
          "autogenerate": { "directory": "docs/recipes/memory" }
        },
        {
          "label": "Voice",
          "collapsed": true,
          "autogenerate": { "directory": "docs/recipes/voice" }
        },
        {
          "label": "Multimodal",
          "collapsed": true,
          "autogenerate": { "directory": "docs/recipes/multimodal" }
        },
        {
          "label": "Prompts",
          "collapsed": true,
          "autogenerate": { "directory": "docs/recipes/prompts" }
        },
        {
          "label": "Infrastructure",
          "collapsed": true,
          "autogenerate": { "directory": "docs/recipes/infrastructure" }
        }
      ]
    },
    {
      "label": "Reference",
      "collapsed": true,
      "items": [
        { "label": "Overview", "slug": "docs/reference" },
        {
          "label": "Providers",
          "collapsed": true,
          "items": [
            { "label": "Overview", "slug": "docs/reference/providers" },
            {
              "label": "LLM",
              "collapsed": true,
              "autogenerate": { "directory": "docs/reference/providers/llm" }
            },
            {
              "label": "Embedding",
              "collapsed": true,
              "autogenerate": { "directory": "docs/reference/providers/embedding" }
            },
            {
              "label": "Vector Stores",
              "collapsed": true,
              "autogenerate": { "directory": "docs/reference/providers/vectorstore" }
            },
            {
              "label": "Voice",
              "collapsed": true,
              "autogenerate": { "directory": "docs/reference/providers/voice" }
            },
            {
              "label": "Document Loaders",
              "collapsed": true,
              "autogenerate": { "directory": "docs/reference/providers/loader" }
            },
            {
              "label": "Guard",
              "collapsed": true,
              "autogenerate": { "directory": "docs/reference/providers/guard" }
            },
            {
              "label": "Eval",
              "collapsed": true,
              "autogenerate": { "directory": "docs/reference/providers/eval" }
            },
            {
              "label": "Observability",
              "collapsed": true,
              "autogenerate": { "directory": "docs/reference/providers/observability" }
            },
            {
              "label": "Workflow",
              "collapsed": true,
              "autogenerate": { "directory": "docs/reference/providers/workflow" }
            },
            {
              "label": "VAD",
              "collapsed": true,
              "autogenerate": { "directory": "docs/reference/providers/vad" }
            },
            {
              "label": "Transport",
              "collapsed": true,
              "autogenerate": { "directory": "docs/reference/providers/transport" }
            },
            {
              "label": "Cache",
              "collapsed": true,
              "autogenerate": { "directory": "docs/reference/providers/cache" }
            },
            {
              "label": "State",
              "collapsed": true,
              "autogenerate": { "directory": "docs/reference/providers/state" }
            },
            {
              "label": "Prompt",
              "collapsed": true,
              "autogenerate": { "directory": "docs/reference/providers/prompt" }
            },
            {
              "label": "MCP",
              "collapsed": true,
              "autogenerate": { "directory": "docs/reference/providers/mcp" }
            }
          ]
        },
        {
          "label": "API",
          "collapsed": true,
          "items": [
            { "label": "Overview", "slug": "docs/reference/api" },
            {
              "label": "Foundation",
              "collapsed": true,
              "autogenerate": { "directory": "docs/reference/api/foundation" }
            },
            {
              "label": "LLM and Agents",
              "collapsed": true,
              "autogenerate": { "directory": "docs/reference/api/llm-agents" }
            },
            {
              "label": "Memory and RAG",
              "collapsed": true,
              "autogenerate": { "directory": "docs/reference/api/memory-rag" }
            },
            {
              "label": "Voice",
              "collapsed": true,
              "autogenerate": { "directory": "docs/reference/api/voice" }
            },
            {
              "label": "Infrastructure",
              "collapsed": true,
              "autogenerate": { "directory": "docs/reference/api/infrastructure" }
            },
            {
              "label": "Protocol and Server",
              "collapsed": true,
              "autogenerate": { "directory": "docs/reference/api/protocol" }
            }
          ]
        },
        {
          "label": "Architecture",
          "collapsed": true,
          "autogenerate": { "directory": "docs/reference/architecture" }
        },
        { "label": "Config Schema", "slug": "docs/reference/config-schema" }
      ]
    },
    {
      "label": "Contributing",
      "collapsed": true,
      "items": [
        { "label": "Overview", "slug": "docs/contributing" },
        { "label": "Development Setup", "slug": "docs/contributing/development-setup" },
        { "label": "Code Style", "slug": "docs/contributing/code-style" },
        { "label": "Testing", "slug": "docs/contributing/testing" },
        { "label": "Pull Requests", "slug": "docs/contributing/pull-requests" },
        { "label": "Releases", "slug": "docs/contributing/releases" },
        {
          "label": "Project Reports",
          "collapsed": true,
          "items": [
            { "label": "Changelog", "slug": "docs/contributing/project-reports/changelog" },
            { "label": "Security", "slug": "docs/contributing/project-reports/security" },
            { "label": "Code Quality", "slug": "docs/contributing/project-reports/code-quality" }
          ]
        }
      ]
    }
  ]
}
```

---

## 7. Redirect Map

| From | To | Reason |
|---|---|---|
| `/docs/getting-started/overview` | `/docs/start` | Section renamed |
| `/docs/getting-started/installation` | `/docs/start/installation` | Section renamed |
| `/docs/getting-started/quick-start` | `/docs/start/quick-start` | Section renamed |
| `/docs/tutorials` | `/docs/guides` | Section dissolved |
| `/docs/tutorials/foundation/custom-runnable` | `/docs/guides/foundations/custom-runnable` | |
| `/docs/tutorials/foundation/middleware-implementation` | `/docs/guides/foundations/middleware-implementation` | |
| `/docs/tutorials/agents/research-agent` | `/docs/guides/capabilities/agents/research-agent` | |
| `/docs/tutorials/agents/multi-agent-orchestration` | `/docs/guides/capabilities/agents/multi-agent-orchestration` | |
| `/docs/tutorials/rag/hybrid-search` | `/docs/guides/capabilities/rag/hybrid-search` | |
| `/docs/tutorials/voice/*` | `/docs/guides/capabilities/voice/*` | Bulk redirect rule |
| `/docs/tutorials/orchestration/*` | `/docs/guides/capabilities/orchestration/*` | Bulk redirect rule |
| `/docs/tutorials/safety/*` | `/docs/guides/production/*` | Bulk redirect rule |
| `/docs/tutorials/server/*` | `/docs/guides/production/*` | Bulk redirect rule |
| `/docs/use-cases` | `/docs/guides` | Section dissolved |
| `/docs/use-cases/search/semantic-search` | `/docs/guides/capabilities/rag/semantic-search` | Likely indexed |
| `/docs/use-cases/agents/research-agent` | `/docs/guides/capabilities/agents/research-agent` | Likely indexed |
| `/docs/use-cases/voice/*` | `/docs/guides/capabilities/voice/*` | Bulk redirect rule |
| `/docs/use-cases/safety/*` | `/docs/guides/production/*` | Bulk redirect rule |
| `/docs/use-cases/infrastructure/*` | `/docs/guides/production/*` | Bulk redirect rule |
| `/docs/integrations` | `/docs/guides/production/integrations` | Section dissolved |
| `/docs/integrations/infrastructure/auth0-jwt` | `/docs/guides/production/integrations/auth0-jwt` | High link volume |
| `/docs/integrations/infrastructure/vault-connector` | `/docs/guides/production/integrations/vault-connector` | High link volume |
| `/docs/integrations/infrastructure/kubernetes-helm` | `/docs/guides/production/integrations/kubernetes-helm` | High link volume |
| `/docs/integrations/llm/*` | `/docs/guides/production/integrations/*` | Bulk redirect rule |
| `/docs/integrations/data/*` | `/docs/guides/production/integrations/*` | Bulk redirect rule |
| `/docs/integrations/voice/*` | `/docs/guides/production/integrations/*` | Bulk redirect rule |
| `/docs/cookbook` | `/docs/recipes` | Section renamed |
| `/docs/cookbook/agents/*` | `/docs/recipes/agents/*` | Bulk redirect rule |
| `/docs/cookbook/llm/*` | `/docs/recipes/llm/*` | Bulk redirect rule |
| `/docs/cookbook/rag/*` | `/docs/recipes/rag/*` | Bulk redirect rule |
| `/docs/cookbook/memory/*` | `/docs/recipes/memory/*` | Bulk redirect rule |
| `/docs/cookbook/voice/*` | `/docs/recipes/voice/*` | Bulk redirect rule |
| `/docs/cookbook/multimodal/*` | `/docs/recipes/multimodal/*` | Bulk redirect rule |
| `/docs/cookbook/prompts/*` | `/docs/recipes/prompts/*` | Bulk redirect rule |
| `/docs/cookbook/infrastructure/*` | `/docs/recipes/infrastructure/*` | Bulk redirect rule |
| `/docs/architecture` | `/docs/reference/architecture/overview` | Section restructured |
| `/docs/architecture/concepts` | `/docs/concepts` | Moved to Concepts section |
| `/docs/architecture/packages` | `/docs/reference/architecture/packages` | |
| `/docs/architecture/providers` | `/docs/reference/architecture/providers` | |
| `/docs/providers` | `/docs/reference/providers` | Section moved into Reference |
| `/docs/providers/llm/*` | `/docs/reference/providers/llm/*` | Bulk redirect rule |
| `/docs/providers/embedding/*` | `/docs/reference/providers/embedding/*` | Bulk redirect rule |
| `/docs/providers/vectorstore/*` | `/docs/reference/providers/vectorstore/*` | Bulk redirect rule |
| `/docs/api-reference` | `/docs/reference/api` | |
| `/docs/api-reference/foundation/*` | `/docs/reference/api/foundation/*` | Bulk redirect rule |
| `/docs/api-reference/llm-agents/*` | `/docs/reference/api/llm-agents/*` | Bulk redirect rule |
| `/docs/api-reference/memory-rag/*` | `/docs/reference/api/memory-rag/*` | Bulk redirect rule |
| `/docs/api-reference/voice/*` | `/docs/reference/api/voice/*` | Bulk redirect rule |
| `/docs/api-reference/infrastructure/*` | `/docs/reference/api/infrastructure/*` | Bulk redirect rule |
| `/docs/api-reference/protocol/*` | `/docs/reference/api/protocol/*` | Bulk redirect rule |
| `/docs/reports` | `/docs/contributing` | Section dissolved into Contributing |
| `/docs/reports/changelog` | `/docs/contributing/project-reports/changelog` | |
| `/docs/reports/security` | `/docs/contributing/project-reports/security` | |
| `/docs/reports/code-quality` | `/docs/contributing/project-reports/code-quality` | |

---

## 8. Risks and Open Questions

**R1 — `ssml-tuning.md` slug collision**
Two files exist: `tutorials/voice/ssml-tuning.md` (step-by-step tutorial) and `cookbook/voice/ssml-tuning.md` (snippet). Under the new IA, the tutorial goes to `guides/capabilities/voice/ssml-tuning.md` and the recipe goes to `recipes/voice/ssml-tuning.md`. Assumed: identical slugs under different top-level sections do not collide in Starlight (they don't — slug uniqueness is per route). No action required beyond confirming the recipes version is actually a self-contained snippet and not a duplicate of the tutorial. If content is identical, delete the recipe version.

**R2 — `architecture/architecture.md` vs `architecture/index.md`**
Assumed these are duplicates covering the same 7-layer overview. If `architecture/architecture.md` contains unique material (e.g. full package dependency graph not in `index.md`), extract that material into a separate reference page before deleting. Affected file: `architecture/architecture.md`.

**R3 — Concepts section content sourcing**
The current `architecture/concepts.md` is the closest match for `concepts/index.md`, but the Concepts section needs 5 additional pages (see Gap Analysis) that do not exist. These are P0 for launch because the Concepts section would be sparse without them. Assumed: a content-writing pass on Concepts is a prerequisite before launch, not a post-launch task. Alternative: delay Concepts until post-launch and redirect `/docs/concepts` to `/docs/reference/architecture/overview` in the interim.

**R4 — Tutorials index pages**
The `tutorials/` subtree has per-subsection `index.md` files (e.g. `tutorials/agents/index.md` is not in the file listing, suggesting it may not exist). Confirmed absent from the listing: if they do not exist, no delete action is needed. If they do exist (the listing is non-exhaustive for subdirectory indexes without autogenerate), add them to the delete list. Affected directories: all `tutorials/*/`.

**R5 — `guides/` already partially migrated**
The current content tree already has a `guides/foundations/`, `guides/capabilities/`, and `guides/production/` structure that partially matches the target IA. This suggests a prior partial migration may have begun. The existing files in these directories (`prompt-engineering.md`, `safety-and-guards.md`, `observability.md`) already have the correct target paths and need no move — only the missing content from Tutorials/Use Cases/Integrations needs to be added to them. Confirm no slug conflicts between existing guides content and incoming files.

**R6 — `tutorials/providers/` placement**
The providers tutorial directory contains a mix: `new-llm-provider.md` (extension developer path → Foundations), `advanced-inference.md` (LLM technique → Foundations), embedding/vectorstore tutorials (RAG), and `multimodal-embeddings.md` (Multimodal). Assumed: no single target bucket; split as described in Section 2. If the team prefers a unified "provider how-tos" bucket under Guides/Production/integrations, 8 files would move there instead.

**R7 — `integrations/` sub-index files**
The Integrations section has many `index.md` files for each subcategory. These are all marked for deletion. If any contain unique introductory prose (not just an autogenerate list), that content should be extracted into the nearest parent guide page before deletion. A content audit pass is recommended before executing the delete.
