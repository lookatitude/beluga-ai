---
title: "AssemblyAI Voice Provider"
description: "AssemblyAI STT for real-time and async transcription in Beluga AI. Speaker diarization, sentiment analysis, and entity detection in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "AssemblyAI, speech-to-text, STT, transcription, speaker diarization, real-time, Go, Beluga AI"
---

AssemblyAI provides speech-to-text transcription with both asynchronous batch processing and real-time WebSocket streaming. The Beluga AI provider uses AssemblyAI's upload-and-poll workflow for batch transcription and the real-time WebSocket API for streaming, delivering word-level timing, speaker labels, and automatic punctuation.

Choose AssemblyAI when you need both real-time streaming and high-quality async batch transcription from a single provider. Its upload-and-poll batch workflow is well-suited for pre-recorded audio processing, while WebSocket streaming handles live audio. For the lowest streaming latency, also evaluate [Deepgram](/providers/voice/deepgram).

## Installation

```go
import _ "github.com/lookatitude/beluga-ai/voice/stt/providers/assemblyai"
```

The blank import registers the `"assemblyai"` provider with the STT registry.

## Configuration

| Field        | Type     | Default  | Description                                 |
|--------------|----------|----------|---------------------------------------------|
| `Language`   | `string` | —        | BCP-47 language code (e.g., `"en"`)         |
| `Punctuation`| `bool`   | `false`  | Enable automatic punctuation                |
| `Diarization`| `bool`   | `false`  | Enable speaker labels                       |
| `SampleRate` | `int`    | `16000`  | Audio sample rate in Hz (for streaming)     |
| `Extra`      | —        | —        | See below                                   |

### Extra Fields

| Key        | Type     | Required | Description                          |
|------------|----------|----------|--------------------------------------|
| `api_key`  | `string` | Yes      | AssemblyAI API key                   |
| `base_url` | `string` | No       | Override REST base URL               |
| `ws_url`   | `string` | No       | Override WebSocket URL               |

## Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/voice/stt"
    _ "github.com/lookatitude/beluga-ai/voice/stt/providers/assemblyai"
)

func main() {
    ctx := context.Background()

    engine, err := stt.New("assemblyai", stt.Config{
        Language: "en",
        Extra:    map[string]any{"api_key": os.Getenv("ASSEMBLYAI_API_KEY")},
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
import "github.com/lookatitude/beluga-ai/voice/stt/providers/assemblyai"

engine, err := assemblyai.New(stt.Config{
    Language:    "en",
    Punctuation: true,
    Diarization: true,
    Extra:       map[string]any{"api_key": os.Getenv("ASSEMBLYAI_API_KEY")},
})
```

## Batch Transcription

The batch workflow uploads audio to AssemblyAI, creates a transcript job, and polls for completion. This is handled automatically by `Transcribe`:

1. Audio bytes are uploaded to AssemblyAI's upload endpoint.
2. A transcript job is created with the uploaded audio URL.
3. The provider polls the transcript status every 500ms until completion.

This makes batch transcription best suited for pre-recorded audio rather than real-time use cases.

## Streaming

AssemblyAI supports native real-time streaming via WebSocket. The provider emits both partial (`PartialTranscript`) and final (`FinalTranscript`) events:

```go
func transcribeStream(ctx context.Context, engine stt.STT, audioStream iter.Seq2[[]byte, error]) {
    for event, err := range engine.TranscribeStream(ctx, audioStream,
        stt.WithSampleRate(16000),
    ) {
        if err != nil {
            log.Printf("stream error: %v", err)
            break
        }
        if event.IsFinal {
            fmt.Printf("[FINAL] %s\n", event.Text)
        } else {
            fmt.Printf("[PARTIAL] %s\n", event.Text)
        }
    }
}
```

Word-level timing is available on transcript events:

```go
for _, word := range event.Words {
    fmt.Printf("  %s [%v - %v] (%.2f)\n",
        word.Text, word.Start, word.End, word.Confidence)
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
    stt.WithLanguage("es"),
    stt.WithPunctuation(true),
    stt.WithDiarization(true),
    stt.WithSampleRate(16000),
)
```

### Custom Endpoint

```go
engine, err := stt.New("assemblyai", stt.Config{
    Language: "en",
    Extra: map[string]any{
        "api_key":  os.Getenv("ASSEMBLYAI_API_KEY"),
        "base_url": "https://assemblyai.internal.corp/v2",
        "ws_url":   "wss://assemblyai.internal.corp/v2/realtime/ws",
    },
})
```
