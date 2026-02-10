// Package inmemory provides an in-memory [workflow.WorkflowStore] for
// development and testing.
//
// State is stored in a thread-safe map and does not persist across process
// restarts. This provider is useful for unit tests, local development,
// and prototyping workflows before selecting a durable backend.
//
// # Usage
//
//	store := inmemory.New()
//
//	executor := workflow.NewExecutor(workflow.WithStore(store))
//	handle, err := executor.Execute(ctx, myWorkflow, workflow.WorkflowOptions{
//	    ID: "test-workflow",
//	})
package inmemory
