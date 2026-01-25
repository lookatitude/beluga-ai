---
title: "Window-based Context Recovery"
package: "memory"
category: "memory"
complexity: "intermediate"
---

# Window-based Context Recovery

## Problem

You need to recover conversation context from a sliding window of recent messages when memory systems fail or when resuming a conversation, ensuring continuity without losing important recent context.

## Solution

Implement a window-based context recovery system that maintains a sliding window of recent messages, tracks context snapshots, and can reconstruct conversation state from these windows. This works because recent messages contain the most relevant context, and maintaining a window provides resilience against memory failures.

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
    
    "github.com/lookatitude/beluga-ai/pkg/memory/iface"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

var tracer = otel.Tracer("beluga.memory.window_recovery")

// WindowContextRecovery provides context recovery from sliding windows
type WindowContextRecovery struct {
    windows      []*ContextWindow
    windowSize   int
    mu           sync.RWMutex
    memory       iface.Memory
}

// ContextWindow represents a window of messages
type ContextWindow struct {
    Messages  []schema.Message
    Timestamp time.Time
    Summary   string
}

// NewWindowContextRecovery creates a new recovery system
func NewWindowContextRecovery(windowSize int, memory iface.Memory) *WindowContextRecovery {
    return &WindowContextRecovery{
        windows:    []*ContextWindow{},
        windowSize: windowSize,
        memory:     memory,
    }
}

// AddMessage adds a message and updates windows
func (wcr *WindowContextRecovery) AddMessage(ctx context.Context, msg schema.Message) error {
    ctx, span := tracer.Start(ctx, "window_recovery.add_message")
    defer span.End()
    
    wcr.mu.Lock()
    defer wcr.mu.Unlock()
    
    // Get or create current window
    currentWindow := wcr.getOrCreateCurrentWindow()
    
    // Add message to current window
    currentWindow.Messages = append(currentWindow.Messages, msg)
    
    // If window is full, finalize it and create new one
    if len(currentWindow.Messages) >= wcr.windowSize {
        wcr.finalizeWindow(ctx, currentWindow)
        wcr.windows = append(wcr.windows, currentWindow)
        
        // Create new window
        wcr.windows = append(wcr.windows, &ContextWindow{
            Messages:  []schema.Message{},
            Timestamp: time.Now(),
        })
    }
    
    span.SetAttributes(
        attribute.Int("window_count", len(wcr.windows)),
        attribute.Int("current_window_size", len(currentWindow.Messages)),
    )
    span.SetStatus(trace.StatusOK, "message added")
    
    return nil
}

// RecoverContext recovers context from windows
func (wcr *WindowContextRecovery) RecoverContext(ctx context.Context, sessionID string) ([]schema.Message, error) {
    ctx, span := tracer.Start(ctx, "window_recovery.recover")
    defer span.End()
    
    span.SetAttributes(attribute.String("session_id", sessionID))
    
    wcr.mu.RLock()
    defer wcr.mu.RUnlock()
    
    // Try to recover from memory first
    if wcr.memory != nil {
        vars, err := wcr.memory.LoadMemoryVariables(ctx, map[string]any{"session_id": sessionID})
        if err == nil && vars != nil {
            if messages, ok := vars["messages"].([]schema.Message); ok && len(messages) > 0 {
                span.SetAttributes(attribute.Bool("recovered_from_memory", true))
                span.SetStatus(trace.StatusOK, "recovered from memory")
                return messages, nil
            }
        }
    }
    
    // Recover from windows
    recovered := []schema.Message{}
    
    // Recover from all windows
    for _, window := range wcr.windows {
        if len(window.Summary) > 0 {
            // Use summary if available
            recovered = append(recovered, schema.NewSystemMessage("Context: "+window.Summary))
        } else {
            // Use actual messages
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
    span.SetStatus(trace.StatusOK, "recovered from windows")
    
    return recovered, nil
}

// getOrCreateCurrentWindow gets or creates the current window
func (wcr *WindowContextRecovery) getOrCreateCurrentWindow() *ContextWindow {
    if len(wcr.windows) == 0 {
        wcr.windows = append(wcr.windows, &ContextWindow{
            Messages:  []schema.Message{},
            Timestamp: time.Now(),
        })
    }
    
    return wcr.windows[len(wcr.windows)-1]
}

// finalizeWindow finalizes a window and creates summary
func (wcr *WindowContextRecovery) finalizeWindow(ctx context.Context, window *ContextWindow) {
    // Create summary of window messages
    summary := wcr.createSummary(window.Messages)
    window.Summary = summary
}

// createSummary creates a summary of messages
func (wcr *WindowContextRecovery) createSummary(messages []schema.Message) string {
    if len(messages) == 0 {
        return ""
    }
    
    // Simple summary: concatenate message types and first 50 chars
    summary := fmt.Sprintf("Window with %d messages", len(messages))
    
    // Extract key information from messages
    for _, msg := range messages {
        content := msg.GetContent()
        if len(content) > 50 {
            content = content[:50] + "..."
        }
        summary += fmt.Sprintf("; %s: %s", msg.GetType(), content)
    }
    
    return summary
}

// GetRecentWindow gets the most recent window
func (wcr *WindowContextRecovery) GetRecentWindow(ctx context.Context, n int) []schema.Message {
    ctx, span := tracer.Start(ctx, "window_recovery.get_recent")
    defer span.End()
    
    wcr.mu.RLock()
    defer wcr.mu.RUnlock()
    
    messages := []schema.Message{}
    
    // Get messages from most recent windows
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

    // Create recovery system
    // memory := yourMemory
    recovery := NewWindowContextRecovery(10, memory)
    
    // Add messages
    recovery.AddMessage(ctx, schema.NewHumanMessage("Hello"))
    recovery.AddMessage(ctx, schema.NewAIMessage("Hi there!"))
    
    // Recover context
    messages, err := recovery.RecoverContext(ctx, "session-123")
    if err != nil {
        log.Fatalf("Failed to recover: %v", err)
    }
    fmt.Printf("Recovered %d messages\n", len(messages))
}
```

## Explanation

Let's break down what's happening:

1. **Sliding windows** - Notice how we maintain multiple windows of messages. When a window fills up, we finalize it and start a new one. This creates a structured history that can be recovered.

2. **Summary creation** - When windows are finalized, we create summaries. This allows us to preserve context without storing every single message, making recovery more efficient.

3. **Layered recovery** - We first try to recover from persistent memory, then fall back to windows. This provides multiple levels of resilience.

```go
**Key insight:** Maintain sliding windows of recent messages with summaries. This provides both detail (recent messages) and efficiency (summarized older messages) for context recovery.

## Testing

```
Here's how to test this solution:
```go
func TestWindowContextRecovery_RecoversContext(t *testing.T) {
    recovery := NewWindowContextRecovery(5, nil)
    
    // Add messages
    for i := 0; i < 12; i++ {
        recovery.AddMessage(context.Background(), schema.NewHumanMessage(fmt.Sprintf("Message %d", i)))
    }
    
    // Recover
    messages, err := recovery.RecoverContext(context.Background(), "test")
    require.NoError(t, err)
    require.Greater(t, len(messages), 0)
}

## Variations

### Time-based Windows

Create windows based on time instead of message count:
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

- **[Memory TTL & Cleanup Strategies](./memory-ttl-cleanup-strategies.md)** - Implement memory expiration
- **[Chatmodels Multi-step History Trimming](./chatmodels-multi-step-history-trimming.md)** - Trim history intelligently
- **[Memory Package Guide](../package_design_patterns.md)** - For a deeper understanding of memory
