---
title: "WebRTC"
description: "Integration guide for the WebRTC-style VAD provider in Beluga AI."
---

The WebRTC provider implements voice activity detection using a dual-metric approach: RMS energy analysis combined with zero-crossing rate (ZCR). This pure Go implementation requires no external dependencies or CGO and effectively distinguishes voiced speech from background noise.

Choose WebRTC VAD when you need a lightweight, pure Go voice activity detector with zero external dependencies. It works well in controlled environments and requires no CGO or model files. The dual-metric approach (energy + zero-crossing rate) effectively rejects high-energy noise that would fool a pure energy detector. For higher accuracy in noisy environments, consider [Silero](/providers/vad/silero).

## Installation

```bash
go get github.com/lookatitude/beluga-ai/voice/vad/providers/webrtc
```

## Configuration

| Field              | Required | Default  | Description                             |
|--------------------|----------|----------|-----------------------------------------|
| `energyThreshold`  | No       | `1000.0` | RMS energy threshold for speech         |
| `zcrThreshold`     | No       | `0.1`    | Zero-crossing rate threshold            |

**Registry configuration keys:**

| Key             | Type      | Maps to            |
|-----------------|-----------|-------------------|
| `threshold`     | `float64` | `energyThreshold` |
| `zcr_threshold` | `float64` | `zcrThreshold`    |

## Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/voice"
    _ "github.com/lookatitude/beluga-ai/voice/vad/providers/webrtc"
)

func main() {
    vad, err := voice.NewVAD("webrtc", map[string]any{
        "threshold":     1000.0,
        "zcr_threshold": 0.1,
    })
    if err != nil {
        log.Fatal(err)
    }

    // audioPCM is 16-bit little-endian PCM audio data
    var audioPCM []byte // ... obtained from audio source

    result, err := vad.DetectActivity(context.Background(), audioPCM)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Speech: %v, Event: %s, Confidence: %.2f\n",
        result.IsSpeech, result.EventType, result.Confidence)
}
```

## Advanced Features

### Dual-Metric Detection

The WebRTC provider uses two complementary metrics to classify audio:

1. **RMS Energy** — Measures signal amplitude. High energy suggests speech or loud sounds.
2. **Zero-Crossing Rate (ZCR)** — Measures how often the signal crosses zero. Noise tends to have high ZCR, while voiced speech has lower ZCR.

Speech is detected when energy exceeds the threshold **and** the zero-crossing rate is below the ZCR threshold. This combination rejects high-energy noise (fans, traffic) that would fool a pure energy detector.

### Confidence Scoring

The confidence score reflects both metrics:

- Base confidence is the ratio of RMS energy to twice the energy threshold, clamped to `[0, 1]`
- When ZCR exceeds the threshold (suggesting noise), confidence is halved

```go
result, err := vad.DetectActivity(ctx, audioPCM)
if err != nil {
    log.Fatal(err)
}

if result.Confidence > 0.8 {
    fmt.Println("High-confidence speech detection")
}
```

### Threshold Tuning

Energy and ZCR thresholds can be adjusted independently:

| Scenario | Energy Threshold | ZCR Threshold | Effect |
|----------|-----------------|---------------|--------|
| Quiet room | `500.0` | `0.1` | More sensitive to soft speech |
| Noisy environment | `2000.0` | `0.05` | Strict filtering of background noise |
| Default | `1000.0` | `0.1` | Balanced for general use |

### Continuous Detection

The provider tracks state transitions between speech and silence:

```go
for _, chunk := range audioChunks {
    result, err := vad.DetectActivity(ctx, chunk)
    if err != nil {
        log.Fatal(err)
    }

    switch result.EventType {
    case voice.VADSpeechStart:
        fmt.Println("Speech started")
    case voice.VADSpeechEnd:
        fmt.Println("Speech ended")
    case voice.VADSilence:
        // Ongoing silence
    }
}
```

## Direct Construction

For compile-time type safety, construct the provider directly:

```go
import "github.com/lookatitude/beluga-ai/voice/vad/providers/webrtc"

vad := webrtc.New(1500.0, 0.08)
```

The constructor takes `energyThreshold` and `zcrThreshold` as positional arguments. Zero values use the defaults.

## Error Handling

```go
result, err := vad.DetectActivity(ctx, audioPCM)
if err != nil {
    log.Printf("VAD detection failed: %v", err)
}
```

Audio data must be 16-bit little-endian PCM. Frames shorter than 4 bytes return a silence result without error.
