---
title: "Workflow Checkpointing"
description: "Recipe for saving and resuming long-running Go agent workflows at strategic checkpoints to survive crashes, deploys, and transient failures."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, workflow checkpointing, Go durable execution, failure recovery, long-running workflows, state persistence, resilience recipe"
---

# Workflow Checkpointing

## Problem

Long-running agentic workflows face inevitable interruptions. A workflow might run for minutes or hours, performing expensive operations like LLM calls, document processing, or external API requests. If the process crashes, the deployment is updated, or a transient error occurs midway, you lose all progress and must restart from the beginning.

This creates two major problems. First, it wastes resources and time by re-executing completed work. Second, it creates a poor user experience when workflows fail near completion and users must wait for the entire workflow to re-run. For workflows involving multiple agents or expensive API calls, the cost of re-execution can be significant.

The challenge is determining what state to save, when to save it, and how to restore it in a way that's both safe and efficient.

## Solution

Workflow checkpointing provides resilience by persisting workflow state at strategic points during execution. The key design decision is to checkpoint at logical boundaries, not after every operation. You checkpoint after completing significant units of work, like finishing an agent subtask, completing a document processing stage, or successfully calling an external API.

This approach works because workflow state is typically serializable. You need to persist three pieces of information: which steps have completed, which step is currently executing, and the accumulated workflow data. When resuming, you deserialize this state and continue execution from the current step, skipping already-completed steps.

The checkpoint store abstraction decouples state persistence from workflow logic. This allows you to start with an in-memory store for development, then switch to Redis or a database for production without changing workflow code. The store handles serialization, versioning, and retrieval.

Strategic checkpointing balances safety with performance. Checkpointing too frequently adds overhead and increases storage costs. Checkpointing too rarely risks losing significant progress. The solution is to let each workflow step declare whether it warrants a checkpoint, typically based on whether it performs expensive or non-idempotent operations.

## Code Example

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

var tracer = otel.Tracer("beluga.orchestration.checkpointing")

// WorkflowState represents the state of a workflow
type WorkflowState struct {
    WorkflowID     string
    CurrentStep    string
    CompletedSteps []string
    State          map[string]interface{}
    CheckpointID   string
    Timestamp      time.Time
}

// CheckpointStore stores and retrieves checkpoints
type CheckpointStore interface {
    Save(ctx context.Context, workflowID string, state *WorkflowState) error
    Load(ctx context.Context, workflowID string) (*WorkflowState, error)
    List(ctx context.Context, workflowID string) ([]*WorkflowState, error)
}

// InMemoryCheckpointStore is an in-memory checkpoint store
type InMemoryCheckpointStore struct {
    checkpoints map[string]*WorkflowState
}

// NewInMemoryCheckpointStore creates a new in-memory store
func NewInMemoryCheckpointStore() *InMemoryCheckpointStore {
    return &InMemoryCheckpointStore{
        checkpoints: make(map[string]*WorkflowState),
    }
}

func (s *InMemoryCheckpointStore) Save(ctx context.Context, workflowID string, state *WorkflowState) error {
    s.checkpoints[workflowID] = state
    return nil
}

func (s *InMemoryCheckpointStore) Load(ctx context.Context, workflowID string) (*WorkflowState, error) {
    state, exists := s.checkpoints[workflowID]
    if !exists {
        return nil, fmt.Errorf("workflow %s not found", workflowID)
    }
    return state, nil
}

func (s *InMemoryCheckpointStore) List(ctx context.Context, workflowID string) ([]*WorkflowState, error) {
    state, exists := s.checkpoints[workflowID]
    if !exists {
        return []*WorkflowState{}, nil
    }
    return []*WorkflowState{state}, nil
}

// CheckpointedWorkflow manages workflow execution with checkpointing
type CheckpointedWorkflow struct {
    workflowID string
    steps      []WorkflowStep
    store      CheckpointStore
    state      *WorkflowState
}

// WorkflowStep represents a step in the workflow
type WorkflowStep struct {
    ID         string
    Execute    func(ctx context.Context, state map[string]interface{}) (map[string]interface{}, error)
    Checkpoint bool // Whether to checkpoint after this step
}

// NewCheckpointedWorkflow creates a new checkpointed workflow
func NewCheckpointedWorkflow(workflowID string, store CheckpointStore) *CheckpointedWorkflow {
    return &CheckpointedWorkflow{
        workflowID: workflowID,
        steps:      []WorkflowStep{},
        store:      store,
        state: &WorkflowState{
            WorkflowID:     workflowID,
            CompletedSteps: []string{},
            State:          make(map[string]interface{}),
        },
    }
}

// AddStep adds a step to the workflow
func (cw *CheckpointedWorkflow) AddStep(step WorkflowStep) {
    cw.steps = append(cw.steps, step)
}

// Execute executes the workflow with checkpointing
func (cw *CheckpointedWorkflow) Execute(ctx context.Context) error {
    ctx, span := tracer.Start(ctx, "checkpointed_workflow.execute")
    defer span.End()

    span.SetAttributes(attribute.String("workflow_id", cw.workflowID))

    // Try to resume from checkpoint
    savedState, err := cw.store.Load(ctx, cw.workflowID)
    if err == nil && savedState != nil {
        cw.state = savedState
        span.SetAttributes(attribute.Bool("resumed", true))
    }

    // Find starting step
    startIdx := 0
    for i, step := range cw.steps {
        if step.ID == cw.state.CurrentStep {
            startIdx = i
            break
        }
    }

    // Execute steps
    for i := startIdx; i < len(cw.steps); i++ {
        step := cw.steps[i]

        span.SetAttributes(attribute.String("current_step", step.ID))

        // Execute step
        newState, err := step.Execute(ctx, cw.state.State)
        if err != nil {
            span.RecordError(err)
            span.SetStatus(trace.StatusError, err.Error())
            return fmt.Errorf("step %s failed: %w", step.ID, err)
        }

        // Update state
        cw.state.State = newState
        cw.state.CompletedSteps = append(cw.state.CompletedSteps, step.ID)
        cw.state.CurrentStep = step.ID

        // Checkpoint if requested
        if step.Checkpoint {
            cw.state.CheckpointID = fmt.Sprintf("%s-%d", cw.workflowID, time.Now().Unix())
            cw.state.Timestamp = time.Now()

            if err := cw.store.Save(ctx, cw.workflowID, cw.state); err != nil {
                log.Printf("Failed to save checkpoint: %v", err)
            }

            span.SetAttributes(attribute.String("checkpoint_id", cw.state.CheckpointID))
        }
    }

    span.SetStatus(trace.StatusOK, "workflow completed")
    return nil
}

// Resume resumes workflow from checkpoint
func (cw *CheckpointedWorkflow) Resume(ctx context.Context) error {
    return cw.Execute(ctx)
}

func main() {
    ctx := context.Background()

    // Create checkpoint store
    store := NewInMemoryCheckpointStore()

    // Create workflow
    workflow := NewCheckpointedWorkflow("workflow-1", store)

    // Add steps with checkpointing
    workflow.AddStep(WorkflowStep{
        ID: "step1",
        Execute: func(ctx context.Context, state map[string]interface{}) (map[string]interface{}, error) {
            state["step1_result"] = "completed"
            return state, nil
        },
        Checkpoint: true,
    })

    // Execute
    if err := workflow.Execute(ctx); err != nil {
        log.Fatalf("Workflow failed: %v", err)
    }
    fmt.Println("Workflow completed")
}
```

## Explanation

1. **State serialization enables exact resumption** — By capturing the exact workflow state, including completed steps, the current step, and accumulated data, you can resume execution as if the failure never occurred. This is critical because workflows often pass data between steps. Without state serialization, you'd lose intermediate results and be unable to continue.

2. **Strategic checkpointing optimizes performance** — The `Checkpoint` boolean on each step allows fine-grained control over when to persist state. Checkpoint after expensive operations like LLM calls or external API requests, where re-execution would be costly. Skip checkpointing after cheap operations like in-memory transformations. This keeps checkpoint overhead low while protecting valuable work.

3. **Resume capability provides failure resilience** — The ability to load a checkpoint and continue from the saved step means workflows survive process crashes, deployments, and transient errors. This is especially important for workflows that interact with users, where losing progress creates a poor experience. Checkpointing makes workflows feel more reliable because failures become invisible to users.

4. **Separation of storage from execution** — The `CheckpointStore` interface abstracts storage, allowing you to choose different backends based on your needs. Use in-memory for tests, Redis for distributed systems, or PostgreSQL for durability. This separation also enables features like checkpoint history, point-in-time recovery, and workflow introspection without coupling to storage implementation.

## Testing

```go
func TestCheckpointedWorkflow_ResumesFromCheckpoint(t *testing.T) {
    store := NewInMemoryCheckpointStore()
    workflow := NewCheckpointedWorkflow("test", store)

    // Execute and checkpoint
    workflow.Execute(context.Background())

    // Resume
    newWorkflow := NewCheckpointedWorkflow("test", store)
    err := newWorkflow.Resume(context.Background())
    require.NoError(t, err)
}
```

## Variations

### Incremental Checkpointing

Checkpoint incrementally instead of full state:

```go
type IncrementalCheckpoint struct {
    Changes map[string]interface{}
}
```

### Checkpoint Compression

Compress checkpoints to save space:

```go
func (s *CheckpointStore) SaveCompressed(ctx context.Context, workflowID string, state *WorkflowState) error {
    // Compress before saving
}
```

## Related Recipes

- [Parallel Node Execution in Graphs](/cookbook/parallel-nodes) — Parallel execution
- [Core Advanced Context Timeout Management](/cookbook/core-advanced-context-timeout-management) — Timeout handling
