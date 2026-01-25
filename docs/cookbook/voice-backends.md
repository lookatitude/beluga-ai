# Voice Backends Configuration

> **Quick recipes for configuring and switching between voice backends (STT, TTS, S2S).**

## Problem

You're building a voice agent and need to:
- Choose the right voice backend for your use case
- Configure providers with proper settings
- Switch between providers without code changes
- Handle fallback when a provider is unavailable

## Solution Overview

Beluga AI's voice system uses a consistent configuration and registry pattern across all voice backends. You can switch providers through configuration, implement automatic fallback, and test with mocks - all without changing your application code.

## Recipes

### Recipe 1: Configure STT Provider

**Scenario**: You need to transcribe audio input using Deepgram.

```go
import (
    "context"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/voice/stt"
    _ "github.com/lookatitude/beluga-ai/pkg/voice/stt/providers/deepgram" // Register
)

func setupSTT(ctx context.Context) (stt.Provider, error) {
    config := stt.DefaultConfig()
    config.Provider = "deepgram"
    config.APIKey = os.Getenv("DEEPGRAM_API_KEY")
    config.Model = "nova-2"
    config.Language = "en-US"
    config.SampleRate = 16000
    config.Channels = 1
    config.EnableStreaming = true

    return stt.NewProvider(ctx, "deepgram", config)
}

// Usage
func transcribe(ctx context.Context, audio []byte) (string, error) {
    provider, err := setupSTT(ctx)
    if err != nil {
        return "", err
    }
    
    return provider.Transcribe(ctx, audio)
}
```

**Why this works**: The registry pattern allows any registered provider to be instantiated with the same configuration structure. The provider name selects the implementation.

---

### Recipe 2: Configure TTS Provider

**Scenario**: You need to generate speech using OpenAI's TTS.

```go
import (
    "context"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/voice/tts"
    _ "github.com/lookatitude/beluga-ai/pkg/voice/tts/providers/openai" // Register
)

func setupTTS(ctx context.Context) (tts.Provider, error) {
    config := tts.DefaultConfig()
    config.Provider = "openai"
    config.APIKey = os.Getenv("OPENAI_API_KEY")
    config.Model = "tts-1-hd"  // High-quality model
    config.Voice = "alloy"     // Available: alloy, echo, fable, onyx, nova, shimmer
    config.Speed = 1.0         // 0.25 to 4.0
    config.SampleRate = 24000

    return tts.NewProvider(ctx, "openai", config)
}

// Usage
func speak(ctx context.Context, text string) ([]byte, error) {
    provider, err := setupTTS(ctx)
    if err != nil {
        return nil, err
    }
    
    return provider.GenerateSpeech(ctx, text)
}
```

---

### Recipe 3: Configure S2S Provider

**Scenario**: You want end-to-end speech conversations using Amazon Nova.

```go
import (
    "context"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/voice/s2s"
    _ "github.com/lookatitude/beluga-ai/pkg/voice/s2s/providers/amazon_nova" // Register
)

func setupS2S(ctx context.Context) (s2s.Provider, error) {
    config := s2s.DefaultConfig()
    config.Provider = "amazon_nova"
    config.LatencyTarget = "low"  // low, medium, high
    config.ReasoningMode = "built-in"  // built-in or external
    
    // AWS credentials (use IAM roles in production)
    config.ProviderSpecific = map[string]any{
        "region":        "us-east-1",
        "model":         "nova-2-sonic",
        "voice_id":      "Ruth",
        "language_code": "en-US",
    }

    return s2s.NewProvider(ctx, "amazon_nova", config)
}

// Usage in a voice session
func createVoiceSession(ctx context.Context) (*session.VoiceSession, error) {
    s2sProvider, err := setupS2S(ctx)
    if err != nil {
        return nil, err
    }
    
    return session.NewVoiceSession(ctx,
        session.WithS2SProvider(s2sProvider),
    )
}
```

---

### Recipe 4: Switch Providers via Configuration

**Scenario**: You want to switch providers based on environment or configuration.

```go
import (
    "context"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/voice/stt"
    // Import all providers you might use
    _ "github.com/lookatitude/beluga-ai/pkg/voice/stt/providers/deepgram"
    _ "github.com/lookatitude/beluga-ai/pkg/voice/stt/providers/google"
    _ "github.com/lookatitude/beluga-ai/pkg/voice/stt/providers/azure"
    _ "github.com/lookatitude/beluga-ai/pkg/voice/stt/providers/openai"
)

type VoiceConfig struct {
    STTProvider string `yaml:"stt_provider" env:"VOICE_STT_PROVIDER"`
    STTAPIKey   string `yaml:"stt_api_key" env:"VOICE_STT_API_KEY"`
    STTModel    string `yaml:"stt_model" env:"VOICE_STT_MODEL"`
    
    TTSProvider string `yaml:"tts_provider" env:"VOICE_TTS_PROVIDER"`
    TTSAPIKey   string `yaml:"tts_api_key" env:"VOICE_TTS_API_KEY"`
    TTSVoice    string `yaml:"tts_voice" env:"VOICE_TTS_VOICE"`
}

func setupSTTFromConfig(ctx context.Context, cfg VoiceConfig) (stt.Provider, error) {
    config := stt.DefaultConfig()
    config.Provider = cfg.STTProvider  // "deepgram", "google", "azure", "openai"
    config.APIKey = cfg.STTAPIKey
    config.Model = cfg.STTModel
    
    return stt.NewProvider(ctx, cfg.STTProvider, config)
}

// Now you can switch providers via environment variables:
// VOICE_STT_PROVIDER=deepgram go run main.go
// VOICE_STT_PROVIDER=google go run main.go
```

---

### Recipe 5: Implement Provider Fallback

**Scenario**: You want automatic fallback if the primary provider fails.

```go
import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/pkg/voice/stt"
)

type FallbackSTT struct {
    providers []stt.Provider
}

func NewFallbackSTT(providers ...stt.Provider) *FallbackSTT {
    return &FallbackSTT{providers: providers}
}

func (f *FallbackSTT) Transcribe(ctx context.Context, audio []byte) (string, error) {
    var lastErr error
    
    for _, provider := range f.providers {
        text, err := provider.Transcribe(ctx, audio)
        if err == nil {
            return text, nil
        }
        
        lastErr = err
        // Log the failure
        slog.Warn("STT provider failed, trying next",
            "provider", provider.GetName(),
            "error", err,
        )
    }
    
    return "", fmt.Errorf("all providers failed, last error: %w", lastErr)
}

// Usage
func setupWithFallback(ctx context.Context) (*FallbackSTT, error) {
    deepgram, _ := stt.NewProvider(ctx, "deepgram", deepgramConfig)
    google, _ := stt.NewProvider(ctx, "google", googleConfig)
    azure, _ := stt.NewProvider(ctx, "azure", azureConfig)
    
    return NewFallbackSTT(deepgram, google, azure), nil
}
```

---

### Recipe 6: Use Mock Providers for Testing

**Scenario**: You need to test your voice agent without real API calls.

```go
import (
    "testing"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/voice/stt"
    "github.com/lookatitude/beluga-ai/pkg/voice/tts"
    "github.com/lookatitude/beluga-ai/pkg/voice/session"
)

func TestVoiceAgent(t *testing.T) {
    ctx := context.Background()

    // Create mock STT provider
    mockSTT := stt.NewAdvancedMockSTTProvider("mock-stt",
        stt.WithTranscriptions(
            "Hello, how can I help you?",
            "I'd like to book a flight",
            "Thank you for your help",
        ),
        stt.WithStreamingDelay(10 * time.Millisecond),
    )

    // Create mock TTS provider
    mockTTS := tts.NewAdvancedMockTTSProvider("mock-tts",
        tts.WithAudioResponses(
            []byte{1, 2, 3, 4}, // Simulated audio data
        ),
    )

    // Create voice session with mocks
    voiceSession, err := session.NewVoiceSession(ctx,
        session.WithSTTProvider(mockSTT),
        session.WithTTSProvider(mockTTS),
        session.WithAgentInstance(agent, agentConfig),
    )
    require.NoError(t, err)

    // Test your voice agent logic
    err = voiceSession.Start(ctx)
    require.NoError(t, err)

    // Simulate audio input
    testAudio := []byte{5, 6, 7, 8}
    err = voiceSession.ProcessAudio(ctx, testAudio)
    require.NoError(t, err)

    // Verify behavior
    // ...
}
```

---

### Recipe 7: Streaming Audio Configuration

**Scenario**: You need to configure streaming for real-time transcription.

```go
import (
    "context"

    "github.com/lookatitude/beluga-ai/pkg/voice/stt"
)

func streamingTranscription(ctx context.Context, audioStream <-chan []byte) (<-chan string, error) {
    config := stt.DefaultConfig()
    config.Provider = "deepgram"
    config.APIKey = os.Getenv("DEEPGRAM_API_KEY")
    config.EnableStreaming = true
    config.SampleRate = 16000
    config.Channels = 1
    
    // Provider-specific streaming settings
    config.ProviderSpecific = map[string]any{
        "interim_results":  true,  // Get partial transcripts
        "punctuate":        true,  // Add punctuation
        "utterance_end_ms": 1000,  // Detect end of utterance
    }

    provider, err := stt.NewProvider(ctx, "deepgram", config)
    if err != nil {
        return nil, err
    }

    // Start streaming session
    session, err := provider.StartStreaming(ctx)
    if err != nil {
        return nil, err
    }

    // Create output channel
    transcripts := make(chan string)

    // Feed audio to session
    go func() {
        defer session.Close()
        for audio := range audioStream {
            if err := session.SendAudio(ctx, audio); err != nil {
                break
            }
        }
    }()

    // Receive transcripts
    go func() {
        defer close(transcripts)
        for result := range session.ReceiveTranscript() {
            if result.Error != nil {
                continue
            }
            if result.IsFinal {
                transcripts <- result.Text
            }
        }
    }()

    return transcripts, nil
}
```

---

### Recipe 8: Multi-Language Configuration

**Scenario**: You need to support multiple languages dynamically.

```go
import (
    "context"

    "github.com/lookatitude/beluga-ai/pkg/voice/stt"
    "github.com/lookatitude/beluga-ai/pkg/voice/tts"
)

type MultiLanguageVoice struct {
    sttConfigs map[string]*stt.Config
    ttsConfigs map[string]*tts.Config
}

func NewMultiLanguageVoice() *MultiLanguageVoice {
    return &MultiLanguageVoice{
        sttConfigs: map[string]*stt.Config{
            "en-US": {
                Provider:   "deepgram",
                APIKey:     os.Getenv("DEEPGRAM_API_KEY"),
                Language:   "en-US",
                Model:      "nova-2",
            },
            "es-ES": {
                Provider:   "deepgram",
                APIKey:     os.Getenv("DEEPGRAM_API_KEY"),
                Language:   "es",
                Model:      "nova-2",
            },
            "fr-FR": {
                Provider:   "google",
                APIKey:     os.Getenv("GOOGLE_API_KEY"),
                Language:   "fr-FR",
                Model:      "latest_long",
            },
        },
        ttsConfigs: map[string]*tts.Config{
            "en-US": {
                Provider: "openai",
                APIKey:   os.Getenv("OPENAI_API_KEY"),
                Voice:    "alloy",
            },
            "es-ES": {
                Provider: "openai",
                APIKey:   os.Getenv("OPENAI_API_KEY"),
                Voice:    "nova",
            },
            "fr-FR": {
                Provider: "elevenlabs",
                APIKey:   os.Getenv("ELEVENLABS_API_KEY"),
                Voice:    "french-voice-id",
            },
        },
    }
}

func (m *MultiLanguageVoice) GetSTTProvider(ctx context.Context, lang string) (stt.Provider, error) {
    config, ok := m.sttConfigs[lang]
    if !ok {
        config = m.sttConfigs["en-US"] // Default to English
    }
    return stt.NewProvider(ctx, config.Provider, config)
}

func (m *MultiLanguageVoice) GetTTSProvider(ctx context.Context, lang string) (tts.Provider, error) {
    config, ok := m.ttsConfigs[lang]
    if !ok {
        config = m.ttsConfigs["en-US"]
    }
    return tts.NewProvider(ctx, config.Provider, config)
}
```

## Provider Reference

### STT Providers

| Provider | Streaming | Languages | Best For |
|----------|-----------|-----------|----------|
| Deepgram | ✅ WebSocket | 30+ | Real-time, accuracy |
| Google | ✅ gRPC | 125+ | Language coverage |
| Azure | ✅ WebSocket | 100+ | Enterprise integration |
| OpenAI (Whisper) | ❌ REST only | 99 | Batch processing |

### TTS Providers

| Provider | Streaming | Voices | Best For |
|----------|-----------|--------|----------|
| OpenAI | ✅ | 6 | Simple, fast |
| ElevenLabs | ✅ | 100+ | Voice cloning, quality |
| Google | ✅ | 220+ | Language coverage |
| Azure | ✅ | 400+ | Enterprise, SSML |

### S2S Providers

| Provider | Streaming | Latency | Best For |
|----------|-----------|---------|----------|
| Amazon Nova | ✅ Bidirectional | Low | Real-time conversations |
| OpenAI Realtime | ✅ Bidirectional | Low | GPT-powered agents |

## Related Resources

- **[Voice Provider Integration Guide](../guides/voice-providers.md)**: Complete provider implementation guide
- **[Voice Sessions Use Case](../use-cases/voice-sessions.md)**: Real-world voice agent patterns
- **[LLM Error Handling Cookbook](./llm-error-handling.md)**: Similar error handling patterns
- **[STT Package Documentation](https://github.com/lookatitude/beluga-ai/blob/main/pkg/voice/stt/README.md)**: Detailed STT docs
- **[TTS Package Documentation](https://github.com/lookatitude/beluga-ai/blob/main/pkg/voice/tts/README.md)**: Detailed TTS docs
- **[S2S Package Documentation](https://github.com/lookatitude/beluga-ai/blob/main/pkg/voice/s2s/README.md)**: Detailed S2S docs
