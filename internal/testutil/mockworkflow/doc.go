// Package mockworkflow provides a mock implementation of the
// workflow.WorkflowStore interface for testing.
//
// This is an internal package and is not part of the public API. It is used
// by workflow engine tests that need a durable state store without an
// external database.
//
// # MockWorkflowStore
//
// [MockWorkflowStore] implements the workflow.WorkflowStore interface with
// in-memory state storage. It supports Save, Load, List, and Delete
// operations, with configurable error injection, custom functions for each
// operation, and call tracking for assertions.
//
// Create a mock with functional options:
//
//	store := mockworkflow.New(
//	    mockworkflow.WithStates([]workflow.WorkflowState{
//	        {WorkflowID: "wf-1", Status: "running"},
//	    }),
//	)
//
// Configure error injection:
//
//	store := mockworkflow.New(
//	    mockworkflow.WithError(errors.New("storage unavailable")),
//	)
//
// Use custom functions for dynamic behavior:
//
//	store := mockworkflow.New(
//	    mockworkflow.WithSaveFunc(func(ctx context.Context, state workflow.WorkflowState) error {
//	        // custom save logic
//	    }),
//	)
//
// Inspect call history:
//
//	_ = store.Save(ctx, state)
//	fmt.Println(store.SaveCalls()) // 1
//	fmt.Println(store.LastState()) // the saved state
//
// The mock is safe for concurrent use. Call [MockWorkflowStore.Reset] to
// clear all state between test cases.
package mockworkflow
