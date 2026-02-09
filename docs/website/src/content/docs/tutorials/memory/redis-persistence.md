---
title: Redis Memory Persistence
description: Implement persistent conversation memory with Redis so agents remember users across service restarts.
---

In-memory conversation history is suitable for development, but production agents need to remember users across process restarts and scale across multiple service instances. Redis provides fast, persistent key-value storage that is well suited for conversation history with configurable TTLs for session management.

## What You Will Build

A Redis-backed conversation history that serializes and deserializes typed messages, supports session TTLs, and integrates with the Beluga AI message system.

## Prerequisites

- A running Redis instance
- Understanding of [Multi-turn Conversations](/tutorials/foundation/multiturn-conversations)

## Step 1: Define the Redis History Store

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
    "github.com/lookatitude/beluga-ai/schema"
)

// RedisHistory stores conversation messages in a Redis list.
type RedisHistory struct {
    client    *redis.Client
    keyPrefix string
    ttl       time.Duration
}

func NewRedisHistory(client *redis.Client, sessionID string, ttl time.Duration) *RedisHistory {
    return &RedisHistory{
        client:    client,
        keyPrefix: "chat:" + sessionID,
        ttl:       ttl,
    }
}
```

## Step 2: Serialize Messages

Preserve the role and content of each message for accurate reconstruction:

```go
// storedMessage is the JSON-serializable representation of a message.
type storedMessage struct {
    Role       string `json:"role"`
    Content    string `json:"content"`
    ToolCallID string `json:"tool_call_id,omitempty"`
    ModelID    string `json:"model_id,omitempty"`
}

func (h *RedisHistory) AddMessage(ctx context.Context, msg schema.Message) error {
    stored := storedMessage{
        Role: string(msg.GetRole()),
    }

    // Extract text content
    for _, part := range msg.GetContent() {
        if tp, ok := part.(schema.TextPart); ok {
            stored.Content = tp.Text
            break
        }
    }

    // Preserve tool-specific fields
    if tm, ok := msg.(*schema.ToolMessage); ok {
        stored.ToolCallID = tm.ToolCallID
    }
    if am, ok := msg.(*schema.AIMessage); ok {
        stored.ModelID = am.ModelID
    }

    data, err := json.Marshal(stored)
    if err != nil {
        return fmt.Errorf("marshal message: %w", err)
    }

    // Append to Redis list
    if err := h.client.RPush(ctx, h.keyPrefix, data).Err(); err != nil {
        return fmt.Errorf("rpush: %w", err)
    }

    // Refresh TTL on each new message
    if h.ttl > 0 {
        h.client.Expire(ctx, h.keyPrefix, h.ttl)
    }

    return nil
}
```

## Step 3: Deserialize Messages

Reconstruct typed messages from the stored JSON:

```go
func (h *RedisHistory) GetMessages(ctx context.Context) ([]schema.Message, error) {
    raws, err := h.client.LRange(ctx, h.keyPrefix, 0, -1).Result()
    if err != nil {
        return nil, fmt.Errorf("lrange: %w", err)
    }

    messages := make([]schema.Message, 0, len(raws))
    for _, raw := range raws {
        var stored storedMessage
        if err := json.Unmarshal([]byte(raw), &stored); err != nil {
            return nil, fmt.Errorf("unmarshal: %w", err)
        }

        var msg schema.Message
        switch schema.Role(stored.Role) {
        case schema.RoleSystem:
            msg = schema.NewSystemMessage(stored.Content)
        case schema.RoleHuman:
            msg = schema.NewHumanMessage(stored.Content)
        case schema.RoleAI:
            msg = schema.NewAIMessage(stored.Content)
        case schema.RoleTool:
            msg = schema.NewToolMessage(stored.ToolCallID, stored.Content)
        default:
            msg = schema.NewHumanMessage(stored.Content)
        }
        messages = append(messages, msg)
    }

    return messages, nil
}
```

## Step 4: Session Management

Clear history and manage session lifecycle:

```go
func (h *RedisHistory) Clear(ctx context.Context) error {
    return h.client.Del(ctx, h.keyPrefix).Err()
}

func (h *RedisHistory) Length(ctx context.Context) (int64, error) {
    return h.client.LLen(ctx, h.keyPrefix).Result()
}
```

## Step 5: Integrate with an Agent

```go
func main() {
    ctx := context.Background()

    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    defer client.Close()

    sessionID := "user-123-session"
    history := NewRedisHistory(client, sessionID, 24*time.Hour)

    // Load existing conversation
    messages, err := history.GetMessages(ctx)
    if err != nil {
        fmt.Printf("Load error: %v\n", err)
        return
    }

    // Add system prompt if new session
    if len(messages) == 0 {
        sysMsg := schema.NewSystemMessage("You are a helpful assistant.")
        if err := history.AddMessage(ctx, sysMsg); err != nil {
            fmt.Printf("Error: %v\n", err)
            return
        }
        messages = append(messages, sysMsg)
    }

    // Add new user message
    userMsg := schema.NewHumanMessage("My name is Alice. What can you help me with?")
    if err := history.AddMessage(ctx, userMsg); err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    messages = append(messages, userMsg)

    // Generate response
    resp, err := model.Generate(ctx, messages)
    if err != nil {
        fmt.Printf("Generate error: %v\n", err)
        return
    }

    // Store AI response
    if err := history.AddMessage(ctx, resp); err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    fmt.Println(resp.Text())
}
```

## Verification

1. Start the agent and chat: "My name is Alice."
2. Restart the application process.
3. Chat: "What is my name?"
4. The agent should answer "Alice" using the persisted history.

## Next Steps

- [Summary and Window Patterns](/tutorials/memory/summary-window) — Optimize memory for long conversations
- [Multi-turn Conversations](/tutorials/foundation/multiturn-conversations) — Message management fundamentals
