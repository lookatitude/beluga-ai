---
title: "LMNT"
description: "LMNT TTS provider for ultra-low-latency voice synthesis."
---

LMNT provides ultra-low-latency text-to-speech synthesis optimized for real-time voice applications. The Beluga AI provider uses the LMNT Speech API for synthesis, supporting configurable voice selection, output format, and speech speed.

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
