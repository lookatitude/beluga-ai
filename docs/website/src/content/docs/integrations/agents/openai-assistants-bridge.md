---
title: OpenAI Assistants API Bridge
description: Bridge OpenAI's Assistants API with Beluga AI's ChatModel interface to use persistent, stateful assistants with tools, file search, and code execution.
---

If your organization has already invested in OpenAI Assistants -- with configured tools, uploaded files, and tuned instructions -- you do not need to recreate that work in Beluga AI. This bridge wraps the Assistants API behind Beluga's `llm.ChatModel` interface so you can use existing assistants within Beluga's agent orchestration, middleware, and routing systems.

This is useful when you want to gradually migrate to Beluga AI, combine Assistants with other LLM providers via the Router, or apply Beluga's middleware (logging, retry, caching) to Assistant calls.

## Overview

The bridge wraps the OpenAI Assistants API (threads, runs, messages) behind a `ChatModel`-compatible interface. This enables:

- Using OpenAI Assistants as a drop-in LLM provider within Beluga agents
- Accessing assistant-specific features (file search, code interpreter) through the unified interface
- Combining assistants with Beluga's middleware, hooks, and routing capabilities

## Prerequisites

- Go 1.23 or later
- Beluga AI framework installed (`go get github.com/lookatitude/beluga-ai`)
- OpenAI API key with Assistants API access
- Familiarity with OpenAI's Assistants API concepts (threads, runs, messages)

## Installation

```bash
go get github.com/lookatitude/beluga-ai
go get github.com/openai/openai-go
```

Set your API key:

```bash
export OPENAI_API_KEY="sk-..."
```

## Building the Assistants Bridge

### Step 1: Define the Bridge Structure

The bridge manages the OpenAI client, assistant ID, and thread lifecycle.

```go
package main

import (
    "context"
    "fmt"
    "iter"
    "log"
    "os"
    "time"

    "github.com/lookatitude/beluga-ai/schema"
    "github.com/openai/openai-go"
    "github.com/openai/openai-go/option"
)

// AssistantsBridge wraps OpenAI Assistants API as a Beluga-compatible provider.
type AssistantsBridge struct {
    client      *openai.Client
    assistantID string
    model       string
}

// NewAssistantsBridge creates a bridge targeting an existing assistant.
func NewAssistantsBridge(apiKey, assistantID string) *AssistantsBridge {
    client := openai.NewClient(option.WithAPIKey(apiKey))
    return &AssistantsBridge{
        client:      client,
        assistantID: assistantID,
        model:       "gpt-4o",
    }
}
```

### Step 2: Implement Generate

Map Beluga's message format to OpenAI thread messages, create a run, and poll for completion.

```go
// Generate sends messages through the Assistants API and returns the response.
func (b *AssistantsBridge) Generate(
    ctx context.Context,
    msgs []schema.Message,
    opts ...func(*generateOpts),
) (*schema.AIMessage, error) {
    // Convert Beluga messages to thread messages
    threadMessages := make([]openai.ThreadMessageParam, 0, len(msgs))
    for _, msg := range msgs {
        role := "user"
        if msg.Role() == schema.RoleAI {
            role = "assistant"
        }
        threadMessages = append(threadMessages, openai.ThreadMessageParam{
            Role:    role,
            Content: messageContent(msg),
        })
    }

    // Create a thread with the messages
    thread, err := b.client.Threads.New(ctx, openai.ThreadCreateParams{
        Messages: threadMessages,
    })
    if err != nil {
        return nil, fmt.Errorf("assistants: create thread: %w", err)
    }

    // Create a run
    run, err := b.client.ThreadsRuns.New(ctx, thread.ID, openai.ThreadRunCreateParams{
        AssistantID: b.assistantID,
    })
    if err != nil {
        return nil, fmt.Errorf("assistants: create run: %w", err)
    }

    // Poll until completion
    for run.Status == "queued" || run.Status == "in_progress" {
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        case <-time.After(500 * time.Millisecond):
        }
        run, err = b.client.ThreadsRuns.Retrieve(ctx, thread.ID, run.ID)
        if err != nil {
            return nil, fmt.Errorf("assistants: poll run: %w", err)
        }
    }

    if run.Status != "completed" {
        return nil, fmt.Errorf("assistants: run ended with status %s", run.Status)
    }

    // Retrieve assistant messages
    msgList, err := b.client.ThreadsMessages.NewList(ctx, thread.ID)
    if err != nil {
        return nil, fmt.Errorf("assistants: list messages: %w", err)
    }

    // Extract the latest assistant response
    for _, msg := range msgList.Data {
        if msg.Role == "assistant" {
            for _, content := range msg.Content {
                if text, ok := content.AsText(); ok {
                    return schema.NewAIMessage(
                        schema.WithText(text.Text.Value),
                    ), nil
                }
            }
        }
    }

    return schema.NewAIMessage(schema.WithText("")), nil
}
```

### Step 3: Create an Assistant Programmatically

If you do not have an existing assistant, create one with the OpenAI API.

```go
func createAssistant(ctx context.Context, client *openai.Client) (string, error) {
    assistant, err := client.Assistants.New(ctx, openai.AssistantCreateParams{
        Model:        "gpt-4o",
        Name:         "Beluga AI Assistant",
        Instructions: "You are a helpful AI assistant integrated with Beluga AI.",
    })
    if err != nil {
        return "", fmt.Errorf("create assistant: %w", err)
    }
    return assistant.ID, nil
}
```

### Step 4: Wrap as a ChatModel

Implement the full `llm.ChatModel` interface so the bridge can be used anywhere a `ChatModel` is expected.

```go
import (
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
)

// AssistantsChatModel adapts AssistantsBridge to the ChatModel interface.
type AssistantsChatModel struct {
    bridge *AssistantsBridge
}

func NewAssistantsChatModel(apiKey, assistantID string) *AssistantsChatModel {
    return &AssistantsChatModel{
        bridge: NewAssistantsBridge(apiKey, assistantID),
    }
}

func (m *AssistantsChatModel) Generate(
    ctx context.Context,
    msgs []schema.Message,
    opts ...llm.GenerateOption,
) (*schema.AIMessage, error) {
    return m.bridge.Generate(ctx, msgs)
}

func (m *AssistantsChatModel) Stream(
    ctx context.Context,
    msgs []schema.Message,
    opts ...llm.GenerateOption,
) iter.Seq2[schema.StreamChunk, error] {
    // Assistants API does not natively support streaming in the same way.
    // Wrap Generate as a single-chunk stream.
    return func(yield func(schema.StreamChunk, error) bool) {
        result, err := m.Generate(ctx, msgs)
        if err != nil {
            yield(schema.StreamChunk{}, err)
            return
        }
        yield(schema.StreamChunk{Delta: result.Text()}, nil)
    }
}

func (m *AssistantsChatModel) BindTools(tools []schema.ToolDefinition) llm.ChatModel {
    // Tool binding is handled on the assistant side via OpenAI's dashboard
    // or API. This is a no-op for the bridge.
    return m
}

func (m *AssistantsChatModel) ModelID() string {
    return "openai-assistants:" + m.bridge.assistantID
}
```

### Step 5: Use in an Agent

```go
func main() {
    ctx := context.Background()

    apiKey := os.Getenv("OPENAI_API_KEY")

    // Create or reference an existing assistant
    client := openai.NewClient(option.WithAPIKey(apiKey))
    assistantID, err := createAssistant(ctx, client)
    if err != nil {
        log.Fatalf("create assistant: %v", err)
    }

    // Use the bridge as a ChatModel
    model := NewAssistantsChatModel(apiKey, assistantID)

    result, err := model.Generate(ctx, []schema.Message{
        schema.NewHumanMessage("What is machine learning?"),
    })
    if err != nil {
        log.Fatalf("generate: %v", err)
    }

    fmt.Println(result.Text())
}
```

## Advanced Topics

### Thread Reuse

Reuse threads across calls to maintain conversation history on the OpenAI side. Store the thread ID and pass it to subsequent calls instead of creating a new thread each time.

```go
type StatefulBridge struct {
    bridge   *AssistantsBridge
    threadID string
}

func (s *StatefulBridge) Generate(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
    if s.threadID == "" {
        // Create thread on first call
        thread, err := s.bridge.client.Threads.New(ctx, openai.ThreadCreateParams{})
        if err != nil {
            return nil, fmt.Errorf("create thread: %w", err)
        }
        s.threadID = thread.ID
    }

    // Add messages to existing thread and run
    // ...
    return nil, nil
}
```

### Assistants with File Search

Configure assistants with file search capabilities through the OpenAI API, then use them via the bridge.

```go
assistant, err := client.Assistants.New(ctx, openai.AssistantCreateParams{
    Model:        "gpt-4o",
    Name:         "Document Assistant",
    Instructions: "Search uploaded files to answer questions.",
    Tools: []openai.AssistantToolParam{
        {Type: "file_search"},
    },
})
```

### Combining with Beluga Middleware

Because the bridge implements `ChatModel`, apply Beluga's LLM middleware for observability, caching, or fallback.

```go
var model llm.ChatModel = NewAssistantsChatModel(apiKey, assistantID)

model = llm.ApplyMiddleware(model,
    llm.WithLogging(logger),
    llm.WithRetry(3),
)
```

## Configuration Reference

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| API Key | OpenAI API key | - | Yes |
| Assistant ID | Existing assistant ID | - | Yes (or create one) |
| Model | Assistant's LLM model | `gpt-4o` | No |
| Instructions | System instructions for the assistant | - | No |

## Troubleshooting

### Assistant not found

**Problem**: Invalid or deleted assistant ID.

**Solution**: Verify the assistant ID exists in your OpenAI account. Create a new assistant if needed using `client.Assistants.New()`.

### Thread creation failed

**Problem**: API key lacks Assistants API permissions.

**Solution**: Confirm your API key has access to the Assistants API. Check the OpenAI dashboard for quota and billing status.

### Run stuck in "queued" status

**Problem**: The polling loop does not complete.

**Solution**: Use a context with a timeout to prevent indefinite polling. Check the OpenAI status page for service disruptions.

```go
ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
defer cancel()
```

## Related Resources

- [LLM Providers](/integrations/llm-providers) — Built-in provider options including OpenAI
- [Mock ChatModel for Testing](/integrations/mock-ui-testing) — Test without real API calls
- [Agent System](/guides/agents) — Using ChatModel providers in agents
