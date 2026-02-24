---
title: "Gemini Live S2S Voice Provider"
description: "Google Gemini Live S2S for bidirectional audio streaming in Beluga AI. Multimodal speech-to-speech with tool calling and real-time AI in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Gemini Live, speech-to-speech, S2S, Google AI, multimodal, bidirectional audio, Go, Beluga AI"
---

Gemini Live provides native speech-to-speech via the Google Gemini Live API over WebSocket, enabling bidirectional audio streaming with Gemini's multimodal models. The provider supports audio and text output, function calling, and server-side voice activity detection.

Choose Gemini Live when you want native speech-to-speech with Google's multimodal Gemini models. Gemini Live handles voice activity detection server-side, simplifying client implementation. It supports function calling and bidirectional audio over WebSocket. For OpenAI's equivalent, consider [OpenAI Realtime](/docs/providers/voice/openai-realtime). For AWS-native S2S, consider [Amazon Nova S2S](/docs/providers/voice/nova-s2s).

## Installation

```go
import _ "github.com/lookatitude/beluga-ai/voice/s2s/providers/gemini"
```

The blank import registers the `"gemini_live"` provider with the S2S registry.

## Configuration

| Field          | Type                      | Default                    | Description                          |
|----------------|---------------------------|----------------------------|--------------------------------------|
| `Voice`        | `string`                  | —                          | Prebuilt voice name                  |
| `Model`        | `string`                  | `"gemini-2.0-flash-exp"`  | Gemini model identifier              |
| `Instructions` | `string`                  | —                          | System instruction for the session   |
| `Tools`        | `[]schema.ToolDefinition` | —                          | Tools available to the model         |
| `Extra`        | —                         | —                          | See below                            |

### Extra Fields

| Key        | Type     | Required | Description                          |
|------------|----------|----------|--------------------------------------|
| `api_key`  | `string` | Yes      | Google AI API key                    |
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
    _ "github.com/lookatitude/beluga-ai/voice/s2s/providers/gemini"
)

func main() {
    ctx := context.Background()

    engine, err := s2s.New("gemini_live", s2s.Config{
        Model: "gemini-2.0-flash-exp",
        Extra: map[string]any{"api_key": os.Getenv("GOOGLE_API_KEY")},
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
import "github.com/lookatitude/beluga-ai/voice/s2s/providers/gemini"

engine, err := gemini.New(s2s.Config{
    Model:        "gemini-2.0-flash-exp",
    Instructions: "You are a helpful voice assistant.",
    Extra:        map[string]any{"api_key": os.Getenv("GOOGLE_API_KEY")},
})
```

## Session Lifecycle

1. **Start**: Opens a WebSocket connection with the API key as a query parameter and sends a setup message configuring the model, generation config (audio modalities, voice), system instructions, and tools.
2. **Audio I/O**: Audio is sent via `SendAudio` as base64-encoded PCM (16kHz) in `realtimeInput.mediaChunks`. Output audio arrives as base64-encoded `inlineData` in `serverContent.modelTurn.parts`.
3. **Text**: Send text via `SendText` using `clientContent.turns` with `turnComplete: true`.
4. **Tool Calls**: The model emits function calls via `toolCall.functionCalls`. Send results back via `SendToolResult` using the `toolResponse` message format.
5. **Interrupt**: Gemini Live handles interruptions via server-side VAD. The `Interrupt` method is a no-op since the server detects user speech automatically.
6. **Close**: Terminates the WebSocket connection.

## Tool Calling

```go
session, err := engine.Start(ctx,
    s2s.WithInstructions("You are a helpful assistant."),
    s2s.WithTools([]schema.ToolDefinition{
        {
            Name:        "search",
            Description: "Search the web for information",
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
        result := executeSearch(event.ToolCall.Arguments)
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
    s2s.WithModel("gemini-2.0-flash-exp"),
    s2s.WithInstructions("You are a helpful voice assistant."),
)
pipeline := voice.Chain(vadProcessor, processor)
```

## Advanced Features

### Voice Configuration

The voice is set in the setup message under `generationConfig.speechConfig.voiceConfig.prebuiltVoiceConfig.voiceName`:

```go
session, err := engine.Start(ctx,
    s2s.WithVoice("Charon"),
    s2s.WithModel("gemini-2.0-flash-exp"),
)
```

### Audio Format

Audio is exchanged in PCM format at 16kHz. The provider handles base64 encoding (sending) and decoding (receiving) transparently. The MIME type is `audio/pcm;rate=16000`.

### Server-Side VAD

Gemini Live uses server-side voice activity detection. The model automatically detects when the user stops speaking, making the `Interrupt` method a no-op. User interruptions are handled by the server detecting new speech input.

### Per-Session Options

```go
session, err := engine.Start(ctx,
    s2s.WithVoice("Kore"),
    s2s.WithModel("gemini-2.0-flash-exp"),
    s2s.WithInstructions("You are a concise assistant. Keep responses under 30 seconds."),
)
```

### Custom Endpoint

```go
engine, err := s2s.New("gemini_live", s2s.Config{
    Model: "gemini-2.0-flash-exp",
    Extra: map[string]any{
        "api_key":  os.Getenv("GOOGLE_API_KEY"),
        "base_url": "wss://custom-gemini-endpoint.example.com/ws",
    },
})
```
