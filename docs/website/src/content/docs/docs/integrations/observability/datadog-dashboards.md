---
title: Datadog AI Dashboards
description: "Export Beluga AI OpenTelemetry metrics and traces to Datadog for LLM latency, token usage, and agent performance dashboards."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Datadog, AI dashboards, Beluga AI, OpenTelemetry Datadog, LLM monitoring, APM integration, Go observability"
---

## Overview

Datadog is the preferred observability platform for teams that need a unified view of AI operations alongside traditional infrastructure metrics. By routing Beluga AI's OpenTelemetry telemetry to Datadog, you get LLM latency, token usage, and agent performance data in the same dashboards where you monitor CPU, memory, and request rates. This eliminates tool-switching and enables cross-correlation between AI behavior and infrastructure health. This guide shows how to export that telemetry to Datadog and set up dashboards for monitoring AI application health and performance.

## Prerequisites

- Go 1.23 or later
- A Beluga AI application with the `o11y` package configured
- A Datadog account with an API key
- Datadog Agent installed (optional, for local development)

## Installation

Install the OpenTelemetry OTLP exporters for traces and metrics:

```bash
go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp
go get go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp
```

Set the required environment variables:

```bash
export DD_API_KEY="your-datadog-api-key"
export DD_OTLP_ENDPOINT="https://trace-intake.datadoghq.com"  # optional, this is the default
```

## Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `DD_API_KEY` | Datadog API key | - | Yes |
| `DD_OTLP_ENDPOINT` | Datadog OTLP intake endpoint | `https://trace-intake.datadoghq.com` | No |
| `DD_SITE` | Datadog site (e.g., `datadoghq.eu`) | `datadoghq.com` | No |
| `DD_SERVICE` | Service name tag | `beluga-ai` | No |

## Usage

### Configure Trace Export

Set up an OTLP trace exporter pointed at the Datadog intake endpoint:

```go
package main

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func setupDatadogTracing(ctx context.Context) (*sdktrace.TracerProvider, error) {
	endpoint := os.Getenv("DD_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://trace-intake.datadoghq.com"
	}

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithHeaders(map[string]string{
			"DD-API-KEY": os.Getenv("DD_API_KEY"),
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("create trace exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("beluga-ai"),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)

	return tp, nil
}
```

### Configure Metrics Export

Set up an OTLP metrics exporter for Datadog:

```go
import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func setupDatadogMetrics(ctx context.Context) (*sdkmetric.MeterProvider, error) {
	endpoint := os.Getenv("DD_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://trace-intake.datadoghq.com"
	}

	exporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(endpoint),
		otlpmetrichttp.WithHeaders(map[string]string{
			"DD-API-KEY": os.Getenv("DD_API_KEY"),
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("create metrics exporter: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
	)

	otel.SetMeterProvider(mp)

	return mp, nil
}
```

### Complete Setup

Wire both exporters into your application:

```go
package main

import (
	"context"
	"log"
)

func main() {
	ctx := context.Background()

	tp, err := setupDatadogTracing(ctx)
	if err != nil {
		log.Fatalf("failed to setup tracing: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Printf("trace provider shutdown error: %v", err)
		}
	}()

	mp, err := setupDatadogMetrics(ctx)
	if err != nil {
		log.Fatalf("failed to setup metrics: %v", err)
	}
	defer func() {
		if err := mp.Shutdown(ctx); err != nil {
			log.Printf("meter provider shutdown error: %v", err)
		}
	}()

	// Beluga AI components now emit traces and metrics to Datadog
	// via the global OTel providers.
	log.Println("Datadog telemetry export configured")
}
```

### Dashboard Template

Import this JSON into Datadog to create an LLM Operations dashboard with key widgets:

```json
{
  "title": "Beluga AI - LLM Operations",
  "widgets": [
    {
      "definition": {
        "type": "timeseries",
        "requests": [
          {
            "q": "sum:gen_ai.client.operation.duration.count{*}.as_count()",
            "display_type": "line"
          }
        ],
        "title": "LLM Calls per Second"
      }
    },
    {
      "definition": {
        "type": "timeseries",
        "requests": [
          {
            "q": "avg:gen_ai.client.operation.duration{*}",
            "display_type": "line"
          }
        ],
        "title": "Average LLM Latency"
      }
    },
    {
      "definition": {
        "type": "timeseries",
        "requests": [
          {
            "q": "sum:gen_ai.client.token.usage{token.type:input}{*}.as_count()",
            "display_type": "bars"
          }
        ],
        "title": "Input Token Usage"
      }
    },
    {
      "definition": {
        "type": "timeseries",
        "requests": [
          {
            "q": "sum:gen_ai.client.token.usage{token.type:output}{*}.as_count()",
            "display_type": "bars"
          }
        ],
        "title": "Output Token Usage"
      }
    }
  ]
}
```

Beluga AI uses the OpenTelemetry `gen_ai.*` semantic conventions for all LLM telemetry. Adjust the metric names in the dashboard queries to match any custom metrics your application emits.

## Advanced Topics

### Trace Sampling

To reduce data volume in high-throughput applications, configure a sampling strategy on the tracer provider:

```go
tp := sdktrace.NewTracerProvider(
	sdktrace.WithBatcher(exporter),
	sdktrace.WithResource(res),
	sdktrace.WithSampler(sdktrace.TraceIDRatioBased(0.1)), // Sample 10% of traces
)
```

### Custom Metrics

Add application-specific metrics alongside the built-in Beluga AI telemetry:

```go
import "go.opentelemetry.io/otel"

meter := otel.Meter("beluga.custom")
counter, err := meter.Int64Counter("beluga.custom.requests_total")
if err != nil {
	log.Fatalf("failed to create counter: %v", err)
}
counter.Add(ctx, 1)
```

### Alerting

Configure Datadog monitors on key metrics:

- **LLM error rate**: Alert when `gen_ai.client.operation.duration` spans with error status exceed a threshold
- **Latency P95**: Alert on high tail latency for LLM calls
- **Token budget**: Alert when cumulative token usage approaches cost limits

### Tagging Strategy

Use consistent tags across all telemetry to enable effective filtering:

- `service`: Application name
- `gen_ai.system`: LLM provider (e.g., `openai`, `anthropic`)
- `gen_ai.request.model`: Model identifier
- `environment`: Deployment environment (`staging`, `production`)

## Troubleshooting

### "Authentication failed"

Verify that `DD_API_KEY` contains a valid Datadog API key. API keys can be managed in **Organization Settings > API Keys** in the Datadog Console.

### Metrics not appearing

Confirm that the OTLP endpoint is correct for your Datadog site. For EU customers, use `https://trace-intake.datadoghq.eu`. Add debug logging to the exporter to verify data is being sent.

## Related Resources

- [LangSmith Debugging](/docs/integrations/langsmith-debugging) -- LLM call debugging and tracing
- [Observability and Tracing](/docs/guides/observability-tracing) -- Beluga AI observability setup
