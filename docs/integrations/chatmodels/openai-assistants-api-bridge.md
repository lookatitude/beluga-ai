# OpenAI Assistants API Bridge

Welcome, colleague! In this integration guide, we're going to integrate OpenAI Assistants API with Beluga AI's chatmodels package. OpenAI Assistants provides persistent, stateful AI assistants with tools, file search, and code execution.

## What you will build

You will create a bridge that connects OpenAI Assistants API with Beluga AI's ChatModel interface, enabling you to use Assistants' advanced features (tools, file search, code execution) within Beluga AI's unified interface.

## Learning Objectives

- ✅ Configure OpenAI Assistants API
- ✅ Create and manage assistants
- ✅ Use assistants with Beluga AI ChatModel interface
- ✅ Understand assistants-specific features

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- OpenAI API key with Assistants API access
- Understanding of OpenAI Assistants

## Step 1: Setup and Installation

Install OpenAI Go SDK:
bash
```bash
go get github.com/openai/openai-go
```

Set environment variable:
bash
```bash
export OPENAI_API_KEY="sk-..."
```

## Step 2: Create Assistant Bridge

Create a bridge between Assistants API and Beluga AI:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/chatmodels/iface"
    "github.com/lookatitude/beluga-ai/pkg/schema"
    "github.com/openai/openai-go"
    "github.com/openai/openai-go/option"
)

type AssistantsBridge struct {
    client    *openai.Client
    assistantID string
}

func NewAssistantsBridge(apiKey, assistantID string) (*AssistantsBridge, error) {
    client := openai.NewClient(option.WithAPIKey(apiKey))
    
    return &AssistantsBridge{
        client:      client,
        assistantID: assistantID,
    }, nil
}

func (b *AssistantsBridge) GenerateMessages(ctx context.Context, messages []schema.Message) ([]schema.Message, error) {
    // Convert Beluga messages to OpenAI format
    threadMessages := make([]openai.ThreadMessageParam, 0)
    for _, msg := range messages {
        threadMessages = append(threadMessages, openai.ThreadMessageParam{
            Role:    string(msg.GetRole()),
            Content: msg.GetContent(),
        })
    }
    
    // Create thread
    thread, err := b.client.Threads.New(ctx, openai.ThreadCreateParams{
        Messages: threadMessages,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create thread: %w", err)
    }
    
    // Run assistant
    run, err := b.client.ThreadsRuns.New(ctx, thread.ID, openai.ThreadRunCreateParams{
        AssistantID: b.assistantID,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create run: %w", err)
    }
    
    // Wait for completion
    for run.Status == "queued" || run.Status == "in_progress" {
        run, err = b.client.ThreadsRuns.Retrieve(ctx, thread.ID, run.ID)
        if err != nil {
            return nil, fmt.Errorf("failed to retrieve run: %w", err)
        }
    }
    
    // Get messages
    threadMessagesResp, err := b.client.ThreadsMessages.NewList(ctx, thread.ID)
    if err != nil {
        return nil, fmt.Errorf("failed to list messages: %w", err)
    }
    
    // Convert to Beluga format
    result := make([]schema.Message, 0)
    for _, msg := range threadMessagesResp.Data {
        if msg.Role == "assistant" {
            for _, content := range msg.Content {
                if text, ok := content.(*openai.MessageContentText); ok {
                    result = append(result, schema.NewAIMessage(text.Text.Value))
                }
            }
        }
    }
    
    return result, nil
}
```

## Step 3: Create Assistant

Create an assistant in OpenAI:
```go
func createAssistant(ctx context.Context, client *openai.Client) (string, error) {
    assistant, err := client.Assistants.New(ctx, openai.AssistantCreateParams{
        Model: openai.F(openai.String("gpt-4")),
        Name:  openai.F(openai.String("Beluga AI Assistant")),
        Instructions: openai.F(openai.String("You are a helpful AI assistant.")),
    })
    if err != nil {
        return "", fmt.Errorf("failed to create assistant: %w", err)
    }

    
    return assistant.ID, nil
}
```

## Step 4: Use with Beluga AI

Integrate with Beluga AI ChatModel:
```go
type AssistantsChatModel struct \{
    bridge *AssistantsBridge
}
go
func NewAssistantsChatModel(apiKey, assistantID string) (*AssistantsChatModel, error) {
    bridge, err := NewAssistantsBridge(apiKey, assistantID)
    if err != nil {
        return nil, err
    }
    
    return &AssistantsChatModel{bridge: bridge}, nil
}

func (m *AssistantsChatModel) GenerateMessages(ctx context.Context, messages []schema.Message) ([]schema.Message, error) {
    return m.bridge.GenerateMessages(ctx, messages)
}

func main() {
    ctx := context.Background()
    
    // Create assistant
    client := openai.NewClient(option.WithAPIKey(os.Getenv("OPENAI_API_KEY")))
    assistantID, err := createAssistant(ctx, client)
    if err != nil {
        log.Fatalf("Failed to create assistant: %v", err)
    }
    
    // Create chat model
    model, err := NewAssistantsChatModel(os.Getenv("OPENAI_API_KEY"), assistantID)
    if err != nil {
        log.Fatalf("Failed to create model: %v", err)
    }
    
    // Use it
    messages := []schema.Message{
        schema.NewHumanMessage("What is machine learning?"),
    }
    
    responses, err := model.GenerateMessages(ctx, messages)
    if err != nil {
        log.Fatalf("Failed to generate: %v", err)
    }
    
    for _, msg := range responses {
        fmt.Printf("Response: %s\n", msg.GetContent())
    }
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `APIKey` | OpenAI API key | - | Yes |
| `AssistantID` | Assistant ID | - | Yes |
| `Model` | Assistant model | `gpt-4` | No |
| `Instructions` | Assistant instructions | - | No |

## Common Issues

### "Assistant not found"

**Problem**: Invalid assistant ID.

**Solution**: Create assistant first or verify ID:assistantID, err := createAssistant(ctx, client)
```

### "Thread creation failed"

**Problem**: API key or permissions issue.

**Solution**: Verify API key has Assistants API access.

## Production Considerations

When using Assistants API in production:

- **Thread management**: Reuse threads when possible
- **Cost monitoring**: Assistants API has different pricing
- **State management**: Assistants maintain state across runs
- **Tool integration**: Leverage assistants' tool capabilities
- **File search**: Use assistants' file search for RAG

## Next Steps

Congratulations! You've integrated OpenAI Assistants with Beluga AI. Next, learn how to:

- **[Custom Mock for UI Testing](./custom-mock-ui-testing.md)** - Create mocks for testing
- **[ChatModels Package Documentation](../../api-docs/packages/chatmodels.md)** - Deep dive into chatmodels
- **[Agents Guide](../../guides/agent-types.md)** - Agent patterns

---

**Ready for more?** Check out the Integrations Index for more integration guides!
