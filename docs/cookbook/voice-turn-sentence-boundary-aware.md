---
title: "Sentence-Boundary-Aware Turn Detection"
package: "voice/turndetection"
category: "voice"
complexity: "intermediate"
---

# Sentence-Boundary-Aware Turn Detection

## Problem

You need to detect when a user has finished speaking in a voice pipeline using sentence boundaries (e.g. `.`, `!`, `?`) and tunable turn length, so the agent responds at natural break points instead of mid-sentence or after long silence-only delays.

## Solution

Use the **heuristic** turn-detection provider from `pkg/voice/turndetection` with `WithSentenceEndMarkers`, `WithMinTurnLength`, and `WithMaxTurnLength`. Optionally use `DetectTurnWithSilence` when you have VAD-derived silence. This works because the heuristic provider applies configurable rules (sentence ends, turn length) to decide turn completion without requiring an ML model.

## Code Example
```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/lookatitude/beluga-ai/pkg/voice/turndetection"
)

var tracer = otel.Tracer("beluga.voice.turndetection.recipe")

func main() {
	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "sentence_boundary_turn_detection")
	defer span.End()

	cfg := turndetection.DefaultConfig()
	detector, err := turndetection.NewProvider(ctx, "heuristic", cfg,
		turndetection.WithSentenceEndMarkers(".!?"),
		turndetection.WithMinTurnLength(8),
		turndetection.WithMaxTurnLength(4000),
		turndetection.WithMinSilenceDuration(400*time.Millisecond),
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(trace.StatusError, err.Error())
		log.Fatalf("create detector: %v", err)
	}

	audio := make([]byte, 1024)
	silence := 450 * time.Millisecond
	done, err := detector.DetectTurnWithSilence(ctx, audio, silence)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(trace.StatusError, err.Error())
		log.Fatalf("detect: %v", err)
	}
	span.SetAttributes(attribute.Bool("turn.detected", done))
	fmt.Printf("Turn detected: %v\n", done)
}
```

## Explanation

1. **Sentence-end markers** — `WithSentenceEndMarkers(".!?")` tells the heuristic provider to treat these runes as potential turn boundaries when combined with other checks (e.g. turn length, silence).

2. **Turn length** — `WithMinTurnLength` and `WithMaxTurnLength` avoid treating very short fragments as complete turns and cap overly long segments. Adjust per domain (e.g. QA vs open-ended chat).

3. **Silence** — `DetectTurnWithSilence` uses VAD- or STT-derived silence. When silence exceeds `MinSilenceDuration`, the provider can signal turn end; use together with sentence boundaries for better accuracy.

4. **OTEL** — The example records errors and a `turn.detected` attribute. Use `turndetection.InitMetrics(meter, tracer)` at startup for full metrics.

**Key insight:** Combine sentence-boundary rules with silence and turn-length limits so you get responsive, natural turn detection without an ONNX model.

## Testing
```go
func TestSentenceBoundaryTurnDetection(t *testing.T) {
	ctx := context.Background()
	cfg := turndetection.DefaultConfig()
	detector, err := turndetection.NewProvider(ctx, "heuristic", cfg,
		turndetection.WithMinSilenceDuration(300*time.Millisecond),
		turndetection.WithSentenceEndMarkers(".!?"),
	)
	require.NoError(t, err)

	_, err = detector.DetectTurn(ctx, []byte{0, 1, 2})
	assert.NoError(t, err)

	done, err := detector.DetectTurnWithSilence(ctx, []byte{0, 1, 2}, 400*time.Millisecond)
	require.NoError(t, err)
	assert.True(t, done)
}
```
Or run manually:
```bash
go run .
```
# Expected: "Turn detected: true" when silence >= MinSilenceDuration

## Variations

### Stricter sentence ends

Add `;` or other markers:
	turndetection.WithSentenceEndMarkers(".!?;"),

### Longer turns (e.g. storytelling)
	turndetection.WithMaxTurnLength(12000),
	turndetection.WithMinSilenceDuration(600*time.Millisecond),

## Related Recipes

- **[ML-Based Barge-In](./voice-turn-ml-based-barge-in.md)** — Use ONNX and turn detection for barge-in.
- **[Heuristic Tuning](../integrations/voice/turn/heuristic-tuning.md)** — Fine-tune MinSilenceDuration and markers.
- **[Voice S2S Handling Speech Interruption](./voice-s2s-handling-speech-interruption.md)** — Interruption and barge-in behavior.
