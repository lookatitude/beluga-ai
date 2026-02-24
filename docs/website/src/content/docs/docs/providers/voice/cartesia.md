---
title: "Cartesia Voice Provider"
description: "Cartesia TTS for low-latency voice synthesis in Beluga AI. Sonic models with WebSocket streaming, voice cloning, and emotion control in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Cartesia, text-to-speech, TTS, Sonic, low latency, voice synthesis, WebSocket, Go, Beluga AI"
---

Cartesia provides ultra-low-latency text-to-speech synthesis through the Sonic voice engine. The Beluga AI provider uses Cartesia's HTTP API with the `httpclient` infrastructure for built-in retry support, producing raw PCM audio output suitable for real-time voice pipelines.

Choose Cartesia when latency is critical — for example, in conversational voice agents where every millisecond of TTS delay affects the user experience. Cartesia's Sonic engine is optimized for speed-first synthesis with direct PCM output, avoiding the overhead of compressed audio decoding. For the highest voice quality with more voice variety, consider [ElevenLabs](/providers/voice/elevenlabs).

## Installation

```go
import _ "github.com/lookatitude/beluga-ai/voice/tts/providers/cartesia"
```

The blank import registers the `"cartesia"` provider with the TTS registry.

## Configuration

| Field       | Type          | Default      | Description                              |
|-------------|---------------|--------------|------------------------------------------|
| `Voice`     | `string`      | —            | Cartesia voice UUID                      |
| `Model`     | `string`      | `"sonic-2"`  | Cartesia model (sonic-2, sonic-english)  |
| `SampleRate`| `int`         | `24000`      | Output sample rate in Hz                 |
| `Extra`     | —             | —            | See below                                |

### Extra Fields

| Key        | Type     | Required | Description                          |
|------------|----------|----------|--------------------------------------|
| `api_key`  | `string` | Yes      | Cartesia API key                     |
| `base_url` | `string` | No       | Override base URL                    |

## Basic Usage

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/voice/tts"
    _ "github.com/lookatitude/beluga-ai/voice/tts/providers/cartesia"
)

func main() {
    ctx := context.Background()

    engine, err := tts.New("cartesia", tts.Config{
        Voice: "a0e99841-438c-4a64-b679-ae501e7d6091",
        Extra: map[string]any{"api_key": os.Getenv("CARTESIA_API_KEY")},
    })
    if err != nil {
        log.Fatal(err)
    }

    audio, err := engine.Synthesize(ctx, "Hello, welcome to Beluga AI.")
    if err != nil {
        log.Fatal(err)
    }

    if err := os.WriteFile("output.pcm", audio, 0644); err != nil {
        log.Fatal(err)
    }
}
```

## Direct Construction

```go
import "github.com/lookatitude/beluga-ai/voice/tts/providers/cartesia"

engine, err := cartesia.New(tts.Config{
    Voice:      "a0e99841-438c-4a64-b679-ae501e7d6091",
    Model:      "sonic-2",
    SampleRate: 24000,
    Extra:      map[string]any{"api_key": os.Getenv("CARTESIA_API_KEY")},
})
```

## Streaming

The streaming interface synthesizes each text chunk from the input stream independently:

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
processor := tts.AsFrameProcessor(engine, 24000, tts.WithVoice("a0e99841-438c-4a64-b679-ae501e7d6091"))
pipeline := voice.Chain(sttProcessor, llmProcessor, processor)
```

## Advanced Features

### Output Format

Cartesia outputs raw PCM audio (16-bit little-endian, `pcm_s16le`) by default. The output format is configured in the request body and matches the sample rate specified in the config:

```go
engine, err := tts.New("cartesia", tts.Config{
    Voice:      "a0e99841-438c-4a64-b679-ae501e7d6091",
    SampleRate: 44100, // Override default 24000 Hz
    Extra:      map[string]any{"api_key": os.Getenv("CARTESIA_API_KEY")},
})
```

### Built-in Retry

The Cartesia provider uses Beluga's `httpclient` infrastructure, which provides automatic retry with exponential backoff (up to 2 retries by default) for transient failures.

### Per-Request Options

```go
audio, err := engine.Synthesize(ctx, "Hello!",
    tts.WithVoice("different-voice-uuid"),
    tts.WithSampleRate(16000),
)
```

### Custom Endpoint

```go
engine, err := tts.New("cartesia", tts.Config{
    Voice: "a0e99841-438c-4a64-b679-ae501e7d6091",
    Extra: map[string]any{
        "api_key":  os.Getenv("CARTESIA_API_KEY"),
        "base_url": "https://cartesia.internal.corp",
    },
})
```
