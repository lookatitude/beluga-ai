package messagebus

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga/pkg/schema" // Assuming schema.Message is defined
)

// MessageHandler defines the function signature for handling messages.
type MessageHandler func(ctx context.Context, message schema.Message) error

// MessageBus defines the interface for a message bus system.
// It allows for publishing messages to topics and subscribing to topics to receive messages.
type MessageBus interface {
	// Publish sends a message to a specific topic.
	Publish(ctx context.Context, topic string, message schema.Message) error
	// Subscribe registers a handler for a specific topic.
	// Multiple handlers can be registered for the same topic.
	Subscribe(topic string, handler MessageHandler) error
	// Unsubscribe removes a handler for a specific topic.
	// This might be more complex in a real system, e.g., requiring a subscription ID.
	// Unsubscribe(topic string, handler MessageHandler) error 
	// Start starts the message bus processing (if applicable, e.g., for background dispatching).
	Start(ctx context.Context) error
	// Stop gracefully shuts down the message bus.
	Stop() error
}

// InMemoryMessageBus is a simple in-memory implementation of the MessageBus interface.
// It is suitable for single-process applications or testing.
// For distributed systems, a more robust message bus like Kafka, RabbitMQ, or NATS would be used.
type InMemoryMessageBus struct {
	subscribers map[string][]MessageHandler
	mu          sync.RWMutex
	stopChan    chan struct{}
}

// NewInMemoryMessageBus creates a new InMemoryMessageBus.
func NewInMemoryMessageBus() *InMemoryMessageBus {
	return &InMemoryMessageBus{
		subscribers: make(map[string][]MessageHandler),
		stopChan:    make(chan struct{}),
	}
}

// Publish sends a message to all subscribers of a topic.
// In this in-memory implementation, handlers are called synchronously.
// A more advanced bus might use goroutines for asynchronous dispatch.
func (imb *InMemoryMessageBus) Publish(ctx context.Context, topic string, message schema.Message) error {
	imb.mu.RLock()
	defer imb.mu.RUnlock()

	handlers, ok := imb.subscribers[topic]
	if !ok {
		// No subscribers for this topic, or handle as an error if required
		fmt.Printf("No subscribers for topic: %s\n", topic)
		return nil
	}

	for _, handler := range handlers {
		// In a production system, consider error handling and recovery per handler.
		// Also, consider if handlers should be called concurrently.
		err := handler(ctx, message)
		if err != nil {
			fmt.Printf("Error handling message on topic %s: %v\n", topic, err)
			// Continue to other handlers or return error based on requirements
		}
	}
	return nil
}

// Subscribe registers a handler for a specific topic.
func (imb *InMemoryMessageBus) Subscribe(topic string, handler MessageHandler) error {
	imb.mu.Lock()
	defer imb.mu.Unlock()

	imb.subscribers[topic] = append(imb.subscribers[topic], handler)
	fmt.Printf("Handler subscribed to topic: %s\n", topic)
	return nil
}

// Start for InMemoryMessageBus is a no-op as publishing is synchronous.
// It can be extended if background processing is added.
func (imb *InMemoryMessageBus) Start(ctx context.Context) error {
	fmt.Println("In-memory message bus started (synchronous dispatch).")
	// Keep running until context is done or stop is called
	go func() {
		select {
		case <-ctx.Done():
			fmt.Println("Message bus context cancelled, stopping.")
			return
		case <-imb.stopChan:
			fmt.Println("Message bus received stop signal.")
			return
		}
	}()
	return nil
}

// Stop signals the message bus to stop.
func (imb *InMemoryMessageBus) Stop() error {
	close(imb.stopChan)
	fmt.Println("In-memory message bus stop requested.")
	return nil
}

