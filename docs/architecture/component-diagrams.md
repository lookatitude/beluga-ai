# Component Diagrams

This document provides detailed component diagrams for the Beluga AI Framework architecture.

## Package Structure

```mermaid
graph TB
    subgraph "Core Packages"
        Schema[schema]
        Core[core]
        Config[config]
        Monitoring[monitoring]
    end
    
    subgraph "AI Packages"
        LLMs[llms]
        ChatModels[chatmodels]
        Embeddings[embeddings]
        Prompts[prompts]
    end
    
    subgraph "Storage Packages"
        VectorStores[vectorstores]
        Memory[memory]
        Retrievers[retrievers]
    end
    
    subgraph "Agent Packages"
        Agents[agents]
        Tools[tools]
    end
    
    subgraph "Orchestration Packages"
        Orchestration[orchestration]
        Scheduler[scheduler]
        MessageBus[messagebus]
    end
    
    subgraph "Infrastructure Packages"
        Server[server]
    end
    
    Schema --> Core
    Config --> Core
    Monitoring --> Core
    
    LLMs --> Schema
    LLMs --> Config
    LLMs --> Monitoring
    
    ChatModels --> LLMs
    ChatModels --> Prompts
    
    Embeddings --> Schema
    Embeddings --> Config
    
    VectorStores --> Embeddings
    VectorStores --> Schema
    
    Memory --> VectorStores
    Memory --> Schema
    
    Retrievers --> VectorStores
    Retrievers --> Embeddings
    
    Agents --> LLMs
    Agents --> Tools
    Agents --> Memory
    
    Orchestration --> Agents
    Orchestration --> Schema
    
    Server --> Agents
    Server --> Orchestration
```

## Interface Hierarchy

```mermaid
graph TB
    subgraph "Core Interfaces"
        Runnable[core.Runnable]
        Retriever[core.Retriever]
    end
    
    subgraph "LLM Interfaces"
        LLM[llms.LLM]
        ChatModel[llms.ChatModel]
    end
    
    subgraph "Agent Interfaces"
        Agent[agents.Agent]
        CompositeAgent[agents.CompositeAgent]
        Planner[agents.Planner]
        Executor[agents.Executor]
    end
    
    subgraph "Memory Interfaces"
        Memory[memory.Memory]
        ChatHistory[memory.ChatHistory]
    end
    
    subgraph "Vector Store Interfaces"
        VectorStore[vectorstores.VectorStore]
    end
    
    subgraph "Orchestration Interfaces"
        Chain[orchestration.Chain]
        Workflow[orchestration.Workflow]
    end
    
    Agent --> Planner
    CompositeAgent --> Agent
    CompositeAgent --> Memory
    CompositeAgent --> Runnable
    
    Chain --> Runnable
    Workflow --> Runnable
    
    Retriever --> VectorStore
    Retriever --> Runnable
```

## Implementation Relationships

```mermaid
graph TB
    subgraph "Base Implementations"
        BaseAgent[agents.BaseAgent]
        BaseTool[agents.BaseTool]
        BaseWorkflow[orchestration.BaseWorkflow]
    end
    
    subgraph "Provider Implementations"
        OpenAILLM[llms.OpenAI]
        AnthropicLLM[llms.Anthropic]
        ReActAgent[agents.ReActAgent]
    end
    
    subgraph "Memory Implementations"
        BufferMemory[memory.BufferMemory]
        VectorMemory[memory.VectorMemory]
    end
    
    subgraph "Vector Store Implementations"
        InMemoryStore[vectorstores.InMemory]
        PgVectorStore[vectorstores.PgVector]
    end
    
    BaseAgent --> Agent
    ReActAgent --> BaseAgent
    
    BaseTool --> Tool
    BaseWorkflow --> Workflow
    
    OpenAILLM --> LLM
    AnthropicLLM --> LLM
    
    BufferMemory --> Memory
    VectorMemory --> Memory
    
    InMemoryStore --> VectorStore
    PgVectorStore --> VectorStore
```

## Factory Pattern Structure

```mermaid
graph TB
    subgraph "Factory Interfaces"
        LLMFactory[llms.Factory]
        EmbeddingFactory[embeddings.Factory]
        VectorStoreFactory[vectorstores.Factory]
        AgentFactory[agents.AgentFactory]
    end
    
    subgraph "Global Registries"
        LLMRegistry[llms.GlobalRegistry]
        EmbeddingRegistry[embeddings.GlobalRegistry]
        VectorStoreRegistry[vectorstores.GlobalRegistry]
        AgentRegistry[agents.AgentRegistry]
    end
    
    subgraph "Providers"
        OpenAIProvider[OpenAI Provider]
        AnthropicProvider[Anthropic Provider]
        CustomProvider[Custom Provider]
    end
    
    LLMFactory --> LLMRegistry
    EmbeddingFactory --> EmbeddingRegistry
    VectorStoreFactory --> VectorStoreRegistry
    AgentFactory --> AgentRegistry
    
    LLMRegistry --> OpenAIProvider
    LLMRegistry --> AnthropicProvider
    LLMRegistry --> CustomProvider
```

## Related Documentation

- [Data Flows](./data-flows.md) - Data flow through the system
- [Sequence Diagrams](./sequences.md) - Interaction sequences
- [Architecture Overview](../architecture.md) - Complete architecture documentation
