# Observability and Distributed Tracing Guide

> **Learn how to implement comprehensive observability in your Beluga AI applications using OpenTelemetry for tracing, metrics, and structured logging.**

## Introduction

When a user request flows through your AI application — from the API endpoint, through an LLM call, into memory retrieval, and back — understanding what happened becomes challenging. Did the LLM take too long? Was there a cache miss? Why did the agent make three tool calls instead of one?

**Distributed tracing** gives you a complete picture. You can see exactly where time was spent, where errors occurred, and how different components interacted. Combined with **metrics** and **structured logging**, you have full observability into your AI system.

In this guide, you'll learn:

- How to set up OpenTelemetry (OTEL) in your Beluga AI application
- How to create and propagate trace spans across components
- How to record meaningful metrics for AI operations
- How to integrate structured logging with trace context
- How to view and analyze traces in your observability backend

## Prerequisites

| Requirement | Why You Need It |
|-------------|-----------------|
| **Go 1.24+** | Required for Beluga AI framework |
| **OpenTelemetry Collector** | Receives and exports telemetry data |
| **Trace Backend** | Jaeger, Zipkin, or cloud provider (optional for development) |
| **Basic OTEL concepts** | Understanding of spans, traces, and attributes |

### Quick Setup

```bash
# Install OpenTelemetry Collector (optional, for local development)
docker run -d --name otel-collector \
  -p 4317:4317 -p 4318:4318 \
  otel/opentelemetry-collector:latest

# Or use Jaeger for local trace visualization
docker run -d --name jaeger \
  -p 16686:16686 -p 4317:4317 \
  jaegertracing/all-in-one:latest
```

## Concepts

Before we dive into code, let's understand the key concepts.

### What is a Trace?

A **trace** represents the entire journey of a request through your system. It's composed of **spans** — each span represents a single operation.

```
Trace: user-request-123
├── Span: API.HandleRequest (50ms)
│   ├── Span: Agent.Process (45ms)
│   │   ├── Span: LLM.Generate (30ms)
│   │   │   └── Span: OpenAI.APICall (28ms)
│   │   ├── Span: Memory.Retrieve (8ms)
│   │   └── Span: Tool.Execute (5ms)
│   └── Span: Response.Format (2ms)
```

### Span Anatomy

Each span contains:

- **Name**: Operation identifier (e.g., "llm.generate")
- **Start/End Time**: Duration of the operation
- **Attributes**: Key-value metadata (model name, token count, etc.)
- **Events**: Timestamped logs within the span
- **Status**: Ok, Error, or Unset
- **Parent**: Link to parent span (creates the hierarchy)

### Context Propagation

The **context** carries trace information across function calls and even service boundaries. Always pass `ctx` through your call chain.

```go
// Context flows through all operations
func HandleRequest(ctx context.Context, req *Request) (*Response, error) {
    // ctx contains trace info
    return processWithAgent(ctx, req)
}

func processWithAgent(ctx context.Context, req *Request) (*Response, error) {
    // ctx still contains trace info, child spans link correctly
    return agent.Process(ctx, req.Query)
}
```

## Step-by-Step Tutorial

### Step 1: Initialize OpenTelemetry

First, set up OTEL in your application's main function:

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/lookatitude/beluga-ai/pkg/monitoring"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func initTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {
    // Create OTLP exporter
    exporter, err := otlptracegrpc.New(ctx,
        otlptracegrpc.WithEndpoint("localhost:4317"),
        otlptracegrpc.WithInsecure(), // Use TLS in production
    )
    if err != nil {
        return nil, err
    }

    // Create resource with service info
    res, err := resource.New(ctx,
        resource.WithAttributes(
            semconv.ServiceName("my-ai-service"),
            semconv.ServiceVersion("1.0.0"),
            semconv.DeploymentEnvironment("development"),
        ),
    )
    if err != nil {
        return nil, err
    }

    // Create TracerProvider
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(res),
        sdktrace.WithSampler(sdktrace.AlwaysSample()), // Sample all in dev
    )

    // Set as global provider
    otel.SetTracerProvider(tp)

    return tp, nil
}

func main() {
    ctx := context.Background()

    // Initialize tracing
    tp, err := initTracer(ctx)
    if err != nil {
        log.Fatal(err)
    }
    defer func() {
        if err := tp.Shutdown(ctx); err != nil {
            log.Printf("Error shutting down tracer: %v", err)
        }
    }()

    // Your application code...
}
```

**What you'll see**: Traces being exported to your OTEL collector.

**Why this works**: Setting the global TracerProvider means all Beluga AI packages automatically use it.

### Step 2: Create Spans in Your Code

Now let's add tracing to your application code:

```go
import (
    "context"
    "time"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/trace"
)

func ProcessUserQuery(ctx context.Context, query string) (*Response, error) {
    // Get tracer for this package
    tracer := otel.Tracer("myapp/handlers")

    // Start a new span
    ctx, span := tracer.Start(ctx, "ProcessUserQuery",
        trace.WithAttributes(
            attribute.String("query.text", query),
            attribute.Int("query.length", len(query)),
        ),
    )
    defer span.End() // Always end spans!

    // Record events for important milestones
    span.AddEvent("starting_processing")

    // Call LLM (this creates a child span automatically)
    start := time.Now()
    response, err := llm.Generate(ctx, query)
    
    if err != nil {
        // Record error on span
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return nil, err
    }

    // Add result attributes
    span.SetAttributes(
        attribute.Int("response.tokens", response.TokenCount),
        attribute.Float64("processing.duration_ms", float64(time.Since(start).Milliseconds())),
    )

    span.AddEvent("processing_complete")
    span.SetStatus(codes.Ok, "")

    return response, nil
}
```

### Step 3: Propagate Context Across Goroutines

When using goroutines, you must explicitly pass context:

```go
func ProcessBatch(ctx context.Context, queries []string) ([]*Response, error) {
    tracer := otel.Tracer("myapp/batch")
    ctx, span := tracer.Start(ctx, "ProcessBatch",
        trace.WithAttributes(
            attribute.Int("batch.size", len(queries)),
        ),
    )
    defer span.End()

    results := make([]*Response, len(queries))
    var wg sync.WaitGroup
    errChan := make(chan error, len(queries))

    for i, query := range queries {
        wg.Add(1)
        
        // Create a child span for each item
        go func(idx int, q string) {
            defer wg.Done()
            
            // Create child span - note we use the parent ctx
            childCtx, childSpan := tracer.Start(ctx, "ProcessBatchItem",
                trace.WithAttributes(
                    attribute.Int("batch.index", idx),
                ),
            )
            defer childSpan.End()

            result, err := ProcessUserQuery(childCtx, q)
            if err != nil {
                childSpan.RecordError(err)
                errChan <- err
                return
            }
            results[idx] = result
        }(i, query)
    }

    wg.Wait()
    close(errChan)

    // Check for errors
    for err := range errChan {
        if err != nil {
            span.RecordError(err)
            return results, err
        }
    }

    return results, nil
}
```

### Step 4: Use Beluga's Built-in Tracing

Beluga AI packages have built-in tracing. Here's how to ensure everything connects:

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/monitoring"
)

func main() {
    ctx := context.Background()

    // Initialize monitoring (includes tracing)
    monitor, err := monitoring.NewMonitor(
        monitoring.WithServiceName("my-ai-service"),
        monitoring.WithOpenTelemetry("localhost:4317"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer monitor.Stop(ctx)

    // Initialize LLM metrics
    meter := otel.Meter("my-ai-service")
    llms.InitMetrics(meter)

    // Create LLM provider - it automatically creates child spans
    config := llms.NewConfig(
        llms.WithProvider("openai"),
        llms.WithModelName("gpt-4"),
        llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    )

    provider, _ := llms.NewProvider(ctx, "openai", config)

    // When you call Generate, it creates spans under your parent span
    tracer := otel.Tracer("myapp")
    ctx, span := tracer.Start(ctx, "user-request")
    defer span.End()

    // This creates child span: llm.Generate
    response, err := provider.Generate(ctx, messages)
}
```

### Step 5: Add Custom Metrics

Metrics complement traces by providing aggregated data:

```go
import (
    "go.opentelemetry.io/otel/metric"
)

// Define metrics once at package level
var (
    requestCounter  metric.Int64Counter
    latencyHist     metric.Float64Histogram
    activeRequests  metric.Int64UpDownCounter
)

func initMetrics(meter metric.Meter) error {
    var err error

    requestCounter, err = meter.Int64Counter(
        "myapp.requests.total",
        metric.WithDescription("Total number of requests"),
        metric.WithUnit("1"),
    )
    if err != nil {
        return err
    }

    latencyHist, err = meter.Float64Histogram(
        "myapp.request.duration",
        metric.WithDescription("Request duration in seconds"),
        metric.WithUnit("s"),
    )
    if err != nil {
        return err
    }

    activeRequests, err = meter.Int64UpDownCounter(
        "myapp.requests.active",
        metric.WithDescription("Number of active requests"),
    )
    if err != nil {
        return err
    }

    return nil
}

// Use metrics in your handlers
func HandleRequest(ctx context.Context, req *Request) (*Response, error) {
    start := time.Now()

    // Record active request
    activeRequests.Add(ctx, 1)
    defer activeRequests.Add(ctx, -1)

    // Process request
    response, err := processRequest(ctx, req)

    // Record metrics
    duration := time.Since(start).Seconds()
    status := "success"
    if err != nil {
        status = "error"
    }

    requestCounter.Add(ctx, 1,
        metric.WithAttributes(
            attribute.String("status", status),
            attribute.String("endpoint", req.Endpoint),
        ),
    )

    latencyHist.Record(ctx, duration,
        metric.WithAttributes(
            attribute.String("endpoint", req.Endpoint),
        ),
    )

    return response, err
}
```

### Step 6: Integrate Structured Logging

Connect logs to traces for full context:

```go
import (
    "log/slog"
    "go.opentelemetry.io/otel/trace"
)

// Create a logger that includes trace context
func logWithTrace(ctx context.Context, level slog.Level, msg string, attrs ...any) {
    // Extract trace context
    spanCtx := trace.SpanContextFromContext(ctx)
    
    if spanCtx.IsValid() {
        // Add trace context to log
        attrs = append(attrs,
            "trace_id", spanCtx.TraceID().String(),
            "span_id", spanCtx.SpanID().String(),
        )
    }

    slog.Log(ctx, level, msg, attrs...)
}

// Usage
func ProcessQuery(ctx context.Context, query string) error {
    logWithTrace(ctx, slog.LevelInfo, "Processing query",
        "query_length", len(query),
    )

    // Now your logs can be correlated with traces!
    
    result, err := doWork(ctx)
    if err != nil {
        logWithTrace(ctx, slog.LevelError, "Query processing failed",
            "error", err.Error(),
        )
        return err
    }

    logWithTrace(ctx, slog.LevelInfo, "Query processed successfully",
        "result_size", len(result),
    )
    
    return nil
}
```

### Step 7: View Traces in Jaeger

Once your application is running and sending traces:

1. Open Jaeger UI: http://localhost:16686
2. Select your service from the dropdown
3. Click "Find Traces"
4. Click on a trace to see the span hierarchy

You'll see something like:

```
my-ai-service: user-request (52ms)
├── my-ai-service: ProcessUserQuery (50ms)
│   ├── llms: llm.Generate (35ms)
│   │   └── openai: openai.CreateChatCompletion (33ms)
│   ├── memory: memory.Retrieve (8ms)
│   └── myapp: formatResponse (2ms)
```

## Code Examples

### Complete Application Example

Here's a complete example tying everything together:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/monitoring"
    "github.com/lookatitude/beluga-ai/pkg/schema"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/metric"
    "go.opentelemetry.io/otel/trace"

    _ "github.com/lookatitude/beluga-ai/pkg/llms/providers/openai"
)

var (
    tracer         trace.Tracer
    requestCounter metric.Int64Counter
    latencyHist    metric.Float64Histogram
)

func main() {
    ctx := context.Background()

    // Initialize monitoring
    monitor, err := monitoring.NewMonitor(
        monitoring.WithServiceName("ai-chat-service"),
        monitoring.WithOpenTelemetry("localhost:4317"),
    )
    if err != nil {
        log.Fatal(err)
    }
    if err := monitor.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer monitor.Stop(ctx)

    // Initialize tracer and metrics
    tracer = otel.Tracer("ai-chat-service")
    meter := otel.Meter("ai-chat-service")
    llms.InitMetrics(meter)
    initAppMetrics(meter)

    // Create LLM provider
    config := llms.NewConfig(
        llms.WithProvider("openai"),
        llms.WithModelName("gpt-4"),
        llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    )
    provider, err := llms.NewProvider(ctx, "openai", config)
    if err != nil {
        log.Fatal(err)
    }

    // Set up HTTP handler
    http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
        handleChat(r.Context(), w, r, provider)
    })

    log.Println("Starting server on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func initAppMetrics(meter metric.Meter) {
    var err error
    requestCounter, err = meter.Int64Counter("chat.requests.total")
    if err != nil {
        log.Fatal(err)
    }
    latencyHist, err = meter.Float64Histogram("chat.request.duration")
    if err != nil {
        log.Fatal(err)
    }
}

func handleChat(ctx context.Context, w http.ResponseWriter, r *http.Request, provider llms.Provider) {
    start := time.Now()

    // Create request span
    ctx, span := tracer.Start(ctx, "handleChat",
        trace.WithAttributes(
            attribute.String("http.method", r.Method),
            attribute.String("http.url", r.URL.Path),
        ),
    )
    defer span.End()

    // Parse request
    query := r.URL.Query().Get("q")
    if query == "" {
        span.SetStatus(codes.Error, "missing query")
        http.Error(w, "Missing query parameter", http.StatusBadRequest)
        return
    }

    span.SetAttributes(attribute.String("query", query))

    // Process with LLM
    messages := []schema.Message{
        schema.NewSystemMessage("You are a helpful assistant."),
        schema.NewHumanMessage(query),
    }

    response, err := provider.Generate(ctx, messages)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        recordMetrics(ctx, "error", time.Since(start))
        http.Error(w, "Failed to generate response", http.StatusInternalServerError)
        return
    }

    // Success
    span.SetStatus(codes.Ok, "")
    span.SetAttributes(attribute.Int("response.length", len(response.GetContent())))
    recordMetrics(ctx, "success", time.Since(start))

    w.Header().Set("Content-Type", "text/plain")
    fmt.Fprint(w, response.GetContent())
}

func recordMetrics(ctx context.Context, status string, duration time.Duration) {
    requestCounter.Add(ctx, 1,
        metric.WithAttributes(attribute.String("status", status)),
    )
    latencyHist.Record(ctx, duration.Seconds(),
        metric.WithAttributes(attribute.String("status", status)),
    )
}
```

## Best Practices

### 1. Meaningful Span Names

```go
// ✅ Good: Descriptive, hierarchical names
"llm.generate"
"memory.retrieve"
"agent.process"
"tool.calculator.execute"

// ❌ Bad: Generic or unclear names
"doStuff"
"process"
"main"
```

### 2. Useful Attributes

```go
// ✅ Good: Attributes that help debugging
span.SetAttributes(
    attribute.String("llm.model", "gpt-4"),
    attribute.Int("llm.input_tokens", 150),
    attribute.Int("llm.output_tokens", 200),
    attribute.Float64("llm.latency_ms", 350.5),
)

// ❌ Bad: Sensitive data or PII
span.SetAttributes(
    attribute.String("user.email", "user@example.com"),  // PII!
    attribute.String("api.key", "sk-...")               // Secret!
)
```

### 3. Sampling in Production

```go
// Development: sample everything
sampler := sdktrace.AlwaysSample()

// Production: sample a percentage
sampler := sdktrace.TraceIDRatioBased(0.1) // 10% of traces

// Production: sample errors always, success sometimes
sampler := sdktrace.ParentBased(
    sdktrace.TraceIDRatioBased(0.1),
    sdktrace.WithRemoteParentSampled(sdktrace.AlwaysSample()),
)
```

### 4. Performance Considerations

```go
// ✅ Good: Check if span is recording before expensive operations
if span.IsRecording() {
    // Expensive attribute computation
    span.SetAttributes(
        attribute.String("expensive.data", computeExpensiveData()),
    )
}

// ✅ Good: Use span events instead of many child spans
span.AddEvent("checkpoint", trace.WithAttributes(
    attribute.String("stage", "validation_complete"),
))
```

## Troubleshooting

### Q: Traces aren't appearing in my backend

**A:** Check these common issues:

1. **OTEL Collector not running**: Verify with `docker ps` or check the endpoint
2. **Wrong endpoint**: Ensure `localhost:4317` is correct for your setup
3. **Firewall blocking**: Check if port 4317 is accessible
4. **Exporter errors**: Check application logs for export failures

```go
// Add debug logging for export errors
exp, err := otlptracegrpc.New(ctx,
    otlptracegrpc.WithEndpoint("localhost:4317"),
    otlptracegrpc.WithInsecure(),
)
if err != nil {
    log.Printf("Failed to create exporter: %v", err)
}
```

### Q: Child spans aren't linked to parents

**A:** Ensure you're passing the context correctly:

```go
// ✅ Correct: Use the context from parent span
ctx, parentSpan := tracer.Start(ctx, "parent")
ctx, childSpan := tracer.Start(ctx, "child")  // Uses ctx from parent

// ❌ Wrong: Using background context breaks the chain
ctx, parentSpan := tracer.Start(ctx, "parent")
_, childSpan := tracer.Start(context.Background(), "child")  // ORPHANED!
```

### Q: High memory usage from tracing

**A:** Tune your configuration:

```go
tp := sdktrace.NewTracerProvider(
    sdktrace.WithBatcher(exporter,
        sdktrace.WithMaxQueueSize(2048),      // Limit queue
        sdktrace.WithMaxExportBatchSize(512), // Smaller batches
        sdktrace.WithBatchTimeout(5 * time.Second),
    ),
    sdktrace.WithSampler(sdktrace.TraceIDRatioBased(0.1)), // Sample less
)
```

### Q: Logs aren't correlated with traces

**A:** Extract and include trace context in logs:

```go
spanCtx := trace.SpanContextFromContext(ctx)
if spanCtx.IsValid() {
    slog.Info("Operation complete",
        "trace_id", spanCtx.TraceID().String(),
        "span_id", spanCtx.SpanID().String(),
    )
}
```

## Related Resources

- **[LLM Provider Integration Guide](./llm-providers.md)**: Built-in tracing in LLM providers
- **[Extensibility Guide](./extensibility.md)**: Adding tracing to custom components
- **[Monitoring Dashboards Use Case](../use-cases/monitoring-dashboards.md)**: Setting up Prometheus/Grafana
- **[Single Binary Deployment](https://github.com/lookatitude/beluga-ai/tree/main/examples/deployment/single_binary/)**: Production deployment with observability
- **[Monitoring Package Documentation](https://github.com/lookatitude/beluga-ai/blob/main/pkg/monitoring/README.md)**: Full monitoring package reference
- **[OpenTelemetry Documentation](https://opentelemetry.io/docs/go/)**: Official OTEL Go docs
