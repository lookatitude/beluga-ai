# STT Package

The STT (Speech-to-Text) package provides interfaces and implementations for converting audio to text using various speech recognition providers.

## Overview

The STT package follows the Beluga AI Framework design patterns, providing:
- **Provider abstraction**: Unified interface for multiple STT providers
- **Streaming support**: Real-time transcription with interim results
- **REST fallback**: Automatic fallback to REST API when streaming unavailable
- **Observability**: OTEL metrics and tracing for all operations
- **Error handling**: Comprehensive error codes and retry logic
- **Configuration**: Flexible configuration with validation

## Supported Providers

- **Deepgram**: WebSocket streaming with REST fallback
- **Google Cloud Speech-to-Text**: REST API with gRPC streaming (streaming requires additional setup)
- **Azure Speech Services**: WebSocket streaming with REST fallback
- **OpenAI Whisper**: REST API (streaming not supported by API)

## Quick Start

### Basic Usage

```go
import (
    "context"
    "os"
    
    "github.com/lookatitude/beluga-ai/pkg/voice/stt"
    "github.com/lookatitude/beluga-ai/pkg/voice/stt/providers/deepgram"
)

func main() {
    ctx := context.Background()
    
    // Create STT provider
    sttProvider, err := deepgram.NewDeepgramSTT(ctx, deepgram.Config{
        APIKey: os.Getenv("DEEPGRAM_API_KEY"),
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Transcribe audio
    audio := []byte{/* your audio data */}
    transcript, err := sttProvider.Transcribe(ctx, audio)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Transcript:", transcript)
}

// Create STT provider
config := stt.DefaultConfig()
config.Provider = "deepgram"
config.APIKey = "your-api-key"
config.Model = "nova-2"

provider, err := stt.NewProvider(ctx, "deepgram", config)
if err != nil {
    log.Fatal(err)
}

// Transcribe audio
audio := []byte{...} // Your audio data
text, err := provider.Transcribe(ctx, audio)
if err != nil {
    log.Fatal(err)
}

fmt.Println("Transcription:", text)
```

### Streaming Usage

```go
// Start streaming session
session, err := provider.StartStreaming(ctx)
if err != nil {
    log.Fatal(err)
}
defer session.Close()

// Send audio chunks
go func() {
    for audioChunk := range audioChunks {
        if err := session.SendAudio(ctx, audioChunk); err != nil {
            log.Printf("Error sending audio: %v", err)
        }
    }
}()

// Receive transcripts
for result := range session.ReceiveTranscript() {
    if result.Error != nil {
        log.Printf("Error: %v", result.Error)
        continue
    }
    
    fmt.Printf("Transcript: %s (Final: %v)\n", result.Text, result.IsFinal)
}
```

## Configuration

### Base Configuration

```go
type Config struct {
    Provider        string        // Provider name: "deepgram", "google", "azure", "openai"
    APIKey          string        // API key for authentication
    BaseURL         string        // Custom API endpoint (optional)
    Model           string        // Model name (provider-specific)
    Language        string        // Language code (ISO 639-1)
    Timeout         time.Duration // Request timeout
    SampleRate      int           // Audio sample rate (8000, 16000, 48000)
    Channels        int           // Audio channels (1 or 2)
    EnableStreaming bool          // Enable streaming support
    MaxRetries      int           // Maximum retry attempts
    RetryDelay      time.Duration // Retry delay
    RetryBackoff    float64       // Retry backoff multiplier
}
```

### Provider-Specific Configuration

Each provider extends the base config with provider-specific settings. See provider documentation for details:
- [Deepgram Configuration](./providers/deepgram/README.md)
- [Google Configuration](./providers/google/README.md)
- [Azure Configuration](./providers/azure/README.md)
- [OpenAI Configuration](./providers/openai/README.md)

## Error Handling

The STT package uses structured error handling with error codes:

```go
if err != nil {
    var sttErr *stt.STTError
    if errors.As(err, &sttErr) {
        switch sttErr.Code {
        case stt.ErrCodeNetworkError:
            // Retryable network error
        case stt.ErrCodeAuthentication:
            // Authentication failed - check API key
        case stt.ErrCodeRateLimit:
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

## Observability

### Metrics

The STT package emits OTEL metrics:

- `stt.transcriptions.total`: Total transcription requests
- `stt.transcriptions.successful`: Successful transcriptions
- `stt.transcriptions.failed`: Failed transcriptions
- `stt.errors.total`: Total errors
- `stt.streams.total`: Total streaming sessions
- `stt.transcription.latency`: Transcription latency histogram
- `stt.stream.latency`: Stream latency histogram
- `stt.streams.active`: Active streaming sessions

### Tracing

All operations create OpenTelemetry spans with attributes:
- `provider`: Provider name
- `model`: Model name
- `language`: Language code
- `audio_size`: Audio data size

## Provider Registry

Providers are automatically registered via `init()` functions. You can also manually register custom providers:

```go
registry := stt.GetRegistry()
registry.Register("custom-provider", func(config *stt.Config) (sttiface.STTProvider, error) {
    return NewCustomProvider(config)
})
```

## Testing

The package includes comprehensive test utilities:

```go
import "github.com/lookatitude/beluga-ai/pkg/voice/stt"

// Create mock provider
mockProvider := stt.NewAdvancedMockSTTProvider("test",
    stt.WithTranscriptions("Hello world"),
    stt.WithStreamingDelay(10*time.Millisecond),
)

// Use in tests
text, err := mockProvider.Transcribe(ctx, audio)
```

## Examples

See the [examples directory](../../../examples/voice/stt/) for complete usage examples.

## Performance

- **Latency**: Sub-200ms for streaming transcription
- **Throughput**: Supports 1000+ audio chunks per second
- **Concurrency**: Thread-safe, supports concurrent requests

## License

Part of the Beluga AI Framework. See main LICENSE file.

