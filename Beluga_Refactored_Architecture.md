# Beluga AI Framework - Refactored Architecture

This document outlines the refactored architecture of the Beluga AI Framework, detailing the module structure, design principles, and key changes implemented during the refactoring process. The primary goals of this refactoring were to improve modularity, extendibility, maintainability, and ease of use, adhering to Go best practices and modern design patterns.

## 1. Overview of Architectural Changes

The refactoring process involved a significant restructuring of the Beluga AI framework to promote a cleaner separation of concerns and a more intuitive developer experience. The core philosophy was to create a layered architecture with well-defined interfaces, making the framework more robust and adaptable to future requirements.

Key changes include:

*   **Modular Package Structure:** The `pkg` directory now houses clearly delineated modules for core functionalities, schema definitions, agents, orchestration, and utilities. This replaces a flatter or less organized structure, making it easier to locate and understand different parts of the framework.
*   **Interface-Driven Design:** Emphasis has been placed on defining clear interfaces for key components like `Agent`, `Tool`, `Scheduler`, `MessageBus`, and `Workflow`. This promotes loose coupling and allows for easier substitution of implementations.
*   **Composition over Inheritance:** The framework now favors composition for building complex functionalities, aligning with Go idioms. `BaseAgent`, `BaseTool`, and `BaseWorkflow` provide foundational structs that can be embedded, rather than relying on classical inheritance.
*   **Dedicated Schema Package:** All core data structures and message types are now centralized in the `pkg/schema` package, providing a single source of truth for data exchange within the framework.
*   **Clear Separation of Agent Components:** The `pkg/agents` directory is further subdivided into `base` (for core agent interfaces and base implementations), `tools` (for tool definitions), `executor` (for agent execution logic), and `factory` (for agent creation).
*   **Orchestration Layer:** The `pkg/orchestration` package introduces components like `Scheduler`, `MessageBus`, and `Workflow` to manage complex interactions and task sequences.
*   **Utility and Core Packages:** Common utilities are placed in `pkg/core/utils`, and core model definitions (if any beyond schema) are in `pkg/core/model`.

## 2. Module Structure and Responsibilities

The refactored `pkg` directory is organized as follows:

### `pkg/schema`
*   **Purpose:** Defines all core data structures, message types, and interfaces for data representation used throughout the framework.
*   **Key Files:**
    *   `message.go`: Defines `Message` interface, various message types (AI, Human, System, Tool, Chat, Function), and `StoredMessage` for persistence.
    *   `document.go`: Defines the `Document` struct for representing text data with metadata.
    *   `history.go`: Defines the `ChatHistory` interface and a `BaseChatHistory` implementation for managing conversation histories.
    *   *(Other schema files like `example.go` would be here if defined)*
*   **Design Principles:** Centralized, clear, and consistent data definitions. Promotes type safety and ease of data exchange between modules.

### `pkg/core`
*   **Purpose:** Contains fundamental utilities and core model definitions that are not specific to a single high-level module like agents or orchestration but are used across the framework.
*   **Sub-packages:**
    *   `utils/`: Houses common utility functions (e.g., `GenerateRandomString`, `ContainsString` in `utils.go`).
    *   `model/`: Contains core data models or abstract representations central to the framework's operation (e.g., `ExampleInterface`, `ExampleModel` in `model.go` as placeholders for more specific core models).
*   **Design Principles:** Provides reusable, foundational components. Keeps the root of `pkg` clean by grouping these lower-level elements.

### `pkg/agents`
*   **Purpose:** Manages all aspects related to AI agents, including their definition, execution, tools, and creation.
*   **Sub-packages:**
    *   `base/`: Defines the core `Agent` interface and the `BaseAgent` struct (`base_agent.go`). `BaseAgent` provides common fields and methods that specific agent implementations can embed and extend. It now includes `Plan` and `Execute` methods to fulfill the `Agent` interface contract.
    *   `tools/`: Defines the `Tool` interface and `BaseTool` struct (`tool.go`) for creating tools that agents can use. Also includes `ToolAgentAction` for representing an agent's decision to use a tool.
    *   `executor/`: Contains the `AgentExecutor` (`executor.go`), which is responsible for running an agent's plan-execute cycle, managing interactions with tools, and handling intermediate steps.
    *   `factory/`: Provides an `AgentFactory` (`factory.go` in `pkg/agents/factory/`) for creating agent instances. This promotes decoupling and allows for easier management of different agent types. *(Note: There was also a `pkg/agents/factory.go` at a higher level, which has been consolidated or clarified by the refactor to be within its own subdirectory for better organization if it's a distinct factory pattern implementation.)*
    *   `adapter/`: (If present and refactored) Would contain adapters for integrating agents with different external systems or protocols.
    *   `config/`: (If present and refactored) Would handle agent-specific configurations.
*   **Design Principles:** Highly modular, interface-driven, and extendable. New agent types and tools can be added by implementing the defined interfaces.

### `pkg/orchestration`
*   **Purpose:** Provides components for managing and coordinating complex sequences of tasks, workflows, and message-based communication between different parts of the framework or external services.
*   **Sub-packages:**
    *   `scheduler/`: Defines a `Scheduler` interface and an `InMemoryScheduler` implementation (`scheduler.go`) for scheduling and executing tasks at specific times or intervals.
    *   `messagebus/`: Defines a `MessageBus` interface and an `InMemoryMessageBus` implementation (`messagebus.go`) for topic-based publish/subscribe messaging.
    *   `workflow/`: Defines `Workflow` and `Task` interfaces, along with a `BaseWorkflow` implementation (`workflow.go`), for creating and executing multi-step workflows.
*   **Design Principles:** Enables complex process management, asynchronous operations, and inter-component communication in a decoupled manner.

### Other Potential `pkg` Subdirectories (based on original structure, to be refactored/confirmed)
*   `llms/`: For Large Language Model integrations.
*   `chatmodels/`: Specific interfaces/implementations for chat-based models.
*   `embeddings/`: For text embedding model integrations.
*   `prompts/`: For managing and formatting prompts.
*   `dataconnection/` (with `loaders/`, `splitters/`): For data loading and preprocessing.
*   `vectorstores/`: For interacting with vector databases.
*   `retrievers/`: For information retrieval components.
*   `memory/`: For agent memory implementations.
*   `chains/`: For implementing sequences of calls (e.g., LLMChains).
*   `config/`: For global framework configuration (distinct from agent-specific config).
*   `monitoring/` (with `log/`): For logging and monitoring framework operations.
*   `db/`: For database interaction utilities.
*   `server/`: For exposing the framework via an API.
*   `ui/`: For any UI-related components if the framework serves a frontend directly.

## 3. Design Patterns and Principles Applied

*   **Interface Segregation Principle:** Interfaces are kept lean and focused on specific functionalities (e.g., `Agent`, `Tool`).
*   **Dependency Inversion Principle:** High-level modules do not depend on low-level modules directly but on abstractions (interfaces). Factories and dependency injection (though not explicitly shown in all stubs) would further support this.
*   **Single Responsibility Principle:** Each module and, ideally, each struct/interface has a clear and single responsibility.
*   **Factory Pattern:** Used in `pkg/agents/factory` to decouple the creation of agent instances from their usage.
*   **Composition:** `BaseAgent`, `BaseTool`, `BaseWorkflow` are designed to be embedded into concrete implementations, promoting code reuse without deep inheritance hierarchies.
*   **Modularity:** The clear package structure enhances modularity, making the framework easier to understand, test, and maintain.
*   **Extensibility:** The use of interfaces and factories makes it straightforward to add new types of agents, tools, LLMs, vector stores, etc., by implementing the relevant interfaces.

## 4. Key Code Changes and Refactoring Highlights

*   **`go.mod` Module Name:** The module is `github.com/lookatitude/beluga-ai`. All internal import paths have been updated to reflect this.
*   **`BaseAgent` Interface Compliance:** The `BaseAgent` in `pkg/agents/base/base_agent.go` now includes placeholder implementations for `Plan` and `Execute` methods to satisfy the `Agent` interface. Specific agent types embedding `BaseAgent` are expected to override these methods with their custom logic.
*   **Import Path Corrections:** Numerous import paths were corrected throughout the codebase to align with the refactored structure and the correct module name.
*   **Unused Import Removal:** Unused imports were removed from various files to ensure clean compilation.
*   **Directory Structure:** New directories were created (e.g., within `pkg/core`, `pkg/agents`, `pkg/orchestration`) to house the refactored components logically.

## 5. How to Extend the Framework

### Adding a New Agent Type:
1.  Define your new agent struct in a new file (e.g., `pkg/agents/myagent/my_custom_agent.go`).
2.  Embed `base.BaseAgent` if you want to reuse its foundational capabilities.
3.  Implement the `base.Agent` interface methods (`Plan`, `Execute`, `GetInputKeys`, `GetOutputKeys`, `GetTools`, `GetName`). Pay special attention to the `Plan` and `Execute` logic for your agent.
4.  Optionally, update the `AgentFactory` in `pkg/agents/factory/factory.go` to include a case for creating your new agent type.

### Adding a New Tool:
1.  Define your new tool struct in a new file (e.g., `pkg/agents/tools/my_custom_tool.go`).
2.  Embed `tools.BaseTool` if desired.
3.  Implement the `tools.Tool` interface methods (`GetName`, `GetDescription`, `Execute`, `GetInputSchema`). The `Execute` method will contain the core logic of your tool.
4.  Make the new tool available to agents, perhaps by adding it to the list of tools provided by the `AgentFactory` or directly when an agent is instantiated.

### Adding a New LLM Integration:
1.  Create a new package under `pkg/llms/` (e.g., `pkg/llms/mynewllm/`).
2.  Define a struct that will interact with the new LLM's API.
3.  Implement a common LLM interface (to be defined or refined, e.g., `LLMCaller` with methods like `Call(prompt string) (string, error)` or `Generate(request LLMRequest) (LLMResponse, error)`).
4.  Update configuration mechanisms to allow selection and configuration of this new LLM.

## 6. Future Considerations and Potential Improvements

*   **Comprehensive Testing:** The framework currently lacks test files. Adding unit and integration tests for all modules is crucial for ensuring stability and maintainability.
*   **Error Handling:** While basic error handling is present, a more robust and consistent error handling strategy across the framework could be beneficial (e.g., custom error types, standardized error responses).
*   **Configuration Management:** A more sophisticated configuration management system (beyond the existing `viper` usage, perhaps with environment-specific files and validation) could be implemented, especially for production deployments.
*   **Dependency Injection:** While factories are used, a more comprehensive dependency injection (DI) framework or pattern could be adopted to manage dependencies between components more explicitly.
*   **Asynchronous Operations:** For components like the `InMemoryMessageBus` and `InMemoryScheduler`, the current implementations are basic. For production use, these would need to be replaced or enhanced with true asynchronous processing, worker pools, and persistence.
*   **Context Propagation:** Ensure `context.Context` is propagated correctly through all relevant calls for cancellation, deadlines, and passing request-scoped values.
*   **Logging and Monitoring:** Standardize logging further and integrate with monitoring/tracing systems for better observability.

This refactored architecture provides a solid foundation for the continued development and evolution of the Beluga AI Framework.

