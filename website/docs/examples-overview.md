---
sidebar_position: 3
---

# Examples Overview

Beluga-AI aims to be practical and easy to get started with. We provide a growing collection of examples to demonstrate how to use various components of the framework. These examples are designed to be runnable and to showcase specific features and common use cases.

## Finding the Examples

All code examples are located within the main Beluga-AI GitHub repository in the [`examples/`](https://github.com/lookatitude/beluga-ai/tree/main/examples) directory.

Each subdirectory there typically focuses on a specific package or concept:

*   **[`examples/llm-usage/`](https://github.com/lookatitude/beluga-ai/tree/main/examples/llm-usage):** Shows how to interact with different LLM providers, including making simple calls, streaming responses, and accessing token usage information.
*   **[`examples/agents/`](https://github.com/lookatitude/beluga-ai/tree/main/examples/agents):** Shows how to create and run agents using executors, LLMs, tools, and memory.
*   **[`examples/rag/`](https://github.com/lookatitude/beluga-ai/tree/main/examples/rag):** Demonstrates Retrieval-Augmented Generation (RAG) pipelines, including loading documents, splitting text, generating embeddings, storing in vector stores, and retrieving relevant documents.
*   **[`examples/orchestration/`](https://github.com/lookatitude/beluga-ai/tree/main/examples/orchestration):** Examples of workflow orchestration, chains, and complex task management.
*   **[`examples/multi-agent/`](https://github.com/lookatitude/beluga-ai/tree/main/examples/multi-agent):** Demonstrates multi-agent systems and agent collaboration patterns.
*   **[`examples/voice/`](https://github.com/lookatitude/beluga-ai/tree/main/examples/voice):** Comprehensive voice agent examples including STT, TTS, VAD, turn detection, and session management.
*   **[`examples/integration/`](https://github.com/lookatitude/beluga-ai/tree/main/examples/integration):** Integration examples showing how to combine multiple Beluga AI components.

## How to Run Examples

1.  **Clone the Repository:**
    ```bash
    git clone https://github.com/lookatitude/beluga-ai.git
    cd beluga-ai
    ```
2.  **Navigate to an Example Directory:**
    For instance, to run the LLM examples:
    ```bash
    cd examples/llm-usage
    ```
3.  **Set Up Configuration (if needed):**
    Many examples rely on API keys or specific model configurations. Ensure you have a `config.yaml` file set up in one of the expected locations (current directory, `$HOME/.beluga-ai/`, or `/etc/beluga-ai/`) or have the necessary environment variables exported (e.g., `BELUGA_LLMS_OPENAI_APIKEY`). Refer to the main [README.md](https://github.com/lookatitude/beluga-ai/blob/main/README) or the [Installation Guide](./getting-started/installation) documentation page for more details.
4.  **Run the Go Program:**
    ```bash
    go run main.go
    ```

## Contributing Examples

We welcome contributions of new examples! If you build something interesting with Beluga-AI or want to showcase a feature not yet covered, please feel free to open a Pull Request. Refer to our [CONTRIBUTING.md](https://github.com/lookatitude/beluga-ai/blob/main/CONTRIBUTING) guide.

We believe these examples will provide a solid starting point for exploring the capabilities of Beluga-AI. As the framework matures, we will continue to add more complex and diverse examples.

