---
title: Capabilities
description: Deep dives into Beluga AI v2's major features and subsystems — RAG, memory, tools, voice, and multimodal processing.
sidebar:
  order: 0
---

These guides cover the major subsystems that give Beluga AI agents their capabilities. Each guide explains the architecture of a subsystem, the design patterns it uses, and how to integrate it into your applications. They assume familiarity with the [Foundation guides](/guides/foundations/) and build on the core patterns established there — registries, middleware, hooks, and streaming.

Every capability follows the same extensibility model: a small Go interface defines the contract, providers register via `init()`, and middleware wraps behavior without modifying implementations. This consistency means that once you learn one subsystem, the patterns transfer directly to the others.

| Guide | Description |
|-------|-------------|
| [RAG Pipeline](./rag-pipeline/) | Build retrieval-augmented generation pipelines with embeddings, vector stores, and advanced retrieval strategies like hybrid search, CRAG, and HyDE |
| [Document Processing](./document-processing/) | Load, parse, and chunk documents from multiple sources and formats for RAG pipelines and knowledge bases |
| [Memory System](./memory-system/) | Implement persistent agent memory using the MemGPT-inspired 3-tier model — Core (always in context), Recall (conversation history), and Archival (vector-searchable long-term storage) |
| [Tools & MCP](./tools-and-mcp/) | Create typed Go tools, organize them in registries, and connect to remote MCP servers for runtime tool discovery and interoperability |
| [Voice AI](./voice-ai/) | Build real-time voice applications using a frame-based processing pipeline with STT, TTS, S2S, VAD, and pluggable transport layers |
| [Multimodal](./multimodal/) | Process images, audio, and video with multimodal language models for document intelligence, visual Q&A, and content analysis |
