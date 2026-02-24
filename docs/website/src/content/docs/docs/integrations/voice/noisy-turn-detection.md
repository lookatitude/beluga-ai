---
title: Noisy Environment Turn Detection
description: "Tune Beluga AI turn detection for noisy environments like contact centers and factory floors to prevent false speech triggers."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "noisy turn detection, contact center voice, Beluga AI, noise-resistant VAD, voice AI tuning, Go voice pipeline"
---

Default turn detection parameters assume clean audio. In real-world deployments -- contact centers with hold music, retail floors with announcements, or factory environments with machinery -- background noise causes frequent false turn-end triggers that interrupt users. This guide covers configuring both heuristic and ONNX turn detection providers for noise-resistant operation, ensuring your voice agent waits for genuine speech completion before responding.

## Overview

The `voice/turndetection` package provides heuristic and ONNX-based turn detection. In noisy settings (contact centers, retail, factory floors), default parameters often produce too many false positives. By adjusting `MinSilenceDuration`, `Threshold`, and turn-length limits, you can achieve reliable detection despite ambient noise.

## Prerequisites

- Go 1.23 or later
- A voice pipeline with VAD providing silence duration
- (Optional) ONNX turn-detection model for the `onnx` provider

## Installation

```bash
go get github.com/lookatitude/beluga-ai
```

## Configuration

### Heuristic Provider for Noisy Settings

Increase `MinSilenceDuration` so brief noise gaps are not treated as end-of-turn. Use `WithMinTurnLength` to filter out very short spurious turns:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/lookatitude/beluga-ai/voice/turndetection"
)

func main() {
    ctx := context.Background()

    cfg := turndetection.DefaultConfig()
    detector, err := turndetection.NewProvider(ctx, "heuristic", cfg,
        turndetection.WithMinSilenceDuration(700*time.Millisecond),
        turndetection.WithMinTurnLength(20),
        turndetection.WithMaxTurnLength(8000),
        turndetection.WithSentenceEndMarkers(".!?"),
    )
    if err != nil {
        log.Fatalf("Failed to create detector: %v", err)
    }

    audio := make([]byte, 2048)
    silence := 800 * time.Millisecond
    done, err := detector.DetectTurnWithSilence(ctx, audio, silence)
    if err != nil {
        log.Fatalf("Detection failed: %v", err)
    }
    fmt.Printf("Turn detected: %v\n", done)
}
```

### ONNX Provider with Higher Threshold

Raise `Threshold` to require stronger model confidence before declaring end-of-turn, reducing false positives in noise:

```go
import "os"

detector, err := turndetection.NewProvider(ctx, "onnx", cfg,
    turndetection.WithModelPath(os.Getenv("TURN_MODEL_PATH")),
    turndetection.WithThreshold(0.6),
    turndetection.WithMinSilenceDuration(600*time.Millisecond),
    turndetection.WithMinTurnLength(15),
)
if err != nil {
    log.Fatalf("Failed to create ONNX detector: %v", err)
}
```

## Configuration Reference

| Option               | Description                        | Default  | Noisy Environment |
|----------------------|------------------------------------|----------|--------------------|
| `MinSilenceDuration` | Minimum silence to trigger turn end | 500 ms  | 600-800 ms         |
| `Threshold`          | ONNX detection threshold (0-1)    | 0.5      | 0.55-0.65          |
| `MinTurnLength`      | Minimum turn length                | 10       | 15-25              |
| `MaxTurnLength`      | Maximum turn length                | 5000     | 8000+              |
| `SentenceEndMarkers` | Heuristic sentence-end characters  | `.!?`    | Keep or extend     |

## Troubleshooting

### Too many false turn-end events in noise

Increase `MinSilenceDuration` (600-800 ms) and, for ONNX, `Threshold` (0.55-0.65). Use `DetectTurnWithSilence` fed by a robust VAD so silence is computed from actual speech absence rather than raw audio energy.

### Missed end-of-turn in noisy segments

Ensure VAD correctly identifies silence. Avoid over-incrementing `MinSilenceDuration`. Consider switching to the ONNX provider if heuristic detection is insufficient. Verify audio format (sample rate, chunk size) matches provider expectations.

### Provider 'onnx' not registered

Import the ONNX provider to trigger registration:

```go
import _ "github.com/lookatitude/beluga-ai/voice/turndetection/providers/onnx"
```

Set `TURN_MODEL_PATH` to a valid ONNX model file path.

## Advanced Topics

### Production Deployment

- Use `turndetection.IsRetryableError(err)` and retry where appropriate
- Call `turndetection.InitMetrics(meter, tracer)` at startup for OpenTelemetry monitoring
- A/B test heuristic vs ONNX providers with different thresholds using metrics before rollout
- Run with real or synthetic noisy audio to validate settings against ground truth

## Related Resources

- [Heuristic Turn Detection Tuning](/docs/integrations/turn-heuristic-tuning)
- [Voice Services Overview](/docs/integrations/voice-services)
