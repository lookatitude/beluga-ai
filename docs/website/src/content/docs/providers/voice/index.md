---
title: "Voice Providers — STT, TTS & S2S"
description: "14 voice providers for speech-to-text, text-to-speech, and speech-to-speech: Deepgram, ElevenLabs, Cartesia, and more. Voice AI in Go with Beluga AI."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "voice providers, STT, TTS, speech-to-text, text-to-speech, Deepgram, ElevenLabs, voice AI, Go, Beluga AI"
---

Beluga AI provides a unified voice pipeline with three provider categories: Speech-to-Text (STT), Text-to-Speech (TTS), and Speech-to-Speech (S2S). Every provider registers itself via `init()`, so a blank import is sufficient to make it available through the registry.

## Architecture

The voice pipeline uses a frame-based processing model. Atomic `Frame` values (audio chunks, text fragments, control signals) flow through linked `FrameProcessor` goroutines via Go channels. Every voice provider can be used standalone or wrapped as a `FrameProcessor` for integration into a pipeline.

```
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│Transport│───>│  VAD    │───>│  STT    │───>│  LLM    │
└─────────┘    └─────────┘    └─────────┘    └─────────┘
                                                   │
                              ┌─────────┐          │
                              │Transport│<───┌─────────┐
                              └─────────┘    │  TTS    │
                                             └─────────┘
```

For S2S providers, the STT/LLM/TTS cascade is replaced by a single provider:

```
┌─────────┐    ┌─────────┐    ┌─────────────────┐    ┌─────────┐
│Transport│───>│  VAD    │───>│  S2S Provider    │───>│Transport│
└─────────┘    └─────────┘    └─────────────────┘    └─────────┘
```

## STT Interface

All STT providers implement:

```go
type STT interface {
    Transcribe(ctx context.Context, audio []byte, opts ...Option) (string, error)
    TranscribeStream(ctx context.Context, audioStream iter.Seq2[[]byte, error], opts ...Option) iter.Seq2[TranscriptEvent, error]
}
```

Instantiate via the registry or direct construction:

```go
import (
    "github.com/lookatitude/beluga-ai/voice/stt"
    _ "github.com/lookatitude/beluga-ai/voice/stt/providers/deepgram"
)

engine, err := stt.New("deepgram", stt.Config{
    Language: "en",
    Model:    "nova-2",
    Extra:    map[string]any{"api_key": os.Getenv("DEEPGRAM_API_KEY")},
})
```

## TTS Interface

All TTS providers implement:

```go
type TTS interface {
    Synthesize(ctx context.Context, text string, opts ...Option) ([]byte, error)
    SynthesizeStream(ctx context.Context, textStream iter.Seq2[string, error], opts ...Option) iter.Seq2[[]byte, error]
}
```

Instantiate via the registry or direct construction:

```go
import (
    "github.com/lookatitude/beluga-ai/voice/tts"
    _ "github.com/lookatitude/beluga-ai/voice/tts/providers/elevenlabs"
)

engine, err := tts.New("elevenlabs", tts.Config{
    Voice: "rachel",
    Extra: map[string]any{"api_key": os.Getenv("ELEVENLABS_API_KEY")},
})
```

## S2S Interface

S2S providers handle bidirectional audio streaming natively:

```go
type S2S interface {
    Start(ctx context.Context, opts ...Option) (Session, error)
}

type Session interface {
    SendAudio(ctx context.Context, audio []byte) error
    SendText(ctx context.Context, text string) error
    SendToolResult(ctx context.Context, result schema.ToolResult) error
    Recv() <-chan SessionEvent
    Interrupt(ctx context.Context) error
    Close() error
}
```

Instantiate via the registry:

```go
import (
    "github.com/lookatitude/beluga-ai/voice/s2s"
    _ "github.com/lookatitude/beluga-ai/voice/s2s/providers/openai"
)

engine, err := s2s.New("openai_realtime", s2s.Config{
    Voice: "alloy",
    Model: "gpt-4o-realtime-preview",
    Extra: map[string]any{"api_key": os.Getenv("OPENAI_API_KEY")},
})
session, err := engine.Start(ctx)
defer session.Close()
```

## Configuration

### STT Config

| Field         | Type              | Description                                        |
|---------------|-------------------|----------------------------------------------------|
| `Language`    | `string`          | BCP-47 language code (e.g., `"en-US"`, `"es"`)    |
| `Model`       | `string`          | Provider-specific model name                       |
| `Punctuation` | `bool`            | Enable automatic punctuation insertion             |
| `Diarization` | `bool`            | Enable speaker diarization                         |
| `SampleRate`  | `int`             | Audio sample rate in Hz                            |
| `Encoding`    | `string`          | Audio encoding format (`"linear16"`, `"opus"`)     |
| `Extra`       | `map[string]any`  | Provider-specific configuration (e.g., `api_key`)  |

### TTS Config

| Field        | Type              | Description                                        |
|--------------|-------------------|----------------------------------------------------|
| `Voice`      | `string`          | Voice identifier (provider-specific)               |
| `Model`      | `string`          | Provider-specific model name                       |
| `SampleRate` | `int`             | Output sample rate in Hz                           |
| `Format`     | `AudioFormat`     | Output format: `pcm`, `opus`, `mp3`, `wav`         |
| `Speed`      | `float64`         | Speech rate multiplier (1.0 = normal)              |
| `Pitch`      | `float64`         | Voice pitch adjustment (-20.0 to 20.0)             |
| `Extra`      | `map[string]any`  | Provider-specific configuration (e.g., `api_key`)  |

### S2S Config

| Field          | Type                   | Description                                    |
|----------------|------------------------|------------------------------------------------|
| `Voice`        | `string`               | Voice identifier (provider-specific)           |
| `Model`        | `string`               | Provider-specific model name                   |
| `Instructions` | `string`               | System prompt for the session                  |
| `Tools`        | `[]schema.ToolDefinition` | Tools available to the S2S session          |
| `SampleRate`   | `int`                  | Audio sample rate in Hz                        |
| `Extra`        | `map[string]any`       | Provider-specific configuration                |

## FrameProcessor Integration

Every voice provider can be wrapped as a `FrameProcessor` for pipeline use:

```go
// STT as FrameProcessor
sttProcessor := stt.AsFrameProcessor(sttEngine, stt.WithLanguage("en"))

// TTS as FrameProcessor
ttsProcessor := tts.AsFrameProcessor(ttsEngine, 24000, tts.WithVoice("rachel"))

// S2S as FrameProcessor
s2sProcessor := s2s.AsFrameProcessor(s2sEngine, s2s.WithVoice("alloy"))

// Chain processors into a pipeline
pipeline := voice.Chain(sttProcessor, llmProcessor, ttsProcessor)
```

## STT Providers

| Provider | Registry Name | Streaming | Description |
|----------|---------------|-----------|-------------|
| [Deepgram](/providers/voice/deepgram) | `deepgram` | Native WebSocket | Real-time STT with Nova-2 models |
| [AssemblyAI](/providers/voice/assemblyai) | `assemblyai` | Native WebSocket | Real-time and async transcription |
| [OpenAI Whisper](/providers/voice/whisper) | `whisper` | Chunked batch | Whisper models via OpenAI API |
| [Gladia](/providers/voice/gladia) | `gladia` | Native WebSocket | Real-time STT with language detection |
| [ElevenLabs STT](/providers/voice/elevenlabs) | `elevenlabs` | Chunked batch | Scribe transcription engine |
| [Groq Whisper](/providers/voice/groq-whisper) | `groq` | Buffered batch | Ultra-fast Whisper inference on LPU |

## TTS Providers

| Provider | Registry Name | Description |
|----------|---------------|-------------|
| [ElevenLabs](/providers/voice/elevenlabs) | `elevenlabs` | Premium voice cloning and synthesis |
| [Cartesia](/providers/voice/cartesia) | `cartesia` | Low-latency Sonic voice engine |
| [PlayHT](/providers/voice/playht) | `playht` | AI voice generation platform |
| [LMNT](/providers/voice/lmnt) | `lmnt` | Ultra-low-latency voice synthesis |
| [Fish Audio](/providers/voice/fish) | `fish` | Open-source voice synthesis |
| [Smallest AI](/providers/voice/smallest) | `smallest` | Lightning-fast TTS engine |
| [Groq TTS](/providers/voice/groq-whisper) | `groq` | Fast TTS via Groq API |

## S2S Providers

| Provider | Registry Name | Description |
|----------|---------------|-------------|
| [OpenAI Realtime](/providers/voice/openai-realtime) | `openai_realtime` | Bidirectional audio via WebSocket |
| [Gemini Live](/providers/voice/gemini-live) | `gemini_live` | Google's live multimodal API |
| [Amazon Nova S2S](/providers/voice/nova-s2s) | `nova` | Nova Sonic via AWS Bedrock |

## Hooks

Each provider category supports lifecycle hooks:

```go
// STT hooks
sttHooks := stt.Hooks{
    OnTranscript: func(ctx context.Context, event stt.TranscriptEvent) {
        log.Printf("transcript: %s (final=%v)", event.Text, event.IsFinal)
    },
    OnUtterance: func(ctx context.Context, text string) {
        log.Printf("utterance complete: %s", text)
    },
}

// TTS hooks
ttsHooks := tts.Hooks{
    BeforeSynthesize: func(ctx context.Context, text string) {
        log.Printf("synthesizing: %s", text)
    },
    OnAudioChunk: func(ctx context.Context, chunk []byte) {
        log.Printf("audio chunk: %d bytes", len(chunk))
    },
}

// S2S hooks
s2sHooks := s2s.Hooks{
    OnTurn: func(ctx context.Context, userText, agentText string) {
        log.Printf("turn: user=%q agent=%q", userText, agentText)
    },
    OnInterrupt: func(ctx context.Context) {
        log.Println("user interrupted")
    },
    OnToolCall: func(ctx context.Context, call schema.ToolCall) {
        log.Printf("tool call: %s", call.Name)
    },
}

// Compose multiple hooks
combined := stt.ComposeHooks(loggingHooks, metricsHooks)
```
