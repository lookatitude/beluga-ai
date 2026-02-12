---
title: "Workflow Providers — Durable Execution"
description: "6 workflow providers for durable execution: Temporal, NATS, Kafka, Dapr, Inngest, in-memory. Fault-tolerant agent workflows in Go with Beluga AI."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "workflow engine, durable execution, Temporal, NATS, Kafka, Go workflow, fault-tolerant, Beluga AI"
---

Beluga AI v2 includes a durable execution engine that manages long-running, fault-tolerant workflows with activity execution, signal handling, retry policies, and event-sourced state persistence. Workflow state providers implement the `WorkflowStore` interface for persisting execution history, while durable executor providers implement the `DurableExecutor` interface for executing workflows with production-grade guarantees.

## Core Interfaces

### DurableExecutor

The primary interface for executing and managing workflows:

```go
type DurableExecutor interface {
    Execute(ctx context.Context, fn WorkflowFunc, opts WorkflowOptions) (WorkflowHandle, error)
    Signal(ctx context.Context, workflowID string, signal Signal) error
    Query(ctx context.Context, workflowID string, queryType string) (any, error)
    Cancel(ctx context.Context, workflowID string) error
}
```

### WorkflowStore

The interface for persisting workflow state:

```go
type WorkflowStore interface {
    Save(ctx context.Context, state WorkflowState) error
    Load(ctx context.Context, workflowID string) (*WorkflowState, error)
    List(ctx context.Context, filter WorkflowFilter) ([]WorkflowState, error)
    Delete(ctx context.Context, workflowID string) error
}
```

### WorkflowContext

Extended context for deterministic workflow execution:

```go
type WorkflowContext interface {
    context.Context

    ExecuteActivity(fn ActivityFunc, input any, opts ...ActivityOption) (any, error)
    ReceiveSignal(name string) <-chan any
    Sleep(d time.Duration) error
}
```

### WorkflowHandle

Access to running or completed workflow executions:

```go
type WorkflowHandle interface {
    ID() string
    RunID() string
    Status() WorkflowStatus
    Result(ctx context.Context) (any, error)
}
```

## Registry Usage

```go
import (
    "context"
    "log"

    "github.com/lookatitude/beluga-ai/workflow"

    // Register the executor provider via blank import
    _ "github.com/lookatitude/beluga-ai/workflow/providers/temporal"
)

func main() {
    executor, err := workflow.New("temporal", workflow.Config{
        Extra: map[string]any{
            "client":     temporalClient,
            "task_queue": "my-workflows",
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    handle, err := executor.Execute(ctx, myWorkflow, workflow.WorkflowOptions{
        ID: "order-123",
    })
    if err != nil {
        log.Fatal(err)
    }

    result, err := handle.Result(ctx)
    if err != nil {
        log.Fatal(err)
    }
}
```

## Available Providers

### Durable Executors

| Provider | Registry Name | Type | Best For |
|---|---|---|---|
| [Default (built-in)](#default-executor) | `default` | In-process goroutine | Development, testing, simple workflows |
| [Temporal](/providers/workflow/temporal) | `temporal` | External orchestrator | Production distributed workflows |

### Workflow Stores

| Provider | Type | Durability | Best For |
|---|---|---|---|
| [In-Memory](/providers/workflow/inmemory) | In-process map | None (process-local) | Development, testing |
| [Dapr](/providers/workflow/dapr) | Dapr state store | Depends on backend | Cloud-native, multi-cloud |
| [Inngest](/providers/workflow/inngest) | HTTP API | Durable | Event-driven serverless |
| [Kafka](/providers/workflow/kafka) | Compacted topic | Durable | Event streaming architectures |
| [NATS](/providers/workflow/nats) | JetStream KV | Durable | Lightweight distributed |
| [Temporal](/providers/workflow/temporal) | Temporal visibility | Managed by Temporal | Temporal-native workflows |

## Default Executor

Beluga provides a built-in `DefaultExecutor` that runs workflows in goroutines with optional state persistence:

```go
import (
    "github.com/lookatitude/beluga-ai/workflow"
    "github.com/lookatitude/beluga-ai/workflow/providers/inmemory"
)

store := inmemory.New()

executor := workflow.NewExecutor(
    workflow.WithStore(store),
    workflow.WithExecutorHooks(workflow.Hooks{
        OnWorkflowStart: func(ctx context.Context, wfID string, input any) {
            log.Printf("Workflow %s started", wfID)
        },
        OnWorkflowComplete: func(ctx context.Context, wfID string, result any) {
            log.Printf("Workflow %s completed: %v", wfID, result)
        },
    }),
)
```

## Workflow Patterns

### Defining a Workflow

```go
func orderWorkflow(ctx workflow.WorkflowContext, input any) (any, error) {
    orderID := input.(string)

    // Execute activities within the workflow context
    validated, err := ctx.ExecuteActivity(validateOrder, orderID,
        workflow.WithActivityRetry(workflow.DefaultRetryPolicy()),
        workflow.WithActivityTimeout(30*time.Second),
    )
    if err != nil {
        return nil, fmt.Errorf("validate: %w", err)
    }

    // Wait for human approval via signal
    approvalCh := ctx.ReceiveSignal("approve")
    select {
    case approval := <-approvalCh:
        if approval != "approved" {
            return nil, fmt.Errorf("order rejected")
        }
    case <-ctx.Done():
        return nil, ctx.Err()
    }

    // Process the order
    result, err := ctx.ExecuteActivity(processOrder, validated)
    if err != nil {
        return nil, fmt.Errorf("process: %w", err)
    }

    return result, nil
}
```

### Executing a Workflow

```go
handle, err := executor.Execute(ctx, orderWorkflow, workflow.WorkflowOptions{
    ID:      "order-456",
    Input:   "ORD-456",
    Timeout: 30 * time.Minute,
})
if err != nil {
    log.Fatal(err)
}

// Send a signal to the running workflow
err = executor.Signal(ctx, "order-456", workflow.Signal{
    Name:    "approve",
    Payload: "approved",
})
if err != nil {
    log.Fatal(err)
}

// Wait for the result
result, err := handle.Result(ctx)
if err != nil {
    log.Fatal(err)
}
```

### Activity Helpers

Beluga provides factory functions for common activity patterns:

```go
// LLM-powered activity
llmAct := workflow.LLMActivity(func(ctx context.Context, prompt string) (string, error) {
    return model.Generate(ctx, prompt)
})

// Tool execution activity
toolAct := workflow.ToolActivity(func(ctx context.Context, name string, args map[string]any) (any, error) {
    return registry.Execute(ctx, name, args)
})

// Human-in-the-loop activity
humanAct := workflow.HumanActivity(hitlManager)
```

## Retry Policies

Activities support configurable retry policies with exponential backoff and jitter:

```go
policy := workflow.RetryPolicy{
    MaxAttempts:        5,
    InitialInterval:   200 * time.Millisecond,
    BackoffCoefficient: 2.0,
    MaxInterval:        30 * time.Second,
}

result, err := ctx.ExecuteActivity(riskyActivity, input,
    workflow.WithActivityRetry(policy),
)
```

The default retry policy uses 3 attempts with 100ms initial interval and 2x backoff.

## Hooks

Hooks observe workflow lifecycle events without modifying execution:

```go
hooks := workflow.Hooks{
    OnWorkflowStart:    func(ctx context.Context, wfID string, input any) { ... },
    OnWorkflowComplete: func(ctx context.Context, wfID string, result any) { ... },
    OnWorkflowFail:     func(ctx context.Context, wfID string, err error) { ... },
    OnActivityStart:    func(ctx context.Context, wfID string, input any) { ... },
    OnActivityComplete: func(ctx context.Context, wfID string, result any) { ... },
    OnSignal:           func(ctx context.Context, wfID string, signal workflow.Signal) { ... },
    OnRetry:            func(ctx context.Context, wfID string, err error) { ... },
}

// Compose multiple hook sets
combined := workflow.ComposeHooks(loggingHooks, metricsHooks)
```

## Middleware

Middleware wraps a `DurableExecutor` to add cross-cutting behavior:

```go
type Middleware func(DurableExecutor) DurableExecutor

// Apply middleware (first in list is outermost)
executor = workflow.ApplyMiddleware(executor, loggingMW, metricsMW)

// Use the WithHooks middleware
executor = workflow.ApplyMiddleware(executor,
    workflow.WithHooks(hooks),
)
```

## Provider Discovery

List all registered executor providers at runtime:

```go
names := workflow.List()
// Returns sorted list: ["default", "temporal"]
```

## Choosing a Provider

| Use Case | Recommended |
|---|---|
| Development and testing | Default executor + [In-Memory](/providers/workflow/inmemory) store |
| Production distributed workflows | [Temporal](/providers/workflow/temporal) executor |
| Cloud-native microservices | Default executor + [Dapr](/providers/workflow/dapr) store |
| Event-driven serverless | Default executor + [Inngest](/providers/workflow/inngest) store |
| Event streaming architectures | Default executor + [Kafka](/providers/workflow/kafka) store |
| Lightweight distributed systems | Default executor + [NATS](/providers/workflow/nats) store |
