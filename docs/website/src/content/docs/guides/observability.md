---
title: Monitoring & Observability
description: Instrument AI applications with OpenTelemetry tracing, metrics, structured logging, and health checks.
---

The `o11y` package provides production-grade observability using OpenTelemetry GenAI semantic conventions. Trace every LLM call, track token usage, monitor latency, and export telemetry to any OpenTelemetry-compatible backend.

## Setup

Initialize tracing and metrics at application startup:

```go
import (
	"github.com/lookatitude/beluga-ai/o11y"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
)

func main() {
	ctx := context.Background()

	// Set up OTLP exporter
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint("localhost:4317"),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize tracer
	shutdown, err := o11y.InitTracer("my-ai-service",
		o11y.WithSpanExporter(exporter),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer shutdown()

	// Initialize metrics
	if err := o11y.InitMeter("my-ai-service"); err != nil {
		log.Fatal(err)
	}

	// Your application code...
}
```

## Distributed Tracing

### Creating Spans

Use `o11y.StartSpan` to trace GenAI operations:

```go
ctx, span := o11y.StartSpan(ctx, "llm.generate", o11y.Attrs{
	o11y.AttrOperationName: "chat",
	o11y.AttrRequestModel:  "gpt-4o",
	o11y.AttrSystem:         "openai",
	o11y.AttrAgentName:      "researcher",
})
defer span.End()

resp, err := model.Generate(ctx, msgs)
if err != nil {
	span.RecordError(err)
	span.SetStatus(o11y.StatusError, err.Error())
	return nil, err
}

span.SetAttributes(o11y.Attrs{
	o11y.AttrResponseModel: resp.Model,
	o11y.AttrInputTokens:   resp.Usage.InputTokens,
	o11y.AttrOutputTokens:  resp.Usage.OutputTokens,
})
span.SetStatus(o11y.StatusOK, "")
```

### GenAI Attribute Constants

| Constant | Key | Description |
|----------|-----|-------------|
| `AttrAgentName` | `gen_ai.agent.name` | Agent performing the operation |
| `AttrOperationName` | `gen_ai.operation.name` | Operation type: `chat`, `embed`, etc. |
| `AttrToolName` | `gen_ai.tool.name` | Tool being invoked |
| `AttrRequestModel` | `gen_ai.request.model` | Requested model ID |
| `AttrResponseModel` | `gen_ai.response.model` | Actual model that served the request |
| `AttrInputTokens` | `gen_ai.usage.input_tokens` | Input token count |
| `AttrOutputTokens` | `gen_ai.usage.output_tokens` | Output token count |
| `AttrSystem` | `gen_ai.system` | Provider system (openai, anthropic, etc.) |

### Automatic Tracing with LLM Hooks

Use LLM hooks to trace every call automatically:

```go
tracingHooks := llm.Hooks{
	BeforeGenerate: func(ctx context.Context, msgs []schema.Message) error {
		ctx, span := o11y.StartSpan(ctx, "llm.generate", o11y.Attrs{
			o11y.AttrOperationName: "chat",
			"gen_ai.messages":      len(msgs),
		})
		// Store span for AfterGenerate
		return nil
	},
	AfterGenerate: func(ctx context.Context, resp *schema.AIMessage, err error) {
		if resp != nil {
			o11y.TokenUsage(ctx, resp.Usage.InputTokens, resp.Usage.OutputTokens)
		}
	},
}

model = llm.ApplyMiddleware(model, llm.WithHooks(tracingHooks))
```

## Metrics

### Built-in Metrics

The `o11y` package provides pre-defined GenAI metrics:

```go
// Record token usage
o11y.TokenUsage(ctx, inputTokens, outputTokens)

// Record operation latency in milliseconds
o11y.OperationDuration(ctx, 245.5)

// Record estimated cost in USD
o11y.Cost(ctx, 0.0032)

// Generic counter
o11y.Counter(ctx, "beluga.tool.invocations", 1)

// Generic histogram
o11y.Histogram(ctx, "beluga.retriever.latency_ms", 125.0)
```

### Pre-defined Metric Instruments

| Metric | Type | Unit | Description |
|--------|------|------|-------------|
| `gen_ai.client.token.usage` | Counter | `{token}` | Input tokens consumed |
| `gen_ai.client.token.usage.output` | Counter | `{token}` | Output tokens produced |
| `gen_ai.client.operation.duration` | Histogram | `ms` | Operation latency |
| `gen_ai.client.estimated_cost` | Counter | `USD` | Estimated cost |

### Prometheus Integration

Export metrics to Prometheus:

```go
import (
	"net/http"
	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
)

// Set up Prometheus exporter
exporter, err := promexporter.New()
if err != nil {
	log.Fatal(err)
}

provider := metric.NewMeterProvider(metric.WithReader(exporter))
otel.SetMeterProvider(provider)

// Initialize Beluga metrics
o11y.InitMeter("my-ai-service")

// Expose metrics endpoint
http.Handle("/metrics", promhttp.Handler())
http.ListenAndServe(":9090", nil)
```

## Structured Logging

Use Go's `slog` for structured logging throughout the framework:

```go
import "github.com/lookatitude/beluga-ai/o11y"

logger := o11y.NewLogger(o11y.LoggerConfig{
	Level:  slog.LevelInfo,
	Format: "json", // "json" or "text"
})

// The logger integrates with LLM middleware
model = llm.ApplyMiddleware(model, llm.WithLogging(logger))
```

### Log Levels for AI Operations

| Level | Usage |
|-------|-------|
| `DEBUG` | Raw LLM requests/responses, tool input/output |
| `INFO` | Operation start/complete, token usage summaries |
| `WARN` | Rate limiting, retries, fallback activations |
| `ERROR` | Failed operations, guard blocks, timeout errors |

### Correlating Logs with Traces

```go
// Logger automatically includes trace context when spans are active
logger.InfoContext(ctx, "llm.generate.complete",
	"model", "gpt-4o",
	"input_tokens", 150,
	"output_tokens", 300,
	"latency_ms", 245,
)
// Output includes trace_id and span_id for correlation
```

## Health Checks

Monitor application health with built-in health check endpoints:

```go
import "github.com/lookatitude/beluga-ai/o11y"

health := o11y.NewHealthChecker()

// Register checks
health.Register("llm", func(ctx context.Context) error {
	_, err := model.Generate(ctx, []schema.Message{
		schema.NewHumanMessage("ping"),
	}, llm.WithMaxTokens(1))
	return err
})

health.Register("vectorstore", func(ctx context.Context) error {
	_, err := store.Search(ctx, []float32{0.1}, 1)
	return err
})

health.Register("redis", func(ctx context.Context) error {
	return redisClient.Ping(ctx).Err()
})

// Expose HTTP endpoints
http.HandleFunc("/healthz", health.LivenessHandler())   // Basic alive check
http.HandleFunc("/readyz", health.ReadinessHandler())    // Full dependency check
```

### Health Check Response Format

```json
{
  "status": "healthy",
  "checks": {
    "llm": {"status": "healthy", "latency_ms": 245},
    "vectorstore": {"status": "healthy", "latency_ms": 12},
    "redis": {"status": "healthy", "latency_ms": 1}
  }
}
```

## LLM-Specific Exporters

Export traces to LLM observability platforms:

```go
import (
	_ "github.com/lookatitude/beluga-ai/o11y/providers/langfuse"
	_ "github.com/lookatitude/beluga-ai/o11y/providers/langsmith"
)
```

### Available Exporters

| Exporter | Import Path | Platform |
|----------|-------------|----------|
| Langfuse | `o11y/providers/langfuse` | Langfuse |
| LangSmith | `o11y/providers/langsmith` | LangSmith |
| Phoenix | `o11y/providers/phoenix` | Arize Phoenix |
| Opik | `o11y/providers/opik` | Comet Opik |

## Grafana Dashboard

Set up a Grafana dashboard for monitoring:

### Key Panels

| Panel | Query | Description |
|-------|-------|-------------|
| Token Usage | `sum(rate(gen_ai_client_token_usage_total[5m]))` | Token consumption rate |
| Operation Latency | `histogram_quantile(0.95, gen_ai_client_operation_duration_bucket)` | P95 latency |
| Error Rate | `sum(rate(gen_ai_errors_total[5m]))` | Error frequency |
| Cost | `sum(increase(gen_ai_client_estimated_cost_total[1h]))` | Hourly cost |

### Alerting Rules

```yaml
groups:
  - name: beluga-alerts
    rules:
      - alert: HighLLMLatency
        expr: histogram_quantile(0.95, gen_ai_client_operation_duration_bucket) > 5000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "LLM P95 latency exceeds 5 seconds"

      - alert: HighTokenUsage
        expr: sum(rate(gen_ai_client_token_usage_total[1h])) > 1000000
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Token usage exceeds 1M/hour"
```

## Next Steps

- [Working with LLMs](/guides/working-with-llms/) — LLM middleware and hooks
- [Safety & Guards](/guides/safety-and-guards/) — Monitor guard decisions
- [Deploying to Production](/guides/deployment/) — Production observability setup
- [RAG Pipeline](/guides/rag-pipeline/) — Trace retrieval performance
