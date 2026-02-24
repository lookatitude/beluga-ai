---
title: "ML-Based Barge-In Detection"
description: "Recipe for combining VAD and ONNX turn detection in Go to accurately distinguish user interruptions from end-of-turn in voice AI pipelines."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, barge-in detection, Go ONNX VAD, turn detection, speech interruption, voice pipeline, ML audio classification"
---

## Problem

You need to support barge-in (user interrupts the agent) using both VAD and turn detection, with the option to use an ONNX model for more accurate turn-end classification in noisy or varied environments. Simple VAD-only barge-in systems trigger on any speech onset, which causes false positives when the user naturally pauses mid-utterance or speaks briefly at turn boundaries. This results in the agent prematurely stopping playback and starting transcription, leading to awkward interruptions and degraded user experience.

The challenge is distinguishing genuine interruptions (barge-in) from natural turn-taking cues (end-of-turn). Voice activity detection alone cannot make this distinction because it operates on acoustic features (energy, pitch) without understanding conversational context. Turn detection adds context by analyzing speech patterns and silence durations to predict when a speaker has finished their turn. Combining both signals allows accurate barge-in detection: speech onset (VAD) confirms the user is speaking, while turn detection confirms they are not simply finishing their previous utterance.

## Solution

Combine VAD (speech onset) with turn detection (turn context). Use the **onnx** provider when you have a turn-detection model; otherwise use **heuristic**. On speech onset during playback, call `DetectTurnWithSilence` to distinguish barge-in from end-of-turn, then stop TTS and switch to listening when barge-in is detected.

The reasoning behind this dual-signal approach is defense against false positives. VAD provides low-latency speech onset detection (10-50ms), which is critical for responsive barge-in. However, VAD is context-free and triggers on any speech, including the user's own turn-ending phrases ("that's all", "thank you"). Turn detection provides context-aware classification, using features like silence duration, pitch trajectory, and learned patterns from training data. The ONNX provider uses a pre-trained neural model for higher accuracy, while the heuristic provider uses rule-based logic (silence thresholds, sentence boundaries) for simpler deployments.

This pattern follows Beluga's FrameProcessor interface for composability. VAD and turn detection are separate processors that can be swapped, tuned, or tested independently. The application combines their outputs in a decision function (`bargeIn = speaking && !done`), which can be customized per use case (e.g., require multiple consecutive frames of barge-in to reduce flapping).

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

1. **VAD for onset** -- `vadProv.Process` returns whether the user is speaking. Use it during TTS playback to detect interruption with low latency. VAD operates on short frames (10-30ms), providing near-instant feedback when speech energy rises above threshold. This low latency is critical for responsive barge-in: delays longer than 200ms feel unnatural to users. However, VAD alone is insufficient because it cannot distinguish between "user started new speech" (barge-in) and "user is finishing previous speech" (turn continuation). This is where turn detection provides context.

2. **Turn detection for context** -- `DetectTurnWithSilence` indicates end-of-turn. If the user is speaking (`speaking == true`) but the turn is **not** done (`done == false`), treat it as barge-in: user is interrupting mid-response. Turn detection analyzes temporal patterns like silence duration and pitch contours to classify whether the speaker intends to yield the floor. The `silence` parameter tunes sensitivity: shorter values (100-200ms) detect barge-in faster but risk false positives; longer values (300-500ms) reduce false positives but increase latency.

3. **ONNX vs heuristic** -- Use `onnx` when you have a model and want better accuracy; use `heuristic` for simpler, model-free setups. Both support `DetectTurnWithSilence`. The ONNX provider loads a pre-trained neural network (e.g., CNN or Transformer-based) trained on conversational datasets to predict turn boundaries. This provides higher accuracy in noisy or multi-speaker environments where heuristics struggle. The heuristic provider uses rule-based logic: if silence exceeds `MinSilenceDuration` and the last audio frame meets certain criteria (low pitch variance, decaying energy), predict end-of-turn. Heuristics work well for controlled environments (quiet rooms, single speaker) and avoid the complexity of model deployment.

**Key insight:** Barge-in = "user started speaking" (VAD) and "we are not at end-of-turn" (turn detection). Turn detection avoids falsely treating end-of-turn as barge-in. This two-signal approach is standard in commercial voice assistants because it balances responsiveness (VAD) with accuracy (turn detection). The pattern here uses Beluga's registry pattern for swapping turn detection providers: change `"onnx"` to `"heuristic"` in `NewProvider` and the rest of the code remains unchanged. This makes it easy to A/B test different providers or fall back to heuristics when the ONNX model is unavailable.

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
