# Custom Health Checks for AI Services

In this tutorial, you'll learn how to implement custom health checks to ensure your AI agents, LLM providers, and vector databases are production-ready.

## Learning Objectives

- ✅ Implement the `HealthChecker` interface
- ✅ Monitor LLM provider availability
- ✅ Check Vector Store connectivity
- ✅ Integrate with orchestration health monitoring

## Prerequisites

- Basic understanding of Beluga AI Core
- Go 1.24+

## Why Custom Health Checks?

Standard HTTP health checks only tell you if the process is running. In AI applications, you need to know:
- Is the OpenAI API reachable?
- Is my pgvector database connected?
- Is the local Ollama instance running?

## Step 1: The HealthChecker Interface

Most core components in Beluga AI implement the `HealthChecker` interface.
```go
type HealthChecker interface {
    CheckHealth(ctx context.Context) HealthResult
}

type HealthResult struct {
    Status  string                 `json:"status"` // "UP", "DOWN", "DEGRADED"
    Details map[string]interface{} `json:"details,omitempty"`
}
```

## Step 2: Implementing a Custom Checker

Let's create a health checker for a custom LLM wrapper.
```go
package main

import (
    "context"
    "github.com/lookatitude/beluga-ai/pkg/monitoring"
)

type MyAIService struct {
    llmProvider llms.Provider
}

func (s *MyAIService) CheckHealth(ctx context.Context) monitoring.HealthResult {
    details := make(map[string]any)
    status := "UP"

    // 1. Check LLM Connectivity
    err := s.llmProvider.Ping(ctx)
    if err != nil {
        status = "DOWN"
        details["llm_error"] = err.Error()
    } else {
        details["llm"] = "connected"
    }

    return monitoring.HealthResult{
        Status:  status,
        Details: details,
    }
}
```

## Step 3: Aggregating Health Checks

Use the `monitoring.HealthRegistry` to collect all component states.
```go
registry := monitoring.NewHealthRegistry()

// Register components
registry.Register("agent-1", myAgent)
registry.Register("vector-db", pgStore)
registry.Register("auth-vault", vaultProvider)

// Get global health
globalHealth := registry.CheckAll(ctx)
```

## Step 4: Exposing via HTTP

Integrate with your server's `/health` endpoint.
```go
func healthHandler(w http.ResponseWriter, r *http.Request) {
    result := registry.CheckAll(r.Context())

    

    if result.Status == "DOWN" {
        w.WriteHeader(http.StatusServiceUnavailable)
    }
    
    json.NewEncoder(w).Encode(result)
}
```

## Verification

1. Start your service.
2. Call `/health`. Verify it returns `UP`.
3. Simulate a failure (e.g., set an invalid API key).
4. Call `/health` again. Verify it returns `DOWN` or `DEGRADED` with details.

## Next Steps

- **[End-to-End Tracing](./monitoring-otel-tracing.md)** - Debug issues found by health checks.
- **[Deploying via REST](../higher-level/server-rest-deployment.md)** - Host your health endpoint.
