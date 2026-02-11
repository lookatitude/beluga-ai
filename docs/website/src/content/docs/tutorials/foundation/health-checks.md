---
title: Health Checks for AI Services
description: Implement health checks to monitor LLM providers, vector databases, and agent availability in production.
---

Standard HTTP health checks only indicate whether a process is running. AI applications require deeper checks — is the LLM provider reachable? Is the vector database connected? Is the local model server responding? These dependencies are external services with their own availability characteristics: LLM providers enforce rate limits and experience outages, vector databases can lose their indexes, and embedding services may silently return degraded results. Custom health checks answer these questions and surface problems before users encounter them.

## What You Will Build

A health check system that monitors LLM provider availability, aggregates component status, and exposes results via an HTTP endpoint.

## Prerequisites

- Go 1.23+
- Understanding of Beluga AI's core architecture

## Step 1: Define Health Check Types

The health check types follow Beluga AI's interface-first design principle — define a small interface (`HealthChecker`) and let each component implement it. The three-state model (`UP`, `DOWN`, `DEGRADED`) is deliberately simple because health endpoints are consumed by orchestrators (Kubernetes, load balancers) that make binary routing decisions. `DEGRADED` provides a middle ground for cases where the service works but at reduced capacity, such as when a fallback LLM provider is active instead of the primary.

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "sync"
)

// HealthStatus represents the health state of a component.
type HealthStatus string

const (
    StatusUp       HealthStatus = "UP"
    StatusDown     HealthStatus = "DOWN"
    StatusDegraded HealthStatus = "DEGRADED"
)

// HealthResult holds the health status of a single component.
type HealthResult struct {
    Status  HealthStatus       `json:"status"`
    Details map[string]any     `json:"details,omitempty"`
}

// HealthChecker is implemented by components that can report their health.
type HealthChecker interface {
    CheckHealth(ctx context.Context) HealthResult
}
```

## Step 2: Implement a Custom Health Checker

This health checker verifies LLM provider connectivity by sending a minimal request. The approach uses `WithMaxTokens(1)` to minimize cost while still exercising the full API path — authentication, model routing, and response generation. A simple TCP connection check would not catch authentication failures or model deprecation, both of which are common in production LLM deployments.

```go
import (
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
)

type LLMHealthChecker struct {
    model llm.ChatModel
}

func NewLLMHealthChecker(model llm.ChatModel) *LLMHealthChecker {
    return &LLMHealthChecker{model: model}
}

func (c *LLMHealthChecker) CheckHealth(ctx context.Context) HealthResult {
    details := map[string]any{
        "model_id": c.model.ModelID(),
    }

    // Send a minimal request to verify connectivity
    msgs := []schema.Message{
        schema.NewHumanMessage("ping"),
    }

    _, err := c.model.Generate(ctx, msgs, llm.WithMaxTokens(1))
    if err != nil {
        return HealthResult{
            Status:  StatusDown,
            Details: map[string]any{
                "model_id": c.model.ModelID(),
                "error":    err.Error(),
            },
        }
    }

    details["status"] = "connected"
    return HealthResult{
        Status:  StatusUp,
        Details: details,
    }
}
```

## Step 3: Build a Health Registry

The registry aggregates health checks from multiple components, following the same `Register()` + collection pattern used throughout Beluga AI. The `sync.RWMutex` protects concurrent access because health checks may be registered during application startup while the HTTP endpoint is already serving readiness probes. The `CheckAll` method uses a read lock so multiple health check requests can execute concurrently without blocking each other.

```go
type HealthRegistry struct {
    mu       sync.RWMutex
    checkers map[string]HealthChecker
}

func NewHealthRegistry() *HealthRegistry {
    return &HealthRegistry{
        checkers: make(map[string]HealthChecker),
    }
}

func (r *HealthRegistry) Register(name string, checker HealthChecker) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.checkers[name] = checker
}

type AggregateResult struct {
    Status     HealthStatus              `json:"status"`
    Components map[string]HealthResult   `json:"components"`
}

func (r *HealthRegistry) CheckAll(ctx context.Context) AggregateResult {
    r.mu.RLock()
    defer r.mu.RUnlock()

    components := make(map[string]HealthResult, len(r.checkers))
    overall := StatusUp

    for name, checker := range r.checkers {
        result := checker.CheckHealth(ctx)
        components[name] = result

        if result.Status == StatusDown {
            overall = StatusDown
        } else if result.Status == StatusDegraded && overall != StatusDown {
            overall = StatusDegraded
        }
    }

    return AggregateResult{
        Status:     overall,
        Components: components,
    }
}
```

The aggregation logic uses worst-case promotion: any `DOWN` component marks the overall status as `DOWN`, and any `DEGRADED` component marks it as `DEGRADED` unless another component is already `DOWN`. This conservative approach ensures that orchestrators route traffic away from instances with failed dependencies.

## Step 4: Expose via HTTP

The health endpoint maps directly to Kubernetes liveness and readiness probes. Returning `503 Service Unavailable` when any component is `DOWN` causes load balancers to stop routing traffic to this instance, which is the desired behavior when an LLM provider or database is unreachable.

```go
func healthHandler(registry *HealthRegistry) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        result := registry.CheckAll(r.Context())

        w.Header().Set("Content-Type", "application/json")
        if result.Status == StatusDown {
            w.WriteHeader(http.StatusServiceUnavailable)
        }

        if err := json.NewEncoder(w).Encode(result); err != nil {
            http.Error(w, "encoding error", http.StatusInternalServerError)
        }
    }
}

func main() {
    registry := NewHealthRegistry()

    // Register components
    // registry.Register("llm-openai", NewLLMHealthChecker(openaiModel))
    // registry.Register("vector-db", NewVectorDBChecker(pgStore))

    http.HandleFunc("/health", healthHandler(registry))

    fmt.Println("Health endpoint at :8080/health")
    if err := http.ListenAndServe(":8080", nil); err != nil {
        fmt.Printf("Server error: %v\n", err)
    }
}
```

## Verification

1. Start your service and register at least one health checker.
2. Call `GET /health` — verify it returns `{"status": "UP", ...}`.
3. Simulate a failure (invalid API key or unreachable endpoint).
4. Call `GET /health` again — verify it returns `"DOWN"` with error details.

## Next Steps

- [OpenTelemetry Tracing](/tutorials/foundation/otel-tracing) — Debug issues found by health checks
- [Prometheus and Grafana](/tutorials/foundation/prometheus-grafana) — Metrics visualization
