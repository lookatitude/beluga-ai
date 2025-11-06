---
sidebar_position: 1
id: intro
---

# Welcome to Beluga-AI

**🚀 PRODUCTION READY:** Beluga AI Framework has completed comprehensive standardization and is now enterprise-grade with consistent patterns, extensive testing, and production-ready observability across all 14 packages. The framework is ready for large-scale deployment and development teams.

**Beluga-ai** is a comprehensive, production-ready framework written in Go, designed for building sophisticated AI and agentic applications. Inspired by pioneering frameworks like [LangChain](https://www.langchain.com/) and [CrewAI](https://www.crewai.com/), Beluga-ai provides a robust set of tools and abstractions to streamline the development of applications leveraging Large Language Models (LLMs).

## Our Vision

The goal of Beluga-ai is to offer a Go-native, performant, and flexible alternative for creating complex AI workflows. We aim to empower Go developers to build next-generation AI applications with enterprise-grade observability, comprehensive testing, and production-ready patterns.

## Key Features

Beluga-AI offers a modular and extensible architecture with enterprise-grade capabilities. Here's what's available in the framework:

*   **Core Abstractions:**
    *   A `Runnable` interface promoting composable and reusable components.
    *   Well-defined schemas for `Message`, `Document`, `ToolCall`, and other fundamental AI concepts.
*   **Configuration Management (`config` package):**
    *   Leverages Viper for easy loading of configurations from YAML files and environment variables.
*   **LLM Integrations (`llms` package):**
    *   A unified `ChatModel` interface with standard methods: `Generate`, `StreamChat`, `BindTools`, `Invoke`, `Batch`, and `Stream`.
    *   Support for a wide range of LLM providers:
        *   OpenAI (GPT-3.5, GPT-4, etc.)
        *   Anthropic (Claude models)
        *   Ollama (for running local LLMs like Llama 3, Mistral)
        *   Google Gemini
        *   Cohere (Command R family)
        *   AWS Bedrock, including:
            *   Anthropic Claude on Bedrock
            *   Mistral AI models on Bedrock
            *   Meta Llama models on Bedrock
    *   Consistent token usage tracking (input, output, total tokens) exposed via `AIMessage.AdditionalArgs["usage"]`.
    *   Flexible tool use support, adapting to each provider's capabilities (may require specific prompt engineering for some Bedrock models when using `InvokeModel`, while others offer more structured interaction).
*   **Tooling (`tools` package):**
    *   A generic `Tool` interface for defining custom capabilities.
    *   Out-of-the-box implementations for Go Functions (`gofunc`) and executing Shell commands (`shell`).
*   **Memory Systems (`memory` package):**
    *   A `Memory` interface for equipping agents with contextual awareness.
    *   Implementations for Buffer Memory and Window Buffer Memory.
*   **Agentic Systems (`agents` package):**
    *   An `Agent` interface for building autonomous entities.
    *   An `Executor` component to run agents with their assigned tools and memory.
    *   Initial structure for ReAct (Reasoning and Acting) agent logic.
*   **Retrieval-Augmented Generation (`rag` package):**
    *   Core interfaces: `Loader`, `Splitter`, `Embedder`, `VectorStore`, and `Retriever`.
    *   Implementations including: `FileLoader`, `CharacterSplitter`, `OllamaEmbedder` (for local embeddings), `InMemoryVectorStore`, and `VectorStoreRetriever`.

For a complete list of currently implemented features and our exciting roadmap, please refer to the main [README.md](https://github.com/lookatitude/beluga-ai/blob/main/README) on our GitHub repository.

## Inspiration

Beluga-AI draws inspiration from the vibrant ecosystem of AI and LLM orchestration frameworks. We have particularly looked at:

*   **[LangChain](https://www.langchain.com/):** A widely adopted framework for developing applications powered by language models. Its comprehensive nature and modular design have been a significant influence.
*   **[CrewAI](https://www.crewai.com/):** A newer framework focused on orchestrating role-playing, autonomous AI agents. Its emphasis on collaborative agent systems is an area we are keenly exploring for future Beluga-AI capabilities.

While inspired by these and other projects, Beluga-AI aims to carve its own niche by providing a Go-centric approach, focusing on performance, type safety, and the idiomatic Go way of building software.

## Getting Started

Dive into the [Installation](./getting-started/installation) guide to set up Beluga-AI in your Go environment, or explore the [Examples Overview](./examples-overview) to see it in action.

