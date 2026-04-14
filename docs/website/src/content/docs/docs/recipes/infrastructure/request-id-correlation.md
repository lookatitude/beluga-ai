---
title: "Request ID Correlation"
description: "Recipe for propagating and correlating request IDs across distributed Go services — trace agent workflows through LLMs, tools, and MCP servers."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, request ID correlation, Go distributed tracing, request propagation, microservice debugging, trace context, observability recipe"
---

## Problem

You need to correlate requests across multiple services in a distributed system, tracking a request's journey from API gateway through multiple microservices for debugging and monitoring. This is essential for distributed tracing in microservice architectures where a single user request triggers calls across dozens of services. Without request correlation, debugging failures requires manually searching logs across services, trying to match timestamps and hoping you find related log entries. For AI systems, this problem is amplified: an agent workflow might invoke multiple LLM providers, query vector databases, execute tools via MCP servers, and hand off to other agents—all part of the same logical request. When something fails, you need to see the complete execution path across all these services to diagnose the root cause.

## Solution

Implement request ID propagation that generates unique request IDs at entry points, propagates them through all service calls (HTTP headers, context, logs), and correlates them in centralized observability systems. This works because you can inject request IDs into Go context and propagate them through the call chain. The design follows distributed tracing best practices: generate IDs once at the system boundary, propagate them as context values within a service and as HTTP headers between services, and include them in all logs and traces. This approach integrates with Beluga's context propagation patterns (every public function takes context.Context as first parameter) and OpenTelemetry's trace correlation, allowing you to reconstruct complete request flows across services.

The key insight is that request IDs must flow through three layers: HTTP headers for inter-service communication, context.Context for intra-service propagation, and structured logs for post-hoc analysis. The propagator checks context first (already injected), then headers (from upstream), and only generates a new ID as a last resort (entry point). This ensures IDs propagate through the full call chain without generating duplicates at service boundaries.

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
	"go.opentelemetry.io/otel/trace"

	"github.com/google/uuid"
)

var tracer = otel.Tracer("beluga.server.request_correlation")

const RequestIDHeader = "X-Request-ID"

type contextKey string

const requestIDKey contextKey = "request_id"

// RequestIDPropagator propagates request IDs across services.
type RequestIDPropagator struct {
	generator func() string
}

func NewRequestIDPropagator() *RequestIDPropagator {
	return &RequestIDPropagator{
		generator: func() string {
			return uuid.New().String()
		},
	}
}

// ExtractRequestID extracts or generates a request ID, storing it in context.
func (rip *RequestIDPropagator) ExtractRequestID(ctx context.Context, headers http.Header) (string, context.Context) {
	ctx, span := tracer.Start(ctx, "request_id.extract")
	defer span.End()

	if reqID, ok := ctx.Value(requestIDKey).(string); ok && reqID != "" {
		span.SetAttributes(attribute.String("request_id", reqID))
		return reqID, ctx
	}

	if reqID := headers.Get(RequestIDHeader); reqID != "" {
		ctx = context.WithValue(ctx, requestIDKey, reqID)
		span.SetAttributes(attribute.String("request_id", reqID))
		return reqID, ctx
	}

	reqID := rip.generator()
	ctx = context.WithValue(ctx, requestIDKey, reqID)
	span.SetAttributes(attribute.String("request_id", reqID))
	span.SetStatus(trace.StatusOK, "request ID generated")
	return reqID, ctx
}

// InjectRequestID injects request ID into outgoing HTTP headers.
func (rip *RequestIDPropagator) InjectRequestID(ctx context.Context, headers http.Header) {
	if reqID := GetRequestID(ctx); reqID != "" {
		headers.Set(RequestIDHeader, reqID)
	}
}

// GetRequestID retrieves the request ID from context.
func GetRequestID(ctx context.Context) string {
	if reqID, ok := ctx.Value(requestIDKey).(string); ok {
		return reqID
	}
	return ""
}

// RequestIDMiddleware creates HTTP middleware for request ID handling.
func RequestIDMiddleware(propagator *RequestIDPropagator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID, ctx := propagator.ExtractRequestID(r.Context(), r.Header)
			w.Header().Set(RequestIDHeader, reqID)
			r = r.WithContext(ctx)

			span := trace.SpanFromContext(ctx)
			if span.IsRecording() {
				span.SetAttributes(attribute.String("request_id", reqID))
			}

			next.ServeHTTP(w, r)
		})
	}
}

func main() {
	propagator := NewRequestIDPropagator()

	headers := http.Header{}
	headers.Set(RequestIDHeader, "existing-request-id")

	reqID, ctx := propagator.ExtractRequestID(context.Background(), headers)
	fmt.Printf("Request ID: %s\n", reqID)

	log.Printf("[%s] Processing request", GetRequestID(ctx))
}
```

## Explanation

1. **ID generation and extraction** — The propagator first checks context, then headers, and only generates a new ID as a last resort. This ensures IDs propagate through the full call chain. This matters because request IDs must remain stable across service boundaries—generating a new ID at each service would fragment the trace and make it impossible to correlate logs. The three-tier check (context → header → generate) handles all scenarios: internal function calls use context, cross-service calls use headers, and entry points generate new IDs. This design prevents duplicate IDs within a single request flow while ensuring every request has an ID for correlation.

2. **Context propagation** — The request ID is stored in context, making it accessible throughout the request lifecycle for logging, tracing, and downstream calls. This matters because Go's context.Context is the standard mechanism for passing request-scoped values through function call chains. By storing the request ID in context, it automatically flows through all functions that follow Beluga's convention of accepting context as the first parameter. This eliminates the need to manually pass request IDs as function parameters or rely on global state, which would not be thread-safe in Go's concurrent execution model.

3. **Header injection** — The request ID is set on outgoing HTTP headers to propagate to downstream services, creating a distributed trace across the system. This matters because context values are process-local—they don't automatically cross service boundaries. When making HTTP calls to other services, the request ID must be explicitly injected into headers. The X-Request-ID header is a de facto standard for this purpose, used by load balancers, API gateways, and observability tools. By injecting the ID into headers, you ensure downstream services can extract it and continue the trace, creating a complete picture of request flow across your distributed system.

Always propagate request IDs through the entire call chain. Generate at entry points, propagate through headers and context, and include in all logs and traces for complete observability.

## Variations

### Hierarchical Request IDs

Maintain parent-child relationships for sub-requests:

```go
type HierarchicalRequestID struct {
	ParentID string
	ChildID  string
}

func GenerateChildID(parentID string) string {
	return fmt.Sprintf("%s.%s", parentID, uuid.New().String()[:8])
}
```

## Related Recipes

- **[Rate Limiting per Project](./rate-limiting)** — Per-project rate limiting
- **[Trace Aggregation for Multi-Agents](./trace-aggregation)** — Aggregate traces across agents
