---
title: Go Context Patterns
description: "Advanced context.Context patterns in Beluga AI for cancellation, timeouts, value propagation, and graceful shutdown of AI pipelines."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Go context, context.Context, Beluga AI, cancellation patterns, timeout management, graceful shutdown, Go AI framework"
---

LLM calls are slow and expensive. A single agent invocation may chain multiple model calls, tool executions, and retrieval operations that take seconds. Without proper context management, a cancelled HTTP request continues burning tokens, a timed-out operation holds connections indefinitely, and debugging across services becomes impossible without trace propagation. Every public function in Beluga AI accepts `context.Context` as its first parameter. Understanding how context flows through the framework is essential for building applications that handle cancellation, enforce timeouts, and propagate request-scoped metadata. This guide covers the context patterns you will use most often.

## Prerequisites

- Go 1.23 or later
- Beluga AI framework installed
- Familiarity with Go concurrency fundamentals

## Context Basics

Create a context with a timeout and pass it to a Beluga AI operation:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := processRequest(ctx, "hello")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Printf("Result: %s\n", result)
}

func processRequest(ctx context.Context, input string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(2 * time.Second):
		return fmt.Sprintf("Processed: %s", input), nil
	}
}
```

## Cancellation Propagation

When an outer context is cancelled, all derived child contexts are cancelled automatically. Use this to propagate shutdown signals through a chain of operations:

```go
func processWithCancellation(ctx context.Context) error {
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		// Simulate a long-running operation
		time.Sleep(10 * time.Second)
		done <- nil
	}()

	select {
	case <-childCtx.Done():
		return childCtx.Err()
	case err := <-done:
		return err
	}
}
```

The caller can cancel the parent context at any time, and the child operation will stop.

## Value Propagation

Attach request-scoped metadata to the context using typed keys. This avoids collisions with other packages:

```go
package main

import (
	"context"
	"fmt"
)

type contextKey string

const (
	requestIDKey contextKey = "request_id"
	userIDKey    contextKey = "user_id"
)

func withRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

func getRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

func main() {
	ctx := context.Background()
	ctx = withRequestID(ctx, "req-abc-123")

	fmt.Printf("Request ID: %s\n", getRequestID(ctx))
}
```

> **Guideline**: Only store request-scoped data in context values (trace IDs, request IDs, tenant IDs). Never store optional parameters or configuration -- use function arguments or options for those.

## Timeout Handling

Apply timeouts to external calls such as LLM invocations and tool executions. The timeout context ensures resources are freed if the call runs too long:

```go
func callWithTimeout(ctx context.Context, duration time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	resultCh := make(chan string, 1)
	go func() {
		// Simulate an external call
		time.Sleep(duration / 2)
		resultCh <- "done"
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case result := <-resultCh:
		return result, nil
	}
}
```

## Graceful Shutdown

Combine `context.WithCancel` with OS signal handling to shut down cleanly:

```go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("Received shutdown signal")
		cancel()
	}()

	err := runWorker(ctx)
	if err != nil && err != context.Canceled {
		fmt.Printf("Worker error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Shutdown complete")
}

func runWorker(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			fmt.Println("Working...")
			time.Sleep(1 * time.Second)
		}
	}
}
```

## Context Patterns Reference

| Pattern | Constructor | Use Case |
|---------|-------------|----------|
| Root context | `context.Background()` | Application initialization, top-level entry points |
| Cancellable | `context.WithCancel(parent)` | Long-running operations, shutdown propagation |
| Timeout | `context.WithTimeout(parent, d)` | Time-limited operations (LLM calls, HTTP requests) |
| Deadline | `context.WithDeadline(parent, t)` | Absolute time limits |
| Values | `context.WithValue(parent, k, v)` | Request metadata (trace ID, tenant ID) |

## Production Example with OTel Tracing

Combine context patterns with OpenTelemetry to propagate trace and span IDs through Beluga AI operations:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type ContextAwareService struct {
	tracer trace.Tracer
}

func NewContextAwareService() *ContextAwareService {
	return &ContextAwareService{
		tracer: otel.Tracer("beluga.core.context"),
	}
}

func (s *ContextAwareService) Process(ctx context.Context, input string) (string, error) {
	ctx, span := s.tracer.Start(ctx, "service.Process",
		trace.WithAttributes(
			attribute.String("input", input),
		),
	)
	defer span.End()

	// Check for cancellation before starting work
	select {
	case <-ctx.Done():
		span.RecordError(ctx.Err())
		return "", ctx.Err()
	default:
	}

	// Simulate work
	time.Sleep(100 * time.Millisecond)

	result := fmt.Sprintf("Processed: %s", input)
	span.SetAttributes(attribute.String("result", result))
	return result, nil
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	service := NewContextAwareService()
	result, err := service.Process(ctx, "test")
	if err != nil {
		log.Fatalf("Processing failed: %v", err)
	}

	fmt.Println(result)
}
```

## Troubleshooting

### Context deadline exceeded

The operation did not complete within the timeout. Increase the timeout or investigate why the downstream call is slow. Always check `ctx.Err()` before starting expensive work:

```go
if ctx.Err() != nil {
	return ctx.Err()
}
```

### Context cancelled

The parent context was cancelled, typically due to a shutdown signal or the caller abandoning the request. Handle gracefully:

```go
if err == context.Canceled {
	// Clean shutdown -- not an error
	return nil
}
```

## Production Considerations

- **Always propagate context** -- pass the incoming `context.Context` to every child call.
- **Check `ctx.Done()` in loops** -- long-running work should periodically check for cancellation.
- **Set timeouts on external calls** -- LLM invocations, HTTP requests, and database queries should all have timeouts.
- **Use `defer cancel()`** -- every `WithCancel`, `WithTimeout`, and `WithDeadline` must have a corresponding cancel call to avoid resource leaks.
- **Clean up resources on cancellation** -- use `defer` to release connections, close files, and flush buffers.

## Related Resources

- [Zap and Logrus Logger Providers](/docs/integrations/zap-logrus) -- Structured logging integration
- [Infrastructure Integrations](/docs/integrations/infrastructure) -- Deployment and infrastructure overview
