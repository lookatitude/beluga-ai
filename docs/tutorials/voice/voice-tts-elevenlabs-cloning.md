# Cloning Voices with ElevenLabs

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this tutorial, we'll use ElevenLabs with Beluga AI to generate high-quality speech using cloned or professional voices. You'll learn how to configure voice stability, similarity settings, and implement streaming TTS for low-latency audio responses.

## Learning Objectives
- ✅ Configure the ElevenLabs TTS provider
- ✅ Use specific Voice IDs
- ✅ Adjust stability and similarity settings
- ✅ Implement streaming TTS for low-latency audio

## Introduction
Welcome, colleague! Standard AI voices are getting better, but sometimes you need a specific brand voice or a custom clone for your application. Let's look at how to integrate ElevenLabs to give our agents a truly professional and unique sound.

## Prerequisites

- ElevenLabs API Key
- At least one Voice ID (from your ElevenLabs Voice Lab)

## Step 1: Configure ElevenLabs

ElevenLabs offers high-fidelity voices but requires a `VoiceID`.
```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/voice/tts"
    "github.com/lookatitude/beluga-ai/pkg/voice/tts/providers/elevenlabs"
)

func main() {
    ctx := context.Background()

    // 1. Configure ElevenLabs
    config := &elevenlabs.Config{
        APIKey:  os.Getenv("ELEVENLABS_API_KEY"),
        VoiceID: "pMsXg91Y39Z99L9Z9999", // Your cloned voice ID
        // Model choice (multilingual v2 is excellent)
        ModelID: "eleven_multilingual_v2",
    }

    // 2. Create provider
    provider, err := elevenlabs.NewProvider(config)
    if err != nil {
        log.Fatal(err)
    }
}
```

## Step 2: Voice Settings (The "Clone" Feel)

Fine-tune how much the AI adheres to the original voice profile.
```
    config.Settings = &elevenlabs.VoiceSettings{
        Stability:       0.5, // 0.0 = random, 1.0 = consistent
        SimilarityBoost: 0.75, // 0.0 = low, 1.0 = high similarity to original
        Style:           0.0, // Exaggeration level
        UseSpeakerBoost: true,
    }

## Step 3: Generating Audio
```go
    // Generate full audio
    audio, err := provider.Synthesize(ctx, "Welcome to the future of voice agents.")
    if err != nil {
        log.Fatal(err)
    }


    // Save to file
    os.WriteFile("output.mp3", audio, 0644)
```

## Step 4: Streaming for Low Latency

In voice applications, waiting for the full audio takes too long. Use streaming.
```go
    // Stream audio chunks as they are generated
    stream, err := provider.StreamSynthesize(ctx, "This sentence is being streamed chunk by chunk.")
    if err != nil {
        log.Fatal(err)
    }

    for chunk := range stream {
        if chunk.Error != nil {
            fmt.Printf("Error: %v\n", chunk.Error)
            break
        }
        // Play audio chunk or pipe to transport
        processAudio(chunk.Data)
    }
```

## Verification

1. Run the script.
2. Listen to `output.mp3`.
3. Adjust `Stability` and `SimilarityBoost` to see how the tone changes.

## Next Steps

- **[SSML Tuning for Expressive Speech](./voice-tts-ssml-tuning.md)** - Add pauses and emphasis.
- **[Native S2S with Amazon Nova](./voice-s2s-amazon-nova.md)** - End-to-end voice models.
