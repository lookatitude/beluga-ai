---
title: "Voice Transport"
description: "Transport layer for voice sessions: WebSocket, LiveKit, Daily, Pipecat"
---

## transport

```go
import "github.com/lookatitude/beluga-ai/voice/transport"
```

Package transport provides the audio transport interface and registry for the
Beluga AI voice pipeline. Transports handle bidirectional audio I/O between
clients and the voice pipeline, abstracting the underlying protocol
(WebSocket, LiveKit, Daily, etc.).

LiveKit is treated as a transport, not a framework dependency. LiveKit's
server provides WebRTC transport, while Beluga handles all STT/LLM/TTS
processing through the frame-based pipeline.

## Core Interface

The `AudioTransport` interface provides bidirectional audio I/O:

```go
type AudioTransport interface {
    Recv(ctx context.Context) (<-chan voice.Frame, error)
    Send(ctx context.Context, frame voice.Frame) error
    AudioOut() io.Writer
    Close() error
}
```

## Registry Pattern

Providers register via `Register` in their init() function and are created
with `New`. Use `List` to discover available providers.

```go
import _ "github.com/lookatitude/beluga-ai/voice/transport"

t, err := transport.New("websocket", transport.Config{URL: "ws://..."})
frames, err := t.Recv(ctx)
for frame := range frames {
    // process incoming audio frame
}
```

## Pipeline Integration

Use `AsVoiceTransport` to adapt an AudioTransport to the voice.Transport
interface expected by the [voice.VoicePipeline].

## Built-in Transport

The package includes a `WebSocketTransport` implementation registered as
"websocket". Configure it with `NewWebSocketTransport` and options
`WithWSSampleRate` and `WithWSChannels`.

## Configuration

The `Config` struct supports URL, authentication token, sample rate,
channel count, and provider-specific extras. Use functional options
`WithURL`, `WithToken`, `WithSampleRate`, and `WithChannels`.

## Available Providers

- websocket — Built-in WebSocket transport (voice/transport)
- livekit — LiveKit WebRTC rooms (voice/transport/providers/livekit)
- daily — Daily.co rooms (voice/transport/providers/daily)
- pipecat — Pipecat server (voice/transport/providers/pipecat)

---

## daily

```go
import "github.com/lookatitude/beluga-ai/voice/transport/providers/daily"
```

Package daily provides the Daily.co transport provider for the Beluga AI
voice pipeline. It implements the [transport.AudioTransport] interface for
bidirectional audio I/O through Daily.co rooms.

## Registration

This package registers itself as "daily" with the transport registry. Import
it with a blank identifier to enable:

```go
import _ "github.com/lookatitude/beluga-ai/voice/transport/providers/daily"
```

## Usage

```go
t, err := transport.New("daily", transport.Config{
    URL:   "https://myapp.daily.co/room",
    Token: "...",
})
frames, err := t.Recv(ctx)
```

## Configuration

The transport requires a Daily.co room URL. An optional authentication token
and sample rate (default 16000 Hz) can be provided via [transport.Config].

## Exported Types

- [Transport] — implements transport.AudioTransport for Daily.co
- [New] — constructor accepting transport.Config

---

## livekit

```go
import "github.com/lookatitude/beluga-ai/voice/transport/providers/livekit"
```

Package livekit provides the LiveKit transport provider for the Beluga AI
voice pipeline. It implements the [transport.AudioTransport] interface for
bidirectional audio I/O through LiveKit rooms.

LiveKit is treated as a TRANSPORT, not a framework dependency. LiveKit
provides WebRTC transport while Beluga handles all STT/LLM/TTS processing
through the frame-based pipeline.

## Registration

This package registers itself as "livekit" with the transport registry.
Import it with a blank identifier to enable:

```go
import _ "github.com/lookatitude/beluga-ai/voice/transport/providers/livekit"
```

## Usage

```go
t, err := transport.New("livekit", transport.Config{
    URL:   "wss://myapp.livekit.cloud",
    Token: "...",
    Extra: map[string]any{"room": "my-room"},
})
frames, err := t.Recv(ctx)
```

## Configuration

Required fields in [transport.Config]:

- URL — LiveKit server URL (required)
- Token — LiveKit authentication token (required)

Optional Extra fields:

- room — LiveKit room name

Default sample rate is 16000 Hz, default channel count is 1 (mono).

## Exported Types

- [Transport] — implements transport.AudioTransport for LiveKit
- [New] — constructor accepting transport.Config

---

## pipecat

```go
import "github.com/lookatitude/beluga-ai/voice/transport/providers/pipecat"
```

Package pipecat provides the Pipecat transport provider for the Beluga AI
voice pipeline. It implements the [transport.AudioTransport] interface for
bidirectional audio I/O through a Pipecat server over WebSocket.

## Registration

This package registers itself as "pipecat" with the transport registry.
Import it with a blank identifier to enable:

```go
import _ "github.com/lookatitude/beluga-ai/voice/transport/providers/pipecat"
```

## Usage

```go
t, err := transport.New("pipecat", transport.Config{
    URL: "ws://localhost:8765",
})
frames, err := t.Recv(ctx)
```

## Configuration

The transport requires a Pipecat server WebSocket URL. Default sample rate
is 16000 Hz.

## Exported Types

- [Transport] — implements transport.AudioTransport for Pipecat
- [New] — constructor accepting transport.Config
