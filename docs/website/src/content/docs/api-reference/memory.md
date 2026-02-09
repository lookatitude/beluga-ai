---
title: Memory Package API
description: API documentation for the MemGPT-inspired 3-tier memory system.
---

```go
import "github.com/lookatitude/beluga-ai/memory"
```

Package memory provides the MemGPT-inspired 3-tier memory system: Core (always-in-context), Recall (searchable history), and Archival (vector-based long-term storage), plus Graph memory for entity-relationships.

## Quick Start

```go
// Composite memory with all tiers
mem := memory.NewComposite(
    memory.WithCore(core),
    memory.WithRecall(recall),
    memory.WithArchival(archival),
    memory.WithGraph(graphStore),
)

// Save conversation turn
err := mem.Save(ctx, inputMsg, outputMsg)

// Load relevant context
msgs, err := mem.Load(ctx, "What did we discuss about Go?")

// Search long-term storage
docs, err := mem.Search(ctx, "Go best practices", 10)
```

## Memory Interface

```go
type Memory interface {
    Save(ctx context.Context, input, output schema.Message) error
    Load(ctx context.Context, query string) ([]schema.Message, error)
    Search(ctx context.Context, query string, k int) ([]schema.Document, error)
    Clear(ctx context.Context) error
}
```

## Core Memory

Always-in-context persona and human blocks:

```go
core := memory.NewCore(memory.CoreConfig{
    PersonaLimit:  2000,
    HumanLimit:    2000,
    SelfEditable:  true,
})

// Set persona
core.SetPersona("You are a helpful Go programming assistant")

// Set human context
core.SetHuman("The user is learning Go and prefers detailed explanations")

// Convert to messages for LLM context
msgs := core.ToMessages()
```

## Recall Memory

Searchable conversation history:

```go
recall := memory.NewRecall(messageStore)

// Save turns
recall.Save(ctx, inputMsg, outputMsg)

// Load all messages
msgs, err := recall.Load(ctx, "")

// Search messages
msgs, err := recall.Load(ctx, "Go concurrency")
```

## Archival Memory

Vector-based long-term storage:

```go
archival, err := memory.NewArchival(memory.ArchivalConfig{
    VectorStore: vectorStore,
    Embedder:    embedder,
})

// Save messages (auto-embedded)
archival.Save(ctx, inputMsg, outputMsg)

// Semantic search
docs, err := archival.Search(ctx, "Go error handling patterns", 10)
```

## Graph Memory

Entity-relationship storage:

```go
type GraphStore interface {
    AddEntity(ctx context.Context, entity memory.Entity) error
    AddRelation(ctx context.Context, from, to, relation string, props map[string]any) error
    Query(ctx context.Context, query string) ([]memory.GraphResult, error)
    Neighbors(ctx context.Context, entityID string, depth int) ([]memory.Entity, []memory.Relation, error)
}

// Add entities
store.AddEntity(ctx, memory.Entity{
    ID:   "user-123",
    Type: "person",
    Properties: map[string]any{
        "name": "Alice",
        "role": "engineer",
    },
})

// Add relationships
store.AddRelation(ctx, "user-123", "project-456", "works_on", nil)

// Query relationships
results, err := store.Query(ctx, "MATCH (p:person)-[:works_on]->(pr:project) RETURN p, pr")

// Get neighbors
entities, relations, err := store.Neighbors(ctx, "user-123", 2)
```

## Composite Memory

Combine all tiers:

```go
mem := memory.NewComposite(
    memory.WithCore(coreMemory),
    memory.WithRecall(recallMemory),
    memory.WithArchival(archivalMemory),
    memory.WithGraph(graphStore),
)

// Access individual tiers
core := mem.Core()
recall := mem.Recall()
archival := mem.Archival()
graph := mem.Graph()
```

## Middleware

Add logging, metrics, etc.:

```go
wrapped := memory.ApplyMiddleware(mem,
    memory.WithHooks(memory.Hooks{
        BeforeSave: func(ctx context.Context, input, output schema.Message) error {
            log.Printf("Saving turn")
            return nil
        },
        AfterSearch: func(ctx context.Context, query string, k int, docs []schema.Document, err error) {
            log.Printf("Found %d docs for query: %s", len(docs), query)
        },
    }),
)
```

## Message Stores

Backend for recall memory:

```go
type MessageStore interface {
    Append(ctx context.Context, msg schema.Message) error
    Search(ctx context.Context, query string, k int) ([]schema.Message, error)
    All(ctx context.Context) ([]schema.Message, error)
    Clear(ctx context.Context) error
}
```

Implementations: `inmemory`, `redis`, `postgres`, `sqlite`, etc.

## Example: Full Memory Setup

```go
// Core memory
core := memory.NewCore(memory.CoreConfig{
    SelfEditable: true,
})
core.SetPersona("Expert Go developer")

// Recall memory
recallStore := inmemory.NewMessageStore()
recall := memory.NewRecall(recallStore)

// Archival memory
embedder, _ := embedding.New("openai", embedCfg)
vectorStore, _ := vectorstore.New("inmemory", storeCfg)
archival, _ := memory.NewArchival(memory.ArchivalConfig{
    VectorStore: vectorStore,
    Embedder:    embedder,
})

// Composite
mem := memory.NewComposite(
    memory.WithCore(core),
    memory.WithRecall(recall),
    memory.WithArchival(archival),
)

// Use in agent
agent := agent.New("assistant",
    agent.WithMemory(mem),
    agent.WithLLM(model),
)
```

## See Also

- [Agent Package](./agent.md) for memory integration
- [RAG Package](./rag.md) for vector storage
- [Schema Package](./schema.md) for message types
