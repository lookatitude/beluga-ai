# Turn Detection Package

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

- **Heuristic**: Rule-based turn detection using sentence endings, questions, and silence
- **ONNX**: Machine learning-based turn detection using ONNX models

## Quick Start

### Basic Usage

```go
import (
    "context"
    "time"
    
    "github.com/lookatitude/beluga-ai/pkg/voice/turndetection/providers/heuristic"
)

func main() {
    ctx := context.Background()
    
    // Create Turn Detection provider
config := turndetection.DefaultConfig()
config.Provider = "heuristic"
config.MinSilenceDuration = 500 * time.Millisecond

provider, err := turndetection.NewProvider(ctx, "heuristic", config)
if err != nil {
    log.Fatal(err)
}

// Detect turn end
audio := []byte{...} // Your audio data
turnEnd, err := provider.DetectTurn(ctx, audio)
if err != nil {
    log.Fatal(err)
}

if turnEnd {
    fmt.Println("Turn end detected!")
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
    MinSilenceDuration time.Duration // Minimum silence duration
    MinTurnLength     int           // Minimum turn length in characters
    MaxTurnLength     int           // Maximum turn length in characters
    SentenceEndMarkers string       // Characters indicating sentence endings
    QuestionMarkers   []string      // Phrases indicating questions
    Threshold         float64       // Detection threshold (0.0-1.0)
    ModelPath         string        // Path to model file (for ML-based providers)
    Timeout           time.Duration // Processing timeout
}
```

### Provider-Specific Configuration

Each provider extends the base config with provider-specific settings. See provider documentation for details:
- [Heuristic Configuration](./providers/heuristic/README.md)
- [ONNX Configuration](./providers/onnx/README.md)

## Error Handling

The Turn Detection package uses structured error handling with error codes:

```go
if err != nil {
    var turnErr *turndetection.TurnDetectionError
    if errors.As(err, &turnErr) {
        switch turnErr.Code {
        case turndetection.ErrCodeTimeout:
            // Processing timeout
        case turndetection.ErrCodeModelLoadFailed:
            // Model loading failed
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
- `ErrCodeProcessingError`: Processing error

## Observability

### Metrics

The Turn Detection package emits OTEL metrics:

- `turndetection.detections.total`: Total turn detections
- `turndetection.turns.detected`: Turns detected
- `turndetection.turns.not_detected`: Turns not detected
- `turndetection.errors.total`: Total errors
- `turndetection.detection.latency`: Detection latency histogram

### Tracing

All operations create OpenTelemetry spans with attributes:
- `provider`: Provider name
- `threshold`: Detection threshold
- `silence_duration`: Silence duration (if applicable)

## Provider Registry

Providers are automatically registered via `init()` functions. You can also manually register custom providers:

```go
registry := turndetection.GetRegistry()
registry.Register("custom-provider", func(config *turndetection.Config) (turndetectioniface.TurnDetector, error) {
    return NewCustomProvider(config)
})
```

## Testing

The package includes comprehensive test utilities:

```go
import "github.com/lookatitude/beluga-ai/pkg/voice/turndetection"

// Create mock provider
mockProvider := turndetection.NewAdvancedMockTurnDetector("test",
    turndetection.WithTurnResults(true, false, true),
    turndetection.WithProcessingDelay(10*time.Millisecond),
)

// Use in tests
turnEnd, err := mockProvider.DetectTurn(ctx, audio)
```

## Examples

See the [examples directory](../../../examples/voice/turndetection/) for complete usage examples.

## Performance

- **Latency**: Sub-5ms for heuristic, sub-20ms for ONNX
- **Throughput**: Supports 1000+ detections per second
- **Concurrency**: Thread-safe, supports concurrent requests

## License

Part of the Beluga AI Framework. See main LICENSE file.

