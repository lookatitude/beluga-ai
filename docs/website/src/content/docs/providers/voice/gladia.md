---
title: "Gladia"
description: "Gladia STT provider for real-time and batch speech-to-text with language detection."
---

Gladia provides speech-to-text transcription with automatic language detection, real-time WebSocket streaming, and asynchronous batch processing. The Beluga AI provider uses Gladia's v2 API for both batch (upload, create transcription, poll) and real-time (live WebSocket) workflows.

## Installation

```go
import _ "github.com/lookatitude/beluga-ai/voice/stt/providers/gladia"
```

The blank import registers the `"gladia"` provider with the STT registry.

## Configuration

| Field        | Type     | Default  | Description                                  |
|--------------|----------|----------|----------------------------------------------|
| `Language`   | `string` | —        | BCP-47 language code (auto-detected if empty)|
| `SampleRate` | `int`    | `16000`  | Audio sample rate in Hz (for streaming)      |
| `Extra`      | —        | —        | See below                                    |

### Extra Fields

| Key        | Type     | Required | Description                          |
|------------|----------|----------|--------------------------------------|
| `api_key`  | `string` | Yes      | Gladia API key                       |
| `base_url` | `string` | No       | Override base URL                    |

## Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/voice/stt"
    _ "github.com/lookatitude/beluga-ai/voice/stt/providers/gladia"
)

func main() {
    ctx := context.Background()

    engine, err := stt.New("gladia", stt.Config{
        Language: "en",
        Extra:    map[string]any{"api_key": os.Getenv("GLADIA_API_KEY")},
    })
    if err != nil {
        log.Fatal(err)
    }

    audio, err := os.ReadFile("recording.wav")
    if err != nil {
        log.Fatal(err)
    }

    text, err := engine.Transcribe(ctx, audio)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Transcript:", text)
}
```

## Direct Construction

```go
import "github.com/lookatitude/beluga-ai/voice/stt/providers/gladia"

engine, err := gladia.New(stt.Config{
    Language: "en",
    Extra:    map[string]any{"api_key": os.Getenv("GLADIA_API_KEY")},
})
```

## Batch Transcription

Batch transcription follows a three-step process handled automatically by `Transcribe`:

1. Audio is uploaded via multipart form to Gladia's upload endpoint.
2. A transcription job is created referencing the uploaded audio URL.
3. The provider polls the transcription status every 500ms until `"done"`.

The full transcript is returned as a single string from the `full_transcript` field.

## Streaming

Gladia supports native real-time streaming. The provider first initiates a live session via HTTP to obtain a WebSocket URL, then streams audio chunks and receives transcript events:

```go
for event, err := range engine.TranscribeStream(ctx, audioStream,
    stt.WithSampleRate(16000),
    stt.WithLanguage("en"),
) {
    if err != nil {
        log.Printf("stream error: %v", err)
        break
    }
    if event.IsFinal {
        fmt.Printf("[FINAL] %s (confidence=%.2f)\n", event.Text, event.Confidence)
    } else {
        fmt.Printf("[PARTIAL] %s\n", event.Text)
    }
}
```

## FrameProcessor Integration

```go
processor := stt.AsFrameProcessor(engine, stt.WithLanguage("en"))
pipeline := voice.Chain(vadProcessor, processor, llmProcessor, ttsProcessor)
```

## Advanced Features

### Per-Request Options

```go
text, err := engine.Transcribe(ctx, audio,
    stt.WithLanguage("fr"),
    stt.WithSampleRate(44100),
)
```

### Custom Endpoint

```go
engine, err := stt.New("gladia", stt.Config{
    Extra: map[string]any{
        "api_key":  os.Getenv("GLADIA_API_KEY"),
        "base_url": "https://gladia.internal.corp/v2",
    },
})
```
