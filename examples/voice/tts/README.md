# TTS (Text-to-Speech) Example

This example demonstrates how to use the TTS package for text-to-speech synthesis.

## Prerequisites

- Go 1.21+
- For real providers: API keys (Deepgram, Azure, Google, OpenAI, ElevenLabs, etc.)

## Running the Example

```bash
go run main.go
```

## What This Example Shows

1. Creating TTS configuration with default settings
2. Creating a TTS provider using the factory pattern
3. Generating speech audio from text
4. Starting streaming sessions for real-time synthesis

## Configuration Options

- `Provider`: Provider name (e.g., "deepgram", "azure", "google", "openai", "elevenlabs", "mock")
- `APIKey`: Provider API key (required for real providers)
- `Voice`: Voice name or ID (provider-specific)
- `Language`: Language code (e.g., "en-US", "es-ES")
- `SampleRate`: Audio sample rate in Hz
- `Channels`: Number of audio channels

## Using Real Providers

To use a real TTS provider:

```go
config := tts.DefaultConfig()
config.Provider = "deepgram"
config.APIKey = os.Getenv("DEEPGRAM_API_KEY")
config.Voice = "nova"
config.Language = "en-US"

provider, err := tts.NewProvider(ctx, config.Provider, config)
```

## See Also

- [TTS Package Documentation](../../../pkg/voice/tts/README.md)
- [Voice Session Example](../simple/main.go)
