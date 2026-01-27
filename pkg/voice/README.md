# Voice Package (DEPRECATED)

> **DEPRECATED**: This package provides backward-compatible shims only.
> Voice functionality has been refactored into independent top-level packages for better reusability.
>
> Please update your imports to use the new package locations listed below.
> This package will be removed in v2.0.

## Migration Guide

| Old Import | New Import |
|------------|------------|
| `pkg/voice/stt` | `pkg/stt` |
| `pkg/voice/tts` | `pkg/tts` |
| `pkg/voice/vad` | `pkg/vad` |
| `pkg/voice/s2s` | `pkg/s2s` |
| `pkg/voice/transport` | `pkg/audiotransport` |
| `pkg/voice/noise` | `pkg/noisereduction` |
| `pkg/voice/turndetection` | `pkg/turndetection` |
| `pkg/voice/backend` | `pkg/voicebackend` |
| `pkg/voice/session` | `pkg/voicesession` |
| `pkg/voice/iface` | `pkg/voiceutils/iface` |

## New Package Structure

```
pkg/
├── stt/                # Speech-to-Text providers
├── tts/                # Text-to-Speech providers
├── vad/                # Voice Activity Detection
├── s2s/                # Speech-to-Speech (end-to-end)
├── audiotransport/     # Audio Transport (WebRTC, WebSocket)
├── noisereduction/     # Noise Cancellation
├── turndetection/      # Turn Detection
├── voicebackend/       # Voice Backend integrations
├── voicesession/       # Session Management
└── voiceutils/         # Shared interfaces and utilities
```

## Quick Start

### Basic Voice Session

```go
import (
    "context"
    "github.com/lookatitude/beluga-ai/pkg/voice/session"
)

// Create providers
sttProvider := // ... your STT provider
ttsProvider := // ... your TTS provider

agentCallback := func(ctx context.Context, transcript string) (string, error) {
    return "Hello! How can I help you?", nil
}

// Create and start session
voiceSession, err := session.NewVoiceSession(ctx,
    session.WithSTTProvider(sttProvider),
    session.WithTTSProvider(ttsProvider),
    session.WithAgentCallback(agentCallback),
)
if err != nil {
    log.Fatal(err)
}

err = voiceSession.Start(ctx)
if err != nil {
    log.Fatal(err)
}

// Process audio
audio := []byte{...}
err = voiceSession.ProcessAudio(ctx, audio)

// Say something
handle, err := voiceSession.Say(ctx, "Hello, user!")

// Stop session
err = voiceSession.Stop(ctx)
```

## Packages

### STT (Speech-to-Text)

Converts audio to text using various providers:
- Deepgram
- Google Cloud Speech-to-Text
- Azure Speech SDK
- OpenAI Whisper

See [STT README](stt/README.md) for details.

### TTS (Text-to-Speech)

Converts text to speech audio using various providers:
- Google Cloud Text-to-Speech
- Azure Speech SDK
- OpenAI TTS
- ElevenLabs

See [TTS README](tts/README.md) for details.

### VAD (Voice Activity Detection)

Detects voice activity in audio:
- Silero VAD
- Energy-based VAD
- WebRTC VAD
- ONNX VAD

See [VAD README](vad/README.md) for details.

### Turn Detection

Detects the end of user turns:
- Silence-based
- ONNX-based

See [Turn Detection README](turndetection/README.md) for details.

### Transport

Handles audio transport:
- WebRTC
- WebSocket

See [Transport README](transport/README.md) for details.

### Noise Cancellation

Removes noise from audio:
- RNNoise
- Spectral Subtraction
- WebRTC Noise Suppression

See [Noise Cancellation README](noise/README.md) for details.

### Session

Manages complete voice interaction sessions with:
- Lifecycle management
- Error recovery
- Timeout handling
- Interruption detection
- Preemptive generation
- Long utterance handling

See [Session README](session/README.md) for details.

## Architecture

The Voice package follows a layered architecture:

1. **Interface Layer**: Shared interfaces (`pkg/voice/iface/`)
2. **Provider Layer**: Provider implementations (`pkg/voice/{package}/providers/`)
3. **Session Layer**: Session management (`pkg/voice/session/`)

## Configuration

All packages support flexible configuration:

```go
config := session.DefaultConfig()
config.Timeout = 30 * time.Minute
config.MaxRetries = 3

voiceSession, err := session.NewVoiceSession(ctx,
    session.WithConfig(config),
    // ... other options
)
```

## Observability

All operations emit:
- **Metrics**: OTEL metrics with operation names
- **Traces**: OTEL spans with session context
- **Logs**: Structured logs with session IDs

## Error Handling

The Voice package uses structured error handling:

```go
if err != nil {
    var sessionErr *session.SessionError
    if errors.As(err, &sessionErr) {
        switch sessionErr.Code {
        case session.ErrCodeSessionNotActive:
            // Handle error
        }
    }
}
```

## Examples

See the [examples directory](../../../examples/voice/) for complete usage examples.

## Performance

- **Latency**: Sub-200ms for most operations
- **Throughput**: 1000+ audio chunks/second
- **Concurrency**: 100+ concurrent sessions

## License

Part of the Beluga AI Framework. See main LICENSE file.

