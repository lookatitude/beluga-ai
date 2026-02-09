---
title: Building a Scalable Voice Backend
description: Deploy a production-ready voice backend with concurrent session management, health checks, and pipeline configuration.
---

Production voice applications must handle many concurrent sessions with predictable latency, graceful degradation, and operational visibility. This tutorial demonstrates how to build a scalable voice backend using Beluga's backend package with session management, health monitoring, and configurable STT/TTS or S2S pipelines.

## What You Will Build

A production-ready voice backend that supports concurrent sessions, configurable pipelines (STT/TTS or S2S), health checks, and session lifecycle management.

## Prerequisites

- Go 1.23+
- API keys for your chosen voice providers
- Completion of [LiveKit and Vapi Integration](/tutorials/voice/livekit-vapi) is recommended

## Step 1: Configure the Backend

Use `vbiface.Config` to define the backend provider, pipeline type, concurrency limits, and observability settings.

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

	log.Println("Voice backend started")
}
```

### Pipeline Types

| Type             | Constant                     | Description                          |
|-----------------|------------------------------|--------------------------------------|
| STT + TTS       | `vbiface.PipelineTypeSTTTTS` | Traditional transcribe-process-synthesize |
| S2S             | `vbiface.PipelineTypeS2S`   | Direct speech-to-speech              |

For STT/TTS pipelines, also set `STTProvider` and `TTSProvider`:

```go
	cfg := &vbiface.Config{
		Provider:     "livekit",
		PipelineType: vbiface.PipelineTypeSTTTTS,
		STTProvider:  "deepgram",
		TTSProvider:  "elevenlabs",
		// ...
	}
```

## Step 2: Create Sessions

Each voice session represents one user interaction. Sessions are created with a `SessionConfig` that specifies the user, transport, and processing pipeline.

```go
	sessionCfg := &vbiface.SessionConfig{
		UserID:        "user-001",
		Transport:     "webrtc",
		ConnectionURL: "wss://your-app.example.com/voice",
		PipelineType:  vbiface.PipelineTypeS2S,
		AgentCallback: func(ctx context.Context, transcript string) (string, error) {
			return processWithAgent(ctx, transcript)
		},
	}

	sess, err := be.CreateSession(ctx, sessionCfg)
	if err != nil {
		log.Fatalf("create session: %v", err)
	}

	if err := sess.Start(ctx); err != nil {
		log.Fatalf("start session: %v", err)
	}
```

## Step 3: Health Checks and Capacity Management

Before creating new sessions, verify the backend is healthy and has capacity.

```go
// acceptSession checks backend health and capacity before creating a session.
func acceptSession(ctx context.Context, be vbiface.VoiceBackend, cfg *vbiface.SessionConfig) (vbiface.VoiceSession, error) {
	// Check health
	status, err := be.HealthCheck(ctx)
	if err != nil {
		return nil, fmt.Errorf("health check failed: %w", err)
	}
	if !status.Healthy {
		return nil, fmt.Errorf("backend is unhealthy")
	}

	// Check capacity
	backendCfg := be.GetConfig()
	if be.GetActiveSessionCount() >= backendCfg.MaxConcurrentSessions {
		return nil, fmt.Errorf("at capacity (%d/%d sessions)",
			be.GetActiveSessionCount(), backendCfg.MaxConcurrentSessions)
	}

	return be.CreateSession(ctx, cfg)
}
```

## Step 4: Session Lifecycle Management

Track and manage active sessions for operational visibility.

```go
// listActiveSessions returns all active sessions with their state.
func listActiveSessions(ctx context.Context, be vbiface.VoiceBackend) {
	sessions, err := be.ListSessions(ctx)
	if err != nil {
		log.Printf("list sessions: %v", err)
		return
	}

	for _, s := range sessions {
		log.Printf("Session %s: state=%s", s.GetID(), s.GetState())
	}
}

// closeSession gracefully terminates a session.
func closeSession(ctx context.Context, be vbiface.VoiceBackend, sessionID string) error {
	if err := be.CloseSession(ctx, sessionID); err != nil {
		return fmt.Errorf("close session %s: %w", sessionID, err)
	}
	log.Printf("Session %s closed", sessionID)
	return nil
}
```

## Step 5: Multiple Provider Support

The backend registry supports multiple providers. Register additional providers via blank imports and switch between them by changing the `Provider` field.

```go
import (
	_ "github.com/lookatitude/beluga-ai/voice/backend/providers/livekit"
	_ "github.com/lookatitude/beluga-ai/voice/backend/providers/vapi"
	_ "github.com/lookatitude/beluga-ai/voice/backend/providers/pipecat"
)

// createBackendForProvider creates a backend with the specified provider.
func createBackendForProvider(ctx context.Context, providerName string) (vbiface.VoiceBackend, error) {
	cfg := &vbiface.Config{
		Provider:              providerName,
		PipelineType:          vbiface.PipelineTypeS2S,
		S2SProvider:           "openai_realtime",
		MaxConcurrentSessions: 100,
		LatencyTarget:         500 * time.Millisecond,
		Timeout:               30 * time.Second,
		EnableTracing:         true,
		EnableMetrics:         true,
	}

	return backend.NewBackend(ctx, providerName, cfg)
}
```

## Architecture

```
                    ┌─────────────────────────┐
                    │    Voice Backend         │
                    │                          │
                    │  ┌────────────────────┐  │
HTTP/WS Request ──▶│  │ Session Manager    │  │
                    │  │  - CreateSession   │  │
                    │  │  - CloseSession    │  │
                    │  │  - ListSessions    │  │
                    │  │  - HealthCheck     │  │
                    │  └────────┬───────────┘  │
                    │           │               │
                    │  ┌────────▼───────────┐  │
                    │  │ Pipeline Orch.     │  │
                    │  │  STT/TTS or S2S    │  │
                    │  │  + VAD, Turn Det.  │  │
                    │  │  + Noise Cancel.   │  │
                    │  └────────────────────┘  │
                    └─────────────────────────┘
```

## Verification

1. Set environment variables for your chosen provider.
2. Start the backend and verify it reports as healthy.
3. Create a session and confirm `GetActiveSessionCount` increments.
4. Close the session and confirm the count decrements.
5. Attempt to exceed `MaxConcurrentSessions` and verify the rejection.

## Next Steps

- [LiveKit and Vapi Integration](/tutorials/voice/livekit-vapi) -- Provider-specific configuration details
- [Native S2S with Amazon Nova](/tutorials/voice/s2s-amazon-nova) -- S2S pipeline configuration
- [Sensitivity Tuning](/tutorials/voice/sensitivity-tuning) -- Tune the VAD and turn detection in your pipeline
