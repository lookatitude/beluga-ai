# Standard Library Context Deep Dive

Welcome, colleague! In this integration guide, we're going to dive deep into Go's standard library `context` package and how it integrates with Beluga AI's core package. Understanding context is crucial for building production-ready applications.

## What you will build

You will learn advanced context patterns for cancellation, timeouts, and value propagation in Beluga AI applications. This enables proper resource cleanup, graceful shutdowns, and request tracing.

## Learning Objectives

- ✅ Use context for cancellation and timeouts
- ✅ Propagate values through context
- ✅ Implement graceful shutdowns
- ✅ Understand context best practices

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- Understanding of Go concurrency

## Step 1: Basic Context Usage

Create a simple example with context:
```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/core"
)

func main() {
    // Create a context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Use context in Beluga AI operation
    runnable := core.NewRunnable(func(ctx context.Context, input interface{}) (interface{}, error) {
        // Check if context is cancelled
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        case <-time.After(2 * time.Second):
            return "Operation completed", nil
        }
    })

    result, err := runnable.Invoke(ctx, nil)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    fmt.Printf("Result: %v\n", result)
}
```

### Verification

Run the example:
bash
```bash
go run main.go
```

You should see:Result: Operation completed
```

## Step 2: Context Cancellation

Implement cancellation propagation:
```go
func processWithCancellation(ctx context.Context) error {
    // Create child context with cancellation
    childCtx, cancel := context.WithCancel(ctx)
    defer cancel()

    // Simulate work that can be cancelled
    done := make(chan error)
    go func() {
        // Long-running operation
        time.Sleep(10 * time.Second)
        done \<- nil
    }()

    select {
    case <-ctx.Done():
        return ctx.Err()
    case err := <-done:
        return err
    }
}
```

## Step 3: Context Values

Propagate values through context:
```go
package main

import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/pkg/core"
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
    ctx = withRequestID(ctx, "req-123")

    runnable := core.NewRunnable(func(ctx context.Context, input interface{}) (interface{}, error) {
        requestID := getRequestID(ctx)
        return fmt.Sprintf("Processing request: %s", requestID), nil
    })

    result, _ := runnable.Invoke(ctx, nil)
    fmt.Println(result)
}
```

## Step 4: Context with Timeout

Implement timeout handling:
```go
func operationWithTimeout(ctx context.Context, duration time.Duration) error {
    ctx, cancel := context.WithTimeout(ctx, duration)
    defer cancel()

    // Operation that respects timeout
    select {
    case <-time.After(duration + 1*time.Second):
        return fmt.Errorf("operation timed out")
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

## Step 5: Graceful Shutdown

Implement graceful shutdown with context:
```go
package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/core"
)

func main() {
    // Create context that cancels on interrupt
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Handle signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

    go func() {
        <-sigChan
        fmt.Println("Shutting down gracefully...")
        cancel()
    }()

    // Run application
    runnable := core.NewRunnable(func(ctx context.Context, input interface{}) (interface{}, error) {
        for {
            select {
            case <-ctx.Done():
                return nil, ctx.Err()
            default:
                fmt.Println("Working...")
                time.Sleep(1 * time.Second)
            }
        }
    })

    _, err := runnable.Invoke(ctx, nil)
    if err != nil && err != context.Canceled {
        fmt.Printf("Error: %v\n", err)
    }
}
```

## Configuration Options

| Pattern | Description | Use Case |
|---------|-------------|----------|
| `context.Background()` | Root context | Application initialization |
| `context.WithCancel()` | Cancellable context | Long-running operations |
| `context.WithTimeout()` | Timeout context | Time-limited operations |
| `context.WithDeadline()` | Deadline context | Absolute time limits |
| `context.WithValue()` | Value context | Request metadata |

## Common Issues

### "Context deadline exceeded"

**Problem**: Operation exceeded timeout.

**Solution**: Check context before long operations:if ctx.Err() != nil \{
```
    return ctx.Err()
}

### "Context cancelled"

**Problem**: Context was cancelled before completion.

**Solution**: Handle cancellation gracefully:if err == context.Canceled {
    // Graceful shutdown
text
    return nil
}
```

## Production Considerations

When using context in production:

- **Always pass context**: Pass context as first parameter
- **Respect cancellation**: Check `ctx.Done()` in loops
- **Propagate context**: Pass context to all child operations
- **Set timeouts**: Use timeouts for external calls
- **Clean up resources**: Use defer for cleanup

## Complete Example

Here's a complete, production-ready example:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/core"
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

    // Check for cancellation
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

## Next Steps

Congratulations! You've mastered context usage in Beluga AI. Next, learn how to:

- **[Zap/Logrus Logger Providers](./zap-logrus-providers.md)** - Structured logging integration
- **[Core Package Documentation](../../api-docs/packages/core.md)** - Deep dive into core package
- **[Best Practices Guide](../../best-practices.md)** - Production patterns

---

**Ready for more?** Check out the Integrations Index for more integration guides!
