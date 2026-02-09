---
title: "Conversation Expiry Logic"
description: "Automatically expire and clean up inactive conversations to free resources and ensure fresh session starts."
---

## Problem

You need to automatically expire and clean up inactive conversations after a period of inactivity, freeing resources and ensuring users start fresh conversations after long gaps.

## Solution

Implement conversation expiry that tracks last activity time per conversation, periodically checks for expired conversations, and cleans up resources (memory, context, state) for expired conversations. Track activity timestamps and run background cleanup processes.

## Code Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "sync"
    "time"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("beluga.messaging.conversation_expiry")

// ConversationState tracks conversation state.
type ConversationState struct {
    ID           string
    LastActivity time.Time
    CreatedAt    time.Time
    ExpiresAt    time.Time
    Active       bool
}

// ConversationExpiryManager manages conversation expiration.
type ConversationExpiryManager struct {
    conversations   map[string]*ConversationState
    ttl             time.Duration
    cleanupInterval time.Duration
    mu              sync.RWMutex
    stopCh          chan struct{}
}

// NewConversationExpiryManager creates a new manager.
func NewConversationExpiryManager(ttl time.Duration, cleanupInterval time.Duration) *ConversationExpiryManager {
    manager := &ConversationExpiryManager{
        conversations:   make(map[string]*ConversationState),
        ttl:             ttl,
        cleanupInterval: cleanupInterval,
        stopCh:          make(chan struct{}),
    }

    go manager.cleanupLoop(context.Background())

    return manager
}

// RegisterConversation registers a new conversation.
func (cem *ConversationExpiryManager) RegisterConversation(ctx context.Context, conversationID string) error {
    ctx, span := tracer.Start(ctx, "expiry_manager.register")
    defer span.End()

    cem.mu.Lock()
    defer cem.mu.Unlock()

    now := time.Now()
    cem.conversations[conversationID] = &ConversationState{
        ID:           conversationID,
        LastActivity: now,
        CreatedAt:    now,
        ExpiresAt:    now.Add(cem.ttl),
        Active:       true,
    }

    span.SetAttributes(
        attribute.String("conversation_id", conversationID),
        attribute.String("expires_at", cem.conversations[conversationID].ExpiresAt.Format(time.RFC3339)),
    )
    span.SetStatus(trace.StatusOK, "conversation registered")

    return nil
}

// UpdateActivity updates last activity time, resetting the expiry timer.
func (cem *ConversationExpiryManager) UpdateActivity(ctx context.Context, conversationID string) error {
    ctx, span := tracer.Start(ctx, "expiry_manager.update_activity")
    defer span.End()

    cem.mu.Lock()
    defer cem.mu.Unlock()

    state, exists := cem.conversations[conversationID]
    if !exists {
        return fmt.Errorf("conversation %s not found", conversationID)
    }

    now := time.Now()
    state.LastActivity = now
    state.ExpiresAt = now.Add(cem.ttl)
    state.Active = true

    span.SetAttributes(
        attribute.String("conversation_id", conversationID),
        attribute.String("new_expires_at", state.ExpiresAt.Format(time.RFC3339)),
    )
    span.SetStatus(trace.StatusOK, "activity updated")

    return nil
}

// IsExpired checks if a conversation is expired.
func (cem *ConversationExpiryManager) IsExpired(conversationID string) bool {
    cem.mu.RLock()
    defer cem.mu.RUnlock()

    state, exists := cem.conversations[conversationID]
    if !exists {
        return true
    }

    return time.Now().After(state.ExpiresAt)
}

// cleanupLoop periodically cleans up expired conversations.
func (cem *ConversationExpiryManager) cleanupLoop(ctx context.Context) {
    ticker := time.NewTicker(cem.cleanupInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            cem.cleanupExpired(ctx)
        case <-cem.stopCh:
            return
        }
    }
}

// cleanupExpired removes expired conversations and their resources.
func (cem *ConversationExpiryManager) cleanupExpired(ctx context.Context) {
    ctx, span := tracer.Start(ctx, "expiry_manager.cleanup")
    defer span.End()

    cem.mu.Lock()
    defer cem.mu.Unlock()

    now := time.Now()
    expired := []string{}

    for id, state := range cem.conversations {
        if now.After(state.ExpiresAt) {
            expired = append(expired, id)
        }
    }

    for _, id := range expired {
        cem.cleanupConversation(ctx, id)
        delete(cem.conversations, id)
    }

    span.SetAttributes(
        attribute.Int("expired_count", len(expired)),
        attribute.Int("remaining_count", len(cem.conversations)),
    )
}

// cleanupConversation cleans up resources for a conversation.
func (cem *ConversationExpiryManager) cleanupConversation(ctx context.Context, conversationID string) {
    log.Printf("Cleaning up conversation %s", conversationID)
    // Clean up memory, context, state, etc.
}

// Stop stops the cleanup loop.
func (cem *ConversationExpiryManager) Stop() {
    close(cem.stopCh)
}

func main() {
    ctx := context.Background()

    manager := NewConversationExpiryManager(1*time.Hour, 5*time.Minute)
    defer manager.Stop()

    if err := manager.RegisterConversation(ctx, "conv-123"); err != nil {
        log.Fatalf("Failed to register: %v", err)
    }

    if err := manager.UpdateActivity(ctx, "conv-123"); err != nil {
        log.Fatalf("Failed to update: %v", err)
    }

    expired := manager.IsExpired("conv-123")
    fmt.Printf("Expired: %v\n", expired)
}
```

## Explanation

1. **Activity tracking** — Last activity time is tracked and the expiry timer resets on each activity. This keeps active conversations alive while allowing inactive ones to expire naturally.

2. **Periodic cleanup** — A background goroutine periodically checks for expired conversations. This ensures cleanup happens automatically without blocking request handlers.

3. **Resource cleanup** — When conversations expire, associated resources (memory, context, state) are cleaned up. This prevents resource leaks from accumulated dead conversations.

4. **Thread safety** — All operations use `sync.RWMutex` for safe concurrent access. Read operations (like `IsExpired`) use read locks, while mutations use write locks.

## Variations

### Activity-Based Extension

Extend expiry only on meaningful activity:

```go
func (cem *ConversationExpiryManager) UpdateActivityIfMeaningful(ctx context.Context, conversationID string, activityType string) {
    // Only extend for certain activity types (messages, not heartbeats)
}
```

### Gradual Expiry

Expire conversations gradually with warning before removal:

```go
type ConversationState struct {
    WarningSent bool
}
```

## Related Recipes

- [Handling Inbound Media](/cookbook/inbound-media) — Handle media attachments in conversations
- [Memory TTL and Cleanup](/cookbook/memory-ttl-cleanup) — Memory expiration strategies
