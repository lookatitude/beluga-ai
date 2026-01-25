---
title: "Correlating Request-IDs across Services"
package: "server"
category: "observability"
complexity: "intermediate"
---

# Correlating Request-IDs across Services

## Problem

You need to correlate requests across multiple services in a distributed system, tracking a request's journey from API gateway through multiple microservices for debugging and monitoring.

## Solution

Implement request ID propagation that generates unique request IDs at entry points, propagates them through all service calls (HTTP headers, context, logs), and correlates them in centralized logging/observability systems. This works because you can inject request IDs into context and propagate them through the call chain.

## Code Example
```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/trace"
    "github.com/google/uuid"
)

var tracer = otel.Tracer("beluga.server.request_correlation")

const RequestIDHeader = "X-Request-ID"
const RequestIDKey = "request_id"

// RequestIDPropagator propagates request IDs across services
type RequestIDPropagator struct {
    generator func() string
}

// NewRequestIDPropagator creates a new propagator
func NewRequestIDPropagator() *RequestIDPropagator {
    return &RequestIDPropagator{
        generator: func() string {
            return uuid.New().String()
        },
    }
}

// ExtractRequestID extracts request ID from context or headers
func (rip *RequestIDPropagator) ExtractRequestID(ctx context.Context, headers http.Header) (string, context.Context) {
    ctx, span := tracer.Start(ctx, "request_id.extract")
    defer span.End()
    
    // Check context first
    if reqID, ok := ctx.Value(RequestIDKey).(string); ok && reqID != "" {
        span.SetAttributes(attribute.String("request_id", reqID))
        span.SetAttributes(attribute.String("source", "context"))
        return reqID, ctx
    }
    
    // Check headers
    if reqID := headers.Get(RequestIDHeader); reqID != "" {
        ctx = context.WithValue(ctx, RequestIDKey, reqID)
        span.SetAttributes(attribute.String("request_id", reqID))
        span.SetAttributes(attribute.String("source", "header"))
        return reqID, ctx
    }
    
    // Generate new request ID
    reqID := rip.generator()
    ctx = context.WithValue(ctx, RequestIDKey, reqID)
    
    span.SetAttributes(
        attribute.String("request_id", reqID),
        attribute.String("source", "generated"),
    )
    span.SetStatus(trace.StatusOK, "request ID extracted/generated")
    
    return reqID, ctx
}

// InjectRequestID injects request ID into headers for outgoing requests
func (rip *RequestIDPropagator) InjectRequestID(ctx context.Context, headers http.Header) {
    if reqID := GetRequestID(ctx); reqID != "" {
        headers.Set(RequestIDKey, reqID)
        headers.Set(RequestIDHeader, reqID) // Also set standard header
    }
}

// GetRequestID gets request ID from context
func GetRequestID(ctx context.Context) string {
    if reqID, ok := ctx.Value(RequestIDKey).(string); ok {
        return reqID
    }
    return ""
}

// RequestIDMiddleware creates HTTP middleware for request ID handling
func RequestIDMiddleware(propagator *RequestIDPropagator) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Extract or generate request ID
            reqID, ctx := propagator.ExtractRequestID(r.Context(), r.Header)

            // Add to response headers
            w.Header().Set(RequestIDHeader, reqID)
            
            // Add to context
            r = r.WithContext(ctx)
            
            // Add to OpenTelemetry context
            ctx = trace.ContextWithSpan(ctx, trace.SpanFromContext(ctx))
            
            // Add request ID to span
            span := trace.SpanFromContext(ctx)
            if span.IsRecording() {
                span.SetAttributes(attribute.String("request_id", reqID))
            }
            
            // Continue with request
            next.ServeHTTP(w, r)
        })
    }
}

// LogWithRequestID logs with request ID
func LogWithRequestID(ctx context.Context, level string, message string, fields map[string]interface{}) {
    reqID := GetRequestID(ctx)

    logFields := []interface{}{
        "request_id", reqID,
        "level", level,
        "message", message,
    }
    
    for k, v := range fields {
        logFields = append(logFields, k, v)
    }
    
    log.Printf("%v", logFields)
}

// PropagateToGRPC propagates request ID to gRPC metadata
func PropagateToGRPC(ctx context.Context) context.Context {
    reqID := GetRequestID(ctx)
    if reqID == "" {
        return ctx
    }
    
    // Add to gRPC metadata
    // md := metadata.New(map[string]string{RequestIDHeader: reqID})
    // ctx = metadata.NewOutgoingContext(ctx, md)
    
    return ctx
}

func main() {
    ctx := context.Background()

    // Create propagator
    propagator := NewRequestIDPropagator()
    
    // Extract from headers (simulated)
    headers := http.Header{}
    headers.Set(RequestIDHeader, "existing-request-id")
    
    reqID, ctx := propagator.ExtractRequestID(ctx, headers)
    fmt.Printf("Request ID: %s\n", reqID)
    // Log with request ID
    LogWithRequestID(ctx, "info", "Processing request", map[string]interface\{\}{
        "user_id": "user-123",
    })
}
```

## Explanation

Let's break down what's happening:

1. **ID generation and extraction** - Notice how we first try to extract an existing request ID from context or headers, and only generate a new one if none exists. This ensures IDs propagate through the call chain.

2. **Context propagation** - We store the request ID in context, allowing it to be accessed throughout the request lifecycle. This makes it available for logging, tracing, and service calls.

3. **Header injection** - We inject the request ID into outgoing HTTP headers, ensuring it propagates to downstream services. This creates a distributed trace across services.

```go
**Key insight:** Always propagate request IDs through the entire call chain. Generate at entry points, propagate through headers/context, and include in all logs and traces for complete observability.

## Testing

```
Here's how to test this solution:
```go
func TestRequestIDPropagator_ExtractsFromHeader(t *testing.T) {
    propagator := NewRequestIDPropagator()
    headers := http.Header{}
    headers.Set(RequestIDHeader, "test-id")
    
    reqID, ctx := propagator.ExtractRequestID(context.Background(), headers)
    require.Equal(t, "test-id", reqID)
    require.Equal(t, "test-id", GetRequestID(ctx))
}

## Variations

### Hierarchical Request IDs

Maintain parent-child relationships:
type HierarchicalRequestID struct {
    ParentID string
    ChildID  string
}
```

### Request ID Formatting

Use custom formats (timestamp, service name):
```go
func (rip *RequestIDPropagator) GenerateWithFormat(service string) string {
    return fmt.Sprintf("%s-%s-%s", service, time.Now().Format("20060102"), uuid.New().String()[:8])
}
```

## Related Recipes

- **[Server Rate Limiting per Project](./server-rate-limiting-per-project.md)** - Per-project rate limiting
- **[Monitoring Trace Aggregation for Multi-agents](./monitoring-trace-aggregation-multi-agents.md)** - Aggregate traces
- **[Server Package Guide](../package_design_patterns.md)** - For a deeper understanding of server
