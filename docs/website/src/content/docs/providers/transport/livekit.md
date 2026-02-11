---
title: "LiveKit"
description: "Integration guide for the LiveKit transport provider in Beluga AI."
---

The LiveKit provider implements bidirectional audio transport through LiveKit rooms. LiveKit is treated as a transport layer — it provides WebRTC connectivity while Beluga handles all STT, LLM, and TTS processing through the frame-based pipeline.

Choose LiveKit when you need a self-hostable, open-source WebRTC infrastructure for production voice deployments. LiveKit provides room management, access token authentication, and scalable media routing. Beluga treats LiveKit as a pure transport layer, handling all voice processing in the frame-based pipeline. For a managed alternative, consider [Daily](/providers/transport/daily).

## Installation

```bash
go get github.com/lookatitude/beluga-ai/voice/transport/providers/livekit
```

## Configuration

| Field        | Required | Default  | Description                              |
|--------------|----------|----------|------------------------------------------|
| `URL`        | Yes      | —        | LiveKit server URL (`wss://...`)         |
| `Token`      | Yes      | —        | LiveKit access token                     |
| `SampleRate` | No       | `16000`  | Audio sample rate in Hz                  |
| `Channels`   | No       | `1`      | Audio channels (1=mono, 2=stereo)        |

**Extra configuration:**

| Key    | Type     | Description                 |
|--------|----------|-----------------------------|
| `room` | `string` | LiveKit room name to join   |

## Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/voice/transport"
    _ "github.com/lookatitude/beluga-ai/voice/transport/providers/livekit"
)

func main() {
    t, err := transport.New("livekit", transport.Config{
        URL:   "wss://myapp.livekit.cloud",
        Token: "your-livekit-token",
        Extra: map[string]any{
            "room": "voice-room",
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    defer t.Close()

    // Receive incoming audio frames
    frames, err := t.Recv(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    for frame := range frames {
        fmt.Printf("Received %d bytes of audio\n", len(frame.Data))
    }
}
```

## Advanced Features

### Sending Audio

Write processed audio back to the LiveKit room using `voice.NewAudioFrame`:

```go
frame := voice.NewAudioFrame(processedPCM, 16000)
err := t.Send(ctx, frame)
if err != nil {
    log.Printf("send failed: %v", err)
}
```

### Room Configuration

Pass the room name through the `Extra` map:

```go
t, err := transport.New("livekit", transport.Config{
    URL:   "wss://myapp.livekit.cloud",
    Token: token,
    Extra: map[string]any{
        "room": "my-voice-room",
    },
})
```

### Voice Pipeline Integration

Wrap the transport for use with Beluga's voice pipeline:

```go
import "github.com/lookatitude/beluga-ai/voice/transport"

audioTransport, err := transport.New("livekit", cfg)
if err != nil {
    log.Fatal(err)
}

voiceTransport := &transport.AsVoiceTransport{T: audioTransport}
// Use voiceTransport with the voice pipeline
```

## Direct Construction

```go
import "github.com/lookatitude/beluga-ai/voice/transport/providers/livekit"

t, err := livekit.New(transport.Config{
    URL:        "wss://myapp.livekit.cloud",
    Token:      "your-livekit-token",
    SampleRate: 16000,
    Channels:   1,
    Extra:      map[string]any{"room": "voice-room"},
})
```

## Error Handling

```go
t, err := transport.New("livekit", transport.Config{
    URL: "wss://myapp.livekit.cloud",
    // Missing Token
})
if err != nil {
    // "livekit: Token is required"
    log.Fatal(err)
}
```

Both `URL` and `Token` are required. The transport returns errors from `Recv` and `Send` if `Close` has already been called.

Refer to the [LiveKit documentation](https://docs.livekit.io/) for server setup, room management, and token generation.
