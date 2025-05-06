<div align="center">
  <img src="./assets/beluga-logo.svg" alt="Beluga-AI Logo" width="300"/>
</div>

# Beluga-ai

**<font color=\'red\'>IMPORTANT NOTE: Beluga-ai is currently in an experimental state. APIs and features may change without notice. It is not recommended for production use at this stage.</font>**

**Beluga-ai** is a comprehensive framework written in Go, designed for building sophisticated AI and agentic applications. Inspired by frameworks like [LangChain](https://www.langchain.com/) and [CrewAI](https://www.crewai.com/), Beluga-ai provides a robust set of tools and abstractions to streamline the development of applications leveraging Large Language Models (LLMs).

## Overview

The goal of Beluga-ai is to offer a Go-native alternative for creating complex AI workflows, including:

*   **LLM Integration:** Seamlessly connect to various LLM providers with a unified interface.
*   **Agent Creation:** Build autonomous agents capable of reasoning, planning, and executing tasks.
*   **Tool Management:** Define, integrate, and manage tools for agents to use.
*   **Memory Management:** Equip agents with different types of memory to maintain context.
*   **Retrieval-Augmented Generation (RAG):** Implement RAG pipelines for knowledge-intensive tasks.
*   **Orchestration:** Define and manage complex workflows (future).
*   **Communication:** Establish protocols for inter-agent communication (future).

## Installation

```bash
go get github.com/lookatitude/beluga-ai
```

## Examples

Detailed usage examples for each package and feature can be found in the [docs/examples](./docs/examples) directory.

## Configuration

Beluga-ai uses Viper for configuration management, supporting YAML files and environment variables. See the [configuration example](./docs/examples/config/main.go) and `config.yml.example` for details.

## Implemented Features (MVP - Alpha)

*   **Core:**
    *   `Runnable` interface for composable components.
    *   Schema definitions (`Message`, `Document`, `ToolCall`, etc.).
*   **Configuration (`config` package):**
    *   Viper-based loading from files (YAML) and environment variables.
*   **LLMs (`llms` package):**
    *   `ChatModel` interface with standard methods: `Generate`, `StreamChat`, `BindTools`, `Invoke`, `Batch`, `Stream`.
    *   Implementations for:
        *   OpenAI (GPT-3.5, GPT-4, etc.)
        *   Anthropic (Claude models)
        *   Ollama (Local LLMs like Llama 3, Mistral, etc.)
        *   Google Gemini
        *   Cohere (Command R, etc.)
        *   AWS Bedrock:
            *   Anthropic Claude
            *   Mistral AI (e.g., Mistral 7B Instruct)
            *   Meta Llama (e.g., Llama 3 8B Instruct)
    *   Consistent token usage tracking (`AIMessage.AdditionalArgs["usage"]`).
    *   Tool use support (provider-dependent, may require prompt engineering for some Bedrock models via `InvokeModel`).
*   **Tools (`tools` package):**
    *   `Tool` interface.
    *   Implementations for Go Functions (`gofunc`) and Shell commands (`shell`).
*   **Memory (`memory` package):**
    *   `Memory` interface.
    *   Implementations for Buffer Memory and Window Buffer Memory.
*   **Agents (`agents` package):**
    *   `Agent` interface.
    *   `Executor` for running agents with tools and memory.
    *   Basic structure for ReAct agent logic.
*   **RAG (`rag` package):**
    *   Interfaces: `Loader`, `Splitter`, `Embedder`, `VectorStore`, `Retriever`.
    *   Implementations: `FileLoader`, `CharacterSplitter`, `OllamaEmbedder`, `InMemoryVectorStore`, `VectorStoreRetriever`.

## Future Plans (Post-MVP / v1.1 and Beyond)

*   **LLMs:**
    *   Complete AWS Bedrock client by adding full support for:
        *   Cohere (Command R+, etc. via Bedrock, ensuring full feature parity with native client)
        *   AI21 Labs (Jamba, etc. via Bedrock)
        *   Amazon Titan (Text models via Bedrock, ensuring full feature parity)
    *   Enhance and standardize tool use capabilities across all Bedrock models, potentially leveraging Bedrock Converse API where beneficial.
*   **Tools:**
    *   Complete API tool implementation for interacting with external REST/GraphQL APIs.
    *   Complete MCP (Multi-Cloud Provider) tool implementation.
    *   Add more built-in tools (e.g., web search, file system operations with safety checks).
*   **Memory:**
    *   Complete Summary Memory (requires LLM for summarization).
    *   Complete Summary Buffer Memory.
    *   Complete Vector Store Memory (integrating `VectorStore` into memory interface).
*   **Agents:**
    *   Complete ReAct (Reasoning and Acting) agent implementation.
    *   Explore and implement other agent types (e.g., Plan-and-Execute).
    *   Develop more sophisticated agent executors with better error handling, planning, and state management.
*   **RAG:**
    *   Complete OpenAI Embedder implementation.
    *   Complete PgVector `VectorStore` implementation.
    *   Add support for other vector databases (e.g., Chroma, Pinecone) via interfaces.
    *   Add more document loaders (e.g., Web Loader, PDF Loader).
    *   Add more text splitters (e.g., Token Splitter, Recursive Character Splitter).
    *   Implement document transformers (e.g., metadata extraction, summarization).
*   **Orchestration (`orchestrator` package):**
    *   Complete Temporal workflow integration for robust agent/task orchestration.
    *   Implement simpler orchestration patterns (e.g., Sequential Chains, basic routing).
*   **Communication (`communication` package):**
    *   Define and implement protocols and mechanisms for inter-agent communication.
*   **Callbacks & Tracing:**
    *   Implement a comprehensive callback system for logging, monitoring, and tracing execution steps.
    *   Integrate with tracing systems (e.g., OpenTelemetry).
*   **Prompt Templating:**
    *   Enhance prompt template management and formatting capabilities, including support for chat templates.
*   **Evaluation:**
    *   Add tools and frameworks for evaluating the performance of LLMs, agents, and RAG pipelines.
*   **Testing:**
    *   Achieve comprehensive unit and integration test coverage across all packages.
*   **Examples & Documentation:**
    *   Develop more complex and diverse usage examples showcasing real-world applications.
    *   Refine all documentation, add tutorials, and generate comprehensive API docs.
    *   Create a dedicated documentation website (GitHub Pages).

## Contributing

Please see the [CONTRIBUTING.md](./CONTRIBUTING.md) file for guidelines on how to contribute to Beluga-ai.

## License

Beluga-ai is licensed under the [Apache 2.0 License](./LICENSE). *(Link to be verified/updated after LICENSE file check)*

