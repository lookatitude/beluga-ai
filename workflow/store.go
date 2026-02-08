package workflow

import "context"

// WorkflowStore is the interface for persisting workflow state.
// Implementations provide durable storage for workflow execution history.
type WorkflowStore interface {
	// Save persists the workflow state.
	Save(ctx context.Context, state WorkflowState) error

	// Load retrieves the workflow state by ID.
	Load(ctx context.Context, workflowID string) (*WorkflowState, error)

	// List returns workflows matching the filter.
	List(ctx context.Context, filter WorkflowFilter) ([]WorkflowState, error)

	// Delete removes a workflow state by ID.
	Delete(ctx context.Context, workflowID string) error
}
