---
title: Overview
description: Introduction to Beluga AI v2 framework.
---

# Beluga AI v2

Beluga AI v2 is a ground-up Go framework for building agentic AI systems. It combines production patterns from Google ADK, OpenAI Agents SDK, LangGraph, ByteDance Eino, and LiveKit into a unified framework.

## Key Features

- **Streaming-first**: Built on Go 1.23+ `iter.Seq2[T, error]` for native streaming
- **Protocol interoperability**: MCP server/client + A2A for agent communication
- **Pluggable everything**: Registry pattern for LLM, tools, memory, voice, and more
- **Production patterns**: OpenTelemetry, circuit breakers, RBAC/ABAC, guard pipelines
- **Voice pipeline**: Frame-based STT/TTS/S2S with transport layer
- **RAG pipeline**: Hybrid search with vector + BM25 + RRF fusion

## Quick Start

```bash
go get github.com/lookatitude/beluga-ai@latest
```

See the [Architecture](/architecture/concepts/) section for design decisions and the [Package Layout](/architecture/packages/) for the full module structure.
