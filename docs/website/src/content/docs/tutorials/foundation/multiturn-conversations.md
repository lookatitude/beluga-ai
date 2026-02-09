---
title: Modeling Multi-turn Conversations
description: Build structured conversation flows using Beluga AI's schema message types and manage conversation history.
---

LLMs are stateless — to maintain a conversation, you must send the full message history (or a relevant subset) with every request. Beluga AI's `schema` package provides structured message types that ensure correct role tagging, multimodal content support, and clean integration with any LLM provider.

## What You Will Build

A structured multi-turn conversation using typed messages, demonstrating how to build, manage, and constrain conversation history.

## Prerequisites

- Go 1.23+
- Understanding of Go interfaces

## Message Roles

Every message in a conversation has a role that tells the LLM how to interpret its content:

| Role | Constant | Usage |
|:---|:---|:---|
| System | `schema.RoleSystem` | Sets behavior, persona, or rules |
| Human | `schema.RoleHuman` | User input |
| AI | `schema.RoleAI` | Model responses |
| Tool | `schema.RoleTool` | Results from tool executions |

## Step 1: Creating Messages

Use the factory functions to create typed messages:

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

A conversation is a `[]schema.Message` slice passed to the model:

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

Pass the conversation history to any `ChatModel`:

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

Implement a multi-turn conversation by maintaining the message history across turns:

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

As conversations grow, they consume more tokens. Implement a sliding window to keep the most recent messages while preserving the system prompt:

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

Messages can carry mixed content types:

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

When a model requests tool invocations, the `AIMessage` carries `ToolCalls`:

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
