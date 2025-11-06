---
sidebar_position: 3
---

# Examples Overview

Beluga-AI aims to be practical and easy to get started with. We provide a growing collection of examples to demonstrate how to use various components of the framework. These examples are designed to be runnable and to showcase specific features and common use cases.

## Finding the Examples

All code examples are located within the main Beluga-AI GitHub repository in the [`docs/examples`](https://github.com/lookatitude/beluga-ai/tree/main/docs/examples) directory.

Each subdirectory there typically focuses on a specific package or concept:

*   **[`docs/examples/config/`](https://github.com/lookatitude/beluga-ai/tree/main/docs/examples/config/main.go):** Demonstrates how to use the configuration management system.
*   **[`docs/examples/llms/`](https://github.com/lookatitude/beluga-ai/tree/main/docs/examples/llms/main.go):** Shows how to interact with different LLM providers, including making simple calls, streaming responses, and accessing token usage information.
*   **[`docs/examples/tools/`](https://github.com/lookatitude/beluga-ai/tree/main/docs/examples/tools/main.go):** Illustrates defining and using tools like Go functions and shell commands.
*   **[`docs/examples/memory/`](https://github.com/lookatitude/beluga-ai/tree/main/docs/examples/memory/main.go):** Provides examples of using different memory types like Buffer and Window Buffer memory.
*   **[`docs/examples/agents/`](https://github.com/lookatitude/beluga-ai/tree/main/docs/examples/agents/main.go):** Shows how to create and run a basic agent using an executor, LLM, tools, and memory.
*   **[`docs/examples/rag/`](https://github.com/lookatitude/beluga-ai/tree/main/docs/examples/rag/main.go):** Demonstrates a simple Retrieval-Augmented Generation (RAG) pipeline, including loading documents, splitting text, generating embeddings (with Ollama), storing in an in-memory vector store, and retrieving relevant documents.

## How to Run Examples

1.  **Clone the Repository:**
    ```bash
    git clone https://github.com/lookatitude/beluga-ai.git
    cd beluga-ai
    ```
2.  **Navigate to an Example Directory:**
    For instance, to run the LLM examples:
    ```bash
    cd docs/examples/llms
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

