---
title: "Groq"
description: "Groq STT and TTS provider for ultra-fast voice processing on LPU hardware."
---

Groq provides both speech-to-text (via Whisper models) and text-to-speech through its OpenAI-compatible API, running on specialized LPU hardware for ultra-fast inference. Beluga AI registers `"groq"` in both the STT and TTS registries.

## Installation

```go
// STT (Whisper)
import _ "github.com/lookatitude/beluga-ai/voice/stt/providers/groq"

// TTS
import _ "github.com/lookatitude/beluga-ai/voice/tts/providers/groq"
```

## STT (Whisper on Groq)

### Configuration

| Field      | Type     | Default              | Description                            |
|------------|----------|----------------------|----------------------------------------|
| `Language` | `string` | —                    | ISO-639 language code                  |
| `Model`    | `string` | `"whisper-large-v3"` | Whisper model identifier               |
| `Extra`    | —        | —                    | See below                              |

#### Extra Fields

| Key        | Type     | Required | Description                          |
|------------|----------|----------|--------------------------------------|
| `api_key`  | `string` | Yes      | Groq API key (`gsk-...`)            |
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
    _ "github.com/lookatitude/beluga-ai/voice/stt/providers/groq"
)

func main() {
    ctx := context.Background()

    engine, err := stt.New("groq", stt.Config{
        Model: "whisper-large-v3",
        Extra: map[string]any{"api_key": os.Getenv("GROQ_API_KEY")},
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

Groq Whisper does not support native streaming. The `TranscribeStream` method buffers all audio chunks and performs a single batch transcription when the stream ends:

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

| Field    | Type          | Default               | Description                              |
|----------|---------------|-----------------------|------------------------------------------|
| `Voice`  | `string`      | `"aura-asteria-en"`   | Voice identifier                         |
| `Model`  | `string`      | `"playai-tts"`        | TTS model identifier                     |
| `Format` | `AudioFormat` | —                     | Output format (`mp3`, `wav`, `pcm`)      |
| `Speed`  | `float64`     | —                     | Speech rate multiplier (1.0 = normal)    |
| `Extra`  | —             | —                     | See below                                |

#### Extra Fields

| Key        | Type     | Required | Description                          |
|------------|----------|----------|--------------------------------------|
| `api_key`  | `string` | Yes      | Groq API key (`gsk-...`)            |
| `base_url` | `string` | No       | Override base URL                    |

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/voice/tts"
    _ "github.com/lookatitude/beluga-ai/voice/tts/providers/groq"
)

func main() {
    ctx := context.Background()

    engine, err := tts.New("groq", tts.Config{
        Voice: "aura-asteria-en",
        Extra: map[string]any{"api_key": os.Getenv("GROQ_API_KEY")},
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
import "github.com/lookatitude/beluga-ai/voice/tts/providers/groq"

engine, err := groq.New(tts.Config{
    Voice: "aura-asteria-en",
    Model: "playai-tts",
    Extra: map[string]any{"api_key": os.Getenv("GROQ_API_KEY")},
})
```

### Streaming

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
// STT
sttProcessor := stt.AsFrameProcessor(sttEngine, stt.WithLanguage("en"))

// TTS
ttsProcessor := tts.AsFrameProcessor(ttsEngine, 24000, tts.WithVoice("aura-asteria-en"))

pipeline := voice.Chain(sttProcessor, llmProcessor, ttsProcessor)
```

## Advanced Features

### OpenAI-Compatible API

Both the STT and TTS providers use Groq's OpenAI-compatible API endpoints (`/audio/transcriptions` for STT, `/audio/speech` for TTS), making it straightforward to swap with other OpenAI-compatible endpoints.

### Per-Request Options

```go
// STT
text, err := sttEngine.Transcribe(ctx, audio,
    stt.WithLanguage("fr"),
    stt.WithModel("whisper-large-v3-turbo"),
)

// TTS
audio, err := ttsEngine.Synthesize(ctx, "Hello!",
    tts.WithVoice("aura-luna-en"),
    tts.WithFormat(tts.FormatWAV),
    tts.WithSpeed(1.2),
)
```
