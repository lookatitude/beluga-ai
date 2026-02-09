---
title: "ML-Based Barge-In Detection"
description: "Combine VAD and ONNX turn detection to distinguish user interruptions from end-of-turn in voice pipelines."
---

## Problem

You need to support barge-in (user interrupts the agent) using both VAD and turn detection, with the option to use an ONNX model for more accurate turn-end classification in noisy or varied environments.

## Solution

Combine VAD (speech onset) with turn detection (turn context). Use the **onnx** provider when you have a turn-detection model; otherwise use **heuristic**. On speech onset during playback, call `DetectTurnWithSilence` to distinguish barge-in from end-of-turn, then stop TTS and switch to listening when barge-in is detected.

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

	"github.com/lookatitude/beluga-ai/voice/turndetection"
	turndetectioniface "github.com/lookatitude/beluga-ai/voice/turndetection/iface"
	"github.com/lookatitude/beluga-ai/voice/vad"
	vadiface "github.com/lookatitude/beluga-ai/voice/vad/iface"
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

	// Barge-in check: VAD says speech, turn detector says not end-of-turn
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

func runBargeInLoop(ctx context.Context, vadProv vadiface.VADProvider, td turndetectioniface.TurnDetector, audio []byte, silence time.Duration) (bool, error) {
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

1. **VAD for onset** -- `vadProv.Process` returns whether the user is speaking. Use it during TTS playback to detect interruption with low latency.

2. **Turn detection for context** -- `DetectTurnWithSilence` indicates end-of-turn. If the user is speaking (`speaking == true`) but the turn is **not** done (`done == false`), treat it as barge-in: user is interrupting mid-response.

3. **ONNX vs heuristic** -- Use `onnx` when you have a model and want better accuracy; use `heuristic` for simpler, model-free setups. Both support `DetectTurnWithSilence`.

**Key insight:** Barge-in = "user started speaking" (VAD) and "we are not at end-of-turn" (turn detection). Turn detection avoids falsely treating end-of-turn as barge-in.

## Variations

### Heuristic-Only Barge-In

Skip ONNX and use heuristic turn detection for simpler deployments:

```go
td, err := turndetection.NewProvider(ctx, "heuristic", tdCfg,
    turndetection.WithMinSilenceDuration(200*time.Millisecond),
)
```

### Stricter Barge-In

Raise ONNX `Threshold` or heuristic `MinSilenceDuration` so only clearer interruptions count as barge-in, reducing false positives.

## Related Recipes

- **[Sentence-Boundary Turns](./sentence-boundary-turns)** -- Heuristic turn detection with sentence boundaries
- **[Handling Speech Interruption](./speech-interruption)** -- Interruption handling in S2S
- **[VAD Sensitivity Profiles](./vad-sensitivity)** -- Tune VAD for different environments
