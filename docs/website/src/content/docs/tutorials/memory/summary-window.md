---
title: Summary and Window Memory Patterns
description: Manage conversation context windows with sliding window, summarization, and hybrid memory strategies.
---

LLMs have limited context windows. Sending the entire history of a year-long conversation will exceed token limits, increase cost, and dilute the model's attention. This tutorial covers three strategies for keeping memory lean and relevant: sliding window, summarization, and hybrid approaches.

## What You Will Build

Three memory management strategies — sliding window (last K messages), summarization (running summary), and a hybrid (summary + recent buffer) — suitable for different use cases.

## Prerequisites

- Understanding of [Multi-turn Conversations](/tutorials/foundation/multiturn-conversations)
- A configured LLM (for summarization)

## Pattern 1: Sliding Window

Keep only the last N messages. Predictable token usage and straightforward implementation:

```go
package main

import (
    "github.com/lookatitude/beluga-ai/schema"
)

// WindowMemory retains only the last windowSize messages.
type WindowMemory struct {
    messages   []schema.Message
    system     *schema.SystemMessage  // always retained
    windowSize int
}

func NewWindowMemory(systemPrompt string, windowSize int) *WindowMemory {
    return &WindowMemory{
        system:     schema.NewSystemMessage(systemPrompt),
        windowSize: windowSize,
    }
}

func (m *WindowMemory) AddMessage(msg schema.Message) {
    m.messages = append(m.messages, msg)

    // Trim to window size
    if len(m.messages) > m.windowSize {
        m.messages = m.messages[len(m.messages)-m.windowSize:]
    }
}

func (m *WindowMemory) GetMessages() []schema.Message {
    // System message is always first
    result := make([]schema.Message, 0, 1+len(m.messages))
    result = append(result, m.system)
    result = append(result, m.messages...)
    return result
}
```

**Trade-offs**:
- Predictable token usage
- Loses earlier context ("My name is Alice" from 20 messages ago)
- Simple to implement

## Pattern 2: Running Summary

Maintain a running summary of the entire conversation using an LLM:

```go
import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/llm"
)

// SummaryMemory maintains a running summary of the conversation.
type SummaryMemory struct {
    summary    string
    summarizer llm.ChatModel
    system     *schema.SystemMessage
    buffer     []schema.Message // Recent unsummarized messages
    maxBuffer  int
}

func NewSummaryMemory(systemPrompt string, summarizer llm.ChatModel, maxBuffer int) *SummaryMemory {
    return &SummaryMemory{
        system:     schema.NewSystemMessage(systemPrompt),
        summarizer: summarizer,
        maxBuffer:  maxBuffer,
    }
}

func (m *SummaryMemory) AddMessage(ctx context.Context, msg schema.Message) error {
    m.buffer = append(m.buffer, msg)

    // When buffer exceeds max, summarize and flush
    if len(m.buffer) >= m.maxBuffer {
        if err := m.summarize(ctx); err != nil {
            return err
        }
    }

    return nil
}

func (m *SummaryMemory) summarize(ctx context.Context) error {
    // Build the text to summarize
    var conversation string
    for _, msg := range m.buffer {
        role := msg.GetRole()
        text := ""
        for _, part := range msg.GetContent() {
            if tp, ok := part.(schema.TextPart); ok {
                text = tp.Text
                break
            }
        }
        conversation += fmt.Sprintf("[%s]: %s\n", role, text)
    }

    prompt := fmt.Sprintf(`Current summary:
%s

New conversation:
%s

Write an updated summary that captures all important information, including names, preferences, and key decisions.`, m.summary, conversation)

    msgs := []schema.Message{
        schema.NewSystemMessage("You are a conversation summarizer. Produce a concise summary preserving key facts."),
        schema.NewHumanMessage(prompt),
    }

    resp, err := m.summarizer.Generate(ctx, msgs, llm.WithMaxTokens(500))
    if err != nil {
        return fmt.Errorf("summarize: %w", err)
    }

    m.summary = resp.Text()
    m.buffer = nil // Clear the buffer

    return nil
}

func (m *SummaryMemory) GetMessages() []schema.Message {
    result := []schema.Message{m.system}

    if m.summary != "" {
        result = append(result,
            schema.NewSystemMessage(fmt.Sprintf("Summary of previous conversation:\n%s", m.summary)),
        )
    }

    result = append(result, m.buffer...)
    return result
}
```

**Trade-offs**:
- Infinite conversation length
- Preserves key facts across the entire history
- Loses detail and nuance (summarization is lossy)
- Adds latency (requires LLM call for each summarization)

## Pattern 3: Hybrid (Summary + Buffer)

The production-recommended approach. Keep the last K messages verbatim for immediate context, and a summary of everything before that:

```go
// HybridMemory combines a running summary with a recent message buffer.
type HybridMemory struct {
    summary    string
    summarizer llm.ChatModel
    system     *schema.SystemMessage
    recent     []schema.Message  // Recent messages (verbatim)
    maxRecent  int               // Max recent messages before summarizing
}

func NewHybridMemory(systemPrompt string, summarizer llm.ChatModel, maxRecent int) *HybridMemory {
    return &HybridMemory{
        system:     schema.NewSystemMessage(systemPrompt),
        summarizer: summarizer,
        maxRecent:  maxRecent,
    }
}

func (m *HybridMemory) AddMessage(ctx context.Context, msg schema.Message) error {
    m.recent = append(m.recent, msg)

    // When recent buffer is full, summarize the older half
    if len(m.recent) > m.maxRecent {
        half := len(m.recent) / 2
        toSummarize := m.recent[:half]
        m.recent = m.recent[half:]

        if err := m.summarizeMessages(ctx, toSummarize); err != nil {
            return err
        }
    }

    return nil
}

func (m *HybridMemory) GetMessages() []schema.Message {
    result := []schema.Message{m.system}

    if m.summary != "" {
        result = append(result,
            schema.NewSystemMessage(fmt.Sprintf("Previous conversation summary:\n%s", m.summary)),
        )
    }

    result = append(result, m.recent...)
    return result
}
```

This approach provides:
- Full fidelity for recent context (the current topic)
- Long-term memory via summarization (names, preferences, decisions)
- Bounded token usage

## Choosing the Right Strategy

| Strategy | Context Length | Detail | Latency | Use Case |
|:---|:---|:---|:---|:---|
| Window | Fixed | High (recent) | None | Short tasks, Q&A |
| Summary | Unlimited | Medium | +LLM call | Long sessions, support |
| Hybrid | Bounded | High + Medium | +LLM call | Production agents |

## Verification

1. Use the Hybrid memory with a max buffer of 10 messages.
2. Send 50 messages to the agent, including "My name is Alice" early in the conversation.
3. After summarization occurs, ask "What is my name?"
4. Verify the agent answers correctly from the summary.

## Next Steps

- [Redis Persistence](/tutorials/memory/redis-persistence) — Persist memory across restarts
- [Research Agent](/tutorials/agents/research-agent) — Use memory in autonomous agents
