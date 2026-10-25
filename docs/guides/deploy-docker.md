# Guide: Deploy on Docker

**Time:** ~15 minutes
**You will build:** a Docker Compose setup running a Beluga agent behind a `runtime.Runner`, serving SSE over HTTP and an A2A Agent Card, with an OpenTelemetry collector.
**Prerequisites:** Docker Desktop or Docker Engine, [First Agent guide](./first-agent.md).

## What you'll learn

- Containerising a Beluga agent.
- Hosting the agent in a `runtime.Runner` with audit and cost plugins.
- Exposing an SSE chat endpoint and an A2A Agent Card.
- Graceful shutdown with `Runner.Shutdown`.

## Step 1 — the agent binary

`runtime.Runner` manages sessions, plugins and a worker pool, but it does not serve HTTP. This binary adds a small SSE handler around `Runner.Run` and mounts the A2A handler next to it.

```go
// cmd/agent/main.go
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/audit"
	"github.com/lookatitude/beluga-ai/v2/config"
	"github.com/lookatitude/beluga-ai/v2/cost"
	"github.com/lookatitude/beluga-ai/v2/llm"
	"github.com/lookatitude/beluga-ai/v2/protocol/a2a"
	"github.com/lookatitude/beluga-ai/v2/runtime"
	"github.com/lookatitude/beluga-ai/v2/runtime/plugins"
	"github.com/lookatitude/beluga-ai/v2/schema"
	"github.com/lookatitude/beluga-ai/v2/server"

	_ "github.com/lookatitude/beluga-ai/v2/llm/providers/openai"
)

type chatRequest struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

// chatHandler streams the Runner's events for one turn as SSE.
func chatHandler(r *runtime.Runner) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var body chatRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, req.Body, 1<<20)).Decode(&body); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		sse, err := server.NewSSEWriter(w)
		if err != nil {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}
		for event, err := range r.Run(req.Context(), body.SessionID, schema.NewHumanMessage(body.Message)) {
			if err != nil {
				_ = sse.WriteEvent(server.SSEEvent{Event: "error", Data: err.Error()})
				return
			}
			data, _ := json.Marshal(map[string]string{"type": string(event.Type), "text": event.Text})
			if sse.WriteEvent(server.SSEEvent{Event: string(event.Type), Data: string(data)}) != nil {
				return
			}
		}
	}
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	model, err := llm.New("openai", config.ProviderConfig{
		Provider: "openai",
		APIKey:   os.Getenv("OPENAI_API_KEY"),
		Model:    os.Getenv("OPENAI_MODEL"),
	})
	if err != nil {
		log.Fatalf("llm.New: %v", err)
	}

	a := agent.New("assistant",
		agent.WithPersona(agent.Persona{Role: "assistant"}),
		agent.WithLLM(model),
	)

	r := runtime.NewRunner(a,
		runtime.WithPlugins(
			plugins.NewAuditPlugin(audit.NewInMemoryStore()),
			plugins.NewCostTracking(cost.NewInMemoryTracker(), cost.Budget{}),
		),
		runtime.WithRunnerConfig(runtime.RunnerConfig{
			SessionTTL:              time.Hour,
			GracefulShutdownTimeout: 30 * time.Second,
		}),
	)

	card := a2a.AgentCard{Name: "assistant", Endpoint: "http://agent:8080"}
	mux := http.NewServeMux()
	mux.Handle("POST /api/chat", chatHandler(r))
	mux.Handle("/", a2a.NewServer(a, card).Handler()) // /.well-known/agent.json, /tasks
	srv := &http.Server{Addr: ":8080", Handler: mux, ReadHeaderTimeout: 10 * time.Second}

	go func() {
		log.Println("listening on :8080")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("serve: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx) // stop accepting requests, finish open ones
	if err := r.Shutdown(shutdownCtx); err != nil {
		log.Printf("runner drain: %v", err)
	}
	log.Println("graceful shutdown complete")
}
```

Sessions live in the Runner's default `InMemorySessionService`, so they do not survive a restart and are not shared between replicas. For shared sessions, implement `runtime.SessionService` over your store and pass it with `runtime.WithSessionService`. The in-memory audit store and cost tracker are placeholders in the same way.

## Step 2 — Dockerfile

```dockerfile
# Dockerfile
FROM golang:1.27-alpine AS build
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
      OTEL_EXPORTER_OTLP_ENDPOINT: http://otel-collector:4318
    ports:
      - "8080:8080"
    depends_on:
      - otel-collector
    restart: unless-stopped
    stop_grace_period: 60s

  otel-collector:
    image: otel/opentelemetry-collector:latest
    volumes:
      - ./otel-config.yaml:/etc/otel-config.yaml
    command: --config /etc/otel-config.yaml
    ports:
      - "4318:4318"
```

The binary above does not initialise an OTel exporter. Call `o11y.BootstrapFromEnv` at startup if you want its spans exported to the collector.

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
curl http://localhost:8080/.well-known/agent.json
```

Expected: an SSE stream of events for the chat request, and the Agent Card JSON.

## Graceful shutdown

`stop_grace_period: 60s` in compose gives the process time to drain. On `SIGTERM` the binary stops accepting new HTTP requests (`http.Server.Shutdown`), then `Runner.Shutdown` waits for in-flight turns in the worker pool, bounded by the context deadline (or `GracefulShutdownTimeout` when the context has none).

Verify by sending a request, then `docker compose stop agent` mid-response. You should see "graceful shutdown complete" and the response should finish before the container exits.

## Multi-agent setup

Run one container per agent and let them find each other through A2A:

```yaml
  agent-a:
    image: myorg/beluga-agent:v1
    environment:
      OPENAI_API_KEY: ${OPENAI_API_KEY}

  agent-b:
    image: myorg/beluga-agent:v1
    environment:
      OPENAI_API_KEY: ${OPENAI_API_KEY}
```

Each container serves its Agent Card at `/.well-known/agent.json` on its own hostname. Use `a2a.NewClient` from another agent to submit tasks to it.

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
