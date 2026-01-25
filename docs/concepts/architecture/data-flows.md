# Data Flows

This document describes how data flows through different components of the Beluga AI Framework.

## RAG Query Processing Flow

```mermaid
flowchart TD
    Start([User Query]) --> LoadMemory[Load Memory Variables]
    LoadMemory --> EmbedQuery[Embed Query]
    EmbedQuery --> Search[Vector Store Similarity Search]
    Search --> Retrieve[Retrieve Relevant Documents]
    Retrieve --> BuildContext[Build Context from Documents]
    BuildContext --> CombineContext[Combine with Memory Context]
    CombineContext --> CreateMessages[Create LLM Messages]
    CreateMessages --> Generate[Generate Response]
    Generate --> SaveMemory[Save to Memory]
    SaveMemory --> End([Final Answer])
    
    style Start fill:#e1f5ff
    style End fill:#c8e6c9
```

## Agent Execution Flow

```mermaid
flowchart TD
    Start([Agent Input]) --> LoadMemory[Load Memory]
    LoadMemory --> Plan[Plan Action]
    Plan --> Decision{Action Type}
    Decision -->|Tool| ExecuteTool[Execute Tool]
    Decision -->|LLM| CallLLM[Call LLM]
    Decision -->|Finish| Finish[Return Final Answer]
    ExecuteTool --> Observe[Observe Result]
    CallLLM --> Observe
    Observe --> UpdateMemory[Update Memory]
    UpdateMemory --> Plan
    Finish --> SaveMemory[Save Context]
    SaveMemory --> End([Agent Output])
    
    style Start fill:#e1f5ff
    style End fill:#c8e6c9
    style Decision fill:#fff9c4
```

## Memory Operations Flow

```mermaid
flowchart TD
    Start([Memory Operation]) --> Type{Operation Type}
    Type -->|Load| LoadVars[Load Memory Variables]
    Type -->|Save| SaveCtx[Save Context]
    Type -->|Clear| ClearMem[Clear Memory]
    
    LoadVars --> BufferMem{Memory Type}
    BufferMem -->|Buffer| GetHistory[Get Chat History]
    BufferMem -->|Vector| SemanticSearch[Semantic Search]
    GetHistory --> Format[Format History]
    SemanticSearch --> Format
    Format --> ReturnVars[Return Variables]
    
    SaveCtx --> SaveType{Memory Type}
    SaveType -->|Buffer| AddMessages[Add Messages]
    SaveType -->|Vector| EmbedSave[Embed and Save]
    AddMessages --> Done[Operation Complete]
    EmbedSave --> Done
    
    ClearMem --> ClearType{Memory Type}
    ClearType -->|Buffer| ClearHistory[Clear History]
    ClearType -->|Vector| ClearVectors[Clear Vectors]
    ClearHistory --> Done
    ClearVectors --> Done
    
    ReturnVars --> End([Memory Variables])
    Done --> End
    
    style Start fill:#e1f5ff
    style End fill:#c8e6c9
    style Type fill:#fff9c4
```

## Orchestration Workflow Flow

```mermaid
flowchart TD
    Start([Workflow Input]) --> Init[Initialize Workflow]
    Init --> ExecuteStep[Execute Step]
    ExecuteStep --> CheckCondition{Condition Check}
    CheckCondition -->|True| PathA[Execute Path A]
    CheckCondition -->|False| PathB[Execute Path B]
    PathA --> NextStep{More Steps?}
    PathB --> NextStep
    NextStep -->|Yes| ExecuteStep
    NextStep -->|No| Aggregate[Aggregate Results]
    Aggregate --> End([Workflow Output])
    
    style Start fill:#e1f5ff
    style End fill:#c8e6c9
    style CheckCondition fill:#fff9c4
```

## Multi-Agent Coordination Flow

```mermaid
flowchart TD
    Start([Task Input]) --> Coordinator[Coordinator Agent]
    Coordinator --> Delegate[Delegate to Agents]
    Delegate --> Agent1[Agent 1]
    Delegate --> Agent2[Agent 2]
    Delegate --> Agent3[Agent 3]
    Agent1 --> Publish1[Publish Result]
    Agent2 --> Publish2[Publish Result]
    Agent3 --> Publish3[Publish Result]
    Publish1 --> MessageBus[Message Bus]
    Publish2 --> MessageBus
    Publish3 --> MessageBus
    MessageBus --> Subscribe[Subscribers Receive]
    Subscribe --> Process[Process Messages]
    Process --> Aggregate[Aggregate Results]
    Aggregate --> End([Final Result])
    
    style Start fill:#e1f5ff
    style End fill:#c8e6c9
    style MessageBus fill:#fff9c4
```

## Embedding Generation Flow

```mermaid
flowchart TD
    Start([Text Input]) --> Validate[Validate Input]
    Validate --> Batch{Batch Size}
    Batch -->|Single| EmbedSingle[Embed Single Text]
    Batch -->|Multiple| EmbedBatch[Embed Batch]
    EmbedSingle --> Vectorize[Generate Embedding Vector]
    EmbedBatch --> Vectorize
    Vectorize --> Normalize[Normalize Vector]
    Normalize --> Return[Return Embedding]
    Return --> End([Embedding Vector])
    
    style Start fill:#e1f5ff
    style End fill:#c8e6c9
```

## Related Documentation

- [Component Diagrams](./component-diagrams.md) - Component structure
- [Sequence Diagrams](./sequences.md) - Interaction sequences
- [Architecture Overview](../../architecture.md) - Complete architecture
