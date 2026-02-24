---
title: "Deepgram Voice Provider"
description: "Deepgram STT for real-time and batch speech-to-text in Beluga AI. Nova-2 models with WebSocket streaming, diarization, and word timing in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Deepgram, speech-to-text, STT, Nova-2, WebSocket streaming, diarization, Go, Beluga AI"
---

Deepgram provides high-accuracy speech-to-text transcription with native WebSocket streaming support. The Beluga AI provider uses Deepgram's REST API for batch transcription and WebSocket API for real-time streaming, delivering word-level timing, speaker diarization, and automatic punctuation.

Choose Deepgram when you need real-time streaming STT with low latency and high accuracy. Its native WebSocket support makes it a strong default for production voice pipelines where interim results matter. For batch-only workloads where cost is the priority, consider [Groq Whisper](/docs/providers/voice/groq-whisper) instead.

## Installation

```go
import _ "github.com/lookatitude/beluga-ai/voice/stt/providers/deepgram"
```

The blank import registers the `"deepgram"` provider with the STT registry.

## Configuration

| Field        | Type     | Default       | Description                                |
|--------------|----------|---------------|--------------------------------------------|
| `Language`   | `string` | —             | BCP-47 language code (e.g., `"en"`, `"es"`)|
| `Model`      | `string` | `"nova-2"`    | Deepgram model (nova-2, nova, enhanced)    |
| `Punctuation`| `bool`   | `false`       | Enable automatic punctuation               |
| `Diarization`| `bool`   | `false`       | Enable speaker identification              |
| `SampleRate` | `int`    | —             | Audio sample rate in Hz                    |
| `Encoding`   | `string` | —             | Audio encoding (`"linear16"`, `"opus"`)    |
| `Extra`      | —        | —             | See below                                  |

### Extra Fields

| Key        | Type     | Required | Description                          |
|------------|----------|----------|--------------------------------------|
| `api_key`  | `string` | Yes      | Deepgram API key                     |
| `base_url` | `string` | No       | Override REST base URL               |
| `ws_url`   | `string` | No       | Override WebSocket base URL          |

## Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/voice/stt"
    _ "github.com/lookatitude/beluga-ai/voice/stt/providers/deepgram"
)

func main() {
    ctx := context.Background()

    engine, err := stt.New("deepgram", stt.Config{
        Language: "en",
        Model:    "nova-2",
        Extra:    map[string]any{"api_key": os.Getenv("DEEPGRAM_API_KEY")},
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

For compile-time type safety, use the provider package directly:

```go
import "github.com/lookatitude/beluga-ai/voice/stt/providers/deepgram"

engine, err := deepgram.New(stt.Config{
    Language:    "en",
    Model:       "nova-2",
    Punctuation: true,
    Diarization: true,
    Extra:       map[string]any{"api_key": os.Getenv("DEEPGRAM_API_KEY")},
})
```

## Streaming

Deepgram supports native real-time streaming via WebSocket. Audio chunks are sent over the socket and transcript events are emitted as they become available, with both interim (partial) and final results.

```go
func transcribeStream(ctx context.Context, engine stt.STT, audioStream iter.Seq2[[]byte, error]) {
    for event, err := range engine.TranscribeStream(ctx, audioStream) {
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
}
```

Transcript events include word-level timing when available:

```go
for event, err := range engine.TranscribeStream(ctx, audioStream) {
    if err != nil {
        log.Printf("error: %v", err)
        break
    }
    for _, word := range event.Words {
        fmt.Printf("  %s [%.2fs - %.2fs] (%.2f)\n",
            word.Text, word.Start.Seconds(), word.End.Seconds(), word.Confidence)
    }
}
```

## FrameProcessor Integration

Wrap the engine as a `FrameProcessor` for use in a voice pipeline:

```go
import "github.com/lookatitude/beluga-ai/voice/stt"

processor := stt.AsFrameProcessor(engine, stt.WithLanguage("en"))

// Use in a pipeline
pipeline := voice.Chain(vadProcessor, processor, llmProcessor, ttsProcessor)
```

## Advanced Features

### Per-Request Options

Override configuration on individual calls:

```go
text, err := engine.Transcribe(ctx, audio,
    stt.WithLanguage("es"),
    stt.WithModel("nova-2"),
    stt.WithPunctuation(true),
    stt.WithDiarization(true),
    stt.WithEncoding("linear16"),
    stt.WithSampleRate(16000),
)
```

### Custom Endpoint

For self-hosted or on-premise Deepgram deployments:

```go
engine, err := stt.New("deepgram", stt.Config{
    Model: "nova-2",
    Extra: map[string]any{
        "api_key":  os.Getenv("DEEPGRAM_API_KEY"),
        "base_url": "https://deepgram.internal.corp/v1",
        "ws_url":   "wss://deepgram.internal.corp/v1",
    },
})
```
