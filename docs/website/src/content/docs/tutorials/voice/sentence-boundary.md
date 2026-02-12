---
title: Sentence-Boundary Turn Detection
description: "Detect user turn completion in Go using heuristic sentence-end markers, configurable silence thresholds, and utterance length constraints with Beluga AI's voice system."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go, tutorial, turn detection, sentence boundary, heuristic, silence threshold, voice"
---

Heuristic turn detection uses simple, predictable rules to determine when a user has completed their utterance. Unlike ML-based approaches that require model files and inference overhead, heuristic detection combines sentence-end markers (`.`, `!`, `?`), minimum silence duration, and utterance length constraints into a fast, deterministic detector. The tradeoff is that heuristics cannot detect conversational nuances like trailing speech or intentional pauses, but for structured interactions (commands, short queries, form-filling) they provide reliable detection with zero dependencies.

## What You Will Build

A heuristic turn detector that identifies user turn completion based on sentence-end punctuation in transcripts, configurable silence thresholds, and minimum/maximum turn length constraints.

## Prerequisites

- Go 1.23+
- Basic familiarity with voice session concepts

## When to Use Heuristic Detection

Heuristic turn detection is appropriate when:

- You want deterministic, predictable behavior that is easy to debug and explain
- Your application processes primarily structured speech (commands, queries, form responses)
- You need minimal compute overhead and no external model files
- You are building a prototype before investing in ML-based detection

For more nuanced detection that handles trailing speech, overlapping turns, and non-verbal cues, see [ML-Based Turn Prediction](/tutorials/voice/ml-turn-prediction).

## Step 1: Create a Heuristic Turn Detector

The heuristic provider follows Beluga's standard registry pattern. The configuration options define the rules the detector applies to each audio frame and transcript pair. `MinSilenceDuration` is the most important parameter -- it controls how long the detector waits after the last detected speech before concluding that the user is done.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lookatitude/beluga-ai/voice/turndetection"
	_ "github.com/lookatitude/beluga-ai/voice/turndetection/providers/heuristic"
)

func main() {
	ctx := context.Background()

	detector, err := turndetection.NewProvider(ctx, "heuristic", turndetection.DefaultConfig(),
		turndetection.WithMinSilenceDuration(400*time.Millisecond),
		turndetection.WithSentenceEndMarkers(".!?"),
		turndetection.WithMinTurnLength(10),
		turndetection.WithMaxTurnLength(5000),
	)
	if err != nil {
		log.Fatalf("create turn detector: %v", err)
	}

	// Test with a simulated audio frame
	audio := make([]byte, 1024)
	done, err := detector.DetectTurn(ctx, audio)
	if err != nil {
		log.Fatalf("detect turn: %v", err)
	}

	fmt.Printf("Turn detected: %v\n", done)
}
```

### Configuration Options

| Option                | Default | Description                                        |
|----------------------|---------|-----------------------------------------------------|
| `MinSilenceDuration` | 500ms   | Silence required after last speech to trigger turn end |
| `SentenceEndMarkers` | `.!?`   | Characters that indicate potential sentence completion |
| `MinTurnLength`      | 10      | Minimum transcript length (characters) for a valid turn |
| `MaxTurnLength`      | 5000    | Maximum transcript length before forcing a turn end |

## Step 2: Detect Turns with Measured Silence

For real-time pipelines where you track silence duration externally (typically from your VAD component), use `DetectTurnWithSilence` to combine your measured silence with the heuristic rules. This separation of concerns is intentional: VAD measures when speech stops, and the turn detector decides whether that stop means the user is done.

```go
	// In your audio processing loop, measure silence duration since last speech
	silenceDuration := 500 * time.Millisecond

	done, err := detector.DetectTurnWithSilence(ctx, audio, silenceDuration)
	if err != nil {
		log.Fatalf("detect turn: %v", err)
	}

	if done {
		fmt.Println("User finished speaking; proceed to LLM/TTS.")
	}
```

The detector combines the measured silence with its internal heuristic rules in a three-step evaluation:

1. Has the silence exceeded `MinSilenceDuration`?
2. Does the current transcript end with a sentence-end marker?
3. Is the transcript length between `MinTurnLength` and `MaxTurnLength`?

All three conditions must be satisfied for a turn to be detected, except when `MaxTurnLength` is reached, which forces a turn end regardless of the other conditions to prevent unbounded accumulation.

## Step 3: Integration with a Voice Session

Combine the heuristic turn detector with a voice session for a complete pipeline. The turn detector is passed as a functional option to the session, which uses it internally to decide when to stop collecting user audio and begin generating a response. This integration means you do not need to manage the turn detection loop yourself -- the session handles it.

```go
import (
	"github.com/lookatitude/beluga-ai/voice/session"
	"github.com/lookatitude/beluga-ai/voice/stt"
	"github.com/lookatitude/beluga-ai/voice/tts"
	"github.com/lookatitude/beluga-ai/voice/turndetection"
)

func createSessionWithTurnDetection(ctx context.Context) {
	sttProvider, err := stt.NewProvider(ctx, "deepgram", stt.DefaultConfig(),
		stt.WithAPIKey("your-key"),
	)
	if err != nil {
		log.Fatalf("create STT: %v", err)
	}

	ttsProvider, err := tts.NewProvider(ctx, "openai", tts.DefaultConfig(),
		tts.WithAPIKey("your-key"),
	)
	if err != nil {
		log.Fatalf("create TTS: %v", err)
	}

	turnDetector, err := turndetection.NewProvider(ctx, "heuristic", turndetection.DefaultConfig(),
		turndetection.WithMinSilenceDuration(400*time.Millisecond),
		turndetection.WithSentenceEndMarkers(".!?"),
	)
	if err != nil {
		log.Fatalf("create turn detector: %v", err)
	}

	sess, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
		session.WithTurnDetector(turnDetector),
		session.WithConfig(session.DefaultConfig()),
	)
	if err != nil {
		log.Fatalf("create session: %v", err)
	}
	defer sess.Stop(ctx)
}
```

## Verification

1. Run the application and verify `DetectTurn` returns `false` for short, incomplete utterances.
2. Use `DetectTurnWithSilence` with a silence duration exceeding `MinSilenceDuration` and verify the turn is detected.
3. Test with transcripts shorter than `MinTurnLength` and verify no premature turn detection.

## Next Steps

- [ML-Based Turn Prediction](/tutorials/voice/ml-turn-prediction) -- Use ONNX models for neural turn detection
- [Sensitivity Tuning](/tutorials/voice/sensitivity-tuning) -- Tune VAD and turn detection together
- [Voice Session Interruptions](/tutorials/voice/session-interruptions) -- Handle barge-in during agent speech
