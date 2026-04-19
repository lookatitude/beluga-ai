# Guide: Deploy on Docker

**Time:** ~15 minutes
**You will build:** a Docker Compose setup running a Beluga agent with Redis for sessions and NATS for the event bus.
**Prerequisites:** Docker Desktop or Docker Engine, [First Agent guide](./first-agent.md).

## What you'll learn

- Containerising a Beluga agent.
- Wiring Redis as a session service.
- Exposing REST/SSE and A2A endpoints.
- Graceful shutdown.

## Step 1 — the agent binary

```go
// cmd/agent/main.go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/lookatitude/beluga-ai/v2/agent"
    "github.com/lookatitude/beluga-ai/v2/llm"
    "github.com/lookatitude/beluga-ai/v2/memory"
    "github.com/lookatitude/beluga-ai/v2/runtime"

    _ "github.com/lookatitude/beluga-ai/v2/llm/providers/openai"
    _ "github.com/lookatitude/beluga-ai/v2/memory/stores/redis"
)

func main() {
    ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer cancel()

    model, err := llm.New("openai", llm.Config{
        "model":   os.Getenv("OPENAI_MODEL"),
        "api_key": os.Getenv("OPENAI_API_KEY"),
    })
    if err != nil {
        log.Fatalf("llm.New: %v", err)
    }

    store, err := memory.NewMessageStore("redis", memory.Config{
        "addr": os.Getenv("REDIS_ADDR"),
    })
    if err != nil {
        log.Fatalf("memory.NewMessageStore: %v", err)
    }

    a := agent.NewLLMAgent(
        agent.WithPersona(agent.Persona{Role: "assistant"}),
        agent.WithLLM(model),
        agent.WithMemory(memory.NewComposite(store)),
    )

    r := runtime.NewRunner(a,
        runtime.WithSessionService(runtime.RedisSessions(os.Getenv("REDIS_ADDR"))),
        runtime.WithRESTEndpoint("/api/chat"),
        runtime.WithA2A("/.well-known/agent.json"),
        runtime.WithPlugin(runtime.AuditPlugin()),
        runtime.WithPlugin(runtime.CostPlugin()),
        runtime.WithDrainTimeout(30 * time.Second),
    )

    log.Println("listening on :8080")
    if err := r.Serve(ctx, ":8080"); err != nil {
        log.Fatalf("serve: %v", err)
    }
    log.Println("graceful shutdown complete")
}
```

## Step 2 — Dockerfile

```dockerfile
# Dockerfile
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/agent ./cmd/agent

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/agent /agent
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/agent"]
```

## Step 3 — docker-compose.yml

```yaml
services:
  agent:
    build: .
    image: myorg/beluga-agent:v1
    environment:
      OPENAI_MODEL: gpt-4o
      OPENAI_API_KEY: ${OPENAI_API_KEY}
      REDIS_ADDR: redis:6379
      OTEL_EXPORTER_OTLP_ENDPOINT: http://otel-collector:4317
    ports:
      - "8080:8080"
    depends_on:
      - redis
      - otel-collector
    restart: unless-stopped
    stop_grace_period: 60s

  redis:
    image: redis:7-alpine
    volumes:
      - redis-data:/data
    restart: unless-stopped

  otel-collector:
    image: otel/opentelemetry-collector:latest
    volumes:
      - ./otel-config.yaml:/etc/otel-config.yaml
    command: --config /etc/otel-config.yaml
    ports:
      - "4317:4317"

volumes:
  redis-data:
```

## Step 4 — run

```bash
export OPENAI_API_KEY=sk-...
docker compose up --build
```

Test:

```bash
curl -N http://localhost:8080/api/chat \
  -H 'Content-Type: application/json' \
  -d '{"session_id":"demo","message":"hello"}'
```

Expected: SSE stream of events back, session persisted in Redis.

## Graceful shutdown

`stop_grace_period: 60s` in compose gives the runner time to drain. The runner listens for `SIGTERM`, stops accepting new requests, waits for active sessions to finish (up to `WithDrainTimeout`), flushes plugin state (audit, cost), and exits cleanly.

Verify by sending a request, then `docker compose stop agent` mid-response. You should see "graceful shutdown complete" and the response should finish before the container exits.

## Multi-agent setup

For a multi-agent system, add NATS:

```yaml
  nats:
    image: nats:2-alpine
    command: -js
    ports: ["4222:4222"]

  agent-a:
    image: myorg/beluga-agent:v1
    environment:
      AGENT_ID: agent-a
      NATS_URL: nats://nats:4222

  agent-b:
    image: myorg/beluga-agent:v1
    environment:
      AGENT_ID: agent-b
      NATS_URL: nats://nats:4222
```

Agents discover each other via A2A (`/.well-known/agent.json`) on their respective hostnames. NATS provides the async event bus for cross-agent notifications.

## Common mistakes

- **Forgetting `stop_grace_period`.** Default is 10s. Not enough to drain a long turn. Set explicitly.
- **Running without a non-root user.** Use the distroless nonroot base or add a `USER` line. Running as root in a container is a CVE waiting to happen.
- **Hardcoding API keys in Dockerfile.** Use environment variables and secrets management.
- **No health check.** Add a `healthcheck` to compose so Docker restarts unhealthy containers.
- **Ignoring `stdout`/`stderr`.** Distroless images still emit to stdout; make sure your log collector is picking them up.

## Related

- [17 — Deployment Modes](../architecture/17-deployment-modes.md) — Docker vs Kubernetes vs Temporal.
- [08 — Runner and Lifecycle](../architecture/08-runner-and-lifecycle.md) — graceful shutdown details.
- [Deploy on Kubernetes](./deploy-kubernetes.md) — the next step for production.
