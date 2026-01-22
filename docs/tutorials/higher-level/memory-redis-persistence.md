# Redis Persistence for Agents

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this tutorial, we'll implement robust, persistent memory for your agents using Redis, ensuring conversations survive service restarts and scale across multiple instances.

## Learning Objectives
- ✅ Implement `ChatMessageHistory` with Redis
- ✅ Serialize/Deserialize messages
- ✅ Manage session TTLs
- ✅ Integrate with `BufferMemory`

## Introduction
Welcome, colleague! In-memory history is great for demos, but for production, your agents need to remember users even if the server reboots. Let's wire up Redis to give our agents a long-term memory.

## Prerequisites

- [Memory Management](../../getting-started/05-memory-management.md)
- Redis instance running

## Step 1: The Redis History Struct

(As previewed in Memory Management, but detailed here).
```go
package main

import (
    "context"
    "encoding/json"
    "github.com/redis/go-redis/v9"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

type RedisHistory struct {
    client *redis.Client
    key    string
}
```

## Step 2: Serialization Logic

This is critical. You must preserve the `Role` (User/AI/System/Tool).
```go
type StoredMessage struct {
    Role    string `json:"role"`
    Content string `json:"content"`
    // Add ToolCallID, Name, etc. if needed
}

func (h *RedisHistory) AddMessage(ctx context.Context, msg schema.Message) error {
    stored := StoredMessage{
        Role:    string(msg.GetRole()),
        Content: msg.GetContent(),
    }
    
    data, _ := json.Marshal(stored)
    
    // RPUSH to append to list
    return h.client.RPush(ctx, h.key, data).Err()
}
```

## Step 3: Deserialization Logic

When reading back, recreate the correct Beluga struct.
```go
func (h *RedisHistory) GetMessages(ctx context.Context) ([]schema.Message, error) {
    // LRANGE 0 -1 gets all
    raws, _ := h.client.LRange(ctx, h.key, 0, -1).Result()
    
    var messages []schema.Message
    for _, raw := range raws {
        var stored StoredMessage
        json.Unmarshal([]byte(raw), &stored)

        
        switch schema.Role(stored.Role) \{
        case schema.RoleHuman:
            messages = append(messages, schema.NewHumanMessage(stored.Content))
        case schema.RoleAssistant:
            messages = append(messages, schema.NewAIMessage(stored.Content))
        // ... handle others
        }
    }
    return messages, nil
}
```

## Step 4: Integration with Agent
```go
func main() {
    sessionID := "user-123-session"
    history := NewRedisHistory(client, sessionID)
    
    mem := memory.NewChatMessageBufferMemory(history)

    
    agent.Initialize(map[string]any{"memory": mem})
}
```

## Verification

1. Start agent. Chat "My name is Alice".
2. Restart agent (kill process).
3. Chat "What is my name?".
4. Agent should answer "Alice".

## Next Steps

- **[Summary & Window Memory Patterns](./memory-summary-window-patterns.md)** - Optimize storage
- **[Vector Store Memory](../../getting-started/05-memory-management.md)** - Long-term recall
