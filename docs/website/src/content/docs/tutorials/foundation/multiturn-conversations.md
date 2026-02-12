---
title: Modeling Multi-Turn Conversations
description: "Build structured multi-turn conversations in Go using Beluga AI's typed messages — manage history with role tagging, multimodal content, and context window constraints."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go, tutorial, multi-turn, conversations, message history, schema, chat context"
---

LLMs are stateless — to maintain a conversation, you must send the full message history (or a relevant subset) with every request. This is a fundamental architectural constraint, not a limitation of any particular provider. Beluga AI's `schema` package provides structured message types that ensure correct role tagging, multimodal content support, and clean integration with any LLM provider. Understanding how to build and manage message history is essential for any application that goes beyond single-shot queries.

## What You Will Build

A structured multi-turn conversation using typed messages, demonstrating how to build, manage, and constrain conversation history.

## Prerequisites

- Go 1.23+
- Understanding of Go interfaces

## Message Roles

Every message in a conversation has a role that tells the LLM how to interpret its content. These roles map to the fundamental participants in an AI interaction: the system (which sets rules), the human (who asks questions), the AI (which responds), and tools (which provide external data). The typed message constructors enforce correct role assignment, preventing common errors like accidentally tagging a system prompt as user input.

| Role | Constant | Usage |
|:---|:---|:---|
| System | `schema.RoleSystem` | Sets behavior, persona, or rules |
| Human | `schema.RoleHuman` | User input |
| AI | `schema.RoleAI` | Model responses |
| Tool | `schema.RoleTool` | Results from tool executions |

## Step 1: Creating Messages

Use the factory functions to create typed messages. These functions return concrete types (`*schema.SystemMessage`, `*schema.HumanMessage`, etc.) rather than the `schema.Message` interface, giving you access to type-specific fields when needed while still satisfying the interface for inclusion in message slices.

```go
package main

import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/schema"
)

func main() {
    // System message — sets the AI's persona
    sysMsg := schema.NewSystemMessage("You are a concise data analyst.")

    // Human message — the user's query
    humanMsg := schema.NewHumanMessage("What is the average rainfall in Seattle?")

    // AI message — the model's response (in practice, this comes from the LLM)
    aiMsg := schema.NewAIMessage("Seattle receives an average of 37 inches of rain per year.")

    // Tool message — result of a tool execution
    toolMsg := schema.NewToolMessage("call_123", `{"result": 37.49}`)

    // Access content via the Text() helper
    fmt.Printf("System: %s\n", sysMsg.Text())
    fmt.Printf("User:   %s\n", humanMsg.Text())
    fmt.Printf("AI:     %s\n", aiMsg.Text())
    fmt.Printf("Tool:   %s\n", toolMsg.Text())
}
```

Each message type stores content as `[]schema.ContentPart`, supporting multimodal content (text, images, audio, video, files). The `Text()` helper extracts and concatenates all `TextPart` values.

## Step 2: Building a Conversation

A conversation is a `[]schema.Message` slice passed to the model. The slice ordering matters — models process messages sequentially and expect a natural conversation flow: system instructions first, then alternating human/AI turns. This slice-based representation keeps conversations simple to construct, inspect, and serialize.

```go
func buildConversation() []schema.Message {
    return []schema.Message{
        schema.NewSystemMessage("You are a helpful assistant."),
        schema.NewHumanMessage("Hi, I'm building a workflow."),
        schema.NewAIMessage("That sounds great! What kind of workflow?"),
        schema.NewHumanMessage("A data processing pipeline."),
    }
}
```

## Step 3: Sending to a ChatModel

Pass the conversation history to any `ChatModel`. The model receives the full history and generates a response that accounts for all prior context. The `Usage` field on the response reports token consumption, which is important for cost tracking and context window management.

```go
func chat(ctx context.Context, model llm.ChatModel) error {
    messages := buildConversation()

    // Generate a response
    resp, err := model.Generate(ctx, messages)
    if err != nil {
        return fmt.Errorf("generate failed: %w", err)
    }

    fmt.Printf("AI: %s\n", resp.Text())
    fmt.Printf("Tokens used: %d\n", resp.Usage.TotalTokens)

    // Append the response to history for the next turn
    messages = append(messages, resp)

    return nil
}
```

## Step 4: Managing a Conversation Loop

Implement a multi-turn conversation by maintaining the message history across turns. Each iteration appends the user query and AI response to the same slice, building up the full conversation context. The model sees the entire history with each call, which allows it to reference earlier parts of the conversation and maintain coherence across turns.

```go
func conversationLoop(ctx context.Context, model llm.ChatModel) error {
    messages := []schema.Message{
        schema.NewSystemMessage("You are a helpful coding assistant."),
    }

    queries := []string{
        "How do I read a file in Go?",
        "Can you show me error handling?",
        "What about concurrent file reading?",
    }

    for _, query := range queries {
        // Add user message
        messages = append(messages, schema.NewHumanMessage(query))

        // Generate response with full history
        resp, err := model.Generate(ctx, messages)
        if err != nil {
            return fmt.Errorf("generate failed: %w", err)
        }

        fmt.Printf("User: %s\n", query)
        fmt.Printf("AI:   %s\n\n", resp.Text())

        // Append AI response to maintain context
        messages = append(messages, resp)
    }

    return nil
}
```

## Step 5: Context Window Management

As conversations grow, they consume more tokens. Every LLM has a finite context window, and exceeding it causes API errors. The sliding window approach below preserves system messages (which define the agent's behavior) while trimming older conversation turns. System messages are always kept because removing them would change the agent's persona mid-conversation, leading to inconsistent behavior.

```go
func trimHistory(messages []schema.Message, maxMessages int) []schema.Message {
    if len(messages) <= maxMessages {
        return messages
    }

    // Separate system messages (always keep) from conversation messages
    var system []schema.Message
    var conversation []schema.Message

    for _, msg := range messages {
        if msg.GetRole() == schema.RoleSystem {
            system = append(system, msg)
        } else {
            conversation = append(conversation, msg)
        }
    }

    // Keep only the last N conversation messages
    maxConv := maxMessages - len(system)
    if maxConv < 0 {
        maxConv = 0
    }
    if len(conversation) > maxConv {
        conversation = conversation[len(conversation)-maxConv:]
    }

    // Reassemble: system messages first, then recent conversation
    result := make([]schema.Message, 0, len(system)+len(conversation))
    result = append(result, system...)
    result = append(result, conversation...)
    return result
}
```

Usage:

```go
// Before sending to the model, trim to the last 20 messages
messages = trimHistory(messages, 20)
resp, err := model.Generate(ctx, messages)
```

## Working with Multimodal Content

Messages can carry mixed content types. The `ContentPart` slice allows combining text and binary data in a single message, which is how vision-capable models receive images alongside text queries.

```go
// Create a message with text and an image
humanMsg := &schema.HumanMessage{
    Parts: []schema.ContentPart{
        schema.TextPart{Text: "What's in this image?"},
        schema.ImagePart{
            URL:      "https://example.com/photo.jpg",
            MimeType: "image/jpeg",
        },
    },
}
```

## AI Messages with Tool Calls

When a model requests tool invocations, the `AIMessage` carries `ToolCalls`. This is the mechanism behind agentic behavior — the model decides which tools to call and with what arguments, and your code executes them and feeds results back as `ToolMessage` entries. The `ID` field links each tool result back to the specific call that requested it.

```go
// AI response with tool calls (returned by the model)
aiResp := &schema.AIMessage{
    Parts:     []schema.ContentPart{schema.TextPart{Text: "Let me look that up."}},
    ToolCalls: []schema.ToolCall{
        {ID: "call_1", Name: "search", Arguments: `{"query": "Go concurrency"}`},
    },
    Usage: schema.Usage{InputTokens: 50, OutputTokens: 20, TotalTokens: 70},
}

// After executing the tool, provide the result
toolResult := schema.NewToolMessage("call_1", `{"results": ["goroutines", "channels"]}`)
```

## Troubleshooting

**Messages appear to lose context**: Ensure you append each AI response to the message history before generating the next turn. LLMs are stateless — they only know what you send them.

**Token limits exceeded**: Implement context window management (Step 5) to keep message counts or token counts within the model's limits. Consider summarization strategies for long-running conversations.

## Next Steps

- [Custom Message Types](/tutorials/foundation/custom-message-types) — Extend the message system with structured data
- [Redis Memory Persistence](/tutorials/memory/redis-persistence) — Persist conversation history across restarts
