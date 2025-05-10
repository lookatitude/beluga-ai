<div align="center">
  <img src="./assets/beluga-logo.svg" alt="Beluga-AI Logo" width="300"/>
</div>

# Beluga-ai

**<font color=\'red\'>IMPORTANT NOTE: Beluga-ai is currently in an experimental state. APIs and features may change without notice. It is not recommended for production use at this stage.</font>**

**Beluga-ai** is a comprehensive framework written in Go, designed for building sophisticated AI and agentic applications. Inspired by frameworks like [LangChain](https://www.langchain.com/) and [CrewAI](https://www.crewai.com/), Beluga-ai provides a robust set of tools and abstractions to streamline the development of applications leveraging Large Language Models (LLMs).

This framework has recently undergone a significant refactoring to improve modularity, extendibility, and maintainability, adhering to Go best practices. For a detailed explanation of the new architecture, please see [Beluga_Refactored_Architecture.md](./Beluga_Refactored_Architecture.md).

## Overview

The goal of Beluga-ai is to offer a Go-native alternative for creating complex AI workflows. The recent refactoring has focused on establishing a clear, layered architecture with well-defined interfaces to support:

*   **LLM Integration:** Seamlessly connect to various LLM providers with a unified interface.
*   **Agent Creation:** Build autonomous agents capable of reasoning, planning, and executing tasks using a modular agent framework.
*   **Tool Management:** Define, integrate, and manage tools for agents to use.
*   **Memory Management:** Equip agents with different types of memory to maintain context.
*   **Retrieval-Augmented Generation (RAG):** Implement RAG pipelines for knowledge-intensive tasks.
*   **Orchestration:** Define and manage complex workflows with dedicated scheduler, message bus, and workflow components.
*   **Communication:** Establish protocols for inter-agent communication (future).

## Key Architectural Features (Post-Refactoring)

The refactored Beluga-ai framework emphasizes a modular and interface-driven design. Key components are now organized within the `pkg` directory:

*   **`pkg/schema`:** Centralized definitions for all core data structures and message types (e.g., `Message`, `Document`, `ChatHistory`).
*   **`pkg/core`:** Foundational utilities (`core/utils`) and core model definitions (`core/model`) used across the framework.
*   **`pkg/agents`:** Comprehensive toolkit for agent development, including:
    *   `base/`: Core `Agent` interface and `BaseAgent` embeddable struct.
    *   `tools/`: `Tool` interface, `BaseTool`, and `ToolAgentAction`.
    *   `executor/`: `AgentExecutor` for managing agent lifecycles.
    *   `factory/`: `AgentFactory` for creating agent instances.
*   **`pkg/orchestration`:** Components for managing complex task sequences:
    *   `scheduler/`: Task scheduling capabilities.
    *   `messagebus/`: Publish/subscribe messaging system.
    *   `workflow/`: Workflow definition and execution engine.
*   **Interface-Driven Design:** Promotes loose coupling and extensibility throughout the framework.
*   **Composition over Inheritance:** Aligns with Go idioms for building complex functionalities.

For a complete breakdown of the architecture, please refer to [Beluga_Refactored_Architecture.md](./Beluga_Refactored_Architecture.md).

## Installation

```bash
go get github.com/lookatitude/beluga-ai
```

## Examples

Detailed usage examples for each package and feature can be found in the [docs/examples](./docs/examples) directory. These examples will be updated to reflect the refactored architecture.

## Configuration

Beluga-ai uses Viper for configuration management, supporting YAML files and environment variables. See the [configuration example](./docs/examples/config/main.go) and `config.yml.example` for details. The core configuration mechanism remains, but module-specific configurations might be more structured post-refactoring.

## Implemented Features (Reflecting Refactored Structure)

While the core functionalities listed in the previous MVP remain, their organization and underlying implementation have been refactored for better modularity and extensibility as described in the architectural overview.

*   **Core & Schema:** As defined in `pkg/core` and `pkg/schema`.
*   **Configuration (`config` package):** Viper-based, as before.
*   **LLMs (`llms` package):** Interface-driven integration with OpenAI, Anthropic, Ollama, Google Gemini, Cohere, and AWS Bedrock.
*   **Tools (`pkg/agents/tools` and older `tools` package):** `Tool` interface and implementations.
*   **Memory (`memory` package):** `Memory` interface and implementations.
*   **Agents (`pkg/agents`):** Modular agent framework with `Agent` interface, `BaseAgent`, `AgentExecutor`, and `AgentFactory`.
*   **RAG (`rag` package):** Interfaces and implementations for `Loader`, `Splitter`, `Embedder`, `VectorStore`, `Retriever`.
*   **Orchestration (`pkg/orchestration`):** Initial implementations for `Scheduler`, `MessageBus`, and `Workflow` management.

## Future Plans (Post-MVP / v1.1 and Beyond)

The refactored architecture provides a stronger foundation for achieving these future goals:

*   **LLMs:** Complete AWS Bedrock client enhancements and standardize tool use across models.
*   **Tools:** Complete API, MCP tool implementations, and add more built-in tools.
*   **Memory:** Complete Summary Memory, Summary Buffer Memory, and Vector Store Memory.
*   **Agents:** Complete ReAct agent, explore other agent types, and enhance agent executors.
*   **RAG:** Complete OpenAI Embedder, PgVector `VectorStore`, add more loaders, splitters, and transformers.
*   **Orchestration (`pkg/orchestration` and older `orchestrator` package):** Deepen Temporal workflow integration and implement simpler chain patterns.
*   **Communication (`communication` package):** Define and implement inter-agent communication.
*   **Callbacks & Tracing:** Implement a comprehensive callback system and integrate with tracing systems.
*   **Prompt Templating:** Enhance prompt template management.
*   **Evaluation:** Add tools for evaluating LLMs, agents, and RAG pipelines.
*   **Testing:** Achieve comprehensive unit and integration test coverage across all refactored packages.
*   **Examples & Documentation:** Develop more complex examples reflecting the new architecture, refine all documentation, and create a dedicated documentation website.

## Contributing

Please see the [CONTRIBUTING.md](./CONTRIBUTING.md) file for guidelines on how to contribute to Beluga-ai.

## License

Beluga-ai is licensed under the [Apache 2.0 License](./LICENSE). *(Link to be verified/updated after LICENSE file check)*

