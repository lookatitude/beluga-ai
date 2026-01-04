# Sequence Diagrams

This document provides sequence diagrams showing interactions between components in the Beluga AI Framework.

## Agent Planning and Execution

```mermaid
sequenceDiagram
    participant User
    participant Agent
    participant Planner
    participant LLM
    participant Executor
    participant Tools
    participant Memory
    
    User->>Agent: Invoke(input)
    Agent->>Memory: LoadMemoryVariables()
    Memory-->>Agent: context
    
    Agent->>Planner: Plan(steps, inputs)
    Planner->>LLM: Generate(prompt)
    LLM-->>Planner: response
    Planner->>Planner: Parse response
    Planner-->>Agent: action, finish
    
    alt Action is Tool
        Agent->>Executor: ExecutePlan(plan)
        Executor->>Tools: Execute(tool, input)
        Tools-->>Executor: result
        Executor-->>Agent: observation
        Agent->>Planner: Plan(updated steps)
    else Action is Finish
        Agent->>Memory: SaveContext()
        Agent-->>User: final answer
    end
```

## RAG Pipeline Execution

```mermaid
sequenceDiagram
    participant User
    participant App
    participant Retriever
    participant VectorStore
    participant Embedder
    participant LLM
    participant Memory
    
    User->>App: Query
    App->>Memory: LoadMemoryVariables()
    Memory-->>App: conversation history
    
    App->>Retriever: GetRelevantDocuments(query)
    Retriever->>Embedder: EmbedQuery(query)
    Embedder-->>Retriever: query embedding
    Retriever->>VectorStore: SimilaritySearch(embedding, k)
    VectorStore-->>Retriever: relevant documents
    Retriever-->>App: documents
    
    App->>App: Build context (memory + documents)
    App->>LLM: Generate(messages with context)
    LLM-->>App: response
    
    App->>Memory: SaveContext(query, response)
    App-->>User: final answer
```

## Multi-Agent Coordination

```mermaid
sequenceDiagram
    participant Coordinator
    participant Agent1
    participant Agent2
    participant Agent3
    participant MessageBus
    participant Scheduler
    
    Coordinator->>Scheduler: Schedule(task1, Agent1)
    Scheduler->>Agent1: Execute(task1)
    Agent1->>Agent1: Process
    Agent1->>MessageBus: Publish(result1, topic1)
    Agent1-->>Scheduler: result1
    Scheduler-->>Coordinator: result1
    
    MessageBus->>Agent2: Notify(topic1, result1)
    Coordinator->>Scheduler: Schedule(task2, Agent2)
    Scheduler->>Agent2: Execute(task2)
    Agent2->>Agent2: Process
    Agent2->>MessageBus: Publish(result2, topic2)
    Agent2-->>Scheduler: result2
    Scheduler-->>Coordinator: result2
    
    MessageBus->>Agent3: Notify(topic2, result2)
    Coordinator->>Scheduler: Schedule(task3, Agent3)
    Scheduler->>Agent3: Execute(task3)
    Agent3->>Agent3: Process
    Agent3-->>Scheduler: result3
    Scheduler-->>Coordinator: final result
```

## Tool Execution Flow

```mermaid
sequenceDiagram
    participant Agent
    participant Executor
    participant ToolRegistry
    participant Tool
    participant ExternalService
    
    Agent->>Executor: ExecutePlan(plan with tool action)
    Executor->>ToolRegistry: GetTool(toolName)
    ToolRegistry-->>Executor: tool
    
    Executor->>Tool: Execute(input)
    Tool->>ExternalService: Call API / Execute
    ExternalService-->>Tool: result
    Tool-->>Executor: tool result
    
    Executor->>Executor: Format result
    Executor-->>Agent: observation
    
    Agent->>Agent: Process observation
    Agent->>Agent: Continue or finish
```

## Chain Execution Flow

```mermaid
sequenceDiagram
    participant User
    participant Chain
    participant Step1
    participant Step2
    participant Step3
    participant Memory
    
    User->>Chain: Invoke(input)
    Chain->>Memory: LoadMemoryVariables()
    Memory-->>Chain: context
    
    Chain->>Step1: Invoke(input + context)
    Step1->>Step1: Process
    Step1-->>Chain: output1
    
    Chain->>Step2: Invoke(output1)
    Step2->>Step2: Process
    Step2-->>Chain: output2
    
    Chain->>Step3: Invoke(output2)
    Step3->>Step3: Process
    Step3-->>Chain: output3
    
    Chain->>Memory: SaveContext(input, output3)
    Chain-->>User: final output
```

## Memory Save and Load Flow

```mermaid
sequenceDiagram
    participant Agent
    participant Memory
    participant ChatHistory
    participant VectorStore
    participant Embedder
    
    Note over Agent,Embedder: Save Context Flow
    Agent->>Memory: SaveContext(inputs, outputs)
    Memory->>ChatHistory: AddMessages(input, output)
    ChatHistory-->>Memory: saved
    
    alt Vector Store Memory
        Memory->>Memory: Create document
        Memory->>Embedder: EmbedDocuments(text)
        Embedder-->>Memory: embeddings
        Memory->>VectorStore: AddDocuments(docs, embeddings)
        VectorStore-->>Memory: saved
    end
    
    Note over Agent,Embedder: Load Context Flow
    Agent->>Memory: LoadMemoryVariables(inputs)
    
    alt Buffer Memory
        Memory->>ChatHistory: GetMessages()
        ChatHistory-->>Memory: messages
        Memory->>Memory: Format messages
    else Vector Store Memory
        Memory->>Memory: Extract query from input
        Memory->>Embedder: EmbedQuery(query)
        Embedder-->>Memory: query embedding
        Memory->>VectorStore: SimilaritySearch(embedding)
        VectorStore-->>Memory: relevant docs
        Memory->>Memory: Format documents
    end
    
    Memory-->>Agent: memory variables
```

## Related Documentation

- [Component Diagrams](./component-diagrams.md) - Component structure
- [Data Flows](./data-flows.md) - Data flow through system
- [Architecture Overview](../architecture.md) - Complete architecture
