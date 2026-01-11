# Noise Cancellation Example

This example demonstrates how to use the Noise Cancellation package for removing background noise from audio streams.

## Prerequisites

- Go 1.21+

## Running the Example

```bash
go run main.go
```

## What This Example Shows

1. Creating Noise Cancellation configuration with default settings
2. Creating a Noise Cancellation provider using the factory pattern
3. Canceling noise from audio data
4. Starting streaming sessions for real-time noise reduction

## Configuration Options

- `Provider`: Provider name (e.g., "rnnoise", "mock")
- `Aggressiveness`: Noise cancellation aggressiveness (0.0-1.0)
- `Quality`: Output quality setting

## Using Real Providers

To use a real Noise Cancellation provider:

```go
config := noise.DefaultConfig()
config.Provider = "rnnoise"
config.Aggressiveness = 0.5

provider, err := noise.NewProvider(ctx, config.Provider, config)
```

## Use Cases

- Improving speech recognition accuracy
- Enhancing audio quality in noisy environments
- Reducing background noise in voice calls
- Preprocessing audio before transcription

## See Also

- [Noise Cancellation Package Documentation](../../../pkg/voice/noise/README.md)
- [Voice Session Example](../simple/main.go)
