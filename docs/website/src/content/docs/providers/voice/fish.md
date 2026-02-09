---
title: "Fish Audio"
description: "Fish Audio TTS provider for open-source voice synthesis."
---

Fish Audio provides text-to-speech synthesis with support for voice cloning via reference IDs. The Beluga AI provider uses the Fish Audio v1 API for synthesis, producing audio output in configurable formats.

## Installation

```go
import _ "github.com/lookatitude/beluga-ai/voice/tts/providers/fish"
```

The blank import registers the `"fish"` provider with the TTS registry.

## Configuration

| Field    | Type          | Default     | Description                              |
|----------|---------------|-------------|------------------------------------------|
| `Voice`  | `string`      | `"default"` | Reference ID for voice cloning           |
| `Format` | `AudioFormat` | —           | Output audio format                      |
| `Extra`  | —             | —           | See below                                |

### Extra Fields

| Key        | Type     | Required | Description                          |
|------------|----------|----------|--------------------------------------|
| `api_key`  | `string` | Yes      | Fish Audio API key                   |
| `base_url` | `string` | No       | Override base URL                    |

## Basic Usage

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/voice/tts"
    _ "github.com/lookatitude/beluga-ai/voice/tts/providers/fish"
)

func main() {
    ctx := context.Background()

    engine, err := tts.New("fish", tts.Config{
        Voice: "default",
        Extra: map[string]any{"api_key": os.Getenv("FISH_API_KEY")},
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
import "github.com/lookatitude/beluga-ai/voice/tts/providers/fish"

engine, err := fish.New(tts.Config{
    Voice: "custom-reference-id",
    Extra: map[string]any{"api_key": os.Getenv("FISH_API_KEY")},
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

### Voice Cloning

Fish Audio uses a reference ID system for voice cloning. Provide a `reference_id` as the `Voice` field to use a cloned voice:

```go
engine, err := tts.New("fish", tts.Config{
    Voice: "your-cloned-voice-reference-id",
    Extra: map[string]any{"api_key": os.Getenv("FISH_API_KEY")},
})
```

### Per-Request Options

```go
audio, err := engine.Synthesize(ctx, "Hello!",
    tts.WithVoice("different-reference-id"),
    tts.WithFormat(tts.FormatMP3),
)
```

### Custom Endpoint

```go
engine, err := tts.New("fish", tts.Config{
    Voice: "default",
    Extra: map[string]any{
        "api_key":  os.Getenv("FISH_API_KEY"),
        "base_url": "https://fish.internal.corp/v1",
    },
})
```
