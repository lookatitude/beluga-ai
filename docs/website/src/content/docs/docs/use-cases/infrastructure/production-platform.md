---
title: Production AI Agent Platform
description: "Deploy AI agents at enterprise scale with observability, auth, resilience, and multi-tenancy. Complete production platform in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "production AI platform, enterprise agent deployment, multi-tenancy AI, agent observability, Beluga AI, Go, AI infrastructure"
---

Moving an AI agent from prototype to production exposes a set of cross-cutting concerns that prototypes ignore. Without retry logic, a single LLM timeout fails the entire user request. Without authorization, any user can invoke any agent with any tool. Without observability, debugging a slow response means reading logs and guessing. Without tenant isolation, one customer's data leaks into another's context window.

These concerns are not optional extras — they are the difference between a demo and a system that handles real traffic. Each layer (resilience, security, observability, multi-tenancy) must compose cleanly with the others: a retry should create child spans in the trace, tenant context should propagate through the guard pipeline, and rate limits should apply per-tenant rather than globally.

Beluga AI provides production-grade infrastructure packages designed to layer around the agent runtime, each following the same registry and middleware patterns so they compose without custom integration code.

## Platform Architecture

The platform layers Beluga AI's infrastructure packages around the agent runtime. Each layer is independent but context-aware — tenant IDs propagate through `context.Context`, OpenTelemetry spans flow through every layer, and resilience policies apply per-provider:

```
┌─────────────────────────────────────────────────────────────┐
│                    API Layer                                │
│  REST / gRPC / MCP Server / A2A Protocol                    │
└─────────────────────────────┬───────────────────────────────┘
                              │
┌─────────────────────────────┴───────────────────────────────┐
│                  Security Layer                             │
│  Auth (RBAC/ABAC) │ Guard Pipeline │ Rate Limiting          │
└─────────────────────────────┬───────────────────────────────┘
                              │
┌─────────────────────────────┴───────────────────────────────┐
│                  Agent Runtime                              │
│  Agent │ Planner │ Executor │ Tools │ Memory │ Handoffs     │
└─────────────────────────────┬───────────────────────────────┘
                              │
┌─────────────────────────────┴───────────────────────────────┐
│                  Resilience Layer                            │
│  Retry │ Circuit Breaker │ Hedge │ Timeout                   │
└─────────────────────────────┬───────────────────────────────┘
                              │
┌─────────────────────────────┴───────────────────────────────┐
│                  Observability Layer                         │
│  OpenTelemetry │ Traces │ Metrics │ Structured Logs          │
└─────────────────────────────────────────────────────────────┘
```

## OpenTelemetry Instrumentation

When an agent request is slow, you need to know whether the bottleneck is the LLM call, tool execution, retrieval, or guard pipeline. Beluga AI uses OpenTelemetry GenAI semantic conventions (`gen_ai.*` attributes) for all LLM and agent operations, creating a trace that flows through every layer — from HTTP request to agent execution to LLM call to tool invocation. This means standard observability tools (Jaeger, Grafana Tempo, Datadog) can visualize the full request lifecycle without custom instrumentation.

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/lookatitude/beluga-ai/o11y"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.39.0"
)

func setupObservability(ctx context.Context) (func(), error) {
    exporter, err := otlptracegrpc.New(ctx,
        otlptracegrpc.WithEndpoint("localhost:4317"),
        otlptracegrpc.WithInsecure(),
    )
    if err != nil {
        return nil, err
    }

    res, err := resource.New(ctx,
        resource.WithAttributes(
            semconv.ServiceName("agent-platform"),
            semconv.ServiceVersion("1.0.0"),
        ),
    )
    if err != nil {
        return nil, err
    }

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(res),
        sdktrace.WithSampler(sdktrace.AlwaysSample()),
    )
    otel.SetTracerProvider(tp)

    shutdown := func() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        tp.Shutdown(ctx)
    }

    return shutdown, nil
}
```

Agent and LLM spans are automatically created with GenAI attributes:

```go
// Automatic span attributes for LLM calls:
// gen_ai.system = "openai"
// gen_ai.request.model = "gpt-4"
// gen_ai.request.temperature = 0.7
// gen_ai.request.max_tokens = 1000
// gen_ai.response.model = "gpt-4-0613"
// gen_ai.usage.input_tokens = 150
// gen_ai.usage.output_tokens = 300

// Add custom business attributes
span.SetAttributes(
    attribute.String("tenant.id", tenantID),
    attribute.String("agent.id", agentID),
    attribute.String("agent.planner", "react"),
)
```

## Authentication and Authorization

Beluga AI's auth package supports RBAC (Role-Based Access Control) and ABAC (Attribute-Based Access Control) for fine-grained permission management.

```go
import (
    "github.com/lookatitude/beluga-ai/auth"
    _ "github.com/lookatitude/beluga-ai/auth/providers/rbac"
)

func setupAuth(ctx context.Context) (auth.Policy, error) {
    policy, err := auth.New("rbac", auth.Config{})
    if err != nil {
        return nil, fmt.Errorf("create auth policy: %w", err)
    }

    return policy, nil
}

// Middleware that checks authorization before agent execution
func authMiddleware(policy auth.Policy, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        userID := r.Header.Get("X-User-ID")
        agentID := r.URL.Query().Get("agent")

        allowed, err := policy.Authorize(r.Context(), userID, auth.PermAgentDelegate, agentID)
        if err != nil || !allowed {
            http.Error(w, "forbidden", http.StatusForbidden)
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

### Capability-Based Security

Restrict agent capabilities per tenant or user:

```go
// Check if the agent can access external APIs
allowed, err := policy.Authorize(ctx, agentID, auth.PermExternalAPI, "https://api.billing.example.com")
if !allowed {
    return fmt.Errorf("agent not authorized for external API access")
}

// Check if the agent can write to memory
allowed, err = policy.Authorize(ctx, agentID, auth.PermMemoryWrite, "archival")
if !allowed {
    return fmt.Errorf("agent not authorized for archival memory writes")
}
```

## Safety Guard Pipeline

LLM-powered systems face safety risks at multiple points: malicious prompts on input, sensitive data in outputs, and dangerous operations through tools. A single-stage filter misses threats that manifest at different stages. Beluga AI's guard pipeline screens content at three stages — input (before the LLM sees it), output (before the user sees it), and tool (before tools execute) — so each risk is caught at the appropriate point in the request lifecycle.

```go
import (
    "github.com/lookatitude/beluga-ai/guard"
    _ "github.com/lookatitude/beluga-ai/guard/providers/pii"
    _ "github.com/lookatitude/beluga-ai/guard/providers/toxicity"
)

func setupGuards(ctx context.Context) error {
    // Input guards: prevent prompt injection and toxic input
    injectionGuard, err := guard.New("prompt_injection", nil)
    if err != nil {
        return err
    }

    // Output guards: prevent PII leakage
    piiGuard, err := guard.New("pii", nil)
    if err != nil {
        return err
    }

    // Tool guards: validate tool inputs
    toolGuard, err := guard.New("tool_auth", nil)
    if err != nil {
        return err
    }

    pipeline := guard.NewPipeline(
        guard.WithInputGuards(injectionGuard),
        guard.WithOutputGuards(piiGuard),
        guard.WithToolGuards(toolGuard),
    )

    // Use pipeline in agent execution
    _ = pipeline
    return nil
}
```

## Resilience

Wrap LLM and tool calls with retry, circuit breaker, and timeout:

```go
import "github.com/lookatitude/beluga-ai/resilience"

// Retry with exponential backoff and jitter
retryPolicy := resilience.RetryPolicy{
    MaxAttempts:     3,
    InitialBackoff:  500 * time.Millisecond,
    MaxBackoff:      10 * time.Second,
    BackoffFactor:   2.0,
    Jitter:          true,
    RetryableErrors: []core.ErrorCode{core.ErrRateLimit, core.ErrTimeout, core.ErrProvider},
}

answer, err := resilience.Retry(ctx, retryPolicy, func(ctx context.Context) (string, error) {
    return agent.Invoke(ctx, question)
})
if err != nil {
    log.Printf("Agent execution failed after retries: %v", err)
}
```

### Circuit Breaker

Prevent cascading failures when a provider is down:

```go
breaker := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
    MaxFailures:     5,
    ResetTimeout:    30 * time.Second,
    HalfOpenMax:     2,
})

result, err := breaker.Execute(ctx, func(ctx context.Context) (any, error) {
    return model.Generate(ctx, msgs)
})
```

## Multi-Tenancy

Isolate data and configuration per tenant using Beluga AI's tenant-aware context:

```go
import "github.com/lookatitude/beluga-ai/core"

// Set tenant context
ctx = core.WithTenant(ctx, core.Tenant{
    ID:   "acme-corp",
    Name: "Acme Corporation",
    Tier: "enterprise",
})

// Tenant ID propagates through all operations:
// - OpenTelemetry spans include tenant.id attribute
// - Memory operations are scoped to the tenant
// - Rate limits are applied per tenant
// - Auth policies are evaluated in tenant context

tenant := core.TenantFromContext(ctx)
log.Printf("Processing request for tenant: %s (tier: %s)", tenant.ID, tenant.Tier)
```

## Health Checks

Implement health check endpoints that verify all dependencies:

```go
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    checks := map[string]string{}

    // Check LLM provider
    _, err := model.Generate(ctx, []schema.Message{
        &schema.HumanMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: "ping"},
        }},
    }, llm.WithMaxTokens(1))
    if err != nil {
        checks["llm"] = "unhealthy: " + err.Error()
    } else {
        checks["llm"] = "healthy"
    }

    // Check vector store
    _, err = store.Search(ctx, make([]float32, 1536), 1)
    if err != nil {
        checks["vectorstore"] = "unhealthy: " + err.Error()
    } else {
        checks["vectorstore"] = "healthy"
    }

    // Return status
    healthy := true
    for _, status := range checks {
        if status != "healthy" {
            healthy = false
            break
        }
    }

    w.Header().Set("Content-Type", "application/json")
    if !healthy {
        w.WriteHeader(http.StatusServiceUnavailable)
    }
    json.NewEncoder(w).Encode(checks)
}
```

## Durable Workflows

For long-running, multi-step operations that must survive process restarts, use Beluga AI's workflow engine:

```go
import (
    "github.com/lookatitude/beluga-ai/workflow"
    _ "github.com/lookatitude/beluga-ai/workflow/providers/default"
)

executor, err := workflow.New("default", workflow.Config{})
if err != nil {
    log.Fatal(err)
}

handle, err := executor.Execute(ctx,
    func(ctx workflow.WorkflowContext, input any) (any, error) {
        // Step 1: Classify the request
        classification, err := ctx.Activity("classify", classifyActivity,
            workflow.WithActivityTimeout(30*time.Second),
        ).Result(ctx)
        if err != nil {
            return nil, err
        }

        // Step 2: Route to specialist
        result, err := ctx.Activity("process", processActivity,
            workflow.WithActivityRetry(resilience.DefaultRetryPolicy()),
        ).Result(ctx)
        if err != nil {
            return nil, err
        }

        return result, nil
    },
    workflow.WorkflowOptions{
        ID:      "support-" + requestID,
        Input:   request,
        Timeout: 30 * time.Minute,
    },
)
```

## API Server Setup

Expose agents through REST, gRPC, or MCP endpoints:

```go
import (
    "github.com/lookatitude/beluga-ai/server"
    _ "github.com/lookatitude/beluga-ai/server/providers/chi"
)

func setupServer(ctx context.Context, agents map[string]agent.Agent) error {
    srv, err := server.New("chi", server.Config{
        Host: "0.0.0.0",
        Port: 8080,
    })
    if err != nil {
        return err
    }

    // Agent execution endpoint
    srv.Handle("POST", "/api/v1/agents/{id}/invoke", func(w http.ResponseWriter, r *http.Request) {
        agentID := chi.URLParam(r, "id")
        a, ok := agents[agentID]
        if !ok {
            http.Error(w, "agent not found", http.StatusNotFound)
            return
        }

        var req struct {
            Input string `json:"input"`
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "invalid request", http.StatusBadRequest)
            return
        }

        result, err := a.Invoke(r.Context(), req.Input)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        json.NewEncoder(w).Encode(map[string]string{"result": result})
    })

    return srv.Start(ctx)
}
```

## Production Checklist

Before deploying to production, verify each layer:

- **Observability**: OpenTelemetry traces and metrics flowing to your collector (Jaeger, Grafana, Datadog)
- **Authentication**: RBAC/ABAC policies configured for all agent operations
- **Safety**: Guard pipeline active on all inputs, outputs, and tool calls
- **Resilience**: Retry policies and circuit breakers configured for all external calls
- **Rate Limiting**: Per-tenant and per-endpoint rate limits in place
- **Health Checks**: Liveness and readiness probes checking all dependencies
- **Multi-Tenancy**: Tenant isolation verified for memory, tools, and configuration
- **Logging**: Structured JSON logs with correlation IDs from OpenTelemetry
- **Error Handling**: Typed errors with `IsRetryable()` checks, graceful degradation
- **Deployment**: Stateless services behind a load balancer, horizontal scaling tested

## Related Resources

- [Observability Guide](/docs/guides/observability/) for OpenTelemetry configuration
- [Safety & Guards](/docs/guides/safety-and-guards/) for auth and guard patterns
- [Multi-Agent Customer Support](/docs/use-cases/multi-agent-support/) for agent orchestration patterns
- [Enterprise RAG Knowledge Base](/docs/use-cases/enterprise-rag/) for RAG in production
