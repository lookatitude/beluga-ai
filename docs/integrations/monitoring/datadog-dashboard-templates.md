# Datadog Dashboard Templates

Welcome, colleague! In this integration guide, we're going to export Beluga AI's OpenTelemetry metrics and traces to Datadog and create pre-built dashboard templates for monitoring your AI applications.

## What you will build

You will configure OpenTelemetry to export metrics and traces to Datadog, and set up dashboard templates to visualize Beluga AI operations, LLM calls, agent performance, and system health.

## Learning Objectives

- ✅ Export OpenTelemetry data to Datadog
- ✅ Create Datadog dashboards for Beluga AI metrics
- ✅ Monitor LLM calls and agent performance
- ✅ Set up alerts for critical metrics

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- Datadog account and API key
- Datadog Agent installed (optional, for local testing)

## Step 1: Setup and Installation

Install Datadog OpenTelemetry exporter:
bash
```bash
go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp
go get go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp
```

## Step 2: Configure Datadog Exporter

Create a Datadog exporter configuration:
```go
package main

import (
    "context"
    "fmt"
    "os"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
    "go.opentelemetry.io/otel/sdk/resource"
    "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func setupDatadogTracing() (*trace.TracerProvider, error) {
    // Datadog OTLP endpoint
    endpoint := os.Getenv("DD_OTLP_ENDPOINT")
    if endpoint == "" {
        endpoint = "https://trace-intake.datadoghq.com"
    }
    
    exporter, err := otlptracehttp.New(context.Background(),
        otlptracehttp.WithEndpoint(endpoint),
        otlptracehttp.WithHeaders(map[string]string{
            "DD-API-KEY": os.Getenv("DD_API_KEY"),
        }),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create exporter: %w", err)
    }
    
    // Create resource
    res, err := resource.New(context.Background(),
        resource.WithAttributes(
            semconv.ServiceName("beluga-ai"),
            semconv.ServiceVersion("1.0.0"),
        ),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create resource: %w", err)
    }
    
    // Create tracer provider
    tp := trace.NewTracerProvider(
        trace.WithBatcher(exporter),
        trace.WithResource(res),
    )

    

    otel.SetTracerProvider(tp)
    
    return tp, nil
}
```

## Step 3: Export Metrics to Datadog

Configure metrics export:
```go
import (
    "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
    "go.opentelemetry.io/otel/sdk/metric"
)
go
func setupDatadogMetrics() (*metric.MeterProvider, error) {
    endpoint := os.Getenv("DD_OTLP_ENDPOINT")
    if endpoint == "" {
        endpoint = "https://trace-intake.datadoghq.com"
    }
    
    exporter, err := otlpmetrichttp.New(context.Background(),
        otlpmetrichttp.WithEndpoint(endpoint),
        otlpmetrichttp.WithHeaders(map[string]string{
            "DD-API-KEY": os.Getenv("DD_API_KEY"),
        }),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create metrics exporter: %w", err)
    }
    
    mp := metric.NewMeterProvider(
        metric.WithReader(metric.NewPeriodicReader(exporter)),
    )

    

    otel.SetMeterProvider(mp)
    
    return mp, nil
}
```

## Step 4: Dashboard Configuration

Create dashboard JSON for Datadog:

```json
{
  "title": "Beluga AI - LLM Operations",
  "widgets": [
    {
      "definition": {
        "type": "timeseries",
        "requests": [
          {
            "q": "sum:beluga.llm.calls_total{*}.as_count()",
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
            "q": "avg:beluga.llm.latency_seconds{*}",
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
            "q": "sum:beluga.llm.errors_total{*}.as_count()",
            "display_type": "bars"
          }
        ],
        "title": "LLM Errors"
      }
    }
```
  ]
```
}

## Step 5: Complete Integration

Here's a complete example:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/monitoring"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
    "go.opentelemetry.io/otel/sdk/resource"
    "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func main() {
    // Setup Datadog tracing
    tp, err := setupDatadogTracing()
    if err != nil {
        log.Fatalf("Failed to setup tracing: %v", err)
    }
    defer tp.Shutdown(context.Background())
    
    // Setup Datadog metrics
    mp, err := setupDatadogMetrics()
    if err != nil {
        log.Fatalf("Failed to setup metrics: %v", err)
    }
    defer mp.Shutdown(context.Background())
    
    // Create Beluga AI monitor with Datadog
    monitor, err := monitoring.NewMonitor(
        monitoring.WithServiceName("beluga-ai"),
        monitoring.WithOpenTelemetry(""),
    )
    if err != nil {
        log.Fatalf("Failed to create monitor: %v", err)
    }
    
    // Use monitor - metrics will be exported to Datadog
    ctx := context.Background()
    monitor.Logger().Info(ctx, "Application started")
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `DD_API_KEY` | Datadog API key | - | Yes |
| `DD_OTLP_ENDPOINT` | Datadog OTLP endpoint | `https://trace-intake.datadoghq.com` | No |
| `DD_SITE` | Datadog site | `datadoghq.com` | No |
| `DD_SERVICE` | Service name | `beluga-ai` | No |

## Common Issues

### "Authentication failed"

**Problem**: Invalid API key.

**Solution**: Verify API key:export DD_API_KEY="your-api-key"
```

### "Metrics not appearing"

**Problem**: Metrics not being exported.

**Solution**: Check endpoint and verify exporter is running:// Add logging to verify export
exporter, err := otlpmetrichttp.New(ctx, 
    otlpmetrichttp.WithEndpoint(endpoint),
    otlpmetrichttp.WithInsecure(), // For testing
)
```

## Production Considerations

When using Datadog in production:

- **Use batching**: Reduce API calls with batched exports
- **Set sampling**: Sample traces to reduce volume
- **Monitor costs**: Track data ingestion volume
- **Set up alerts**: Configure alerts for critical metrics
- **Use tags**: Add tags for better filtering

## Next Steps

Congratulations! You've integrated Datadog with Beluga AI. Next, learn how to:

- **[LangSmith Debugging Integration](./langsmith-debugging-integration.md)** - Debug LLM calls
- **[Monitoring Package Documentation](../../api/packages/monitoring.md)** - Deep dive into monitoring
- **[Observability Guide](../../guides/observability-tracing.md)** - Complete observability setup

---

**Ready for more?** Check out the [Integrations Index](./README.md) for more integration guides!
