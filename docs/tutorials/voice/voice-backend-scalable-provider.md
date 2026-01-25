# Building a Scalable Voice Provider

**What you will build:** A production-ready voice backend using Beluga AI's `pkg/voice/backend` with a scalable provider (LiveKit, Vapi, Pipecat, Vocode, or Cartesia). You'll use `NewBackend`, `CreateSession`, `SessionConfig`, and pipeline types (STT/TTS or S2S) to support many concurrent sessions.

## Learning Objectives

- Create a voice backend with `backend.NewBackend` and `iface.Config`
- Use `CreateSession` with `SessionConfig` (UserID, Transport, ConnectionURL, PipelineType)
- Choose STT/TTS vs S2S pipeline and configure providers
- Scale concurrent sessions with `MaxConcurrentSessions` and health checks

## Prerequisites

- Go 1.24+
- Beluga AI (`go get github.com/lookatitude/beluga-ai`)
- [Voice Backends LiveKit & Vapi](./voice-backends-livekit-vapi.md) and API keys for your chosen provider

## Step 1: Configure the Backend

Use `iface.Config` with `Provider`, `PipelineType`, and provider-specific settings. For LiveKit, set `ProviderConfig` or use the provider's config builder.
```go
package main

import (
	"context"
	"log"
	"os"
	"time"

	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	_ "github.com/lookatitude/beluga-ai/pkg/voice/backend/providers/livekit"
)

func main() {
	ctx := context.Background()

	cfg := &vbiface.Config{
		Provider:                "livekit",
		PipelineType:            vbiface.PipelineTypeS2S,
		S2SProvider:             "openai_realtime",
		ProviderConfig:          map[string]any{
			"url":        os.Getenv("LIVEKIT_URL"),
			"api_key":    os.Getenv("LIVEKIT_API_KEY"),
			"api_secret": os.Getenv("LIVEKIT_API_SECRET"),
		},
		MaxConcurrentSessions:   100,
		LatencyTarget:           500 * time.Millisecond,
		Timeout:                 30 * time.Second,
		EnableTracing:           true,
		EnableMetrics:           true,
	}

	be, err := backend.NewBackend(ctx, "livekit", cfg)
	if err != nil {
		log.Fatalf("backend: %v", err)
	}
	defer be.Stop(ctx)
}
```

## Step 2: Create Sessions with SessionConfig

Each session needs `UserID`, `Transport` (webrtc or websocket), `ConnectionURL`, and `PipelineType`. Pass an `AgentCallback` or `AgentInstance` for STT/TTS or S2S pipelines.
```go
	sessionCfg := &vbiface.SessionConfig{
		UserID:        "user-1",
		Transport:     "webrtc",
		ConnectionURL: "wss://your-app.example.com/voice",
		PipelineType:  vbiface.PipelineTypeS2S,
		AgentCallback: func(ctx context.Context, transcript string) (string, error) {
			return "You said: " + transcript, nil
		},
	}

	sess, err := be.CreateSession(ctx, sessionCfg)
	if err != nil {
		log.Fatalf("create session: %v", err)
	}
	_ = sess
```

## Step 3: Health Checks and Session Count

Before creating sessions, call `HealthCheck` and respect `GetActiveSessionCount` / `MaxConcurrentSessions`.
```text
go
go
	status, err := be.HealthCheck(ctx)
	if err != nil || status == nil || !status.Healthy {
		log.Fatal("backend unhealthy")
	}
	if be.GetActiveSessionCount() >= cfg.MaxConcurrentSessions {
		log.Println("at capacity; reject or queue")
		return
	}
```

## Step 4: Multiple Providers (Vapi, Pipecat, Vocode, Cartesia)

Use the same pattern with `Provider: "vapi"`, `"pipecat"`, `"vocode"`, or `"cartesia"` and their `ProviderConfig` fields. Register providers via `_ "github.com/lookatitude/beluga-ai/pkg/voice/backend/providers/vapi"` etc.

## Verification

1. Set env vars (`LIVEKIT_URL`, `LIVEKIT_API_KEY`, `LIVEKIT_API_SECRET` or equivalent).
2. Run the app, create a session, and verify `GetActiveSessionCount` increases.
3. Call `CloseSession` and confirm the count decreases.

## Next Steps

- **[LiveKit Webhooks](../../integrations/voice/backend/livekit-webhooks-integration.md)** — Webhook handling.
- **[Vapi Custom Tools](../../integrations/voice/backend/vapi-custom-tools.md)** — Custom tools with Vapi.
- **[Scaling Concurrent Streams](../../cookbook/voice-backend-scaling-concurrent-streams.md)** — Production scaling.
