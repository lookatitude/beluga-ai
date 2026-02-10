// Package nats provides a NATS JetStream KV-backed [workflow.WorkflowStore]
// implementation for durable workflow state persistence.
//
// It uses NATS Key-Value stores for reliable, distributed workflow state
// management. Workflow state is stored as JSON values keyed by workflow ID.
// The store can use an existing NATS connection or create its own.
//
// # Usage
//
//	store, err := nats.New(nats.Config{
//	    URL:    "nats://localhost:4222",
//	    Bucket: "workflows",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer store.Close()
//
//	executor := workflow.NewExecutor(workflow.WithStore(store))
//
// # Configuration
//
// [Config] accepts a URL (defaults to nats.DefaultURL), a Bucket name
// (defaults to "beluga_workflows"), and an optional pre-existing *nats.Conn.
// When a Conn is provided, the store does not own the connection and will
// not close it.
//
// # Dependencies
//
// This provider requires the NATS Go client (github.com/nats-io/nats.go)
// and JetStream support (github.com/nats-io/nats.go/jetstream).
package nats
