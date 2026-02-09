---
title: Voice Package API
description: API documentation for the frame-based voice pipeline.
---

```go
import "github.com/lookatitude/beluga-ai/voice"
```

Package voice provides a frame-based voice and multimodal pipeline inspired by Pipecat. Atomic `Frame` units flow through linked `FrameProcessor` goroutines via Go channels, supporting cascading (STT → LLM → TTS), native S2S, and hybrid modes.

## Quick Start

```go
pipe := voice.NewPipeline(
    voice.WithTransport(transport),
    voice.WithVAD(vad),
    voice.WithSTT(sttProcessor),
    voice.WithLLM(llmProcessor),
    voice.WithTTS(ttsProcessor),
)

if err := pipe.Run(ctx); err != nil {
    log.Fatal(err)
}
```

## Frame

Atomic unit of data:

```go
type Frame struct {
    Type     FrameType      // "audio", "text", "control", "image"
    Data     []byte         // Raw payload
    Metadata map[string]any // Properties (sample_rate, encoding, signal, etc.)
}

// Constructors
audioFrame := voice.NewAudioFrame(pcmData, 16000)
textFrame := voice.NewTextFrame("Hello world")
controlFrame := voice.NewControlFrame(voice.SignalEndOfUtterance)
imageFrame := voice.NewImageFrame(jpegData, "image/jpeg")
```

### Frame Types

```go
const (
    FrameAudio   FrameType = "audio"
    FrameText    FrameType = "text"
    FrameControl FrameType = "control"
    FrameImage   FrameType = "image"
)
```

### Control Signals

```go
const (
    SignalStart          = "start"
    SignalStop           = "stop"
    SignalInterrupt      = "interrupt"
    SignalEndOfUtterance = "end_of_utterance"
)
```

## FrameProcessor

Process frame streams:

```go
type FrameProcessor interface {
    Process(ctx context.Context, in <-chan Frame, out chan<- Frame) error
}
```

### Chain Processors

```go
chain := voice.Chain(vadProcessor, sttProcessor, llmProcessor, ttsProcessor)
```

### Custom Processor

```go
processor := voice.FrameProcessorFunc(func(ctx context.Context, in <-chan Frame, out chan<- Frame) error {
    defer close(out)
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case frame, ok := <-in:
            if !ok {
                return nil
            }
            // Process frame
            transformed := transformFrame(frame)
            out <- transformed
        }
    }
})
```

## Voice Pipeline

Cascading STT → LLM → TTS:

```go
pipe := voice.NewPipeline(
    voice.WithTransport(wsTransport),
    voice.WithVAD(energyVAD),
    voice.WithSTT(openaiSTT),
    voice.WithLLM(llmProcessor),
    voice.WithTTS(elevenLabsTTS),
    voice.WithChannelBufferSize(128),
    voice.WithHooks(voice.Hooks{
        OnSpeechStart: func(ctx context.Context) {
            log.Println("Speech detected")
        },
        OnTranscript: func(ctx context.Context, text string) {
            log.Printf("Transcribed: %s", text)
        },
    }),
)

pipe.Run(ctx)
```

## VAD (Voice Activity Detection)

```go
type VAD interface {
    DetectActivity(ctx context.Context, audio []byte) (ActivityResult, error)
}

type ActivityResult struct {
    IsSpeech   bool
    EventType  VADEventType // "speech_start", "speech_end", "silence"
    Confidence float64
}
```

### Energy VAD

```go
vad := voice.NewEnergyVAD(voice.EnergyVADConfig{
    Threshold: 1500.0, // RMS energy threshold
})
```

## Voice Session

Track conversation state:

```go
session := voice.NewSession("session-123")

// State transitions
session.Transition(voice.StateListening)
session.Transition(voice.StateSpeaking)
session.Transition(voice.StateIdle)

// Add turns
session.AddTurn(voice.Turn{
    ID:        "turn-1",
    UserText:  "Hello",
    AgentText: "Hi there!",
    StartTime: time.Now(),
})

// Query state
state := session.CurrentState()
count := session.TurnCount()
lastTurn := session.LastTurn()
```

## Hybrid Pipeline

Switch between S2S and cascade modes:

```go
hybrid := voice.NewHybridPipeline(
    voice.WithS2S(s2sProcessor),
    voice.WithCascade(cascadePipeline),
    voice.WithSwitchPolicy(voice.OnToolOverload), // Switch after 3 tool calls
)

hybrid.Run(ctx)

// Update state to trigger switch
hybrid.UpdateState(toolCalls, turnCount)

// Check current mode
mode := hybrid.CurrentMode() // "s2s" or "cascade"
```

### Custom Switch Policy

```go
policy := voice.SwitchPolicyFunc(func(ctx context.Context, state voice.PipelineState) bool {
    return state.ToolCallCount > 5 && state.CurrentMode == voice.ModeS2S
})

hybrid := voice.NewHybridPipeline(
    voice.WithSwitchPolicy(policy),
)
```

## Transport

Audio I/O abstraction:

```go
type Transport interface {
    Recv(ctx context.Context) (<-chan Frame, error)
    Send(ctx context.Context, frame Frame) error
    Close() error
}
```

Implementations: WebSocket, WebRTC, RTMP, etc.

## Hooks

```go
hooks := voice.Hooks{
    OnSpeechStart: func(ctx context.Context) {
        log.Println("User started speaking")
    },
    OnSpeechEnd: func(ctx context.Context) {
        log.Println("User stopped speaking")
    },
    OnTranscript: func(ctx context.Context, text string) {
        log.Printf("STT: %s", text)
    },
    OnResponse: func(ctx context.Context, text string) {
        log.Printf("LLM: %s", text)
    },
    OnError: func(ctx context.Context, err error) error {
        log.Printf("Error: %v", err)
        return err // or return nil to suppress
    },
}

pipe := voice.NewPipeline(voice.WithHooks(hooks))
```

## Latency Budget

Target: < 800ms end-to-end

- Transport: < 50ms
- VAD: < 1ms
- STT: < 200ms
- LLM TTFT: < 300ms
- TTS TTFB: < 200ms
- Return: < 50ms

## Example: WebSocket Voice Agent

```go
// Transport
transport := websocketTransport(conn)

// VAD
vad := voice.NewEnergyVAD(voice.EnergyVADConfig{Threshold: 1500})

// STT
stt := openaiSTT(apiKey)

// LLM
llm := llmProcessor(model)

// TTS
tts := elevenlabsTTS(apiKey)

// Session
session := voice.NewSession(sessionID)

// Pipeline
pipe := voice.NewPipeline(
    voice.WithTransport(transport),
    voice.WithVAD(vad),
    voice.WithSTT(stt),
    voice.WithLLM(llm),
    voice.WithTTS(tts),
    voice.WithSession(session),
)

// Run
if err := pipe.Run(ctx); err != nil {
    log.Fatal(err)
}
```

## See Also

- [Agent Package](./agent.md) for LLM integration
- [Schema Package](./schema.md) for message types
- [Core Package](./core.md) for streaming patterns
