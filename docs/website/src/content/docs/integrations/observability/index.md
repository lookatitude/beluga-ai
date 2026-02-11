---
title: Monitoring & Observability
description: Instrument Beluga AI with OpenTelemetry, Langfuse, LangSmith, Arize Phoenix, and structured logging.
sidebar:
  order: 0
---

AI applications are uniquely difficult to debug. An agent might produce the wrong answer because of a prompt issue, a retrieval miss, a tool error, or a model hallucination -- and without visibility into the full execution chain, you are guessing. Observability gives you the data to understand what happened, why it happened, and how much it cost.

Beluga AI provides built-in observability through OpenTelemetry (OTel) using GenAI semantic conventions, structured logging via `slog`, health checks, and LLM-specific trace exporters. The `o11y` package is the central integration point, and it works with any OTel-compatible backend.

## Observability Architecture

```
Beluga AI Application
  ├── OTel Traces (gen_ai.* attributes)
  │     ├── Jaeger / Tempo
  │     ├── Datadog APM
  │     └── Grafana Cloud
  ├── OTel Metrics (gen_ai.usage.*)
  │     ├── Prometheus
  │     └── Datadog Metrics
  ├── Structured Logs (slog)
  │     ├── stdout / stderr
  │     └── Log aggregator
  └── LLM Trace Exporters
        ├── Langfuse
        ├── LangSmith (Opik)
        └── Arize Phoenix
```

## OpenTelemetry Setup

Beluga uses OTel SDK v1.40.0 with GenAI semantic conventions (semconv v1.39.0).

### Basic OTel Configuration

```go
package main

import (
    "context"
    "log"

    "github.com/lookatitude/beluga-ai/o11y"
)

func main() {
    ctx := context.Background()

    // Initialize OTel with OTLP exporter
    shutdown, err := o11y.Init(ctx, o11y.Config{
        ServiceName:    "my-agent",
        ServiceVersion: "1.0.0",
        OTLPEndpoint:   "localhost:4317",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer shutdown(ctx)

    // All Beluga operations now emit traces and metrics automatically
}
```

### GenAI Attributes

Beluga traces use OTel GenAI semantic conventions:

| Attribute | Description | Example |
|-----------|-------------|---------|
| `gen_ai.system` | Provider system | `"openai"` |
| `gen_ai.operation.name` | Operation type | `"chat"`, `"embed"` |
| `gen_ai.request.model` | Requested model | `"gpt-4o"` |
| `gen_ai.response.model` | Actual model used | `"gpt-4o-2024-08-06"` |
| `gen_ai.usage.input_tokens` | Input tokens | `150` |
| `gen_ai.usage.output_tokens` | Output tokens | `89` |
| `gen_ai.agent.name` | Agent name | `"customer-support"` |
| `gen_ai.tool.name` | Tool invoked | `"search_database"` |

### Custom Span Attributes

```go
tracer := o11y.Tracer("my-agent")

ctx, span := tracer.Start(ctx, "process_request", o11y.Attrs{
    "gen_ai.operation.name": "chat",
    "gen_ai.request.model":  "gpt-4o",
    "tenant.id":             tenantID,
})
defer span.End()

// After LLM call completes
span.SetAttributes(
    o11y.AttrInputTokens, 150,
    o11y.AttrOutputTokens, 89,
)
```

## Metrics

Beluga emits OTel metrics for LLM operations, latency, and resource usage.

### Prometheus Integration

Configure Prometheus scraping with the OTel Prometheus exporter:

```go
shutdown, err := o11y.Init(ctx, o11y.Config{
    ServiceName:  "my-agent",
    MetricsPort:  9090,
    MetricsPath:  "/metrics",
})
```

Key metrics exposed:

| Metric | Type | Description |
|--------|------|-------------|
| `gen_ai_client_operation_duration` | Histogram | LLM call latency |
| `gen_ai_client_token_usage` | Counter | Token consumption |
| `gen_ai_server_request_duration` | Histogram | Server-side latency |
| `beluga_agent_invocations_total` | Counter | Agent execution count |
| `beluga_tool_calls_total` | Counter | Tool invocations |

### Grafana Dashboards

With Prometheus as a data source, build Grafana dashboards for:

- **LLM Performance**: Latency percentiles (p50, p95, p99) by model and provider
- **Token Usage**: Input/output tokens over time, cost estimation
- **Agent Activity**: Invocations, tool calls, error rates
- **RAG Pipeline**: Embedding latency, search latency, retrieval quality

Example PromQL queries:

```text
# P95 latency by model
histogram_quantile(0.95, rate(gen_ai_client_operation_duration_bucket[5m]))

# Token usage rate by provider
rate(gen_ai_client_token_usage[5m])

# Error rate
rate(beluga_agent_invocations_total{status="error"}[5m])
  / rate(beluga_agent_invocations_total[5m])
```

## LLM Trace Exporters

The `o11y` package provides a `TraceExporter` interface for sending detailed LLM call data to specialized observability platforms.

```go
type TraceExporter interface {
    ExportLLMCall(ctx context.Context, data LLMCallData) error
}
```

### Langfuse

Langfuse provides open-source LLM observability with prompt management and evaluation.

```go
import _ "github.com/lookatitude/beluga-ai/o11y/providers/langfuse"

exporter, err := o11y.NewTraceExporter("langfuse", config.ProviderConfig{
    Options: map[string]any{
        "public_key":  os.Getenv("LANGFUSE_PUBLIC_KEY"),
        "secret_key":  os.Getenv("LANGFUSE_SECRET_KEY"),
        "host":        "https://cloud.langfuse.com",
    },
})
```

### LangSmith (Opik)

```go
import _ "github.com/lookatitude/beluga-ai/o11y/providers/opik"

exporter, err := o11y.NewTraceExporter("opik", config.ProviderConfig{
    Options: map[string]any{
        "api_key":  os.Getenv("OPIK_API_KEY"),
        "project":  "my-project",
    },
})
```

### Arize Phoenix

Arize Phoenix provides open-source LLM tracing with embedding visualization.

```go
import _ "github.com/lookatitude/beluga-ai/o11y/providers/phoenix"

exporter, err := o11y.NewTraceExporter("phoenix", config.ProviderConfig{
    Options: map[string]any{
        "endpoint": "http://localhost:6006",
    },
})
```

### Multi-Exporter

Export to multiple backends simultaneously:

```go
multi := o11y.NewMultiExporter(langfuseExporter, phoenixExporter)

err := multi.ExportLLMCall(ctx, o11y.LLMCallData{
    Model:        "gpt-4o",
    Provider:     "openai",
    InputTokens:  150,
    OutputTokens: 89,
    Duration:     450 * time.Millisecond,
    Cost:         0.0023,
    Messages:     serializedMessages,
    Response:     serializedResponse,
})
```

## Structured Logging

Beluga uses Go's `slog` package for structured logging.

### Configuration

```go
import "log/slog"

// JSON output for production
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))
slog.SetDefault(logger)
```

### Log Attributes

Beluga middleware adds structured fields to log entries:

```go
model = llm.ApplyMiddleware(model, llm.WithLogging(logger))

// Produces logs like:
// {"level":"INFO","msg":"llm.generate","model":"gpt-4o","input_tokens":150,"output_tokens":89,"duration_ms":450}
```

## Health Checks

The `o11y` package provides health check endpoints for load balancers and orchestrators.

```go
health := o11y.NewHealthChecker()
health.Register("llm", func(ctx context.Context) error {
    _, err := model.Generate(ctx, []schema.Message{
        schema.NewUserMessage(schema.Text("ping")),
    })
    return err
})
health.Register("vectorstore", func(ctx context.Context) error {
    _, err := store.Search(ctx, zeroVec, 1)
    return err
})

// Expose at /healthz
http.Handle("/healthz", health.Handler())
```

## Datadog Integration

Datadog receives telemetry through the OTel Collector with the Datadog exporter:

```yaml
# otel-collector-config.yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317

exporters:
  datadog:
    api:
      key: ${DD_API_KEY}
    traces:
      span_name_as_resource_name: true

service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [datadog]
    metrics:
      receivers: [otlp]
      exporters: [datadog]
```

Point Beluga's OTLP endpoint to the collector:

```go
shutdown, err := o11y.Init(ctx, o11y.Config{
    ServiceName:  "my-agent",
    OTLPEndpoint: "localhost:4317",
})
```

## Choosing an Observability Stack

| Need | Recommended |
|------|------------|
| Full APM + infrastructure | Datadog |
| Open-source, self-hosted | Grafana + Tempo + Prometheus |
| LLM-specific debugging | Langfuse or Arize Phoenix |
| Quick local development | Arize Phoenix (local) |
| Enterprise with existing OTel | Your existing OTel collector |
