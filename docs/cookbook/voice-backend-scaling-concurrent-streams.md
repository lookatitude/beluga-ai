---
title: "Scaling Concurrent Voice Streams"
package: "voice/backend"
category: "voice"
complexity: "advanced"
---

# Scaling Concurrent Voice Streams

## Problem

You need to support many concurrent voice sessions (100+) on a single backend instance without exhausting resources, hitting provider limits, or degrading latency. You must respect `MaxConcurrentSessions`, handle backpressure, and fail gracefully when at capacity.

## Solution

Use `pkg/voice/backend` with `MaxConcurrentSessions` and `HealthCheck`. Before creating a session, check `GetActiveSessionCount()` and reject or queue when at capacity. Use a connection/session limiter (e.g. semaphore or middleware) that aligns with `MaxConcurrentSessions`. Emit OTEL metrics for active sessions, rejections, and health. This works because the backend and registry already support multiple providers and config-driven limits; you add application-level enforcement and observability.

## Code Example
```go
package main

import (
	"context"
	"errors"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	_ "github.com/lookatitude/beluga-ai/pkg/voice/backend/providers/livekit"
)

var (
	tracer  = otel.Tracer("beluga.voice.backend.scaling")
	meter   = otel.Meter("beluga.voice.backend")
	counter metric.Int64UpDownCounter
)

func init() {
	var err error
	counter, err = meter.Int64UpDownCounter("voice.backend.active_sessions")
	if err != nil {
		log.Printf("metrics: %v", err)
	}
}

func main() {
	ctx := context.Background()
	cfg := &vbiface.Config{
		Provider:              "livekit",
		PipelineType:          vbiface.PipelineTypeS2S,
		S2SProvider:           "openai_realtime",
		MaxConcurrentSessions: 100,
		ProviderConfig:        map[string]any{},
	}
	be, err := backend.NewBackend(ctx, "livekit", cfg)
	if err != nil {
		log.Fatalf("backend: %v", err)
	}
	defer be.Stop(ctx)

	// Create session with capacity check
	sc := &vbiface.SessionConfig{
		UserID:        "user-1",
		Transport:     "webrtc",
		ConnectionURL: "wss://example.com/voice",
		PipelineType:  vbiface.PipelineTypeS2S,
		AgentCallback: func(ctx context.Context, t string) (string, error) { return t, nil },
	}
	sess, err := createSessionWithLimit(ctx, be, sc, 100)
	if err != nil {
		log.Printf("create session: %v", err)
		return
	}
	_ = sess
}

func createSessionWithLimit(ctx context.Context, be vbiface.VoiceBackend, sc *vbiface.SessionConfig, max int) (vbiface.VoiceSession, error) {
	ctx, span := tracer.Start(ctx, "create_session_with_limit")
	defer span.End()

	n := be.GetActiveSessionCount()
	span.SetAttributes(attribute.Int("active_sessions", n))
	if n >= max {
		span.SetStatus(trace.StatusError, "at capacity")
		return nil, errors.New("voice backend at capacity")
	}

	status, err := be.HealthCheck(ctx)
	if err != nil || status == nil || !status.Healthy {
		span.RecordError(err)
		span.SetStatus(trace.StatusError, "unhealthy")
		return nil, errors.New("backend unhealthy")
	}

	sess, err := be.CreateSession(ctx, sc)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(trace.StatusError, err.Error())
		return nil, err
	}
	if counter != nil {
		_, _ = counter.Add(ctx, 1) // decrement with Add(-1) when closing session
	}
	return sess, nil
}
```

## Explanation

1. **Capacity check** — `GetActiveSessionCount()` and `MaxConcurrentSessions` ensure you don't over-create. Reject or queue when at limit.

2. **Health check** — `HealthCheck` before create avoids starting sessions on an unhealthy backend. Reduces cascading failures.

3. **Metrics** — Record active sessions (and optionally rejections) via OTEL. Use for scaling and alerting.

4. **OTEL** — Span attributes and status make it easy to debug capacity and health issues.

**Key insight:** The backend itself doesn't enforce `MaxConcurrentSessions`; your application must. Use `GetActiveSessionCount` and `HealthCheck` consistently at session creation time.

## Testing

```go
- Unit-test `createSessionWithLimit`: mock backend with fixed `GetActiveSessionCount` and `HealthCheck`; assert reject when at capacity and success when under.
- Load-test: create sessions up to limit, then one more; assert the extra is rejected.

## Variations

### Queue instead of reject

When at capacity, enqueue create requests and process when a session closes. Use a worker pool and a queue (e.g. channel or job queue).

### Per-tenant limits

Maintain per-tenant active counts and enforce a lower cap per tenant while keeping a global `MaxConcurrentSessions`.

### Graceful shutdown

On shutdown, stop accepting new sessions and wait for active sessions to drain (or timeout) before exiting.

## Related Recipes

- **[Voice Backends Configuration](./voice-backends.md)** — Config and provider setup.
- **[LiveKit Webhooks](../integrations/voice/backend/livekit-webhooks-integration.md)** — Lifecycle and cleanup.
- **[Voice Session Persistence](../integrations/voice/session/voice-session-persistence.md)** — Session state.

```
