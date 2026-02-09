---
title: "Smallest AI"
description: "Smallest AI TTS provider for lightning-fast voice synthesis."
---

Smallest AI provides lightning-fast text-to-speech synthesis with low-latency models optimized for real-time applications. The Beluga AI provider uses the Smallest AI v1 API for synthesis, supporting configurable voice, model, and speed settings.

## Installation

```go
import _ "github.com/lookatitude/beluga-ai/voice/tts/providers/smallest"
```

The blank import registers the `"smallest"` provider with the TTS registry.

## Configuration

| Field    | Type          | Default       | Description                              |
|----------|---------------|---------------|------------------------------------------|
| `Voice`  | `string`      | `"emily"`     | Voice identifier                         |
| `Model`  | `string`      | `"lightning"` | Model identifier (lightning)             |
| `Speed`  | `float64`     | —             | Speech rate multiplier (1.0 = normal)    |
| `Extra`  | —             | —             | See below                                |

### Extra Fields

| Key        | Type     | Required | Description                          |
|------------|----------|----------|--------------------------------------|
| `api_key`  | `string` | Yes      | Smallest AI API key                  |
| `base_url` | `string` | No       | Override base URL                    |

## Basic Usage

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/voice/tts"
    _ "github.com/lookatitude/beluga-ai/voice/tts/providers/smallest"
)

func main() {
    ctx := context.Background()

    engine, err := tts.New("smallest", tts.Config{
        Voice: "emily",
        Extra: map[string]any{"api_key": os.Getenv("SMALLEST_API_KEY")},
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
import "github.com/lookatitude/beluga-ai/voice/tts/providers/smallest"

engine, err := smallest.New(tts.Config{
    Voice: "emily",
    Model: "lightning",
    Extra: map[string]any{"api_key": os.Getenv("SMALLEST_API_KEY")},
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
processor := tts.AsFrameProcessor(engine, 24000, tts.WithVoice("emily"))
pipeline := voice.Chain(sttProcessor, llmProcessor, processor)
```

## Advanced Features

### Per-Request Options

```go
audio, err := engine.Synthesize(ctx, "Hello!",
    tts.WithVoice("different-voice"),
    tts.WithModel("lightning"),
    tts.WithSpeed(1.3),
)
```

### Custom Endpoint

```go
engine, err := tts.New("smallest", tts.Config{
    Voice: "emily",
    Extra: map[string]any{
        "api_key":  os.Getenv("SMALLEST_API_KEY"),
        "base_url": "https://smallest.internal.corp/v1",
    },
})
```
