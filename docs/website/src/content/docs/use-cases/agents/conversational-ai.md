---
title: Conversational AI Assistant
description: Build a personalized conversational AI with persistent 3-tier memory using Beluga AI's MemGPT-inspired architecture.
---

Traditional chatbots lose context between sessions, forcing users to repeat themselves. They cannot learn preferences, recall past interactions, or build a relationship over time. A conversational AI assistant with persistent memory solves this by maintaining three tiers of memory: core context always in the prompt, searchable conversation history, and long-term archival storage backed by vector search.

## Solution Architecture

Beluga AI implements a MemGPT-inspired 3-tier memory system:

- **Core memory**: Always present in the context window. Contains the persona definition and key facts about the user. Self-editable — the agent can update its understanding of the user over time.
- **Recall memory**: Searchable conversation history. Stores full messages and retrieves relevant past exchanges by semantic similarity.
- **Archival memory**: Long-term vector storage for facts, preferences, and knowledge extracted from conversations.

```
┌──────────────────────────────────────┐
│           Context Window             │
│                                      │
│  ┌────────────────────────────────┐  │
│  │  Core Memory (always present)  │  │
│  │  - Persona: "Helpful advisor"  │  │
│  │  - Human: "Prefers concise     │  │
│  │    answers, works in finance"  │  │
│  └────────────────────────────────┘  │
│                                      │
│  ┌────────────────────────────────┐  │
│  │  Recall Memory (recent turns)  │  │
│  │  - Last N messages             │  │
│  │  - Relevant past exchanges     │  │
│  └────────────────────────────────┘  │
│                                      │
│  ┌────────────────────────────────┐  │
│  │  Current Conversation          │  │
│  │  - User message                │  │
│  └────────────────────────────────┘  │
└──────────────────────────────────────┘
                  │
                  │ Search
                  ▼
┌──────────────────────────────────────┐
│  Archival Memory (vector store)      │
│  - Extracted facts & preferences     │
│  - Past conversation summaries       │
│  - Domain knowledge                  │
└──────────────────────────────────────┘
```

## Setting Up 3-Tier Memory

```go
package main

import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/memory"
    "github.com/lookatitude/beluga-ai/rag/embedding"
    "github.com/lookatitude/beluga-ai/rag/vectorstore"

    _ "github.com/lookatitude/beluga-ai/memory/stores/inmemory"
    _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/openai"
    _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/pgvector"
)

type ConversationAssistant struct {
    core     *memory.Core
    recall   *memory.Recall
    archival *memory.Archival
    model    llm.ChatModel
}

func NewConversationAssistant(ctx context.Context) (*ConversationAssistant, error) {
    // Core memory: always in context, self-editable
    core := memory.NewCore(memory.CoreConfig{
        PersonaLimit: 2000, // Max chars for persona block
        HumanLimit:   2000, // Max chars for human info block
        SelfEditable: true, // Agent can update its understanding
    })
    core.SetPersona("You are a helpful personal assistant. You remember " +
        "past conversations and user preferences to provide personalized help.")

    // Recall memory: searchable conversation history
    messageStore, err := memory.NewMessageStore("inmemory", nil)
    if err != nil {
        return nil, fmt.Errorf("create message store: %w", err)
    }
    recall := memory.NewRecall(messageStore)

    // Archival memory: long-term vector storage
    embedder, err := embedding.New("openai", nil)
    if err != nil {
        return nil, fmt.Errorf("create embedder: %w", err)
    }

    store, err := vectorstore.New("pgvector", nil)
    if err != nil {
        return nil, fmt.Errorf("create vector store: %w", err)
    }

    archival, err := memory.NewArchival(memory.ArchivalConfig{
        VectorStore: store,
        Embedder:    embedder,
    })
    if err != nil {
        return nil, fmt.Errorf("create archival memory: %w", err)
    }

    model, err := llm.New("openai", nil)
    if err != nil {
        return nil, fmt.Errorf("create model: %w", err)
    }

    return &ConversationAssistant{
        core:     core,
        recall:   recall,
        archival: archival,
        model:    model,
    }, nil
}
```

## Conversation Flow

Each turn assembles context from all three memory tiers, generates a response, and saves the exchange back into memory.

```go
func (ca *ConversationAssistant) Chat(ctx context.Context, userMessage string) (string, error) {
    // 1. Build context from all memory tiers
    msgs := ca.buildContext(ctx, userMessage)

    // 2. Add the current user message
    humanMsg := &schema.HumanMessage{Parts: []schema.ContentPart{
        schema.TextPart{Text: userMessage},
    }}
    msgs = append(msgs, humanMsg)

    // 3. Generate response
    resp, err := ca.model.Generate(ctx, msgs)
    if err != nil {
        return "", fmt.Errorf("generate: %w", err)
    }

    responseText := resp.Parts[0].(schema.TextPart).Text

    // 4. Save to recall memory
    if err := ca.recall.Save(ctx, humanMsg, resp); err != nil {
        return "", fmt.Errorf("save recall: %w", err)
    }

    // 5. Extract and archive important facts
    ca.archiveIfRelevant(ctx, userMessage, responseText)

    return responseText, nil
}

func (ca *ConversationAssistant) buildContext(ctx context.Context, query string) []schema.Message {
    var msgs []schema.Message

    // Core memory: always first (optimizes prompt caching)
    msgs = append(msgs, ca.core.ToMessages()...)

    // Recall memory: recent conversation history
    recent, err := ca.recall.Load(ctx, query)
    if err == nil {
        msgs = append(msgs, recent...)
    }

    // Archival memory: relevant long-term facts
    archived, err := ca.archival.Search(ctx, query, 3)
    if err == nil && len(archived) > 0 {
        var archiveContext string
        for _, doc := range archived {
            archiveContext += "- " + doc.Content + "\n"
        }
        msgs = append(msgs, &schema.SystemMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: "Relevant facts from past conversations:\n" + archiveContext},
        }})
    }

    return msgs
}
```

## Self-Updating Memory

The assistant can update its core memory as it learns about the user:

```go
func (ca *ConversationAssistant) archiveIfRelevant(ctx context.Context, userMsg, response string) {
    // Use the LLM to decide if this exchange contains important facts
    msgs := []schema.Message{
        &schema.SystemMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: "Extract any new facts about the user from this exchange. " +
                "Return a JSON array of facts, or an empty array if none."},
        }},
        &schema.HumanMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: fmt.Sprintf("User: %s\nAssistant: %s", userMsg, response)},
        }},
    }

    type Facts struct {
        Items []string `json:"items"`
    }

    structured := llm.NewStructured[Facts](ca.model)
    facts, err := structured.Generate(ctx, msgs)
    if err != nil || len(facts.Items) == 0 {
        return
    }

    // Store extracted facts in archival memory
    for _, fact := range facts.Items {
        doc := schema.Document{
            Content:  fact,
            Metadata: map[string]any{"source": "conversation", "timestamp": time.Now().Unix()},
        }
        embedding, err := ca.archival.cfg.Embedder.EmbedSingle(ctx, fact)
        if err != nil {
            continue
        }
        ca.archival.cfg.VectorStore.Add(ctx, []schema.Document{doc}, [][]float32{embedding})
    }

    // Update core memory if we learned something fundamental about the user
    for _, fact := range facts.Items {
        if isCoreFact(fact) {
            current := ca.core.GetHuman()
            ca.core.SetHuman(current + "\n- " + fact)
        }
    }
}
```

## Streaming Responses

Stream responses token by token for a natural conversational feel:

```go
func (ca *ConversationAssistant) StreamChat(ctx context.Context, userMessage string) iter.Seq2[schema.StreamChunk, error] {
    msgs := ca.buildContext(ctx, userMessage)
    msgs = append(msgs, &schema.HumanMessage{Parts: []schema.ContentPart{
        schema.TextPart{Text: userMessage},
    }})

    return ca.model.Stream(ctx, msgs)
}
```

## Multi-Session Persistence

For production deployments, use durable stores so conversations survive restarts:

```go
import (
    _ "github.com/lookatitude/beluga-ai/memory/stores/redis"
    _ "github.com/lookatitude/beluga-ai/memory/stores/postgres"
)

// Redis for recall memory (fast read/write)
messageStore, err := memory.NewMessageStore("redis", config.ProviderConfig{
    "addr": "localhost:6379",
    "prefix": "user:" + userID,
})

// PostgreSQL + pgvector for archival memory (persistent, searchable)
store, err := vectorstore.New("pgvector", config.ProviderConfig{
    "connection_string": os.Getenv("DATABASE_URL"),
    "table_name": "archival_memory",
})
```

## Production Considerations

### Context Window Management

Core memory consumes a fixed portion of the context window. Monitor and manage it:

```go
// Check core memory size before adding facts
persona := ca.core.GetPersona()
human := ca.core.GetHuman()

if len(persona)+len(human) > 3000 {
    // Summarize the human profile to fit within limits
    summarized, err := summarizeProfile(ctx, ca.model, human)
    if err == nil {
        ca.core.SetHuman(summarized)
    }
}
```

### Observability

Track memory operations, context sizes, and retrieval quality:

```go
span.SetAttributes(
    attribute.Int("memory.core_size", len(ca.core.GetPersona())+len(ca.core.GetHuman())),
    attribute.Int("memory.recall_messages", len(recent)),
    attribute.Int("memory.archival_results", len(archived)),
    attribute.Int("memory.total_context_tokens", estimateTokens(msgs)),
)
```

### Privacy and Data Retention

- Encrypt memory stores at rest (database-level encryption for PostgreSQL, TLS for Redis)
- Implement per-user memory isolation — each user has their own core, recall, and archival stores
- Provide a `Clear()` method for users to delete their data (GDPR right to erasure)
- Set TTLs on recall memory to automatically expire old conversations
- Never store PII in core memory — use the guard pipeline to screen before saving

### Scaling

- **Core memory**: In-memory per session, persisted on session end. Lightweight.
- **Recall memory**: Redis for sub-millisecond lookups. Shard by user ID.
- **Archival memory**: pgvector with HNSW index for fast approximate nearest neighbor search at scale.
- Deploy the assistant as a stateless service. All state lives in the memory stores.

## Related Resources

- [Memory System Guide](/guides/memory-system/) for detailed memory configuration
- [Building Your First Agent](/guides/first-agent/) for combining memory with planning strategies
- [Voice AI Applications](/use-cases/voice-applications/) for adding voice interfaces to the assistant
