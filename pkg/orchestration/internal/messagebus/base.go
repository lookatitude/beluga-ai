// Package messagebus defines interfaces and mechanisms for inter-agent communication.
package messagebus

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time" // Added missing import
)

// Note: Message and MessageBus types are defined in messagebus.go to avoid duplication

// --- In-Memory Channel-Based Message Bus ---

// ChannelMessageBus implements MessageBus using Go channels for local communication.
// Note: This implementation uses the MessageBus interface from messagebus.go.
type ChannelMessageBus struct {
	subs        map[string]chan Message                  // Map topic to their message channel
	subscribers map[string]map[string]context.CancelFunc // Map topic -> subscriberID -> cancel function
	mu          sync.RWMutex
	closed      bool
}

// NewChannelMessageBus creates a new in-memory message bus.
func NewChannelMessageBus() *ChannelMessageBus {
	return &ChannelMessageBus{
		subs:        make(map[string]chan Message),
		subscribers: make(map[string]map[string]context.CancelFunc),
	}
}

// Publish sends a message to all subscribers of the topic.
func (b *ChannelMessageBus) Publish(ctx context.Context, topic string, payload any, metadata map[string]any) error {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return errors.New("message bus is closed")
	}
	subChan, exists := b.subs[topic]
	b.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no subscribers for topic %s", topic)
	}

	msg := Message{
		ID:       fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		Topic:    topic,
		Payload:  payload,
		Metadata: metadata,
	}

	select {
	case subChan <- msg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout publishing message to topic %s", topic)
	}
}

// Subscribe creates a subscription for the topic.
func (b *ChannelMessageBus) Subscribe(ctx context.Context, topic string, handler HandlerFunc) (string, error) {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return "", errors.New("message bus is closed")
	}

	if _, exists := b.subs[topic]; !exists {
		b.subs[topic] = make(chan Message, 100)
	}
	subChan := b.subs[topic]
	subID := fmt.Sprintf("sub-%d", time.Now().UnixNano())

	// Create a cancellable context for this subscriber to allow cleanup
	subCtx, cancel := context.WithCancel(ctx)

	// Track the subscriber for cleanup
	if b.subscribers[topic] == nil {
		b.subscribers[topic] = make(map[string]context.CancelFunc)
	}
	b.subscribers[topic][subID] = cancel

	b.mu.Unlock()

	// Start a goroutine to process messages for this subscriber
	go func() {
		defer func() {
			// Ensure goroutine exits cleanly
		}()
		for {
			select {
			case msg, ok := <-subChan:
				if !ok {
					// Channel closed, exit goroutine
					return
				}
				// Check context before processing message
				select {
				case <-subCtx.Done():
					return
				default:
					_ = handler(subCtx, msg)
				}
			case <-subCtx.Done():
				return
			}
		}
	}()

	return subID, nil
}

// Unsubscribe removes a subscription and cancels its goroutine.
func (b *ChannelMessageBus) Unsubscribe(ctx context.Context, topic, subscriberID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return errors.New("message bus is closed")
	}

	// Cancel the subscriber's context to stop the goroutine
	if topicSubscribers, ok := b.subscribers[topic]; ok {
		if cancel, exists := topicSubscribers[subscriberID]; exists {
			cancel() // This will cause the goroutine to exit
			delete(topicSubscribers, subscriberID)

			// Clean up empty topic entries
			if len(topicSubscribers) == 0 {
				delete(b.subscribers, topic)
				// Close and remove the channel if no more subscribers
				if ch, exists := b.subs[topic]; exists {
					close(ch)
					delete(b.subs, topic)
				}
			}
			return nil
		}
	}

	return fmt.Errorf("subscriber %s not found for topic %s", subscriberID, topic)
}

// Start is a no-op.
func (b *ChannelMessageBus) Start(ctx context.Context) error {
	return nil
}

// Stop closes the message bus.
func (b *ChannelMessageBus) Stop(ctx context.Context) error {
	return b.Close()
}

// GetName returns the bus name.
func (b *ChannelMessageBus) GetName() string {
	return "channel"
}

// Close shuts down the bus and closes all channels.
func (b *ChannelMessageBus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return errors.New("message bus already closed")
	}

	b.closed = true

	// Cancel all subscriber contexts to stop goroutines
	for _, topicSubscribers := range b.subscribers {
		for _, cancel := range topicSubscribers {
			cancel()
		}
	}
	b.subscribers = make(map[string]map[string]context.CancelFunc)

	// Close all channels
	for id, ch := range b.subs {
		delete(b.subs, id)
		close(ch)
	}
	return nil
}

// Ensure ChannelMessageBus implements the interface.
var _ MessageBus = (*ChannelMessageBus)(nil)

// TODO:
// - Define standard message content types (e.g., TaskDelegation, InformationShare, StatusUpdate).
// - Integrate MessageBus with AgentExecutor or a higher-level orchestrator (like CrewAI's Crew).
// - Consider error handling for Send (e.g., retries, dead-letter queues).
// - Explore other bus implementations (e.g., NATS, Redis Pub/Sub).
