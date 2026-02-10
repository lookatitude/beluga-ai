// Package kafka provides a Kafka-backed [workflow.WorkflowStore] implementation
// for the Beluga AI workflow engine.
//
// Workflow state is stored as JSON messages in a Kafka compacted topic, where
// the workflow ID serves as the message key. Deletions are performed by writing
// tombstone messages (nil value). An in-memory cache provides fast reads while
// Kafka provides durable persistence.
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
//	executor := workflow.NewExecutor(workflow.WithStore(store))
//
// # Configuration
//
// [Config] requires a [Writer] for producing messages. A [Reader] is optional
// and used for consuming messages from the topic. The Topic defaults to
// "beluga-workflows".
//
// # Testing
//
// Use [NewWithWriterReader] with mock Writer and Reader implementations for
// unit testing without a running Kafka broker.
package kafka
