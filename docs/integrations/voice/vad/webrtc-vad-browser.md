# WebRTC VAD in Browser

Welcome, colleague! In this guide we'll use **WebRTC VAD** with Beluga AI's `pkg/voice/vad` in a browser-oriented architecture. The `webrtc` provider uses WebRTC's built-in VAD logic; you'll run Beluga in a backend and stream audio from the browser, or use a Go-based WebRTC stack that embeds the same VAD logic.

## What you will build

You will configure the **webrtc** VAD provider, use `Process` or `ProcessStream` on audio (e.g. from browser WebRTC or uploaded chunks), and optionally expose a small API that the front end calls with audio. This allows browser clients to benefit from consistent VAD logic running in your Go backend.

## Learning Objectives

- ✅ Configure the webrtc VAD provider
- ✅ Use `WithThreshold`, `WithSampleRate`, and `WithFrameSize`
- ✅ Process audio from browser (e.g. WebSocket, HTTP upload)
- ✅ Return VAD decisions (speech vs non-speech) to the client or pipeline

## Prerequisites

- Go 1.24 or later
- Beluga AI (`go get github.com/lookatitude/beluga-ai`)
- Browser client sending audio (e.g. WebSocket, fetch) or server-side WebRTC

## Step 1: Setup and Installation
bash
```bash
go get github.com/lookatitude/beluga-ai
```

## Step 2: Create WebRTC VAD Provider
```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lookatitude/beluga-ai/pkg/voice/vad"
	_ "github.com/lookatitude/beluga-ai/pkg/voice/vad/providers/webrtc"
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
		log.Fatalf("vad: %v", err)
	}

	audio := make([]byte, 1024)
	speech, err := provider.Process(ctx, audio)
	if err != nil {
		log.Fatalf("process: %v", err)
	}
	fmt.Printf("Speech: %v\n", speech)
}
```

## Step 3: Accept Browser Audio

Expose an HTTP or WebSocket endpoint that receives raw audio (e.g. PCM 16-bit mono, 16 kHz). Decode chunks and pass to `provider.Process` or `ProcessStream`. Return `{"speech": true/false}` or stream results back.

## Step 4: ProcessStream for Real-Time

For streaming from the browser:
```text
go
go
	audioCh := make(chan []byte, 16)
	resultCh, _ := provider.ProcessStream(ctx, audioCh)
	go func() {
		for r := range resultCh {
			// send r.Speech to client via WebSocket or SSE
		}
	}()
	// Feed audioCh from WebSocket or HTTP chunked upload
```

## Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `Threshold` | Speech detection threshold (0–1) | 0.5 |
| `SampleRate` | Audio sample rate | 16000 |
| `FrameSize` | Frame size in samples | 512 |
| `MinSpeechDuration` | Min speech duration | 250 ms |
| `MaxSilenceDuration` | Max silence duration | 500 ms |

## Common Issues

### "Browser audio format mismatch"

**Problem**: Client sends different sample rate or channels.

**Solution**: Resample to 16 kHz mono (or match `SampleRate`) before passing to VAD. Document expected format (e.g. PCM 16-bit, 16 kHz, mono).

### "High latency"

**Problem**: Large buffers or slow round-trips.

**Solution**: Use smaller chunks and `ProcessStream`; keep WebSocket or HTTP flow tight. Consider running VAD in a worker to avoid blocking.

### "Provider 'webrtc' not found"

**Problem**: Webrtc provider not registered.

**Solution**: Import `_ "github.com/lookatitude/beluga-ai/pkg/voice/vad/providers/webrtc"` to register the provider.

## Production Considerations

- **CORS and auth**: Secure your audio endpoint; validate origin and tokens.
- **Rate limiting**: Limit requests per client to avoid abuse.
- **Monitoring**: Log and metric request volume, latency, and errors.

## Next Steps

- **[ONNX Runtime for Edge VAD](./onnx-runtime-edge-vad.md)** — Silero on edge.
- **[Custom VAD with Silero](../../../../tutorials/voice/voice-vad-custom-silero.md)** — Silero provider.
- **[VAD Sensitivity Profiles](../../../../cookbook/voice-vad-sensitivity-profiles.md)** — Tuning.
