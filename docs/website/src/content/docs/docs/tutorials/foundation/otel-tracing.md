---
title: OpenTelemetry Tracing Tutorial
description: "Implement distributed tracing in Go with OpenTelemetry GenAI conventions — custom spans, automatic LLM instrumentation, and Jaeger visualization with Beluga AI."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go, tutorial, OpenTelemetry, tracing, GenAI conventions, Jaeger, distributed tracing"
---

Metrics tell you what happened (error rate increased). Traces tell you why (Agent A called Tool B, which timed out calling API C). In complex AI workflows with multiple LLM calls, tool invocations, and retrieval steps, distributed tracing is essential for debugging and performance analysis. Without traces, diagnosing a slow agent requires guessing which of its many internal operations caused the delay.

Beluga AI uses the OpenTelemetry GenAI semantic conventions (`gen_ai.*` attribute namespace) for LLM observability. These conventions are an emerging standard that ensures consistent attribute naming across providers, making it possible to build dashboards and alerts that work regardless of which LLM provider is in use.

## What You Will Build

An instrumented application with OpenTelemetry tracing, including custom spans, automatic LLM instrumentation, and Jaeger visualization.

## Prerequisites

- Go 1.23+
- Docker (for running Jaeger)

## Step 1: Install Dependencies

```bash
go get go.opentelemetry.io/otel
go get go.opentelemetry.io/otel/sdk
go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp
```

## Step 2: Initialize the Tracer Provider

Set up an OTLP exporter that sends traces to a collector (Jaeger, Grafana Tempo, etc.). The OTLP protocol is used because it is the vendor-neutral OpenTelemetry standard — the same exporter configuration works with Jaeger, Grafana Tempo, Datadog, and other backends without code changes. The `resource` attaches service metadata to every span, enabling trace filtering by service name and version in the visualization UI.

```go
package main

import (
    "context"
    "fmt"
    "log"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func initTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {
    exporter, err := otlptracehttp.New(ctx,
        otlptracehttp.WithEndpoint("localhost:4318"),
        otlptracehttp.WithInsecure(),
    )
    if err != nil {
        return nil, fmt.Errorf("create exporter: %w", err)
    }

    res, err := resource.New(ctx,
        resource.WithAttributes(
            semconv.ServiceName("beluga-agent"),
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

## Step 3: Instrument Your Application

Create a root span and pass the context through your pipeline. The root span represents the entire workflow, and Beluga AI components automatically create child spans when they receive a traced context. This automatic instrumentation means that LLM calls, tool executions, and retrieval operations appear as child spans without explicit instrumentation code — you only need to propagate the context.

```go
func main() {
    ctx := context.Background()

    tp, err := initTracer(ctx)
    if err != nil {
        log.Fatalf("init tracer: %v", err)
    }
    defer func() {
        if err := tp.Shutdown(ctx); err != nil {
            log.Printf("shutdown tracer: %v", err)
        }
    }()

    tracer := otel.Tracer("main")

    // Create root span for the entire workflow
    ctx, span := tracer.Start(ctx, "agent-workflow")
    defer span.End()

    // Pass ctx to Beluga AI components — they attach child spans automatically
    if err := runPipeline(ctx); err != nil {
        span.RecordError(err)
        log.Printf("pipeline error: %v", err)
    }
}
```

## Step 4: Add Custom Spans

Instrument your own logic with custom spans and attributes. Each span represents a unit of work in the trace timeline. Adding attributes (like `document.id` and `document.type`) enables filtering and grouping in the trace viewer — for example, finding all traces that processed PDF documents or identifying which document ID caused a failure.

```go
func processDocument(ctx context.Context, docID string) error {
    tracer := otel.Tracer("document-processor")
    ctx, span := tracer.Start(ctx, "process-document")
    defer span.End()

    // Add attributes for filtering and analysis
    span.SetAttributes(
        attribute.String("document.id", docID),
        attribute.String("document.type", "pdf"),
    )

    // Step 1: Parse
    ctx, parseSpan := tracer.Start(ctx, "parse-document")
    // ... parsing logic ...
    parseSpan.End()

    // Step 2: Embed
    ctx, embedSpan := tracer.Start(ctx, "embed-document")
    // ... embedding logic ...
    embedSpan.AddEvent("embedding-complete", trace.WithAttributes(
        attribute.Int("dimensions", 1536),
    ))
    embedSpan.End()

    return nil
}
```

## Step 5: GenAI Semantic Conventions

Beluga AI's `o11y` package uses the OpenTelemetry GenAI conventions. These standardized attribute names ensure that dashboards, alerts, and analysis queries work across all LLM providers without provider-specific logic.

| Attribute | Description |
|:---|:---|
| `gen_ai.system` | Provider name (openai, anthropic, etc.) |
| `gen_ai.request.model` | Model ID (gpt-4o, claude-3-opus, etc.) |
| `gen_ai.request.temperature` | Sampling temperature |
| `gen_ai.request.max_tokens` | Max token limit |
| `gen_ai.usage.input_tokens` | Prompt token count |
| `gen_ai.usage.output_tokens` | Completion token count |
| `gen_ai.response.finish_reason` | Stop reason (stop, length, tool_calls) |

## Step 6: Run Jaeger

Start Jaeger to collect and visualize traces. Jaeger's all-in-one image includes the collector, storage, and UI in a single container, making it suitable for local development.

```bash
docker run -d --name jaeger \
  -p 16686:16686 \
  -p 4317:4317 \
  -p 4318:4318 \
  jaegertracing/all-in-one:latest
```

Access the Jaeger UI at `http://localhost:16686`. Select the `beluga-agent` service to view traces.

## Verification

1. Run your instrumented application.
2. Open the Jaeger UI at `http://localhost:16686`.
3. Select the `beluga-agent` service.
4. Find traces and inspect the span timeline — verify parent-child relationships across LLM calls, tool executions, and retrieval steps.

## Next Steps

- [Prometheus and Grafana](/docs/tutorials/foundation/prometheus-grafana) — Metrics visualization
- [Health Checks](/docs/tutorials/foundation/health-checks) — Component health monitoring
