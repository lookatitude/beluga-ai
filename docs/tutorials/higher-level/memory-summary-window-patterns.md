# Summary & Window Memory Patterns

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this tutorial, we'll implement strategies to manage conversation context windows effectively. We'll explore Window memory (keeping the last K messages) and Summary memory (maintaining a running summary), then combine them for a robust hybrid approach.

## Learning Objectives
- ✅ Implement Window Memory (Last K messages)
- ✅ Implement Summary Memory (Running summary)
- ✅ Combine both (Summary Buffer Window)

## Introduction
Welcome, colleague! LLMs have limited context windows. If you send a year's worth of chat history, you'll hit token limits and burn through your budget. Let's look at how to keep our agents' memories lean and relevant.

## Prerequisites

- [Memory Management](../../getting-started/05-memory-management.md)

## Why Not Just Buffer?

LLMs have limited context windows (e.g., 8k, 128k tokens). Passing the entire history of a year-long chat will:
1. Crash the request (context length exceeded).
2. Cost a fortune.
3. Dilute attention.

## Pattern 1: Sliding Window

Keep only the last N interactions. Good for "in-the-moment" tasks.
mem, _ := memory.NewMemory(
```
    memory.MemoryTypeBufferWindow,
    memory.WithWindowSize(5), // Last 5 pairs
)

**Pros**: predictable token usage.
**Cons**: forgets earlier important details ("My name is Alice").

## Pattern 2: Summarization

Maintain a running summary of the conversation.
// Requires an LLM to perform the summarization
mem, _ := memory.NewConversationSummaryMemory(
    history,
    llm,
    "history",
)
```

**How it works**:
1. Current Summary: "User Alice introduced herself."
2. New Interaction: "Alice asks about cats."
3. LLM updates Summary: "User Alice introduced herself and asked about cats."

**Pros**: infinite duration.
**Cons**: loss of detail/nuance; latency (requires LLM call to update).

## Pattern 3: The Hybrid (Summary + Buffer)

Keep the last K messages verbatim (for immediate context/flow) AND a summary of everything before that.
mem, _ := memory.NewConversationSummaryBufferMemory(
```
    history,
    llm,
    "history",
    2000, // Max tokens to keep in buffer before moving to summary
)

This is the gold standard for long-running agents.

## Verification

1. Use Summary Memory.
2. Feed it 50 messages.
3. Ask a question about the first message.
4. Verify the agent answers using the summary.

## Next Steps

- **[Redis Persistence](./memory-redis-persistence.md)** - Persist these complex memory structures
- **[Building a Research Agent](./agents-research-agent.md)** - Use memory in agents
