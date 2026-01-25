# Multi-Provider Session Routing

Welcome, colleague! In this guide we'll integrate **multi-provider session routing** with Beluga AI voice: choosing STT, TTS, or S2S providers per session (e.g. by tenant, region, or feature flag) and wiring them into `pkg/voice/session`.

## What you will build

You will implement a router that selects STT, TTS, or S2S providers per session (e.g. by `session_id`, tenant, or A/B bucket) and creates a `VoiceSession` with the chosen providers. This allows different backends (OpenAI, Deepgram, Azure, etc.) per use case without code changes.

## Learning Objectives

- ✅ Route sessions to different STT/TTS/S2S providers based on context
- ✅ Create `VoiceSession` with selected providers
- ✅ Use registries or factories for provider creation
- ✅ Keep routing logic testable and config-driven

## Prerequisites

- Go 1.24 or later
- Beluga AI (`go get github.com/lookatitude/beluga-ai`)
- Multiple STT/TTS or S2S providers configured

## Step 1: Setup and Installation
bash
```bash
go get github.com/lookatitude/beluga-ai
```

## Step 2: Define Routing Context

Decide what drives provider selection. Example:
```
	type RouteContext struct {
		SessionID string
		TenantID  string
		Region    string
		Features  map[string]bool // e.g. "s2s": true
	}

## Step 3: Implement a Router
	import (
		"github.com/lookatitude/beluga-ai/pkg/voice/iface"
		s2siface "github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
	)

```go
	func SelectProviders(ctx context.Context, rc RouteContext) (stt iface.STTProvider, tts iface.TTSProvider, s2s s2siface.S2SProvider, err error) {
		if rc.Features["s2s"] {
			s2s, err = createS2SProvider(ctx, rc.TenantID, rc.Region)
			return nil, nil, s2s, err
		}
		stt, err = createSTTProvider(ctx, rc.TenantID, rc.Region)
		if err != nil {
			return nil, nil, nil, err
		}
		tts, err = createTTSProvider(ctx, rc.TenantID, rc.Region)
		return stt, tts, nil, err
	}
```

Use config or a lookup table to map `TenantID`/`Region` to concrete provider types and configs.

## Step 4: Create Session with Routed Providers
```text
go
go
	stt, tts, s2s, err := SelectProviders(ctx, routeCtx)
	if err != nil {
		return nil, err
	}
	if s2s != nil {
		return session.NewVoiceSession(ctx,
			session.WithS2SProvider(s2s),
			session.WithConfig(session.DefaultConfig()),
		)
	}
	return session.NewVoiceSession(ctx,
		session.WithSTTProvider(stt),
		session.WithTTSProvider(tts),
		session.WithConfig(session.DefaultConfig()),
	)
```

## Step 5: Optional VAD and Turn Detection

Add VAD and turn detection per route if needed (e.g. same provider for all, or selected by region):

```
	import (
		"github.com/lookatitude/beluga-ai/pkg/voice/vad"
		"github.com/lookatitude/beluga-ai/pkg/voice/turndetection"
	)

go
```go
	vad, _ := vad.NewProvider(ctx, "webrtc", vad.DefaultConfig())
	td, _ := turndetection.NewProvider(ctx, "heuristic", turndetection.DefaultConfig())
	sess, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(stt),
		session.WithTTSProvider(tts),
		session.WithVADProvider(vad),
		session.WithTurnDetector(td),
		session.WithConfig(session.DefaultConfig()),
	)
```

## Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| Routing key | Tenant, region, feature flags | Application-defined |
| Provider map | Tenant/region → STT/TTS/S2S config | Config file or DB |
| Fallback | Default provider when route missing | Application-defined |

## Common Issues

### "Provider not found for tenant"

**Problem**: Routing context references a tenant with no provider mapping.

**Solution**: Fall back to a default provider; log and metric for missing mappings.

### "Session creation fails after route"

**Problem**: Routed provider creation or config validation fails.

**Solution**: Validate provider config at startup or in a readiness check; return clear errors and retry where appropriate.

### "Switching provider mid-session"

**Problem**: Need to change STT/TTS mid-call (e.g. failover).

**Solution**: Session API does not support swapping providers mid-session. Failover typically requires a new session or reconnection; design routing for session start only.

## Production Considerations

- **Caching**: Reuse provider instances per tenant/region where possible; avoid creating new ones every session.
- **Observability**: Tag metrics and traces with route context (tenant, region) for debugging.
- **Testing**: Unit-test router with mock providers; integration-test one or two routes.

## Next Steps

- **[Voice Session Persistence](./voice-session-persistence.md)** — Persist and restore session state.
- **[Voice Backends](../../../cookbook/voice-backends.md)** — Backend providers (LiveKit, Vapi).
- **[Voice Sessions](../../../use-cases/voice-sessions.md)** — Session architecture.
