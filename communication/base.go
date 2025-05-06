// Package communication defines interfaces and mechanisms for inter-agent communication.
package communication

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time" // Added missing import
)

// Message represents a unit of communication between agents.
type Message struct {
	ID        string         // Unique message identifier
	Sender    string         // Identifier of the sending agent
	Recipient string         // Identifier of the receiving agent (can be broadcast ID)
	Content   any            // Payload of the message (e.g., string, structured data)
	Metadata  map[string]any // Additional context (timestamps, type, etc.)
}

// MessageBus defines an interface for sending and receiving messages between agents.
// Implementations could use channels, network protocols, message queues, etc.
type MessageBus interface {
	// Send transmits a message.
	Send(ctx context.Context, msg Message) error
	// Receive waits for and returns the next message intended for a specific recipient.
	// Implementations might filter based on recipient ID.
	Receive(ctx context.Context, recipientID string) (Message, error)
	// Subscribe allows an agent to listen for messages on a specific topic or for itself.
	// Returns a channel for receiving messages and an error.
	Subscribe(ctx context.Context, recipientID string) (<-chan Message, error)
	// Unsubscribe stops listening for messages for a recipient.
	Unsubscribe(ctx context.Context, recipientID string) error
	// Close shuts down the message bus.
	Close() error
}

// --- In-Memory Channel-Based Message Bus ---

// ChannelMessageBus implements MessageBus using Go channels for local communication.
type ChannelMessageBus struct {
	subs   map[string]chan Message // Map recipientID to their message channel
	mu     sync.RWMutex
	closed bool
}

// NewChannelMessageBus creates a new in-memory message bus.
func NewChannelMessageBus() *ChannelMessageBus {
	return &ChannelMessageBus{
		subs: make(map[string]chan Message),
	}
}

// Send broadcasts the message to the recipient's channel if subscribed.
func (b *ChannelMessageBus) Send(ctx context.Context, msg Message) error {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return errors.New("message bus is closed")
	}
	subChan, exists := b.subs[msg.Recipient]
	b.mu.RUnlock()

	if !exists {
		// Optionally handle undeliverable messages (e.g., log, error, dead-letter queue)
		return fmt.Errorf("recipient 	%s	 not subscribed or does not exist", msg.Recipient)
	}

	select {
	case subChan <- msg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second): // Add a timeout to prevent indefinite blocking if receiver is stuck
		return fmt.Errorf("timeout sending message to recipient 	%s	", msg.Recipient)
	}
}

// Receive is not the primary way to use this bus; Subscribe is preferred.
// This implementation provides a basic single-message receive.
func (b *ChannelMessageBus) Receive(ctx context.Context, recipientID string) (Message, error) {
	subChan, err := b.getOrCreateChannel(recipientID)
	if err != nil {
		// This case shouldn't happen with getOrCreateChannel, but handle defensively
		return Message{}, err
	}

	select {
	case msg, ok := <-subChan:
		if !ok {
			return Message{}, errors.New("message bus closed or subscription ended")
		}
		return msg, nil
	case <-ctx.Done():
		return Message{}, ctx.Err()
	}
}

// getOrCreateChannel retrieves or creates the channel for a recipient.
// This is mainly for the basic Receive method; Subscribe manages its own channels.
func (b *ChannelMessageBus) getOrCreateChannel(recipientID string) (chan Message, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil, errors.New("message bus is closed")
	}
	subChan, exists := b.subs[recipientID]
	if !exists {
		// Create a buffered channel to avoid blocking Send if Receive isn't immediate
		subChan = make(chan Message, 10) // Buffer size 10
		b.subs[recipientID] = subChan
	}
	return subChan, nil
}

// Subscribe creates a dedicated channel for the recipient.
func (b *ChannelMessageBus) Subscribe(ctx context.Context, recipientID string) (<-chan Message, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil, errors.New("message bus is closed")
	}
	if _, exists := b.subs[recipientID]; exists {
		return nil, fmt.Errorf("recipient 	%s	 is already subscribed", recipientID)
	}

	// Create a buffered channel for the subscription
	subChan := make(chan Message, 100) // Larger buffer for subscribers
	b.subs[recipientID] = subChan
	return subChan, nil
}

// Unsubscribe removes the recipient's channel.
func (b *ChannelMessageBus) Unsubscribe(ctx context.Context, recipientID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return errors.New("message bus is closed")
	}

	subChan, exists := b.subs[recipientID]
	if exists {
		delete(b.subs, recipientID)
		close(subChan) // Close the channel to signal subscribers
	}
	return nil
}

// Close shuts down the bus and closes all subscription channels.
func (b *ChannelMessageBus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return errors.New("message bus already closed")
	}

	b.closed = true
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
