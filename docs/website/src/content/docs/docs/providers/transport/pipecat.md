---
title: "Pipecat Transport Provider"
description: "Pipecat-compatible transport for voice pipelines in Beluga AI. Frame-based audio streaming with pipeline integration and protocol bridging in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Pipecat, audio transport, voice pipeline, frame-based, pipeline integration, protocol bridge, Go, Beluga AI"
---

The Pipecat provider implements bidirectional audio transport through a Pipecat server over WebSocket. It enables Beluga's voice pipeline to interoperate with Pipecat-based audio processing systems.

Choose Pipecat when you need to bridge Beluga's voice pipeline with an existing Pipecat-based processing system over WebSocket. This is useful for hybrid architectures where some audio processing happens in a Pipecat server while Beluga handles the agent logic. For WebRTC-based transport, consider [LiveKit](/providers/transport/livekit) or [Daily](/providers/transport/daily).

## Installation

```bash
go get github.com/lookatitude/beluga-ai/voice/transport/providers/pipecat
```

## Configuration

| Field        | Required | Default  | Description                              |
|--------------|----------|----------|------------------------------------------|
| `URL`        | Yes      | —        | Pipecat server WebSocket URL             |
| `SampleRate` | No       | `16000`  | Audio sample rate in Hz                  |

## Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/voice/transport"
    _ "github.com/lookatitude/beluga-ai/voice/transport/providers/pipecat"
)

func main() {
    t, err := transport.New("pipecat", transport.Config{
        URL: "ws://localhost:8765",
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

Write processed audio back to the Pipecat server using `voice.NewAudioFrame`:

```go
frame := voice.NewAudioFrame(processedPCM, 16000)
err := t.Send(ctx, frame)
if err != nil {
    log.Printf("send failed: %v", err)
}
```

### Raw Audio Output

For direct audio piping without frame wrapping:

```go
writer := t.AudioOut()
_, err := writer.Write(rawPCMData)
if err != nil {
    log.Printf("audio write failed: %v", err)
}
```

## Direct Construction

```go
import "github.com/lookatitude/beluga-ai/voice/transport/providers/pipecat"

t, err := pipecat.New(transport.Config{
    URL:        "ws://localhost:8765",
    SampleRate: 16000,
})
```

## Error Handling

```go
t, err := transport.New("pipecat", transport.Config{})
if err != nil {
    // "pipecat: URL is required"
    log.Fatal(err)
}

frames, err := t.Recv(ctx)
if err != nil {
    // Transport may be closed
    log.Fatal(err)
}
```

The transport returns errors from `Recv` and `Send` if `Close` has already been called.

Refer to the [Pipecat documentation](https://docs.pipecat.ai/) for server setup and configuration.
