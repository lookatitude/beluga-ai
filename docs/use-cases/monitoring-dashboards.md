# Use Case: Monitoring Dashboards for AI Applications

> **How to set up comprehensive monitoring dashboards for Beluga AI applications using Prometheus and Grafana.**

## Overview

Your AI application is in production, handling thousands of requests per day. Users are reporting occasional slowdowns, but you can't see where the bottlenecks are. Your LLM costs are rising, but you don't know which features are consuming the most tokens. When errors occur, you're debugging blind.

This use case shows you how to build a comprehensive monitoring dashboard that answers these questions and more.

## Business Context

### The Challenge

A mid-sized SaaS company deployed an AI-powered customer support system using Beluga AI. After launch, they faced several challenges:

- **Unpredictable latency**: Response times varied from 500ms to 15 seconds with no clear pattern
- **Cost overruns**: LLM costs exceeded budget by 40% in the first month
- **Silent failures**: 5% of requests failed silently, frustrating users
- **No visibility**: Engineers couldn't diagnose issues without adding print statements

### The Solution

By implementing proper observability with Prometheus and Grafana, the team gained:

- **Real-time latency monitoring**: P50, P95, P99 latency metrics
- **Cost tracking**: Token usage per feature, per user tier
- **Error visibility**: Immediate alerts on error rate spikes
- **Capacity planning**: Traffic patterns for resource optimization

## Requirements

### Functional Requirements

| Requirement | Description |
|-------------|-------------|
| **Latency Tracking** | Track P50, P95, P99 latency for all LLM calls |
| **Error Monitoring** | Count and categorize all errors |
| **Token Usage** | Track input/output tokens per request |
| **Active Requests** | Monitor concurrent request count |
| **Health Status** | Real-time health of all components |

### Non-Functional Requirements

| Requirement | Target |
|-------------|--------|
| **Dashboard Load Time** | < 2 seconds |
| **Metric Retention** | 30 days detailed, 1 year aggregated |
| **Alert Latency** | < 30 seconds from event to alert |
| **Data Resolution** | 15-second intervals |

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         AI Application                               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │
│  │   Agents    │  │    LLMs     │  │   Memory    │                 │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘                 │
│         │                │                │                         │
│         └────────────────┼────────────────┘                         │
│                          │                                          │
│                    ┌─────▼─────┐                                    │
│                    │   OTEL    │                                    │
│                    │  Metrics  │                                    │
│                    └─────┬─────┘                                    │
└──────────────────────────┼──────────────────────────────────────────┘
                           │
                           ▼
            ┌──────────────────────────┐
            │   OpenTelemetry Collector │
            └──────────────┬───────────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
              ▼            ▼            ▼
      ┌───────────┐ ┌───────────┐ ┌───────────┐
      │ Prometheus│ │   Jaeger  │ │   Loki    │
      │ (Metrics) │ │ (Traces)  │ │  (Logs)   │
      └─────┬─────┘ └───────────┘ └───────────┘
            │
            ▼
      ┌───────────┐
      │  Grafana  │
      │(Dashboards)│
      └───────────┘
```

## Implementation

### Step 1: Configure OTEL Metrics Export

First, set up your application to export metrics:

```go
package main

import (
    "context"
    "log"
    "net/http"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/monitoring"
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

    // Initialize Beluga AI metrics
    meter := provider.Meter("beluga-ai")
    llms.InitMetrics(meter)

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

### Step 2: Add Custom Metrics

Add application-specific metrics:

```go
import (
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

// Record a complete request
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

    // Estimate cost (simplified)
    cost := estimateCost(model, inputTokens, outputTokens)
    m.costGauge.Add(ctx, cost, metric.WithAttributes(
        attribute.String("model", model),
    ))
}
```

### Step 3: Deploy Prometheus

Create `prometheus.yml`:

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
    metric_relabel_configs:
      - source_labels: [__name__]
        regex: 'go_.*'
        action: drop
```

### Step 4: Create Grafana Dashboards

#### Dashboard: AI Service Overview

```json
{
  "title": "AI Service Overview",
  "panels": [
    {
      "title": "Request Rate",
      "type": "graph",
      "targets": [
        {
          "expr": "sum(rate(ai_requests_total[5m])) by (feature)",
          "legendFormat": "{{feature}}"
        }
      ]
    },
    {
      "title": "P95 Latency",
      "type": "graph",
      "targets": [
        {
          "expr": "histogram_quantile(0.95, sum(rate(ai_request_duration_seconds_bucket[5m])) by (le, model))",
          "legendFormat": "{{model}}"
        }
      ]
    },
    {
      "title": "Error Rate",
      "type": "graph",
      "targets": [
        {
          "expr": "sum(rate(ai_errors_total[5m])) by (error_type)",
          "legendFormat": "{{error_type}}"
        }
      ]
    },
    {
      "title": "Active Requests",
      "type": "gauge",
      "targets": [
        {
          "expr": "sum(ai_active_requests)"
        }
      ]
    }
  ]
}
```

#### Dashboard: LLM Costs

```json
{
  "title": "LLM Cost Analysis",
  "panels": [
    {
      "title": "Daily Cost by Model",
      "type": "graph",
      "targets": [
        {
          "expr": "sum(increase(ai_cost_dollars[24h])) by (model)",
          "legendFormat": "{{model}}"
        }
      ]
    },
    {
      "title": "Token Usage by Feature",
      "type": "piechart",
      "targets": [
        {
          "expr": "sum(increase(ai_tokens_total[24h])) by (feature)"
        }
      ]
    },
    {
      "title": "Cost per Request",
      "type": "stat",
      "targets": [
        {
          "expr": "sum(increase(ai_cost_dollars[24h])) / sum(increase(ai_requests_total[24h]))"
        }
      ]
    }
  ]
}
```

### Step 5: Configure Alerts

Create `alerts/ai-service.yml`:

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

      - alert: HighActiveRequests
        expr: sum(ai_active_requests) > 100
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High number of active requests"
          description: "{{ $value }} active requests"
```

### Step 6: Deploy with Docker Compose

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  ai-service:
    build: .
    ports:
      - "8080:8080"
      - "9090:9090"
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317

  otel-collector:
    image: otel/opentelemetry-collector:latest
    ports:
      - "4317:4317"
    volumes:
      - ./otel-config.yaml:/etc/otel-config.yaml
    command: ["--config=/etc/otel-config.yaml"]

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9091:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - ./alerts:/etc/prometheus/alerts

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    volumes:
      - ./grafana/dashboards:/var/lib/grafana/dashboards
      - ./grafana/provisioning:/etc/grafana/provisioning
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin

  alertmanager:
    image: prom/alertmanager:latest
    ports:
      - "9093:9093"
    volumes:
      - ./alertmanager.yml:/etc/alertmanager/alertmanager.yml
```

## Results

After implementing the monitoring system, the team achieved:

### Visibility Improvements

| Metric | Before | After |
|--------|--------|-------|
| Mean time to detect issues | 30+ minutes | < 2 minutes |
| Root cause analysis time | 2+ hours | 15 minutes |
| Cost visibility | None | Real-time |

### Performance Improvements

- Identified that 80% of high-latency requests were due to a single slow tool
- Optimized token usage by detecting verbose prompts, reducing costs by 30%
- Implemented rate limiting based on real-time active request data

### Key Metrics Dashboard

The team now monitors:

- **Request volume**: Requests per second by feature
- **Latency percentiles**: P50, P95, P99 by model
- **Error rate**: Errors per second with breakdown by type
- **Token efficiency**: Tokens per request over time
- **Cost burn rate**: Real-time and projected daily cost

## Lessons Learned

### What Worked Well

1. **Starting simple**: Basic metrics first, then added complexity
2. **Alerting discipline**: Alert on symptoms, not causes
3. **Cost attribution**: Tagging requests by feature enabled budget allocation
4. **Dashboard hierarchy**: Overview → Detail → Debug flow

### What We'd Do Differently

1. **Earlier implementation**: Should have added observability from day one
2. **Standardized labels**: Inconsistent label naming caused confusion early on
3. **Log correlation**: Would add trace IDs to logs earlier

### Recommendations

1. **Use OTEL from the start**: Beluga AI has built-in support
2. **Tag by business dimension**: Feature, user tier, experiment variant
3. **Set up alerts early**: Don't wait for the first incident
4. **Review dashboards regularly**: Remove noise, add signal

## Related Resources

- **[Observability Tracing Guide](../guides/observability-tracing.md)**: Detailed tracing setup
- **[Single Binary Deployment](../../examples/deployment/single_binary/)**: Production deployment
- **[Batch Processing Use Case](./batch-processing.md)**: Monitoring batch workloads
- **[LLM Error Handling Cookbook](../cookbook/llm-error-handling.md)**: Error handling patterns
