---
title: Prometheus and Grafana Setup
description: "Export Beluga AI metrics to Prometheus and visualize them in Grafana dashboards — token usage, LLM latency distributions, and error rates with OpenTelemetry in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go, tutorial, Prometheus, Grafana, metrics, OpenTelemetry, dashboards, monitoring"
---

Observability is essential for production AI applications. While traces show individual request flows, metrics provide aggregate visibility — request rates, error rates, token consumption, and latency distributions. These aggregate views reveal patterns that traces alone cannot: gradual latency degradation, increasing error rates from a specific provider, or token consumption trends that affect cost forecasting. Beluga AI's observability layer is built on OpenTelemetry, making it straightforward to export metrics to Prometheus and visualize them in Grafana.

## What You Will Build

A metrics pipeline that exports Beluga AI metrics to Prometheus and displays them in Grafana dashboards, covering token usage, latency, and error rates.

## Prerequisites

- Go 1.23+
- Docker (for Prometheus and Grafana)
- Understanding of [OpenTelemetry Tracing](/tutorials/foundation/otel-tracing)

## The Metrics Pipeline

The pipeline follows the standard OpenTelemetry architecture, where each layer has a single responsibility:

1. **Instrumentation** — Your code records metrics via OpenTelemetry meters
2. **SDK** — The OTel SDK aggregates metrics in memory
3. **Exporter** — An HTTP handler exposes metrics at `/metrics`
4. **Scraper** — Prometheus polls `/metrics` on a configured interval

This pull-based model (Prometheus scrapes your application) is preferred for production because it decouples metric collection from application performance — if Prometheus is temporarily unavailable, your application is unaffected.

## Step 1: Set Up the Prometheus Exporter

The Prometheus exporter bridges OpenTelemetry's metric API to Prometheus's text exposition format. It registers as a `metric.Reader` with the OTel SDK and serves collected metrics on an HTTP endpoint. The separate metrics port (`:2222`) isolates metrics traffic from application traffic, which is a common practice for security and load balancing.

```go
package main

import (
    "fmt"
    "log"
    "net/http"

    "go.opentelemetry.io/otel"
    promexporter "go.opentelemetry.io/otel/exporters/prometheus"
    "go.opentelemetry.io/otel/sdk/metric"
)

func setupMetrics() error {
    // Create the Prometheus exporter
    exporter, err := promexporter.New()
    if err != nil {
        return fmt.Errorf("create prometheus exporter: %w", err)
    }

    // Create and register the meter provider
    provider := metric.NewMeterProvider(
        metric.WithReader(exporter),
    )
    otel.SetMeterProvider(provider)

    // Serve the /metrics endpoint
    http.Handle("/metrics", exporter)

    go func() {
        fmt.Println("Metrics available at :2222/metrics")
        if err := http.ListenAndServe(":2222", nil); err != nil {
            log.Printf("metrics server error: %v", err)
        }
    }()

    return nil
}
```

## Step 2: Record Custom Metrics

Use the OpenTelemetry meter API to record application-specific metrics. Counters track cumulative totals (total requests, total errors), while histograms capture distributions (latency percentiles). The meter name (`"beluga-agent"`) groups related metrics and appears as a prefix in Prometheus, making it easy to filter dashboards to your application's metrics.

```go
import (
    "context"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    otelmetric "go.opentelemetry.io/otel/metric"
)

func recordMetrics(ctx context.Context) error {
    meter := otel.Meter("beluga-agent")

    // Counter: track total LLM requests
    requestCounter, err := meter.Int64Counter("llm_requests_total",
        otelmetric.WithDescription("Total number of LLM requests"),
    )
    if err != nil {
        return err
    }

    // Histogram: track response latency
    latencyHist, err := meter.Float64Histogram("llm_request_duration_seconds",
        otelmetric.WithDescription("LLM request duration in seconds"),
    )
    if err != nil {
        return err
    }

    // Record a request
    attrs := attribute.NewSet(
        attribute.String("model", "gpt-4o"),
        attribute.String("provider", "openai"),
    )

    requestCounter.Add(ctx, 1, otelmetric.WithAttributeSet(attrs))
    latencyHist.Record(ctx, 1.234, otelmetric.WithAttributeSet(attrs))

    return nil
}
```

## Step 3: Configure Prometheus

Create a `prometheus.yml` file. The `scrape_interval` of 15 seconds is a good default — shorter intervals increase storage cost, while longer intervals reduce alerting responsiveness.

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'beluga-agent'
    static_configs:
      - targets: ['host.docker.internal:2222']
```

Run Prometheus with Docker:

```bash
docker run -d --name prometheus \
  -p 9090:9090 \
  -v $(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml \
  prom/prometheus
```

## Step 4: Set Up Grafana

Run Grafana:

```bash
docker run -d --name grafana -p 3000:3000 grafana/grafana
```

Configure the data source:
1. Open `http://localhost:3000` (login: admin/admin)
2. Go to Configuration > Data Sources > Add data source
3. Select Prometheus
4. Set URL to `http://host.docker.internal:9090`
5. Save and test

## Step 5: Useful PromQL Queries

Create dashboards with these queries. Each query targets a specific operational concern for AI applications:

| Metric | PromQL |
|:---|:---|
| Request rate | `rate(llm_requests_total[1m])` |
| P95 latency | `histogram_quantile(0.95, rate(llm_request_duration_seconds_bucket[5m]))` |
| Error rate | `rate(llm_requests_total{status="error"}[1m])` |
| Token usage | `sum(rate(llm_tokens_total[5m])) by (model)` |

The P95 latency query is particularly important for AI applications because LLM response times have high variance — a model might respond in 500ms for simple queries but take 10 seconds for complex reasoning. Monitoring the 95th percentile surfaces these slow requests that affect user experience.

## Cardinality Guidelines

Be careful with metric labels. High-cardinality labels (user IDs, message content, request IDs) create millions of time series and can crash Prometheus. Use bounded, categorical labels:

- Model name (gpt-4o, claude-3-opus)
- Provider (openai, anthropic)
- Status (success, error)
- Tier (free, paid)

Avoid: user IDs, session IDs, message text, timestamps.

## Verification

1. Start your application with the metrics endpoint.
2. Verify `curl localhost:2222/metrics` returns Prometheus text format.
3. Open Prometheus UI at `http://localhost:9090` and query `llm_requests_total`.
4. Open Grafana at `http://localhost:3000` and create a dashboard.

## Next Steps

- [OpenTelemetry Tracing](/tutorials/foundation/otel-tracing) — Distributed trace visualization
- [Health Checks](/tutorials/foundation/health-checks) — Component health monitoring
