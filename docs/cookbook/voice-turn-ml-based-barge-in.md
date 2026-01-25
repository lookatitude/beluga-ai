---
title: "ML-Based Barge-In with Turn Detection"
package: "voice/turndetection"
category: "voice"
complexity: "advanced"
---

# ML-Based Barge-In with Turn Detection

## Problem

You need to support barge-in (user interrupts the agent) using both VAD and turn detection, with the option to use an ONNX model for more accurate turn-end classification in noisy or varied environments.

## Solution

Combine `pkg/voice/vad` (speech onset) with `pkg/voice/turndetection` (turn context). Use the **onnx** provider when you have a turn-detection model; otherwise use **heuristic**. On speech onset during playback, call `DetectTurnWithSilence` to distinguish barge-in from end-of-turn, then stop TTS and switch to listening when barge-in is detected. This works because VAD gives you low-latency "user is speaking" and turn detection prevents false barge-in from end-of-turn edge cases.

## Code Example
```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/lookatitude/beluga-ai/pkg/voice/vad"
	"github.com/lookatitude/beluga-ai/pkg/voice/turndetection"
	turndetectioniface "github.com/lookatitude/beluga-ai/pkg/voice/turndetection/iface"
	vadiface "github.com/lookatitude/beluga-ai/pkg/voice/vad/iface"
)

var tracer = otel.Tracer("beluga.voice.bargein.recipe")

func main() {
	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "ml_bargein_setup")
	defer span.End()

	vadCfg := vad.DefaultConfig()
	vadProv, err := vad.NewProvider(ctx, "webrtc", vadCfg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(trace.StatusError, err.Error())
		log.Fatalf("vad: %v", err)
	}

	modelPath := os.Getenv("TURN_MODEL_PATH")
	if modelPath == "" {
		modelPath = filepath.Join(os.TempDir(), "turn_detection.onnx")
	}
	tdCfg := turndetection.DefaultConfig()
	td, err := turndetection.NewProvider(ctx, "onnx", tdCfg,
		turndetection.WithModelPath(modelPath),
		turndetection.WithThreshold(0.5),
		turndetection.WithMinSilenceDuration(250*time.Millisecond),
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(trace.StatusError, err.Error())
		log.Fatalf("turn detector: %v", err)
	}

	// Simulate barge-in check: VAD says speech, turn detector says not end-of-turn
	audio := make([]byte, 1024)
	speaking, _ := vadProv.Process(ctx, audio)
	silence := 100 * time.Millisecond
	done, _ := td.DetectTurnWithSilence(ctx, audio, silence)

	bargeIn := speaking && !done
	span.SetAttributes(
		attribute.Bool("vad.speaking", speaking),
		attribute.Bool("turn.done", done),
		attribute.Bool("barge_in", bargeIn),
	)
	fmt.Printf("Barge-in: %v (speaking=%v, turn_done=%v)\n", bargeIn, speaking, done)
}

func runBargeInLoop(ctx context.Context, vadProv vadiface.VADProvider, td turndetectioniface.TurnDetector, audio []byte, silence time.Duration) (bargeIn bool, err error) {
	speaking, err := vadProv.Process(ctx, audio)
	if err != nil {
		return false, err
	}
	if !speaking {
		return false, nil
	}
	done, err := td.DetectTurnWithSilence(ctx, audio, silence)
	if err != nil {
		return false, err
	}
	return !done, nil
}
```

## Explanation

1. **VAD for onset** — `vadProv.Process` returns whether the user is speaking. Use it during TTS playback to detect interruption.

2. **Turn detection for context** — `DetectTurnWithSilence` indicates end-of-turn. If the user is speaking (`speaking == true`) but we have **not** reached end-of-turn (`done == false`), we treat it as barge-in: user is interrupting mid-response.

3. **ONNX vs heuristic** — Use `onnx` when you have a model and want better accuracy; use `heuristic` for simpler, model-free setups. Both support `DetectTurnWithSilence`.

4. **OTEL** — Record `vad.speaking`, `turn.done`, and `barge_in` for debugging and tuning. Use `turndetection.InitMetrics` and VAD metrics in production.

```go
**Key insight:** Barge-in = "user started speaking" (VAD) and "we are not at end-of-turn" (turn detection). Turn detection avoids falsely treating end-of-turn as barge-in.

## Testing

```
Use `vad.NewAdvancedMockVADProvider` and `turndetection.NewAdvancedMockTurnDetector` from `test_utils` in `*_test.go` to unit-test barge-in logic. Or run manually:
```bash
export TURN_MODEL_PATH=/path/to/turn_detection.onnx  # or leave unset for default path
go run .
```

## Variations

### Heuristic-only barge-in

Skip ONNX and use heuristic turn detection:
```go
	td, err := turndetection.NewProvider(ctx, "heuristic", tdCfg,
		turndetection.WithMinSilenceDuration(200*time.Millisecond),
	)
```

### Stricter barge-in (fewer false positives)

Raise ONNX `Threshold` or heuristic `MinSilenceDuration` so that only clearer interruptions count as barge-in.

## Related Recipes

- **[Sentence-Boundary-Aware Turn Detection](./voice-turn-sentence-boundary-aware.md)** — Heuristic turn detection and sentence boundaries.
- **[Voice S2S Handling Speech Interruption](./voice-s2s-handling-speech-interruption.md)** — Interruption handling in S2S.
- **[Barge-In Detection Use Case](../use-cases/voice-turn-barge-in-detection.md)** — Full architecture and metrics.
