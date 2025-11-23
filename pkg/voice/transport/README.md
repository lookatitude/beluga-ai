# Transport Package

The Transport package provides interfaces and implementations for audio transport over various protocols.

## Overview

The Transport package follows the Beluga AI Framework design patterns, providing:
- **Provider abstraction**: Unified interface for multiple transport protocols
- **Real-time audio**: Low-latency audio streaming
- **Connection management**: Automatic reconnection and error handling
- **Observability**: OTEL metrics and tracing for all operations
- **Error handling**: Comprehensive error codes and retry logic
- **Configuration**: Flexible configuration with validation

## Supported Providers

- **WebRTC**: Peer-to-peer audio transport with STUN/TURN support
- **WebSocket**: WebSocket-based audio transport with keepalive

## Quick Start

### Basic Usage

```go
import (
    "context"
    
    "github.com/lookatitude/beluga-ai/pkg/voice/transport/providers/websocket"
)

func main() {
    ctx := context.Background()
    
    // Create Transport provider
config := transport.DefaultConfig()
config.Provider = "webrtc"
config.URL = "wss://example.com/signaling"

provider, err := transport.NewProvider(ctx, "webrtc", config)
if err != nil {
    log.Fatal(err)
}

// Connect
// Note: Connect is not part of the interface but may be implemented by providers
// For WebRTC, you would establish connection via signaling first

// Send audio
audio := []byte{...} // Your audio data
err = provider.SendAudio(ctx, audio)
if err != nil {
    log.Fatal(err)
}

// Receive audio
audioCh := provider.ReceiveAudio()
for audio := range audioCh {
    // Process received audio
    processAudio(audio)
}

// Set callback for received audio
provider.OnAudioReceived(func(audio []byte) {
    // Handle received audio
    processAudio(audio)
})

// Close connection
err = provider.Close()
if err != nil {
    log.Fatal(err)
}
```

## Configuration

### Base Configuration

```go
type Config struct {
    Provider          string        // Provider name: "webrtc", "websocket"
    URL               string        // Connection URL
    SampleRate        int           // Audio sample rate in Hz
    Channels          int           // Number of audio channels
    BitDepth          int           // Audio bit depth
    Codec             string        // Audio codec ("pcm", "opus", "g711")
    ConnectTimeout    time.Duration // Connection timeout
    ReconnectAttempts int           // Maximum reconnection attempts
    ReconnectDelay    time.Duration // Delay between reconnection attempts
    SendBufferSize    int           // Send buffer size
    ReceiveBufferSize int           // Receive buffer size
    Timeout           time.Duration // Operation timeout
}
```

### Provider-Specific Configuration

Each provider extends the base config with provider-specific settings. See provider documentation for details:
- [WebRTC Configuration](./providers/webrtc/README.md)
- [WebSocket Configuration](./providers/websocket/README.md)

## Error Handling

The Transport package uses structured error handling with error codes:

```go
if err != nil {
    var transportErr *transport.TransportError
    if errors.As(err, &transportErr) {
        switch transportErr.Code {
        case transport.ErrCodeNotConnected:
            // Transport not connected - need to connect first
        case transport.ErrCodeConnectionFailed:
            // Connection failed - retryable
        case transport.ErrCodeTimeout:
            // Operation timeout - retryable
        }
    }
}
```

### Error Codes

- `ErrCodeInvalidConfig`: Invalid configuration
- `ErrCodeNotConnected`: Transport not connected
- `ErrCodeConnectionFailed`: Connection failed
- `ErrCodeConnectionTimeout`: Connection timeout
- `ErrCodeDisconnected`: Connection disconnected
- `ErrCodeNetworkError`: Network error
- `ErrCodeProtocolError`: Protocol error
- `ErrCodeTimeout`: Operation timeout

## Observability

### Metrics

The Transport package emits OTEL metrics:

- `transport.connections.total`: Total connections
- `transport.disconnections.total`: Total disconnections
- `transport.audio.sent`: Audio packets sent
- `transport.audio.received`: Audio packets received
- `transport.bytes.sent`: Bytes sent
- `transport.bytes.received`: Bytes received
- `transport.errors.total`: Total errors
- `transport.connection.latency`: Connection latency histogram
- `transport.connections.active`: Active connections

### Tracing

All operations create OpenTelemetry spans with attributes:
- `provider`: Provider name
- `url`: Connection URL
- `codec`: Audio codec
- `sample_rate`: Sample rate

## Provider Registry

Providers are automatically registered via `init()` functions. You can also manually register custom providers:

```go
registry := transport.GetRegistry()
registry.Register("custom-provider", func(config *transport.Config) (transportiface.Transport, error) {
    return NewCustomTransport(config)
})
```

## Testing

The package includes comprehensive test utilities:

```go
import "github.com/lookatitude/beluga-ai/pkg/voice/transport"

// Create mock transport
mockTransport := transport.NewAdvancedMockTransport("test",
    transport.WithAudioData([]byte{1, 2, 3}, []byte{4, 5, 6}),
    transport.WithConnected(true),
    transport.WithProcessingDelay(10*time.Millisecond),
)

// Use in tests
err := mockTransport.SendAudio(ctx, audio)
audioCh := mockTransport.ReceiveAudio()
```

## Examples

See the [examples directory](../../../examples/voice/transport/) for complete usage examples.

## Performance

- **Latency**: Sub-50ms for WebRTC, sub-100ms for WebSocket
- **Throughput**: Supports 1000+ packets per second
- **Concurrency**: Thread-safe, supports concurrent operations

## License

Part of the Beluga AI Framework. See main LICENSE file.

