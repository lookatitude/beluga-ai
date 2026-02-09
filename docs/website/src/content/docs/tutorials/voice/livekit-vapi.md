---
title: Integrating with LiveKit and Vapi
description: Connect Beluga AI to production voice platforms like LiveKit and Vapi for WebRTC networking, room management, and scalable audio delivery.
---

Production voice applications require infrastructure for WebRTC networking, STUN/TURN servers, jitter buffering, and room management. Platforms like LiveKit and Vapi handle this infrastructure, allowing Beluga AI to focus on the intelligence layer. This tutorial demonstrates how to connect Beluga's voice backend to these platforms.

## What You Will Build

A voice agent that connects to a LiveKit room or Vapi endpoint, processes participant audio through Beluga's STT/TTS or S2S pipeline, and publishes agent audio back to the room.

## Prerequisites

- LiveKit API credentials (URL, API key, API secret) or Vapi API key
- Completion of [Native S2S with Amazon Nova](/tutorials/voice/s2s-amazon-nova) or a working STT/TTS configuration

## Why Third-Party Backends?

While Beluga provides STT, TTS, and S2S capabilities, production voice applications need:

- **WebRTC Networking** -- STUN/TURN servers, ICE candidates, jitter buffers
- **Room Management** -- Multiple participants, track subscriptions, media routing
- **Global Edge Networks** -- Low-latency audio delivery across regions
- **Recording and Monitoring** -- Session recording, quality metrics, debugging tools

Beluga's backend package integrates with these platforms so your agent logic remains the same regardless of the transport layer.

## Step 1: Configure the LiveKit Backend

Create a voice backend using the LiveKit provider.

```go
package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/lookatitude/beluga-ai/voice/backend"
	vbiface "github.com/lookatitude/beluga-ai/voice/backend/iface"
	_ "github.com/lookatitude/beluga-ai/voice/backend/providers/livekit"
)

func main() {
	ctx := context.Background()

	cfg := &vbiface.Config{
		Provider:     "livekit",
		PipelineType: vbiface.PipelineTypeS2S,
		S2SProvider:  "openai_realtime",
		ProviderConfig: map[string]any{
			"url":        os.Getenv("LIVEKIT_URL"),
			"api_key":    os.Getenv("LIVEKIT_API_KEY"),
			"api_secret": os.Getenv("LIVEKIT_API_SECRET"),
		},
		MaxConcurrentSessions: 100,
		LatencyTarget:         500 * time.Millisecond,
		Timeout:               30 * time.Second,
		EnableTracing:         true,
		EnableMetrics:         true,
	}

	be, err := backend.NewBackend(ctx, "livekit", cfg)
	if err != nil {
		log.Fatalf("create backend: %v", err)
	}
	defer be.Stop(ctx)

	if err := be.Start(ctx); err != nil {
		log.Fatalf("start backend: %v", err)
	}

	log.Println("Backend started")
}
```

## Step 2: Create a Session

Each voice session represents one participant interaction in the room.

```go
	sessionCfg := &vbiface.SessionConfig{
		UserID:        "user-001",
		Transport:     "webrtc",
		ConnectionURL: "wss://your-app.example.com/voice",
		PipelineType:  vbiface.PipelineTypeS2S,
		AgentCallback: func(ctx context.Context, transcript string) (string, error) {
			// Process transcript and return response
			return "You said: " + transcript, nil
		},
	}

	session, err := be.CreateSession(ctx, sessionCfg)
	if err != nil {
		log.Fatalf("create session: %v", err)
	}

	if err := session.Start(ctx); err != nil {
		log.Fatalf("start session: %v", err)
	}
	defer session.Stop(ctx)
```

The `AgentCallback` receives transcribed text from the participant and returns the agent's response. For S2S pipelines, the callback is optional since the S2S model handles the full audio-to-audio flow.

## Step 3: Handle Audio Events

The session provides methods to receive participant audio and send agent audio back.

```go
	// Process incoming audio from the participant
	go func() {
		for audio := range session.ReceiveAudio() {
			if err := session.ProcessAudio(ctx, audio); err != nil {
				log.Printf("process audio error: %v", err)
			}
		}
	}()
```

For STT/TTS pipelines, the backend orchestrator handles the full flow: incoming audio goes through noise cancellation, VAD, and STT; the transcript is passed to the agent callback; and the response is synthesized and sent back.

## Step 4: Vapi Integration

Vapi provides a managed voice pipeline where Beluga acts as the "brain" (LLM) behind a Vapi call. Vapi handles the telephony, STT, and TTS; your application provides the response logic.

```go
package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/lookatitude/beluga-ai/voice/backend"
	vbiface "github.com/lookatitude/beluga-ai/voice/backend/iface"
	_ "github.com/lookatitude/beluga-ai/voice/backend/providers/vapi"
)

func main() {
	ctx := context.Background()

	cfg := &vbiface.Config{
		Provider:     "vapi",
		PipelineType: vbiface.PipelineTypeSTTTTS,
		STTProvider:  "deepgram",
		TTSProvider:  "elevenlabs",
		ProviderConfig: map[string]any{
			"api_key": os.Getenv("VAPI_API_KEY"),
		},
		Timeout:       30 * time.Second,
		EnableTracing: true,
	}

	be, err := backend.NewBackend(ctx, "vapi", cfg)
	if err != nil {
		log.Fatalf("create Vapi backend: %v", err)
	}
	defer be.Stop(ctx)

	log.Println("Vapi backend ready for incoming calls")
}
```

## Architecture

```
┌───────────────────────────────────────────────────┐
│                   LiveKit Room                     │
│                                                    │
│  User ◀──WebRTC──▶ LiveKit Server ◀──▶ Beluga AI  │
│                                                    │
│         Audio In ──▶ [Pipeline] ──▶ Audio Out      │
│                     STT/TTS or S2S                 │
└───────────────────────────────────────────────────┘
```

## Verification

1. Start a LiveKit room via the LiveKit dashboard or CLI.
2. Run the Beluga application and verify the backend starts without errors.
3. Join the room as a participant and speak.
4. Confirm the agent responds with synthesized audio.
5. Check `GetActiveSessionCount()` reflects the active sessions.

## Next Steps

- [Sensitivity Tuning](/tutorials/voice/sensitivity-tuning) -- Optimize VAD and turn detection for your environment
- [Scalable Voice Backend](/tutorials/voice/scalable-backend) -- Scale to hundreds of concurrent sessions
- [Real-time STT Streaming](/tutorials/voice/stt-realtime-streaming) -- Use custom STT with backend providers
