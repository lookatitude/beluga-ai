---
title: "VAD Providers — Voice Detection"
description: "2 voice activity detection providers: Silero and WebRTC VAD. Detect speech in audio streams with confidence scoring in Go with Beluga AI."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "voice activity detection, VAD, Silero VAD, WebRTC VAD, speech detection, Go voice, Beluga AI"
---

Beluga AI provides a unified `voice.VAD` interface for detecting speech in audio streams. VAD providers analyze audio frames and report whether speech is present, along with confidence scores and state transitions (speech start, speech end, silence).

## How It Works

All VAD providers implement the same interface:

```go
type VAD interface {
    DetectActivity(ctx context.Context, audio []byte) (ActivityResult, error)
}
```

The `ActivityResult` contains:

```go
type ActivityResult struct {
    IsSpeech   bool         // true if speech was detected
    EventType  VADEventType // speech_start, speech_end, or silence
    Confidence float64      // detection confidence (0.0 to 1.0)
}
```

You can instantiate any provider two ways:

**Via the registry** (recommended for dynamic configuration):

```go
import (
    "github.com/lookatitude/beluga-ai/voice"
    _ "github.com/lookatitude/beluga-ai/voice/vad/providers/silero"
)

vad, err := voice.NewVAD("silero", map[string]any{
    "threshold": 0.5,
})
```

**Via direct construction** (for compile-time type safety):

```go
import "github.com/lookatitude/beluga-ai/voice/vad/providers/silero"

vad, err := silero.New(silero.Config{
    Threshold:  0.5,
    SampleRate: 16000,
})
```

## Event Types

VAD providers track state transitions between speech and silence:

| Event Type     | Constant            | Description                        |
|----------------|---------------------|------------------------------------|
| Speech Start   | `VADSpeechStart`    | Transition from silence to speech  |
| Speech End     | `VADSpeechEnd`      | Transition from speech to silence  |
| Silence        | `VADSilence`        | No speech detected (ongoing)       |

## Built-in Energy VAD

The `voice` package includes a built-in energy-threshold VAD that requires no external dependencies. It computes the RMS energy of 16-bit PCM audio and compares it against a configurable threshold:

```go
import "github.com/lookatitude/beluga-ai/voice"

vad, err := voice.NewVAD("energy", map[string]any{
    "threshold": 1000.0,
})
```

## Available Providers

| Provider | Registry Name | Description |
|----------|---------------|-------------|
| Energy   | `energy`      | Built-in RMS energy threshold detector |
| [Silero](/providers/vad/silero) | `silero` | ONNX model-based detection with energy fallback |
| [WebRTC](/providers/vad/webrtc) | `webrtc` | Energy + zero-crossing rate analysis |

## Provider Discovery

List all registered VAD providers at runtime:

```go
for _, name := range voice.ListVAD() {
    fmt.Println(name)
}
```

## Choosing a Provider

| Use Case | Recommended Provider | Reason |
|----------|---------------------|--------|
| Quick prototyping | `energy` | Zero dependencies, built-in |
| Production accuracy | `silero` | Neural network-based detection |
| Low-latency | `webrtc` | Lightweight dual-metric analysis |
| Noise filtering | `webrtc` | Zero-crossing rate rejects noise |
