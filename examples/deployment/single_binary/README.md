# Single Binary Deployment Example

This example demonstrates how to build and deploy a production-ready Beluga AI application as a single binary with proper health checks, configuration management, graceful shutdown, and OTEL instrumentation.

## Prerequisites

- **Go 1.24+**: Required for the Beluga AI framework
- **OTEL Collector** (optional): For exporting traces and metrics
- **Docker** (optional): For containerized deployment

## What You'll Learn

- Building a single binary with embedded assets
- Implementing health check endpoints
- Loading configuration from multiple sources
- Graceful shutdown with request draining
- OTEL setup for production observability

## Files

| File | Description |
|------|-------------|
| `main.go` | Production-ready deployment example |
| `main_test.go` | Test suite for deployment features |
| `deployment_guide.md` | Comprehensive deployment guide |

## Usage

### Build the Binary

```bash
go build -o beluga-server ./main.go
```

### Run with Configuration

```bash
# Using environment variables
export BELUGA_PORT=8080
export OPENAI_API_KEY=sk-...
./beluga-server

# Using config file
./beluga-server --config config.yaml
```

### Health Checks

```bash
# Liveness check
curl http://localhost:8080/healthz

# Readiness check
curl http://localhost:8080/readyz
```

### Graceful Shutdown

The server handles SIGTERM and SIGINT signals, draining requests before shutdown:

```bash
# Send shutdown signal
kill -TERM $(pidof beluga-server)
```

## Testing

```bash
go test -v ./...
```

## Configuration Options

| Option | Environment Variable | Default | Description |
|--------|---------------------|---------|-------------|
| Port | `BELUGA_PORT` | `8080` | HTTP server port |
| Shutdown Timeout | `BELUGA_SHUTDOWN_TIMEOUT` | `30s` | Graceful shutdown timeout |
| OTEL Endpoint | `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4317` | OTEL collector endpoint |

## Related Examples

- **[Monitoring Example](../../monitoring/basic/)**: Basic OTEL setup
- **[Config Formats Example](../../config/)**: Configuration patterns

## Related Documentation

- **[Observability Tracing Guide](../../../docs/guides/observability-tracing.md)**: Distributed tracing setup
- **[Monitoring Dashboards Use Case](../../../docs/use-cases/monitoring-dashboards.md)**: Prometheus and Grafana setup
