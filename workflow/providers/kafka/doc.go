// Package kafka provides a Kafka-backed [workflow.WorkflowStore] implementation
// for the Beluga AI workflow engine.
//
// Workflow state is stored as JSON messages in a Kafka compacted topic, where
// the workflow ID serves as the message key. Deletions are performed by writing
// tombstone messages (nil value). Load and List are served from an in-memory
// index of the latest state per workflow. The index starts empty in a new
// process; call [Store.Restore] at startup to rebuild it from the topic,
// otherwise states written before a restart are not visible.
//
// # Usage
//
//	store, err := kafka.New(kafka.Config{
//	    Writer: kafkaWriter,
//	    Reader: kafkaReader,
//	    Topic:  "beluga-workflows",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer store.Close()
//
//	// Rebuild the index from the compacted topic after a restart.
//	rctx, cancel := context.WithTimeout(ctx, 10*time.Second)
//	_, err = store.Restore(rctx)
//	cancel()
//
//	executor := workflow.NewExecutor(workflow.WithStore(store))
//
// # Configuration
//
// [Config] requires a [Writer] for producing messages. A [Reader] is optional
// and only used by [Store.Restore]. This package does not connect to Kafka
// itself: broker addresses and the topic are configured on the Writer and
// Reader you supply.
//
// # Testing
//
// Use [NewWithWriterReader] with mock Writer and Reader implementations for
// unit testing without a running Kafka broker.
package kafka
