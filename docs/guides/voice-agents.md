# Voice Agents Guide

This guide provides comprehensive information on using the Voice Agents feature in Beluga AI.

## Overview

Voice Agents enable natural voice interactions between users and AI agents, supporting:
- Real-time speech-to-text transcription
- Text-to-speech audio generation
- Voice activity detection
- Turn detection
- Audio transport
- Noise cancellation
- Complete session management

## Architecture

The Voice Agents feature is built on a modular architecture:

```
User Audio → Transport → Noise Cancellation → VAD → STT → Agent → TTS → Transport → User Audio
```

### Components

1. **Transport**: Handles audio I/O (WebRTC, WebSocket)
2. **Noise Cancellation**: Removes background noise
3. **VAD**: Detects when user is speaking
4. **STT**: Converts speech to text
5. **Agent**: Processes text and generates responses
6. **TTS**: Converts text to speech
7. **Session**: Manages the complete interaction lifecycle

## Quick Start

See the [quickstart guide](https://github.com/lookatitude/beluga-ai/blob/main/specs/004-feature-voice-agents/quickstart.md) for a complete getting started example.

## Provider Selection

### STT Providers

- **Deepgram**: Fast, accurate, streaming support
- **Google Cloud**: High accuracy, multiple languages
- **Azure Speech**: Enterprise features, custom models
- **OpenAI Whisper**: Open-source, good accuracy

### TTS Providers

- **Google Cloud**: Natural voices, multiple languages
- **Azure Speech**: Neural voices, SSML support
- **OpenAI TTS**: Fast, good quality
- **ElevenLabs**: High-quality, voice cloning

### VAD Providers

- **Silero VAD**: Fast, accurate, ONNX-based
- **Energy-based**: Simple, low latency
- **WebRTC**: Built-in browser support
- **ONNX VAD**: Custom models

## Configuration

### Basic Configuration

```go
config := session.DefaultConfig()
config.Timeout = 30 * time.Minute
config.MaxRetries = 3
config.EnableKeepAlive = true
```

### Advanced Configuration

```go
config := &session.Config{
    SessionID:         "custom-session-id",
    Timeout:           30 * time.Minute,
    AutoStart:         false,
    EnableKeepAlive:   true,
    KeepAliveInterval: 30 * time.Second,
    MaxRetries:        5,
    RetryDelay:        1 * time.Second,
}
```

## Best Practices

### 1. Provider Selection

- Use streaming providers for real-time interactions
- Choose providers based on latency requirements
- Consider fallback providers for reliability

### 2. Error Handling

- Always handle errors gracefully
- Implement retry logic for transient errors
- Use circuit breakers for provider failures

### 3. Performance

- Use streaming for low latency
- Chunk large audio for processing
- Monitor session metrics

### 4. Security

- Secure API keys
- Use encrypted transport
- Validate audio input

## Troubleshooting

See the [troubleshooting guide](voice-troubleshooting.md) for common issues and solutions.

## Examples

See the [examples directory](https://github.com/lookatitude/beluga-ai/tree/main/examples/voice/) for complete usage examples.

## API Reference

See individual package READMEs for detailed API documentation.

