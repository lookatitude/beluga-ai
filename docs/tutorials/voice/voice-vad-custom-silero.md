# Custom VAD Models with Silero

**What you will build:** A VAD provider using the **silero** implementation in `pkg/voice/vad` with a custom ONNX model path. You'll set `ModelPath`, `Threshold`, and `SampleRate` for Silero-based voice activity detection.

## Learning Objectives

- Configure the Silero VAD provider with a custom ONNX model
- Use `WithModelPath`, `WithThreshold`, and `WithSampleRate`
- Call `Process` and `ProcessStream` for voice activity detection
- Integrate VAD with a voice session or pipeline

## Prerequisites

- Go 1.24+
- Beluga AI (`go get github.com/lookatitude/beluga-ai`)
- Silero VAD ONNX model (or use default if available)
- [Voice Sensitivity Tuning](./voice-sensitivity-tuning.md) (recommended)

## Step 1: Create a Silero VAD Provider
```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/lookatitude/beluga-ai/pkg/voice/vad"
	_ "github.com/lookatitude/beluga-ai/pkg/voice/vad/providers/silero"
)

func main() {
	ctx := context.Background()

	modelPath := os.Getenv("SILERO_VAD_MODEL_PATH")
	if modelPath == "" {
		modelPath = filepath.Join(os.TempDir(), "silero_vad.onnx")
	}

	cfg := vad.DefaultConfig()
	provider, err := vad.NewProvider(ctx, "silero", cfg,
		vad.WithModelPath(modelPath),
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
	fmt.Printf("Speech detected: %v\n", speech)
}
```

## Step 2: Use ProcessStream for Real-Time Audio

For streaming pipelines, use `ProcessStream` with an audio channel:
```text
go
go
	audioCh := make(chan []byte, 8)
	resultCh, err := provider.ProcessStream(ctx, audioCh)
	if err != nil {
		log.Fatalf("process stream: %v", err)
	}
	go func() {
		for r := range resultCh {
			fmt.Printf("Speech: %v\n", r.Speech)
		}
	}()
	// Send audio chunks to audioCh, then close(audioCh)
```

## Step 3: Tune Threshold and Duration

- **Threshold**: Raise (e.g. 0.6) to reduce false positives in noise; lower (e.g. 0.4) for higher sensitivity.
- **MinSpeechDuration** / **MaxSilenceDuration**: Use `WithMinSpeechDuration` and `WithMaxSilenceDuration` to filter brief noise or merge short speech segments.

## Verification
```bash
export SILERO_VAD_MODEL_PATH=/path/to/silero_vad.onnx  # optional
go run .
```
# Expected: "Speech detected: false" or "true" depending on audio
```

## Next Steps

- **[Noise-Resistant VAD](../../use-cases/voice-vad-noise-resistant.md)** — Use case for noisy environments.
- **[WebRTC VAD in Browser](../../integrations/voice/vad/webrtc-vad-browser.md)** — Browser-based VAD.
- **[VAD Sensitivity Profiles](../../cookbook/voice-vad-sensitivity-profiles.md)** — Recipe for tuning.
