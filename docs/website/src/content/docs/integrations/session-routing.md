---
title: Multi-Provider Session Routing
description: Route voice sessions to different STT, TTS, or S2S providers based on tenant, region, or feature flags.
---

Multi-provider session routing enables selecting different voice providers per session based on context such as tenant identity, geographic region, or feature flags. This guide covers implementing a routing layer on top of Beluga AI's `voice/session` package.

## Overview

Rather than hardcoding a single STT/TTS provider, session routing evaluates context at session creation time and selects the appropriate providers. This supports multi-tenant deployments, A/B testing, and regional optimization without code changes.

## Prerequisites

- Go 1.23 or later
- Multiple STT/TTS or S2S providers configured
- Routing criteria defined (tenant, region, feature flags)

## Installation

```bash
go get github.com/lookatitude/beluga-ai
```

## Configuration

### Define Routing Context

Establish what drives provider selection for each session:

```go
type RouteContext struct {
    SessionID string
    TenantID  string
    Region    string
    Features  map[string]bool
}
```

### Implement the Router

Select providers based on the routing context:

```go
import (
    "context"

    "github.com/lookatitude/beluga-ai/voice/stt"
    "github.com/lookatitude/beluga-ai/voice/tts"
    s2spkg "github.com/lookatitude/beluga-ai/voice/s2s"
)

func SelectProviders(ctx context.Context, rc RouteContext) (stt.Provider, tts.Provider, s2spkg.Provider, error) {
    if rc.Features["s2s"] {
        s2sProvider, err := createS2SProvider(ctx, rc.TenantID, rc.Region)
        return nil, nil, s2sProvider, err
    }
    sttProvider, err := createSTTProvider(ctx, rc.TenantID, rc.Region)
    if err != nil {
        return nil, nil, nil, err
    }
    ttsProvider, err := createTTSProvider(ctx, rc.TenantID, rc.Region)
    return sttProvider, ttsProvider, nil, err
}
```

Use a configuration file or database lookup to map tenant/region to concrete provider types and configurations.

## Usage

### Create Sessions with Routed Providers

```go
import "github.com/lookatitude/beluga-ai/voice/session"

sttProvider, ttsProvider, s2sProvider, err := SelectProviders(ctx, routeCtx)
if err != nil {
    return nil, err
}
if s2sProvider != nil {
    return session.NewVoiceSession(ctx,
        session.WithS2SProvider(s2sProvider),
        session.WithConfig(session.DefaultConfig()),
    )
}
return session.NewVoiceSession(ctx,
    session.WithSTTProvider(sttProvider),
    session.WithTTSProvider(ttsProvider),
    session.WithConfig(session.DefaultConfig()),
)
```

### Add VAD and Turn Detection per Route

Optionally attach VAD and turn detection providers selected by route:

```go
import (
    "github.com/lookatitude/beluga-ai/voice/vad"
    "github.com/lookatitude/beluga-ai/voice/turndetection"
)

vadProvider, err := vad.NewProvider(ctx, "webrtc", vad.DefaultConfig())
if err != nil {
    return nil, err
}
td, err := turndetection.NewProvider(ctx, "heuristic", turndetection.DefaultConfig())
if err != nil {
    return nil, err
}
sess, err := session.NewVoiceSession(ctx,
    session.WithSTTProvider(sttProvider),
    session.WithTTSProvider(ttsProvider),
    session.WithVADProvider(vadProvider),
    session.WithTurnDetector(td),
    session.WithConfig(session.DefaultConfig()),
)
```

## Configuration Reference

| Option       | Description                             | Default             |
|--------------|-----------------------------------------|---------------------|
| Routing key  | Tenant, region, feature flags           | Application-defined |
| Provider map | Tenant/region to STT/TTS/S2S config    | Config file or DB   |
| Fallback     | Default provider when route is missing  | Application-defined |

## Troubleshooting

### Provider not found for tenant

Fall back to a default provider. Log and emit metrics for missing mappings to identify configuration gaps.

### Session creation fails after routing

Validate provider configuration at startup or in a readiness check. Return clear errors with the routing context included for debugging.

### Switching providers mid-session

The session API does not support swapping providers during an active session. Provider failover requires creating a new session. Design routing logic to execute at session start only.

## Advanced Topics

### Production Deployment

- **Caching**: Reuse provider instances per tenant/region to avoid creating new ones for every session
- **Observability**: Tag metrics and traces with route context (tenant, region) for debugging
- **Testing**: Unit-test the router with mock providers; integration-test one or two representative routes

## Related Resources

- [Voice Session Persistence](/integrations/session-persistence)
- [Voice Services Overview](/integrations/voice-services)
