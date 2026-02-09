---
title: "Scaling Concurrent Voice Streams"
description: "Support many concurrent voice sessions with capacity management, health checks, and backpressure."
---

## Problem

You need to support many concurrent voice sessions (100+) on a single backend instance without exhausting resources, hitting provider limits, or degrading latency. You must enforce capacity limits, handle backpressure, and fail gracefully when at capacity.

## Solution

Use the voice backend with `MaxConcurrentSessions` and `HealthCheck`. Before creating a session, check `GetActiveSessionCount()` and reject or queue when at capacity. Emit OpenTelemetry metrics for active sessions, rejections, and health status.

## Code Example

```go
package main

import (
	"context"
	"errors"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/lookatitude/beluga-ai/voice/backend"
	vbiface "github.com/lookatitude/beluga-ai/voice/backend/iface"
	_ "github.com/lookatitude/beluga-ai/voice/backend/providers/livekit"
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
		counter.Add(ctx, 1)
	}
	return sess, nil
}
```

## Explanation

1. **Capacity check** -- `GetActiveSessionCount()` and `MaxConcurrentSessions` ensure you do not over-create. Reject or queue when at the limit.

2. **Health check** -- `HealthCheck` before create avoids starting sessions on an unhealthy backend and reduces cascading failures.

3. **Metrics** -- Record active sessions (and optionally rejections) via OpenTelemetry. Use for scaling decisions and alerting.

**Key insight:** The backend itself does not enforce `MaxConcurrentSessions`; your application must. Use `GetActiveSessionCount` and `HealthCheck` consistently at session creation time.

## Variations

### Queue Instead of Reject

When at capacity, enqueue create requests and process when a session closes. Use a worker pool and a queue (channel or job queue).

### Per-Tenant Limits

Maintain per-tenant active counts and enforce a lower cap per tenant while keeping a global `MaxConcurrentSessions`.

### Graceful Shutdown

On shutdown, stop accepting new sessions and wait for active sessions to drain (or timeout) before exiting.

## Related Recipes

- **[Voice Backends Configuration](./voice-backends)** -- Config and provider setup
- **[S2S Voice Metrics](./s2s-voice-metrics)** -- Custom metrics for voice systems
- **[LLM Error Handling](./llm-error-handling)** -- Similar error handling and retry patterns
