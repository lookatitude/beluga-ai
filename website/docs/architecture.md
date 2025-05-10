---
sidebar_position: 2
---

# Architecture Overview

Beluga-AI is designed with a modular and extensible architecture to facilitate the creation of complex AI and agentic applications in Go. The core philosophy is to provide clear interfaces and composable components that developers can easily understand, use, and extend.

## Core Components

The framework is built around several key packages and concepts:

*   **`schema`**: Defines the fundamental data structures used throughout the framework, such as `Message` (Human, AI, System, Tool), `Document`, `ToolCall`, and `TokenUsage`. This ensures consistency and interoperability between different modules.

*   **`core` (`runnable`)**: Introduces the `Runnable` interface, a central piece for creating composable data processing pipelines. Many components in Beluga-AI implement this interface, allowing them to be chained together in a declarative way.

*   **`config`**: Manages application configuration using Viper. It supports loading settings from YAML files and environment variables, making it easy to configure LLM API keys, model names, and other parameters.

*   **`llms`**: Provides a unified `ChatModel` interface for interacting with various Large Language Models. Each supported provider (OpenAI, Anthropic, Ollama, Bedrock, Gemini, Cohere) has its own implementation, abstracting away provider-specific details. This package handles request formatting, API calls, response parsing, streaming, tool binding, and token usage tracking.

*   **`tools`**: Defines a `Tool` interface that agents can use to interact with the external world or perform specific tasks. Implementations are provided for executing Go functions (`gofunc`) and shell commands (`shell`), with a clear path for adding custom tools.

*   **`memory`**: Offers a `Memory` interface and implementations (e.g., `BufferMemory`, `WindowBufferMemory`) to provide LLMs and agents with short-term and long-term context. This is crucial for maintaining coherent conversations and learning from past interactions.

*   **`agents`**: Contains the logic for creating and running autonomous agents. The `Agent` interface and the `Executor` component allow agents to use LLMs, tools, and memory to reason, plan, and execute tasks.

*   **`rag` (Retrieval-Augmented Generation)**: Implements the components necessary for RAG pipelines. This includes:
    *   `Loader`: For loading documents from various sources (e.g., `FileLoader`).
    *   `Splitter`: For breaking down large documents into smaller chunks (e.g., `CharacterSplitter`).
    *   `Embedder`: For generating vector embeddings of text (e.g., `OllamaEmbedder`).
    *   `VectorStore`: For storing and querying vector embeddings (e.g., `InMemoryVectorStore`).
    *   `Retriever`: For fetching relevant documents based on a query (e.g., `VectorStoreRetriever`).

*   **`orchestrator` (Planned)**: This package will house components for managing complex workflows involving multiple agents and services. Integration with workflow engines like Temporal is planned.

*   **`communication` (Planned)**: This package will define protocols and mechanisms for inter-agent communication, enabling more sophisticated collaborative agent systems.

## Design Principles

*   **Modularity:** Each component is designed to be as self-contained as possible, promoting separation of concerns.
*   **Extensibility:** Interfaces are used extensively to allow developers to easily add new LLM providers, tools, memory types, or agent behaviors.
*   **Go-idiomatic:** The framework strives to follow Go best practices and conventions, making it feel natural for Go developers.
*   **Type Safety:** Leveraging Go's static typing to catch errors at compile time and improve code reliability.
*   **Performance:** While providing high-level abstractions, performance considerations are kept in mind, especially for a compiled language like Go.

## Workflow Example (Conceptual)

A typical workflow in Beluga-AI might involve:

1.  Loading configuration (API keys, model choices).
2.  Initializing an LLM client (e.g., OpenAI).
3.  Defining a set of tools available to an agent.
4.  Setting up a memory module for the agent.
5.  Creating an agent with the LLM, tools, and memory.
6.  Using an `Executor` to run the agent with an initial input or task.
7.  The agent interacts with the LLM, uses tools if necessary, updates its memory, and generates a response or takes actions.

This modular design allows for flexible construction of various AI-powered applications, from simple LLM-backed chatbots to complex multi-agent systems performing RAG.

As the project evolves, we will continue to refine this architecture and add more capabilities, always with the goal of making advanced AI development in Go more accessible and powerful.

