// Package temporal provides a Temporal-backed [workflow.DurableExecutor] and
// [workflow.WorkflowStore] for the Beluga workflow engine.
//
// It maps Beluga workflows to Temporal workflows and Beluga activities to
// Temporal activities. The package wraps the Temporal SDK client to provide
// a seamless integration with Beluga's workflow interfaces.
//
// # Executor
//
// [Executor] implements [workflow.DurableExecutor] backed by Temporal. It
// translates Beluga workflow execution into Temporal workflow runs:
//
//	executor, err := temporal.NewExecutor(temporal.Config{
//	    Client:    temporalClient,
//	    TaskQueue: "beluga-workflows",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	handle, err := executor.Execute(ctx, myWorkflow, workflow.WorkflowOptions{
//	    ID: "order-123",
//	})
//	result, err := handle.Result(ctx)
//
// # Store
//
// [Store] implements [workflow.WorkflowStore] using Temporal's visibility API.
// Since Temporal manages workflow state internally, Save and Delete are no-ops.
// Load retrieves workflow state by getting the Temporal workflow run:
//
//	store := temporal.NewStore(temporalClient, "default")
//
// # Registry Integration
//
// The Temporal provider registers itself as "temporal" in the workflow registry
// via init(). Create an executor through the registry:
//
//	executor, err := workflow.New("temporal", workflow.Config{
//	    Extra: map[string]any{
//	        "client":     temporalClient,
//	        "task_queue": "my-queue",
//	    },
//	})
//
// # Dependencies
//
// This provider requires the Temporal Go SDK (go.temporal.io/sdk).
package temporal
