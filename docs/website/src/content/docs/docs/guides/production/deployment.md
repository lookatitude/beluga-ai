---
title: Deploying to Production
description: "Deploy AI agents as production HTTP services with framework-agnostic server adapters, circuit breakers, retry policies, hot-reload, and Kubernetes manifests."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go, deployment, HTTP server, resilience, circuit breaker, Kubernetes, Docker"
---

Deploy Beluga agents as production-grade HTTP services with built-in resilience, configuration hot-reload, multi-tenant support, and container-ready architecture. The deployment patterns described here separate your agent logic from the HTTP framework, resilience policies, and infrastructure concerns, so each layer can be changed independently.

## HTTP Server Adapters

The `server` package provides the `ServerAdapter` interface that decouples agent logic from HTTP frameworks. This means the same agent works with `net/http`, Gin, Fiber, Echo, Chi, gRPC, or Connect without changing a single line of agent code. You choose the HTTP framework that matches your team's preferences and existing infrastructure, and the adapter handles the translation between HTTP requests and agent invocations.

This pattern follows the same design philosophy as the rest of Beluga: interfaces define the contract, implementations are swappable via the registry, and your application code depends only on the interface.

```go
type ServerAdapter interface {
	RegisterAgent(path string, a agent.Agent) error
	RegisterHandler(path string, handler http.Handler) error
	Serve(ctx context.Context, addr string) error
	Shutdown(ctx context.Context) error
}
```

### Basic REST API

The following example creates a complete HTTP service from an agent. The adapter automatically generates two endpoints for each registered agent: a synchronous invocation endpoint and a streaming endpoint using Server-Sent Events (SSE).

```go
import (
	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/server"
	_ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

func main() {
	ctx := context.Background()

	model, err := llm.New("openai", llm.ProviderConfig{
		APIKey: os.Getenv("OPENAI_API_KEY"),
		Model:  "gpt-4o",
	})
	if err != nil {
		log.Fatal(err)
	}

	a := agent.New("assistant",
		agent.WithLLM(model),
		agent.WithPersona(agent.Persona{Role: "helpful assistant"}),
	)

	adapter := server.NewStdlibAdapter(server.Config{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	})

	// Register agent endpoints
	err = adapter.RegisterAgent("/v1/chat", a)
	if err != nil {
		log.Fatal(err)
	}

	// Start server
	log.Println("Serving on :8080")
	if err := adapter.Serve(ctx, ":8080"); err != nil {
		log.Fatal(err)
	}
}
```

This creates two endpoints:
- `POST /v1/chat/invoke` — Synchronous invocation
- `POST /v1/chat/stream` — Server-Sent Events (SSE) streaming

### Framework Adapters

Each adapter follows the same `ServerAdapter` interface, so switching frameworks requires only changing the import and constructor call. The adapter handles framework-specific routing, middleware integration, and response writing.

| Adapter | Import Path | Framework |
|---------|-------------|-----------|
| `stdlib` | Built-in | `net/http` (default) |
| `gin` | `server/providers/gin` | Gin |
| `fiber` | `server/providers/fiber` | Fiber |
| `echo` | `server/providers/echo` | Echo |
| `chi` | `server/providers/chi` | Chi |
| `grpc` | `server/providers/grpc` | gRPC |
| `connect` | `server/providers/connect` | Connect-Go |

```go
import _ "github.com/lookatitude/beluga-ai/server/providers/gin"

adapter, err := server.New("gin", server.Config{
	ReadTimeout:  10 * time.Second,
	WriteTimeout: 30 * time.Second,
})
```

## Resilience Patterns

The `resilience` package provides production resilience primitives that protect your application from the failure modes common in AI services: provider outages, rate limiting, high tail latency, and traffic spikes. Each pattern addresses a specific failure mode, and they compose together as LLM middleware.

### Circuit Breaker

When a downstream service (like an LLM provider) is failing, continuing to send requests wastes resources and increases latency. The circuit breaker pattern monitors failure rates and "opens" the circuit after a threshold is reached, immediately failing subsequent requests instead of waiting for timeouts. After a cooldown period, the circuit enters a half-open state and allows a limited number of test requests through. If those succeed, the circuit closes and normal operation resumes.

```go
import "github.com/lookatitude/beluga-ai/resilience"

cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
	MaxFailures:     5,              // Open after 5 failures
	Timeout:         30 * time.Second, // Try half-open after 30s
	HalfOpenMax:     2,              // Allow 2 test requests in half-open
})

result, err := cb.Execute(ctx, func(ctx context.Context) (string, error) {
	return model.Generate(ctx, msgs)
})
```

### Retry with Backoff

Transient failures (network blips, temporary rate limits, provider-side errors) often resolve on their own within seconds. The retry pattern re-attempts failed operations with exponential backoff and jitter, which avoids thundering herd problems where all retries hit the provider simultaneously. The `IsRetryable` function from the `core` package ensures that only transient errors are retried; permanent failures (invalid API key, malformed request) fail immediately.

```go
retrier := resilience.NewRetrier(resilience.RetrierConfig{
	MaxAttempts:  3,
	InitialDelay: 100 * time.Millisecond,
	MaxDelay:     5 * time.Second,
	Multiplier:   2.0,
	Jitter:       true,
	IsRetryable:  core.IsRetryable,
})

result, err := retrier.Execute(ctx, func(ctx context.Context) (string, error) {
	resp, err := model.Generate(ctx, msgs)
	if err != nil {
		return "", err
	}
	return resp.Text(), nil
})
```

### Hedge Requests

LLM providers occasionally exhibit high tail latency: most requests complete in under a second, but a small percentage take 10 seconds or more. The hedge pattern addresses this by sending a duplicate request after a delay. Whichever request completes first wins, and the slower one is canceled. This dramatically reduces P99 latency at the cost of slightly higher average request volume.

```go
hedger := resilience.NewHedger(resilience.HedgerConfig{
	Delay:      200 * time.Millisecond, // Send hedge after 200ms
	MaxHedges:  1,                       // At most 1 hedge
})

result, err := hedger.Execute(ctx, func(ctx context.Context) (string, error) {
	resp, err := model.Generate(ctx, msgs)
	if err != nil {
		return "", err
	}
	return resp.Text(), nil
})
```

### Rate Limiting

Rate limiting protects downstream services from being overwhelmed by traffic spikes. This is especially important with LLM providers that enforce per-minute or per-second token limits, where exceeding the limit results in errors or temporary bans. The token bucket algorithm allows short bursts while enforcing a sustained rate over time.

```go
limiter := resilience.NewRateLimiter(resilience.RateLimiterConfig{
	Rate:  100,                // 100 requests per second
	Burst: 20,                 // Allow burst of 20
})

err := limiter.Wait(ctx) // Blocks until a token is available
if err != nil {
	// Context canceled or deadline exceeded
}
```

### Combining Resilience Patterns

Resilience patterns compose as LLM middleware using the standard `ApplyMiddleware` function. The composition order matters: middleware applies right-to-left, so the last middleware in the list becomes the outermost wrapper. In the example below, the execution order for each request is: rate limit check, then retry with backoff, then circuit breaker check, then the actual LLM call.

```go
// Compose resilience: rate limit → retry → circuit breaker → call
model = llm.ApplyMiddleware(model,
	resilience.AsLLMMiddleware(
		resilience.WithCircuitBreaker(cb),
		resilience.WithRetry(retrier),
		resilience.WithRateLimit(limiter),
	),
)
```

## Configuration Management

The `config` package supports loading configuration from files and environment variables with hot-reload capability. Environment variables override file values, following the twelve-factor app methodology. Hot-reload watches the configuration file for changes and invokes a callback when the configuration updates, enabling runtime adjustments without restarting the service.

```go
import "github.com/lookatitude/beluga-ai/config"

type AppConfig struct {
	LLMProvider string `json:"llm_provider" env:"LLM_PROVIDER" default:"openai"`
	LLMModel    string `json:"llm_model" env:"LLM_MODEL" default:"gpt-4o"`
	APIKey      string `json:"api_key" env:"OPENAI_API_KEY" required:"true"`
	Port        int    `json:"port" env:"PORT" default:"8080"`
	MaxTokens   int    `json:"max_tokens" env:"MAX_TOKENS" default:"4096"`
}

// Load from file with environment variable overrides
cfg, err := config.Load[AppConfig]("config.json")
if err != nil {
	log.Fatal(err)
}

// Enable hot-reload (watches file for changes)
watcher, err := config.Watch[AppConfig]("config.json", func(newCfg AppConfig) {
	log.Println("Configuration updated")
	// Reinitialize components with new config
})
if err != nil {
	log.Fatal(err)
}
defer watcher.Close()
```

## Docker Deployment

The multi-stage Dockerfile below produces a minimal container image with only the compiled binary and CA certificates. The `CGO_ENABLED=0` flag ensures a fully static binary that does not depend on system libraries, making it portable across Linux distributions.

### Dockerfile

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /agent ./cmd/agent

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /agent /agent
EXPOSE 8080
ENTRYPOINT ["/agent"]
```

### Docker Compose

This compose file sets up the agent alongside Redis (for memory/cache) and an OpenTelemetry Collector (for trace and metric export). The health check ensures that the container orchestrator restarts the agent if it becomes unresponsive.

```yaml
version: "3.9"
services:
  agent:
    build: .
    ports:
      - "8080:8080"
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - PORT=8080
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/healthz"]
      interval: 10s
      timeout: 5s
      retries: 3

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  otel-collector:
    image: otel/opentelemetry-collector:latest
    ports:
      - "4317:4317"  # gRPC
      - "4318:4318"  # HTTP
```

## Kubernetes Deployment

The Kubernetes manifests below deploy the agent with horizontal scaling, secrets management, resource limits, and health probes. The liveness probe detects process hangs, while the readiness probe ensures traffic is only routed to instances with healthy dependencies (LLM provider, vector store, cache).

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: beluga-agent
spec:
  replicas: 3
  selector:
    matchLabels:
      app: beluga-agent
  template:
    metadata:
      labels:
        app: beluga-agent
    spec:
      containers:
        - name: agent
          image: myregistry/beluga-agent:latest
          ports:
            - containerPort: 8080
          env:
            - name: OPENAI_API_KEY
              valueFrom:
                secretKeyRef:
                  name: llm-secrets
                  key: openai-api-key
            - name: OTEL_EXPORTER_OTLP_ENDPOINT
              value: "http://otel-collector:4317"
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: "1"
              memory: 512Mi
          livenessProbe:
            httpGet:
              path: /healthz
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /readyz
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: beluga-agent
spec:
  selector:
    app: beluga-agent
  ports:
    - port: 80
      targetPort: 8080
  type: ClusterIP
```

## Multi-Tenant Setup

Multi-tenant deployments serve multiple customers from the same service instance. Beluga supports tenant isolation through context-based routing, where each request carries a tenant identifier that flows through the entire pipeline. This identifier can be used to select tenant-specific models, apply per-tenant rate limits, and isolate memory stores.

```go
import "github.com/lookatitude/beluga-ai/core"

// Middleware to extract tenant from request
func tenantMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.Header.Get("X-Tenant-ID")
		if tenantID == "" {
			http.Error(w, "missing tenant ID", http.StatusBadRequest)
			return
		}
		ctx := core.WithTenant(r.Context(), core.Tenant{
			ID: tenantID,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
```

## Graceful Shutdown

Production services must handle shutdown signals cleanly to avoid dropping in-flight requests or losing buffered telemetry data. The pattern below uses Go's `signal.NotifyContext` to create a context that cancels on SIGINT or SIGTERM, giving the server time to finish active requests before the process exits. The tracer shutdown flushes any buffered spans to the exporter.

```go
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Initialize components...

	// Start server (blocks until context canceled)
	if err := adapter.Serve(ctx, ":8080"); err != nil {
		log.Printf("Server stopped: %v", err)
	}

	// Cleanup
	log.Println("Shutting down...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	tracerShutdown() // Flush pending traces
	log.Println("Shutdown complete")
}
```

## Production Checklist

This checklist covers the essential configuration for a production Beluga deployment. Each item maps to patterns described in the guides linked below.

| Category | Item |
|----------|------|
| **Security** | API keys in environment variables or secrets manager |
| **Security** | Guard pipeline for input/output validation |
| **Security** | HITL policies for high-risk operations |
| **Resilience** | Circuit breakers on all external calls |
| **Resilience** | Retry with backoff for LLM calls |
| **Resilience** | Rate limiting per tenant |
| **Observability** | OpenTelemetry tracing enabled |
| **Observability** | Prometheus metrics exported |
| **Observability** | Health check endpoints exposed |
| **Observability** | Structured JSON logging with trace correlation |
| **Infrastructure** | Horizontal scaling via replicas |
| **Infrastructure** | Resource limits configured |
| **Infrastructure** | Graceful shutdown handling |

## Next Steps

- [Monitoring & Observability](/docs/guides/production/observability/) — Production monitoring setup with distributed tracing
- [Safety & Guards](/docs/guides/production/safety-and-guards/) — Production safety configuration and guard pipelines
- [Working with LLMs](/docs/guides/working-with-llms/) — Multi-provider resilience and fallback strategies
- [Memory System](/docs/guides/memory-system/) — Production memory backends with Redis and PostgreSQL
