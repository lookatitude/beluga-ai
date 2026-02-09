---
title: Heuristic Turn Detection Tuning
description: Fine-tune heuristic turn detection parameters for silence duration, sentence markers, and turn length.
---

The heuristic turn detection provider uses configurable silence duration, punctuation rules, and turn-length limits to determine when a speaker has finished their turn. This guide covers tuning these parameters for different use cases without deploying an ONNX model.

## Overview

The `voice/turndetection` package's heuristic provider offers a lightweight, configuration-driven approach to turn detection. By adjusting `MinSilenceDuration`, `SentenceEndMarkers`, and turn-length limits, you can optimize for responsiveness or accuracy depending on your scenario.

## Prerequisites

- Go 1.23 or later

## Installation

```bash
go get github.com/lookatitude/beluga-ai
```

## Configuration

Create a heuristic detector with tuned parameters:

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
        turndetection.WithMinSilenceDuration(500*time.Millisecond),
        turndetection.WithSentenceEndMarkers(".!?"),
        turndetection.WithMinTurnLength(10),
        turndetection.WithMaxTurnLength(5000),
    )
    if err != nil {
        log.Fatalf("Failed to create detector: %v", err)
    }

    audio := make([]byte, 1024)
    done, err := detector.DetectTurn(ctx, audio)
    if err != nil {
        log.Fatalf("Detection failed: %v", err)
    }
    fmt.Printf("Turn detected: %v\n", done)
}
```

## Usage

### DetectTurnWithSilence

When you have silence duration from VAD or STT, use `DetectTurnWithSilence` for more accurate detection:

```go
silence := 550 * time.Millisecond
done, err := detector.DetectTurnWithSilence(ctx, audio, silence)
if err != nil {
    log.Fatalf("Detection failed: %v", err)
}
// done == true when silence >= MinSilenceDuration (500 ms)
fmt.Printf("Turn detected: %v\n", done)
```

Test with silence values just below and above `MinSilenceDuration` to verify the threshold behavior.

### Tuning Guidelines

| Goal                  | Adjustment                                                              |
|-----------------------|-------------------------------------------------------------------------|
| Faster response       | Decrease `MinSilenceDuration` (300-400 ms). Risk: more false turn-end. |
| Fewer false turn-end  | Increase `MinSilenceDuration` (600-800 ms). Risk: slower response.     |
| Stricter sentence end | Use `WithSentenceEndMarkers(".!?")` or add `;` etc.                    |
| Longer turns          | Increase `WithMaxTurnLength` (8000-10000).                             |

Example for cautious detection with longer silence and longer maximum turn length:

```go
detector, err := turndetection.NewProvider(ctx, "heuristic", cfg,
    turndetection.WithMinSilenceDuration(700*time.Millisecond),
    turndetection.WithMaxTurnLength(10000),
)
```

## Configuration Reference

| Option               | Description                  | Default |
|----------------------|------------------------------|---------|
| `MinSilenceDuration` | Min silence to trigger turn end | 500 ms |
| `SentenceEndMarkers` | Sentence-end characters      | `.!?`   |
| `MinTurnLength`      | Minimum turn length          | 10      |
| `MaxTurnLength`      | Maximum turn length          | 5000    |
| `Timeout`            | Operation timeout            | 1 s     |

## Troubleshooting

### Turn detected too late

Lower `MinSilenceDuration` (350-450 ms). Ensure you use `DetectTurnWithSilence` with accurate silence values from VAD/STT.

### Turn detected too early / user gets cut off

Increase `MinSilenceDuration` (600-700 ms). Optionally increase `MinTurnLength` so very short segments are not considered complete turns.

### Provider 'heuristic' not found

The heuristic package registers itself via `init()`. Ensure you import the `voice/turndetection` package:

```go
import "github.com/lookatitude/beluga-ai/voice/turndetection"
```

## Advanced Topics

### Production Deployment

- Call `turndetection.InitMetrics(meter, tracer)` at startup for OpenTelemetry monitoring of turn-end rate and latency
- Use `cfg.Validate()` when building `Config` manually instead of using `DefaultConfig()` with options
- Pass a `context.Context` with timeout or cancellation to `NewProvider` and detection methods

## Related Resources

- [Custom Turn Detectors for Noisy Environments](/integrations/noisy-turn-detection)
- [Voice Services Overview](/integrations/voice-services)
