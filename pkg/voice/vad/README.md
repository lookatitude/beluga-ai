# VAD Package

The VAD (Voice Activity Detection) package provides interfaces and implementations for detecting voice activity in audio streams.

## Overview

The VAD package follows the Beluga AI Framework design patterns, providing:
- **Provider abstraction**: Unified interface for multiple VAD algorithms
- **Streaming support**: Real-time voice activity detection on audio streams
- **Low latency**: Optimized for real-time processing
- **Observability**: OTEL metrics and tracing for all operations
- **Error handling**: Comprehensive error codes and retry logic
- **Configuration**: Flexible configuration with validation

## Supported Providers

- **Silero**: ONNX-based VAD model with high accuracy
- **Energy-based**: Simple energy threshold detection with adaptive thresholds
- **WebRTC**: WebRTC's built-in VAD with multiple sensitivity modes
- **RNNoise**: RNNoise-based VAD with noise suppression

## Quick Start

### Basic Usage

```go
import (
    "context"
    
    "github.com/lookatitude/beluga-ai/pkg/voice/vad/providers/silero"
)

func main() {
    ctx := context.Background()
    
    // Create VAD provider
    vadProvider, err := silero.NewSileroVAD(ctx, silero.Config{
        ModelPath: "path/to/silero_vad.onnx",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Process audio
    audio := []byte{/* your audio data */}
    hasSpeech, err := vadProvider.Process(ctx, audio)
    if err != nil {
        log.Fatal(err)
    }
    
    if hasSpeech {
        fmt.Println("Speech detected in audio")
    } else {
        fmt.Println("No speech detected")
    }
}
```

// Create VAD provider
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
audio := []byte{...} // Your audio data
hasVoice, err := provider.Process(ctx, audio)
if err != nil {
    log.Fatal(err)
}

if hasVoice {
    fmt.Println("Voice activity detected!")
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
        fmt.Printf("Voice detected (confidence: %.2f)\n", result.Confidence)
    }
}
```

## Configuration

### Base Configuration

```go
type Config struct {
    Provider          string        // Provider name: "silero", "energy", "webrtc", "rnnoise"
    Threshold         float64       // Speech detection threshold (0.0-1.0)
    FrameSize         int           // Audio frame size in samples
    SampleRate        int           // Audio sample rate in Hz
    MinSpeechDuration time.Duration // Minimum speech duration
    MaxSilenceDuration time.Duration // Maximum silence duration
    EnablePreprocessing bool        // Enable audio preprocessing
    ModelPath         string        // Path to model file (for ML-based providers)
    Timeout           time.Duration // Processing timeout
}
```

### Provider-Specific Configuration

Each provider extends the base config with provider-specific settings. See provider documentation for details:
- [Silero Configuration](./providers/silero/README.md)
- [Energy Configuration](./providers/energy/README.md)
- [WebRTC Configuration](./providers/webrtc/README.md)
- [RNNoise Configuration](./providers/rnnoise/README.md)

## Error Handling

The VAD package uses structured error handling with error codes:

```go
if err != nil {
    var vadErr *vad.VADError
    if errors.As(err, &vadErr) {
        switch vadErr.Code {
        case vad.ErrCodeTimeout:
            // Processing timeout
        case vad.ErrCodeModelLoadFailed:
            // Model loading failed
        case vad.ErrCodeFrameSizeError:
            // Invalid frame size
        }
    }
}
```

### Error Codes

- `ErrCodeInvalidConfig`: Invalid configuration
- `ErrCodeInternalError`: Internal processing error
- `ErrCodeTimeout`: Processing timeout
- `ErrCodeModelLoadFailed`: Model loading failed
- `ErrCodeModelNotFound`: Model file not found
- `ErrCodeFrameSizeError`: Invalid frame size
- `ErrCodeSampleRateError`: Invalid sample rate

## Observability

### Metrics

The VAD package emits OTEL metrics:

- `vad.frames.processed`: Total processed audio frames
- `vad.speech.detected`: Speech detected events
- `vad.silence.detected`: Silence detected events
- `vad.errors.total`: Total errors
- `vad.processing.latency`: Processing latency histogram

### Tracing

All operations create OpenTelemetry spans with attributes:
- `provider`: Provider name
- `frame_size`: Frame size
- `sample_rate`: Sample rate
- `threshold`: Detection threshold

## Provider Registry

Providers are automatically registered via `init()` functions. You can also manually register custom providers:

```go
registry := vad.GetRegistry()
registry.Register("custom-provider", func(config *vad.Config) (vadiface.VADProvider, error) {
    return NewCustomProvider(config)
})
```

## Testing

The package includes comprehensive test utilities:

```go
import "github.com/lookatitude/beluga-ai/pkg/voice/vad"

// Create mock provider
mockProvider := vad.NewAdvancedMockVADProvider("test",
    vad.WithSpeechResults(true, false, true),
    vad.WithProcessingDelay(10*time.Millisecond),
)

// Use in tests
hasVoice, err := mockProvider.Process(ctx, audio)
```

## Examples

See the [examples directory](../../../examples/voice/vad/) for complete usage examples.

## Performance

- **Latency**: Sub-10ms for most providers
- **Throughput**: Supports 1000+ frames per second
- **Concurrency**: Thread-safe, supports concurrent requests

## License

Part of the Beluga AI Framework. See main LICENSE file.

