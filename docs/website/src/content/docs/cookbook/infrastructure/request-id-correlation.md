---
title: "Correlating Request IDs across Services"
description: "Propagate and correlate request IDs across distributed services for debugging and monitoring."
---

## Problem

You need to correlate requests across multiple services in a distributed system, tracking a request's journey from API gateway through multiple microservices for debugging and monitoring.

## Solution

Implement request ID propagation that generates unique request IDs at entry points, propagates them through all service calls (HTTP headers, context, logs), and correlates them in centralized observability systems. This works because you can inject request IDs into Go context and propagate them through the call chain.

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

1. **ID generation and extraction** -- The propagator first checks context, then headers, and only generates a new ID as a last resort. This ensures IDs propagate through the full call chain.

2. **Context propagation** -- The request ID is stored in context, making it accessible throughout the request lifecycle for logging, tracing, and downstream calls.

3. **Header injection** -- The request ID is set on outgoing HTTP headers to propagate to downstream services, creating a distributed trace across the system.

**Key insight:** Always propagate request IDs through the entire call chain. Generate at entry points, propagate through headers and context, and include in all logs and traces for complete observability.

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

- **[Rate Limiting per Project](./rate-limiting)** -- Per-project rate limiting
- **[Trace Aggregation for Multi-Agents](./trace-aggregation)** -- Aggregate traces across agents
