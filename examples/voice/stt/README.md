# STT (Speech-to-Text) Example

This example demonstrates how to use the STT package for speech-to-text transcription.

## Prerequisites

- Go 1.21+
- For real providers: API keys (Deepgram, Azure, Google, OpenAI, etc.)

## Running the Example

```bash
go run main.go
```

## What This Example Shows

1. Creating STT configuration with default settings
2. Creating an STT provider using the factory pattern
3. Transcribing audio data
4. Starting streaming sessions for real-time transcription

## Configuration Options

- `Provider`: Provider name (e.g., "deepgram", "azure", "google", "openai", "whisper", "mock")
- `APIKey`: Provider API key (required for real providers)
- `Language`: Language code (e.g., "en-US", "es-ES")
- `SampleRate`: Audio sample rate in Hz
- `Channels`: Number of audio channels

## Using Real Providers

To use a real STT provider:

```go
config := stt.DefaultConfig()
config.Provider = "deepgram"
config.APIKey = os.Getenv("DEEPGRAM_API_KEY")
config.Language = "en-US"

provider, err := stt.NewProvider(ctx, config.Provider, config)
```

## See Also

- [STT Package Documentation](../../../pkg/voice/stt/README.md)
- [Voice Session Example](../simple/main.go)
