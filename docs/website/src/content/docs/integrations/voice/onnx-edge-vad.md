---
title: ONNX Runtime Edge VAD
description: Run Silero VAD with ONNX Runtime on edge devices for voice activity detection in Beluga AI.
---

Running VAD on the edge eliminates the network round-trip for speech detection, which reduces latency and keeps audio data local for privacy-sensitive deployments. Silero VAD with ONNX Runtime enables voice activity detection on resource-constrained devices such as Raspberry Pi, embedded gateways, and kiosk hardware. Choose this approach when you need offline-capable VAD or when sending raw audio to a cloud service is not acceptable. This guide covers configuring the Silero VAD provider for low-resource environments.

## Overview

The `voice/vad` package's Silero provider loads an ONNX model to detect speech in audio frames. By tuning `Threshold`, `FrameSize`, and `SampleRate`, you can balance detection accuracy against CPU and memory usage on edge hardware.

## Prerequisites

- Go 1.23 or later
- Silero VAD ONNX model file (`silero_vad.onnx`)
- ONNX Runtime libraries available on target platform

## Installation

```bash
go get github.com/lookatitude/beluga-ai
```

Ensure the ONNX model is available on the edge device (bundled in the binary, downloaded at startup, or placed in read-only storage).

## Configuration

Create a Silero VAD provider optimized for edge deployment:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "path/filepath"
    "time"

    "github.com/lookatitude/beluga-ai/voice/vad"
    _ "github.com/lookatitude/beluga-ai/voice/vad/providers/silero"
)

func main() {
    ctx := context.Background()

    modelPath := os.Getenv("SILERO_VAD_MODEL_PATH")
    if modelPath == "" {
        modelPath = filepath.Join("/opt/vad", "silero_vad.onnx")
    }

    cfg := vad.DefaultConfig()
    provider, err := vad.NewProvider(ctx, "silero", cfg,
        vad.WithModelPath(modelPath),
        vad.WithThreshold(0.5),
        vad.WithSampleRate(16000),
        vad.WithFrameSize(512),
        vad.WithMinSpeechDuration(200*time.Millisecond),
        vad.WithMaxSilenceDuration(500*time.Millisecond),
    )
    if err != nil {
        log.Fatalf("Failed to create VAD provider: %v", err)
    }

    audio := make([]byte, 1024)
    speech, err := provider.Process(ctx, audio)
    if err != nil {
        log.Fatalf("Processing failed: %v", err)
    }
    fmt.Printf("Speech detected: %v\n", speech)
}
```

## Usage

### ProcessStream for Real-Time Pipelines

Use `ProcessStream` when processing a live microphone stream on the edge. Feed audio chunks from a capture loop and consume `VADResult` values for downstream logic:

```go
audioCh := make(chan []byte, 16)
resultCh, err := provider.ProcessStream(ctx, audioCh)
if err != nil {
    log.Fatalf("Stream setup failed: %v", err)
}

go func() {
    for result := range resultCh {
        if result.Speech {
            // Forward speech frames to STT or processing pipeline
        }
    }
}()

// Feed audioCh from microphone capture loop
```

### Model Loading and Startup

The Silero provider loads the ONNX model on first use (lazy loading). To avoid first-request latency, warm up with a dummy `Process` call during initialization:

```go
// Warm up model during init
_, _ = provider.Process(ctx, make([]byte, 1024))
```

## Configuration Reference

| Option              | Description         | Default  | Edge Notes                     |
|---------------------|---------------------|----------|--------------------------------|
| `ModelPath`         | Path to ONNX model  | -        | Use local storage or /tmp      |
| `Threshold`         | Detection threshold  | 0.5      | 0.5-0.6 typical                |
| `FrameSize`         | Frame size (samples) | 512      | Smaller = less CPU, coarser    |
| `SampleRate`        | Sample rate (Hz)     | 16000    | Match input audio              |
| `MinSpeechDuration` | Min speech duration  | 250 ms   | Tune to reduce false triggers  |
| `MaxSilenceDuration`| Max silence duration | 500 ms   | Tune for turn-taking behavior  |

## Troubleshooting

### Model load failed or file not found

Verify the path, file permissions, and that the model file exists on the device. Use absolute paths. Check available disk space and memory.

### High CPU usage on edge

Reduce the effective frame rate (process every second frame), increase `FrameSize` slightly, or use a smaller Silero variant if available. Profile with `pprof` to identify bottlenecks.

### ONNX Runtime not found

Ensure ONNX Runtime libraries are installed or bundled for your target OS/architecture. Build with appropriate CGO tags.

## Advanced Topics

### Production Deployment

- Warm up the model during initialization to avoid first-request latency
- Monitor model load time, `Process` latency, and memory usage
- Plan for model updates (file replacement + restart) without breaking active sessions
- Use absolute model paths to avoid working directory issues on embedded systems

## Related Resources

- [WebRTC VAD in Browser](/integrations/webrtc-browser-vad)
- [Voice Services Overview](/integrations/voice-services)
