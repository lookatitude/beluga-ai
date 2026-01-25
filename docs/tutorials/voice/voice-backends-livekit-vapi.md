# Integrating with Third-Party Voice Backends

In this tutorial, you'll learn how to connect Beluga AI to professional voice platforms like LiveKit and Vapi for production-grade voice applications.

## Learning Objectives

- ✅ Understand the Voice Backend architecture
- ✅ Connect to a LiveKit room
- ✅ Implement a Vapi-powered voice agent
- ✅ Manage participant audio tracks

## Prerequisites

- [Native S2S with Amazon Nova](./voice-s2s-amazon-nova.md)
- API Keys for LiveKit or Vapi

## Why Third-Party Backends?

While Beluga AI provides raw STT/TTS, platforms like LiveKit handle:
- **WebRTC Networking**: Stun/Turn servers, jitter buffers.
- **Room Management**: Multiple speakers, video support.
- **Scalability**: Global edge networks for audio.

## Step 1: Configure the Backend Provider
```go
package main

import (
    "github.com/lookatitude/beluga-ai/pkg/voice/backend"
    "github.com/lookatitude/beluga-ai/pkg/voice/backend/providers/livekit"
)

func main() {
    config := &livekit.Config{
        URL:       os.Getenv("LIVEKIT_URL"),
        APIKey:    os.Getenv("LIVEKIT_API_KEY"),
        APISecret: os.Getenv("LIVEKIT_API_SECRET"),
    }
    
    provider, _ := livekit.NewProvider(config)
}
```

## Step 2: Joining a Session
```text
go
go
    session, err := provider.JoinSession(ctx, "room-123", "ai-bot")
    if err != nil {
        log.Fatal(err)
    }
    defer session.Leave()
```

## Step 3: Connecting an Agent to the Room

The backend acts as the "ears and mouth" for your Beluga Agent.






    // Pipe backend audio events to your S2S model or STT
    session.OnTrackSubscribed(func(track *livekit.Track) \{
        // Handle incoming user audio
    })

```
    // Send agent audio back to the room
    session.PublishAudio(agentAudioBuffer)

## Step 4: Vapi Integration

Vapi provides an end-to-end managed voice pipeline. Beluga AI can act as the "Brain" (LLM) behind a Vapi call.






import "github.com/lookatitude/beluga-ai/pkg/voice/backend/providers/vapi"

// Vapi calls your webhook with user transcripts
// You respond with text or audio URLs
```

## Verification

1. Start a LiveKit room via their dashboard.
2. Run your Go application.
3. Verify "ai-bot" appears in the participant list.
4. Speak into the room and check logs for audio event detection.

## Next Steps

- **[Tuning VAD and Turn Detection](./voice-sensitivity-tuning.md)** - Optimize responsiveness.
- **[Real-time STT Streaming](./voice-stt-realtime-streaming.md)** - Use custom STT with backends.
