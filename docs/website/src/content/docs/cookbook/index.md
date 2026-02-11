---
title: Cookbook
description: "63 focused recipes for common tasks: agent patterns, LLM tricks, RAG optimization, voice tuning, and more."
---

Short, focused recipes that solve specific problems. Unlike tutorials, which build understanding progressively, cookbook recipes are self-contained solutions you can apply directly. Each recipe includes the problem statement, solution code, and an explanation of why the approach works. Browse by category or search for a specific topic.

## Recipe Categories

| Category | Recipes | Description |
|----------|---------|-------------|
| [Agent Recipes](./agents/) | Agent tool failures, parallel execution, custom patterns. Handle real-world agent challenges: graceful tool failure recovery, concurrent tool execution for throughput, and reusable agent patterns that compose with the `BaseAgent` abstraction. |
| [LLM Recipes](./llm/) | History trimming, streaming metadata, token counting, error handling. Solve common LLM integration problems: manage conversation history within token limits, extract metadata from streaming responses, count tokens accurately across providers, and implement resilient error handling with retry strategies. |
| [RAG Recipes](./rag/) | Batch embeddings, parent document retrieval, Cohere reranking. Optimize retrieval pipelines: batch embedding operations for throughput, parent document retrieval for preserving context around matched chunks, metadata-based filtering, and reranking with cross-encoder models for precision. |
| [Memory Recipes](./memory/) | TTL cleanup, context recovery, conversation expiry. Manage agent memory at scale: configure time-to-live policies for automatic cleanup, recover conversation context after service restarts, and implement expiry policies that balance storage costs with context availability. |
| [Voice Recipes](./voice/) | Stream scaling, backends, speech interruption, latency optimization. Tune production voice pipelines: scale concurrent audio streams, configure backend providers for your deployment, handle speech interruptions naturally, measure and optimize glass-to-glass latency, and implement jitter buffering for network resilience. |
| [Multimodal Recipes](./multimodal/) | Multiple images, inbound media handling. Process diverse content types: batch-process multiple images in a single LLM call, handle inbound media from messaging platforms with automatic format detection, and implement capability-based fallbacks when providers lack multimodal support. |
| [Prompt Recipes](./prompts/) | Dynamic templates, partial substitution. Build flexible prompt systems: construct prompts dynamically from runtime data, use partial template substitution for reusable prompt fragments, and manage prompt versioning across environments. |
| [Infrastructure Recipes](./infrastructure/) | Config hot reload, rate limiting, request ID correlation. Solve cross-cutting production concerns: hot-reload configuration without restarting, enforce per-tenant rate limits, propagate request IDs through distributed traces, detect and redact PII in LLM payloads, validate schemas at system boundaries, and manage circuit breakers for external dependencies. |

Each category contains a comprehensive overview plus individual recipe pages. Browse the sidebar categories to find specific topics.

## Recipe Format

Each recipe follows a consistent pattern:

1. **Problem** -- What you are trying to solve and why it matters in production
2. **Solution** -- Working code you can copy directly into your project
3. **Explanation** -- Why the approach works, what tradeoffs it makes, and when to consider alternatives
