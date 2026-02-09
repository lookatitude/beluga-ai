---
title: "Voice Backends Configuration"
description: "Configure and switch between STT, TTS, and S2S voice backends with fallback support."
---

## Problem

You are building a voice agent and need to choose the right voice backend, configure providers with proper settings, switch between providers without code changes, and handle fallback when a provider is unavailable.

## Solution

Beluga AI's voice system uses a consistent registry pattern across all voice backends. You can switch providers through configuration, implement automatic fallback, and test with mocks without changing application code.

## Recipe 1: Configure STT Provider

Set up Deepgram for real-time speech-to-text:

```go
package main

import (
	"context"
	"os"

	"github.com/lookatitude/beluga-ai/voice/stt"
	_ "github.com/lookatitude/beluga-ai/voice/stt/providers/deepgram" // Register provider
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
```

## Recipe 2: Configure TTS Provider

Set up OpenAI for text-to-speech:

```go
import (
	"context"
	"os"

	"github.com/lookatitude/beluga-ai/voice/tts"
	_ "github.com/lookatitude/beluga-ai/voice/tts/providers/openai" // Register provider
)

func setupTTS(ctx context.Context) (tts.Provider, error) {
	config := tts.DefaultConfig()
	config.Provider = "openai"
	config.APIKey = os.Getenv("OPENAI_API_KEY")
	config.Model = "tts-1-hd"
	config.Voice = "alloy"
	config.Speed = 1.0
	config.SampleRate = 24000

	return tts.NewProvider(ctx, "openai", config)
}
```

## Recipe 3: Configure S2S Provider

Set up Amazon Nova for end-to-end speech conversations:

```go
import (
	"context"

	"github.com/lookatitude/beluga-ai/voice/s2s"
	_ "github.com/lookatitude/beluga-ai/voice/s2s/providers/amazon_nova" // Register provider
)

func setupS2S(ctx context.Context) (s2s.Provider, error) {
	config := s2s.DefaultConfig()
	config.Provider = "amazon_nova"
	config.LatencyTarget = "low"
	config.ReasoningMode = "built-in"
	config.ProviderSpecific = map[string]any{
		"region":        "us-east-1",
		"model":         "nova-2-sonic",
		"voice_id":      "Ruth",
		"language_code": "en-US",
	}

	return s2s.NewProvider(ctx, "amazon_nova", config)
}
```

## Recipe 4: Switch Providers via Configuration

Switch providers based on environment variables without code changes:

```go
import (
	"context"

	"github.com/lookatitude/beluga-ai/voice/stt"
	_ "github.com/lookatitude/beluga-ai/voice/stt/providers/deepgram"
	_ "github.com/lookatitude/beluga-ai/voice/stt/providers/google"
	_ "github.com/lookatitude/beluga-ai/voice/stt/providers/azure"
)

type VoiceConfig struct {
	STTProvider string `yaml:"stt_provider" env:"VOICE_STT_PROVIDER"`
	STTAPIKey   string `yaml:"stt_api_key" env:"VOICE_STT_API_KEY"`
	STTModel    string `yaml:"stt_model" env:"VOICE_STT_MODEL"`
}

func setupSTTFromConfig(ctx context.Context, cfg VoiceConfig) (stt.Provider, error) {
	config := stt.DefaultConfig()
	config.Provider = cfg.STTProvider
	config.APIKey = cfg.STTAPIKey
	config.Model = cfg.STTModel

	return stt.NewProvider(ctx, cfg.STTProvider, config)
}
```

## Recipe 5: Implement Provider Fallback

Automatic fallback if the primary provider fails:

```go
import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lookatitude/beluga-ai/voice/stt"
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
		slog.Warn("STT provider failed, trying next",
			"provider", provider.GetName(),
			"error", err,
		)
	}
	return "", fmt.Errorf("all providers failed, last error: %w", lastErr)
}
```

## Provider Reference

### STT Providers

| Provider | Streaming | Languages | Best For |
|----------|-----------|-----------|----------|
| Deepgram | WebSocket | 30+ | Real-time, accuracy |
| Google | gRPC | 125+ | Language coverage |
| Azure | WebSocket | 100+ | Enterprise integration |
| OpenAI (Whisper) | REST only | 99 | Batch processing |

### TTS Providers

| Provider | Streaming | Voices | Best For |
|----------|-----------|--------|----------|
| OpenAI | Yes | 6 | Simple, fast |
| ElevenLabs | Yes | 100+ | Voice cloning, quality |
| Google | Yes | 220+ | Language coverage |
| Azure | Yes | 400+ | Enterprise, SSML |

### S2S Providers

| Provider | Streaming | Latency | Best For |
|----------|-----------|---------|----------|
| Amazon Nova | Bidirectional | Low | Real-time conversations |
| OpenAI Realtime | Bidirectional | Low | GPT-powered agents |

## Related Recipes

- **[LLM Error Handling](./llm-error-handling)** -- Similar error handling and retry patterns
- **[Voice Stream Scaling](./voice-stream-scaling)** -- Scaling concurrent voice streams
- **[S2S Voice Metrics](./s2s-voice-metrics)** -- Custom metrics for voice systems
