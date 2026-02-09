---
title: Voice Services
description: Integrate speech-to-text, text-to-speech, speech-to-speech, VAD, and transport providers for voice AI pipelines.
sidebar:
  order: 0
---

Beluga AI provides a frame-based voice pipeline with separate registries for speech-to-text (STT), text-to-speech (TTS), speech-to-speech (S2S), voice activity detection (VAD), and real-time transport. Each category follows the standard registry pattern.

## Architecture Overview

```
Audio In → Transport → VAD → STT → Agent → TTS → Transport → Audio Out
                              └─── S2S (bypasses STT/TTS) ──┘
```

The voice pipeline processes audio as discrete `Frame` values through a chain of `FrameProcessor` implementations. STT, TTS, and S2S engines can each operate as frame processors within this pipeline.

## STT Providers

Speech-to-text providers convert audio to text, supporting both batch and streaming transcription.

| Provider | Registry Name | Streaming | Languages | Import Path |
|----------|--------------|-----------|-----------|-------------|
| Deepgram | `deepgram` | Yes | 36+ | `voice/stt/providers/deepgram` |
| AssemblyAI | `assemblyai` | Yes | 20+ | `voice/stt/providers/assemblyai` |
| Whisper | `whisper` | No | 99 | `voice/stt/providers/whisper` |
| ElevenLabs | `elevenlabs` | Yes | 29 | `voice/stt/providers/elevenlabs` |
| Groq | `groq` | No | 50+ | `voice/stt/providers/groq` |
| Gladia | `gladia` | Yes | 100+ | `voice/stt/providers/gladia` |

### STT Interface

```go
type STT interface {
    Transcribe(ctx context.Context, audio []byte) (string, error)
    TranscribeStream(ctx context.Context, audio iter.Seq2[[]byte, error]) iter.Seq2[TranscriptEvent, error]
}
```

### Deepgram STT

```bash
export DEEPGRAM_API_KEY="..."
```

```go
import _ "github.com/lookatitude/beluga-ai/voice/stt/providers/deepgram"

engine, err := stt.New("deepgram", stt.Config{
    Language:   "en",
    SampleRate: 16000,
    Extra: map[string]any{
        "api_key": os.Getenv("DEEPGRAM_API_KEY"),
        "model":   "nova-2",
    },
})
if err != nil {
    log.Fatal(err)
}

// Batch transcription
text, err := engine.Transcribe(ctx, audioBytes)

// Streaming transcription
for event, err := range engine.TranscribeStream(ctx, audioStream) {
    if err != nil {
        break
    }
    if event.IsFinal {
        fmt.Printf("[%v] %s\n", event.Timestamp, event.Text)
    }
}
```

### Whisper STT (Groq-hosted)

```go
import _ "github.com/lookatitude/beluga-ai/voice/stt/providers/groq"

engine, err := stt.New("groq", stt.Config{
    Language: "en",
    Extra: map[string]any{
        "api_key": os.Getenv("GROQ_API_KEY"),
        "model":   "whisper-large-v3",
    },
})
```

### Using STT as a Frame Processor

```go
processor := stt.AsFrameProcessor(engine)
// processor implements voice.FrameProcessor and can be inserted into the pipeline
```

## TTS Providers

Text-to-speech providers convert text to audio, supporting both batch and streaming synthesis.

| Provider | Registry Name | Streaming | Voices | Import Path |
|----------|--------------|-----------|--------|-------------|
| ElevenLabs | `elevenlabs` | Yes | 1000+ | `voice/tts/providers/elevenlabs` |
| Cartesia | `cartesia` | Yes | Custom | `voice/tts/providers/cartesia` |
| PlayHT | `playht` | Yes | 600+ | `voice/tts/providers/playht` |
| LMNT | `lmnt` | Yes | Custom | `voice/tts/providers/lmnt` |
| Fish Audio | `fish` | Yes | Custom | `voice/tts/providers/fish` |
| Groq | `groq` | No | Standard | `voice/tts/providers/groq` |
| Smallest AI | `smallest` | Yes | Custom | `voice/tts/providers/smallest` |

### TTS Interface

```go
type TTS interface {
    Synthesize(ctx context.Context, text string) ([]byte, error)
    SynthesizeStream(ctx context.Context, text iter.Seq2[string, error]) iter.Seq2[[]byte, error]
    OutputFormat() AudioFormat
    SampleRate() int
}
```

### ElevenLabs TTS

```bash
export ELEVENLABS_API_KEY="..."
```

```go
import _ "github.com/lookatitude/beluga-ai/voice/tts/providers/elevenlabs"

engine, err := tts.New("elevenlabs", tts.Config{
    Voice:      "rachel",
    SampleRate: 24000,
    Extra: map[string]any{
        "api_key":        os.Getenv("ELEVENLABS_API_KEY"),
        "model_id":       "eleven_multilingual_v2",
        "output_format":  "pcm_24000",
    },
})
if err != nil {
    log.Fatal(err)
}

// Batch synthesis
audio, err := engine.Synthesize(ctx, "Hello, how can I help you?")

// Streaming synthesis
for chunk, err := range engine.SynthesizeStream(ctx, textStream) {
    if err != nil {
        break
    }
    transport.Send(chunk)
}
```

### Cartesia TTS

```go
import _ "github.com/lookatitude/beluga-ai/voice/tts/providers/cartesia"

engine, err := tts.New("cartesia", tts.Config{
    Voice:      "sonic-english",
    SampleRate: 24000,
    Extra: map[string]any{
        "api_key": os.Getenv("CARTESIA_API_KEY"),
    },
})
```

### Using TTS as a Frame Processor

```go
processor := tts.AsFrameProcessor(engine, 24000)
```

## S2S Providers

Speech-to-speech providers handle bidirectional audio conversations, bypassing separate STT and TTS stages for lower latency.

| Provider | Registry Name | Model | Import Path |
|----------|--------------|-------|-------------|
| OpenAI Realtime | `openai_realtime` | GPT-4o Realtime | `voice/s2s/providers/openai` |
| Amazon Nova | `nova` | Nova S2S | `voice/s2s/providers/nova` |
| Google Gemini | `gemini` | Gemini Live | `voice/s2s/providers/gemini` |
| Silero | `silero` | Silero S2S | `voice/s2s/providers/silero` |

### S2S Interface

```go
type S2S interface {
    Start(ctx context.Context) (Session, error)
}

type Session interface {
    SendAudio(ctx context.Context, audio []byte) error
    SendText(ctx context.Context, text string) error
    Receive(ctx context.Context) iter.Seq2[Event, error]
    Close() error
}
```

### OpenAI Realtime

```go
import _ "github.com/lookatitude/beluga-ai/voice/s2s/providers/openai"

engine, err := s2s.New("openai_realtime", s2s.Config{
    Voice: "alloy",
    Model: "gpt-4o-realtime-preview",
    Extra: map[string]any{
        "api_key": os.Getenv("OPENAI_API_KEY"),
    },
})
if err != nil {
    log.Fatal(err)
}

session, err := engine.Start(ctx)
if err != nil {
    log.Fatal(err)
}
defer session.Close()

// Send audio and receive responses
if err := session.SendAudio(ctx, audioChunk); err != nil {
    log.Fatal(err)
}

for event, err := range session.Receive(ctx) {
    if err != nil {
        break
    }
    switch event.Type {
    case s2s.EventAudio:
        transport.Send(event.Audio)
    case s2s.EventTranscript:
        fmt.Println("Agent:", event.Text)
    }
}
```

## VAD Providers

Voice activity detection determines when speech starts and stops in an audio stream.

| Provider | Registry Name | Type | Import Path |
|----------|--------------|------|-------------|
| Silero | `silero` | Neural network | `voice/vad/providers/silero` |
| WebRTC | `webrtc` | Traditional DSP | `voice/vad/providers/webrtc` |

### Silero VAD

Silero VAD uses a neural network model for high-accuracy speech detection.

```go
import _ "github.com/lookatitude/beluga-ai/voice/vad/providers/silero"

detector, err := vad.New("silero", vad.Config{
    Threshold:  0.5,
    SampleRate: 16000,
})
```

### WebRTC VAD

WebRTC VAD is lighter weight and suitable for environments where neural network inference is too expensive.

```go
import _ "github.com/lookatitude/beluga-ai/voice/vad/providers/webrtc"

detector, err := vad.New("webrtc", vad.Config{
    Threshold:  0.5,
    SampleRate: 16000,
})
```

## Transport Providers

Transport providers handle real-time audio streaming between clients and the voice pipeline.

| Provider | Registry Name | Protocol | Import Path |
|----------|--------------|----------|-------------|
| WebSocket | `websocket` | WebSocket | `voice/transport` |
| LiveKit | `livekit` | WebRTC (LiveKit) | `voice/transport/providers/livekit` |
| Daily | `daily` | WebRTC (Daily) | `voice/transport/providers/daily` |
| Pipecat | `pipecat` | Pipecat protocol | `voice/transport/providers/pipecat` |

### WebSocket Transport

WebSocket is the built-in transport for browser-based audio streaming.

```go
transport, err := transport.New("websocket", transport.Config{
    SampleRate: 16000,
    Channels:   1,
})
```

### LiveKit Transport

LiveKit provides WebRTC-based transport with room management for multi-party voice.

```go
import _ "github.com/lookatitude/beluga-ai/voice/transport/providers/livekit"

transport, err := transport.New("livekit", transport.Config{
    Extra: map[string]any{
        "url":       os.Getenv("LIVEKIT_URL"),
        "api_key":   os.Getenv("LIVEKIT_API_KEY"),
        "api_secret": os.Getenv("LIVEKIT_API_SECRET"),
        "room_name": "voice-agent-room",
    },
})
```

## Complete Voice Pipeline

Assemble a full voice pipeline by combining STT, TTS, VAD, and transport:

```go
package main

import (
    "context"
    "log"

    "github.com/lookatitude/beluga-ai/voice"
    "github.com/lookatitude/beluga-ai/voice/stt"
    "github.com/lookatitude/beluga-ai/voice/tts"
    "github.com/lookatitude/beluga-ai/voice/vad"

    _ "github.com/lookatitude/beluga-ai/voice/stt/providers/deepgram"
    _ "github.com/lookatitude/beluga-ai/voice/tts/providers/elevenlabs"
    _ "github.com/lookatitude/beluga-ai/voice/vad/providers/silero"
)

func main() {
    ctx := context.Background()

    sttEngine, err := stt.New("deepgram", stt.Config{Language: "en"})
    if err != nil {
        log.Fatal(err)
    }

    ttsEngine, err := tts.New("elevenlabs", tts.Config{Voice: "rachel"})
    if err != nil {
        log.Fatal(err)
    }

    vadEngine, err := vad.New("silero", vad.Config{Threshold: 0.5})
    if err != nil {
        log.Fatal(err)
    }

    pipeline := voice.NewPipeline(
        vad.AsFrameProcessor(vadEngine),
        stt.AsFrameProcessor(sttEngine),
        // Your agent processor here
        tts.AsFrameProcessor(ttsEngine, 24000),
    )

    if err := pipeline.Run(ctx); err != nil {
        log.Fatal(err)
    }
}
```

## Choosing Voice Providers

| Component | Low Latency | High Accuracy | Cost Effective |
|-----------|------------|---------------|----------------|
| STT | Deepgram Nova-2 | AssemblyAI | Groq Whisper |
| TTS | Cartesia | ElevenLabs | Groq |
| S2S | OpenAI Realtime | OpenAI Realtime | Gemini Live |
| VAD | WebRTC | Silero | WebRTC |
| Transport | WebSocket | LiveKit | WebSocket |
