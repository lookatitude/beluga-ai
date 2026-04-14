---
title: "Deploy API — Dockerfile, Compose, and Health Endpoints"
description: "Deploy package API reference for Beluga AI. Multi-stage Dockerfile generation, Docker Compose manifest generation, and HTTP liveness/readiness health endpoints for agent services."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "deploy API, Dockerfile, Docker Compose, health endpoint, liveness, readiness, GenerateDockerfile, GenerateCompose, HealthEndpoint, Beluga AI, Go, reference"
---

```go
import "github.com/lookatitude/beluga-ai/deploy"
```

Package deploy provides utilities for packaging and running Beluga AI agents
as container workloads. It generates production-ready Dockerfiles and Docker
Compose manifests from typed configuration structs, and exposes HTTP handlers
for Kubernetes liveness and readiness probes.

## GenerateDockerfile

GenerateDockerfile produces a two-stage Dockerfile from DockerfileConfig. The
builder stage compiles the agent binary with CGO disabled. The final stage
uses a minimal runtime image (distroless by default) and runs as nonroot.

### DockerfileConfig

```go
type DockerfileConfig struct {
    BaseImage   string // default: "gcr.io/distroless/static-debian12"
    GoVersion   string // default: "1.23"; must match ^[0-9]+\.[0-9]+(\.[0-9]+)?$
    AgentConfig string // path to agent config file; required; no path traversal
    Port        int    // required; must be in [1, 65535]
}
```

All fields are validated before the Dockerfile is rendered. GoVersion and
BaseImage are checked against allowlists to prevent Dockerfile instruction
injection. AgentConfig is cleaned with filepath.Clean and rejected if it
resolves outside the build context or is absolute.

```go
import (
    "fmt"

    "github.com/lookatitude/beluga-ai/deploy"
)

dockerfile, err := deploy.GenerateDockerfile(deploy.DockerfileConfig{
    GoVersion:   "1.23",
    BaseImage:   "gcr.io/distroless/static-debian12",
    AgentConfig: "config/agent.yaml",
    Port:        8080,
})
if err != nil {
    fmt.Println("error:", err)
} else {
    fmt.Print(dockerfile)
}
```

The generated Dockerfile:

```dockerfile
# syntax=docker/dockerfile:1
FROM golang:1.23 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /agent ./cmd/agent

FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=builder /agent /app/agent
COPY config/agent.yaml /config/
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/agent"]
```

## GenerateCompose

GenerateCompose produces a Docker Compose v3.9 YAML manifest from ComposeConfig.
Each AgentDeployment becomes a service with a read-only config volume, a port
mapping to container port 8080, optional environment variables, and optional
depends_on declarations.

### ComposeConfig and AgentDeployment

```go
type ComposeConfig struct {
    Agents []AgentDeployment // at least one required
}

type AgentDeployment struct {
    Name        string            // required; must match ^[a-zA-Z0-9_-]+$
    ConfigPath  string            // required; no path traversal; bind-mounted at /config/
    Port        int               // required; host port mapped to container 8080
    DependsOn   []string          // must reference names declared in Agents
    Environment map[string]string // keys must be valid POSIX variable names
}
```

All service names, config paths, environment keys, and DependsOn references
are validated before rendering. Newlines are rejected in all string fields to
prevent YAML injection.

```go
import (
    "fmt"

    "github.com/lookatitude/beluga-ai/deploy"
)

yaml, err := deploy.GenerateCompose(deploy.ComposeConfig{
    Agents: []deploy.AgentDeployment{
        {
            Name:       "planner",
            ConfigPath: "config/planner.yaml",
            Port:       8080,
            Environment: map[string]string{
                "OPENAI_API_KEY": "${OPENAI_API_KEY}",
            },
        },
        {
            Name:       "executor",
            ConfigPath: "config/executor.yaml",
            Port:       8081,
            DependsOn:  []string{"planner"},
        },
    },
})
if err != nil {
    fmt.Println("error:", err)
} else {
    fmt.Print(yaml)
}
```

DependsOn references are validated against the set of declared service names;
referencing an undeclared service returns an error.

## HealthEndpoint

HealthEndpoint exposes HTTP handlers for Kubernetes liveness and readiness
probes. Health endpoints are intentionally unauthenticated — restrict network
access via infrastructure controls such as NetworkPolicy rather than
application-layer authentication.

```go
import (
    "context"
    "fmt"
    "net/http"

    "github.com/lookatitude/beluga-ai/deploy"
)

h := deploy.NewHealthEndpoint()

// Register a readiness check (e.g. database reachability).
h.AddCheck("db", func(ctx context.Context) error {
    return pingDatabase(ctx)
})

mux := http.NewServeMux()
mux.HandleFunc("/healthz", h.Healthz())
mux.HandleFunc("/readyz", h.Readyz())

if err := http.ListenAndServe(":8081", mux); err != nil {
    fmt.Println("server error:", err)
}
```

### Healthz

Healthz returns an http.HandlerFunc that always responds 200 OK with
`{"status":"ok"}`. It confirms the process is alive. Only GET and HEAD are
accepted; all other methods receive 405.

### Readyz

Readyz returns an http.HandlerFunc that runs all registered checks with a
5-second per-check deadline derived from the request context. The response is:

- **200 OK** — all checks passed; body includes per-check status.
- **503 Service Unavailable** — one or more checks failed; failing check names
  are listed but error details are suppressed to prevent information disclosure
  to unauthenticated callers.

Only GET and HEAD are accepted; all other methods receive 405.

```json
// 200 response when all checks pass:
{"status":"ok","checks":{"db":{"status":"ok"}}}

// 503 response when a check fails:
{"status":"fail","checks":{"db":{"status":"unhealthy"}}}
```

`AddCheck` is safe to call after the server has started. Checks are executed
in registration order. Log check errors internally via your observability
stack; do not surface them in the HTTP response.

## Related Packages

- `k8s` — Kubernetes operator that generates Deployment, Service, and HPA manifests from Agent and Team CRDs.
- `server` — HTTP adapters (gin, fiber, echo, chi, gRPC) that mount the health handlers on a shared mux.
- `o11y` — OpenTelemetry integration; instrument health check latency with OTel metrics.
- `config` — Load and validate agent configuration files referenced by AgentConfig and ConfigPath.
