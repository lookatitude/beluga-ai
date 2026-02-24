---
title: Infrastructure Integrations
description: "Deploy Beluga AI with Kubernetes, Vault, Redis, NATS, Auth0, and HTTP framework adapters like Gin, Fiber, Echo, and Chi."
sidebar:
  order: 0
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI infrastructure, Kubernetes deployment, HashiCorp Vault, Redis, NATS, Gin adapter, Go HTTP framework"
---

AI applications have the same infrastructure requirements as any production service -- secrets management, container orchestration, authentication, caching, and HTTP routing -- plus additional concerns around API key rotation, token budget management, and model failover. Beluga AI integrates with common infrastructure services so you can deploy with confidence using the tools your operations team already knows. This page covers the key infrastructure integrations.

## HTTP Framework Adapters

Beluga's `server` package provides adapters that mount Beluga handlers onto your existing HTTP framework.

### Available Adapters

| Framework | Adapter | Import Path |
|-----------|---------|-------------|
| Gin | `gin` | `server/adapters/gin` |
| Fiber | `fiber` | `server/adapters/fiber` |
| Echo | `echo` | `server/adapters/echo` |
| Chi | `chi` | `server/adapters/chi` |
| Huma | `huma` | `server/adapters/huma` |
| gRPC | `grpc` | `server/adapters/grpc` |
| Connect-Go | `connect` | `server/adapters/connect` |

### Gin Adapter

```go
package main

import (
    "log"

    "github.com/gin-gonic/gin"
    "github.com/lookatitude/beluga-ai/server"
    adapter "github.com/lookatitude/beluga-ai/server/adapters/gin"
)

func main() {
    router := gin.Default()

    belugaHandler := server.NewHandler(
        server.WithAgent(myAgent),
    )

    adapter.Mount(router, "/api/v1", belugaHandler)

    log.Fatal(router.Run(":8080"))
}
```

### Chi Adapter

```go
package main

import (
    "log"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/lookatitude/beluga-ai/server"
    adapter "github.com/lookatitude/beluga-ai/server/adapters/chi"
)

func main() {
    r := chi.NewRouter()

    belugaHandler := server.NewHandler(
        server.WithAgent(myAgent),
    )

    adapter.Mount(r, "/api/v1", belugaHandler)

    log.Fatal(http.ListenAndServe(":8080", r))
}
```

### Echo Adapter

```go
import (
    "github.com/labstack/echo/v4"
    adapter "github.com/lookatitude/beluga-ai/server/adapters/echo"
)

e := echo.New()
adapter.Mount(e, "/api/v1", belugaHandler)
```

### Fiber Adapter

```go
import (
    "github.com/gofiber/fiber/v2"
    adapter "github.com/lookatitude/beluga-ai/server/adapters/fiber"
)

app := fiber.New()
adapter.Mount(app, "/api/v1", belugaHandler)
```

### gRPC Adapter

```go
import (
    "google.golang.org/grpc"
    adapter "github.com/lookatitude/beluga-ai/server/adapters/grpc"
)

grpcServer := grpc.NewServer()
adapter.Register(grpcServer, belugaHandler)
```

## Server-Sent Events (SSE)

The `server` package includes built-in SSE support for streaming agent responses to browser clients:

```go
handler := server.NewHandler(
    server.WithAgent(myAgent),
    server.WithSSE(true),
)

// Client receives streaming responses at /api/v1/chat/stream
```

## Redis

Redis serves multiple roles in Beluga AI: caching, vector storage, distributed locking, and session state.

### Cache Integration

```go
import (
    "github.com/lookatitude/beluga-ai/cache"
    "github.com/lookatitude/beluga-ai/config"
)

c, err := cache.New("redis", config.ProviderConfig{
    Options: map[string]any{
        "address":  "localhost:6379",
        "password": os.Getenv("REDIS_PASSWORD"),
        "db":       0.0,
        "ttl":      "1h",
    },
})
if err != nil {
    log.Fatal(err)
}

// Use as LLM response cache
model = llm.ApplyMiddleware(model, cache.WithLLMCache(c))
```

### Redis Vector Store

See the [Vector Stores](/integrations/vector-stores) page for using Redis as a vector database.

## NATS Message Bus

NATS provides lightweight messaging for distributed agent communication and event streaming.

```go
import (
    "github.com/nats-io/nats.go"
)

nc, err := nats.Connect(os.Getenv("NATS_URL"))
if err != nil {
    log.Fatal(err)
}
defer nc.Close()

// Publish agent events
nc.Publish("agent.events", eventJSON)

// Subscribe to events from other agents
nc.Subscribe("agent.events", func(msg *nats.Msg) {
    // Handle event
})
```

NATS JetStream provides persistent message streaming with at-least-once delivery for durable agent workflows.

## Authentication

### JWT Authentication

Secure Beluga API endpoints with JWT validation:

```go
import "github.com/lookatitude/beluga-ai/auth"

// Configure JWT authentication
authenticator, err := auth.New("jwt", auth.Config{
    Issuer:   "https://auth.example.com",
    Audience: "beluga-api",
    JWKSURL:  "https://auth.example.com/.well-known/jwks.json",
})
if err != nil {
    log.Fatal(err)
}

// Apply as server middleware
handler := server.NewHandler(
    server.WithAgent(myAgent),
    server.WithMiddleware(server.WithAuth(authenticator)),
)
```

### RBAC and ABAC

Beluga's `auth` package supports role-based and attribute-based access control:

```go
import "github.com/lookatitude/beluga-ai/auth"

// Define policies
policy := auth.NewPolicy(
    auth.Allow("admin", "agent/*"),
    auth.Allow("user", "agent/chat"),
    auth.Deny("user", "agent/admin/*"),
)

// Check access
allowed, err := policy.Check(ctx, auth.Request{
    Subject:  userRole,
    Resource: "agent/chat",
    Action:   "invoke",
})
```

### Auth0 Integration

Use Auth0 as your identity provider with Beluga's JWT authentication:

```go
authenticator, err := auth.New("jwt", auth.Config{
    Issuer:   "https://your-tenant.auth0.com/",
    Audience: "https://api.example.com",
    JWKSURL:  "https://your-tenant.auth0.com/.well-known/jwks.json",
})
```

## HashiCorp Vault

Use Vault for managing API keys and secrets instead of environment variables.

```go
import (
    vault "github.com/hashicorp/vault/api"
)

client, err := vault.NewClient(vault.DefaultConfig())
if err != nil {
    log.Fatal(err)
}

// Read LLM API key from Vault
secret, err := client.Logical().Read("secret/data/beluga/openai")
if err != nil {
    log.Fatal(err)
}

apiKey := secret.Data["data"].(map[string]any)["api_key"].(string)

model, err := llm.New("openai", config.ProviderConfig{
    Model:  "gpt-4o",
    APIKey: apiKey,
})
```

For automated rotation, use Vault's dynamic secrets or combine with Beluga's config hot-reload:

```go
cfg, err := config.Load[AppConfig]("config.yaml",
    config.WithHotReload(true),
    config.WithReloadInterval(30 * time.Second),
)
```

## Kubernetes Deployment

### Helm Chart

Deploy Beluga AI applications with the standard container pattern:

```yaml
# deployment.yaml
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
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9090"
    spec:
      containers:
      - name: agent
        image: your-registry/beluga-agent:latest
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 9090
          name: metrics
        env:
        - name: OPENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: beluga-secrets
              key: openai-api-key
        livenessProbe:
          httpGet:
            path: /healthz
            port: http
          initialDelaySeconds: 5
        readinessProbe:
          httpGet:
            path: /healthz
            port: http
          initialDelaySeconds: 5
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

### Secrets Management

```yaml
# secrets.yaml
apiVersion: v1
kind: Secret
metadata:
  name: beluga-secrets
type: Opaque
stringData:
  openai-api-key: "sk-..."
  anthropic-api-key: "sk-ant-..."
  redis-password: "..."
```

For production, use External Secrets Operator to sync from Vault, AWS Secrets Manager, or GCP Secret Manager.

### Horizontal Pod Autoscaler

Scale based on request rate or queue depth:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: beluga-agent-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: beluga-agent
  minReplicas: 2
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

## Configuration Management

Beluga's `config` package loads configuration from files, environment variables, and supports hot-reload:

```go
import "github.com/lookatitude/beluga-ai/config"

type AppConfig struct {
    LLM struct {
        Provider string `json:"provider" required:"true"`
        Model    string `json:"model" default:"gpt-4o"`
        APIKey   string `json:"api_key" required:"true"`
    } `json:"llm"`
    Server struct {
        Port    int    `json:"port" default:"8080"`
        Host    string `json:"host" default:"0.0.0.0"`
    } `json:"server"`
}

cfg, err := config.Load[AppConfig]("config.yaml",
    config.WithEnvPrefix("BELUGA"),
    config.WithHotReload(true),
)
if err != nil {
    log.Fatal(err)
}
```

Environment variables override file values using the `BELUGA_` prefix:

```bash
export BELUGA_LLM_API_KEY="sk-..."
export BELUGA_SERVER_PORT="9090"
```

## Resilience

Beluga's `resilience` package provides circuit breakers, retries, hedging, and rate limiting for infrastructure integrations:

```go
import "github.com/lookatitude/beluga-ai/resilience"

// Circuit breaker for external APIs
cb := resilience.NewCircuitBreaker(resilience.CBConfig{
    MaxFailures:  5,
    ResetTimeout: 30 * time.Second,
})

// Retry with exponential backoff
retry := resilience.NewRetry(resilience.RetryConfig{
    MaxAttempts: 3,
    BaseDelay:   100 * time.Millisecond,
    MaxDelay:    5 * time.Second,
})

// Rate limiter
limiter := resilience.NewRateLimiter(resilience.RateConfig{
    Rate:  100,
    Burst: 20,
})
```

Apply as LLM middleware:

```go
model = llm.ApplyMiddleware(model,
    resilience.WithCircuitBreaker(cb),
    resilience.WithRetry(retry),
    resilience.WithRateLimit(limiter),
)
```
