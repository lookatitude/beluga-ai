# Turn Detection Example

This example demonstrates how to use the Turn Detection package for identifying when a speaker has finished speaking.

## Prerequisites

- Go 1.21+

## Running the Example

```bash
go run main.go
```

## What This Example Shows

1. Creating Turn Detection configuration with default settings
2. Creating a Turn Detection provider using the factory pattern
3. Detecting turn completion in audio streams
4. Starting streaming sessions for real-time detection

## Configuration Options

- `Provider`: Provider name (e.g., "silence", "energy", "mock")
- `SilenceDuration`: Duration of silence to consider turn complete
- `MinSpeechDuration`: Minimum duration of speech before turn can be detected
- `EnergyThreshold`: Energy threshold for energy-based detection

## Using Real Providers

To use a real Turn Detection provider:

```go
config := turndetection.DefaultConfig()
config.Provider = "silence"
config.SilenceDuration = 500 * time.Millisecond
config.MinSpeechDuration = 200 * time.Millisecond

provider, err := turndetection.NewProvider(ctx, config.Provider, config)
```

## Use Cases

- Natural conversation flow in voice agents
- Preventing interruptions while user is speaking
- Triggering agent responses at appropriate times
- Managing turn-taking in multi-party conversations

## See Also

- [Turn Detection Package Documentation](../../../pkg/voice/turndetection/README.md)
- [Voice Session Example](../simple/main.go)
