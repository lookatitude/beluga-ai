// Package o11y provides observability primitives for the Beluga AI framework:
// OpenTelemetry-based tracing and metrics following GenAI semantic conventions,
// structured logging via slog, health checks, and LLM-specific trace exporting.
//
// # Tracing
//
// Tracing is built on OpenTelemetry with GenAI semantic convention attributes
// (gen_ai.* namespace). [StartSpan] creates spans with typed attributes,
// and [InitTracer] configures the global OTel tracer provider:
//
//	shutdown, err := o11y.InitTracer("my-service",
//	    o11y.WithSpanExporter(exporter),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer shutdown()
//
//	ctx, span := o11y.StartSpan(ctx, "llm.generate", o11y.Attrs{
//	    o11y.AttrRequestModel: "gpt-4o",
//	    o11y.AttrSystem:       "openai",
//	})
//	defer span.End()
//
// The [Span] interface wraps OTel spans with a simplified API for setting
// attributes, recording errors, and setting status codes.
//
// # Metrics
//
// Pre-registered GenAI metric instruments track token usage, operation
// duration, and estimated cost following OTel conventions:
//
//	o11y.TokenUsage(ctx, inputTokens, outputTokens)
//	o11y.OperationDuration(ctx, durationMs)
//	o11y.Cost(ctx, estimatedUSD)
//
// [InitMeter] configures the package-level meter with a service name.
// Generic [Counter] and [Histogram] functions allow recording custom metrics.
//
// # Logging
//
// [Logger] wraps slog.Logger with context-aware convenience methods and
// functional options for configuration:
//
//	logger := o11y.NewLogger(
//	    o11y.WithLogLevel("debug"),
//	    o11y.WithJSON(),
//	)
//	logger.Info(ctx, "request completed",
//	    "model", "gpt-4o",
//	    "tokens", 150,
//	)
//
// Loggers propagate through context via [WithLogger] and [FromContext].
//
// # Trace Exporting
//
// The [TraceExporter] interface captures detailed LLM call data for analysis
// backends. [LLMCallData] holds the full details of a single invocation
// including model, provider, tokens, cost, messages, and response.
// [MultiExporter] fans out to multiple backends simultaneously:
//
//	multi := o11y.NewMultiExporter(langfuseExp, phoenixExp)
//	err := multi.ExportLLMCall(ctx, data)
//
// Provider implementations include Langfuse, LangSmith, Opik, and Phoenix
// in the o11y/providers/ subpackages.
//
// # Health Checks
//
// The [HealthChecker] interface provides health probes for components.
// [HealthRegistry] aggregates named checkers and runs them concurrently
// via [HealthRegistry.CheckAll]:
//
//	registry := o11y.NewHealthRegistry()
//	registry.Register("database", dbChecker)
//	registry.Register("cache", cacheChecker)
//	results := registry.CheckAll(ctx)
//
// [HealthCheckerFunc] adapts plain functions to the HealthChecker interface.
//
// # GenAI Attribute Constants
//
// The package exports standard GenAI semantic convention attribute keys:
// [AttrAgentName], [AttrOperationName], [AttrToolName], [AttrRequestModel],
// [AttrResponseModel], [AttrInputTokens], [AttrOutputTokens], and [AttrSystem].
package o11y
