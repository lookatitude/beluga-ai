---
title: Long-Running Workflows with Durable Execution
description: Build durable, long-running agent workflows that survive process restarts using the workflow package and optional Temporal integration.
---

Standard agents run in memory. If the process restarts, the agent's state is lost. For workflows that span hours or days, require human approval signals, or need reliable retries, the `workflow` package provides a durable execution engine with event-sourced state persistence.

## What You Will Build

A research workflow that executes an agent as an activity, waits for a human approval signal, and produces a final report. The workflow survives process restarts and handles timeouts gracefully.

## Prerequisites

- Familiarity with the `agent` and `orchestration` packages
- Understanding of context and error handling in Go

## Core Concepts

### Durable Execution

The `workflow` package defines a `DurableExecutor` interface for executing workflows that persist their state. The built-in executor uses event sourcing, but Temporal can be used as a provider for production deployments.

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

An `ActivityFunc` performs a unit of work within a workflow. Activities can have retry policies and timeouts:

```go
type ActivityFunc func(ctx context.Context, input any) (any, error)
```

## Step 1: Define Activities

Wrap your AI logic as workflow activities:

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

A `WorkflowFunc` receives a `WorkflowContext` for deterministic execution. Use it to execute activities and wait for signals:

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

Create a `DurableExecutor` and start the workflow:

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

External systems or humans send signals to running workflows to approve, reject, or provide data:

```go
func approveWorkflow(ctx context.Context, executor workflow.DurableExecutor, workflowID string) error {
    return executor.Signal(ctx, workflowID, workflow.Signal{
        Name:    "approval",
        Payload: true,
    })
}
```

## Step 5: Retrieve Results

The `WorkflowHandle` blocks until the workflow completes:

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

For production deployments requiring distributed execution, persistent storage, and advanced features, use the Temporal provider:

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
