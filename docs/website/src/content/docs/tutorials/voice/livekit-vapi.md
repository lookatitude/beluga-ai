---
title: Integrating with LiveKit and Vapi
description: Connect Beluga AI to production voice platforms like LiveKit and Vapi for WebRTC networking, room management, and scalable audio delivery.
---

Production voice applications require infrastructure for WebRTC networking, STUN/TURN servers, jitter buffering, and room management. Building this infrastructure from scratch is a multi-month effort that is tangential to the core value of your voice agent. Platforms like LiveKit and Vapi handle this infrastructure, allowing Beluga AI to focus on the intelligence layer -- STT, LLM reasoning, TTS, and conversation management. This tutorial demonstrates how to connect Beluga's voice backend to these platforms.

## What You Will Build

A voice agent that connects to a LiveKit room or Vapi endpoint, processes participant audio through Beluga's STT/TTS or S2S pipeline, and publishes agent audio back to the room.

## Prerequisites

- LiveKit API credentials (URL, API key, API secret) or Vapi API key
- Completion of [Native S2S with Amazon Nova](/tutorials/voice/s2s-amazon-nova) or a working STT/TTS configuration

## Why Third-Party Backends?

While Beluga provides STT, TTS, and S2S capabilities, production voice applications need additional infrastructure that is outside the scope of an AI framework:

- **WebRTC Networking** -- STUN/TURN servers, ICE candidates, jitter buffers. These handle NAT traversal, firewall bypass, and packet reordering that are required for real-time audio over the internet.
- **Room Management** -- Multiple participants, track subscriptions, media routing. Managing who hears whom and how audio is mixed requires a dedicated media server.
- **Global Edge Networks** -- Low-latency audio delivery across regions. Edge servers reduce the round-trip time between the user and the nearest processing node.
- **Recording and Monitoring** -- Session recording, quality metrics, debugging tools. Operational visibility is essential for diagnosing audio quality issues in production.

Beluga's backend package integrates with these platforms so your agent logic remains the same regardless of the transport layer. You can switch from LiveKit to Vapi by changing the provider name without rewriting your agent callback or pipeline configuration.

## Step 1: Configure the LiveKit Backend

Create a voice backend using the LiveKit provider. The backend uses Beluga's standard registry pattern -- the blank import registers the `"livekit"` factory, and `backend.NewBackend` creates a configured instance. The `vbiface.Config` struct centralizes all backend settings including provider credentials, pipeline type, concurrency limits, and observability flags.

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

Each voice session represents one participant interaction in the room. The `AgentCallback` is the integration point between the backend infrastructure and your application logic -- it receives the transcribed text and returns the agent's response. This separation means your agent logic is transport-agnostic: the same callback works with LiveKit, Vapi, or any other backend provider.

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

For S2S pipelines, the callback is optional since the S2S model handles the full audio-to-audio flow. The callback is primarily used with STT/TTS pipelines where the application needs to process the transcribed text before generating a response.

## Step 3: Handle Audio Events

The session provides methods to receive participant audio and send agent audio back. The backend orchestrator handles the full pipeline flow: incoming audio goes through noise cancellation, VAD, and STT; the transcript is passed to the agent callback; and the response is synthesized and sent back. You only need to bridge the session's audio channel with your transport layer.

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

## Step 4: Vapi Integration

Vapi provides a managed voice pipeline where Beluga acts as the "brain" (LLM) behind a Vapi call. In this model, Vapi handles the telephony, STT, and TTS; your application provides only the response logic. This architecture is useful when you want telephony support (PSTN, SIP) without managing telephony infrastructure, or when Vapi's built-in STT/TTS selection meets your needs and you want to minimize the components you operate.

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
