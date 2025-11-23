---
title: Text-to-Speech (TTS)
sidebar_position: 3
---

# Text-to-Speech (TTS)

The TTS package provides interfaces and implementations for converting text to speech audio using various TTS providers.

## Overview

The TTS package follows the Beluga AI Framework design patterns, providing:

- **Provider abstraction**: Unified interface for multiple TTS providers
- **Streaming support**: Real-time audio generation with streaming
- **Voice customization**: Multiple voices, speeds, pitches, and styles
- **Observability**: OTEL metrics and tracing for all operations
- **Error handling**: Comprehensive error codes and retry logic
- **Configuration**: Flexible configuration with validation

## Supported Providers

### OpenAI TTS

- **Features**: High-quality voices, multiple models, streaming
- **Models**: TTS-1, TTS-1-HD
- **Voices**: Alloy, Echo, Fable, Onyx, Nova, Shimmer
- **Best for**: High-quality, natural-sounding speech
- **Streaming**: ✅ Yes
- **Languages**: English

### Google Cloud Text-to-Speech

- **Features**: Natural voices, SSML support, multiple languages
- **Models**: Standard, Wavenet, Neural2
- **Voices**: 100+ voices across 40+ languages
- **Best for**: Multi-language support, SSML features
- **Streaming**: ✅ Yes
- **Languages**: 40+ languages

### Azure Speech Services

- **Features**: Neural voices, SSML support, style control
- **Models**: Neural, Standard
- **Voices**: 200+ neural voices
- **Best for**: Enterprise deployments, style control
- **Streaming**: ✅ Yes
- **Languages**: 100+ languages

### ElevenLabs

- **Features**: Premium voices, voice cloning, emotion control
- **Models**: Eleven Multilingual v2, Eleven Monolingual v1
- **Voices**: 100+ premium voices, custom voices
- **Best for**: Premium quality, voice cloning
- **Streaming**: ✅ Yes
- **Languages**: 29 languages

## Quick Start

### Basic Usage

```go
import (
    "context"
    "os"
    "log"
    
    "github.com/lookatitude/beluga-ai/pkg/voice/tts"
    "github.com/lookatitude/beluga-ai/pkg/voice/tts/providers/openai"
)

func main() {
    ctx := context.Background()
    
    // Create TTS provider using factory
    config := tts.DefaultConfig()
    config.Provider = "openai"
    config.APIKey = os.Getenv("OPENAI_API_KEY")
    config.Model = "tts-1"
    config.Voice = "alloy"
    
    provider, err := tts.NewProvider(ctx, "openai", config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Generate speech
    text := "Hello, this is a test of text-to-speech."
    audio, err := provider.GenerateSpeech(ctx, text)
    if err != nil {
        log.Fatal(err)
    }
    
    // Use audio data (save to file, send over network, etc.)
    fmt.Printf("Generated %d bytes of audio\n", len(audio))
}
```

### Direct Provider Usage

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/voice/tts/providers/openai"
)

// Create provider directly
ttsProvider, err := openai.NewOpenAITTS(ctx, openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "tts-1",
    Voice:  "alloy",
})
```

### Streaming Usage

```go
// Generate speech with streaming
reader, err := provider.StreamGenerate(ctx, text)
if err != nil {
    log.Fatal(err)
}

// Read audio chunks as they're generated
buffer := make([]byte, 4096)
for {
    n, err := reader.Read(buffer)
    if err == io.EOF {
        break
    }
    if err != nil {
        log.Fatal(err)
    }
    
    // Process audio chunk (send to client, save to file, etc.)
    processAudioChunk(buffer[:n])
}
```

## Configuration

### Base Configuration

```go
type Config struct {
    Provider        string        // Provider name: "openai", "google", "azure", "elevenlabs"
    APIKey          string        // API key for authentication
    BaseURL         string        // Custom API endpoint (optional)
    Model           string        // Model name (provider-specific)
    Voice           string        // Voice name (provider-specific)
    Language        string        // Language code (ISO 639-1)
    Speed           float64       // Speech speed (0.25-4.0, default: 1.0)
    Pitch           float64       // Voice pitch (provider-specific)
    Timeout         time.Duration // Request timeout (default: 30s)
    SampleRate      int           // Audio sample rate (provider-specific)
    Format          string        // Audio format: "mp3", "wav", "pcm"
    EnableStreaming bool          // Enable streaming support (default: true)
    MaxRetries      int           // Maximum retry attempts (default: 3)
    RetryDelay      time.Duration // Retry delay (default: 1s)
}
```

### Voice Selection

```go
// OpenAI voices
config.Voice = "alloy"  // Neutral, balanced
config.Voice = "echo"   // Warm, friendly
config.Voice = "fable"   // Expressive, dramatic
config.Voice = "onyx"    // Deep, authoritative
config.Voice = "nova"    // Bright, energetic
config.Voice = "shimmer" // Soft, gentle

// Google Cloud voices
config.Voice = "en-US-Wavenet-D"  // Male, US English
config.Voice = "en-US-Neural2-C"  // Female, US English

// Azure voices
config.Voice = "en-US-AriaNeural"  // Female, US English
config.Voice = "en-US-GuyNeural"   // Male, US English
```

## Error Handling

The TTS package uses structured error handling with error codes:

```go
import (
    "errors"
    "github.com/lookatitude/beluga-ai/pkg/voice/tts"
)

if err != nil {
    var ttsErr *tts.TTSError
    if errors.As(err, &ttsErr) {
        switch ttsErr.Code {
        case tts.ErrCodeNetworkError:
            // Retryable network error
        case tts.ErrCodeAuthentication:
            // Authentication failed - check API key
        case tts.ErrCodeRateLimit:
            // Rate limit exceeded - wait and retry
        case tts.ErrCodeInvalidText:
            // Invalid text input - check text length/format
        }
    }
}
```

### Error Codes

| Code | Description | Retryable |
|------|-------------|-----------|
| `ErrCodeInvalidConfig` | Invalid configuration | ❌ No |
| `ErrCodeNetworkError` | Network connectivity issue | ✅ Yes |
| `ErrCodeTimeout` | Request timeout | ✅ Yes |
| `ErrCodeRateLimit` | Rate limit exceeded | ✅ Yes (with delay) |
| `ErrCodeAuthentication` | Authentication failed | ❌ No |
| `ErrCodeInvalidText` | Invalid text input | ❌ No |
| `ErrCodeStreamError` | Streaming error | ✅ Yes |

## Observability

### Metrics

The TTS package emits OTEL metrics:

- `tts.generations.total`: Total generation requests (counter)
- `tts.generations.successful`: Successful generations (counter)
- `tts.generations.failed`: Failed generations (counter)
- `tts.errors.total`: Total errors by error code (counter)
- `tts.generation.latency`: Generation latency (histogram)
- `tts.audio.size`: Generated audio size (histogram)

### Tracing

All operations create OpenTelemetry spans with attributes:

- `provider`: Provider name
- `model`: Model name
- `voice`: Voice name
- `language`: Language code
- `text_length`: Input text length
- `audio_size`: Generated audio size

## Performance

### Latency

- **Streaming**: First chunk in 100-300ms
- **Non-streaming**: 500ms - 2s depending on text length
- **Long text**: 2-5s for paragraphs

### Best Practices

1. **Use streaming for real-time applications**: Lower perceived latency
2. **Cache common phrases**: Reduce API calls for repeated text
3. **Choose appropriate voice**: Match voice to application context
4. **Monitor audio quality**: Ensure sample rate and format match requirements
5. **Implement retry logic**: Handle transient errors gracefully

## Examples

See the [examples directory](https://github.com/lookatitude/beluga-ai/tree/main/examples/voice) for complete usage examples.

## API Reference

For complete API documentation, see the [TTS API Reference](../api/packages/voice/tts).

## Next Steps

- [Speech-to-Text (STT)](./stt) - Convert speech to text
- [Voice Activity Detection (VAD)](./vad) - Detect voice in audio
- [Session Management](./session) - Manage voice interactions

