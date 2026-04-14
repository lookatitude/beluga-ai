---
title: "Workflow API — Durable Execution Engine"
description: "Workflow package API reference for Beluga AI. Durable execution with activities, signals, retry policies, and Temporal/Dapr/Kafka/NATS providers."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "workflow API, durable execution, DurableExecutor, activities, signals, Temporal, Dapr, Kafka, Beluga AI, Go, reference"
---

## workflow

```go
import "github.com/lookatitude/beluga-ai/workflow"
```

Package workflow provides a durable execution engine for the Beluga AI framework.

It enables reliable, long-running workflows with activity execution, signal
handling, retry policies, and event-sourced state persistence. The package
provides its own built-in execution engine and supports external providers
(Temporal, Dapr, Inngest, Kafka, NATS) via the registry pattern.

## Core Interfaces

The `DurableExecutor` interface manages workflow lifecycle:

```go
type DurableExecutor interface {
    Execute(ctx context.Context, fn WorkflowFunc, opts WorkflowOptions) (WorkflowHandle, error)
    Signal(ctx context.Context, workflowID string, signal Signal) error
    Query(ctx context.Context, workflowID string, queryType string) (any, error)
    Cancel(ctx context.Context, workflowID string) error
}
```

`WorkflowContext` extends context.Context with deterministic execution primitives:

```go
type WorkflowContext interface {
    context.Context
    ExecuteActivity(fn ActivityFunc, input any, opts ...ActivityOption) (any, error)
    ReceiveSignal(name string) <-chan any
    Sleep(d time.Duration) error
}
```

`WorkflowStore` persists workflow state for recovery and auditing.

## Defining Workflows

Workflows are plain Go functions that receive a `WorkflowContext`:

```go
func OrderWorkflow(ctx workflow.WorkflowContext, input any) (any, error) {
    // Execute an activity with retry
    result, err := ctx.ExecuteActivity(processPayment, input,
        workflow.WithActivityRetry(workflow.DefaultRetryPolicy()),
        workflow.WithActivityTimeout(30 * time.Second),
    )
    if err != nil {
        return nil, err
    }

    // Wait for an external signal
    ch := ctx.ReceiveSignal("approval")
    select {
    case approval := <-ch:
        return approval, nil
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}
```

## Executing Workflows

Use the `DefaultExecutor` or create one via the registry:

```go
executor := workflow.NewExecutor(
    workflow.WithStore(inmemory.New()),
    workflow.WithExecutorHooks(hooks),
)

handle, err := executor.Execute(ctx, OrderWorkflow, workflow.WorkflowOptions{
    ID:      "order-123",
    Input:   orderData,
    Timeout: 10 * time.Minute,
})

result, err := handle.Result(ctx)
```

## Activity Helpers

Pre-built activity constructors integrate with framework components:

- [LLMActivity] — wraps an LLM invocation as an activity
- [ToolActivity] — wraps a tool execution as an activity
- [HumanActivity] — wraps human-in-the-loop interaction as an activity

## Retry Policies

`RetryPolicy` configures exponential backoff with jitter for activities:

```go
policy := workflow.RetryPolicy{
    MaxAttempts:        5,
    InitialInterval:   100 * time.Millisecond,
    BackoffCoefficient: 2.0,
    MaxInterval:        30 * time.Second,
}
```

## Signals and Queries

Running workflows can receive external `Signal` messages and respond
to queries:

```go
err := executor.Signal(ctx, "order-123", workflow.Signal{
    Name:    "approval",
    Payload: "approved",
})

status, err := executor.Query(ctx, "order-123", "status")
```

## Event-Sourced State

Workflow execution is recorded as a sequence of `HistoryEvent` values in
`WorkflowState`. This enables replay-based recovery and audit trails.

## Registry

External providers register via `Register` and are created with `New`:

```go
// Registration (typically in init())
workflow.Register("temporal", temporalFactory)

// Creation
executor, err := workflow.New("temporal", workflow.Config{
    Extra: map[string]any{"client": temporalClient},
})

providers := workflow.List() // ["default", "temporal", ...]
```

## Hooks and Middleware

`Hooks` provide lifecycle callbacks for workflow events. `Middleware` wraps
a DurableExecutor to add cross-cutting behavior:

```go
hooks := workflow.Hooks{
    OnWorkflowStart:    func(ctx context.Context, id string, input any) { ... },
    OnWorkflowComplete: func(ctx context.Context, id string, result any) { ... },
    OnWorkflowFail:     func(ctx context.Context, id string, err error) { ... },
}

wrapped := workflow.ApplyMiddleware(executor, workflow.WithHooks(hooks))
```

---

## dapr

```go
import "github.com/lookatitude/beluga-ai/workflow/providers/dapr"
```

Package dapr provides a Dapr state store-backed [workflow.WorkflowStore]
implementation for the Beluga AI workflow engine.

It uses Dapr's state management API for persisting workflow state as
JSON-encoded documents, with workflow IDs as keys. An in-memory index
of workflow IDs is maintained for listing operations.

## Usage

```go
store, err := dapr.New(dapr.Config{
    Client:    daprClient,
    StoreName: "statestore",
})
if err != nil {
    log.Fatal(err)
}

executor := workflow.NewExecutor(workflow.WithStore(store))
```

## Configuration

`Config` accepts a `StateClient` (the subset of the Dapr client interface
used for state operations) and an optional StoreName (defaults to "statestore").

## Testing

Use `NewWithClient` with a mock `StateClient` implementation for unit testing
without a running Dapr sidecar.

---

## inmemory

```go
import "github.com/lookatitude/beluga-ai/workflow/providers/inmemory"
```

Package inmemory provides an in-memory [workflow.WorkflowStore] for
development and testing.

State is stored in a thread-safe map and does not persist across process
restarts. This provider is useful for unit tests, local development,
and prototyping workflows before selecting a durable backend.

## Usage

```go
store := inmemory.New()

executor := workflow.NewExecutor(workflow.WithStore(store))
handle, err := executor.Execute(ctx, myWorkflow, workflow.WorkflowOptions{
    ID: "test-workflow",
})
```

---

## inngest

```go
import "github.com/lookatitude/beluga-ai/workflow/providers/inngest"
```

Package inngest provides an Inngest-backed [workflow.WorkflowStore]
implementation for the Beluga AI workflow engine.

It stores workflow state using Inngest's event-driven durable execution
platform via its HTTP API. State is persisted remotely through PUT/GET/DELETE
requests, and an in-memory cache is maintained for listing operations.

## Usage

```go
store, err := inngest.New(inngest.Config{
    BaseURL:  "http://localhost:8288",
    EventKey: "my-event-key",
})
if err != nil {
    log.Fatal(err)
}

executor := workflow.NewExecutor(workflow.WithStore(store))
```

## Configuration

`Config` accepts a BaseURL (defaults to "http://localhost:8288"), an optional
EventKey for authentication, and an optional `HTTPClient` (defaults to
http.DefaultClient).

---

## kafka

```go
import "github.com/lookatitude/beluga-ai/workflow/providers/kafka"
```

Package kafka provides a Kafka-backed [workflow.WorkflowStore] implementation
for the Beluga AI workflow engine.

Workflow state is stored as JSON messages in a Kafka compacted topic, where
the workflow ID serves as the message key. Deletions are performed by writing
tombstone messages (nil value). An in-memory cache provides fast reads while
Kafka provides durable persistence.

## Usage

```go
store, err := kafka.New(kafka.Config{
    Writer: kafkaWriter,
    Reader: kafkaReader,
    Topic:  "beluga-workflows",
})
if err != nil {
    log.Fatal(err)
}
defer store.Close()

executor := workflow.NewExecutor(workflow.WithStore(store))
```

## Configuration

`Config` requires a `Writer` for producing messages. A `Reader` is optional
and used for consuming messages from the topic. The Topic defaults to
"beluga-workflows".

## Testing

Use `NewWithWriterReader` with mock Writer and Reader implementations for
unit testing without a running Kafka broker.

---

## nats

```go
import "github.com/lookatitude/beluga-ai/workflow/providers/nats"
```

Package nats provides a NATS JetStream KV-backed [workflow.WorkflowStore]
implementation for durable workflow state persistence.

It uses NATS Key-Value stores for reliable, distributed workflow state
management. Workflow state is stored as JSON values keyed by workflow ID.
The store can use an existing NATS connection or create its own.

## Usage

```go
store, err := nats.New(nats.Config{
    URL:    "nats://localhost:4222",
    Bucket: "workflows",
})
if err != nil {
    log.Fatal(err)
}
defer store.Close()

executor := workflow.NewExecutor(workflow.WithStore(store))
```

## Configuration

`Config` accepts a URL (defaults to nats.DefaultURL), a Bucket name
(defaults to "beluga_workflows"), and an optional pre-existing *nats.Conn.
When a Conn is provided, the store does not own the connection and will
not close it.

## Dependencies

This provider requires the NATS Go client (github.com/nats-io/nats.go)
and JetStream support (github.com/nats-io/nats.go/jetstream).

---

## temporal

```go
import "github.com/lookatitude/beluga-ai/workflow/providers/temporal"
```

Package temporal provides a Temporal-backed [workflow.DurableExecutor] and
[workflow.WorkflowStore] for the Beluga workflow engine.

It maps Beluga workflows to Temporal workflows and Beluga activities to
Temporal activities. The package wraps the Temporal SDK client to provide
a seamless integration with Beluga's workflow interfaces.

## Executor

`Executor` implements [workflow.DurableExecutor] backed by Temporal. It
translates Beluga workflow execution into Temporal workflow runs:

```go
executor, err := temporal.NewExecutor(temporal.Config{
    Client:    temporalClient,
    TaskQueue: "beluga-workflows",
})
if err != nil {
    log.Fatal(err)
}

handle, err := executor.Execute(ctx, myWorkflow, workflow.WorkflowOptions{
    ID: "order-123",
})
result, err := handle.Result(ctx)
```

## Store

`Store` implements [workflow.WorkflowStore] using Temporal's visibility API.
Since Temporal manages workflow state internally, Save and Delete are no-ops.
Load retrieves workflow state by getting the Temporal workflow run:

```go
store := temporal.NewStore(temporalClient, "default")
```

## Registry Integration

The Temporal provider registers itself as "temporal" in the workflow registry
via init(). Create an executor through the registry:

```go
executor, err := workflow.New("temporal", workflow.Config{
    Extra: map[string]any{
        "client":     temporalClient,
        "task_queue": "my-queue",
    },
})
```

## Dependencies

This provider requires the Temporal Go SDK (go.temporal.io/sdk).
