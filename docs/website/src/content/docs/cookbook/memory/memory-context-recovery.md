---
title: "Window-Based Context Recovery"
description: "Recover conversation context from sliding windows of recent messages when memory systems fail or sessions resume after interruption."
---

## Problem

You need to recover conversation context from a sliding window of recent messages when memory systems fail or when resuming a conversation, ensuring continuity without losing important recent context.

Memory systems can fail: Redis goes down, database connections drop, or the process restarts. When the primary memory store is unavailable, the agent loses all conversation context, leading to confused responses that ignore everything discussed previously. Window-based context recovery provides a resilience layer: even if the persistent memory fails, the agent can reconstruct enough context from recent message windows to maintain conversational coherence. Additionally, when resuming long conversations, loading the full history is wasteful -- a windowed summary of older context plus recent messages provides sufficient context at lower cost.

## Solution

Implement a window-based context recovery system that maintains a sliding window of recent messages, tracks context snapshots, and can reconstruct conversation state from these windows. Messages are organized into fixed-size windows. When a window fills, it is summarized and a new window begins. Recovery first attempts the persistent memory store, then falls back to reconstructing context from windows. Recent windows contain full messages for maximum fidelity, while older windows are represented by summaries.

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

    "github.com/lookatitude/beluga-ai/memory"
    "github.com/lookatitude/beluga-ai/schema"
)

var tracer = otel.Tracer("beluga.memory.window_recovery")

// WindowContextRecovery provides context recovery from sliding windows.
type WindowContextRecovery struct {
    windows    []*ContextWindow
    windowSize int
    mu         sync.RWMutex
    memory     memory.Memory
}

// ContextWindow represents a window of messages.
type ContextWindow struct {
    Messages  []schema.Message
    Timestamp time.Time
    Summary   string
}

// NewWindowContextRecovery creates a new recovery system.
func NewWindowContextRecovery(windowSize int, mem memory.Memory) *WindowContextRecovery {
    return &WindowContextRecovery{
        windows:    []*ContextWindow{},
        windowSize: windowSize,
        memory:     mem,
    }
}

// AddMessage adds a message and updates windows.
func (wcr *WindowContextRecovery) AddMessage(ctx context.Context, msg schema.Message) error {
    ctx, span := tracer.Start(ctx, "window_recovery.add_message")
    defer span.End()

    wcr.mu.Lock()
    defer wcr.mu.Unlock()

    currentWindow := wcr.getOrCreateCurrentWindow()

    currentWindow.Messages = append(currentWindow.Messages, msg)

    // If window is full, finalize it and create a new one
    if len(currentWindow.Messages) >= wcr.windowSize {
        wcr.finalizeWindow(ctx, currentWindow)
        wcr.windows = append(wcr.windows, currentWindow)

        wcr.windows = append(wcr.windows, &ContextWindow{
            Messages:  []schema.Message{},
            Timestamp: time.Now(),
        })
    }

    span.SetAttributes(
        attribute.Int("window_count", len(wcr.windows)),
        attribute.Int("current_window_size", len(currentWindow.Messages)),
    )

    return nil
}

// RecoverContext recovers context from windows.
func (wcr *WindowContextRecovery) RecoverContext(ctx context.Context, sessionID string) ([]schema.Message, error) {
    ctx, span := tracer.Start(ctx, "window_recovery.recover")
    defer span.End()

    span.SetAttributes(attribute.String("session_id", sessionID))

    wcr.mu.RLock()
    defer wcr.mu.RUnlock()

    // Try to recover from persistent memory first
    if wcr.memory != nil {
        vars, err := wcr.memory.LoadMemoryVariables(ctx, map[string]any{"session_id": sessionID})
        if err == nil && vars != nil {
            if messages, ok := vars["messages"].([]schema.Message); ok && len(messages) > 0 {
                span.SetAttributes(attribute.Bool("recovered_from_memory", true))
                return messages, nil
            }
        }
    }

    // Fall back to window recovery
    recovered := []schema.Message{}

    for _, window := range wcr.windows {
        if len(window.Summary) > 0 {
            recovered = append(recovered, schema.NewSystemMessage("Context: "+window.Summary))
        } else {
            recovered = append(recovered, window.Messages...)
        }
    }

    // Add current window messages
    if len(wcr.windows) > 0 {
        currentWindow := wcr.windows[len(wcr.windows)-1]
        recovered = append(recovered, currentWindow.Messages...)
    }

    span.SetAttributes(
        attribute.Int("recovered_message_count", len(recovered)),
        attribute.Int("window_count_used", len(wcr.windows)),
    )

    return recovered, nil
}

// getOrCreateCurrentWindow gets or creates the current window.
func (wcr *WindowContextRecovery) getOrCreateCurrentWindow() *ContextWindow {
    if len(wcr.windows) == 0 {
        wcr.windows = append(wcr.windows, &ContextWindow{
            Messages:  []schema.Message{},
            Timestamp: time.Now(),
        })
    }

    return wcr.windows[len(wcr.windows)-1]
}

// finalizeWindow finalizes a window and creates a summary.
func (wcr *WindowContextRecovery) finalizeWindow(ctx context.Context, window *ContextWindow) {
    summary := wcr.createSummary(window.Messages)
    window.Summary = summary
}

// createSummary creates a summary of messages.
func (wcr *WindowContextRecovery) createSummary(messages []schema.Message) string {
    if len(messages) == 0 {
        return ""
    }

    summary := fmt.Sprintf("Window with %d messages", len(messages))

    for _, msg := range messages {
        content := msg.GetContent()
        if len(content) > 50 {
            content = content[:50] + "..."
        }
        summary += fmt.Sprintf("; %s: %s", msg.GetType(), content)
    }

    return summary
}

// GetRecentWindow gets the most recent n windows of messages.
func (wcr *WindowContextRecovery) GetRecentWindow(ctx context.Context, n int) []schema.Message {
    ctx, span := tracer.Start(ctx, "window_recovery.get_recent")
    defer span.End()

    wcr.mu.RLock()
    defer wcr.mu.RUnlock()

    messages := []schema.Message{}

    start := len(wcr.windows) - n
    if start < 0 {
        start = 0
    }

    for i := start; i < len(wcr.windows); i++ {
        window := wcr.windows[i]
        if len(window.Summary) > 0 {
            messages = append(messages, schema.NewSystemMessage(window.Summary))
        } else {
            messages = append(messages, window.Messages...)
        }
    }

    span.SetAttributes(
        attribute.Int("window_count", n),
        attribute.Int("message_count", len(messages)),
    )

    return messages
}

func main() {
    ctx := context.Background()

    // mem := your memory.Memory instance (or nil for window-only recovery)
    recovery := NewWindowContextRecovery(10, mem)

    recovery.AddMessage(ctx, schema.NewHumanMessage("Hello"))
    recovery.AddMessage(ctx, schema.NewAIMessage("Hi there!"))

    messages, err := recovery.RecoverContext(ctx, "session-123")
    if err != nil {
        log.Fatalf("Failed to recover: %v", err)
    }
    fmt.Printf("Recovered %d messages\n", len(messages))
}
```

## Explanation

1. **Sliding windows** -- Messages are organized into fixed-size windows. When a window fills up, it is finalized (summarized) and a new window starts. This creates a structured history where older context is compressed into summaries while recent context retains full message detail. The window size controls the granularity of summarization: smaller windows produce more frequent summaries, larger windows retain more raw messages.

2. **Summary creation** -- When windows are finalized, summaries are generated to compress the full message content into a compact representation. In production, you would use the LLM itself to create these summaries (see the Memory Compression recipe). The summary preserves the essential context without storing every message, making recovery more efficient for older conversation history.

3. **Layered recovery** -- The system uses a fallback strategy: first try the persistent memory store (Redis, PostgreSQL, etc.), then fall back to window-based recovery if persistent memory is unavailable. This provides multiple levels of resilience: the persistent store handles normal operation, and windows handle failure scenarios. The layered approach means a Redis outage degrades gracefully rather than causing total context loss.

4. **Recent message priority** -- The most recent window always contains full messages (not summaries), ensuring that the immediately relevant context is preserved in complete detail. Older windows contribute summaries, which use fewer tokens while preserving the key facts and decisions from earlier in the conversation.

## Variations

### Time-Based Windows

Create windows based on time instead of message count, which is useful for conversations with long idle periods:

```go
type TimeBasedWindow struct {
    Duration time.Duration
}
```

### Compression

Compress old windows to save memory:

```go
func (wcr *WindowContextRecovery) CompressOldWindows(ctx context.Context) error {
    // Compress windows older than threshold
}
```

## Related Recipes

- [Memory TTL and Cleanup](/cookbook/memory-ttl-cleanup) -- Implement memory expiration
- [History Trimming](/cookbook/history-trimming) -- Trim conversation history intelligently
