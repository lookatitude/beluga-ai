---
title: "LMNT Voice Provider"
description: "LMNT TTS for ultra-low-latency voice synthesis in Beluga AI. Real-time text-to-speech with natural voices and WebSocket streaming in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "LMNT, text-to-speech, TTS, ultra-low-latency, voice synthesis, WebSocket, real-time, Go, Beluga AI"
---

LMNT provides ultra-low-latency text-to-speech synthesis optimized for real-time voice applications. The Beluga AI provider uses the LMNT Speech API for synthesis, supporting configurable voice selection, output format, and speech speed.

Choose LMNT when you need the lowest possible TTS latency for real-time conversational agents. LMNT is purpose-built for speed, making it a strong choice for voice pipelines where response time is the primary concern. For broader voice variety or voice cloning, consider [ElevenLabs](/providers/voice/elevenlabs) or [PlayHT](/providers/voice/playht).

## Installation

```go
import _ "github.com/lookatitude/beluga-ai/voice/tts/providers/lmnt"
```

The blank import registers the `"lmnt"` provider with the TTS registry.

## Configuration

| Field    | Type          | Default    | Description                                |
|----------|---------------|------------|--------------------------------------------|
| `Voice`  | `string`      | `"lily"`   | LMNT voice identifier                     |
| `Format` | `AudioFormat` | —          | Output audio format                        |
| `Speed`  | `float64`     | —          | Speech rate multiplier (1.0 = normal)      |
| `Extra`  | —             | —          | See below                                  |

### Extra Fields

| Key        | Type     | Required | Description                          |
|------------|----------|----------|--------------------------------------|
| `api_key`  | `string` | Yes      | LMNT API key                         |
| `base_url` | `string` | No       | Override base URL                    |

## Basic Usage

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/voice/tts"
    _ "github.com/lookatitude/beluga-ai/voice/tts/providers/lmnt"
)

func main() {
    ctx := context.Background()

    engine, err := tts.New("lmnt", tts.Config{
        Voice: "lily",
        Extra: map[string]any{"api_key": os.Getenv("LMNT_API_KEY")},
    })
    if err != nil {
        log.Fatal(err)
    }

    audio, err := engine.Synthesize(ctx, "Hello, welcome to Beluga AI.")
    if err != nil {
        log.Fatal(err)
    }

    if err := os.WriteFile("output.wav", audio, 0644); err != nil {
        log.Fatal(err)
    }
}
```

## Direct Construction

```go
import "github.com/lookatitude/beluga-ai/voice/tts/providers/lmnt"

engine, err := lmnt.New(tts.Config{
    Voice: "lily",
    Extra: map[string]any{"api_key": os.Getenv("LMNT_API_KEY")},
})
```

## Streaming

The streaming interface synthesizes each text chunk independently:

```go
for chunk, err := range engine.SynthesizeStream(ctx, textStream) {
    if err != nil {
        log.Printf("error: %v", err)
        break
    }
    transport.Send(chunk)
}
```

## FrameProcessor Integration

```go
processor := tts.AsFrameProcessor(engine, 24000, tts.WithVoice("lily"))
pipeline := voice.Chain(sttProcessor, llmProcessor, processor)
```

## Advanced Features

### Per-Request Options

```go
audio, err := engine.Synthesize(ctx, "Hello!",
    tts.WithVoice("daniel"),
    tts.WithSpeed(1.5),
)
```

### Custom Endpoint

```go
engine, err := tts.New("lmnt", tts.Config{
    Voice: "lily",
    Extra: map[string]any{
        "api_key":  os.Getenv("LMNT_API_KEY"),
        "base_url": "https://lmnt.internal.corp/v1",
    },
})
```
