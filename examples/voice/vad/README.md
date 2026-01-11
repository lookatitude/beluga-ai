# VAD (Voice Activity Detection) Example

This example demonstrates how to use the VAD package for detecting when speech is present in audio streams.

## Prerequisites

- Go 1.21+

## Running the Example

```bash
go run main.go
```

## What This Example Shows

1. Creating VAD configuration with default settings
2. Creating a VAD provider using the factory pattern
3. Detecting voice activity in audio frames
4. Starting streaming sessions for real-time detection

## Configuration Options

- `Provider`: Provider name (e.g., "webrtc", "silero", "mock")
- `FrameSize`: Audio frame size in milliseconds
- `SilenceThreshold`: Threshold for silence detection (0.0-1.0)
- `MinSpeechDuration`: Minimum duration of speech to trigger detection

## Using Real Providers

To use a real VAD provider:

```go
config := vad.DefaultConfig()
config.Provider = "webrtc"
config.FrameSize = 30 * time.Millisecond
config.SilenceThreshold = 0.5

provider, err := vad.NewProvider(ctx, config.Provider, config)
```

## Use Cases

- Filtering silence from audio streams
- Triggering transcription only when speech is detected
- Reducing API costs by processing only speech segments
- Improving audio quality by removing background noise

## See Also

- [VAD Package Documentation](../../../pkg/voice/vad/README.md)
- [Voice Session Example](../simple/main.go)
