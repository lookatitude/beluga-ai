// Package inngest provides an Inngest-backed [workflow.WorkflowStore]
// implementation for the Beluga AI workflow engine.
//
// It stores workflow state using Inngest's event-driven durable execution
// platform via its HTTP API. State is persisted remotely through PUT/GET/DELETE
// requests, and an in-memory cache is maintained for listing operations.
//
// # Usage
//
//	store, err := inngest.New(inngest.Config{
//	    BaseURL:  "http://localhost:8288",
//	    EventKey: "my-event-key",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	executor := workflow.NewExecutor(workflow.WithStore(store))
//
// # Configuration
//
// [Config] accepts a BaseURL (defaults to "http://localhost:8288"), an optional
// EventKey for authentication, and an optional [HTTPClient] (defaults to
// http.DefaultClient).
package inngest
