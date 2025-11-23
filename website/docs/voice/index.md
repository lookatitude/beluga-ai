---
title: Voice Agents
sidebar_position: 1
---

# Voice Agents

The Voice Agents feature provides a comprehensive framework for building voice-enabled AI applications. It supports real-time speech-to-text transcription, text-to-speech synthesis, voice activity detection, turn detection, audio transport, noise cancellation, and complete session management.

## Overview

Voice Agents enable natural voice interactions between users and AI agents, supporting:

- **Real-time Speech-to-Text**: Convert spoken audio to text using multiple providers
- **Text-to-Speech**: Generate natural-sounding speech from text
- **Voice Activity Detection**: Detect when users are speaking
- **Turn Detection**: Identify when users have finished speaking
- **Audio Transport**: Handle audio I/O over WebRTC or WebSocket
- **Noise Cancellation**: Remove background noise from audio
- **Session Management**: Complete lifecycle management for voice interactions

## Architecture

The Voice Agents feature follows a modular architecture with independent packages for each functionality:

```
User Audio → Transport → Noise Cancellation → VAD → STT → Agent → TTS → Transport → User Audio
```

### Component Flow

1. **Transport**: Receives audio from the user (WebRTC/WebSocket)
2. **Noise Cancellation**: Removes background noise
3. **VAD**: Detects when the user is speaking
4. **STT**: Converts speech to text
5. **Agent**: Processes text and generates responses
6. **TTS**: Converts text to speech
7. **Transport**: Sends audio back to the user
8. **Session**: Manages the complete interaction lifecycle

## Quick Start

### Basic Voice Session

```go
import (
    "context"
    "log"
    
    "github.com/lookatitude/beluga-ai/pkg/voice/session"
    "github.com/lookatitude/beluga-ai/pkg/voice/stt/providers/deepgram"
    "github.com/lookatitude/beluga-ai/pkg/voice/tts/providers/openai"
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
    
    // Create TTS provider
    ttsProvider, err := openai.NewOpenAITTS(ctx, openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Create agent callback
    agentCallback := func(ctx context.Context, transcript string) (string, error) {
        // Process transcript and generate response
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
    
    // Start session
    err = voiceSession.Start(ctx)
    if err != nil {
        log.Fatal(err)
    }
    
    // Process audio (in real application, this would come from transport)
    audio := []byte{/* audio data */}
    err = voiceSession.ProcessAudio(ctx, audio)
    if err != nil {
        log.Fatal(err)
    }
    
    // Stop session
    err = voiceSession.Stop(ctx)
    if err != nil {
        log.Fatal(err)
    }
}
```

## Packages

### [Speech-to-Text (STT)](./stt)

Converts audio to text using various providers:
- **Deepgram**: Fast, accurate, streaming support
- **Google Cloud Speech-to-Text**: High accuracy, multiple languages, streaming support
- **Azure Speech Services**: Enterprise features, custom models, streaming support
- **OpenAI Whisper**: Open-source, good accuracy, batch processing only (no streaming)

[Learn more about STT →](./stt)

### [Text-to-Speech (TTS)](./tts)

Converts text to speech audio using various providers:
- **OpenAI TTS**: High-quality voices, multiple models
- **Google Cloud Text-to-Speech**: Natural voices, SSML support
- **Azure Speech Services**: Neural voices, style support
- **ElevenLabs**: Premium voices, voice cloning

[Learn more about TTS →](./tts)

### [Voice Activity Detection (VAD)](./vad)

Detects voice activity in audio streams:
- **Silero VAD**: ONNX-based, high accuracy
- **Energy-based**: Simple, adaptive thresholds
- **WebRTC VAD**: Built-in, multiple sensitivity modes
- **RNNoise**: Deep learning-based with noise suppression

[Learn more about VAD →](./vad)

### [Turn Detection](./turndetection)

Detects the end of user turns in conversations:
- **Heuristic**: Rule-based using silence and sentence endings
- **ONNX**: Machine learning-based turn detection

[Learn more about Turn Detection →](./turndetection)

### [Transport](./transport)

Handles audio transport over various protocols:
- **WebRTC**: Peer-to-peer, low latency
- **WebSocket**: Simple, reliable streaming

[Learn more about Transport →](./transport)

### [Noise Cancellation](./noise)

Removes noise from audio signals:
- **Spectral Subtraction**: FFT-based, adaptive
- **RNNoise**: Deep learning-based suppression
- **WebRTC**: Built-in noise suppression

[Learn more about Noise Cancellation →](./noise)

### [Session Management](./session)

Manages complete voice interaction sessions with:
- Lifecycle management (start, stop, state transitions)
- Error recovery and retry logic
- Timeout handling
- Interruption detection
- Preemptive generation
- Long utterance handling

[Learn more about Session Management →](./session)

## Key Features

### Streaming Support

All components support real-time streaming for low-latency interactions:

```go
// Streaming STT
session, err := sttProvider.StartStreaming(ctx)
// Send audio chunks and receive transcripts in real-time

// Streaming TTS
reader, err := ttsProvider.StreamGenerate(ctx, text)
// Read audio chunks as they're generated
```

### Error Recovery

Automatic error recovery with configurable retry logic:

```go
config := session.DefaultConfig()
config.MaxRetries = 3
config.RetryDelay = 1 * time.Second
```

### Observability

Comprehensive observability with OTEL metrics and tracing:

- **Metrics**: Operation counts, latencies, error rates
- **Traces**: Distributed tracing across all components
- **Logs**: Structured logging with session context

### Configuration

Flexible configuration with validation:

```go
config := session.DefaultConfig()
config.Timeout = 30 * time.Minute
config.EnableKeepAlive = true
config.KeepAliveInterval = 30 * time.Second
```

## Performance

The Voice Agents feature is optimized for production use:

- **Latency**: Sub-200ms for most operations
- **Throughput**: 1000+ audio chunks/second
- **Concurrency**: 100+ concurrent sessions
- **Reliability**: Automatic error recovery and fallback

## Examples

See the [examples directory](https://github.com/lookatitude/beluga-ai/tree/main/examples/voice) for complete usage examples:

- [Simple Voice Agent](../examples/voice/simple)
- [Multi-Provider Setup](../examples/voice/multi_provider)
- [Streaming Example](../examples/voice/streaming)
- [Interruption Handling](../examples/voice/interruption)
- [Preemptive Generation](../examples/voice/preemptive)

## Next Steps

1. **Choose Providers**: Select STT, TTS, and other providers based on your needs
2. **Configure Session**: Set up session configuration and options
3. **Integrate Transport**: Connect audio transport (WebRTC or WebSocket)
4. **Handle Events**: Implement callbacks for state changes and errors
5. **Monitor Performance**: Set up observability and monitoring

## Related Documentation

- [Voice Agents Guide](../guides/voice-agents) - Comprehensive usage guide
- [Voice Providers Guide](../guides/voice-providers) - Provider selection and configuration
- [Voice Performance Guide](../guides/voice-performance) - Performance tuning
- [Voice Troubleshooting](../guides/voice-troubleshooting) - Common issues and solutions
- [API Reference](../api/packages/voice/) - Complete API documentation

