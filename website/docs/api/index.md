---
title: API Reference
sidebar_position: 1
---

# Beluga-AI API Reference

Welcome to the Beluga-AI API reference documentation. This section provides detailed information about the various packages and components that make up the Beluga-AI framework.

## Available Packages

Beluga-AI is organized into several packages, each focusing on specific functionality:

### Core Packages
- [Core](/docs/api/packages/core) - Core components and interfaces
- [Schema](/docs/api/packages/schema) - Data schemas and type definitions
- [Configuration](/docs/api/packages/config) - Configuration management for Beluga-AI applications

### LLM Packages
- [LLMs Base](/docs/api/packages/llms_base) - Large Language Model integrations and abstractions
- [ChatModels](/docs/api/packages/chatmodels) - Chat model interfaces and implementations
- [LLM Providers](/docs/api/packages/llms/) - Provider-specific implementations:
  - [OpenAI](/docs/api/packages/llms/openai) - OpenAI GPT models
  - [Anthropic](/docs/api/packages/llms/anthropic) - Anthropic Claude models
  - [AWS Bedrock](/docs/api/packages/llms/bedrock) - AWS Bedrock models
  - [Ollama](/docs/api/packages/llms/ollama) - Local Ollama models
  - [Cohere](/docs/api/packages/llms/cohere) - Cohere models

### Agent Packages
- [Agents](/docs/api/packages/agents) - Agent framework and implementations
- [Tools](/docs/api/packages/tools) - Tool abstractions and implementations
- [Orchestration](/docs/api/packages/orchestration) - Workflow orchestration and task management

### Memory & RAG Packages
- [Memory](/docs/api/packages/memory) - Memory systems for maintaining context and state
- [RAG](/docs/api/packages/rag) - Retrieval Augmented Generation components
- [Embeddings](/docs/api/packages/embeddings) - Embedding model interfaces and implementations
- [Vector Stores](/docs/api/packages/vectorstores) - Vector database interfaces and implementations
- [Retrievers](/docs/api/packages/retrievers) - Document retrieval components

### Supporting Packages
- [Prompts](/docs/api/packages/prompts) - Prompt management and formatting
- [Monitoring](/docs/api/packages/monitoring) - Observability, metrics, and tracing
- [Server](/docs/api/packages/server) - HTTP server components

## Getting Started

If you're new to Beluga-AI, we recommend starting with the [Introduction](/) guide before diving into the API documentation.

## API Stability

Beluga-AI is under active development. While we strive to maintain backwards compatibility, some APIs may change between releases. Check the [release notes](https://github.com/lookatitude/beluga-ai/releases) for information about API changes and deprecations.

