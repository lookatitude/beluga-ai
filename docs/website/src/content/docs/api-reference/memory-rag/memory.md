---
title: "Memory Package"
description: "MemGPT-inspired 3-tier memory: Core, Recall, Archival, graph memory, composite"
---

```go
import "github.com/lookatitude/beluga-ai/memory"
```

Package memory provides the MemGPT-inspired 3-tier memory system for Beluga AI agents.

The memory system implements the MemGPT model with three tiers, each serving
a distinct role in agent cognition:

- Core: always-in-context persona and human blocks (editable by the agent)
- Recall: searchable conversation history (message-level persistence)
- Archival: vector-based long-term storage (embedding + retrieval)

Additionally, a graph memory tier provides entity-relationship storage for
structured knowledge representation via the `GraphStore` interface.

## Memory Interface

The primary interface is `Memory`, which all tiers implement:

```go
type Memory interface {
    Save(ctx context.Context, input, output schema.Message) error
    Load(ctx context.Context, query string) ([]schema.Message, error)
    Search(ctx context.Context, query string, k int) ([]schema.Document, error)
    Clear(ctx context.Context) error
}
```

## Registry Pattern

The package follows Beluga's standard registry pattern. Providers register
via init() and are instantiated with `New`:

```go
mem, err := memory.New("composite", cfg)
if err != nil {
    log.Fatal(err)
}
err = mem.Save(ctx, input, output)
msgs, err := mem.Load(ctx, "search query")
```

Built-in provider names: "core", "recall", "archival", "composite".
Use `List` to discover all registered providers.

## Core Memory

`Core` holds persona and human text blocks that are always included in
the LLM context window. These blocks are small, high-value information
the agent needs constant access to. The agent can self-edit these blocks
if `CoreConfig`.SelfEditable is true:

```go
core := memory.NewCore(memory.CoreConfig{
    PersonaLimit: 2000,
    HumanLimit:   2000,
    SelfEditable: true,
})
err := core.SetPersona("I am a helpful assistant")
msgs := core.ToMessages() // returns system messages for LLM context
```

## Recall Memory

`Recall` stores searchable conversation history via a `MessageStore` backend.
Every message exchanged during agent interactions is persisted:

```go
store := inmemory.NewMessageStore()
recall := memory.NewRecall(store)
err := recall.Save(ctx, userMsg, aiMsg)
history, err := recall.Load(ctx, "previous topic")
```

## Archival Memory

`Archival` provides long-term storage backed by vector embeddings, enabling
semantic search over historical content. It requires a vector store and
an embedder:

```go
archival, err := memory.NewArchival(memory.ArchivalConfig{
    VectorStore: vs,
    Embedder:    emb,
})
docs, err := archival.Search(ctx, "relevant topic", 10)
```

## Composite Memory

`CompositeMemory` combines all tiers into a unified `Memory` implementation.
Each tier is optional — only configured tiers participate in operations:

```go
mem := memory.NewComposite(
    memory.WithCore(core),
    memory.WithRecall(recall),
    memory.WithArchival(archival),
    memory.WithGraph(graphStore),
)
```

## Graph Memory

The `GraphStore` interface provides entity-relationship storage using
`Entity` and `Relation` types. Graph stores support adding entities,
creating relations, executing queries, and neighbor traversal.

## Middleware and Hooks

Memory operations can be wrapped with `Middleware` for cross-cutting concerns
and observed via `Hooks` callbacks:

```go
hooked := memory.ApplyMiddleware(mem, memory.WithHooks(memory.Hooks{
    BeforeSave: func(ctx context.Context, input, output schema.Message) error {
        log.Println("saving memory")
        return nil
    },
}))
```

Multiple hooks are merged with `ComposeHooks`. For Before* hooks and OnError,
the first error returned short-circuits the chain.

## Store Providers

Message and graph store backends are in sub-packages under memory/stores/:

- memory/stores/inmemory — in-memory (development/testing)
- memory/stores/redis — Redis sorted set
- memory/stores/postgres — PostgreSQL table
- memory/stores/sqlite — SQLite table (pure Go, no CGO)
- memory/stores/mongodb — MongoDB collection
- memory/stores/neo4j — Neo4j graph database
- memory/stores/memgraph — Memgraph graph database
- memory/stores/dragonfly — DragonflyDB (Redis-compatible)
