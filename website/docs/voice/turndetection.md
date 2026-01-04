---
title: Turn Detection
sidebar_position: 5
---

# Turn Detection

The Turn Detection package provides interfaces and implementations for detecting the end of a user's turn in a conversation.

## Overview

The Turn Detection package follows the Beluga AI Framework design patterns, providing:

- **Provider abstraction**: Unified interface for multiple turn detection algorithms
- **Silence-based detection**: Detects turns based on silence duration
- **ML-based detection**: Uses machine learning models for accurate turn detection
- **Observability**: OTEL metrics and tracing for all operations
- **Error handling**: Comprehensive error codes and retry logic
- **Configuration**: Flexible configuration with validation

## Supported Providers

### Heuristic

- **Features**: Rule-based using sentence endings, questions, and silence
- **Model**: None (algorithm-based)
- **Best for**: Simple use cases, low latency requirements
- **Latency**: &lt;10ms
- **Accuracy**: 80-85%

### ONNX

- **Features**: Machine learning-based turn detection
- **Model**: ONNX model file required
- **Best for**: High accuracy requirements
- **Latency**: &lt;50ms
- **Accuracy**: 90-95%

## Quick Start

### Basic Usage

```go
import (
    "context"
    "time"
    "log"
    
    "github.com/lookatitude/beluga-ai/pkg/voice/turndetection"
)

func main() {
    ctx := context.Background()
    
    // Create Turn Detection provider
    config := turndetection.DefaultConfig()
    config.Provider = "heuristic"
    config.MinSilenceDuration = 500 * time.Millisecond
    
    provider, err := turndetection.NewDetector(ctx, "heuristic", config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Detect turn end
    audio := []byte{/* your audio data */}
    turnEnd, err := provider.DetectTurn(ctx, audio)
    if err != nil {
        log.Fatal(err)
    }
    
    if turnEnd {
        fmt.Println("Turn end detected!")
    }
}
```

### Silence-Based Detection

```go
// Detect turn with silence duration
silenceDuration := 600 * time.Millisecond
turnEnd, err := provider.DetectTurnWithSilence(ctx, audio, silenceDuration)
if err != nil {
    log.Fatal(err)
}

if turnEnd {
    fmt.Println("Turn end detected based on silence!")
}
```

## Configuration

### Base Configuration

```go
type Config struct {
    Provider          string        // Provider name: "heuristic", "onnx"
    MinSilenceDuration time.Duration // Minimum silence duration (default: 500ms)
    MinTurnLength     int           // Minimum turn length in characters
    MaxTurnLength     int           // Maximum turn length in characters
    SentenceEndMarkers string       // Characters indicating sentence endings
    QuestionMarkers   []string      // Phrases indicating questions
    Threshold         float64       // Detection threshold (0.0-1.0)
    ModelPath         string        // Path to model file (for ML-based providers)
    Timeout           time.Duration // Processing timeout
}
```

## Error Handling

The Turn Detection package uses structured error handling:

```go
import (
    "errors"
    "github.com/lookatitude/beluga-ai/pkg/voice/turndetection"
)

if err != nil {
    var tdErr *turndetection.TurnDetectionError
    if errors.As(err, &tdErr) {
        switch tdErr.Code {
        case turndetection.ErrCodeModelError:
            // Model loading/processing error
        }
    }
}
```

## Observability

### Metrics

- `turndetection.detections.total`: Total detection operations (counter)
- `turndetection.detections.turn_end`: Turn end detected count (counter)
- `turndetection.processing.latency`: Processing latency (histogram)

## Performance

- **Latency**: &lt;50ms for most providers
- **Accuracy**: 80-95% depending on provider
- **CPU Usage**: Low (&lt;5% on modern CPUs)

## API Reference

For complete API documentation, see the [Turn Detection API Reference](../api/packages/voice/turndetection).

## Next Steps

- [Voice Activity Detection (VAD)](./vad) - Detect voice in audio
- [Session Management](./session) - Manage voice interactions

