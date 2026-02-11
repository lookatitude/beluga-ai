---
title: "Sentence-Boundary Turn Detection"
description: "Detect turn completion using sentence boundaries, turn length, and silence for natural voice interaction."
---

## Problem

Determining when a user has finished speaking is fundamental to responsive voice interaction but deceptively difficult. Turn detection that relies solely on silence waits for long pauses (500-1000ms) before responding, making conversations feel sluggish. Turn detection that triggers too early interrupts users mid-sentence, cutting off their thoughts and requiring them to repeat themselves. The challenge is particularly acute with natural speech patterns: users pause briefly between phrases, trail off without clear endings, or speak in fragments rather than complete sentences. A system that waits for grammatically complete sentences may never respond; a system that triggers on every pause fragments coherent multi-sentence thoughts.

Different conversation styles complicate this further. Some users speak in short, clipped phrases with minimal silence. Others deliver long, flowing narratives with frequent pauses for breath or thought. A fixed threshold that works for one style fails for the other. You need turn detection that adapts to content structure, not just acoustic properties.

## Solution

The heuristic turn detection provider combines multiple signals to identify natural turn boundaries: sentence-end markers (`.`, `!`, `?`) indicate grammatical completion, minimum turn length prevents treating short fragments as complete turns, maximum turn length ensures long utterances eventually trigger processing, and silence duration provides acoustic confirmation of turn completion. These rules compose to create responsive yet accurate detection. A user's speech that ends with a sentence marker, exceeds the minimum length, and is followed by sufficient silence clearly indicates turn completion. A long utterance that exceeds maximum length triggers processing even without a sentence marker, ensuring the system eventually responds to lengthy monologues.

This rule-based approach avoids the overhead and complexity of ML-based turn detection while providing tunable behavior for different use cases. Short minimum lengths and brief silence work for command-and-control scenarios where users speak in terse phrases. Longer minimums and extended silence suit open-ended conversations where users express complex thoughts. The heuristic provider enables configuration without model training or ONNX runtime overhead.

Combining STT-derived text (for sentence markers and length) with VAD-derived silence (for acoustic confirmation) provides more reliable detection than either signal alone. Sentence boundaries without silence fail when users pause mid-sentence (e.g., "I want to... actually, never mind"). Silence without sentence boundaries fails when users trail off mid-thought or network issues cause gaps in audio delivery.

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

The code demonstrates provider creation using functional options, a standard Beluga pattern for configurable components. The provider is registered via the `turndetection` package's registry, instantiated through `NewProvider`, and configured using `WithX()` options.

## Explanation

1. **Sentence-end markers** -- The heuristic provider scans STT transcripts for the specified runes (`.`, `!`, `?`), treating them as potential turn boundaries when combined with other conditions. A transcript ending with a sentence marker has higher probability of being complete than one ending mid-word. The provider does not trigger on sentence markers alone because users often speak multi-sentence thoughts ("I need to book a flight. I want to leave tomorrow.") that should be processed together. Markers provide evidence of natural break points but require confirmation from other signals (turn length, silence) to avoid premature triggering. This is why sentence boundaries are necessary but not sufficient conditions for turn detection.

2. **Turn length** -- `WithMinTurnLength` sets a character count threshold that must be exceeded before turn detection activates, preventing very short utterances like "um" or "yes" from triggering prematurely when they are sentence fragments rather than complete thoughts. `WithMaxTurnLength` caps the maximum characters in a single turn, ensuring long-winded speakers eventually trigger processing even if they never reach a sentence boundary or silence threshold. These bounds adapt to use case: question-answering systems use short minimums (8-10 characters) and moderate maximums (1000-2000 characters) for concise exchanges, while storytelling or transcription use cases employ longer minimums (20-30 characters) and very high maximums (10000+ characters) to avoid fragmenting narratives.

3. **Silence** -- `DetectTurnWithSilence` accepts a VAD-derived silence duration representing how long the user has been silent. When silence exceeds `MinSilenceDuration` and other conditions are met (sentence boundary, sufficient length), the provider signals turn completion. Silence provides acoustic confirmation that the user has stopped speaking, not just paused for breath. The silence threshold must be long enough to avoid triggering on natural inter-phrase pauses (typically 200-300ms) but short enough to feel responsive (400-600ms). Tuning this value trades responsiveness against fragmentation: lower values respond faster but may cut users off, higher values wait longer but feel sluggish.

**Key insight:** Combine sentence-boundary rules with silence and turn-length limits for responsive, natural turn detection without requiring ML models. The heuristic provider offers predictable, tunable behavior suitable for most voice interaction scenarios. When rules are insufficient (e.g., detecting barge-in intent or handling multiple overlapping speakers), consider ML-based turn detection with ONNX models, but start with heuristics because they are simpler, faster, and more transparent.

## Variations

### Stricter Sentence Ends

Add semicolons, colons, or ellipses to the sentence-end marker set for handling specific text formats or transcription styles that use these punctuation marks.

```go
turndetection.WithSentenceEndMarkers(".!?;:…"),
```

### Longer Turns (Storytelling)

For longer-form input like dictation or storytelling, increase both maximum turn length and minimum silence duration to avoid fragmenting narratives.

```go
turndetection.WithMaxTurnLength(12000),
turndetection.WithMinSilenceDuration(600*time.Millisecond),
```

## Related Recipes

- **[ML-Based Barge-In](./ml-barge-in)** -- ONNX-based turn detection for barge-in
- **[Handling Speech Interruption](./speech-interruption)** -- Interruption and barge-in behavior
- **[Handling Long Utterances](./long-utterances)** -- Chunking for long user input
