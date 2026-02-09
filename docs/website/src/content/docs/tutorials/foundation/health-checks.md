---
title: Health Checks for AI Services
description: Implement health checks to monitor LLM providers, vector databases, and agent availability in production.
---

Standard HTTP health checks only indicate whether a process is running. AI applications require deeper checks — is the LLM provider reachable? Is the vector database connected? Is the local model server responding? Custom health checks answer these questions.

## What You Will Build

A health check system that monitors LLM provider availability, aggregates component status, and exposes results via an HTTP endpoint.

## Prerequisites

- Go 1.23+
- Understanding of Beluga AI's core architecture

## Step 1: Define Health Check Types

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

Create a health checker for an LLM provider that verifies API connectivity:

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

Aggregate health checks from multiple components:

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

## Step 4: Expose via HTTP

Integrate the health registry with an HTTP endpoint:

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
