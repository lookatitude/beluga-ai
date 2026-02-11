---
title: Cloning Voices with ElevenLabs
description: Configure ElevenLabs for high-fidelity voice synthesis with cloned voices, stability tuning, and streaming TTS.
---

ElevenLabs provides high-fidelity voice synthesis with support for cloned voices, fine-grained voice settings, and streaming audio generation. Voice cloning creates a synthetic voice that closely matches a real person's voice characteristics, which is valuable for brand consistency, personalization, and accessibility applications. This tutorial demonstrates how to configure the ElevenLabs TTS provider, adjust voice characteristics for cloned voice fidelity, and implement streaming synthesis for low-latency audio output.

## What You Will Build

A text-to-speech pipeline using ElevenLabs that generates speech with a cloned or professional voice, supports stability and similarity tuning, and streams audio chunks for low-latency playback.

## Prerequisites

- ElevenLabs API key
- At least one Voice ID from the ElevenLabs Voice Lab dashboard

## Step 1: Configure the ElevenLabs Provider

Create a TTS provider using the registry with your Voice ID and model selection. The blank import triggers the ElevenLabs provider's `init()` function, which registers its factory with the TTS registry. The `WithVoice` option takes the Voice ID from ElevenLabs -- this is not a voice name but a unique identifier that maps to a specific voice profile, including cloned voices you have created in the Voice Lab.

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/lookatitude/beluga-ai/voice/tts"
	_ "github.com/lookatitude/beluga-ai/voice/tts/providers/elevenlabs"
)

func main() {
	ctx := context.Background()

	provider, err := tts.NewProvider(ctx, "elevenlabs", tts.DefaultConfig(),
		tts.WithAPIKey(os.Getenv("ELEVENLABS_API_KEY")),
		tts.WithVoice("pMsXg91Y39Z99L9Z9999"), // Your cloned Voice ID
		tts.WithModel("eleven_multilingual_v2"),
	)
	if err != nil {
		log.Fatalf("create TTS provider: %v", err)
	}

	_ = provider
}
```

The `eleven_multilingual_v2` model provides high-quality multilingual synthesis with support for 29 languages. For English-only applications, `eleven_monolingual_v1` offers slightly lower latency because it does not need to resolve language ambiguity in the input text.

## Step 2: Voice Settings for Clone Fidelity

ElevenLabs exposes stability and similarity controls that determine how closely the generated speech matches the original voice profile. These settings are critical for cloned voices because they control the balance between natural variation (which makes speech sound more human) and fidelity to the original recording (which makes it sound more like the target speaker). Pass these via provider-specific configuration.

```go
	provider, err := tts.NewProvider(ctx, "elevenlabs", &tts.Config{
		Provider: "elevenlabs",
		APIKey:   os.Getenv("ELEVENLABS_API_KEY"),
		Voice:    "pMsXg91Y39Z99L9Z9999",
		Model:    "eleven_multilingual_v2",
		ProviderSpecific: map[string]any{
			"stability":        0.5,  // 0.0 = expressive variation, 1.0 = consistent delivery
			"similarity_boost": 0.75, // 0.0 = general voice, 1.0 = close match to original
			"style":            0.0,  // Exaggeration level for the voice style
			"use_speaker_boost": true, // Enhance speaker clarity
		},
	})
	if err != nil {
		log.Fatalf("create TTS provider: %v", err)
	}
```

| Setting           | Range     | Effect                                      |
|-------------------|-----------|----------------------------------------------|
| `stability`       | 0.0 - 1.0 | Lower values add natural variation; higher values produce consistent output. For conversational agents, 0.3-0.6 sounds more natural. For narration, 0.7-0.9 sounds more professional. |
| `similarity_boost`| 0.0 - 1.0 | How closely the output matches the original voice sample. Higher values are essential for cloned voices where speaker identity matters. |
| `style`           | 0.0 - 1.0 | Exaggeration of the voice's style characteristics. Use sparingly -- values above 0.5 can sound unnatural. |
| `use_speaker_boost` | bool    | Enhances clarity, recommended for cloned voices where the training data may have imperfect recording quality. |

## Step 3: Generate Full Audio

For non-streaming use cases (offline generation, batch processing, pre-recorded messages), generate the complete audio buffer at once. This approach is simpler but introduces latency equal to the full synthesis time, making it unsuitable for real-time voice agents.

```go
	audio, err := provider.GenerateSpeech(ctx, "Welcome to the future of voice agents.")
	if err != nil {
		log.Fatalf("generate speech: %v", err)
	}

	if err := os.WriteFile("output.mp3", audio, 0644); err != nil {
		log.Fatalf("write file: %v", err)
	}

	log.Println("Audio saved to output.mp3")
```

## Step 4: Streaming Synthesis

For voice applications where time-to-first-byte matters, use streaming synthesis. Audio chunks arrive as they are generated, allowing playback to begin before the full response is ready. This is the preferred approach for real-time voice agents because it reduces perceived latency from seconds to hundreds of milliseconds -- the user hears the beginning of the response while the rest is still being synthesized.

```go
	streamProvider, err := tts.NewProvider(ctx, "elevenlabs", tts.DefaultConfig(),
		tts.WithAPIKey(os.Getenv("ELEVENLABS_API_KEY")),
		tts.WithVoice("pMsXg91Y39Z99L9Z9999"),
		tts.WithModel("eleven_multilingual_v2"),
		tts.WithEnableStreaming(true),
	)
	if err != nil {
		log.Fatalf("create streaming TTS provider: %v", err)
	}

	reader, err := streamProvider.StreamGenerate(ctx, "This sentence is being streamed chunk by chunk.")
	if err != nil {
		log.Fatalf("stream generate: %v", err)
	}

	// Read audio chunks as they arrive
	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			// Play or forward the audio chunk
			processAudioChunk(buf[:n])
		}
		if err != nil {
			break
		}
	}
```

Streaming synthesis reduces perceived latency from seconds to hundreds of milliseconds, which is critical for interactive voice agents where users expect near-immediate responses.

## Architecture

```
Text Input ──▶ ElevenLabs API ──▶ Audio Chunks
                    │                   │
                    │  (streaming)       ▼
                    │              StreamGenerate()
                    │                   │
                    ▼                   ▼
              GenerateSpeech()   io.Reader (chunked)
                    │
                    ▼
              []byte (complete)
```

## Verification

1. Set the `ELEVENLABS_API_KEY` environment variable.
2. Run the script to generate `output.mp3` and verify playback.
3. Adjust `stability` between 0.3 and 0.8 to hear the variation in delivery.
4. Compare `similarity_boost` at 0.5 vs 0.9 to evaluate voice fidelity.
5. Test streaming synthesis and verify that audio begins playing before the full text is synthesized.

## Next Steps

- [SSML Tuning for Expressive Speech](/tutorials/voice/ssml-tuning) -- Add pauses, emphasis, and prosody control
- [Native S2S with Amazon Nova](/tutorials/voice/s2s-amazon-nova) -- End-to-end voice-to-voice models
- [Voice Session Interruptions](/tutorials/voice/session-interruptions) -- Integrate TTS into full voice sessions
