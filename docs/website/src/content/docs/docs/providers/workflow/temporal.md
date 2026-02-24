---
title: "Temporal Workflow Provider"
description: "Production-grade durable execution with Temporal in Beluga AI. Distributed workflow orchestration with retries, signals, and versioning in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Temporal, durable execution, workflow orchestration, distributed workflows, fault-tolerant, Go, Beluga AI"
---

The Temporal provider implements both the `workflow.DurableExecutor` and `workflow.WorkflowStore` interfaces using [Temporal](https://temporal.io). Temporal is a distributed, durable execution platform that provides workflow orchestration with fault tolerance, signal handling, activity execution, and replay-based recovery. This provider maps Beluga workflow concepts directly to Temporal primitives.

Choose Temporal when you need production-grade durable execution with full workflow orchestration: activity retries, signal-based human-in-the-loop, deterministic replay for recovery, and distributed task queues. Temporal is the right choice for long-running, mission-critical workflows where fault tolerance is essential. For lightweight development and testing, use the [In-Memory](/docs/providers/workflow/inmemory) store instead.

This is the only provider that registers into the `workflow.DurableExecutor` registry via `init()`, making it available through `workflow.New("temporal", cfg)`.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/workflow/providers/temporal
```

Start a local Temporal server for development:

```bash
temporal server start-dev
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "go.temporal.io/sdk/client"

    "github.com/lookatitude/beluga-ai/workflow"
    _ "github.com/lookatitude/beluga-ai/workflow/providers/temporal"
)

func main() {
    // Create a Temporal SDK client
    c, err := client.Dial(client.Options{
        HostPort: "localhost:7233",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer c.Close()

    // Create executor via the registry
    executor, err := workflow.New("temporal", workflow.Config{
        Extra: map[string]any{
            "client":     c,
            "task_queue": "my-workflows",
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    handle, err := executor.Execute(ctx, myWorkflow, workflow.WorkflowOptions{
        ID:      "order-123",
        Input:   "process payment",
        Timeout: 30 * time.Minute,
    })
    if err != nil {
        log.Fatal(err)
    }

    result, err := handle.Result(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Result:", result)
}

func myWorkflow(ctx workflow.WorkflowContext, input any) (any, error) {
    result, err := ctx.ExecuteActivity(processPayment, input)
    if err != nil {
        return nil, fmt.Errorf("process payment: %w", err)
    }
    return result, nil
}

func processPayment(ctx context.Context, input any) (any, error) {
    return fmt.Sprintf("paid: %v", input), nil
}
```

## Configuration

### Executor Configuration

| Parameter | Type | Default | Description |
|---|---|---|---|
| `Client` | `client.Client` | (required) | Temporal SDK client |
| `TaskQueue` | `string` | `"beluga-workflows"` | Task queue for workflows and activities |
| `DefaultTimeout` | `time.Duration` | `10 * time.Minute` | Default workflow execution timeout |

### Registry Configuration

When using the registry (`workflow.New`), pass configuration via `Extra`:

| Key | Type | Default | Description |
|---|---|---|---|
| `"client"` | `client.Client` | (required) | Temporal SDK client |
| `"task_queue"` | `string` | `"beluga-workflows"` | Task queue name |

## Direct Construction

Create an executor directly without the registry:

```go
import (
    "go.temporal.io/sdk/client"
    temporalwf "github.com/lookatitude/beluga-ai/workflow/providers/temporal"
)

c, err := client.Dial(client.Options{
    HostPort: "localhost:7233",
})
if err != nil {
    log.Fatal(err)
}
defer c.Close()

executor, err := temporalwf.NewExecutor(temporalwf.Config{
    Client:         c,
    TaskQueue:      "my-queue",
    DefaultTimeout: 1 * time.Hour,
})
if err != nil {
    log.Fatal(err)
}
```

## DurableExecutor API

### Execute

```go
func (e *Executor) Execute(ctx context.Context, fn workflow.WorkflowFunc, opts workflow.WorkflowOptions) (workflow.WorkflowHandle, error)
```

Starts a new Temporal workflow execution. The Beluga `WorkflowFunc` is wrapped in a Temporal-compatible function that creates a `WorkflowContext` bridging Beluga operations to Temporal primitives. If no ID is provided in `opts`, one is generated using a timestamp.

### Signal

```go
func (e *Executor) Signal(ctx context.Context, workflowID string, signal workflow.Signal) error
```

Sends a signal to a running Temporal workflow. The `Signal.Name` maps to the Temporal signal name and `Signal.Payload` carries the data.

### Query

```go
func (e *Executor) Query(ctx context.Context, workflowID string, queryType string) (any, error)
```

Queries a running Temporal workflow. The `queryType` is passed as the Temporal query type. The response is deserialized into `any`.

### Cancel

```go
func (e *Executor) Cancel(ctx context.Context, workflowID string) error
```

Requests cancellation of a running Temporal workflow.

## WorkflowContext

The Temporal provider bridges the Beluga `WorkflowContext` interface to Temporal's workflow primitives:

### ExecuteActivity

```go
func (c *temporalContext) ExecuteActivity(fn workflow.ActivityFunc, input any, opts ...workflow.ActivityOption) (any, error)
```

Executes an activity within the Temporal workflow context. Activities run on the configured task queue with a default 5-minute start-to-close timeout. The result is decoded from Temporal's response payload.

### ReceiveSignal

```go
func (c *temporalContext) ReceiveSignal(name string) <-chan any
```

Returns a Go channel that delivers signal payloads. Internally, this launches a Temporal coroutine (`workflow.Go`) that listens on the named signal channel and bridges payloads to a standard Go channel.

### Sleep

```go
func (c *temporalContext) Sleep(d time.Duration) error
```

Pauses workflow execution for the specified duration using Temporal's deterministic timer. Unlike `time.Sleep`, this is recorded in the workflow history and replayed correctly during recovery.

## WorkflowHandle

The `WorkflowHandle` returned by `Execute` wraps a Temporal `WorkflowRun`:

```go
handle, err := executor.Execute(ctx, myWorkflow, opts)

// Get the workflow ID
id := handle.ID()

// Get the Temporal run ID
runID := handle.RunID()

// Check status (always "running" until Result() is called)
status := handle.Status()

// Wait for the workflow to complete
result, err := handle.Result(ctx)
```

## WorkflowStore

The Temporal provider also implements `workflow.WorkflowStore` for scenarios where you need the store interface. Since Temporal manages state internally, most operations are lightweight:

```go
store := temporalwf.NewStore(temporalClient, "default")
```

| Method | Behavior |
|---|---|
| `Save` | No-op (Temporal manages state) |
| `Load` | Retrieves the workflow run and returns basic state |
| `List` | Returns `nil` (requires Temporal visibility features) |
| `Delete` | No-op (Temporal manages lifecycle) |

## Signals and Human-in-the-Loop

Temporal's signal mechanism integrates with Beluga's workflow patterns for human-in-the-loop scenarios:

```go
func approvalWorkflow(ctx workflow.WorkflowContext, input any) (any, error) {
    // Execute the initial processing activity
    processed, err := ctx.ExecuteActivity(preProcess, input)
    if err != nil {
        return nil, err
    }

    // Wait for human approval via signal
    approvalCh := ctx.ReceiveSignal("approval")
    decision := <-approvalCh
    if decision != "approved" {
        return nil, fmt.Errorf("rejected by reviewer")
    }

    // Continue processing after approval
    return ctx.ExecuteActivity(postProcess, processed)
}

// Send approval from an external system
err := executor.Signal(ctx, "wf-123", workflow.Signal{
    Name:    "approval",
    Payload: "approved",
})
```

## Retry Policies

Temporal retry policies can be configured at the workflow level via Beluga's `RetryPolicy`:

```go
// The Temporal provider maps Beluga retry policies to Temporal retry policies:
// - MaxAttempts      -> MaximumAttempts
// - InitialInterval  -> InitialInterval
// - BackoffCoefficient -> BackoffCoefficient
// - MaxInterval      -> MaximumInterval
```

Activity-level retries use Beluga's `WithActivityRetry` option, which is applied via the Temporal activity options.

## Dependencies

This provider depends on the Temporal Go SDK:

- `go.temporal.io/sdk/client`
- `go.temporal.io/sdk/workflow`
- `go.temporal.io/sdk/temporal`

## Context Limitations

The Temporal `WorkflowContext` bridges Temporal's deterministic execution model to Go's `context.Context`. Note the following:

- `Done()` returns `nil` because Temporal uses its own internal channel type for cancellation. Use `Err()` to check cancellation status instead of `select` on `Done()`.
- `Sleep` is durable -- it survives process restarts, unlike `time.Sleep`.
- Activities run within the configured task queue and are distributed across Temporal workers.
- Signal channels are bridged via a Temporal coroutine (`workflow.Go`), so they behave like standard Go channels from the caller's perspective.

## Production Considerations

- **Task Queue**: Use dedicated task queues per workflow type for isolation.
- **Timeouts**: Set appropriate workflow and activity timeouts to prevent resource leaks.
- **Workers**: Deploy Temporal workers to execute the registered workflows and activities.
- **Namespace**: Use separate Temporal namespaces for development, staging, and production.
- **Retry Policies**: Configure retry policies on activities to handle transient failures gracefully.
