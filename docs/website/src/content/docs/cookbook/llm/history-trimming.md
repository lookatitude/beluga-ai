---
title: "Multi-step History Trimming"
description: "Manage conversation history that grows beyond token limits with intelligent trimming and summarization."
---

## Problem

You need to manage conversation history that grows beyond token limits, intelligently trimming old messages while preserving important context and maintaining conversation coherence.

## Solution

Implement a history trimming strategy that prioritizes recent messages, preserves system messages and summaries, and uses semantic similarity to keep the most relevant historical context. This works because you can analyze message importance, create summaries of trimmed content, and maintain a sliding window of recent messages.

## Why This Matters

Every LLM has a finite context window, and conversation history grows without bound. Without management, a chatbot that works perfectly for 10 exchanges will fail silently on exchange 50 when the history exceeds the model's token limit. The naive solution -- truncating from the beginning -- loses the system prompt and early context that may be critical to the conversation's purpose.

The three-tier trimming strategy in this recipe addresses this by categorizing messages into system messages (always preserved), recent messages (preserved as the active conversation window), and old messages (candidates for summarization or removal). System messages contain the agent's instructions and personality, so losing them changes behavior. Recent messages contain the immediate conversational context that the user expects the agent to remember. Old messages contain historical context that may or may not be relevant to the current topic.

Summarization is the key differentiator between intelligent trimming and simple truncation. When old messages are summarized rather than discarded, the agent retains awareness of what was discussed earlier without paying the full token cost. The summarizer itself is an LLM call, which adds latency and cost, so it should only be invoked when the history actually exceeds the token budget. The aggressive trim fallback ensures the conversation always stays within limits even when summarization is unavailable or insufficient -- this is the safety net that prevents API errors from oversized requests.

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

    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
)

var tracer = otel.Tracer("beluga.chatmodels.history_trimming")

// HistoryTrimmer manages conversation history with intelligent trimming
type HistoryTrimmer struct {
    maxTokens  int
    keepSystem bool
    keepRecent int
    summarizer llm.ChatModel
}

// NewHistoryTrimmer creates a new history trimmer
func NewHistoryTrimmer(maxTokens int, keepRecent int, keepSystem bool, summarizer llm.ChatModel) *HistoryTrimmer {
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

    for _, msg := range messages {
        if msg.GetType() == "system" {
            trimmed = append(trimmed, msg)
            tokens += ht.estimateTokens([]schema.Message{msg})
        }
    }

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
        total += len(msg.GetContent()) / 4
        total += 5
    }
    return total
}

func main() {
    ctx := context.Background()

    // Create trimmer (pass nil summarizer if not available)
    trimmer := NewHistoryTrimmer(1000, 5, true, nil)

    messages := []schema.Message{
        schema.NewSystemMessage("You are a helpful assistant"),
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

1. **Priority-based categorization** -- Messages are split into three categories: system messages (always preserved because they define agent behavior), recent messages (the last N messages that form the active conversation window), and old messages (everything else, which is a candidate for summarization or removal). This categorization ensures the most important context survives trimming.

2. **Summarization as compression** -- Old messages are summarized into a single system message using an LLM call when a summarizer is available. This compresses potentially thousands of tokens of conversation history into a concise summary that preserves key facts and decisions. The summary is injected between system messages and recent messages, maintaining chronological coherence.

3. **Aggressive trim as safety net** -- If summarization is unavailable (nil summarizer) or the summary itself is too large, aggressive trimming takes over. It works backwards from the most recent messages, adding them until the token budget is exhausted. System messages are always included first, guaranteeing that agent instructions survive even extreme trimming.

4. **OTel observability** -- Spans record the input message count, output message count, and number of trimmed messages. This data helps you tune the `maxTokens` and `keepRecent` parameters based on actual conversation patterns rather than guesswork. A high trim rate might indicate the need for a larger context window or more aggressive summarization.

## Testing

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
```

## Variations

### Token-aware Trimming

Use actual token counting instead of estimation:

```go
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

- **[Streaming Chunks with Metadata](./streaming-metadata)** — Stream with metadata
- **[Memory Context Recovery](./memory-context-recovery)** — Recover context from windows
