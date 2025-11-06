# Memory Concepts

This document explains how memory works in Beluga AI, including memory types, conversation history management, and persistence strategies.

## Memory Types

Beluga AI supports multiple memory types for different use cases.

### Buffer Memory

Stores all conversation messages:

```go
mem, err := memory.NewMemory(memory.MemoryTypeBuffer)
```

**Use when:**
- Need complete conversation history
- Short conversations
- Full context required

### Window Memory

Stores only the last N messages:

```go
mem, err := memory.NewMemory(
    memory.MemoryTypeWindow,
    memory.WithWindowSize(10),
)
```

**Use when:**
- Long conversations
- Need to limit context size
- Recent context is most important

### Summary Memory

Summarizes old messages, keeps recent ones:

```go
mem, err := memory.NewMemory(memory.MemoryTypeSummary)
```

**Use when:**
- Very long conversations
- Need to preserve key information
- Balance between context and size

### Vector Store Memory

Semantic search over conversation history:

```go
mem, err := memory.NewMemory(
    memory.MemoryTypeVectorStore,
    memory.WithVectorStore(vectorStore),
    memory.WithEmbedder(embedder),
)
```

**Use when:**
- Need semantic search
- Large conversation histories
- Finding relevant past context

## Conversation History

### ChatMessageHistory Interface

```go
type ChatMessageHistory interface {
    AddMessage(ctx context.Context, message Message) error
    AddUserMessage(ctx context.Context, content string) error
    AddAIMessage(ctx context.Context, content string) error
    GetMessages(ctx context.Context) ([]Message, error)
    Clear(ctx context.Context) error
}
```

### Using Message History

```go
history := memory.NewChatMessageHistory()

history.AddUserMessage(ctx, "Hello!")
history.AddAIMessage(ctx, "Hi there!")

messages, _ := history.GetMessages(ctx)
```

## Memory Operations

### Save Context

```go
inputs := map[string]any{
    "input": "user message",
}
outputs := map[string]any{
    "output": "ai response",
}

mem.SaveContext(ctx, inputs, outputs)
```

### Load Memory Variables

```go
vars, err := mem.LoadMemoryVariables(ctx, map[string]any{})
history := vars["history"].(string)
```

### Clear Memory

```go
mem.Clear(ctx)
```

## Memory with Agents

### Attaching Memory to Agents

```go
mem, _ := memory.NewMemory(memory.MemoryTypeBuffer)

agent.Initialize(map[string]interface{}{
    "memory": mem,
})
```

### Memory in Conversations

Agents automatically:
1. Load memory before processing
2. Save context after responding
3. Maintain conversation history

## Context Window Management

### Token Limits

Monitor token usage to stay within limits:

```go
// Estimate tokens
tokens := estimateTokens(messages)

if tokens > maxTokens {
    // Use window or summary memory
}
```

### Window Sizing

Choose window size based on:
- Model context window
- Average message length
- Desired history depth

## Persistence Strategies

### In-Memory

Default for development:
- Fast access
- Lost on restart
- No persistence

### File-Based

Simple persistence:

```go
// Save
data, _ := json.Marshal(memoryVars)
os.WriteFile("memory.json", data, 0644)

// Load
data, _ := os.ReadFile("memory.json")
json.Unmarshal(data, &memoryVars)
```

### Database

Production persistence:

```go
// Save to database
db.SaveMemory(userID, memoryVars)

// Load from database
memoryVars := db.LoadMemory(userID)
```

## Best Practices

1. **Choose appropriate type**: Select memory type based on use case
2. **Monitor size**: Watch memory growth in long conversations
3. **Implement persistence**: Save important conversations
4. **Clear when needed**: Clear memory for new sessions
5. **Use summaries**: Use summary memory for very long conversations

## Related Concepts

- [Core Concepts](./core.md) - Foundation patterns
- [Agent Concepts](./agents.md) - Agent memory integration
- [RAG Concepts](./rag.md) - Vector store memory

---

**Next:** Learn about [RAG Concepts](./rag.md) or [Orchestration Concepts](./orchestration.md)

