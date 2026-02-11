---
title: ML-Based Turn Prediction with ONNX
description: Use an ONNX neural network model for accurate, context-aware turn-end detection in voice applications.
---

Machine learning-based turn prediction uses a trained neural network to determine when a user has finished speaking, offering more nuanced detection than rule-based heuristics. While heuristic detection relies on silence duration and punctuation -- signals that are easy to compute but miss conversational nuance -- ML models learn patterns from real conversations: trailing intonation, filler words, breathing patterns, and prosodic cues that indicate turn boundaries. This tutorial demonstrates how to configure the ONNX turn detection provider with a custom model for production voice applications.

## What You Will Build

A turn detector powered by an ONNX model that analyzes audio features to predict turn boundaries, with configurable thresholds and silence constraints.

## Prerequisites

- Go 1.23+
- An ONNX turn-detection model file (or a placeholder for development)
- Completion of [Sentence-Boundary Turn Detection](/tutorials/voice/sentence-boundary) is recommended

## When to Use ML-Based Detection

ML-based turn detection is appropriate when:

- Users speak naturally with pauses, filler words, and trailing speech
- Your application handles multiple languages or dialects
- Heuristic rules produce too many false positives or false negatives
- You need to distinguish intentional pauses from end-of-turn silence

For simpler use cases with structured speech (commands, short queries), the heuristic provider in [Sentence-Boundary Turn Detection](/tutorials/voice/sentence-boundary) is sufficient and avoids the complexity of model management.

## Step 1: Configure the ONNX Provider

The ONNX provider uses Beluga's standard registry pattern -- the blank import registers the `"onnx"` factory, and `turndetection.NewProvider` creates a configured instance. The functional options control the model file path and detection parameters. The `Threshold` option is the primary tuning knob: it determines the confidence level at which the model's prediction is accepted as a turn boundary.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/lookatitude/beluga-ai/voice/turndetection"
	_ "github.com/lookatitude/beluga-ai/voice/turndetection/providers/onnx"
)

func main() {
	ctx := context.Background()

	modelPath := os.Getenv("TURN_MODEL_PATH")
	if modelPath == "" {
		modelPath = filepath.Join(os.TempDir(), "turn_detection.onnx")
	}

	detector, err := turndetection.NewProvider(ctx, "onnx", turndetection.DefaultConfig(),
		turndetection.WithModelPath(modelPath),
		turndetection.WithThreshold(0.5),
		turndetection.WithMinSilenceDuration(300*time.Millisecond),
		turndetection.WithMinTurnLength(5),
		turndetection.WithMaxTurnLength(10000),
	)
	if err != nil {
		log.Fatalf("create ONNX turn detector: %v", err)
	}

	// Analyze an audio frame (16 kHz mono, ~64ms)
	audio := make([]byte, 2048)
	done, err := detector.DetectTurn(ctx, audio)
	if err != nil {
		log.Fatalf("detect turn: %v", err)
	}

	fmt.Printf("Turn detected: %v\n", done)
}
```

## Configuration Options

| Option               | Default              | Description                                  |
|---------------------|----------------------|----------------------------------------------|
| `ModelPath`         | `turn_detection.onnx`| Path to the ONNX model file                 |
| `Threshold`         | 0.5                  | Detection confidence threshold (0.0-1.0)     |
| `MinSilenceDuration`| 500ms                | Minimum silence before considering turn end   |
| `MinTurnLength`     | 10                   | Minimum turn length (samples/characters)      |
| `MaxTurnLength`     | 5000                 | Maximum turn length before forcing end        |

## Step 2: Combined Detection with Silence Duration

For real-time pipelines where you measure silence duration externally (for example, from your VAD provider), pass it to `DetectTurnWithSilence` for combined inference. This approach is more accurate than relying on the model alone because it fuses two independent signals: the model's acoustic analysis and the measured silence duration. The model may be uncertain about a turn boundary, but if the measured silence exceeds the minimum threshold, the combined signal is strong enough to trigger.

```go
	silence := 400 * time.Millisecond
	done, err := detector.DetectTurnWithSilence(ctx, audio, silence)
	if err != nil {
		log.Fatalf("detect turn: %v", err)
	}

	if done {
		fmt.Println("Turn end detected; ready for response generation.")
	}
```

The ONNX provider combines the model's confidence score with the measured silence duration. If both the model confidence exceeds the threshold and the silence exceeds `MinSilenceDuration`, the turn is considered complete. This dual-condition approach reduces false positives from either signal alone.

## Step 3: Threshold Tuning

The threshold controls the sensitivity of the model's predictions. Tuning it is an application-specific decision that depends on how your users interact with the agent and what the cost of a wrong prediction is:

- **Lower threshold (0.3-0.4)**: More aggressive detection, responds faster but may trigger on mid-sentence pauses. Use this for fast-paced interactions where responsiveness matters more than accuracy, such as customer service bots handling simple queries.
- **Default threshold (0.5)**: Balanced detection for general conversation.
- **Higher threshold (0.6-0.8)**: Conservative detection, waits longer but avoids false triggers. Use this for applications where premature responses are disruptive, such as tutoring or therapy contexts.

```go
	// For a fast-paced customer service bot
	detector, err := turndetection.NewProvider(ctx, "onnx", turndetection.DefaultConfig(),
		turndetection.WithModelPath(modelPath),
		turndetection.WithThreshold(0.4), // Respond quickly
		turndetection.WithMinSilenceDuration(250*time.Millisecond),
	)

	// For a patient tutoring assistant
	detector, err := turndetection.NewProvider(ctx, "onnx", turndetection.DefaultConfig(),
		turndetection.WithModelPath(modelPath),
		turndetection.WithThreshold(0.7), // Wait for clear turn boundaries
		turndetection.WithMinSilenceDuration(600*time.Millisecond),
	)
```

## Step 4: Model Selection

The ONNX provider works with any compatible turn-detection model. The ONNX Runtime format is chosen because it provides cross-platform inference without requiring a specific ML framework at runtime. When selecting a model, consider these factors:

| Factor          | Guidance                                        |
|----------------|--------------------------------------------------|
| Input format   | Must match your audio format (sample rate, encoding) |
| Latency        | Smaller models run faster; target < 10ms per frame |
| Accuracy       | Evaluate on your domain-specific test set        |
| Language        | Multilingual models vs. language-specific models  |

## Verification

1. Set the `TURN_MODEL_PATH` environment variable to your ONNX model path.
2. Run the application with sample audio.
3. Verify that `DetectTurn` correctly identifies turn boundaries.
4. Test with a range of threshold values and measure the false positive/negative rate.
5. Compare latency and accuracy against the heuristic provider.

## Next Steps

- [Sentence-Boundary Turn Detection](/tutorials/voice/sentence-boundary) -- Heuristic alternative for simpler use cases
- [Sensitivity Tuning](/tutorials/voice/sensitivity-tuning) -- Combine turn detection with VAD tuning
- [Custom Silero VAD](/tutorials/voice/custom-silero-vad) -- Neural VAD for voice activity detection
