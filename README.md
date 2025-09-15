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

The refactored Beluga-ai framework emphasizes a modular and interface-driven design with advanced features for production readiness. Key components are now organized within the `pkg` directory:

*   **`pkg/schema`:** Centralized definitions for all core data structures.
*   **`pkg/core`:** Foundational utilities, dependency injection container, and core model definitions.
*   **`pkg/llms`:** `LLM` interface, provider implementations (e.g., `openai`, `anthropic`), and `LLMProviderFactory`.
*   **`pkg/prompts`:** `PromptAdapter` interface and implementations for model-specific prompt formatting.
*   **`pkg/agents`:** Comprehensive toolkit for agent development (`base`, `tools`, `executor`, `factory`).
    *   `pkg/agents/tools/providers`: Contains implementations for various tool types (e.g., `shell_tool.go`, `gofunction_tool.go`).
*   **`pkg/memory`:** `Memory` interface, basic implementations, and `VectorStoreMemory`.
    *   `pkg/vectorstores`: `VectorStore` interface, provider implementations (e.g., `inmemory`, `pgvector`, `pinecone`), and `VectorStoreProviderFactory`.
    *   `pkg/embeddings`: `Embedder` interface and implementations (e.g., `openai`).
*   **`pkg/orchestration`:** Advanced components for managing complex task sequences.
    *   Enhanced scheduler with worker pools, retry mechanisms, and circuit breakers.
    *   `pkg/orchestration/workflow/factory`: `WorkflowProviderFactory` for different workflow engines.
*   **`pkg/config`:** Advanced configuration management with validation, environment variable support, and defaults.
*   **`pkg/monitoring`:** Comprehensive observability suite including:
    *   Structured logging with context propagation.
    *   Metrics collection and statistical analysis.
    *   Distributed tracing with span support.
    *   Health checking and alerting.

For a complete breakdown of the architecture, please refer to [Beluga_Refactored_Architecture.md](./Beluga_Refactored_Architecture.md).

## Installation

```bash
go get github.com/lookatitude/beluga-ai
```

## Examples

Detailed usage examples, including how to configure and use different providers for LLMs, VectorStores, and Tools, can be found in the `/examples` directory (to be populated as features are implemented).

## Configuration

Beluga-ai uses Viper for advanced configuration management with validation, environment variable support, and automatic defaults. Configuration can be provided via YAML files, environment variables, or programmatically.

### Configuration Sources (in order of precedence):
1. Environment variables (prefixed with `BELUGA_`)
2. Configuration files (YAML/JSON)
3. Default values

### Example Configuration File (`config.yaml`):

```yaml
# Global settings
app_name: "beluga-ai-app"
log_level: "info"
server_port: 8080

# LLM Providers
llm_providers:
  - name: "openai-gpt4"
    provider: "openai"
    model_name: "gpt-4"
    api_key: "${OPENAI_API_KEY}"  # Environment variable reference
    default_call_options:
      temperature: 0.7
      max_tokens: 1000

# Embedding Providers
embedding_providers:
  - name: "openai-embeddings"
    provider: "openai"
    model_name: "text-embedding-ada-002"
    api_key: "${OPENAI_API_KEY}"

# Vector Stores
vector_stores:
  - name: "pinecone-store"
    provider: "pinecone"
    api_key: "${PINECONE_API_KEY}"
    index_name: "beluga-index"

# Tools
tools:
  - name: "calculator"
    provider: "calculator"
    enabled: true

# Agents
agents:
  - name: "data-analyzer"
    type: "AnalyzerAgent"
    max_retries: 3
```

### Environment Variables:
```bash
export BELUGA_APP_NAME="my-beluga-app"
export BELUGA_LOG_LEVEL="debug"
export BELUGA_OPENAI_API_KEY="your-api-key-here"
export BELUGA_PINECONE_API_KEY="your-pinecone-key"
```

### Configuration Validation:
The framework automatically validates configuration on load and provides detailed error messages for missing required fields or invalid values.

## Advanced Features

### Dependency Injection
Beluga-ai includes a comprehensive dependency injection system with functional options patterns:

```go
// Create an agent factory with DI
factory, err := agents.NewAgentFactoryWithOptions(
    agents.WithConfigProvider(configProvider),
    agents.WithContainer(diContainer),
)

// Create agents using fluent builders
agent, err := agents.NewAgentBuilder(factory).
    WithName("data-analyzer").
    WithType("AnalyzerAgent").
    WithAnalysisType("comprehensive").
    Build()
```

### Asynchronous Processing
Advanced orchestration with worker pools, retry mechanisms, and circuit breakers:

```go
// Create enhanced scheduler with worker pool
scheduler := orchestration.NewEnhancedScheduler(10) // 10 workers
scheduler.Start()
defer scheduler.Stop()

// Add tasks with retry configuration
task := &orchestration.EnhancedTask{
    Task: orchestration.Task{
        ID: "data-processing",
        Execute: processData,
    },
    MaxRetries: 3,
    Timeout: 30 * time.Second,
    RequiresCircuitBreaker: true,
}

scheduler.AddEnhancedTask(task)

// Run asynchronously
stats := scheduler.RunAsync()
fmt.Printf("Processed %d tasks\n", stats.CompletedTasks)
```

### Observability & Monitoring
Comprehensive observability suite with structured logging, metrics, and tracing:

```go
// Structured logging with context
logger := monitoring.NewStructuredLogger("my-service",
    monitoring.WithJSONOutput(),
    monitoring.WithFileOutput("logs/app.log"))

ctx, span := tracer.StartSpan(ctx, "process-request")
defer tracer.FinishSpan(span)

logger.Info(ctx, "Processing request", map[string]interface{}{
    "user_id": 12345,
    "request_type": "data_analysis",
})

// Metrics collection
metrics := monitoring.NewMetricsCollector()
timer := metrics.StartTimer(ctx, "request_duration", map[string]string{
    "endpoint": "/api/process",
})
defer timer.Stop(ctx, "Request processing time")

metrics.Counter(ctx, "requests_total", "Total requests", 1, map[string]string{
    "method": "POST",
    "status": "200",
})
```

## Current Status

Beluga AI Framework has completed a comprehensive redesign following Go best practices and SOLID principles. All core packages have been refactored with:

✅ **Completed Phases:**
- **Phase 1 (Foundational):** Core, Schema, Config packages with comprehensive error handling, metrics, and validation
- **Phase 2 (Observability):** Monitoring package with OTEL tracing, metrics, logging, and health checks
- **Phase 3 (AI Components):** LLMs, ChatModels, Embeddings, Prompts, Memory, Retrievers, VectorStores with unified interfaces
- **Phase 4 (Agents):** Complete agent framework with ReAct agents, tool integration, and executor
- **Phase 5 (Orchestration):** Workflow engine with Chain, Graph, and Workflow support
- **Phase 6 (Server):** REST and MCP server implementations
- **Phase 7 (Cross-Package/Global):** Standardized package structure with iface/, config.go, metrics.go, errors.go

🔧 **Architecture Improvements:**
- Interface Segregation Principle (ISP) throughout
- Dependency Inversion Principle (DIP) with constructor injection
- Single Responsibility Principle (SRP) for focused packages
- Composition over Inheritance patterns
- Factory patterns for provider registration
- Functional options for configuration
- OpenTelemetry integration for observability
- Structured error handling with custom error types
- Comprehensive test coverage with table-driven tests

## Implemented Features

*   **LLMs (`pkg/llms`):** ✅ Unified ChatModel/LLM interfaces with OpenAI, Anthropic, Bedrock providers
*   **ChatModels (`pkg/chatmodels`):** ✅ ChatModel interface with OpenAI provider and Runnable implementation
*   **Embeddings (`pkg/embeddings`):** ✅ Embedder interface with OpenAI, Ollama providers and Runnable implementation
*   **Prompts (`pkg/prompts`):** ✅ Prompt template system with dynamic loading and rendering
*   **Memory (`pkg/memory`):** ✅ Memory interface with BufferMemory, SummaryMemory, and VectorStoreMemory
*   **Retrievers (`pkg/retrievers`):** ✅ Retriever interface with Runnable implementation and vectorstore integration
*   **VectorStores (`pkg/vectorstores`):** ✅ VectorStore interface with InMemory, PgVector, Pinecone providers
*   **Agents (`pkg/agents`):** ✅ Complete agent framework with ReAct agents, tool integration, and executor
*   **Tools (`pkg/agents/tools`):** ✅ Tool interface with Echo, Calculator, Shell, and GoFunction implementations
*   **Orchestration (`pkg/orchestration`):** ✅ Workflow engine with Chain, Graph, and Workflow support
*   **Server (`pkg/server`):** ✅ REST and MCP server implementations with streaming support
*   **Configuration Management (`pkg/config`):** ✅ Advanced configuration with validation, environment variables, and schema support
*   **Observability (`pkg/monitoring`):** ✅ Comprehensive monitoring with OTEL tracing, metrics, logging, and health checks
*   **Schema (`pkg/schema`):** ✅ Centralized data structures with validation and type safety

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


