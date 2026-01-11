# Transport Example

This example demonstrates how to use the Transport package for audio data transmission between components.

## Prerequisites

- Go 1.21+

## Running the Example

```bash
go run main.go
```

## What This Example Shows

1. Creating Transport configuration with default settings
2. Creating a Transport provider using the factory pattern
3. Sending audio data through the transport
4. Receiving audio data from the transport
5. Closing connections properly

## Configuration Options

- `Provider`: Provider name (e.g., "websocket", "webrtc", "mock")
- `BufferSize`: Buffer size for audio data
- `Timeout`: Connection timeout duration
- `URL`: Connection URL (for WebSocket, WebRTC, etc.)

## Using Real Providers

To use a real Transport provider:

```go
config := transport.DefaultConfig()
config.Provider = "websocket"
config.URL = "ws://localhost:8080/audio"
config.BufferSize = 4096
config.Timeout = 30 * time.Second

provider, err := transport.NewProvider(ctx, config.Provider, config)
```

## Use Cases

- WebSocket-based audio streaming
- WebRTC peer-to-peer audio transmission
- Custom transport protocols
- Audio routing between voice pipeline components

## See Also

- [Transport Package Documentation](../../../pkg/voice/transport/README.md)
- [Voice Session Example](../simple/main.go)
