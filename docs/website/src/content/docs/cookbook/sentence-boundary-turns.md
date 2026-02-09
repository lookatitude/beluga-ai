---
title: "Sentence-Boundary Turn Detection"
description: "Detect turn completion using sentence boundaries, turn length, and silence for natural voice interaction."
---

## Problem

You need to detect when a user has finished speaking using sentence boundaries (`.`, `!`, `?`) and tunable turn length, so the agent responds at natural break points instead of mid-sentence or after long silence-only delays.

## Solution

Use the **heuristic** turn-detection provider with `WithSentenceEndMarkers`, `WithMinTurnLength`, and `WithMaxTurnLength`. Optionally use `DetectTurnWithSilence` when you have VAD-derived silence duration. The heuristic provider applies configurable rules without requiring an ML model.

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

	"github.com/lookatitude/beluga-ai/voice/turndetection"
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

1. **Sentence-end markers** -- `WithSentenceEndMarkers(".!?")` tells the heuristic provider to treat these runes as potential turn boundaries when combined with other checks (turn length, silence).

2. **Turn length** -- `WithMinTurnLength` and `WithMaxTurnLength` avoid treating very short fragments as complete turns and cap overly long segments. Adjust per domain (e.g. QA vs open-ended chat).

3. **Silence** -- `DetectTurnWithSilence` uses VAD- or STT-derived silence. When silence exceeds `MinSilenceDuration`, the provider signals turn end; combine with sentence boundaries for better accuracy.

**Key insight:** Combine sentence-boundary rules with silence and turn-length limits for responsive, natural turn detection without an ONNX model.

## Variations

### Stricter Sentence Ends

Add semicolons or other markers:

```go
turndetection.WithSentenceEndMarkers(".!?;"),
```

### Longer Turns (Storytelling)

For longer-form input, increase limits:

```go
turndetection.WithMaxTurnLength(12000),
turndetection.WithMinSilenceDuration(600*time.Millisecond),
```

## Related Recipes

- **[ML-Based Barge-In](./ml-barge-in)** -- ONNX-based turn detection for barge-in
- **[Handling Speech Interruption](./speech-interruption)** -- Interruption and barge-in behavior
- **[Handling Long Utterances](./long-utterances)** -- Chunking for long user input
