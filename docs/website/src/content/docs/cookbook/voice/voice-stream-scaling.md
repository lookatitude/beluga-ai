---
title: "Scaling Concurrent Voice Streams"
description: "Support many concurrent voice sessions with capacity management, health checks, and backpressure."
---

## Problem

You need to support many concurrent voice sessions (100+) on a single backend instance without exhausting resources, hitting provider limits, or degrading latency. You must enforce capacity limits, handle backpressure, and fail gracefully when at capacity. Voice backends consume significant resources per session: WebRTC connections, TTS/STT API rate limits, memory for audio buffers, and goroutines for frame processing. Without explicit capacity management, a surge of incoming sessions can overwhelm the backend, causing cascading failures, degraded quality for all users, or provider-side throttling that affects unrelated workloads.

Traditional load balancers alone do not solve this problem because they do not understand voice session semantics. A session that successfully establishes a connection might still fail mid-stream if the backend becomes overloaded. You need application-level capacity checks and health monitoring that account for active sessions, not just connection counts.

## Solution

Use the voice backend with `MaxConcurrentSessions` and `HealthCheck`. Before creating a session, check `GetActiveSessionCount()` and reject or queue when at capacity. Emit OpenTelemetry metrics for active sessions, rejections, and health status. This design enforces capacity limits at the application layer, where you have full visibility into session state and can make intelligent decisions about admission control.

The reasoning behind this pattern is defense in depth. `MaxConcurrentSessions` is a configuration parameter, not an enforced limit within the backend itself, because different deployments have different constraints: some prioritize throughput, others prioritize per-session quality. By requiring the application to check capacity explicitly, the framework encourages deliberate resource management. Health checks catch degraded backends before they accept new sessions, reducing wasted setup overhead and improving user experience through fast failure.

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

1. **Capacity check** -- `GetActiveSessionCount()` and `MaxConcurrentSessions` ensure you do not over-create. Reject or queue when at the limit. This check happens before `CreateSession`, avoiding wasted work. The capacity limit is tied to backend resources: for example, a LiveKit backend might support 100 concurrent WebRTC sessions based on available CPU and network bandwidth. The application enforces this limit because the backend itself has no way to predict how many sessions your specific workload can handle. This shifts responsibility to the operator, who knows their deployment constraints.

2. **Health check** -- `HealthCheck` before create avoids starting sessions on an unhealthy backend and reduces cascading failures. A backend may report unhealthy due to upstream provider failures (TTS API down), internal errors (goroutine leaks), or resource exhaustion (disk full for recordings). Checking health before session creation provides fast failure: the user receives an immediate error rather than experiencing a partially established session that fails mid-stream. This also prevents snowballing, where new sessions consume resources attempting to start, further degrading the backend.

3. **Metrics** -- Record active sessions (and optionally rejections) via OpenTelemetry. Use for scaling decisions and alerting. The UpDownCounter pattern tracks session lifecycle: increment on create, decrement on close. This metric feeds into autoscaling policies (scale up when approaching capacity), operational dashboards (visualize load over time), and alerts (page when rejections spike). Recording rejections separately helps distinguish between capacity-limited rejections (expected under load) and error-driven rejections (unhealthy backend), guiding troubleshooting.

**Key insight:** The backend itself does not enforce `MaxConcurrentSessions`; your application must. Use `GetActiveSessionCount` and `HealthCheck` consistently at session creation time. This design is deliberate: capacity management depends on deployment-specific factors like hardware, provider quotas, and SLAs. By making capacity checks explicit, the framework forces you to think about resource limits and fail gracefully rather than silently degrading.

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
