---
title: "PlayHT Voice Provider"
description: "PlayHT TTS for AI voice generation in Beluga AI. Premium voice cloning, conversational voices, and streaming text-to-speech in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "PlayHT, text-to-speech, TTS, voice cloning, AI voice generation, streaming, Go, Beluga AI"
---

PlayHT provides AI-powered text-to-speech with voice cloning and multiple output formats. The Beluga AI provider uses the PlayHT v2 API for synthesis, supporting configurable voice selection, output format, and speech speed.

Choose PlayHT when you need voice cloning with flexible output formats (MP3, WAV, PCM, Opus) and fine-grained speed control. PlayHT's zero-shot voice cloning lets you create custom voices from short audio samples. For the lowest synthesis latency, consider [Cartesia](/providers/voice/cartesia) or [LMNT](/providers/voice/lmnt).

## Installation

```go
import _ "github.com/lookatitude/beluga-ai/voice/tts/providers/playht"
```

The blank import registers the `"playht"` provider with the TTS registry.

## Configuration

| Field    | Type          | Default | Description                                     |
|----------|---------------|---------|-------------------------------------------------|
| `Voice`  | `string`      | —       | Voice URL (e.g., `s3://voice-cloning-zero-shot/...`) |
| `Format` | `AudioFormat` | `"mp3"` | Output format (`mp3`, `wav`, `pcm`, `opus`)     |
| `Speed`  | `float64`     | —       | Speech rate multiplier (1.0 = normal)           |
| `Extra`  | —             | —       | See below                                       |

### Extra Fields

| Key        | Type     | Required | Description                          |
|------------|----------|----------|--------------------------------------|
| `api_key`  | `string` | Yes      | PlayHT API key                       |
| `user_id`  | `string` | Yes      | PlayHT user ID                       |
| `base_url` | `string` | No       | Override base URL                    |

## Basic Usage

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/voice/tts"
    _ "github.com/lookatitude/beluga-ai/voice/tts/providers/playht"
)

func main() {
    ctx := context.Background()

    engine, err := tts.New("playht", tts.Config{
        Voice: "s3://voice-cloning-zero-shot/775ae416-49bb-4fb6-bd45-740f205d3571/jennifersaad/manifest.json",
        Extra: map[string]any{
            "api_key": os.Getenv("PLAYHT_API_KEY"),
            "user_id": os.Getenv("PLAYHT_USER_ID"),
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    audio, err := engine.Synthesize(ctx, "Hello, welcome to Beluga AI.")
    if err != nil {
        log.Fatal(err)
    }

    if err := os.WriteFile("output.mp3", audio, 0644); err != nil {
        log.Fatal(err)
    }
}
```

## Direct Construction

```go
import "github.com/lookatitude/beluga-ai/voice/tts/providers/playht"

engine, err := playht.New(tts.Config{
    Voice: "s3://voice-cloning-zero-shot/775ae416-49bb-4fb6-bd45-740f205d3571/jennifersaad/manifest.json",
    Extra: map[string]any{
        "api_key": os.Getenv("PLAYHT_API_KEY"),
        "user_id": os.Getenv("PLAYHT_USER_ID"),
    },
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
processor := tts.AsFrameProcessor(engine, 24000)
pipeline := voice.Chain(sttProcessor, llmProcessor, processor)
```

## Advanced Features

### Per-Request Options

```go
audio, err := engine.Synthesize(ctx, "Hello!",
    tts.WithVoice("different-voice-url"),
    tts.WithFormat(tts.FormatWAV),
    tts.WithSpeed(1.2),
)
```

### Authentication

PlayHT requires both an API key and a user ID. These are sent as `Authorization: Bearer <api_key>` and `X-USER-ID: <user_id>` headers respectively.

### Custom Endpoint

```go
engine, err := tts.New("playht", tts.Config{
    Extra: map[string]any{
        "api_key":  os.Getenv("PLAYHT_API_KEY"),
        "user_id":  os.Getenv("PLAYHT_USER_ID"),
        "base_url": "https://playht.internal.corp/api/v2",
    },
})
```
