---
title: "Transport Providers — Voice I/O"
description: "3 voice transport providers for bidirectional audio: Daily, LiveKit, Pipecat. WebRTC and WebSocket audio transport in Go with Beluga AI."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "voice transport, audio transport, LiveKit, Daily, WebRTC, WebSocket, Go voice pipeline, Beluga AI"
---

Beluga AI provides a unified `transport.AudioTransport` interface for bidirectional audio I/O between clients and the voice pipeline. Transport providers abstract the underlying protocol (WebSocket, WebRTC, server-to-server) so that voice processing logic remains transport-agnostic.

## How It Works

All transport providers implement the same interface:

```go
type AudioTransport interface {
    Recv(ctx context.Context) (<-chan voice.Frame, error)
    Send(ctx context.Context, frame voice.Frame) error
    AudioOut() io.Writer
    Close() error
}
```

You can instantiate any provider two ways:

**Via the registry** (recommended for dynamic configuration):

```go
import (
    "github.com/lookatitude/beluga-ai/voice/transport"
    _ "github.com/lookatitude/beluga-ai/voice/transport/providers/livekit"
)

t, err := transport.New("livekit", transport.Config{
    URL:   "wss://myapp.livekit.cloud",
    Token: "...",
})
```

**Via direct construction** (for compile-time type safety):

```go
import "github.com/lookatitude/beluga-ai/voice/transport/providers/livekit"

t, err := livekit.New(transport.Config{
    URL:   "wss://myapp.livekit.cloud",
    Token: "...",
})
```

## Configuration

All providers accept `transport.Config`:

| Field        | Type              | Description                              |
|--------------|-------------------|------------------------------------------|
| `URL`        | `string`          | Transport endpoint URL                   |
| `Token`      | `string`          | Authentication token                     |
| `SampleRate` | `int`             | Audio sample rate in Hz (default: 16000) |
| `Channels`   | `int`             | Audio channels: 1=mono, 2=stereo         |
| `Extra`      | `map[string]any`  | Provider-specific configuration          |

Functional options are also available:

```go
cfg := transport.Config{}
transport.WithURL("wss://...")(&cfg)
transport.WithToken("...")(&cfg)
transport.WithSampleRate(16000)(&cfg)
transport.WithChannels(1)(&cfg)
```

## Available Providers

| Provider | Registry Name | Protocol | Description |
|----------|---------------|----------|-------------|
| WebSocket | `websocket` | WebSocket | Built-in WebSocket transport |
| [Daily](/providers/transport/daily) | `daily` | WebRTC | Daily.co room-based audio transport |
| [LiveKit](/providers/transport/livekit) | `livekit` | WebRTC | LiveKit room-based audio transport |
| [Pipecat](/providers/transport/pipecat) | `pipecat` | WebSocket | Pipecat server audio transport |

## Built-in WebSocket Transport

The `transport` package includes a built-in WebSocket transport that requires no additional imports:

```go
import "github.com/lookatitude/beluga-ai/voice/transport"

t, err := transport.New("websocket", transport.Config{
    URL:        "ws://localhost:8080/audio",
    SampleRate: 16000,
    Channels:   1,
})
```

Or construct directly with WebSocket-specific options:

```go
t := transport.NewWebSocketTransport("ws://localhost:8080/audio",
    transport.WithWSSampleRate(16000),
    transport.WithWSChannels(1),
)
```

## Design Philosophy

LiveKit, Daily, and similar services are treated as **transports**, not framework dependencies. These services provide the real-time communication layer (WebRTC, WebSocket), while Beluga handles all STT, LLM, and TTS processing through its frame-based pipeline. This separation ensures that switching transports does not require changes to voice processing logic.

## Pipeline Integration

Use `AsVoiceTransport` to adapt an `AudioTransport` for the voice pipeline:

```go
audioTransport, err := transport.New("livekit", cfg)
if err != nil {
    log.Fatal(err)
}

voiceTransport := &transport.AsVoiceTransport{T: audioTransport}

pipe := voice.NewPipeline(
    voice.WithTransport(voiceTransport),
    voice.WithVAD(vad),
    voice.WithSTT(sttEngine),
    voice.WithLLM(model),
    voice.WithTTS(ttsEngine),
)
```

## Provider Discovery

List all registered transport providers at runtime:

```go
for _, name := range transport.List() {
    fmt.Println(name)
}
```

## Choosing a Provider

| Use Case | Recommended Provider | Reason |
|----------|---------------------|--------|
| Browser clients | `livekit` or `daily` | WebRTC handles NAT traversal and adaptive bitrate |
| Server-to-server | `websocket` | Simple protocol, no WebRTC overhead |
| Pipecat interop | `pipecat` | Native integration with Pipecat pipelines |
| Low-latency production | `livekit` | Global edge network with SFU routing |
| Quick prototyping | `websocket` | Built-in, zero external dependencies |
