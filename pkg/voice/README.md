# Voice Package

The Voice package provides a comprehensive framework for building voice-enabled AI agents, supporting speech-to-text (STT), text-to-speech (TTS), voice activity detection (VAD), turn detection, audio transport, noise cancellation, and complete session management.

## Overview

The Voice package follows the Beluga AI Framework design patterns, providing:
- **Modular Architecture**: Independent packages for each voice functionality
- **Provider Abstraction**: Multiple provider implementations for each component
- **Complete Session Management**: Full lifecycle management for voice interactions
- **Advanced Features**: Error recovery, timeouts, interruption handling, preemptive generation
- **Streaming Support**: Real-time streaming for STT, TTS, and agent responses
- **Observability**: OTEL metrics and tracing throughout
- **Configuration**: Flexible configuration with validation

## Package Structure

```
pkg/voice/
├── iface/              # Shared interfaces and types
├── stt/                # Speech-to-Text package
├── tts/                # Text-to-Speech package
├── vad/                # Voice Activity Detection package
├── turndetection/      # Turn Detection package
├── transport/          # Audio Transport package
├── noise/              # Noise Cancellation package
└── session/           # Session Management package
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

