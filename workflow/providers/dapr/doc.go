// Package dapr provides a Dapr state store-backed [workflow.WorkflowStore]
// implementation for the Beluga AI workflow engine.
//
// It uses Dapr's state management API for persisting workflow state as
// JSON-encoded documents, with workflow IDs as keys. The list of workflow IDs
// is persisted under [IndexKey] so that List works across processes and
// restarts (best effort under concurrent writers; Load by ID is
// authoritative).
//
// # Usage
//
//	store, err := dapr.New(dapr.Config{
//	    Client:    daprClient,
//	    StoreName: "statestore",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	executor := workflow.NewExecutor(workflow.WithStore(store))
//
// # Configuration
//
// [Config] accepts a [StateClient] (the subset of the Dapr client interface
// used for state operations) and an optional StoreName (defaults to "statestore").
//
// # Testing
//
// Use [NewWithClient] with a mock [StateClient] implementation for unit testing
// without a running Dapr sidecar.
package dapr
