---
title: Voice AI Pipeline
description: Build real-time voice applications with frame-based processing, STT, TTS, S2S, and transport layers.
---

The `voice` package provides a frame-based processing pipeline for building real-time voice AI applications. Audio frames flow through linked processors — STT, LLM, TTS — connected by Go channels, targeting sub-800ms end-to-end latency.

## Architecture

```
┌───────────┐    ┌─────┐    ┌─────┐    ┌─────┐    ┌───────────┐
│ Transport │───→│ VAD │───→│ STT │───→│ LLM │───→│    TTS    │───→ Transport
│ (WebSocket│    │     │    │     │    │     │    │           │    (audio out)
│  LiveKit) │    └─────┘    └─────┘    └─────┘    └───────────┘
└───────────┘
```

Three pipeline modes:

| Mode | Flow | Latency | Use Case |
|------|------|---------|----------|
| **Cascading** | STT → LLM → TTS | ~800ms | Full control, tool use |
| **S2S** | Audio → Model → Audio | ~300ms | Low latency, native multimodal |
| **Hybrid** | S2S default, cascade fallback | ~300-800ms | Best of both worlds |

## Core Concepts

### Frames

Frames are the atomic data unit flowing through the pipeline:

```go
// Frame types
voice.FrameAudio   // Raw audio data (PCM, opus)
voice.FrameText    // Text (transcript, LLM output)
voice.FrameControl // Control signals (start, stop, interrupt)
voice.FrameImage   // Image/video for multimodal

// Create frames
audioFrame := voice.NewAudioFrame(pcmData, 16000) // 16kHz audio
textFrame := voice.NewTextFrame("Hello, world!")
controlFrame := voice.NewControlFrame(voice.SignalInterrupt)
```

### Frame Processors

Every pipeline stage implements `FrameProcessor`:

```go
type FrameProcessor interface {
	Process(ctx context.Context, in <-chan Frame, out chan<- Frame) error
}
```

Processors read from `in`, process frames, and write to `out`. They run as goroutines connected by buffered channels.

### Chaining Processors

Build pipelines by chaining processors:

```go
pipeline := voice.Chain(
	vadProcessor,
	sttProcessor,
	llmProcessor,
	ttsProcessor,
)

// Run the pipeline
err := pipeline.Process(ctx, audioIn, audioOut)
```

## Speech-to-Text (STT)

Convert audio to text with batch or streaming transcription:

```go
import (
	"github.com/lookatitude/beluga-ai/voice/stt"
	_ "github.com/lookatitude/beluga-ai/voice/stt/providers/deepgram"
)

engine, err := stt.New("deepgram", stt.ProviderConfig{
	APIKey: os.Getenv("DEEPGRAM_API_KEY"),
	Model:  "nova-2",
})
if err != nil {
	log.Fatal(err)
}

// Batch transcription
text, err := engine.Transcribe(ctx, audioBytes,
	stt.WithLanguage("en-US"),
	stt.WithPunctuation(true),
)

// Streaming transcription
for event, err := range engine.TranscribeStream(ctx, audioStream) {
	if err != nil {
		log.Printf("STT error: %v", err)
		break
	}
	if event.IsFinal {
		fmt.Printf("Final: %s\n", event.Text)
	} else {
		fmt.Printf("Interim: %s\n", event.Text)
	}
}

// Use as FrameProcessor in a pipeline
sttProcessor := stt.AsFrameProcessor(engine, stt.WithLanguage("en-US"))
```

### STT Providers

| Provider | Import Path | Features |
|----------|-------------|----------|
| Deepgram | `voice/stt/providers/deepgram` | Real-time, diarization |
| Whisper | `voice/stt/providers/whisper` | OpenAI, offline-capable |
| AssemblyAI | `voice/stt/providers/assemblyai` | Universal model |
| Gladia | `voice/stt/providers/gladia` | Multi-language |
| ElevenLabs | `voice/stt/providers/elevenlabs` | Scribe model |
| Groq | `voice/stt/providers/groq` | Fast Whisper |

## Text-to-Speech (TTS)

Convert text to natural-sounding audio:

```go
import (
	"github.com/lookatitude/beluga-ai/voice/tts"
	_ "github.com/lookatitude/beluga-ai/voice/tts/providers/elevenlabs"
)

engine, err := tts.New("elevenlabs", tts.ProviderConfig{
	APIKey: os.Getenv("ELEVENLABS_API_KEY"),
	Model:  "eleven_turbo_v2_5",
})
if err != nil {
	log.Fatal(err)
}

// Batch synthesis
audio, err := engine.Synthesize(ctx, "Hello, how can I help you today?",
	tts.WithVoice("rachel"),
)

// Streaming synthesis (low latency)
for chunk, err := range engine.SynthesizeStream(ctx, textStream) {
	if err != nil {
		break
	}
	// Play audio chunk immediately
	player.Write(chunk.Data)
}

// Use as FrameProcessor
ttsProcessor := tts.AsFrameProcessor(engine, tts.WithVoice("rachel"))
```

### TTS Providers

| Provider | Import Path | Features |
|----------|-------------|----------|
| ElevenLabs | `voice/tts/providers/elevenlabs` | Voice cloning, low latency |
| Cartesia | `voice/tts/providers/cartesia` | Sonic model, fast |
| PlayHT | `voice/tts/providers/playht` | Voice cloning |
| LMNT | `voice/tts/providers/lmnt` | Real-time streaming |
| Fish | `voice/tts/providers/fish` | Open-source |
| Smallest | `voice/tts/providers/smallest` | Lightweight |
| Groq | `voice/tts/providers/groq` | Fast synthesis |

## Speech-to-Speech (S2S)

Native audio-in/audio-out models for ultra-low latency:

```go
import (
	"github.com/lookatitude/beluga-ai/voice/s2s"
	_ "github.com/lookatitude/beluga-ai/voice/s2s/providers/openai"
)

engine, err := s2s.New("openai", s2s.ProviderConfig{
	APIKey: os.Getenv("OPENAI_API_KEY"),
	Model:  "gpt-4o-realtime",
})
if err != nil {
	log.Fatal(err)
}

// Process audio directly
for outFrame, err := range engine.Process(ctx, audioInStream) {
	if err != nil {
		break
	}
	transport.Send(outFrame)
}
```

### S2S Providers

| Provider | Import Path | Model |
|----------|-------------|-------|
| OpenAI | `voice/s2s/providers/openai` | GPT-4o Realtime |
| Amazon Nova | `voice/s2s/providers/nova` | Nova S2S |
| Google Gemini | `voice/s2s/providers/gemini` | Gemini Live |

## Voice Activity Detection (VAD)

Detect when users are speaking:

```go
import "github.com/lookatitude/beluga-ai/voice"

vad := voice.NewVAD(voice.VADConfig{
	Threshold:      0.5,  // Speech probability threshold
	MinSpeechDuration: 250 * time.Millisecond,
	MinSilenceDuration: 300 * time.Millisecond,
	Provider:       "silero", // or "webrtc"
})
```

### VAD Providers

| Provider | Type | Accuracy |
|----------|------|----------|
| Silero | Neural network | High |
| WebRTC | Energy-based | Fast, lower accuracy |

## Transport Layer

Connect voice pipelines to the outside world:

```go
import (
	"github.com/lookatitude/beluga-ai/voice/transport"
	_ "github.com/lookatitude/beluga-ai/voice/transport/providers/livekit"
)

// WebSocket transport (built-in)
ws := transport.NewWebSocket(transport.WebSocketConfig{
	Addr: ":8080",
	Path: "/ws/voice",
})

// LiveKit transport
lk, err := transport.New("livekit", transport.ProviderConfig{
	Options: map[string]any{
		"url":    os.Getenv("LIVEKIT_URL"),
		"api_key": os.Getenv("LIVEKIT_API_KEY"),
		"secret":  os.Getenv("LIVEKIT_SECRET"),
	},
})
```

### Transport Providers

| Provider | Import Path | Type |
|----------|-------------|------|
| WebSocket | Built-in | Direct browser connection |
| LiveKit | `voice/transport/providers/livekit` | WebRTC SFU |
| Daily | `voice/transport/providers/daily` | WebRTC platform |
| Pipecat | `voice/transport/providers/pipecat` | Pipeline framework |

## Hybrid Pipeline

Combine S2S for speed with cascade fallback for tool use:

```go
hybrid := voice.NewHybridPipeline(voice.HybridConfig{
	S2SEngine:    s2sEngine,
	STT:          sttEngine,
	TTS:          ttsEngine,
	LLM:          model,
	FallbackOnToolUse: true, // Switch to cascade when tools needed
})
```

## Complete Voice Agent Example

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/voice"
	"github.com/lookatitude/beluga-ai/voice/stt"
	"github.com/lookatitude/beluga-ai/voice/tts"
	"github.com/lookatitude/beluga-ai/voice/transport"

	_ "github.com/lookatitude/beluga-ai/llm/providers/openai"
	_ "github.com/lookatitude/beluga-ai/voice/stt/providers/deepgram"
	_ "github.com/lookatitude/beluga-ai/voice/tts/providers/elevenlabs"
)

func main() {
	ctx := context.Background()

	// Set up STT
	sttEngine, err := stt.New("deepgram", stt.ProviderConfig{
		APIKey: os.Getenv("DEEPGRAM_API_KEY"),
	})
	if err != nil {
		log.Fatal(err)
	}

	// Set up TTS
	ttsEngine, err := tts.New("elevenlabs", tts.ProviderConfig{
		APIKey: os.Getenv("ELEVENLABS_API_KEY"),
	})
	if err != nil {
		log.Fatal(err)
	}

	// Set up LLM
	model, err := llm.New("openai", llm.ProviderConfig{
		APIKey: os.Getenv("OPENAI_API_KEY"),
		Model:  "gpt-4o",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Build pipeline
	pipe := voice.NewPipeline(
		voice.WithVAD(voice.NewVAD(voice.VADConfig{Provider: "silero"})),
		voice.WithSTT(stt.AsFrameProcessor(sttEngine)),
		voice.WithTTS(tts.AsFrameProcessor(ttsEngine)),
	)

	// Set up WebSocket transport
	ws := transport.NewWebSocket(transport.WebSocketConfig{
		Addr: ":8080",
		Path: "/ws/voice",
	})

	// Create voice session
	session := voice.NewSession(voice.SessionConfig{
		Pipeline:  pipe,
		Transport: ws,
		LLM:       model,
	})

	log.Println("Voice agent listening on :8080")
	if err := session.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
```

## Latency Budget

| Stage | Target | Description |
|-------|--------|-------------|
| Transport | < 50ms | WebSocket/WebRTC hop |
| VAD | < 1ms | Speech detection |
| STT | < 200ms | Transcription |
| LLM TTFT | < 300ms | First token latency |
| TTS TTFB | < 200ms | First audio byte |
| Return | < 50ms | Transport return |
| **Total** | **< 800ms** | End-to-end |

## Next Steps

- [Building Your First Agent](/guides/first-agent/) — Combine voice with agent logic
- [Tools & MCP](/guides/tools-and-mcp/) — Give voice agents tool access
- [Monitoring & Observability](/guides/observability/) — Track voice pipeline latency
- [Deploying to Production](/guides/deployment/) — Deploy voice agents at scale
