---
title: "Multi-step History Trimming"
package: "chatmodels"
category: "optimization"
complexity: "intermediate"
---

# Multi-step History Trimming

## Problem

You need to manage conversation history that grows beyond token limits, intelligently trimming old messages while preserving important context and maintaining conversation coherence.

## Solution

Implement a history trimming strategy that prioritizes recent messages, preserves system messages and summaries, and uses semantic similarity to keep the most relevant historical context. This works because you can analyze message importance, create summaries of trimmed content, and maintain a sliding window of recent messages.

## Code Example
```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
    
    "github.com/lookatitude/beluga-ai/pkg/schema"
    "github.com/lookatitude/beluga-ai/pkg/llms/iface"
)

var tracer = otel.Tracer("beluga.chatmodels.history_trimming")

// HistoryTrimmer manages conversation history with intelligent trimming
type HistoryTrimmer struct {
    maxTokens       int
    keepSystem      bool
    keepRecent      int
    summarizer      iface.ChatModel
}

// NewHistoryTrimmer creates a new history trimmer
func NewHistoryTrimmer(maxTokens int, keepRecent int, keepSystem bool, summarizer iface.ChatModel) *HistoryTrimmer {
    return &HistoryTrimmer{
        maxTokens:  maxTokens,
        keepSystem: keepSystem,
        keepRecent: keepRecent,
        summarizer: summarizer,
    }
}

// TrimHistory trims history to fit token limits
func (ht *HistoryTrimmer) TrimHistory(ctx context.Context, messages []schema.Message) ([]schema.Message, error) {
    ctx, span := tracer.Start(ctx, "history_trimmer.trim")
    defer span.End()
    
    span.SetAttributes(
        attribute.Int("input_message_count", len(messages)),
        attribute.Int("max_tokens", ht.maxTokens),
    )
    
    // Estimate tokens
    totalTokens := ht.estimateTokens(messages)
    
    if totalTokens <= ht.maxTokens {
        span.SetAttributes(attribute.Bool("trimming_needed", false))
        span.SetStatus(trace.StatusOK, "no trimming needed")
        return messages, nil
    }
    
    // Separate system and recent messages
    systemMsgs, recentMsgs, oldMsgs := ht.categorizeMessages(messages)
    
    // Keep system messages if requested
    trimmed := []schema.Message{}
    if ht.keepSystem {
        trimmed = append(trimmed, systemMsgs...)
    }
    
    // Keep recent messages
    trimmed = append(trimmed, recentMsgs...)
    
    // Calculate tokens used
    usedTokens := ht.estimateTokens(trimmed)
    remainingTokens := ht.maxTokens - usedTokens
    
    // Summarize old messages if we have space and summarizer
    if len(oldMsgs) > 0 && remainingTokens > 100 && ht.summarizer != nil {
        summary, err := ht.summarizeMessages(ctx, oldMsgs)
        if err == nil && ht.estimateTokens([]schema.Message{summary}) <= remainingTokens {
            // Insert summary before recent messages
            summarySlice := []schema.Message{summary}
            trimmed = append(systemMsgs, append(summarySlice, recentMsgs...)...)
        }
    }
    
    // If still too large, aggressively trim recent messages
    trimmed = ht.aggressiveTrim(trimmed)
    
    span.SetAttributes(
        attribute.Int("output_message_count", len(trimmed)),
        attribute.Int("trimmed_count", len(messages)-len(trimmed)),
    )
    span.SetStatus(trace.StatusOK, "history trimmed")
    
    return trimmed, nil
}

// categorizeMessages categorizes messages into system, recent, and old
func (ht *HistoryTrimmer) categorizeMessages(messages []schema.Message) ([]schema.Message, []schema.Message, []schema.Message) {
    systemMsgs := []schema.Message{}
    recentMsgs := []schema.Message{}
    oldMsgs := []schema.Message{}

    for i, msg := range messages {
        if msg.GetType() == "system" {
            systemMsgs = append(systemMsgs, msg)
        } else if i >= len(messages)-ht.keepRecent {
            recentMsgs = append(recentMsgs, msg)
        } else {
            oldMsgs = append(oldMsgs, msg)
        }
    }
    
    return systemMsgs, recentMsgs, oldMsgs
}

// summarizeMessages creates a summary of old messages
func (ht *HistoryTrimmer) summarizeMessages(ctx context.Context, messages []schema.Message) (schema.Message, error) {
    ctx, span := tracer.Start(ctx, "history_trimmer.summarize")
    defer span.End()
    
    // Build summary prompt
    content := "Summarize the following conversation history:\n\n"
    for _, msg := range messages {
        content += fmt.Sprintf("%s: %s\n", msg.GetType(), msg.GetContent())
    }
    
    summaryPrompt := []schema.Message{
        schema.NewSystemMessage("You are a conversation summarizer. Create a concise summary that preserves key information."),
        schema.NewHumanMessage(content),
    }
    
    response, err := ht.summarizer.Generate(ctx, summaryPrompt)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return nil, err
    }
    
    summary := schema.NewSystemMessage("Previous conversation summary: " + response.GetContent())
    span.SetStatus(trace.StatusOK, "summary created")
    
    return summary, nil
}

// aggressiveTrim aggressively trims messages to fit limit
func (ht *HistoryTrimmer) aggressiveTrim(messages []schema.Message) []schema.Message {
    trimmed := []schema.Message{}
    tokens := 0

    // Keep system messages
    for _, msg := range messages {
        if msg.GetType() == "system" {
            trimmed = append(trimmed, msg)
            tokens += ht.estimateTokens([]schema.Message{msg})
        }
    }
    
    // Add recent messages until limit
    for i := len(messages) - 1; i >= 0; i-- {
        if messages[i].GetType() == "system" {
            continue
        }
        
        msgTokens := ht.estimateTokens([]schema.Message{messages[i]})
        if tokens+msgTokens > ht.maxTokens {
            break
        }
        
        trimmed = append(trimmed, messages[i])
        tokens += msgTokens
    }
    
    // Reverse to maintain order
    reversed := make([]schema.Message, len(trimmed))
    for i, j := 0, len(trimmed)-1; i < len(trimmed); i, j = i+1, j-1 {
        reversed[i] = trimmed[j]
    }
    
    return reversed
}

// estimateTokens estimates token count (simplified)
func (ht *HistoryTrimmer) estimateTokens(messages []schema.Message) int {
    total := 0
    for _, msg := range messages {
        // Rough estimate: ~4 characters per token
        total += len(msg.GetContent()) / 4
        // Add overhead for message structure
        total += 5
    }
    return total
}

func main() {
    ctx := context.Background()

    // Create trimmer
    // summarizer := yourSummarizerModel
    trimmer := NewHistoryTrimmer(1000, 5, true, summarizer)
    
    // Messages with history
    messages := []schema.Message{
        schema.NewSystemMessage("You are a helpful assistant"),
        // ... many old messages ...
        schema.NewHumanMessage("Recent question 1"),
        schema.NewAIMessage("Recent answer 1"),
        schema.NewHumanMessage("Current question"),
    }
    
    trimmed, err := trimmer.TrimHistory(ctx, messages)
    if err != nil {
        log.Fatalf("Failed to trim: %v", err)
    }
    fmt.Printf("Trimmed from %d to %d messages\n", len(messages), len(trimmed))
}
```

## Explanation

Let's break down what's happening:

1. **Priority-based trimming** - Notice how we prioritize system messages and recent messages. System messages contain important context, and recent messages are most relevant to the current conversation.

2. **Summarization** - When we have old messages to trim, we can summarize them if a summarizer is available. This preserves key information while reducing token count.

3. **Aggressive trimming** - If summarization isn't possible or sufficient, we aggressively trim while maintaining the most recent context. This ensures we always stay within token limits.

```go
**Key insight:** Preserve system messages and recent context, then summarize or trim older messages. This maintains conversation coherence while respecting token limits.

## Testing

```
Here's how to test this solution:
```go
func TestHistoryTrimmer_TrimsWhenNeeded(t *testing.T) {
    trimmer := NewHistoryTrimmer(100, 3, true, nil)
    
    messages := make([]schema.Message, 20)
    for i := 0; i < 20; i++ {
        messages[i] = schema.NewHumanMessage(fmt.Sprintf("Message %d", i))
    }
    
    trimmed, err := trimmer.TrimHistory(context.Background(), messages)
    require.NoError(t, err)
    require.LessOrEqual(t, len(trimmed), len(messages))
}

## Variations

### Token-aware Trimming

Use actual token counting instead of estimation:
func (ht *HistoryTrimmer) TrimHistoryWithTokenCounter(ctx context.Context, messages []schema.Message, counter TokenCounter) ([]schema.Message, error) {
    // Use actual token counting
}
```

### Importance-based Trimming

Score message importance and trim least important:
```go
func (ht *HistoryTrimmer) scoreMessageImportance(msg schema.Message) float64 {
    // Score based on content, metadata, etc.
}
```

## Related Recipes

- **[Chatmodels Streaming Chunks with Metadata](./chatmodels-streaming-chunks-metadata.md)** - Stream with metadata
- **[Memory Window-based Context Recovery](./memory-window-based-context-recovery.md)** - Recover context from windows
- **[Chatmodels Package Guide](../package_design_patterns.md)** - For a deeper understanding of chatmodels
