---
title: Voice Activity Detection (VAD)
sidebar_position: 4
---

# Voice Activity Detection (VAD)

The VAD package provides interfaces and implementations for detecting voice activity in audio streams.

## Overview

The VAD package follows the Beluga AI Framework design patterns, providing:

- **Provider abstraction**: Unified interface for multiple VAD algorithms
- **Streaming support**: Real-time voice activity detection on audio streams
- **Low latency**: Optimized for real-time processing
- **Observability**: OTEL metrics and tracing for all operations
- **Error handling**: Comprehensive error codes and retry logic
- **Configuration**: Flexible configuration with validation

## Supported Providers

### Silero VAD

- **Features**: ONNX-based, high accuracy, low latency
- **Model**: Silero VAD v4
- **Best for**: Production applications requiring high accuracy
- **Latency**: \<50ms
- **Accuracy**: 95%+

### Energy-based VAD

- **Features**: Simple, adaptive thresholds, no model required
- **Model**: None (algorithm-based)
- **Best for**: Simple use cases, low resource requirements
- **Latency**: \<10ms
- **Accuracy**: 80-90%

### WebRTC VAD

- **Features**: Built-in VAD, multiple sensitivity modes
- **Model**: Built into WebRTC
- **Best for**: WebRTC-based applications
- **Latency**: \<20ms
- **Accuracy**: 85-90%

### RNNoise VAD

- **Features**: Deep learning-based, noise suppression included
- **Model**: RNNoise model
- **Best for**: Noisy environments
- **Latency**: \<30ms
- **Accuracy**: 90-95%

## Quick Start

### Basic Usage

```go
import (
    "context"
    "log"
    
    "github.com/lookatitude/beluga-ai/pkg/voice/vad"
    "github.com/lookatitude/beluga-ai/pkg/voice/vad/providers/silero"
)

func main() {
    ctx := context.Background()
    
    // Create VAD provider using factory
    config := vad.DefaultConfig()
    config.Provider = "silero"
    config.Threshold = 0.5
    config.SampleRate = 16000
    config.FrameSize = 512
    
    provider, err := vad.NewProvider(ctx, "silero", config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Process audio
    audio := []byte{/* your audio data */}
    hasVoice, err := provider.Process(ctx, audio)
    if err != nil {
        log.Fatal(err)
    }
    
    if hasVoice {
        fmt.Println("Voice activity detected!")
    }
}
```

### Streaming Usage

```go
// Process audio stream
audioCh := make(chan []byte, 10)
resultCh, err := provider.ProcessStream(ctx, audioCh)
if err != nil {
    log.Fatal(err)
}

// Send audio chunks
go func() {
    for audioChunk := range audioChunks {
        audioCh <- audioChunk
    }
    close(audioCh)
}()

// Receive VAD results
for result := range resultCh {
    if result.Error != nil {
        log.Printf("Error: %v", result.Error)
        continue
    }
    
    if result.HasVoice {
        fmt.Println("Voice detected at", result.Timestamp)
    }
}
```

## Configuration

### Base Configuration

```go
type Config struct {
    Provider        string        // Provider name: "silero", "energy", "webrtc", "rnnoise"
    Threshold       float64       // Detection threshold (0.0-1.0, default: 0.5)
    SampleRate      int           // Audio sample rate (8000, 16000, 48000)
    FrameSize       int           // Frame size in samples (default: 512)
    MinSpeechDuration time.Duration // Minimum speech duration (default: 250ms)
    MinSilenceDuration time.Duration // Minimum silence duration (default: 500ms)
    ModelPath       string        // Path to model file (for ML-based providers)
    Timeout         time.Duration // Processing timeout
}
```

### Provider-Specific Configuration

Each provider has additional configuration options. See the [API Reference](../api/packages/voice/vad) for details.

## Error Handling

The VAD package uses structured error handling:

```go
import (
    "errors"
    "github.com/lookatitude/beluga-ai/pkg/voice/vad"
)

if err != nil {
    var vadErr *vad.VADError
    if errors.As(err, &vadErr) {
        switch vadErr.Code {
        case vad.ErrCodeFrameSizeError:
            // Invalid frame size - adjust configuration
        case vad.ErrCodeModelError:
            // Model loading/processing error
        }
    }
}
```

## Observability

### Metrics

- `vad.detections.total`: Total detection operations (counter)
- `vad.detections.voice`: Voice detected count (counter)
- `vad.detections.silence`: Silence detected count (counter)
- `vad.processing.latency`: Processing latency (histogram)

## Performance

- **Latency**: \<50ms for most providers
- **Throughput**: 1000+ frames per second
- **CPU Usage**: Low (\<5% on modern CPUs)

## API Reference

For complete API documentation, see the [VAD API Reference](../api/packages/voice/vad).

## Next Steps

- [Speech-to-Text (STT)](./stt) - Convert speech to text
- [Turn Detection](./turndetection) - Detect turn endings
- [Session Management](./session) - Manage voice interactions

