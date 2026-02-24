---
title: "OpenAI Whisper Voice Provider"
description: "OpenAI Whisper STT for batch speech-to-text in Beluga AI. High-accuracy multilingual transcription via the Audio Transcriptions API in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "OpenAI Whisper, speech-to-text, STT, batch transcription, multilingual, audio API, Go, Beluga AI"
---

OpenAI Whisper provides highly accurate batch speech-to-text transcription through the OpenAI Audio Transcriptions API. The Beluga AI provider uploads audio as multipart form data and returns the transcribed text. Whisper does not support native streaming; the streaming interface transcribes each audio chunk independently as a batch request.

Choose Whisper when you already use the OpenAI API and need reliable batch transcription without adding another vendor. Whisper excels at accuracy across many languages but does not provide real-time interim results. For real-time streaming with partial transcripts, use [Deepgram](/providers/voice/deepgram) or [AssemblyAI](/providers/voice/assemblyai).

## Installation

```go
import _ "github.com/lookatitude/beluga-ai/voice/stt/providers/whisper"
```

The blank import registers the `"whisper"` provider with the STT registry.

## Configuration

| Field      | Type     | Default       | Description                              |
|------------|----------|---------------|------------------------------------------|
| `Language` | `string` | —             | ISO-639 language code (e.g., `"en"`)     |
| `Model`    | `string` | `"whisper-1"` | Whisper model identifier                 |
| `Extra`    | —        | —             | See below                                |

### Extra Fields

| Key        | Type     | Required | Description                          |
|------------|----------|----------|--------------------------------------|
| `api_key`  | `string` | Yes      | OpenAI API key                       |
| `base_url` | `string` | No       | Override API base URL                |

## Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/voice/stt"
    _ "github.com/lookatitude/beluga-ai/voice/stt/providers/whisper"
)

func main() {
    ctx := context.Background()

    engine, err := stt.New("whisper", stt.Config{
        Model: "whisper-1",
        Extra: map[string]any{"api_key": os.Getenv("OPENAI_API_KEY")},
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
import "github.com/lookatitude/beluga-ai/voice/stt/providers/whisper"

engine, err := whisper.New(stt.Config{
    Model:    "whisper-1",
    Language: "en",
    Extra:    map[string]any{"api_key": os.Getenv("OPENAI_API_KEY")},
})
```

## Streaming

Whisper does not support native real-time streaming. The `TranscribeStream` method transcribes each audio chunk independently as a separate batch request. Each chunk produces a final transcript event:

```go
for event, err := range engine.TranscribeStream(ctx, audioStream) {
    if err != nil {
        log.Printf("error: %v", err)
        break
    }
    // All events from Whisper are final (no partial results)
    fmt.Printf("[FINAL] %s\n", event.Text)
}
```

For real-time transcription with interim results, consider [Deepgram](/providers/voice/deepgram) or [AssemblyAI](/providers/voice/assemblyai) which support native WebSocket streaming.

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
    stt.WithModel("whisper-1"),
)
```

### Custom Endpoint

Use an alternative OpenAI-compatible endpoint (e.g., Azure OpenAI):

```go
engine, err := stt.New("whisper", stt.Config{
    Model: "whisper-1",
    Extra: map[string]any{
        "api_key":  os.Getenv("OPENAI_API_KEY"),
        "base_url": "https://my-instance.openai.azure.com/openai/deployments/whisper-1",
    },
})
```
