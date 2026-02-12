---
title: "ElevenLabs Voice Provider"
description: "ElevenLabs STT and TTS for voice cloning and synthesis in Beluga AI. Premium voices, Scribe transcription, and WebSocket streaming in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "ElevenLabs, text-to-speech, voice cloning, TTS, STT, Scribe, premium voice, Go, Beluga AI"
---

ElevenLabs provides both speech-to-text (via the Scribe engine) and text-to-speech (with voice cloning and premium synthesis). Beluga AI registers two separate providers: `"elevenlabs"` in the STT registry and `"elevenlabs"` in the TTS registry.

Choose ElevenLabs when voice quality is the top priority — particularly for applications requiring natural-sounding speech with voice cloning, custom voices, or multilingual synthesis. The TTS output is widely regarded as among the most expressive available. For lower-latency TTS in real-time pipelines, also evaluate [Cartesia](/providers/voice/cartesia) or [LMNT](/providers/voice/lmnt).

## Installation

```go
// STT (Scribe)
import _ "github.com/lookatitude/beluga-ai/voice/stt/providers/elevenlabs"

// TTS
import _ "github.com/lookatitude/beluga-ai/voice/tts/providers/elevenlabs"
```

## STT (Scribe)

### Configuration

| Field      | Type     | Default        | Description                            |
|------------|----------|----------------|----------------------------------------|
| `Language` | `string` | —              | Language code for transcription        |
| `Model`    | `string` | `"scribe_v1"`  | Scribe model identifier                |
| `Extra`    | —        | —              | See below                              |

#### Extra Fields

| Key        | Type     | Required | Description                          |
|------------|----------|----------|--------------------------------------|
| `api_key`  | `string` | Yes      | ElevenLabs API key (`xi-...`)        |
| `base_url` | `string` | No       | Override base URL                    |

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/voice/stt"
    _ "github.com/lookatitude/beluga-ai/voice/stt/providers/elevenlabs"
)

func main() {
    ctx := context.Background()

    engine, err := stt.New("elevenlabs", stt.Config{
        Extra: map[string]any{"api_key": os.Getenv("ELEVENLABS_API_KEY")},
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

### Streaming

ElevenLabs Scribe does not support native streaming. The `TranscribeStream` method transcribes each audio chunk independently as a batch request:

```go
for event, err := range engine.TranscribeStream(ctx, audioStream) {
    if err != nil {
        log.Printf("error: %v", err)
        break
    }
    fmt.Printf("[FINAL] %s\n", event.Text)
}
```

For real-time streaming with interim results, consider [Deepgram](/providers/voice/deepgram) or [AssemblyAI](/providers/voice/assemblyai).

---

## TTS

### Configuration

| Field       | Type          | Default                     | Description                          |
|-------------|---------------|-----------------------------|--------------------------------------|
| `Voice`     | `string`      | `"21m00Tcm4TlvDq8ikWAM"`   | Voice ID (Rachel by default)         |
| `Model`     | `string`      | `"eleven_monolingual_v1"`   | TTS model identifier                 |
| `Format`    | `AudioFormat` | —                           | Output audio format                  |
| `Extra`     | —             | —                           | See below                            |

#### Extra Fields

| Key        | Type     | Required | Description                          |
|------------|----------|----------|--------------------------------------|
| `api_key`  | `string` | Yes      | ElevenLabs API key (`xi-...`)        |
| `base_url` | `string` | No       | Override base URL                    |

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/voice/tts"
    _ "github.com/lookatitude/beluga-ai/voice/tts/providers/elevenlabs"
)

func main() {
    ctx := context.Background()

    engine, err := tts.New("elevenlabs", tts.Config{
        Voice: "rachel",
        Extra: map[string]any{"api_key": os.Getenv("ELEVENLABS_API_KEY")},
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

### Direct Construction

```go
import "github.com/lookatitude/beluga-ai/voice/tts/providers/elevenlabs"

engine, err := elevenlabs.New(tts.Config{
    Voice: "rachel",
    Model: "eleven_multilingual_v2",
    Extra: map[string]any{"api_key": os.Getenv("ELEVENLABS_API_KEY")},
})
```

### Streaming

The TTS streaming interface synthesizes each text chunk from the input stream independently:

```go
for chunk, err := range engine.SynthesizeStream(ctx, textStream) {
    if err != nil {
        log.Printf("error: %v", err)
        break
    }
    transport.Send(chunk)
}
```

### FrameProcessor Integration

```go
processor := tts.AsFrameProcessor(engine, 24000, tts.WithVoice("rachel"))
pipeline := voice.Chain(sttProcessor, llmProcessor, processor)
```

## Advanced Features

### Voice Settings

The provider sends stability and similarity boost parameters with each synthesis request. The default values (stability: 0.5, similarity_boost: 0.75) provide a balanced output. For production use, you can adjust these through the ElevenLabs dashboard per voice.

### Per-Request Options

```go
// STT
text, err := sttEngine.Transcribe(ctx, audio,
    stt.WithLanguage("fr"),
    stt.WithModel("scribe_v1"),
)

// TTS
audio, err := ttsEngine.Synthesize(ctx, "Bonjour!",
    tts.WithVoice("custom-voice-id"),
    tts.WithModel("eleven_multilingual_v2"),
)
```
