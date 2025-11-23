# Voice Provider Configuration Guide

This guide explains how to configure and use different voice providers in Beluga AI.

## STT Providers

### Deepgram

```go
import "github.com/lookatitude/beluga-ai/pkg/voice/stt/providers/deepgram"

sttProvider, err := deepgram.NewDeepgramSTT(ctx, deepgram.Config{
    APIKey: os.Getenv("DEEPGRAM_API_KEY"),
})
```

**Features:**
- Streaming support
- Low latency
- Multiple languages
- Custom models

### Google Cloud Speech-to-Text

```go
import "github.com/lookatitude/beluga-ai/pkg/voice/stt/providers/google"

sttProvider, err := google.NewGoogleSTT(ctx, google.Config{
    CredentialsJSON: os.Getenv("GOOGLE_CREDENTIALS"),
})
```

**Features:**
- High accuracy
- Multiple languages
- Custom models
- Speaker diarization

### Azure Speech SDK

```go
import "github.com/lookatitude/beluga-ai/pkg/voice/stt/providers/azure"

sttProvider, err := azure.NewAzureSTT(ctx, azure.Config{
    SubscriptionKey: os.Getenv("AZURE_SUBSCRIPTION_KEY"),
    Region:          "eastus",
})
```

**Features:**
- Enterprise features
- Custom models
- Speaker recognition
- Language identification

### OpenAI Whisper

```go
import "github.com/lookatitude/beluga-ai/pkg/voice/stt/providers/openai"

sttProvider, err := openai.NewOpenAIWhisperSTT(ctx, openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
})
```

**Features:**
- Open-source model
- Good accuracy
- Multiple languages
- Translation support

## TTS Providers

### Google Cloud Text-to-Speech

```go
import "github.com/lookatitude/beluga-ai/pkg/voice/tts/providers/google"

ttsProvider, err := google.NewGoogleTTS(ctx, google.Config{
    CredentialsJSON: os.Getenv("GOOGLE_CREDENTIALS"),
})
```

**Features:**
- Natural voices
- Multiple languages
- SSML support
- Neural voices

### Azure Speech SDK

```go
import "github.com/lookatitude/beluga-ai/pkg/voice/tts/providers/azure"

ttsProvider, err := azure.NewAzureTTS(ctx, azure.Config{
    SubscriptionKey: os.Getenv("AZURE_SUBSCRIPTION_KEY"),
    Region:          "eastus",
})
```

**Features:**
- Neural voices
- SSML support
- Custom voices
- Prosody control

### OpenAI TTS

```go
import "github.com/lookatitude/beluga-ai/pkg/voice/tts/providers/openai"

ttsProvider, err := openai.NewOpenAITTS(ctx, openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
})
```

**Features:**
- Fast generation
- Good quality
- Multiple voices
- Streaming support

### ElevenLabs

```go
import "github.com/lookatitude/beluga-ai/pkg/voice/tts/providers/elevenlabs"

ttsProvider, err := elevenlabs.NewElevenLabsTTS(ctx, elevenlabs.Config{
    APIKey: os.Getenv("ELEVENLABS_API_KEY"),
})
```

**Features:**
- High-quality voices
- Voice cloning
- Emotional control
- Streaming support

## VAD Providers

### Silero VAD

```go
import "github.com/lookatitude/beluga-ai/pkg/voice/vad/providers/silero"

vadProvider, err := silero.NewSileroVAD(ctx, silero.Config{
    ModelPath: "path/to/model.onnx",
})
```

**Features:**
- Fast inference
- ONNX-based
- Good accuracy
- Low latency

### Energy-based VAD

```go
import "github.com/lookatitude/beluga-ai/pkg/voice/vad/providers/energy"

vadProvider, err := energy.NewEnergyVAD(ctx, energy.Config{
    Threshold: 0.5,
})
```

**Features:**
- Simple implementation
- Low latency
- No model required
- Configurable threshold

## Provider Selection Guide

### For Low Latency

- STT: Deepgram or Google Cloud (streaming)
- TTS: OpenAI TTS or ElevenLabs (streaming)
- VAD: Silero VAD or Energy-based

### For High Accuracy

- STT: Google Cloud or Azure
- TTS: Google Cloud or Azure
- VAD: Silero VAD

### For Cost Efficiency

- STT: OpenAI Whisper (self-hosted)
- TTS: OpenAI TTS
- VAD: Energy-based

## Fallback Configuration

Configure fallback providers for reliability:

```go
// Primary provider
primarySTT := deepgram.NewDeepgramSTT(...)

// Fallback provider
fallbackSTT := google.NewGoogleSTT(...)

// Session will automatically fallback on primary failure
voiceSession, err := session.NewVoiceSession(ctx,
    session.WithSTTProvider(primarySTT),
    // Fallback would be configured internally
)
```

## Environment Variables

Set provider API keys via environment variables:

```bash
export DEEPGRAM_API_KEY="your-key"
export GOOGLE_CREDENTIALS="path/to/credentials.json"
export AZURE_SUBSCRIPTION_KEY="your-key"
export OPENAI_API_KEY="your-key"
export ELEVENLABS_API_KEY="your-key"
```

## Testing Providers

Use mock providers for testing:

```go
import "github.com/lookatitude/beluga-ai/pkg/voice/session"

mockSTT := session.NewMockSTTProvider()
mockTTS := session.NewMockTTSProvider()

voiceSession, err := session.NewVoiceSession(ctx,
    session.WithSTTProvider(mockSTT),
    session.WithTTSProvider(mockTTS),
)
```

## Performance Tuning

- Use streaming for real-time interactions
- Configure appropriate timeouts
- Monitor provider latency
- Implement caching where appropriate

## Troubleshooting

See the [troubleshooting guide](voice-troubleshooting.md) for provider-specific issues.

