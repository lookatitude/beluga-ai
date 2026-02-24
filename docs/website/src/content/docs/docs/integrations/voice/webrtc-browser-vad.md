---
title: WebRTC Browser VAD
description: "Use lightweight WebRTC-based voice activity detection with browser audio streaming in Beluga AI for consistent speech detection."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "WebRTC VAD, browser voice detection, Beluga AI, speech detection, DSP algorithm, voice pipeline Go, client-side audio"
---

Browser-based applications need a fast, deterministic way to distinguish speech from silence before sending audio upstream. The WebRTC VAD provider uses traditional DSP algorithms that consume minimal CPU and produce consistent results across all clients, making it the right choice when you need predictable behavior without the overhead of a neural network model. Audio is streamed from the browser to a Go backend where VAD decisions are computed and returned.

## Overview

The `voice/vad` package's `webrtc` provider uses WebRTC-compatible VAD logic running in your Go backend. Browser clients send audio (via WebSocket or HTTP) and receive speech/non-speech decisions, ensuring consistent VAD behavior across all clients.

## Prerequisites

- Go 1.23 or later
- Browser client capable of sending audio (WebSocket, HTTP upload)
- Audio format: PCM 16-bit mono, 16 kHz

## Installation

```bash
go get github.com/lookatitude/beluga-ai
```

## Configuration

Create a WebRTC VAD provider:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/voice/vad"
    _ "github.com/lookatitude/beluga-ai/voice/vad/providers/webrtc"
)

func main() {
    ctx := context.Background()

    cfg := vad.DefaultConfig()
    provider, err := vad.NewProvider(ctx, "webrtc", cfg,
        vad.WithThreshold(0.5),
        vad.WithSampleRate(16000),
        vad.WithFrameSize(512),
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

### Accept Browser Audio

Expose an HTTP or WebSocket endpoint that receives raw audio from the browser. Decode chunks and pass them to `provider.Process` or `ProcessStream`. Return `{"speech": true}` or `{"speech": false}` to the client.

### ProcessStream for Real-Time

For continuous streaming from the browser:

```go
audioCh := make(chan []byte, 16)
resultCh, err := provider.ProcessStream(ctx, audioCh)
if err != nil {
    log.Fatalf("Stream setup failed: %v", err)
}

go func() {
    for r := range resultCh {
        // Send r.Speech to client via WebSocket or SSE
    }
}()
// Feed audioCh from WebSocket or HTTP chunked upload
```

## Configuration Reference

| Option              | Description                 | Default |
|---------------------|-----------------------------|---------|
| `Threshold`         | Speech detection threshold  | 0.5     |
| `SampleRate`        | Audio sample rate (Hz)      | 16000   |
| `FrameSize`         | Frame size in samples       | 512     |
| `MinSpeechDuration` | Minimum speech duration     | 250 ms  |
| `MaxSilenceDuration`| Maximum silence duration    | 500 ms  |

## Troubleshooting

### Browser audio format mismatch

Resample to 16 kHz mono (or match the configured `SampleRate`) before passing audio to VAD. Document the expected format (PCM 16-bit, 16 kHz, mono) for client developers.

### High latency

Use smaller audio chunks and `ProcessStream`. Keep WebSocket or HTTP round-trips tight. Consider running VAD in a dedicated goroutine to avoid blocking request handlers.

### Provider 'webrtc' not found

Import the WebRTC provider to trigger registration:

```go
import _ "github.com/lookatitude/beluga-ai/voice/vad/providers/webrtc"
```

## Advanced Topics

### Production Deployment

- **CORS and authentication**: Secure your audio endpoint; validate origin and tokens
- **Rate limiting**: Limit requests per client to prevent abuse
- **Monitoring**: Log and track request volume, latency, and error rates with OpenTelemetry

## Related Resources

- [ONNX Runtime Edge VAD](/integrations/onnx-edge-vad)
- [Voice Services Overview](/integrations/voice-services)
