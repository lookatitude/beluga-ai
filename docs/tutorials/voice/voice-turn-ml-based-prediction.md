# ML-Based Turn Prediction with ONNX

**What you will build:** A turn detector using the **onnx** provider and an ONNX model (e.g. turn-detection model). You'll configure `ModelPath`, `Threshold`, and `MinSilenceDuration` for ML-driven turn detection.

## Learning Objectives

- Configure the ONNX turn-detection provider
- Set `ModelPath`, `Threshold`, and silence/turn-length options
- Use `DetectTurn` and `DetectTurnWithSilence` with the ONNX provider

## Prerequisites

- Go 1.24+
- Beluga AI (`go get github.com/lookatitude/beluga-ai`)
- An ONNX turn-detection model (or use a placeholder path for structure)

## Step 1: Configure the ONNX Provider
```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/turndetection"
)

func main() {
	ctx := context.Background()

	modelPath := os.Getenv("TURN_MODEL_PATH")
	if modelPath == "" {
		modelPath = filepath.Join(os.TempDir(), "turn_detection.onnx")
	}

	cfg := turndetection.DefaultConfig()
	detector, err := turndetection.NewProvider(ctx, "onnx", cfg,
		turndetection.WithModelPath(modelPath),
		turndetection.WithThreshold(0.5),
		turndetection.WithMinSilenceDuration(300*time.Millisecond),
		turndetection.WithMinTurnLength(5),
		turndetection.WithMaxTurnLength(10000),
	)
	if err != nil {
		log.Fatalf("create ONNX turn detector: %v", err)
	}

	audio := make([]byte, 2048) // 16 kHz mono, ~64 ms
	done, err := detector.DetectTurn(ctx, audio)
	if err != nil {
		log.Fatalf("detect turn: %v", err)
	}
	fmt.Printf("Turn detected: %v\n", done)
}
```

## Step 2: Use With Silence Duration
```text
go
go
	silence := 400 * time.Millisecond
	done, err := detector.DetectTurnWithSilence(ctx, audio, silence)
	if err != nil {
		log.Fatalf("detect turn: %v", err)
	}
	if done {
		fmt.Println("Turn end detected; ready for response.")
	}
```

## Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `ModelPath` | Path to ONNX model file | `turn_detection.onnx` |
| `Threshold` | Detection threshold (0–1) | `0.5` |
| `MinSilenceDuration` | Min silence to treat as turn end | `500ms` |
| `MinTurnLength` | Min turn length (samples/chars) | `10` |
| `MaxTurnLength` | Max turn length | `5000` |

## Verification
```bash
export TURN_MODEL_PATH=/path/to/your/turn_detection.onnx  # or leave unset for placeholder
go run .
```
# Expected: "Turn detected: false" or "Turn detected: true" depending on model and audio
```

## Next Steps

- **[Sentence-Boundary Detection](./voice-turn-sentence-boundary-detection.md)** — Heuristic provider.
- **[Heuristic Tuning](../integrations/voice/turn/heuristic-tuning.md)** — Fine-tune MinSilenceDuration and markers.
