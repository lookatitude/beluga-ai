package messagebus

import (
	"context"
	"fmt"
	"sync"
)

// Message represents a generic message that can be passed on the bus.
// It includes a topic, a payload, and optional metadata.
type Message struct {
	Payload  any
	Metadata map[string]any
	ID       string
	Topic    string
}

// HandlerFunc is a function type that processes a message.
// It receives the context and the message.
type HandlerFunc func(ctx context.Context, msg Message) error

// MessageBus defines the interface for an asynchronous message passing system.
// This allows different components of the AI framework to communicate in a decoupled manner.
type MessageBus interface {
	// Publish sends a message to a specific topic.
	Publish(ctx context.Context, topic string, payload any, metadata map[string]any) error

	// Subscribe registers a handler function for a given topic.
	// Multiple handlers can subscribe to the same topic.
	// The subscriberID is returned and can be used to unsubscribe.
	Subscribe(ctx context.Context, topic string, handler HandlerFunc) (string, error)

	// Unsubscribe removes a handler for a given topic using its subscriberID.
	Unsubscribe(ctx context.Context, topic, subscriberID string) error

	// Start begins processing messages. This might be a no-op for some implementations.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the message bus, ensuring all pending messages are processed if possible.
	Stop(ctx context.Context) error

	// GetName returns the name of the message bus provider (e.g., "inmemory", "kafka", "redis").
	GetName() string
}

// InMemoryMessageBus is a simple in-memory implementation of the MessageBus interface.
// It is suitable for single-process applications or testing.
type InMemoryMessageBus struct {
	subscribers map[string]map[string]HandlerFunc
	stopChan    chan struct{}
	name        string
	nextSubID   int
	mu          sync.RWMutex
}

// NewInMemoryMessageBus creates a new InMemoryMessageBus.
func NewInMemoryMessageBus() *InMemoryMessageBus {
	return &InMemoryMessageBus{
		subscribers: make(map[string]map[string]HandlerFunc),
		nextSubID:   1,
		name:        "inmemory",
		stopChan:    make(chan struct{}),
	}
}

// Publish sends a message to all subscribers of the topic.
// In this simple implementation, handlers are called synchronously.
func (imb *InMemoryMessageBus) Publish(ctx context.Context, topic string, payload any, metadata map[string]any) error {
	imb.mu.Lock() // Changed from RLock to Lock to allow modification of nextSubID
	defer imb.mu.Unlock()

	msgID := imb.nextSubID // Get current ID
	imb.nextSubID++        // Increment for the next message or subscriber

	msg := Message{
		// ID should be generated, e.g., using uuid.NewString()
		ID:       fmt.Sprintf("msg-%d", msgID), // Use the captured msgID
		Topic:    topic,
		Payload:  payload,
		Metadata: metadata,
	}

	if topicSubscribers, ok := imb.subscribers[topic]; ok {
		for _, handler := range topicSubscribers {
			// In a real async bus, this would be non-blocking (e.g., send to a channel for the handler goroutine)
			go func(h HandlerFunc, m Message) { // Execute handler in a new goroutine for pseudo-asynchronicity
				err := h(ctx, m)
				if err != nil {
					// TODO: Add proper logging for handler errors
					_, _ = fmt.Printf("InMemoryMessageBus: error in handler for topic %s: %v\n", m.Topic, err)
				}
			}(handler, msg)
		}
	}
	return nil
}

// Subscribe registers a handler for a topic.
func (imb *InMemoryMessageBus) Subscribe(ctx context.Context, topic string, handler HandlerFunc) (string, error) {
	imb.mu.Lock()
	defer imb.mu.Unlock()

	if _, ok := imb.subscribers[topic]; !ok {
		imb.subscribers[topic] = make(map[string]HandlerFunc)
	}

	subscriberID := fmt.Sprintf("sub-%d", imb.nextSubID)
	imb.nextSubID++
	imb.subscribers[topic][subscriberID] = handler
	return subscriberID, nil
}

// Unsubscribe removes a handler from a topic.
func (imb *InMemoryMessageBus) Unsubscribe(ctx context.Context, topic, subscriberID string) error {
	imb.mu.Lock()
	defer imb.mu.Unlock()

	if topicSubscribers, ok := imb.subscribers[topic]; ok {
		delete(topicSubscribers, subscriberID)
		if len(topicSubscribers) == 0 {
			delete(imb.subscribers, topic)
		}
	}
	return nil
}

// Start is a no-op for this simple synchronous in-memory bus.
func (imb *InMemoryMessageBus) Start(ctx context.Context) error {
	// For a more complex bus, this would start worker goroutines, etc.
	return nil
}

// Stop signals the message bus to stop.
func (imb *InMemoryMessageBus) Stop(ctx context.Context) error {
	close(imb.stopChan)
	// Add any cleanup logic if necessary
	return nil
}

// GetName returns the name of the message bus.
func (imb *InMemoryMessageBus) GetName() string {
	return imb.name
}

// Ensure InMemoryMessageBus implements the MessageBus interface.
var _ MessageBus = (*InMemoryMessageBus)(nil)
