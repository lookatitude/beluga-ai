---
title: Noise Cancellation
sidebar_position: 7
---

# Noise Cancellation

The Noise Cancellation package provides interfaces and implementations for removing noise from audio signals.

## Overview

The Noise Cancellation package follows the Beluga AI Framework design patterns, providing:

- **Provider abstraction**: Unified interface for multiple noise cancellation algorithms
- **Real-time processing**: Low-latency noise reduction for streaming audio
- **Adaptive algorithms**: Self-adjusting noise profiles for optimal performance
- **Observability**: OTEL metrics and tracing for all operations
- **Error handling**: Comprehensive error codes and retry logic
- **Configuration**: Flexible configuration with validation

## Supported Providers

### Spectral Subtraction

- **Features**: FFT-based, adaptive noise profiles, no model required
- **Model**: None (algorithm-based)
- **Best for**: General noise reduction, low resource requirements
- **Latency**: &lt;20ms
- **Noise Reduction**: 10-20dB

### RNNoise

- **Features**: Deep learning-based, high quality
- **Model**: RNNoise model file required
- **Best for**: High-quality noise suppression
- **Latency**: &lt;30ms
- **Noise Reduction**: 15-25dB

### WebRTC

- **Features**: Built-in noise suppression
- **Model**: Built into WebRTC
- **Best for**: WebRTC-based applications
- **Latency**: &lt;15ms
- **Noise Reduction**: 10-15dB

## Quick Start

### Basic Usage

```go
import (
    "context"
    "log"
    
    "github.com/lookatitude/beluga-ai/pkg/voice/noise"
)

func main() {
    ctx := context.Background()
    
    // Create Noise Cancellation provider
    config := noise.DefaultConfig()
    config.Provider = "spectral"
    config.NoiseReductionLevel = 0.7
    config.SampleRate = 16000
    config.FrameSize = 512
    
    provider, err := noise.NewProvider(ctx, "spectral", config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Process audio
    audio := []byte{/* your audio data */}
    cleaned, err := provider.Process(ctx, audio)
    if err != nil {
        log.Fatal(err)
    }
    
    // Use cleaned audio
    processAudio(cleaned)
}
```

### Streaming Usage

```go
// Process audio stream
audioCh := make(chan []byte, 10)
cleanedCh, err := provider.ProcessStream(ctx, audioCh)
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

// Receive cleaned audio
for cleaned := range cleanedCh {
    // Process cleaned audio
    processAudio(cleaned)
}
```

## Configuration

### Base Configuration

```go
type Config struct {
    Provider                string        // Provider name: "spectral", "rnnoise", "webrtc"
    NoiseReductionLevel     float64       // Noise reduction level (0.0-1.0, default: 0.5)
    SampleRate              int           // Audio sample rate in Hz
    FrameSize               int           // Frame size in samples
    EnableAdaptiveProcessing bool          // Enable adaptive noise reduction
    ModelPath               string        // Path to model file (for ML-based providers)
    Timeout                 time.Duration // Processing timeout
}
```

## Error Handling

The Noise Cancellation package uses structured error handling:

```go
import (
    "errors"
    "github.com/lookatitude/beluga-ai/pkg/voice/noise"
)

if err != nil {
    var noiseErr *noise.NoiseCancellationError
    if errors.As(err, &noiseErr) {
        switch noiseErr.Code {
        case noise.ErrCodeFrameSizeError:
            // Invalid frame size - adjust configuration
        case noise.ErrCodeModelError:
            // Model loading/processing error
        }
    }
}
```

## Observability

### Metrics

- `noise.processing.total`: Total processing operations (counter)
- `noise.processing.latency`: Processing latency (histogram)
- `noise.reduction.level`: Noise reduction level (gauge)

## Performance

- **Latency**: &lt;30ms for most providers
- **CPU Usage**: Low to moderate (5-15% on modern CPUs)
- **Memory**: Low (&lt;50MB for most providers)

## API Reference

For complete API documentation, see the [Noise Cancellation API Reference](../api/packages/voice/noise).

## Next Steps

- [Audio Transport](./transport) - Handle audio I/O
- [Session Management](./session) - Manage voice interactions

