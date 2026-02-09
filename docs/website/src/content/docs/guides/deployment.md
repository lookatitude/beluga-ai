---
title: Deploying to Production
description: Deploy AI agents as REST APIs with HTTP adapters, resilience patterns, configuration management, and container orchestration.
---

Deploy Beluga agents as production-grade HTTP services with built-in resilience, configuration hot-reload, multi-tenant support, and container-ready architecture.

## HTTP Server Adapters

The `server` package provides the `ServerAdapter` interface for exposing agents over HTTP:

```go
type ServerAdapter interface {
	RegisterAgent(path string, a agent.Agent) error
	RegisterHandler(path string, handler http.Handler) error
	Serve(ctx context.Context, addr string) error
	Shutdown(ctx context.Context) error
}
```

### Basic REST API

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

The `resilience` package provides production resilience primitives:

### Circuit Breaker

Prevent cascading failures when downstream services are unhealthy:

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

Retry failed operations with exponential backoff:

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

Send duplicate requests and return the fastest response:

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

Control request throughput:

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

The `config` package supports loading configuration from files and environment variables with hot-reload:

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

Isolate tenants using context-based routing:

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

Handle signals for clean shutdown:

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
| **Observability** | Structured JSON logging |
| **Infrastructure** | Horizontal scaling via replicas |
| **Infrastructure** | Resource limits configured |
| **Infrastructure** | Graceful shutdown handling |

## Next Steps

- [Monitoring & Observability](/guides/observability/) — Production monitoring setup
- [Safety & Guards](/guides/safety-and-guards/) — Production safety configuration
- [Working with LLMs](/guides/working-with-llms/) — Multi-provider resilience
- [Memory System](/guides/memory-system/) — Production memory backends
