# Single Binary Deployment Guide

> **Learn how to build and deploy a self-contained Beluga AI application as a single binary with proper health checks, graceful shutdown, and observability.**

## Overview

This guide shows you how to deploy a Beluga AI application as a single, self-contained binary. This approach simplifies deployment, reduces dependencies, and makes your application easier to manage in production.

## Benefits of Single Binary Deployment

| Benefit | Description |
|---------|-------------|
| **Simplicity** | One file to deploy, no dependency hell |
| **Portability** | Runs on any compatible Linux/macOS/Windows system |
| **Fast Startup** | No package installation, immediate execution |
| **Easy Rollback** | Just run the previous binary |
| **Container-Friendly** | Minimal container images |

## Prerequisites

- Go 1.24+
- Beluga AI Framework
- (Optional) Docker for container builds
- (Optional) OpenTelemetry Collector for observability

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Single Binary                             │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │   HTTP      │  │   Health    │  │   Metrics   │         │
│  │   Server    │  │   Checks    │  │   Endpoint  │         │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘         │
│         │                │                │                  │
│         └────────────────┼────────────────┘                  │
│                          │                                   │
│              ┌───────────▼───────────┐                       │
│              │    Application Core    │                       │
│              ├───────────────────────┤                       │
│              │  • LLM Provider        │                       │
│              │  • Agents              │                       │
│              │  • Memory              │                       │
│              │  • Tools               │                       │
│              └───────────────────────┘                       │
│                          │                                   │
│              ┌───────────▼───────────┐                       │
│              │     Observability     │                       │
│              │  • OTEL Tracing       │                       │
│              │  • Prometheus Metrics │                       │
│              │  • Structured Logging │                       │
│              └───────────────────────┘                       │
└─────────────────────────────────────────────────────────────┘
```

## Building the Binary

### Standard Build

```bash
# Build for current platform
go build -o ai-service ./main.go

# Run
./ai-service
```

### Optimized Build

```bash
# Build with optimizations
CGO_ENABLED=0 go build \
    -ldflags="-s -w -X main.version=$(git describe --tags) -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -trimpath \
    -o ai-service \
    ./main.go

# Check size
ls -lh ai-service
```

### Cross-Compilation

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ai-service-linux-amd64 ./main.go

# Linux ARM64 (for AWS Graviton, Apple Silicon)
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o ai-service-linux-arm64 ./main.go

# macOS AMD64
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o ai-service-darwin-amd64 ./main.go

# Windows
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o ai-service-windows-amd64.exe ./main.go
```

## Configuration Management

### Environment Variables

The example supports configuration via environment variables:

```bash
# Required
export OPENAI_API_KEY=sk-...

# Optional - Server
export PORT=8080
export METRICS_PORT=9090
export SHUTDOWN_TIMEOUT=30s

# Optional - Observability
export OTEL_ENDPOINT=localhost:4317
export OTEL_SERVICE_NAME=ai-service
export LOG_LEVEL=info
export LOG_FORMAT=json

# Optional - LLM
export LLM_PROVIDER=openai
export LLM_MODEL=gpt-4
export LLM_TIMEOUT=30s
```

### Configuration File (Optional)

You can also use a configuration file:

```yaml
# config.yaml
server:
  port: 8080
  metrics_port: 9090
  shutdown_timeout: 30s

observability:
  otel_endpoint: localhost:4317
  service_name: ai-service
  log_level: info
  log_format: json

llm:
  provider: openai
  model: gpt-4
  timeout: 30s
```

Load with:

```bash
./ai-service --config config.yaml
```

## Health Checks

The application exposes health check endpoints:

| Endpoint | Purpose |
|----------|---------|
| `/health/live` | Liveness probe - is the process running? |
| `/health/ready` | Readiness probe - is the app ready for traffic? |
| `/health/startup` | Startup probe - has the app finished initialization? |

### Kubernetes Probes

```yaml
apiVersion: v1
kind: Pod
spec:
  containers:
    - name: ai-service
      livenessProbe:
        httpGet:
          path: /health/live
          port: 8080
        initialDelaySeconds: 5
        periodSeconds: 10
      readinessProbe:
        httpGet:
          path: /health/ready
          port: 8080
        initialDelaySeconds: 5
        periodSeconds: 5
      startupProbe:
        httpGet:
          path: /health/startup
          port: 8080
        failureThreshold: 30
        periodSeconds: 10
```

## Graceful Shutdown

The application handles shutdown signals gracefully:

1. **Receives SIGTERM/SIGINT**
2. **Stops accepting new requests** (returns 503)
3. **Completes in-flight requests** (up to timeout)
4. **Flushes telemetry data**
5. **Closes connections**
6. **Exits**

```bash
# Send graceful shutdown signal
kill -TERM $(pgrep ai-service)

# Logs show:
# {"level":"info","msg":"Received shutdown signal","signal":"terminated"}
# {"level":"info","msg":"Stopping HTTP server"}
# {"level":"info","msg":"Waiting for in-flight requests","count":5}
# {"level":"info","msg":"Flushing telemetry"}
# {"level":"info","msg":"Shutdown complete"}
```

## Docker Deployment

### Minimal Dockerfile

```dockerfile
# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w" \
    -trimpath \
    -o ai-service \
    ./examples/deployment/single_binary/main.go

# Runtime stage
FROM scratch

# Copy binary
COPY --from=builder /app/ai-service /ai-service

# Copy CA certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Expose ports
EXPOSE 8080 9090

# Run
ENTRYPOINT ["/ai-service"]
```

### Build and Run

```bash
# Build image
docker build -t ai-service:latest .

# Run container
docker run -d \
    --name ai-service \
    -p 8080:8080 \
    -p 9090:9090 \
    -e OPENAI_API_KEY=$OPENAI_API_KEY \
    -e OTEL_ENDPOINT=host.docker.internal:4317 \
    ai-service:latest
```

## Systemd Service

For non-containerized deployments:

```ini
# /etc/systemd/system/ai-service.service
[Unit]
Description=AI Service
After=network.target

[Service]
Type=simple
User=ai-service
Group=ai-service

ExecStart=/opt/ai-service/ai-service
Restart=always
RestartSec=5

# Environment
EnvironmentFile=/etc/ai-service/environment

# Security
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/ai-service

# Resource limits
MemoryMax=2G
CPUQuota=200%

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable ai-service
sudo systemctl start ai-service
sudo systemctl status ai-service
```

## Monitoring Integration

The binary exposes Prometheus metrics at `:9090/metrics`:

```bash
# Check metrics
curl http://localhost:9090/metrics
```

Key metrics:

- `ai_requests_total` - Request count by endpoint and status
- `ai_request_duration_seconds` - Request latency histogram
- `ai_tokens_total` - Token usage by direction
- `ai_active_requests` - Current in-flight requests
- `ai_errors_total` - Error count by type

## Running the Example

```bash
# Navigate to the example
cd examples/deployment/single_binary

# Set environment
export OPENAI_API_KEY=sk-...

# Build
go build -o ai-service ./main.go

# Run
./ai-service

# In another terminal, test
curl http://localhost:8080/health/ready
curl "http://localhost:8080/chat?q=Hello"
curl http://localhost:9090/metrics
```

## Related Resources

- **[Observability Tracing Guide](../../../docs/guides/observability-tracing.md)**: OTEL setup
- **[Monitoring Dashboards](../../../docs/use-cases/monitoring-dashboards.md)**: Prometheus/Grafana
- **[LLM Provider Integration](../../../docs/guides/llm-providers.md)**: Provider configuration
