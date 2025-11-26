# Use Case 5: Conversational AI Assistant with Long-Term Memory

## Overview & Objectives

### Business Problem

Traditional chatbots lack memory of past conversations, requiring users to repeat context. They can't learn from interactions or provide personalized responses based on user history. Long-term memory is essential for building meaningful, context-aware conversational experiences.

### Solution Approach

This use case implements an advanced conversational AI assistant that:
- Maintains conversation history across sessions
- Uses semantic memory for retrieving relevant past conversations
- Summarizes long conversations to manage context
- Provides personalized responses based on user history
- Supports multiple memory backends

### Key Benefits

- **Persistent Memory**: Remembers conversations across sessions
- **Semantic Retrieval**: Finds relevant past conversations by meaning
- **Context Management**: Summarizes long conversations intelligently
- **Personalization**: Adapts responses based on user history
- **Scalable Architecture**: Handles millions of conversations

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                    User Interface                                │
│              (Web, Mobile, Chat, Voice)                         │
└────────────────────────────┬────────────────────────────────────┘
                              │
                              │ HTTP/REST
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    REST Server (pkg/server)                      │
└────────────────────────────┬────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│              Conversation Manager                                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │  1. Load     │→ │  2. Generate │→ │  3. Save      │         │
│  │  Context      │  │  Response    │  │  Context     │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
└────────────────────────────┬────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   Memory     │    │  ChatModels  │    │  VectorStores│
│  (pkg/memory)│    │  (pkg/       │    │  (pkg/       │
│              │    │  chatmodels) │    │  vectorstores)│
│  - Buffer    │    │              │    │              │
│  - Summary   │    │  - OpenAI    │    │  - PgVector  │
│  - VectorStore│   │  - Anthropic │    │  - Pinecone  │
└──────────────┘    └──────────────┘    └──────────────┘
        │                     │                     │
        └─────────────────────┼─────────────────────┘
                              │
                              ▼
              ┌────────────────────────┐
              │   Embeddings            │
              │  (pkg/embeddings)      │
              └────────────────────────┘
```

## Component Usage

### Beluga AI Packages Used

1. **pkg/chatmodels**
   - ChatModel interface for conversations
   - OpenAI provider implementation

2. **pkg/memory**
   - BufferMemory for recent conversation history
   - SummaryMemory for long conversation summarization
   - VectorStoreMemory for semantic retrieval

3. **pkg/vectorstores**
   - Store conversation embeddings
   - Semantic search for past conversations

4. **pkg/embeddings**
   - Generate embeddings for conversations
   - Support OpenAI and Ollama

5. **pkg/prompts**
   - Template-based prompt generation
   - Context injection

6. **pkg/monitoring**
   - Conversation metrics
   - Memory usage tracking

7. **pkg/server**
   - REST API for conversations

## Implementation Guide

### Step 1: Create Conversation Manager

```go
type ConversationManager struct {
    chatModel    chatmodels.ChatModel
    bufferMemory memory.Memory
    summaryMemory memory.Memory
    vectorMemory memory.Memory
    embedder     embeddings.Embedder
}

func NewConversationManager(ctx context.Context, cfg *config.Config) (*ConversationManager, error) {
    // Initialize ChatModel
    chatModel, err := chatmodels.NewChatModel(ctx, "openai",
        chatmodels.WithAPIKey(cfg.GetString("chatmodel.openai.api_key")),
        chatmodels.WithModel("gpt-4"),
    )
    if err != nil {
        return nil, err
    }

    // Initialize memory types
    bufferMemory := memory.NewBufferMemory()
    summaryMemory := memory.NewSummaryMemory(chatModel)
    
    // Initialize vector store for semantic memory
    vectorStore, err := vectorstores.NewVectorStore(ctx, "pgvector",
        vectorstores.WithProviderConfig("connection_string",
            cfg.GetString("vectorstore.postgres.connection_string")),
    )
    if err != nil {
        return nil, err
    }

    embedder, err := embeddings.NewEmbedder(ctx, "openai",
        embeddings.WithAPIKey(cfg.GetString("embeddings.openai.api_key")),
    )
    if err != nil {
        return nil, err
    }

    vectorMemory := memory.NewVectorStoreMemory(vectorStore, embedder)

    return &ConversationManager{
        chatModel:     chatModel,
        bufferMemory:  bufferMemory,
        summaryMemory: summaryMemory,
        vectorMemory:  vectorMemory,
        embedder:      embedder,
    }, nil
}
```

### Step 2: Implement Conversation Flow

```go
func (cm *ConversationManager) Chat(ctx context.Context, userID string, message string) (string, error) {
    // Load context from all memory types
    context, err := cm.loadContext(ctx, userID, message)
    if err != nil {
        return "", err
    }

    // Generate response
    messages := []schema.Message{
        schema.NewSystemMessage("You are a helpful assistant with access to conversation history."),
    }
    
    // Add context from memory
    for _, msg := range context {
        messages = append(messages, msg)
    }
    
    // Add current user message
    messages = append(messages, schema.NewHumanMessage(message))

    response, err := cm.chatModel.Generate(ctx, messages)
    if err != nil {
        return "", err
    }

    // Save to memory
    if err := cm.saveContext(ctx, userID, message, response.GetContent()); err != nil {
        log.Printf("Failed to save context: %v", err)
    }

    return response.GetContent(), nil
}

func (cm *ConversationManager) loadContext(ctx context.Context, userID string, query string) ([]schema.Message, error) {
    var messages []schema.Message

    // Load from buffer memory (recent conversations)
    bufferCtx, err := cm.bufferMemory.LoadMemoryVariables(ctx, map[string]any{
        "user_id": userID,
    })
    if err == nil {
        if history, ok := bufferCtx["history"].([]schema.Message); ok {
            messages = append(messages, history...)
        }
    }

    // Load from summary memory (long-term summary)
    summaryCtx, err := cm.summaryMemory.LoadMemoryVariables(ctx, map[string]any{
        "user_id": userID,
    })
    if err == nil {
        if summary, ok := summaryCtx["summary"].(string); ok && summary != "" {
            messages = append(messages, schema.NewSystemMessage("Previous conversation summary: "+summary))
        }
    }

    // Load from vector memory (semantically relevant past conversations)
    vectorCtx, err := cm.vectorMemory.LoadMemoryVariables(ctx, map[string]any{
        "user_id": userID,
        "query":   query,
    })
    if err == nil {
        if relevant, ok := vectorCtx["relevant"].([]schema.Message); ok {
            messages = append(messages, relevant...)
        }
    }

    return messages, nil
}

func (cm *ConversationManager) saveContext(ctx context.Context, userID string, userMsg string, aiMsg string) error {
    // Save to buffer memory
    err := cm.bufferMemory.SaveContext(ctx,
        map[string]any{"user_id": userID, "input": userMsg},
        map[string]any{"output": aiMsg},
    )

    // Save to vector memory for semantic retrieval
    conversation := fmt.Sprintf("User: %s\nAssistant: %s", userMsg, aiMsg)
    doc := schema.NewDocument(conversation, map[string]string{
        "user_id": userID,
        "timestamp": time.Now().Format(time.RFC3339),
    })
    
    // Embed and store
    embedding, err := cm.embedder.EmbedDocuments(ctx, []string{conversation})
    if err == nil {
        vectorStore := cm.vectorMemory.(*memory.VectorStoreMemory).GetVectorStore()
        vectorStore.AddDocuments(ctx, []schema.Document{doc})
    }

    // Update summary memory periodically
    if shouldSummarize(ctx, userID) {
        cm.summarizeConversation(ctx, userID)
    }

    return nil
}

func (cm *ConversationManager) summarizeConversation(ctx context.Context, userID string) error {
    // Get recent messages from buffer
    bufferCtx, err := cm.bufferMemory.LoadMemoryVariables(ctx, map[string]any{
        "user_id": userID,
    })
    if err != nil {
        return err
    }

    history, ok := bufferCtx["history"].([]schema.Message)
    if !ok || len(history) == 0 {
        return nil
    }

    // Generate summary using LLM
    prompt := "Summarize the following conversation:\n\n"
    for _, msg := range history {
        prompt += msg.GetContent() + "\n"
    }

    summary, err := cm.chatModel.Generate(ctx, []schema.Message{
        schema.NewHumanMessage(prompt),
    })
    if err != nil {
        return err
    }

    // Save summary
    return cm.summaryMemory.SaveContext(ctx,
        map[string]any{"user_id": userID},
        map[string]any{"summary": summary.GetContent()},
    )
}
```

## Workflow & Data Flow

### End-to-End Process Flow

1. **User Message Reception**
   ```
   User Message → Load Context from Memory
   ```

2. **Context Assembly**
   ```
   Buffer Memory + Summary Memory + Vector Memory → Combined Context
   ```

3. **Response Generation**
   ```
   Context + User Message → ChatModel → Response
   ```

4. **Memory Update**
   ```
   User Message + Response → Save to All Memory Types
   ```

## Observability Setup

### Metrics to Monitor

- `conversations_total`: Total conversations
- `conversation_duration_seconds`: Conversation length
- `memory_operations_total`: Memory read/write operations
- `memory_retrieval_duration_seconds`: Memory retrieval time
- `context_size_tokens`: Context size in tokens

## Configuration Examples

### Complete YAML Configuration

```yaml
# config.yaml
app:
  name: "conversational-assistant"
  version: "1.0.0"

chatmodel:
  provider: "openai"
  openai:
    api_key: "${OPENAI_API_KEY}"
    model: "gpt-4"
    temperature: 0.7

memory:
  buffer:
    enabled: true
    max_messages: 50
  summary:
    enabled: true
    summarize_interval: 20  # messages
  vectorstore:
    enabled: true
    provider: "pgvector"
    connection_string: "${POSTGRES_CONNECTION_STRING}"
    search_k: 5

embeddings:
  provider: "openai"
  openai:
    api_key: "${OPENAI_API_KEY}"
    model: "text-embedding-ada-002"

server:
  host: "0.0.0.0"
  port: 8080
```

## Deployment Considerations

### Production Requirements

- **Database**: PostgreSQL with pgvector for vector storage
- **Compute**: Sufficient CPU for embedding generation
- **Storage**: Large capacity for conversation history
- **Memory**: RAM for in-memory buffer operations

## Testing Strategy

### Unit Tests

```go
func TestConversationManager(t *testing.T) {
    cm := createTestConversationManager(t)
    
    response, err := cm.Chat(context.Background(), "user1", "Hello")
    require.NoError(t, err)
    assert.NotEmpty(t, response)
    
    // Test context retention
    response2, err := cm.Chat(context.Background(), "user1", "What did I say?")
    require.NoError(t, err)
    assert.Contains(t, response2, "Hello")
}
```

## Troubleshooting Guide

### Common Issues

1. **Context Too Large**
   - Use summary memory
   - Reduce buffer size
   - Implement context truncation

2. **Slow Memory Retrieval**
   - Optimize vector store queries
   - Cache frequent retrievals
   - Use faster embedding models

3. **Memory Not Persisting**
   - Check database connectivity
   - Verify memory save operations
   - Monitor error logs

## Conclusion

This Conversational AI Assistant demonstrates Beluga AI's capabilities in building memory-aware conversational systems. The architecture showcases:

- **Multi-Memory Architecture**: Buffer, summary, and vector memory
- **Semantic Retrieval**: Find relevant past conversations
- **Context Management**: Intelligent conversation summarization
- **Personalization**: User-specific conversation history

The system can be extended with:
- Multi-user conversation support
- Emotion detection and response
- Multi-language support
- Voice interface integration
- Advanced personalization

