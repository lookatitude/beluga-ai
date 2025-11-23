# Noise Cancellation Package

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

- **Spectral Subtraction**: FFT-based noise reduction with adaptive noise profiles
- **RNNoise**: Deep learning-based noise suppression (requires model file)
- **WebRTC**: WebRTC's built-in noise suppression

## Quick Start

### Basic Usage

```go
import (
    "context"
    
    "github.com/lookatitude/beluga-ai/pkg/voice/noise/providers/spectral"
)

func main() {
    ctx := context.Background()
    
    // Create Noise Cancellation provider

// Create Noise Cancellation provider
config := noise.DefaultConfig()
config.Provider = "spectral"

provider, err := noise.NewProvider(ctx, "spectral", config)
if err != nil {
    log.Fatal(err)
}

// Process audio
audio := []byte{...} // Your audio data
cleaned, err := provider.Process(ctx, audio)
if err != nil {
    log.Fatal(err)
}

// Process audio stream
audioCh := make(chan []byte, 10)
// ... send audio chunks to audioCh ...
cleanedCh, err := provider.ProcessStream(ctx, audioCh)
if err != nil {
    log.Fatal(err)
}

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

### Provider-Specific Configuration

Each provider extends the base config with provider-specific settings. See provider documentation for details:
- [Spectral Configuration](./providers/spectral/README.md)
- [RNNoise Configuration](./providers/rnnoise/README.md)
- [WebRTC Configuration](./providers/webrtc/README.md)

## Error Handling

The Noise Cancellation package uses structured error handling with error codes:

```go
if err != nil {
    var noiseErr *noise.NoiseCancellationError
    if errors.As(err, &noiseErr) {
        switch noiseErr.Code {
        case noise.ErrCodeFrameSizeError:
            // Invalid frame size - adjust configuration
        case noise.ErrCodeProcessingError:
            // Processing error - retryable
        case noise.ErrCodeTimeout:
            // Operation timeout - retryable
        }
    }
}
```

### Error Codes

- `ErrCodeInvalidConfig`: Invalid configuration
- `ErrCodeInternalError`: Internal processing error
- `ErrCodeInvalidInput`: Invalid input audio data
- `ErrCodeTimeout`: Operation timeout
- `ErrCodeUnsupportedProvider`: Provider not registered
- `ErrCodeModelLoadFailed`: Failed to load model file
- `ErrCodeModelNotFound`: Model file not found
- `ErrCodeProcessingError`: Processing error (retryable)
- `ErrCodeFrameSizeError`: Invalid frame size
- `ErrCodeSampleRateError`: Invalid sample rate

## Observability

### Metrics

The Noise Cancellation package emits OTEL metrics:

- `noise.frames.processed`: Total processed audio frames
- `noise.bytes.processed`: Total bytes processed
- `noise.bytes.output`: Total bytes output
- `noise.errors.total`: Total Noise Cancellation errors
- `noise.processing.latency`: Processing latency histogram

### Tracing

All operations create OpenTelemetry spans with attributes:
- `provider`: Provider name
- `sample_rate`: Sample rate
- `frame_size`: Frame size
- `noise_reduction_level`: Noise reduction level

## Provider Registry

Providers are automatically registered via `init()` functions. You can also manually register custom providers:

```go
registry := noise.GetRegistry()
registry.Register("custom-provider", func(config *noise.Config) (noiseiface.NoiseCancellation, error) {
    return NewCustomProvider(config)
})
```

## Testing

The package includes comprehensive test utilities:

```go
import "github.com/lookatitude/beluga-ai/pkg/voice/noise"

// Create mock noise cancellation
mockNoise := noise.NewAdvancedMockNoiseCancellation("test",
    noise.WithProcessedAudio([]byte{1, 2, 3}, []byte{4, 5, 6}),
    noise.WithProcessingDelay(10*time.Millisecond),
)

// Use in tests
cleaned, err := mockNoise.Process(ctx, audio)
```

## Examples

See the [examples directory](../../../examples/voice/noise/) for complete usage examples.

## Performance

- **Latency**: Sub-10ms for Spectral, sub-5ms for RNNoise, sub-20ms for WebRTC
- **Throughput**: Supports 1000+ frames per second
- **Concurrency**: Thread-safe, supports concurrent operations

## License

Part of the Beluga AI Framework. See main LICENSE file.

