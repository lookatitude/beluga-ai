# ONNX Runtime for Edge VAD

Welcome, colleague! In this guide we'll run **Silero VAD** with ONNX Runtime in edge or resource-constrained environments. You'll use `pkg/voice/vad` with the **silero** provider, point `ModelPath` to a Silero ONNX model, and tune for low CPU/memory.

## What you will build

You will configure the Silero VAD provider with an ONNX model path, run it on edge devices (e.g. Raspberry Pi, embedded), and optionally use `ProcessStream` for real-time pipelines. You'll tune `Threshold`, `FrameSize`, and `SampleRate` for accuracy and resource use.

## Learning Objectives

- ✅ Configure Silero VAD with `WithModelPath` and ONNX model
- ✅ Tune `Threshold`, `FrameSize`, and `SampleRate` for edge
- ✅ Use `Process` or `ProcessStream` in low-resource environments
- ✅ Understand model placement and startup behavior

## Prerequisites

- Go 1.24 or later
- Beluga AI (`go get github.com/lookatitude/beluga-ai`)
- Silero VAD ONNX model (e.g. `silero_vad.onnx`)
- ONNX Runtime available (Beluga's Silero provider uses it)

## Step 1: Setup and Installation
bash
```bash
go get github.com/lookatitude/beluga-ai
```

Ensure the ONNX model is available on the edge device (e.g. bundled binary, download at startup, or read-only storage).

## Step 2: Configure Silero for Edge

Use a smaller frame size and conservative threshold to balance CPU and accuracy:
```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/vad"
	_ "github.com/lookatitude/beluga-ai/pkg/voice/vad/providers/silero"
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

## Step 3: Model Loading and Startup

The Silero provider loads the ONNX model at first use. Ensure `ModelPath` is readable and that the device has enough memory for the model. Lazy loading can cause a short delay on the first `Process` call.

## Step 4: ProcessStream for Real-Time Edge Pipelines

Use `ProcessStream` when processing a live mic stream on the edge. Feed audio chunks (e.g. from a capture loop) into the channel and consume `VADResult` for downstream logic.

## Configuration Options

| Option | Description | Default | Edge Notes |
|--------|-------------|---------|------------|
| `ModelPath` | Path to ONNX model | — | Use local storage or tmp |
| `Threshold` | Detection threshold | 0.5 | 0.5–0.6 typical |
| `FrameSize` | Frame size | 512 | Smaller = less CPU, coarser |
| `SampleRate` | Sample rate | 16000 | Match input |
| `MinSpeechDuration` | Min speech | 250 ms | Tune for false triggers |
| `MaxSilenceDuration` | Max silence | 500 ms | Tune for turn-taking |

## Common Issues

### "Model load failed" or "file not found"

**Problem**: `ModelPath` wrong or not readable on edge.

**Solution**: Verify path, permissions, and that the model file exists. Use absolute paths if needed. Check available disk/memory.

### "High CPU usage on edge"

**Problem**: Model or frame rate too heavy.

**Solution**: Reduce effective frame rate (e.g. process every second frame), increase `FrameSize` slightly, or use a smaller Silero variant if available. Profile with pprof.

### "ONNX Runtime not found"

**Problem**: CGO or ONNX dependency missing on target.

**Solution**: Build with appropriate tags for your OS/arch. Ensure ONNX Runtime libs are installed or bundled per Beluga/ONNX docs.

## Production Considerations

- **Startup**: Model load on first use; warm up with a dummy `Process` call during init if you want to avoid first-request latency.
- **Monitoring**: Metric model load time, `Process` latency, and OOM risk on edge.
- **Updates**: Plan for model updates (replace file, restart) without breaking running sessions.

## Next Steps

- **[WebRTC VAD in Browser](./webrtc-vad-browser.md)** — Browser-oriented VAD.
- **[Custom VAD with Silero](../../../tutorials/voice/voice-vad-custom-silero.md)** — Silero tutorial.
- **[Noise-Resistant VAD](../../../use-cases/voice-vad-noise-resistant.md)** — Noisy environments.
