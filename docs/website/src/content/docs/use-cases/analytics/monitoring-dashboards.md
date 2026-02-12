---
title: AI Monitoring Dashboards
description: "Set up comprehensive monitoring dashboards for AI applications with OpenTelemetry, Prometheus, and Grafana. Track token costs and latency."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "AI monitoring dashboard, LLM observability, Prometheus Grafana, token metrics, AI operations, Beluga AI, Go, production monitoring"
---

AI applications in production generate metrics that traditional APM tools don't track: token consumption per request, LLM response latency (which varies 10x between simple and complex prompts), cost per feature, and model-specific error rates. Without purpose-built dashboards, teams cannot answer critical operational questions: "Why did LLM costs spike 3x this morning?", "Which agent is consuming the most tokens?", or "Is the rate limit on our Anthropic account affecting P95 latency?"

Standard HTTP metrics (request count, latency, error rate) miss the AI-specific signals that drive cost and quality. Token usage, model selection, guard pipeline rejections, and agent tool calls all need dedicated visualization to make informed operational decisions.

Implementing monitoring dashboards with OpenTelemetry, Prometheus, and Grafana provides real-time insights into these AI-specific metrics alongside standard application health indicators.

## Solution Architecture

Beluga AI provides built-in OpenTelemetry integration through the `o11y/` package, using GenAI semantic conventions (`gen_ai.*` attributes) for standardized metric naming. Applications export metrics to an OpenTelemetry Collector, which forwards them to Prometheus for storage and Grafana for visualization. Because all Beluga AI components emit metrics through the same OTel pipeline, a single dashboard can show end-to-end request flow from HTTP ingress through agent execution to LLM response.

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│ AI App       │───▶│ OTEL         │───▶│ Prometheus   │
│ (Agents,     │    │ Collector    │    │ (Metrics)    │
│  LLMs,       │    └──────────────┘    └──────┬───────┘
│  Memory)     │                                │
└──────────────┘                                ▼
                                         ┌──────────────┐
                                         │ Grafana      │
                                         │ (Dashboards) │
                                         └──────────────┘
```

## OpenTelemetry Metrics Setup

Configure OpenTelemetry metrics export with Prometheus:

```go
package main

import (
    "context"
    "log"
    "net/http"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/prometheus"
    "go.opentelemetry.io/otel/sdk/metric"
)

func initMetrics() (*metric.MeterProvider, http.Handler, error) {
    // Create Prometheus exporter
    exporter, err := prometheus.New()
    if err != nil {
        return nil, nil, err
    }

    // Create MeterProvider
    provider := metric.NewMeterProvider(
        metric.WithReader(exporter),
    )

    // Set as global
    otel.SetMeterProvider(provider)

    return provider, exporter, nil
}

func main() {
    ctx := context.Background()

    // Initialize metrics
    provider, promHandler, err := initMetrics()
    if err != nil {
        log.Fatal(err)
    }
    defer provider.Shutdown(ctx)

    // Expose metrics endpoint
    http.Handle("/metrics", promHandler)
    go func() {
        log.Println("Metrics server on :9090")
        http.ListenAndServe(":9090", nil)
    }()

    // Your application code...
}
```

## Application Metrics

Add application-specific metrics for LLM calls, agent operations, and token usage:

```go
package main

import (
    "context"
    "time"

    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
)

type AppMetrics struct {
    requestCounter   metric.Int64Counter
    latencyHist      metric.Float64Histogram
    tokenCounter     metric.Int64Counter
    activeRequests   metric.Int64UpDownCounter
    errorCounter     metric.Int64Counter
    costGauge        metric.Float64Counter
}

func NewAppMetrics(meter metric.Meter) (*AppMetrics, error) {
    m := &AppMetrics{}
    var err error

    m.requestCounter, err = meter.Int64Counter(
        "ai_requests_total",
        metric.WithDescription("Total AI requests"),
    )
    if err != nil {
        return nil, err
    }

    m.latencyHist, err = meter.Float64Histogram(
        "ai_request_duration_seconds",
        metric.WithDescription("AI request duration"),
        metric.WithUnit("s"),
    )
    if err != nil {
        return nil, err
    }

    m.tokenCounter, err = meter.Int64Counter(
        "ai_tokens_total",
        metric.WithDescription("Total tokens used"),
    )
    if err != nil {
        return nil, err
    }

    m.activeRequests, err = meter.Int64UpDownCounter(
        "ai_active_requests",
        metric.WithDescription("Currently active requests"),
    )
    if err != nil {
        return nil, err
    }

    m.errorCounter, err = meter.Int64Counter(
        "ai_errors_total",
        metric.WithDescription("Total errors"),
    )
    if err != nil {
        return nil, err
    }

    m.costGauge, err = meter.Float64Counter(
        "ai_cost_dollars",
        metric.WithDescription("Estimated cost in dollars"),
    )
    if err != nil {
        return nil, err
    }

    return m, nil
}

// RecordRequest records a complete request with all metrics
func (m *AppMetrics) RecordRequest(ctx context.Context,
    feature string,
    model string,
    duration time.Duration,
    inputTokens, outputTokens int,
    err error,
) {
    attrs := []attribute.KeyValue{
        attribute.String("feature", feature),
        attribute.String("model", model),
    }

    // Record request count
    status := "success"
    if err != nil {
        status = "error"
        m.errorCounter.Add(ctx, 1, metric.WithAttributes(
            append(attrs, attribute.String("error_type", getErrorType(err)))...,
        ))
    }
    attrs = append(attrs, attribute.String("status", status))

    m.requestCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
    m.latencyHist.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

    // Record tokens
    m.tokenCounter.Add(ctx, int64(inputTokens), metric.WithAttributes(
        attribute.String("direction", "input"),
        attribute.String("model", model),
    ))
    m.tokenCounter.Add(ctx, int64(outputTokens), metric.WithAttributes(
        attribute.String("direction", "output"),
        attribute.String("model", model),
    ))

    // Estimate cost
    cost := estimateCost(model, inputTokens, outputTokens)
    m.costGauge.Add(ctx, cost, metric.WithAttributes(
        attribute.String("model", model),
    ))
}
```

## Prometheus Configuration

Configure Prometheus to scrape the application metrics endpoint:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093

rule_files:
  - "alerts/*.yml"

scrape_configs:
  - job_name: 'ai-service'
    static_configs:
      - targets: ['ai-service:9090']
```

## Grafana Dashboards

Create dashboards to visualize key metrics. Use PromQL queries to aggregate and display data:

### Request Rate Dashboard

```json
{
  "title": "Request Rate",
  "type": "graph",
  "targets": [
    {
      "expr": "sum(rate(ai_requests_total[5m])) by (feature)",
      "legendFormat": "{{feature}}"
    }
  ]
}
```

### Latency Percentiles

```json
{
  "title": "P95 Latency",
  "type": "graph",
  "targets": [
    {
      "expr": "histogram_quantile(0.95, sum(rate(ai_request_duration_seconds_bucket[5m])) by (le, model))",
      "legendFormat": "{{model}}"
    }
  ]
}
```

### Token Usage

```json
{
  "title": "Token Usage by Feature",
  "type": "piechart",
  "targets": [
    {
      "expr": "sum(increase(ai_tokens_total[24h])) by (feature)"
    }
  ]
}
```

## Alerting

Configure alerts for critical conditions:

```yaml
groups:
  - name: ai-service-alerts
    rules:
      - alert: HighErrorRate
        expr: |
          sum(rate(ai_errors_total[5m]))
          / sum(rate(ai_requests_total[5m])) > 0.05
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "High error rate in AI service"
          description: "Error rate is {{ $value | humanizePercentage }}"

      - alert: HighLatency
        expr: |
          histogram_quantile(0.95,
            sum(rate(ai_request_duration_seconds_bucket[5m])) by (le)
          ) > 5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High P95 latency"
          description: "P95 latency is {{ $value | humanizeDuration }}"

      - alert: CostSpike
        expr: |
          sum(increase(ai_cost_dollars[1h]))
          > sum(increase(ai_cost_dollars[1h] offset 1d)) * 2
        for: 30m
        labels:
          severity: warning
        annotations:
          summary: "Unusual cost increase detected"
          description: "Hourly cost is 2x higher than same hour yesterday"
```

## Production Considerations

### Metric Cardinality

Limit high-cardinality labels to prevent metric explosion. Use finite label sets for user IDs and feature names:

```go
// Good: finite label set
attrs := []attribute.KeyValue{
    attribute.String("tier", getUserTier(userID)),  // "free", "pro", "enterprise"
    attribute.String("feature", feature),            // fixed set of features
}

// Bad: unbounded cardinality
attrs := []attribute.KeyValue{
    attribute.String("user_id", userID),  // millions of unique values
}
```

### Data Retention

Configure retention policies based on resolution and age:
- High-resolution metrics: 7 days
- Downsampled hourly: 90 days
- Downsampled daily: 2 years

### Dashboard Organization

Organize dashboards hierarchically:
1. **Overview**: System health, request rate, error rate, P95 latency
2. **Feature Detail**: Per-feature metrics, token usage, cost
3. **Debug**: Low-level traces, error logs, detailed breakdowns

## Related Resources

- [Observability Guide](/guides/observability/) for OpenTelemetry setup
- [PII Leakage Detection](/use-cases/pii-leakage-detection/) for security monitoring
- [Token Cost Attribution](/use-cases/token-cost-attribution/) for cost tracking
