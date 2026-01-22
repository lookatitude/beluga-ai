# Custom Turn Detectors for Noisy Environments

Welcome, colleague! In this integration guide, we'll tune Beluga AI's `pkg/voice/turndetection` for noisy environments (e.g. contact centers, retail, factory floors). You'll use both heuristic and ONNX providers with adjusted thresholds and silence settings so turn detection stays reliable despite background noise.

## What you will build

You will create a turn-detection setup that works in noisy environments by tuning `MinSilenceDuration`, `Threshold`, sentence-end markers, and optional ONNX model path. This integration allows you to reduce false turn-end triggers and missed turn-end events when background noise or overlapping speech is present.

## Learning Objectives

- ✅ Configure heuristic and ONNX turn detectors for noisy settings
- ✅ Tune `MinSilenceDuration`, `Threshold`, and turn-length limits
- ✅ Use `DetectTurnWithSilence` in a streaming pipeline
- ✅ Understand configuration trade-offs and verification

## Prerequisites

- Go 1.24 or later
- Beluga AI (`go get github.com/lookatitude/beluga-ai`)
- (Optional) ONNX turn-detection model for `onnx` provider

## Step 1: Setup and Installation
bash
```bash
go get github.com/lookatitude/beluga-ai
```

Ensure you have a voice pipeline (e.g. VAD, STT) that can supply audio chunks and silence duration. See [Voice Sessions](../../../../use-cases/voice-sessions.md) and [Low-Latency Turn Prediction](../../../../use-cases/voice-turn-low-latency-prediction.md).

## Step 2: Heuristic Provider for Noisy Environments

Increase `MinSilenceDuration` so brief noise gaps are not treated as end-of-turn. Use `WithMinTurnLength` to avoid very short spurious turns.
```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/turndetection"
)

func main() {
	ctx := context.Background()

	cfg := turndetection.DefaultConfig()
	detector, err := turndetection.NewProvider(ctx, "heuristic", cfg,
		turndetection.WithMinSilenceDuration(700*time.Millisecond),
		turndetection.WithMinTurnLength(20),
		turndetection.WithMaxTurnLength(8000),
		turndetection.WithSentenceEndMarkers(".!?"),
	)
	if err != nil {
		log.Fatalf("create detector: %v", err)
	}

	audio := make([]byte, 2048)
	silence := 800 * time.Millisecond
	done, err := detector.DetectTurnWithSilence(ctx, audio, silence)
	if err != nil {
		log.Fatalf("detect: %v", err)
	}
	fmt.Printf("Turn detected: %v\n", done)
}
```

### Verification
```bash
go run .
```
# Expected: "Turn detected: true" when silence >= 700 ms
```

## Step 3: ONNX Provider with Higher Threshold

For the ONNX provider, raise `Threshold` to require stronger model confidence before declaring end-of-turn. This reduces false positives in noise.
go
```go
	detector, err := turndetection.NewProvider(ctx, "onnx", cfg,
		turndetection.WithModelPath(os.Getenv("TURN_MODEL_PATH")),
		turndetection.WithThreshold(0.6),
		turndetection.WithMinSilenceDuration(600*time.Millisecond),
		turndetection.WithMinTurnLength(15),
	)
```

### Verification

Run with real or synthetic noisy audio. Log turn-end events and compare to ground truth; tune `Threshold` and `MinSilenceDuration` until false positive rate is acceptable.

## Configuration Options

| Option | Description | Default | Noisy Env Suggestion |
|--------|-------------|---------|----------------------|
| `MinSilenceDuration` | Min silence to treat as turn end | 500 ms | 600–800 ms |
| `Threshold` | ONNX detection threshold (0–1) | 0.5 | 0.55–0.65 |
| `MinTurnLength` | Min turn length | 10 | 15–25 |
| `MaxTurnLength` | Max turn length | 5000 | 8000+ |
| `SentenceEndMarkers` | Heuristic sentence ends | `.!?` | Keep or extend |

## Common Issues

### "Too many false turn-end events in noise"

**Problem**: Brief noise gaps or overlapping speech trigger turn end.

**Solution**: Increase `MinSilenceDuration` (e.g. 600–800 ms) and, for ONNX, `Threshold` (e.g. 0.55–0.65). Prefer `DetectTurnWithSilence` fed by a robust VAD so silence is computed from actual speech absence.

### "Missed end-of-turn in noisy segments"

**Problem**: User stopped speaking but turn detector rarely fires.

**Solution**: Ensure VAD correctly marks silence; avoid over-incrementing `MinSilenceDuration`. Consider ONNX if heuristic is insufficient. Verify audio format (sample rate, chunk size) matches provider expectations.

### "provider 'onnx' not registered" or model load errors

**Problem**: ONNX provider not linked or model path invalid.

**Solution**: Import `_ "github.com/lookatitude/beluga-ai/pkg/voice/turndetection/providers/onnx"` to register the provider. Set `TURN_MODEL_PATH` to a valid ONNX model file.

## Production Considerations

- **Error handling**: Check `turndetection.IsRetryableError(err)` and retry where appropriate.
- **Monitoring**: Use `turndetection.InitMetrics(meter, tracer)` and OTEL to track latency and turn-end rates.
- **A/B testing**: Compare heuristic vs ONNX and different thresholds using metrics before rollout.

## Next Steps

- **[Heuristic Tuning](./heuristic-tuning.md)** — Fine-tune MinSilenceDuration, sentence-end markers, and question markers.
- **[Low-Latency Turn Prediction](../../../../use-cases/voice-turn-low-latency-prediction.md)** — Use case and architecture.
- **[Voice Sessions](../../../../use-cases/voice-sessions.md)** — End-to-end voice pipeline.
