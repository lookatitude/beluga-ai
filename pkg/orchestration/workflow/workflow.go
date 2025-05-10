package workflow

import (
	"context"
	"fmt"

	// Import other necessary packages like tasks, scheduler, messagebus
)

// WorkflowState represents the state of a workflow instance.
type WorkflowState string

const (
	WorkflowStatePending   WorkflowState = "PENDING"
	WorkflowStateRunning   WorkflowState = "RUNNING"
	WorkflowStateCompleted WorkflowState = "COMPLETED"
	WorkflowStateFailed    WorkflowState = "FAILED"
	WorkflowStateCancelled WorkflowState = "CANCELLED"
)

// Task represents a single step or unit of work within a workflow.
// This might be more detailed in a real system, including dependencies, inputs, outputs, etc.
type Task struct {
	ID          string
	Name        string
	ExecuteFunc func(ctx context.Context, inputs map[string]interface{}) (map[string]interface{}, error)
	// Add other fields like dependencies, retries, timeout
}

// Workflow defines the interface for a workflow.
// A workflow is a sequence or graph of tasks that achieve a larger goal.
type Workflow interface {
	// GetID returns the unique identifier of the workflow definition.
	GetID() string
	// GetName returns the human-readable name of the workflow.
	GetName() string
	// GetTasks returns the tasks that make up the workflow.
	// The order or structure might imply execution order or dependencies.
	GetTasks() []Task
	// Execute starts the workflow with the given initial inputs.
	// It returns the final output of the workflow or an error.
	Execute(ctx context.Context, initialInputs map[string]interface{}) (map[string]interface{}, error)
}

// WorkflowInstance represents a running or completed instance of a workflow definition.
type WorkflowInstance struct {
	InstanceID   string
	WorkflowID   string
	State        WorkflowState
	CurrentTask  string
	Inputs       map[string]interface{}
	Outputs      map[string]interface{}
	Error        error
	// Add timestamps, logs, etc.
}

// BaseWorkflow provides a basic structure for workflow implementations.
type BaseWorkflow struct {
	ID    string
	Name  string
	Tasks []Task
	// Potentially add a reference to a scheduler or message bus if the workflow manages its own execution flow
}

// NewBaseWorkflow creates a new BaseWorkflow.
func NewBaseWorkflow(id, name string, tasks []Task) *BaseWorkflow {
	return &BaseWorkflow{
		ID:    id,
		Name:  name,
		Tasks: tasks,
	}
}

func (bw *BaseWorkflow) GetID() string      { return bw.ID }
func (bw *BaseWorkflow) GetName() string    { return bw.Name }
func (bw *BaseWorkflow) GetTasks() []Task { return bw.Tasks }

// Execute runs the workflow tasks sequentially.
// This is a very simple sequential executor. Real workflows can be much more complex (parallel tasks, conditions, loops).
func (bw *BaseWorkflow) Execute(ctx context.Context, initialInputs map[string]interface{}) (map[string]interface{}, error) {
	currentOutputs := initialInputs
	var err error

	fmt.Printf("Starting workflow: %s (ID: %s)\n", bw.Name, bw.ID)

	for _, task := range bw.Tasks {
		fmt.Printf("Executing task: %s (ID: %s) in workflow %s\n", task.Name, task.ID, bw.Name)
		// Here, you might pass previous task outputs as inputs to the current task
		// This requires a more sophisticated input/output mapping mechanism
		currentOutputs, err = task.ExecuteFunc(ctx, currentOutputs)
		if err != nil {
			fmt.Printf("Error executing task %s (ID: %s) in workflow %s: %v\n", task.Name, task.ID, bw.Name, err)
			return nil, fmt.Errorf("workflow %s failed at task %s: %w", bw.Name, task.Name, err)
		}
		fmt.Printf("Task %s (ID: %s) completed successfully.\n", task.Name, task.ID)
	}

	fmt.Printf("Workflow %s (ID: %s) completed successfully.\n", bw.Name, bw.ID)
	return currentOutputs, nil
}

