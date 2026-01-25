---
title: "Workflow Checkpointing"
package: "orchestration"
category: "resilience"
complexity: "advanced"
---

# Workflow Checkpointing

## Problem

You need to save workflow state at checkpoints so you can resume long-running workflows after failures, interruptions, or deployments without losing progress.

## Solution

Implement a checkpoint system that saves workflow state at strategic points, stores checkpoints persistently, and can resume workflows from any checkpoint. This works because you can serialize workflow state, store it, and restore it to continue execution.

## Code Example
```go
package main

import (
    "context"
    "encoding/json"
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
    WorkflowID    string
    CurrentStep   string
    CompletedSteps []string
    State         map[string]interface{}
    CheckpointID  string
    Timestamp     time.Time
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
    // In-memory implementation - return single state
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
    ID        string
    Execute   func(ctx context.Context, state map[string]interface{}) (map[string]interface{}, error)
    Checkpoint bool // Whether to checkpoint after this step
}

// NewCheckpointedWorkflow creates a new checkpointed workflow
func NewCheckpointedWorkflow(workflowID string, store CheckpointStore) *CheckpointedWorkflow {
    return &CheckpointedWorkflow{
        workflowID: workflowID,
        steps:      []WorkflowStep{},
        store:      store,
        state: &WorkflowState{
            WorkflowID:    workflowID,
            CompletedSteps: []string{},
            State:         make(map[string]interface{}),
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

Let's break down what's happening:

1. **State serialization** - Notice how we store workflow state including completed steps, current step, and workflow data. This allows us to resume exactly where we left off.

2. **Strategic checkpointing** - We checkpoint at strategic points (marked by steps), not after every operation. This balances safety with performance.

3. **Resume capability** - When resuming, we load the checkpoint and continue from the current step. This makes workflows resilient to failures.

```go
**Key insight:** Checkpoint at logical boundaries (after completing significant work), not after every operation. This provides safety without excessive overhead.

## Testing

```
Here's how to test this solution:
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

## Variations

### Incremental Checkpointing

Checkpoint incrementally instead of full state:
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

- **[Orchestration Parallel Node Execution in Graphs](./orchestration-parallel-node-execution.md)** - Parallel execution
- **[Core Advanced Context Timeout Management](./core-advanced-context-timeout-management.md)** - Timeout handling
- **[Orchestration Package Guide](../package_design_patterns.md)** - For a deeper understanding of orchestration
