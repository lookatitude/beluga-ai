---
title: "Memory TTL and Cleanup"
description: "Recipe for expiring old conversation memory in Go using TTL policies — bound storage growth, ensure GDPR compliance, and optimize search performance."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, memory TTL, Go memory cleanup, data retention, GDPR compliance, storage optimization, TTL policy recipe"
---

## Problem

You need to automatically expire and clean up old conversation memory based on time-to-live (TTL) policies, preventing memory from growing unbounded and ensuring privacy compliance by removing stale data.

Memory systems without TTL enforcement face three problems. First, storage costs grow linearly with time as entries accumulate. Second, search performance degrades as the number of entries increases, because more candidates must be evaluated. Third, privacy regulations (GDPR, CCPA) may require automatic data deletion after a retention period. TTL-based cleanup addresses all three concerns: it bounds storage growth, keeps search sets manageable, and provides automatic compliance with data retention policies.

## Solution

Implement a TTL-based cleanup system that tracks memory creation times, periodically checks for expired entries, and removes them while preserving recent and important context. Each memory entry is tagged with a creation timestamp and an expiration time. A background goroutine periodically scans entries and removes expired ones. Priority-based TTL extension allows important sessions (e.g., active support conversations) to live longer than default sessions.

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

    "github.com/lookatitude/beluga-ai/memory"
)

var tracer = otel.Tracer("beluga.memory.ttl_cleanup")

// TTLMemoryWrapper wraps memory with TTL support.
type TTLMemoryWrapper struct {
    memory        memory.Memory
    ttl           time.Duration
    entries       map[string]*MemoryEntry
    mu            sync.RWMutex
    cleanupTicker *time.Ticker
    stopCh        chan struct{}
}

// MemoryEntry represents a memory entry with TTL metadata.
type MemoryEntry struct {
    Key        string
    Value      interface{}
    CreatedAt  time.Time
    AccessedAt time.Time
    ExpiresAt  time.Time
    Priority   int // Higher priority entries expire later
}

// NewTTLMemoryWrapper creates a new TTL wrapper.
func NewTTLMemoryWrapper(mem memory.Memory, ttl time.Duration, cleanupInterval time.Duration) *TTLMemoryWrapper {
    wrapper := &TTLMemoryWrapper{
        memory:  mem,
        ttl:     ttl,
        entries: make(map[string]*MemoryEntry),
        stopCh:  make(chan struct{}),
    }

    wrapper.cleanupTicker = time.NewTicker(cleanupInterval)
    go wrapper.cleanupLoop(context.Background())

    return wrapper
}

// SaveContext saves context with TTL tracking.
func (tmw *TTLMemoryWrapper) SaveContext(ctx context.Context, inputs, outputs map[string]interface{}) error {
    ctx, span := tracer.Start(ctx, "ttl_memory.save_context")
    defer span.End()

    tmw.mu.Lock()
    defer tmw.mu.Unlock()

    if err := tmw.memory.SaveContext(ctx, inputs, outputs); err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return err
    }

    sessionID := fmt.Sprintf("%v", inputs["session_id"])
    entry := &MemoryEntry{
        Key:        sessionID,
        Value:      outputs,
        CreatedAt:  time.Now(),
        AccessedAt: time.Now(),
        ExpiresAt:  time.Now().Add(tmw.ttl),
        Priority:   0,
    }

    tmw.entries[sessionID] = entry

    span.SetAttributes(
        attribute.String("session_id", sessionID),
        attribute.String("expires_at", entry.ExpiresAt.Format(time.RFC3339)),
    )
    span.SetStatus(trace.StatusOK, "context saved with TTL")

    return nil
}

// LoadMemoryVariables loads variables with TTL check.
func (tmw *TTLMemoryWrapper) LoadMemoryVariables(ctx context.Context, inputs map[string]interface{}) (map[string]interface{}, error) {
    ctx, span := tracer.Start(ctx, "ttl_memory.load")
    defer span.End()

    sessionID := fmt.Sprintf("%v", inputs["session_id"])

    tmw.mu.Lock()
    entry, exists := tmw.entries[sessionID]
    if exists {
        if time.Now().After(entry.ExpiresAt) {
            delete(tmw.entries, sessionID)
            tmw.mu.Unlock()

            span.SetAttributes(attribute.Bool("expired", true))
            span.SetStatus(trace.StatusOK, "entry expired")

            return tmw.memory.LoadMemoryVariables(ctx, inputs)
        }

        entry.AccessedAt = time.Now()
    }
    tmw.mu.Unlock()

    vars, err := tmw.memory.LoadMemoryVariables(ctx, inputs)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return nil, err
    }

    span.SetAttributes(attribute.Bool("expired", false))
    span.SetStatus(trace.StatusOK, "variables loaded")

    return vars, nil
}

// cleanupLoop periodically cleans up expired entries.
func (tmw *TTLMemoryWrapper) cleanupLoop(ctx context.Context) {
    for {
        select {
        case <-tmw.cleanupTicker.C:
            tmw.cleanupExpired(ctx)
        case <-tmw.stopCh:
            return
        }
    }
}

// cleanupExpired removes expired entries.
func (tmw *TTLMemoryWrapper) cleanupExpired(ctx context.Context) {
    ctx, span := tracer.Start(ctx, "ttl_memory.cleanup")
    defer span.End()

    tmw.mu.Lock()
    defer tmw.mu.Unlock()

    now := time.Now()
    expired := []string{}

    for key, entry := range tmw.entries {
        if now.After(entry.ExpiresAt) {
            expired = append(expired, key)
        }
    }

    for _, key := range expired {
        delete(tmw.entries, key)
    }

    span.SetAttributes(
        attribute.Int("expired_count", len(expired)),
        attribute.Int("remaining_count", len(tmw.entries)),
    )
}

// ExtendTTL extends the TTL for a session.
func (tmw *TTLMemoryWrapper) ExtendTTL(ctx context.Context, sessionID string, additionalTTL time.Duration) error {
    ctx, span := tracer.Start(ctx, "ttl_memory.extend")
    defer span.End()

    tmw.mu.Lock()
    defer tmw.mu.Unlock()

    entry, exists := tmw.entries[sessionID]
    if !exists {
        return fmt.Errorf("session %s not found", sessionID)
    }

    entry.ExpiresAt = entry.ExpiresAt.Add(additionalTTL)

    span.SetAttributes(
        attribute.String("session_id", sessionID),
        attribute.String("new_expires_at", entry.ExpiresAt.Format(time.RFC3339)),
    )
    span.SetStatus(trace.StatusOK, "TTL extended")

    return nil
}

// SetPriority sets priority for an entry (higher priority = longer TTL).
func (tmw *TTLMemoryWrapper) SetPriority(ctx context.Context, sessionID string, priority int) {
    tmw.mu.Lock()
    defer tmw.mu.Unlock()

    entry, exists := tmw.entries[sessionID]
    if exists {
        entry.Priority = priority
        extension := time.Duration(priority) * time.Hour
        entry.ExpiresAt = entry.ExpiresAt.Add(extension)
    }
}

// Stop stops the cleanup goroutine.
func (tmw *TTLMemoryWrapper) Stop() {
    close(tmw.stopCh)
    if tmw.cleanupTicker != nil {
        tmw.cleanupTicker.Stop()
    }
}

func main() {
    ctx := context.Background()

    // mem := your memory.Memory instance
    ttlMemory := NewTTLMemoryWrapper(mem, 24*time.Hour, 1*time.Hour)
    defer ttlMemory.Stop()

    inputs := map[string]interface{}{"session_id": "session-123", "message": "Hello"}
    outputs := map[string]interface{}{"response": "Hi!"}

    if err := ttlMemory.SaveContext(ctx, inputs, outputs); err != nil {
        log.Fatalf("Failed to save: %v", err)
    }

    vars, err := ttlMemory.LoadMemoryVariables(ctx, map[string]interface{}{"session_id": "session-123"})
    if err != nil {
        log.Fatalf("Failed to load: %v", err)
    }
    fmt.Printf("Loaded: %v\n", vars)
}
```

## Explanation

1. **TTL tracking** -- Each memory entry is associated with a creation time and an expiration time. When entries are loaded, the wrapper checks the TTL before returning data. Expired entries are treated as non-existent, ensuring stale data is never served. This provides automatic data lifecycle management without requiring callers to manage expiration themselves.

2. **Periodic cleanup** -- A background goroutine periodically scans all entries and removes expired ones. This prevents the entry map from growing unbounded even if entries are never accessed (and thus never checked lazily). The cleanup interval should balance between responsiveness (removing expired entries promptly) and overhead (lock contention from frequent scans).

3. **Access-based timestamps** -- When entries are accessed, their `AccessedAt` timestamp is updated. This information can be used for more sophisticated TTL policies, such as extending the TTL for frequently-accessed sessions or implementing LRU-style eviction. The access tracking adds minimal overhead since it piggybacks on the existing lock acquisition.

4. **Priority-based TTL** -- High-priority sessions (e.g., active support conversations, VIP users) can have their TTL extended automatically via `SetPriority`. Each priority level adds one hour to the expiration time. This balances memory efficiency with preserving valuable context for important interactions.

## Variations

### Lazy Expiration

Check expiration only on access instead of running a background cleanup, reducing overhead for small deployments:

```go
func (tmw *TTLMemoryWrapper) LoadWithLazyExpiration(ctx context.Context, inputs map[string]interface{}) (map[string]interface{}, error) {
    // Check expiration on load only
}
```

### Gradual Expiration

Gradually reduce priority instead of hard expiration, preserving summaries of expired conversations:

```go
func (tmw *TTLMemoryWrapper) GradualExpiration(ctx context.Context) {
    // Reduce priority over time
}
```

## Related Recipes

- [Window-Based Context Recovery](/docs/cookbook/memory-context-recovery) -- Recover context from sliding windows
- [Conversation Expiry Logic](/docs/cookbook/conversation-expiry) -- Expire inactive conversations
