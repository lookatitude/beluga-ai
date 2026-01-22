# Sentence-Boundary Turn Detection with Heuristic Provider

**What you will build:** A turn detector that identifies when a user has finished speaking using sentence-end markers (e.g. `.`, `!`, `?`), minimum/maximum turn length, and optional question markers. Uses `pkg/voice/turndetection` with the **heuristic** provider.

## Learning Objectives

- Configure the heuristic turn-detection provider
- Use `MinSilenceDuration`, `SentenceEndMarkers`, and `MinTurnLength` / `MaxTurnLength`
- Integrate turn detection with a voice session or STT pipeline

## Prerequisites

- Go 1.24+
- Beluga AI (`go get github.com/lookatitude/beluga-ai`)
- Basic familiarity with [Voice Sessions](../../use-cases/voice-sessions.md) or STT

## Step 1: Create a Heuristic Turn Detector
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
		turndetection.WithMinSilenceDuration(400*time.Millisecond),
		turndetection.WithSentenceEndMarkers(".!?"),
		turndetection.WithMinTurnLength(10),
		turndetection.WithMaxTurnLength(5000),
	)
	if err != nil {
		log.Fatalf("create turn detector: %v", err)
	}
	defer func() { _ = detector }()

	// Simulated audio chunk (heuristic often works with transcript length; audio used for interface)
	audio := make([]byte, 1024)
	done, err := detector.DetectTurn(ctx, audio)
	if err != nil {
		log.Fatalf("detect turn: %v", err)
	}
	fmt.Printf("Turn detected: %v\n", done)
}
```

## Step 2: Use DetectTurnWithSilence

For real-time pipelines, combine silence duration with heuristic rules:
```text
go
go
	silenceDuration := 500 * time.Millisecond
	done, err := detector.DetectTurnWithSilence(ctx, audio, silenceDuration)
	if err != nil {
		log.Fatalf("detect turn: %v", err)
	}
	if done {
		fmt.Println("User finished speaking; proceed to LLM/TTS.")
	}
```

## Verification
```bash
go run .
```
# Expected: "Turn detected: false" (or true when silence >= MinSilenceDuration with DetectTurnWithSilence)
```

## Next Steps

- **[ML-Based Turn Prediction](./voice-turn-ml-based-prediction.md)** — Use the ONNX provider for model-driven detection.
- **[Custom Turn Detectors in Noisy Environments](../../integrations/voice/turn/custom-turn-detectors-noisy-environments.md)** — Tune for noisy settings.
