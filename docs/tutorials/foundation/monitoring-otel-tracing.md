# End-to-End Tracing with OpenTelemetry

In this tutorial, you'll learn how to implement comprehensive distributed tracing in your Beluga AI applications using OpenTelemetry (OTEL).

## Learning Objectives

- ✅ Configure OpenTelemetry SDK
- ✅ Instrument your application
- ✅ Propagate context across boundaries
- ✅ Visualize traces in Jaeger

## Prerequisites

- Basic understanding of monitoring (see [Prometheus & Grafana Setup](./monitoring-prometheus-grafana.md))
- Jaeger running locally (via Docker)
- Go 1.24+

## Why Tracing?

Metrics tell you *what* happened (e.g., "error rate increased"). Tracing tells you *why* it happened (e.g., "Agent A called Tool B, which timed out calling API C"). In complex AI workflows, tracing is essential for debugging.

## Step 1: Install Dependencies
bash
```bash
go get go.opentelemetry.io/otel
go get go.opentelemetry.io/otel/sdk
go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp
```

## Step 2: Initialize Tracing Provider

Create a setup function to initialize the OTEL tracer.
```go
package main

import (
    "context"
    "fmt"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

func initTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {
    // Create OTLP exporter (sends traces to Jaeger/Collector)
    exporter, err := otlptrace.New(ctx, otlptracehttp.NewClient(
        otlptracehttp.WithEndpoint("localhost:4318"),
        otlptracehttp.WithInsecure(),
    ))
    if err != nil {
        return nil, fmt.Errorf("failed to create exporter: %w", err)
    }

    // Identify your resource
    res, err := resource.New(ctx,
        resource.WithAttributes(
            semconv.ServiceName("beluga-agent"),
            semconv.ServiceVersion("1.0.0"),
        ),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create resource: %w", err)
    }

    // Create Tracer Provider
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(res),
    )

    // Set as global provider
    otel.SetTracerProvider(tp)

    
    return tp, nil
}
```

## Step 3: Instrument Your Code

Beluga AI components are already instrumented. You just need to start the root span.
```go
func main() {
    ctx := context.Background()
    
    // Initialize Tracer
    tp, err := initTracer(ctx)
    if err != nil {
        panic(err)
    }
    defer tp.Shutdown(ctx)
    
    // Create a Tracer
    tracer := otel.Tracer("main")
    
    // Start Root Span
    ctx, span := tracer.Start(ctx, "run-agent-workflow")
    defer span.End()
    
    // Your Agent Logic Here
    // The context 'ctx' now contains the trace ID
    runAgent(ctx)
}

func runAgent(ctx context.Context) {
    // Beluga AI components will automatically attach child spans
    // to the context provided here.
    agent.Invoke(ctx, input) 
}
```

## Step 4: Adding Custom Spans

If you have custom logic, add your own spans.
```go
func customLogic(ctx context.Context) {
    tracer := otel.Tracer("custom-logic")
    ctx, span := tracer.Start(ctx, "calculate-metrics")
    defer span.End()
    
    // Add attributes
    span.SetAttributes(attribute.String("user.id", "123"))
    
    // Simulate work
    time.Sleep(100 * time.Millisecond)
    
    // Record events
    span.AddEvent("calculation-complete")
}
```

## Step 5: Running Jaeger

Run Jaeger to visualize traces:
```bash
docker run -d --name jaeger \
```
  -e COLLECTOR_ZIPKIN_HOST_PORT=:9411 \
  -p 5775:5775/udp \
  -p 6831:6831/udp \
  -p 6832:6832/udp \
  -p 5778:5778 \
  -p 16686:16686 \
  -p 14268:14268 \
  -p 14250:14250 \
  -p 9411:9411 \
  -p 4317:4317 \
  -p 4318:4318 \
  jaegertracing/all-in-one:1.50
```

Access UI at `http://localhost:16686`.

## Verification

1. Run the Go application.
2. Open Jaeger UI.
3. Select service `beluga-agent`.
4. Find traces and inspect the timeline.

## Next Steps

- **[Prometheus & Grafana Setup](./monitoring-prometheus-grafana.md)** - Metrics visualization
- **[Production Deployment](../../getting-started/07-production-deployment.md)** - Deploying your stack
