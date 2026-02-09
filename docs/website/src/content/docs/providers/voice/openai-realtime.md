---
title: "OpenAI Realtime"
description: "OpenAI Realtime S2S provider for bidirectional audio streaming with GPT-4o."
---

OpenAI Realtime provides native speech-to-speech via the OpenAI Realtime API over WebSocket, bypassing the traditional STT/LLM/TTS cascade for lower end-to-end latency. The provider supports bidirectional audio streaming, server-side voice activity detection (VAD), tool calling, and text transcripts of both user and agent speech.

## Installation

```go
import _ "github.com/lookatitude/beluga-ai/voice/s2s/providers/openai"
```

The blank import registers the `"openai_realtime"` provider with the S2S registry.

## Configuration

| Field          | Type                      | Default                        | Description                          |
|----------------|---------------------------|--------------------------------|--------------------------------------|
| `Voice`        | `string`                  | `"alloy"`                      | Voice (alloy, echo, shimmer, etc.)   |
| `Model`        | `string`                  | `"gpt-4o-realtime-preview"`    | Realtime model identifier            |
| `Instructions` | `string`                  | —                              | System prompt for the session        |
| `Tools`        | `[]schema.ToolDefinition` | —                              | Tools available to the model         |
| `Extra`        | —                         | —                              | See below                            |

### Extra Fields

| Key        | Type     | Required | Description                          |
|------------|----------|----------|--------------------------------------|
| `api_key`  | `string` | Yes      | OpenAI API key (`sk-...`)            |
| `base_url` | `string` | No       | Override WebSocket URL               |

## Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/voice/s2s"
    _ "github.com/lookatitude/beluga-ai/voice/s2s/providers/openai"
)

func main() {
    ctx := context.Background()

    engine, err := s2s.New("openai_realtime", s2s.Config{
        Voice: "alloy",
        Model: "gpt-4o-realtime-preview",
        Extra: map[string]any{"api_key": os.Getenv("OPENAI_API_KEY")},
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
import openai_realtime "github.com/lookatitude/beluga-ai/voice/s2s/providers/openai"

engine, err := openai_realtime.New(s2s.Config{
    Voice:        "alloy",
    Model:        "gpt-4o-realtime-preview",
    Instructions: "You are a helpful voice assistant.",
    Extra:        map[string]any{"api_key": os.Getenv("OPENAI_API_KEY")},
})
```

## Session Lifecycle

1. **Start**: Opens a WebSocket connection and sends a `session.update` message with voice, modalities, audio format, and VAD configuration.
2. **Audio I/O**: Audio is sent via `SendAudio` (base64-encoded PCM16) and received as `EventAudioOutput` events (decoded from base64).
3. **Text**: Send text messages via `SendText`, which creates a conversation item and triggers a response.
4. **Tool Calls**: The model emits `EventToolCall` events. Send results back via `SendToolResult`.
5. **Interrupt**: Call `Interrupt` to cancel the current model response (sends `response.cancel`).
6. **Close**: Terminates the WebSocket connection.

## Tool Calling

The OpenAI Realtime API supports tool calling within S2S sessions:

```go
session, err := engine.Start(ctx,
    s2s.WithInstructions("You are a helpful assistant with access to tools."),
    s2s.WithTools([]schema.ToolDefinition{
        {
            Name:        "get_weather",
            Description: "Get current weather for a location",
            InputSchema: map[string]any{
                "type": "object",
                "properties": map[string]any{
                    "location": map[string]any{"type": "string"},
                },
                "required": []string{"location"},
            },
        },
    }),
)

for event := range session.Recv() {
    if event.Type == s2s.EventToolCall {
        result := executeWeatherTool(event.ToolCall.Arguments)
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

Wrap the S2S engine as a `FrameProcessor` for use in a voice pipeline. This is the recommended pattern for the hybrid pipeline mode:

```go
processor := s2s.AsFrameProcessor(engine,
    s2s.WithVoice("alloy"),
    s2s.WithInstructions("You are a helpful voice assistant."),
)
pipeline := voice.Chain(vadProcessor, processor)
```

The `AsFrameProcessor` wrapper handles:
- Forwarding audio frames to `SendAudio`
- Forwarding text frames to `SendText`
- Converting control frames (interrupt signal) to `Interrupt` calls
- Emitting audio output, text output, and end-of-utterance control frames

## Advanced Features

### Server-Side VAD

The provider configures server-side voice activity detection by default (`"turn_detection": {"type": "server_vad"}`). The server detects when the user stops speaking and automatically triggers a response.

### Audio Format

Audio is exchanged in PCM16 format (16-bit linear PCM). The provider handles base64 encoding/decoding transparently.

### Per-Session Options

```go
session, err := engine.Start(ctx,
    s2s.WithVoice("echo"),
    s2s.WithModel("gpt-4o-realtime-preview"),
    s2s.WithInstructions("Respond concisely."),
)
```

### Custom Endpoint

```go
engine, err := s2s.New("openai_realtime", s2s.Config{
    Voice: "alloy",
    Extra: map[string]any{
        "api_key":  os.Getenv("OPENAI_API_KEY"),
        "base_url": "wss://custom-realtime.openai.azure.com/v1/realtime",
    },
})
```
