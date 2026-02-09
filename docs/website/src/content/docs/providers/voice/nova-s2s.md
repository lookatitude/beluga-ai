---
title: "Amazon Nova S2S"
description: "Amazon Nova Sonic S2S provider for bidirectional audio streaming via AWS Bedrock."
---

Amazon Nova provides native speech-to-speech via the AWS Bedrock Runtime API with Nova Sonic. The provider connects via WebSocket for bidirectional audio streaming, supporting tool calling, input transcription, and user interruption handling.

## Installation

```go
import _ "github.com/lookatitude/beluga-ai/voice/s2s/providers/nova"
```

The blank import registers the `"nova"` provider with the S2S registry.

## Configuration

| Field          | Type                      | Default                      | Description                          |
|----------------|---------------------------|------------------------------|--------------------------------------|
| `Model`        | `string`                  | `"amazon.nova-sonic-v1:0"`   | Nova model identifier                |
| `Instructions` | `string`                  | —                            | System prompt for the session        |
| `Tools`        | `[]schema.ToolDefinition` | —                            | Tools available to the model         |
| `Extra`        | —                         | —                            | See below                            |

### Extra Fields

| Key        | Type     | Required | Description                              |
|------------|----------|----------|------------------------------------------|
| `region`   | `string` | No       | AWS region (default: `"us-east-1"`)      |
| `base_url` | `string` | No       | Override WebSocket URL                   |

## Authentication

Amazon Nova uses AWS IAM authentication. Ensure your AWS credentials are configured via environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`) or an IAM instance profile. The provider constructs the WebSocket URL using the configured region and model.

## Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/voice/s2s"
    _ "github.com/lookatitude/beluga-ai/voice/s2s/providers/nova"
)

func main() {
    ctx := context.Background()

    engine, err := s2s.New("nova", s2s.Config{
        Model: "amazon.nova-sonic-v1:0",
        Extra: map[string]any{"region": "us-east-1"},
    })
    if err != nil {
        log.Fatal(err)
    }

    session, err := engine.Start(ctx)
    if err != nil {
        log.Fatal(err)
    }
    defer session.Close()

    // Send audio to the session
    if err := session.SendAudio(ctx, audioChunk); err != nil {
        log.Fatal(err)
    }

    // Receive events
    for event := range session.Recv() {
        switch event.Type {
        case s2s.EventAudioOutput:
            playAudio(event.Audio)
        case s2s.EventTextOutput:
            fmt.Printf("Agent: %s", event.Text)
        case s2s.EventTranscript:
            fmt.Printf("User said: %s\n", event.Text)
        case s2s.EventToolCall:
            fmt.Printf("Tool call: %s(%s)\n", event.ToolCall.Name, event.ToolCall.Arguments)
        case s2s.EventTurnEnd:
            fmt.Println("--- turn complete ---")
        case s2s.EventError:
            log.Printf("error: %v", event.Error)
        }
    }
}
```

## Direct Construction

```go
import "github.com/lookatitude/beluga-ai/voice/s2s/providers/nova"

engine, err := nova.New(s2s.Config{
    Model:        "amazon.nova-sonic-v1:0",
    Instructions: "You are a helpful voice assistant.",
    Extra:        map[string]any{"region": "us-west-2"},
})
```

## Session Lifecycle

1. **Start**: Opens a WebSocket connection to the Bedrock Runtime endpoint and sends a `sessionStart` message with inference configuration, system instructions, and tool config.
2. **Audio I/O**: Audio is sent via `SendAudio` as base64-encoded chunks in `inputAudio` messages. Output audio arrives as base64-encoded `audioChunk` fields in `contentBlockDelta` events.
3. **Text**: Send text via `SendText` using `inputText` messages with a content array.
4. **Tool Calls**: The model emits `toolUse` events with tool use ID, name, and input. Send results back via `SendToolResult` using the `toolResult` message format with status.
5. **Interrupt**: Call `Interrupt` to send an `inputAudioInterrupt` message, signaling that the user is speaking over the agent.
6. **Close**: Terminates the WebSocket connection.

## Tool Calling

```go
session, err := engine.Start(ctx,
    s2s.WithInstructions("You have access to a knowledge base."),
    s2s.WithTools([]schema.ToolDefinition{
        {
            Name:        "lookup",
            Description: "Look up information in the knowledge base",
            InputSchema: map[string]any{
                "type": "object",
                "properties": map[string]any{
                    "query": map[string]any{"type": "string"},
                },
                "required": []string{"query"},
            },
        },
    }),
)

for event := range session.Recv() {
    if event.Type == s2s.EventToolCall {
        result := lookupKnowledgeBase(event.ToolCall.Arguments)
        if err := session.SendToolResult(ctx, schema.ToolResult{
            CallID:  event.ToolCall.ID,
            Content: []schema.ContentPart{schema.TextPart{Text: result}},
        }); err != nil {
            log.Printf("send tool result: %v", err)
        }
    }
}
```

## FrameProcessor Integration

```go
processor := s2s.AsFrameProcessor(engine,
    s2s.WithModel("amazon.nova-sonic-v1:0"),
    s2s.WithInstructions("You are a helpful voice assistant."),
)
pipeline := voice.Chain(vadProcessor, processor)
```

## Advanced Features

### User Interruption

Nova supports explicit user interruption via the `inputAudioInterrupt` message type:

```go
if err := session.Interrupt(ctx); err != nil {
    log.Printf("interrupt error: %v", err)
}
```

### Input Transcription

Nova emits `inputTranscript` events with the user's speech transcribed to text, accessible via `EventTranscript` events in the session.

### AWS Region Selection

The WebSocket endpoint is constructed from the region and model: `wss://bedrock-runtime.{region}.amazonaws.com/model/{model}/converse-stream`.

```go
engine, err := s2s.New("nova", s2s.Config{
    Model: "amazon.nova-sonic-v1:0",
    Extra: map[string]any{"region": "eu-west-1"},
})
```

### Custom Endpoint

For VPC endpoints or custom deployments:

```go
engine, err := s2s.New("nova", s2s.Config{
    Model: "amazon.nova-sonic-v1:0",
    Extra: map[string]any{
        "region":   "us-east-1",
        "base_url": "wss://bedrock.vpce.internal.corp/model/amazon.nova-sonic-v1:0/converse-stream",
    },
})
```

### Per-Session Options

```go
session, err := engine.Start(ctx,
    s2s.WithModel("amazon.nova-sonic-v1:0"),
    s2s.WithInstructions("You are a customer service agent."),
    s2s.WithTools(toolDefs),
)
```
