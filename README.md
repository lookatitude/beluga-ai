# Beluga AI Framework - README (Augmented for Extensibility)

<div align="center">
  <img src="./assets/beluga-logo.svg" alt="Beluga-AI Logo" width="300"/>
</div>

# Beluga-ai

**<font color=\'red\'>IMPORTANT NOTE: Beluga-ai is currently in an experimental state. APIs and features may change without notice. It is not recommended for production use at this stage.</font>**

**Beluga-ai** is a comprehensive framework written in Go, designed for building sophisticated AI and agentic applications. Inspired by frameworks like [LangChain](https://www.langchain.com/) and [CrewAI](https://www.crewai.com/), Beluga-ai provides a robust set of tools and abstractions to streamline the development of applications leveraging Large Language Models (LLMs).

This framework has recently undergone a significant refactoring to improve modularity, extendibility, and maintainability, adhering to Go best practices. For a detailed explanation of the new architecture, please see [Beluga_Refactored_Architecture.md](./Beluga_Refactored_Architecture.md).

## Overview

The goal of Beluga-ai is to offer a Go-native alternative for creating complex AI workflows. The recent refactoring has focused on establishing a clear, layered architecture with well-defined interfaces to support:

*   **Extensible LLM Integration:** Seamlessly connect to various LLM providers (e.g., OpenAI, Anthropic, Google Gemini) with a unified interface and provider-specific adapters.
*   **Agent Creation:** Build autonomous agents capable of reasoning, planning, and executing tasks using a modular agent framework.
*   **Extensible Tool Management:** Define, integrate, and manage diverse tools (e.g., Shell, Go Functions, API callers) for agents to use through a common interface.
*   **Extensible Memory Management:** Equip agents with different types of memory, supporting various backends, including multiple vector database providers (e.g., InMemory, pgvector, Pinecone, Weaviate).
*   **Retrieval-Augmented Generation (RAG):** Implement RAG pipelines with swappable components for data loading, splitting, embedding, and retrieval.
*   **Extensible Orchestration:** Define and manage complex workflows with a flexible engine, potentially integrating with external orchestrators.
*   **Communication:** Establish protocols for inter-agent communication (future).

## Core Principle: Extensibility via Provider Interfaces

Beluga-ai is built with extensibility at its core. Key components are designed around Go interfaces, allowing developers to easily implement and integrate their own providers or third-party services. This is typically achieved through:

1.  **Provider-Agnostic Interfaces:** Clear Go interfaces for LLMs, Tools, Memory, VectorStores, Workflow Engines, etc.
2.  **Provider-Specific Implementations:** Concrete structs that implement these interfaces for specific services (e.g., `OpenAI_LLM`, `PgVectorStore`).
3.  **Configuration-Driven Selection:** YAML configuration files allow users to specify which provider to use for each component.
4.  **Factory Patterns:** Factories instantiate the correct provider implementation based on the configuration.

See the [Beluga_Refactored_Architecture.md](./Beluga_Refactored_Architecture.md) for more details on how to extend specific components.

## Key Architectural Features (Post-Refactoring)

The refactored Beluga-ai framework emphasizes a modular and interface-driven design. Key components are now organized within the `pkg` directory:

*   **`pkg/schema`:** Centralized definitions for all core data structures.
*   **`pkg/core`:** Foundational utilities and core model definitions.
*   **`pkg/llms`:** `LLM` interface, provider implementations (e.g., `openai`, `anthropic`), and `LLMProviderFactory`.
*   **`pkg/prompts`:** `PromptAdapter` interface and implementations for model-specific prompt formatting.
*   **`pkg/agents`:** Comprehensive toolkit for agent development (`base`, `tools`, `executor`, `factory`).
    *   `pkg/agents/tools/providers`: Contains implementations for various tool types (e.g., `shell_tool.go`, `gofunction_tool.go`).
*   **`pkg/memory`:** `Memory` interface, basic implementations, and `VectorStoreMemory`.
    *   `pkg/vectorstores`: `VectorStore` interface, provider implementations (e.g., `inmemory`, `pgvector`, `pinecone`), and `VectorStoreProviderFactory`.
    *   `pkg/embeddings`: `Embedder` interface and implementations (e.g., `openai`).
*   **`pkg/orchestration`:** Components for managing complex task sequences (`scheduler`, `messagebus`, `workflow`).
    *   `pkg/orchestration/workflow/factory`: `WorkflowProviderFactory` for different workflow engines.
*   **`pkg/config`:** Configuration models and Viper-based loading, supporting provider selection.

For a complete breakdown of the architecture, please refer to [Beluga_Refactored_Architecture.md](./Beluga_Refactored_Architecture.md).

## Installation

```bash
go get github.com/lookatitude/beluga-ai
```

## Examples

Detailed usage examples, including how to configure and use different providers for LLMs, VectorStores, and Tools, can be found in the `/examples` directory (to be populated as features are implemented).

## Configuration

Beluga-ai uses Viper for configuration management. An example `config.yaml` would look like:

```yaml
llm_provider: "openai" # or "anthropic", "gemini"
openai_config:
  api_key: "your_openai_api_key"
  default_model: "gpt-4"

vector_store_provider: "pgvector" # or "pinecone", "weaviate", "inmemory"
pgvector_config:
  connection_string: "postgres://user:pass@host:port/db"

# ... other configurations
```

## Implemented and Planned Features (with Extensibility)

*   **LLMs (`pkg/llms`):** Interface-driven integration. Initial: OpenAI. Planned: Anthropic, Google Gemini, Cohere, AWS Bedrock. Includes `PromptAdapter` for model-specific formatting.
*   **Tools (`pkg/agents/tools`):** `Tool` interface. Initial: Echo, Calculator. Planned: Shell, GoFunction, WebSearch, MCPServerTool.
*   **Memory & VectorStores (`pkg/memory`, `pkg/vectorstores`):** `Memory` and `VectorStore` interfaces. Initial: BufferMemory, InMemoryVectorStore. Planned: WindowBufferMemory, VectorStoreMemory with PgVector, Pinecone, Weaviate backends.
*   **Agents (`pkg/agents`):** Modular agent framework.
*   **RAG (`rag` package - conceptual):** Composable RAG pipeline components (Loaders, Splitters, Embedders, Retrievers) will also follow extensible patterns.
*   **Orchestration (`pkg/orchestration`):** `WorkflowEngine` interface. Initial: InMemorySequentialEngine. Planned: Integration with external orchestrators like Temporal.

## Future Plans (Post-MVP / v1.1 and Beyond)

The refactored architecture and focus on extensibility provide a stronger foundation for achieving these future goals:

*   **Provider Implementations:** Systematically add more providers for LLMs, VectorStores, Tools, and Workflow Engines as outlined in the developer tasks.
*   **Enhanced RAG:** Complete diverse loaders, splitters, and retriever implementations.
*   **Advanced Agent Types:** Explore and implement more sophisticated agent types (e.g., ReAct, planning agents).
*   **Inter-Agent Communication:** Define and implement robust protocols.
*   **Callbacks, Tracing, Evaluation:** Implement comprehensive systems for these aspects.
*   **Comprehensive Testing & Documentation:** Achieve high test coverage and create detailed documentation for users and developers, especially on extending the framework.

## Contributing

Please see the [CONTRIBUTING.md](./CONTRIBUTING.md) file for guidelines on how to contribute to Beluga-ai, including how to add new provider implementations.

## License

Beluga-ai is licensed under the [Apache 2.0 License](./LICENSE).


