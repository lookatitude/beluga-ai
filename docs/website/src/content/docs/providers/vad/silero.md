---
title: "Silero"
description: "Integration guide for the Silero VAD provider in Beluga AI."
---

The Silero provider implements voice activity detection using the Silero VAD ONNX model. When the ONNX runtime is unavailable, it falls back to an energy-based detector calibrated to approximate Silero's sensitivity. This provider requires CGO.

Choose Silero when you need high-accuracy, neural network-based voice activity detection. The ONNX model provides more precise speech/silence classification than energy-based methods, especially in noisy environments. Note that it requires CGO. For a pure Go alternative with no CGO dependency, use [WebRTC](/providers/vad/webrtc).

## Installation

```bash
go get github.com/lookatitude/beluga-ai/voice/vad/providers/silero
```

This package uses the `cgo` build tag. Ensure CGO is enabled in your build environment.

## Configuration

| Field        | Required | Default  | Description                             |
|--------------|----------|----------|-----------------------------------------|
| `Threshold`  | No       | `0.5`    | Speech probability threshold (0.0-1.0)  |
| `SampleRate` | No       | `16000`  | Audio sample rate in Hz (8000 or 16000) |
| `ModelPath`  | No       | —        | Path to Silero VAD ONNX model file      |

**Registry configuration keys:**

| Key           | Type      | Maps to      |
|---------------|-----------|--------------|
| `threshold`   | `float64` | `Threshold`  |
| `sample_rate` | `int`     | `SampleRate` |
| `model_path`  | `string`  | `ModelPath`  |

## Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/voice"
    _ "github.com/lookatitude/beluga-ai/voice/vad/providers/silero"
)

func main() {
    vad, err := voice.NewVAD("silero", map[string]any{
        "threshold":  0.5,
        "sample_rate": 16000,
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

### ONNX Model Loading

When a `ModelPath` is provided, the Silero provider loads the ONNX model for neural network-based speech detection. Without a model path, it uses an energy-based fallback calibrated to approximate Silero's output:

```go
vad, err := voice.NewVAD("silero", map[string]any{
    "threshold":  0.5,
    "model_path": "/models/silero_vad.onnx",
})
```

### Continuous Detection with State Tracking

The provider tracks speech state internally and emits transition events:

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

### Threshold Tuning

Lower thresholds increase sensitivity (more false positives), while higher thresholds reduce sensitivity (more missed speech):

- `0.3` — High sensitivity, suitable for quiet environments
- `0.5` — Balanced default for general use
- `0.7` — Low sensitivity, rejects ambient noise

## Direct Construction

For compile-time type safety, construct the provider directly:

```go
import "github.com/lookatitude/beluga-ai/voice/vad/providers/silero"

vad, err := silero.New(silero.Config{
    Threshold:  0.5,
    SampleRate: 16000,
    ModelPath:  "/models/silero_vad.onnx",
})
```

## Error Handling

```go
result, err := vad.DetectActivity(ctx, audioPCM)
if err != nil {
    log.Printf("VAD detection failed: %v", err)
    // Handle error — audio frame may be malformed
}
```

Audio data must be 16-bit little-endian PCM. Frames shorter than 2 bytes return a silence result without error.
