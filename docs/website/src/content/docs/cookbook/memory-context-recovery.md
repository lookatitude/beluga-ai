---
title: "Window-Based Context Recovery"
description: "Recover conversation context from sliding windows of recent messages when memory systems fail or sessions resume."
---

## Problem

You need to recover conversation context from a sliding window of recent messages when memory systems fail or when resuming a conversation, ensuring continuity without losing important recent context.

## Solution

Implement a window-based context recovery system that maintains a sliding window of recent messages, tracks context snapshots, and can reconstruct conversation state from these windows. Recent messages contain the most relevant context, and maintaining a window provides resilience against memory failures.

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

1. **Sliding windows** — Messages are organized into windows of a fixed size. When a window fills up, it is finalized and a new window is started. This creates a structured history that can be recovered efficiently.

2. **Summary creation** — When windows are finalized, summaries are created. This preserves context without storing every single message, making recovery more efficient for older conversation history.

3. **Layered recovery** — The system first tries to recover from persistent memory, then falls back to windows. This provides multiple levels of resilience against different failure modes.

4. **Recent message priority** — The most recent window always contains full messages (not summaries), ensuring that the immediately relevant context is preserved in full detail.

## Variations

### Time-Based Windows

Create windows based on time instead of message count:

```go
type TimeBasedWindow struct {
    Duration time.Duration
}
```

### Compression

Compress old windows to save space:

```go
func (wcr *WindowContextRecovery) CompressOldWindows(ctx context.Context) error {
    // Compress windows older than threshold
}
```

## Related Recipes

- [Memory TTL and Cleanup](/cookbook/memory-ttl-cleanup) — Implement memory expiration
- [History Trimming](/cookbook/history-trimming) — Trim conversation history intelligently
