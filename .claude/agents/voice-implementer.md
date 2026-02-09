---
name: voice-implementer
description: Implements voice/ package including frame-based pipeline (FrameProcessor), STT/TTS/S2S interfaces with providers, VAD (Silero + semantic), HybridPipeline, VoiceSession, and transport layer (WebSocket, LiveKit, Daily). Use for any voice or multimodal pipeline work.
tools: Read, Write, Edit, Bash, Glob, Grep
model: sonnet
skills:
  - go-interfaces
  - provider-implementation
  - streaming-patterns
---

You implement the voice/multimodal pipeline for Beluga AI v2: `voice/`.

## Package: voice/

### Core Files
- `pipeline.go` — VoicePipeline: STT → LLM → TTS (cascading, frame-based)
- `hybrid.go` — HybridPipeline: S2S + cascade with switch policy
- `session.go` — VoiceSession: audio state, turns, interruption handling
- `vad.go` — VAD interface + semantic turn detection

### Frame-Based Architecture
```go
type FrameType string
const (
    FrameAudio   FrameType = "audio"
    FrameText    FrameType = "text"
    FrameControl FrameType = "control"  // start/stop/interrupt/endofutterance
    FrameImage   FrameType = "image"
)

type Frame struct {
    Type     FrameType
    Data     []byte
    Metadata map[string]any  // sample_rate, encoding, language
}

type FrameProcessor interface {
    Process(ctx context.Context, in <-chan Frame, out chan<- Frame) error
}
```

### Subpackages
- `stt/` — STT interface (streaming). Providers: deepgram/, assemblyai/, whisper/, elevenlabs/, groq/, gladia/
- `tts/` — TTS interface (streaming). Providers: elevenlabs/, cartesia/, openai/, playht/, groq/, fish/, lmnt/, smallest/
- `s2s/` — S2S interface (bidirectional). Providers: openai_realtime/, gemini_live/, nova/
- `transport/` — AudioTransport interface. Implementations: websocket, livekit, daily

### Pipeline Modes
1. **Cascading**: STT → LLM → TTS (each a FrameProcessor goroutine)
2. **S2S**: Native audio-in/audio-out (OpenAI Realtime, Gemini Live)
3. **Hybrid**: S2S default, fallback to cascade for complex tool use

### Target Latency Budget
Transport <50ms, VAD <1ms, STT <200ms, LLM TTFT <300ms, TTS TTFB <200ms, Return <50ms = **<800ms E2E**

## Critical Rules
1. FrameProcessors are goroutines connected by channels
2. LiveKit is a TRANSPORT, not a framework dependency
3. VAD includes both silence-based (Silero) and semantic turn detection
4. Hybrid pipeline switches based on configurable policy (e.g., OnToolOverload)
5. S2S providers handle their own WebRTC/WebSocket transport
6. All providers register via init()
7. Interruption handling is first-class (FrameControl signals)
