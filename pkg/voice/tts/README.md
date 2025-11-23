# TTS Package

The TTS (Text-to-Speech) package provides interfaces and implementations for converting text to speech audio using various TTS providers.

## Overview

The TTS package follows the Beluga AI Framework design patterns, providing:
- **Provider abstraction**: Unified interface for multiple TTS providers
- **Streaming support**: Real-time audio generation with streaming
- **Voice customization**: Multiple voices, speeds, pitches, and styles
- **Observability**: OTEL metrics and tracing for all operations
- **Error handling**: Comprehensive error codes and retry logic
- **Configuration**: Flexible configuration with validation

## Supported Providers

- **OpenAI**: TTS-1 and TTS-1-HD models with multiple voices
- **Google Cloud Text-to-Speech**: Standard and Wavenet voices with SSML support
- **Azure Speech Services**: Neural voices with SSML and style support
- **ElevenLabs**: High-quality voices with voice cloning support

## Quick Start

### Basic Usage

```go
import (
    "context"
    "os"
    
    "github.com/lookatitude/beluga-ai/pkg/voice/tts/providers/openai"
)

func main() {
    ctx := context.Background()
    
    // Create TTS provider
    ttsProvider, err := openai.NewOpenAITTS(ctx, openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Generate speech
    text := "Hello, this is a test of text-to-speech."
    audio, err := ttsProvider.GenerateSpeech(ctx, text)
    if err != nil {
        log.Fatal(err)
    }
    
    // Use audio data (e.g., save to file, send over network)
    fmt.Printf("Generated %d bytes of audio\n", len(audio))
}
```

// Create TTS provider
config := tts.DefaultConfig()
config.Provider = "openai"
config.APIKey = "your-api-key"
config.Model = "tts-1"
config.Voice = "alloy"

provider, err := tts.NewProvider(ctx, "openai", config)
if err != nil {
    log.Fatal(err)
}

// Generate speech
text := "Hello, world!"
audio, err := provider.GenerateSpeech(ctx, text)
if err != nil {
    log.Fatal(err)
}

// Save or play audio
// ...
```

### Streaming Usage

```go
// Generate speech with streaming
reader, err := provider.StreamGenerate(ctx, text)
if err != nil {
    log.Fatal(err)
}

// Read audio chunks
buffer := make([]byte, 4096)
for {
    n, err := reader.Read(buffer)
    if err == io.EOF {
        break
    }
    if err != nil {
        log.Fatal(err)
    }
    // Process audio chunk
    processAudio(buffer[:n])
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
    Speed           float64       // Speech speed (0.25-4.0)
    Pitch           float64       // Pitch adjustment (-20.0 to 20.0)
    Volume          float64       // Volume (0.0-1.0)
    Timeout         time.Duration // Request timeout
    SampleRate      int           // Audio sample rate
    BitDepth        int           // Audio bit depth
    EnableStreaming bool          // Enable streaming support
    EnableSSML      bool          // Enable SSML support
    MaxRetries      int           // Maximum retry attempts
    RetryDelay      time.Duration // Retry delay
    RetryBackoff    float64       // Retry backoff multiplier
}
```

### Provider-Specific Configuration

Each provider extends the base config with provider-specific settings. See provider documentation for details:
- [OpenAI Configuration](./providers/openai/README.md)
- [Google Configuration](./providers/google/README.md)
- [Azure Configuration](./providers/azure/README.md)
- [ElevenLabs Configuration](./providers/elevenlabs/README.md)

## Error Handling

The TTS package uses structured error handling with error codes:

```go
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
        }
    }
}
```

### Error Codes

- `ErrCodeInvalidConfig`: Invalid configuration
- `ErrCodeNetworkError`: Network connectivity issue
- `ErrCodeTimeout`: Request timeout
- `ErrCodeRateLimit`: Rate limit exceeded
- `ErrCodeAuthentication`: Authentication failed
- `ErrCodeStreamError`: Streaming error
- `ErrCodeStreamClosed`: Stream closed
- `ErrCodeInvalidSSML`: Invalid SSML format

## Observability

### Metrics

The TTS package emits OTEL metrics:

- `tts.generations.total`: Total TTS generations
- `tts.generations.successful`: Successful generations
- `tts.generations.failed`: Failed generations
- `tts.errors.total`: Total errors
- `tts.streams.total`: Total streaming sessions
- `tts.generation.latency`: Generation latency histogram
- `tts.stream.latency`: Stream latency histogram
- `tts.streams.active`: Active streaming sessions

### Tracing

All operations create OpenTelemetry spans with attributes:
- `provider`: Provider name
- `model`: Model name
- `voice`: Voice name
- `text_length`: Text length

## Provider Registry

Providers are automatically registered via `init()` functions. You can also manually register custom providers:

```go
registry := tts.GetRegistry()
registry.Register("custom-provider", func(config *tts.Config) (ttsiface.TTSProvider, error) {
    return NewCustomProvider(config)
})
```

## Testing

The package includes comprehensive test utilities:

```go
import "github.com/lookatitude/beluga-ai/pkg/voice/tts"

// Create mock provider
mockProvider := tts.NewAdvancedMockTTSProvider("test",
    tts.WithAudioResponses([]byte("audio data")),
    tts.WithStreamingDelay(10*time.Millisecond),
)

// Use in tests
audio, err := mockProvider.GenerateSpeech(ctx, "Hello")
```

## Examples

See the [examples directory](../../../examples/voice/tts/) for complete usage examples.

## Performance

- **Latency**: Sub-500ms for standard generation
- **Throughput**: Supports 100+ requests per second
- **Concurrency**: Thread-safe, supports concurrent requests

## License

Part of the Beluga AI Framework. See main LICENSE file.

