---
title: "Observability Providers"
description: "4 observability providers for LLM tracing: Langfuse, LangSmith, Opik, Phoenix. OpenTelemetry-based tracing and metrics in Go with Beluga AI."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "LLM observability, AI tracing, Langfuse, LangSmith, OpenTelemetry, Go observability, Beluga AI"
---

Beluga AI v2 provides a unified observability layer built on OpenTelemetry with support for exporting LLM trace data to external platforms. The `o11y` package handles tracing, metrics, structured logging, and health checks. Observability providers implement the `TraceExporter` interface to send LLM call data to platforms such as Langfuse, LangSmith, Opik, and Phoenix.

## TraceExporter Interface

All observability providers implement the `TraceExporter` interface:

```go
type TraceExporter interface {
    ExportLLMCall(ctx context.Context, data LLMCallData) error
}
```

## LLMCallData

The `LLMCallData` struct captures all details of an LLM invocation:

```go
type LLMCallData struct {
    Model       string
    Provider    string
    InputTokens int
    OutputTokens int
    Duration    time.Duration
    Cost        float64
    Messages    []schema.Message
    Response    string
    Error       string
    Metadata    map[string]any
}
```

## Built-in Observability

### Tracing (OpenTelemetry)

The `o11y` package provides native OpenTelemetry tracing with GenAI semantic conventions:

```go
import "github.com/lookatitude/beluga-ai/o11y"

// Initialize tracer with default OTLP exporter
shutdown, err := o11y.InitTracer("my-service")
if err != nil {
    log.Fatal(err)
}
defer shutdown()

// Create spans for AI operations
ctx, span := o11y.StartSpan(ctx, "llm.generate", o11y.Attrs{
    o11y.AttrRequestModel: "gpt-4o",
    o11y.AttrSystem:       "openai",
})
defer span.End()

// Record errors
if err != nil {
    span.RecordError(err)
    span.SetStatus(o11y.StatusError, err.Error())
}
```

### GenAI Semantic Conventions

The tracing layer uses standardized attribute names from the OpenTelemetry GenAI conventions:

| Attribute | Constant | Description |
|---|---|---|
| `gen_ai.agent.name` | `AttrAgentName` | Agent identifier |
| `gen_ai.operation.name` | `AttrOperationName` | Operation type |
| `gen_ai.tool.name` | `AttrToolName` | Tool identifier |
| `gen_ai.request.model` | `AttrRequestModel` | Requested model |
| `gen_ai.response.model` | `AttrResponseModel` | Actual model used |
| `gen_ai.usage.input_tokens` | `AttrInputTokens` | Input token count |
| `gen_ai.usage.output_tokens` | `AttrOutputTokens` | Output token count |
| `gen_ai.system` | `AttrSystem` | Provider system name |

### Metrics

Record token usage, latency, and cost metrics through the `o11y` package:

```go
// Initialize meter
err := o11y.InitMeter("my-service")
if err != nil {
    log.Fatal(err)
}

// Record token usage
o11y.TokenUsage(ctx, 500, 150)

// Record operation duration (milliseconds)
o11y.OperationDuration(ctx, 1250.0)

// Record estimated cost (USD)
o11y.Cost(ctx, 0.003)

// Custom counters and histograms
o11y.Counter(ctx, "tool.calls", 1)
o11y.Histogram(ctx, "retrieval.latency_ms", 45.2)
```

### Structured Logging

The `Logger` type wraps `slog` with context-aware methods:

```go
logger := o11y.NewLogger(
    o11y.WithLogLevel("debug"),
    o11y.WithJSON(),
)

// Attach to context for propagation
ctx = o11y.WithLogger(ctx, logger)

// Use from context
log := o11y.FromContext(ctx)
log.Info(ctx, "generation complete",
    "model", "gpt-4o",
    "tokens", 150,
)
```

### Health Checks

Register health checkers for infrastructure components:

```go
registry := o11y.NewHealthRegistry()
registry.Register("database", o11y.HealthCheckerFunc(func(ctx context.Context) o11y.HealthResult {
    return o11y.HealthResult{
        Status:    o11y.Healthy,
        Component: "database",
        Message:   "connection pool active",
    }
}))

results := registry.CheckAll(ctx)
for _, r := range results {
    fmt.Printf("%s: %s (%s)\n", r.Component, r.Status, r.Message)
}
```

## MultiExporter

Fan out trace data to multiple platforms simultaneously:

```go
import (
    "github.com/lookatitude/beluga-ai/o11y"
    "github.com/lookatitude/beluga-ai/o11y/providers/langfuse"
    "github.com/lookatitude/beluga-ai/o11y/providers/phoenix"
)

lfExporter, err := langfuse.New(
    langfuse.WithPublicKey(os.Getenv("LANGFUSE_PUBLIC_KEY")),
    langfuse.WithSecretKey(os.Getenv("LANGFUSE_SECRET_KEY")),
)
if err != nil {
    log.Fatal(err)
}

pxExporter, err := phoenix.New(
    phoenix.WithBaseURL("http://localhost:6006"),
)
if err != nil {
    log.Fatal(err)
}

multi := o11y.NewMultiExporter(lfExporter, pxExporter)

err = multi.ExportLLMCall(ctx, o11y.LLMCallData{
    Model:        "gpt-4o",
    Provider:     "openai",
    InputTokens:  500,
    OutputTokens: 150,
    Duration:     1200 * time.Millisecond,
    Response:     "The capital of France is Paris.",
})
```

## Tracer Options

| Option | Description |
|---|---|
| `WithSpanExporter(exp)` | Use a custom OTel span exporter |
| `WithSampler(s)` | Use a custom OTel sampler |
| `WithSyncExport()` | Synchronous export (useful for testing) |

## Available Providers

| Provider | Description |
|---|---|
| [Langfuse](/providers/observability/langfuse) | Open-source LLM observability with trace and generation tracking |
| [LangSmith](/providers/observability/langsmith) | LangChain's observability platform for tracing LLM runs |
| [Opik](/providers/observability/opik) | Comet's LLM observability platform with workspace management |
| [Phoenix](/providers/observability/phoenix) | Arize's open-source LLM observability with OTel-native spans |
