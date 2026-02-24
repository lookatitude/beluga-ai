---
title: "STT API — Speech-to-Text Providers"
description: "Voice STT API reference for Beluga AI. Speech-to-text interface with Deepgram, AssemblyAI, Whisper, Groq, ElevenLabs, and Gladia providers."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "STT API, speech-to-text, Deepgram, AssemblyAI, Whisper, Groq, transcription, voice, Beluga AI, Go, reference"
---

## stt

```go
import "github.com/lookatitude/beluga-ai/voice/stt"
```

Package stt provides the speech-to-text (STT) interface and provider registry
for the Beluga AI voice pipeline. Providers implement the `STT` interface and
register themselves via init() for discovery.

## Core Interface

The `STT` interface supports both batch and streaming transcription:

```go
type STT interface {
    Transcribe(ctx context.Context, audio []byte, opts ...Option) (string, error)
    TranscribeStream(ctx context.Context, audioStream iter.Seq2[[]byte, error], opts ...Option) iter.Seq2[TranscriptEvent, error]
}
```

## Transcript Events

Streaming transcription produces `TranscriptEvent` values containing the
transcribed text, finality flag, confidence score, timestamp, detected
language, and optional word-level timing via `Word`.

## Registry Pattern

Providers register via `Register` in their init() function and are created
with `New`. Use `List` to discover available providers.

```go
import _ "github.com/lookatitude/beluga-ai/voice/stt/providers/deepgram"

engine, err := stt.New("deepgram", stt.Config{Language: "en"})
text, err := engine.Transcribe(ctx, audioBytes)

// Streaming:
for event, err := range engine.TranscribeStream(ctx, audioStream) {
    if err != nil { break }
    fmt.Printf("[%v] %s (final=%v)\n", event.Timestamp, event.Text, event.IsFinal)
}
```

## Frame Processor Integration

Use `AsFrameProcessor` to wrap an STT engine as a voice.FrameProcessor
for integration with the cascading pipeline.

```go
processor := stt.AsFrameProcessor(engine)
```

## Configuration

The `Config` struct supports language, model, punctuation, diarization,
sample rate, encoding, and provider-specific extras. Use functional options
like `WithLanguage`, `WithModel`, and `WithPunctuation` to configure
individual operations.

## Hooks

The `Hooks` struct provides callbacks: OnTranscript (each event),
OnUtterance (finalized text), and OnError. Use `ComposeHooks` to merge.

## Available Providers

- deepgram — Deepgram Nova-2 (voice/stt/providers/deepgram)
- assemblyai — AssemblyAI (voice/stt/providers/assemblyai)
- whisper — OpenAI Whisper (voice/stt/providers/whisper)
- groq — Groq Whisper (voice/stt/providers/groq)
- elevenlabs — ElevenLabs Scribe (voice/stt/providers/elevenlabs)
- gladia — Gladia (voice/stt/providers/gladia)

---

## assemblyai

```go
import "github.com/lookatitude/beluga-ai/voice/stt/providers/assemblyai"
```

Package assemblyai provides the AssemblyAI STT provider for the Beluga AI
voice pipeline. It uses the AssemblyAI Transcription API for batch
transcription and WebSocket API for real-time streaming.

## Registration

This package registers itself as "assemblyai" with the stt registry. Import
it with a blank identifier to enable:

```go
import _ "github.com/lookatitude/beluga-ai/voice/stt/providers/assemblyai"
```

## Usage

```go
engine, err := stt.New("assemblyai", stt.Config{
    Language: "en",
    Extra: map[string]any{"api_key": "..."},
})
text, err := engine.Transcribe(ctx, audioBytes)
```

Batch transcription uploads audio, creates a transcript, and polls for
completion. Streaming uses the real-time WebSocket endpoint for low-latency
partial and final transcripts with word-level timing.

## Configuration

Required configuration in Config.Extra:

- api_key — AssemblyAI API key (required)
- base_url — Custom REST API base URL (optional)
- ws_url — Custom WebSocket URL (optional)

## Exported Types

- [Engine] — implements stt.STT using AssemblyAI
- [New] — constructor accepting stt.Config

---

## deepgram

```go
import "github.com/lookatitude/beluga-ai/voice/stt/providers/deepgram"
```

Package deepgram provides the Deepgram STT provider for the Beluga AI
voice pipeline. It uses the Deepgram HTTP API for batch transcription and
WebSocket API for real-time streaming.

## Registration

This package registers itself as "deepgram" with the stt registry. Import
it with a blank identifier to enable:

```go
import _ "github.com/lookatitude/beluga-ai/voice/stt/providers/deepgram"
```

## Usage

```go
engine, err := stt.New("deepgram", stt.Config{Language: "en", Model: "nova-2"})
text, err := engine.Transcribe(ctx, audioBytes)
```

Streaming transcription uses the Deepgram WebSocket API for low-latency
partial and final transcripts with word-level timing.

## Configuration

Required configuration in Config.Extra:

- api_key — Deepgram API key (required)
- base_url — Custom REST API base URL (optional)
- ws_url — Custom WebSocket URL (optional)

The default model is "nova-2". Language, punctuation, diarization, encoding,
and sample rate are all supported through [stt.Config].

## Exported Types

- [Engine] — implements stt.STT using Deepgram
- [New] — constructor accepting stt.Config

---

## elevenlabs

```go
import "github.com/lookatitude/beluga-ai/voice/stt/providers/elevenlabs"
```

Package elevenlabs provides the ElevenLabs Scribe STT provider for the
Beluga AI voice pipeline. It uses the ElevenLabs Speech-to-Text API for
batch transcription via multipart form upload.

## Registration

This package registers itself as "elevenlabs" with the stt registry. Import
it with a blank identifier to enable:

```go
import _ "github.com/lookatitude/beluga-ai/voice/stt/providers/elevenlabs"
```

## Usage

```go
engine, err := stt.New("elevenlabs", stt.Config{
    Extra: map[string]any{"api_key": "xi-..."},
})
text, err := engine.Transcribe(ctx, audioBytes)
```

ElevenLabs Scribe does not support native streaming. TranscribeStream falls
back to transcribing each audio chunk independently.

## Configuration

Required configuration in Config.Extra:

- api_key — ElevenLabs API key (required)
- base_url — Custom API base URL (optional)

The default model is "scribe_v1".

## Exported Types

- [Engine] — implements stt.STT using ElevenLabs Scribe
- [New] — constructor accepting stt.Config

---

## gladia

```go
import "github.com/lookatitude/beluga-ai/voice/stt/providers/gladia"
```

Package gladia provides the Gladia STT provider for the Beluga AI voice
pipeline. It uses the Gladia API for batch transcription and WebSocket
streaming for real-time transcription.

## Registration

This package registers itself as "gladia" with the stt registry. Import it
with a blank identifier to enable:

```go
import _ "github.com/lookatitude/beluga-ai/voice/stt/providers/gladia"
```

## Usage

```go
engine, err := stt.New("gladia", stt.Config{
    Language: "en",
    Extra: map[string]any{"api_key": "..."},
})
text, err := engine.Transcribe(ctx, audioBytes)
```

Batch transcription uploads audio via multipart form, creates a
transcription job, and polls for the result. Streaming uses Gladia's live
WebSocket endpoint.

## Configuration

Required configuration in Config.Extra:

- api_key — Gladia API key (required)
- base_url — Custom API base URL (optional)

## Exported Types

- [Engine] — implements stt.STT using Gladia
- [New] — constructor accepting stt.Config

---

## groq

```go
import "github.com/lookatitude/beluga-ai/voice/stt/providers/groq"
```

Package groq provides the Groq STT provider for the Beluga AI voice pipeline.
It uses the Groq Whisper endpoint (OpenAI-compatible API) for transcription.

## Registration

This package registers itself as "groq" with the stt registry. Import it
with a blank identifier to enable:

```go
import _ "github.com/lookatitude/beluga-ai/voice/stt/providers/groq"
```

## Usage

```go
engine, err := stt.New("groq", stt.Config{
    Model: "whisper-large-v3",
    Extra: map[string]any{"api_key": "gsk-..."},
})
text, err := engine.Transcribe(ctx, audioBytes)
```

Groq Whisper does not support native streaming. TranscribeStream buffers
all audio chunks and transcribes them in a single batch request.

## Configuration

Required configuration in Config.Extra:

- api_key — Groq API key (required)
- base_url — Custom API base URL (optional)

The default model is "whisper-large-v3".

## Exported Types

- [Engine] — implements stt.STT using Groq Whisper
- [New] — constructor accepting stt.Config

---

## whisper

```go
import "github.com/lookatitude/beluga-ai/voice/stt/providers/whisper"
```

Package whisper provides the OpenAI Whisper STT provider for the Beluga AI
voice pipeline. It uses the OpenAI Audio Transcriptions API for batch
transcription via multipart form upload.

## Registration

This package registers itself as "whisper" with the stt registry. Import it
with a blank identifier to enable:

```go
import _ "github.com/lookatitude/beluga-ai/voice/stt/providers/whisper"
```

## Usage

```go
engine, err := stt.New("whisper", stt.Config{
    Model: "whisper-1",
    Extra: map[string]any{"api_key": "sk-..."},
})
text, err := engine.Transcribe(ctx, audioBytes)
```

Whisper does not support native streaming. TranscribeStream transcribes
each audio chunk independently as a batch request.

## Configuration

Required configuration in Config.Extra:

- api_key — OpenAI API key (required)
- base_url — Custom API base URL (optional)

The default model is "whisper-1".

## Exported Types

- [Engine] — implements stt.STT using OpenAI Whisper
- [New] — constructor accepting stt.Config
