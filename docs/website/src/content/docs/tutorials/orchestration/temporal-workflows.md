---
title: Long-Running Workflows with Durable Execution
description: Build durable, long-running agent workflows that survive process restarts using the workflow package and optional Temporal integration.
---

Standard agents run in memory. If the process restarts, the agent's state is lost -- including any work it has already completed. For workflows that span hours or days, require human approval signals, or need reliable retries across failures, the `workflow` package provides a durable execution engine with event-sourced state persistence. Beluga AI owns its durable execution engine rather than depending on Temporal by default, but provides Temporal as a provider option for production deployments that need distributed execution and advanced features.

## What You Will Build

A research workflow that executes an agent as an activity, waits for a human approval signal, and produces a final report. The workflow survives process restarts and handles timeouts gracefully.

## Prerequisites

- Familiarity with the `agent` and `orchestration` packages
- Understanding of context and error handling in Go

## Core Concepts

### Durable Execution

The `workflow` package defines a `DurableExecutor` interface that follows Beluga AI's registry pattern -- `workflow.New()` creates an executor by name ("default" for the built-in engine, "temporal" for the Temporal provider). The built-in executor uses event sourcing to persist workflow state: every activity completion, signal receipt, and state transition is recorded as an event. On restart, the executor replays the event log to reconstruct the workflow's state without re-executing completed activities.

```go
import "github.com/lookatitude/beluga-ai/workflow"

// DurableExecutor manages workflow lifecycle.
type DurableExecutor interface {
    Execute(ctx context.Context, fn WorkflowFunc, opts WorkflowOptions) (WorkflowHandle, error)
    Signal(ctx context.Context, workflowID string, signal Signal) error
    Query(ctx context.Context, workflowID string, queryType string) (any, error)
    Cancel(ctx context.Context, workflowID string) error
}
```

### Activities

An `ActivityFunc` performs a unit of work within a workflow. Activities are the boundary between deterministic workflow logic and non-deterministic side effects (API calls, LLM invocations, database writes). By wrapping side effects in activities, the workflow engine knows which operations need to be re-executed on replay and which can be skipped because their results were already recorded.

```go
type ActivityFunc func(ctx context.Context, input any) (any, error)
```

## Step 1: Define Activities

Each activity encapsulates a unit of AI work. The research activity would call an LLM agent in production; the report activity generates the final output. Activities can have retry policies (for transient failures like rate limits) and timeouts (to bound execution time for slow operations). These are configured per-activity rather than globally, because different operations have different reliability characteristics.

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/lookatitude/beluga-ai/workflow"
)

// researchActivity performs the AI research step.
func researchActivity(ctx context.Context, input any) (any, error) {
    topic := input.(string)
    // In a real application, this would call an LLM agent.
    return fmt.Sprintf("Research findings on %q: key insights discovered.", topic), nil
}

// reportActivity generates the final report.
func reportActivity(ctx context.Context, input any) (any, error) {
    findings := input.(string)
    return fmt.Sprintf("Final Report\n\n%s\n\nConclusion: Action recommended.", findings), nil
}
```

## Step 2: Define the Workflow

A `WorkflowFunc` receives a `WorkflowContext` for deterministic execution. The `WorkflowContext` provides methods for executing activities, waiting for signals, and sleeping -- all of which are recorded in the event log for durability. The workflow itself must be deterministic: given the same inputs and signals, it must produce the same sequence of activity calls. This is why you use `ctx.ExecuteActivity()` rather than calling functions directly -- the context tracks which activities have already completed during replay.

The `WaitForSignal` call pauses the workflow until an external signal arrives or the timeout expires. This is how human-in-the-loop approval works: the workflow suspends, a human reviews the research findings, and sends an approval or rejection signal. During the wait, the workflow's state is persisted, so a process restart will resume the wait where it left off.

```go
func researchWorkflow(ctx workflow.WorkflowContext, input any) (any, error) {
    topic := input.(string)

    // Step 1: Execute the research activity with retry.
    findings, err := ctx.ExecuteActivity(researchActivity, topic,
        workflow.WithActivityRetry(workflow.RetryPolicy{
            MaxAttempts:  3,
            InitialDelay: 5 * time.Second,
        }),
        workflow.WithActivityTimeout(10 * time.Minute),
    )
    if err != nil {
        return nil, fmt.Errorf("research failed: %w", err)
    }

    // Step 2: Wait for human approval signal.
    signal, err := ctx.WaitForSignal("approval", 24*time.Hour)
    if err != nil {
        return nil, fmt.Errorf("approval timed out: %w", err)
    }

    approved, ok := signal.Payload.(bool)
    if !ok || !approved {
        return "Workflow rejected by reviewer.", nil
    }

    // Step 3: Generate the final report.
    report, err := ctx.ExecuteActivity(reportActivity, findings,
        workflow.WithActivityTimeout(5 * time.Minute),
    )
    if err != nil {
        return nil, fmt.Errorf("report generation failed: %w", err)
    }

    return report, nil
}
```

## Step 3: Execute the Workflow

The `WorkflowOptions` include a unique ID for the workflow instance, which serves as the correlation key for signals, queries, and cancellation. The timeout bounds the total workflow duration, providing a safety net for workflows that might otherwise run indefinitely waiting for signals that never arrive.

```go
func main() {
    ctx := context.Background()

    // Create executor using the built-in engine.
    executor, err := workflow.New("default", workflow.Config{})
    if err != nil {
        fmt.Printf("executor creation failed: %v\n", err)
        return
    }

    // Start the workflow.
    handle, err := executor.Execute(ctx, researchWorkflow, workflow.WorkflowOptions{
        ID:      "research-q4-2025",
        Input:   "Q4 revenue analysis",
        Timeout: 48 * time.Hour,
    })
    if err != nil {
        fmt.Printf("workflow start failed: %v\n", err)
        return
    }

    fmt.Printf("Workflow started: %s (run: %s)\n", handle.ID(), handle.RunID())
}
```

## Step 4: Send Signals

External systems or humans send signals to running workflows to provide data, approve actions, or trigger transitions. The signal name ("approval") must match what the workflow is waiting for in its `WaitForSignal` call. In a production system, this would be called from an API handler when a human clicks "Approve" in a review dashboard.

```go
func approveWorkflow(ctx context.Context, executor workflow.DurableExecutor, workflowID string) error {
    return executor.Signal(ctx, workflowID, workflow.Signal{
        Name:    "approval",
        Payload: true,
    })
}
```

## Step 5: Retrieve Results

The `WorkflowHandle` blocks until the workflow completes, providing a synchronous API for waiting on asynchronous workflows. The `Status()` method reports the current state (running, completed, failed, timed out), and `Result()` returns the final output.

```go
func waitForResult(ctx context.Context, handle workflow.WorkflowHandle) {
    result, err := handle.Result(ctx)
    if err != nil {
        fmt.Printf("workflow failed: %v\n", err)
        return
    }

    fmt.Printf("Status: %s\n", handle.Status())
    fmt.Printf("Result: %v\n", result)
}
```

## Using Temporal as a Provider

For production deployments requiring distributed execution across multiple workers, persistent storage with a database backend, and advanced features like visibility queries and workflow versioning, use the Temporal provider. The provider wraps your `WorkflowFunc` and `ActivityFunc` types into native Temporal workflows and activities, handling serialization, retries, and signal routing transparently. Your workflow code does not change -- only the provider name and configuration.

```go
import _ "github.com/lookatitude/beluga-ai/workflow/providers/temporal"

executor, err := workflow.New("temporal", workflow.Config{
    Extra: map[string]any{
        "host":       "localhost:7233",
        "namespace":  "default",
        "task_queue": "beluga-tasks",
    },
})
```

The Temporal provider wraps your `WorkflowFunc` and `ActivityFunc` types into native Temporal workflows and activities, handling serialization, retries, and signal routing transparently.

## Verification

1. Start a workflow execution.
2. Query the workflow status and verify it reports `running`.
3. Send an approval signal.
4. Verify the workflow completes and returns the expected report.
5. Test timeout behavior by not sending a signal within the timeout window.

## Next Steps

- [DAG Workflows](/tutorials/orchestration/dag-workflows) -- Complex graph-based flows that can be used as activities
- [Human-in-the-Loop](/tutorials/safety/human-in-loop) -- Confidence-based approval policies for tool execution
