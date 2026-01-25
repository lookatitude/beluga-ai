# Heuristic Turn Detection Tuning

Welcome, colleague! In this guide we'll tune the **heuristic** turn-detection provider (`pkg/voice/turndetection`) using `MinSilenceDuration`, sentence-end markers, question markers, and min/max turn length. You'll see how each option affects behavior and how to verify your setup.

## What you will build

You will configure and tune the heuristic turn detector for your use case: adjust silence duration, punctuation rules, and turn-length limits. This integration allows you to improve accuracy and latency without deploying an ONNX model.

## Learning Objectives

- ✅ Configure the heuristic provider with `Config` and options
- ✅ Tune `MinSilenceDuration`, `SentenceEndMarkers`, and turn-length limits
- ✅ Use `DetectTurn` and `DetectTurnWithSilence` correctly
- ✅ Verify behavior and iterate on settings

## Prerequisites

- Go 1.24 or later
- Beluga AI (`go get github.com/lookatitude/beluga-ai`)

## Step 1: Setup and Installation
bash
```bash
go get github.com/lookatitude/beluga-ai
```

## Step 2: Basic Heuristic Configuration

Create a heuristic detector with defaults, then override key options:
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
		turndetection.WithMinSilenceDuration(500*time.Millisecond),
		turndetection.WithSentenceEndMarkers(".!?"),
		turndetection.WithMinTurnLength(10),
		turndetection.WithMaxTurnLength(5000),
	)
	if err != nil {
		log.Fatalf("create detector: %v", err)
	}

	audio := make([]byte, 1024)
	done, err := detector.DetectTurn(ctx, audio)
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
# Expected: "Turn detected: false" (DetectTurn with placeholder audio)
```

## Step 3: Use DetectTurnWithSilence

Heuristic turn-end often relies on silence. Use `DetectTurnWithSilence` when you have silence duration from VAD or STT:
go
```go
	silence := 550 * time.Millisecond
	done, err := detector.DetectTurnWithSilence(ctx, audio, silence)
	if err != nil {
		log.Fatalf("detect: %v", err)
	}
	// done == true when silence >= MinSilenceDuration (500 ms)
	fmt.Printf("Turn detected: %v\n", done)
```

### Verification

Try `silence` just below and above `MinSilenceDuration`; `done` should flip when silence meets or exceeds the configured minimum.

## Step 4: Tuning MinSilenceDuration and Markers

| Goal | Change |
|------|--------|
| Faster response | Decrease `MinSilenceDuration` (e.g. 300–400 ms). Risk: more false turn-end. |
| Fewer false turn-end | Increase `MinSilenceDuration` (e.g. 600–800 ms). Risk: slower response. |
| Stricter sentence ends | Use `WithSentenceEndMarkers(".!?")` (default) or add `;` etc. |
| Longer turns | Increase `WithMaxTurnLength` (e.g. 8000–10000). |

Example: longer silence and longer max turn for cautious detection:
```text
go
go
	detector, err := turndetection.NewProvider(ctx, "heuristic", cfg,
		turndetection.WithMinSilenceDuration(700*time.Millisecond),
		turndetection.WithMaxTurnLength(10000),
	)
```

## Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `MinSilenceDuration` | Min silence to treat as turn end | 500 ms |
| `SentenceEndMarkers` | Heuristic sentence-end runes | `.!?` |
| `MinTurnLength` | Min turn length | 10 |
| `MaxTurnLength` | Max turn length | 5000 |
| `Timeout` | Operation timeout | 1 s |

## Common Issues

### "Turn detected too late"

**Problem**: Users wait too long before the system responds.

**Solution**: Lower `MinSilenceDuration` (e.g. 350–450 ms). Ensure you use `DetectTurnWithSilence` with accurate silence from VAD/STT.

### "Turn detected too early / user gets cut off"

**Problem**: System treats brief pauses as end-of-turn.

**Solution**: Increase `MinSilenceDuration` (e.g. 600–700 ms). Optionally increase `MinTurnLength` so very short segments are not considered complete turns.

### "Provider 'heuristic' not found"

**Problem**: Heuristic provider not registered.

**Solution**: The `heuristic` package registers itself via `init()`. Ensure you import `github.com/lookatitude/beluga-ai/pkg/voice/turndetection` (and thus the `heuristic` subpackage).

## Production Considerations

- **Observability**: Call `turndetection.InitMetrics(meter, tracer)` at startup and use OTEL to monitor turn-end rate and latency.
- **Validation**: Run `cfg.Validate()` if you build `Config` manually instead of using `DefaultConfig()` and options.
- **Context**: Pass a `context.Context` with timeout or cancellation to `NewProvider` and `DetectTurn` / `DetectTurnWithSilence`.

## Next Steps

- **[Custom Turn Detectors for Noisy Environments](./custom-turn-detectors-noisy-environments.md)** — Tuning for noisy settings.
- **[Sentence-Boundary Turn Detection](../../../tutorials/voice/voice-turn-sentence-boundary-detection.md)** — Tutorial on heuristic provider.
- **[ML-Based Turn Prediction](../../../tutorials/voice/voice-turn-ml-based-prediction.md)** — ONNX provider.
