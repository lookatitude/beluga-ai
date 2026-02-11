---
title: Summary and Window Memory Patterns
description: Manage conversation context windows with sliding window, summarization, and hybrid memory strategies.
---

LLMs have limited context windows. Sending the entire history of a year-long conversation will exceed token limits, increase cost, and dilute the model's attention on recent context. Effective memory management is about keeping the most relevant information in the context window while discarding or compressing the rest. This tutorial covers three strategies with different trade-offs: sliding window for simplicity, summarization for infinite history, and a hybrid approach that combines the strengths of both. Beluga AI's memory system draws from the **MemGPT 3-tier model** (Core, Recall, Archival), and these patterns map to the Core and Recall tiers.

## What You Will Build

Three memory management strategies -- sliding window (last K messages), summarization (running summary), and a hybrid (summary + recent buffer) -- suitable for different use cases.

## Prerequisites

- Understanding of [Multi-turn Conversations](/tutorials/foundation/multiturn-conversations)
- A configured LLM (for summarization)

## Pattern 1: Sliding Window

The simplest memory strategy: keep only the last N messages and discard everything older. This works because LLMs tend to pay most attention to recent messages, and for short tasks like Q&A, the last few exchanges contain all the context needed. The system message is stored separately and always prepended, ensuring the agent's persona and instructions are never evicted.

The trade-off is clear: predictable token usage and zero latency overhead, but complete loss of older context. If a user says "My name is Alice" in message 5 and your window is 10, the agent will forget the name after 10 more exchanges.

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

A running summary preserves key facts from the entire conversation by periodically using an LLM to compress the buffer into a text summary. When the buffer reaches `maxBuffer` messages, the summarizer generates an updated summary that incorporates both the previous summary and the new messages, then flushes the buffer.

This approach enables infinite conversation length -- no matter how many messages are exchanged, the context window only contains the system prompt, the summary, and the current buffer. The cost is an extra LLM call each time the buffer fills up, and the inherent lossiness of summarization (nuance, exact wording, and minor details may be lost). The summarizer prompt explicitly instructs the model to preserve names, preferences, and key decisions, which are the facts most likely to be needed later.

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

The production-recommended approach combines the strengths of both strategies. The last K messages are kept verbatim for full-fidelity recent context (the current topic, exact phrasing, tool call details), while everything older is compressed into a running summary that preserves long-term facts (user preferences, decisions, identities).

The hybrid memory summarizes the **older half** of the buffer when it overflows, rather than the entire buffer. This ensures that the most recent messages are never summarized prematurely -- they remain verbatim in the buffer where the model can reference exact details. The summary grows incrementally, adding new facts from each batch of summarized messages while retaining all previous summary content.

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

- [Redis Persistence](/tutorials/memory/redis-persistence) -- Persist memory across restarts
- [Research Agent](/tutorials/agents/research-agent) -- Use memory in autonomous agents
