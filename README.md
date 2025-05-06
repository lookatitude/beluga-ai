# Beluga-ai

**Beluga-ai** is a comprehensive framework written in Go, designed for building sophisticated AI and agentic applications. Inspired by frameworks like LangChain and CrewAI, Beluga-ai provides a robust set of tools and abstractions to streamline the development of applications leveraging Large Language Models (LLMs).

## Overview

The goal of Beluga-ai is to offer a Go-native alternative for creating complex AI workflows, including:

*   **LLM Integration:** Seamlessly connect to various LLM providers (OpenAI, Anthropic, Ollama, AWS Bedrock) with a unified interface.
*   **Agent Creation:** Build autonomous agents capable of reasoning, planning, and executing tasks using available tools.
*   **Tool Management:** Define, integrate, and manage tools (Go functions, shell commands, APIs) for agents to use.
*   **Memory Management:** Equip agents with different types of memory (buffer, window, summary, vector store) to maintain context and learn from interactions.
*   **Retrieval-Augmented Generation (RAG):** Implement RAG pipelines with components for loading, splitting, embedding, and retrieving documents from vector stores (in-memory, PgVector).
*   **Orchestration:** Define and manage complex workflows and agent interactions (with planned integration for Temporal).
*   **Communication:** Establish protocols for inter-agent communication (planned feature).

## Installation

```bash
go get github.com/lookatitude/beluga-ai
```
*(Note: This assumes the repository is publicly available at this path. Adjust as necessary.)*

## Basic Usage

*(Example code demonstrating a simple LLM call or agent interaction will be added here.)*

## Examples

Detailed usage examples for each package and feature can be found in the [docs/examples](./docs/examples) directory.

## Configuration

Beluga-ai uses Viper for configuration management, supporting YAML files and environment variables.

1.  **Config File:** Create a `config.yaml` file in one of the following locations:
    *   Current directory (`.`)
    *   `$HOME/.beluga-ai/`
    *   `/etc/beluga-ai/`

    Example `config.yaml`:
    ```yaml
    llms:
      openai:
        api_key: "YOUR_OPENAI_API_KEY"
        model: "gpt-4o"
      anthropic:
        api_key: "YOUR_ANTHROPIC_API_KEY"
        model: "claude-3-haiku-20240307"
      ollama:
        base_url: "http://localhost:11434"
        model: "llama3"
    # rag:
    #   pgvector:
    #     connection_string: "postgres://user:password@host:port/dbname"
    ```

2.  **Environment Variables:** Set environment variables prefixed with `BELUGA_`. Use underscores (`_`) instead of periods (`.`).
    ```bash
    export BELUGA_LLMS_OPENAI_APIKEY="YOUR_OPENAI_API_KEY"
    export BELUGA_LLMS_ANTHROPIC_APIKEY="YOUR_ANTHROPIC_API_KEY"
    export BELUGA_LLMS_OLLAMA_BASEURL="http://localhost:11434"
    ```

The framework loads configuration automatically using the `config.LoadConfig()` function.

## Implemented Features

*   **Core:**
    *   `Runnable` interface for composable components.
    *   Schema definitions (`Message`, `Document`, `ToolCall`, etc.).
*   **Configuration:**
    *   Viper-based loading from files (YAML) and environment variables (`config` package).
*   **LLMs (`llms` package):**
    *   `ChatModel` interface.
    *   Implementations for OpenAI, Anthropic, Ollama, AWS Bedrock (Anthropic provider), Google Gemini, Cohere (including streaming and tool use where applicable).
    *   Consistent token usage tracking (input, output, total tokens) available in `AIMessage.AdditionalArgs["usage"]` for all supported models.
    *   Basic structure for AWS Bedrock.
    *   Standard `Generate`, `StreamChat`, `BindTools`, `Invoke`, `Batch`, `Stream` methods.
*   **Tools (`tools` package):**
    *   `Tool` interface.
    *   Implementations for Go Functions (`gofunc`), Shell commands (`shell`).
    *   Basic structures for API calls (`api`) and MCP (`mcp`).
*   **Memory (`memory` package):**
    *   `Memory` interface.
    *   Implementations for Buffer Memory, Window Buffer Memory.
    *   Basic structures for Summary, Summary Buffer, and Vector Store memory.
*   **Agents (`agents` package):**
    *   `Agent` interface.
    *   `Executor` implementation for running agents with tools and memory.
    *   Basic structure for ReAct agent logic.
*   **RAG (`rag` package):**
    *   Interfaces for `Loader`, `Splitter`, `Embedder`, `VectorStore`, `Retriever`.
    *   Implementations: `FileLoader`, `CharacterSplitter`, `OllamaEmbedder`, `InMemoryVectorStore`, `VectorStoreRetriever`.
    *   Basic structures for `OpenAIEmbedder`, `PgVectorStore`.
*   **Orchestration (`orchestrator` package):**
    *   Basic `Orchestrator` interface.
    *   Basic structure for Temporal integration.
*   **Communication (`communication` package):**
    *   Basic `Communicator` interface.

## Future Plans / To Be Implemented

*   **LLMs:**
    *   Complete AWS Bedrock client implementation.
    *   Implement token usage tracking more consistently across models. (Implemented)
*   **Tools:**
    *   Complete API tool implementation for interacting with external REST/GraphQL APIs.
    *   Complete MCP tool implementation.
    *   Add more built-in tools (e.g., web search, file system operations with safety checks).
*   **Memory:**
    *   Complete Summary Memory (requires LLM for summarization).
    *   Complete Summary Buffer Memory.
    *   Complete Vector Store Memory (requires Vector Store implementation).
*   **Agents:**
    *   Complete ReAct (Reasoning and Acting) agent implementation.
    *   Explore and implement other agent types (e.g., Plan-and-Execute).
    *   Develop more sophisticated agent executors with better error handling and planning.
*   **RAG:**
    *   Complete OpenAI Embedder implementation.
    *   Complete PgVector VectorStore implementation.
    *   Add support for other vector databases (e.g., Chroma, Pinecone) via interfaces.
    *   Add more document loaders (e.g., Web Loader, PDF Loader).
    *   Add more text splitters (e.g., Token Splitter, Recursive Character Splitter).
    *   Implement document transformers (e.g., metadata extraction).
*   **Orchestration:**
    *   Complete Temporal workflow integration for robust agent/task orchestration.
    *   Implement simpler orchestration patterns (e.g., Sequential Chains, basic routing).
*   **Communication:**
    *   Define and implement protocols and mechanisms for inter-agent communication.
*   **Callbacks & Tracing:**
    *   Implement a callback system for logging, monitoring, and tracing execution steps.
    *   Integrate with tracing systems (e.g., OpenTelemetry).
*   **Prompt Templating:**
    *   Enhance prompt template management and formatting capabilities.
*   **Evaluation:**
    *   Add tools and frameworks for evaluating the performance of agents and RAG pipelines.
*   **Testing:**
    *   Add comprehensive unit and integration tests across all packages.
*   **Examples & Documentation:**
    *   Add more complex and diverse usage examples.
    *   Refine documentation and add tutorials.

## Contributing

*(Contribution guidelines can be added here.)*

## License

*(License information can be added here.)*

