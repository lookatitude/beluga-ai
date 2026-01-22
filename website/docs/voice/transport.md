---
title: Audio Transport
sidebar_position: 6
---

# Audio Transport

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

### WebRTC

- **Features**: Peer-to-peer, low latency, STUN/TURN support
- **Best for**: Direct peer connections, low latency requirements
- **Latency**: \<100ms
- **Protocol**: WebRTC (UDP)

### WebSocket

- **Features**: Simple, reliable, keepalive support
- **Best for**: Server-based applications, firewall-friendly
- **Latency**: 100-200ms
- **Protocol**: WebSocket (TCP)

## Quick Start

### Basic Usage

```go
import (
    "context"
    "log"
    
    "github.com/lookatitude/beluga-ai/pkg/voice/transport"
    "github.com/lookatitude/beluga-ai/pkg/voice/transport/providers/websocket"
)

func main() {
    ctx := context.Background()
    
    // Create Transport provider
    config := transport.DefaultConfig()
    config.Provider = "websocket"
    config.URL = "wss://example.com/audio"
    config.SampleRate = 16000
    config.Channels = 1
    config.Codec = "pcm"
    
    provider, err := transport.NewProvider(ctx, "websocket", config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Set callback for received audio
    provider.OnAudioReceived(func(audio []byte) {
        // Handle received audio
        processAudio(audio)
    })
    
    // Send audio
    audio := []byte{/* your audio data */}
    err = provider.SendAudio(ctx, audio)
    if err != nil {
        log.Fatal(err)
    }
    
    // Close connection
    err = provider.Close(ctx)
    if err != nil {
        log.Fatal(err)
    }
}
```

## Configuration

### Base Configuration

```go
type Config struct {
    Provider          string        // Provider name: "webrtc", "websocket"
    URL               string        // Connection URL
    SampleRate        int           // Audio sample rate in Hz
    Channels          int           // Number of audio channels (1 or 2)
    BitDepth          int           // Audio bit depth (8, 16, 24, 32)
    Codec             string        // Audio codec ("pcm", "opus", "g711")
    ConnectTimeout    time.Duration // Connection timeout
    ReconnectAttempts int           // Maximum reconnection attempts
    ReconnectDelay    time.Duration // Delay between reconnection attempts
    SendBufferSize    int           // Send buffer size
    ReceiveBufferSize int           // Receive buffer size
    Timeout           time.Duration // Operation timeout
}
```

## Error Handling

The Transport package uses structured error handling:

```go
import (
    "errors"
    "github.com/lookatitude/beluga-ai/pkg/voice/transport"
)

if err != nil {
    var transErr *transport.TransportError
    if errors.As(err, &transErr) {
        switch transErr.Code {
        case transport.ErrCodeConnectionError:
            // Connection error - may retry
        case transport.ErrCodeTimeout:
            // Timeout - may retry
        }
    }
}
```

## Observability

### Metrics

- `transport.connections.total`: Total connections (counter)
- `transport.connections.active`: Active connections (gauge)
- `transport.audio.sent`: Audio bytes sent (counter)
- `transport.audio.received`: Audio bytes received (counter)
- `transport.latency`: Transport latency (histogram)

## Performance

- **Latency**: \<200ms for WebRTC, 100-200ms for WebSocket
- **Throughput**: 1000+ audio chunks per second
- **Reliability**: Automatic reconnection on failure

## API Reference

For complete API documentation, see the [Transport API Reference](../api/packages/voice/transport).

## Next Steps

- [Noise Cancellation](./noise) - Remove noise from audio
- [Session Management](./session) - Manage voice interactions

