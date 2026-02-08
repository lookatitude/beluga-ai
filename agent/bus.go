package agent

import (
	"context"
	"sync"
	"time"
)

// AgentEvent is an event published through the event bus for inter-agent
// communication.
type AgentEvent struct {
	// Type identifies the kind of event.
	Type string
	// SourceID is the agent that published this event.
	SourceID string
	// Payload carries event-specific data.
	Payload any
	// Timestamp is when the event was created.
	Timestamp time.Time
}

// EventBus is the interface for agent-to-agent async messaging.
type EventBus interface {
	// Publish sends an event to the given topic.
	Publish(ctx context.Context, topic string, event AgentEvent) error
	// Subscribe registers a handler for events on the given topic.
	Subscribe(ctx context.Context, topic string, handler func(AgentEvent)) (Subscription, error)
}

// Subscription represents an active event subscription.
type Subscription interface {
	// Unsubscribe removes this subscription.
	Unsubscribe() error
}

// InMemoryBus is an in-process EventBus implementation using channels.
type InMemoryBus struct {
	mu          sync.RWMutex
	subscribers map[string][]subscriber
	nextID      int
}

type subscriber struct {
	id      int
	handler func(AgentEvent)
}

// NewInMemoryBus creates a new InMemoryBus.
func NewInMemoryBus() *InMemoryBus {
	return &InMemoryBus{
		subscribers: make(map[string][]subscriber),
	}
}

// Publish sends an event to all subscribers on the topic.
func (b *InMemoryBus) Publish(ctx context.Context, topic string, event AgentEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	b.mu.RLock()
	subs := make([]subscriber, len(b.subscribers[topic]))
	copy(subs, b.subscribers[topic])
	b.mu.RUnlock()

	for _, sub := range subs {
		sub.handler(event)
	}
	return nil
}

// Subscribe registers a handler for events on the given topic.
func (b *InMemoryBus) Subscribe(ctx context.Context, topic string, handler func(AgentEvent)) (Subscription, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.nextID++
	id := b.nextID

	b.subscribers[topic] = append(b.subscribers[topic], subscriber{
		id:      id,
		handler: handler,
	})

	return &inMemorySub{bus: b, topic: topic, id: id}, nil
}

type inMemorySub struct {
	bus   *InMemoryBus
	topic string
	id    int
}

func (s *inMemorySub) Unsubscribe() error {
	s.bus.mu.Lock()
	defer s.bus.mu.Unlock()

	subs := s.bus.subscribers[s.topic]
	for i, sub := range subs {
		if sub.id == s.id {
			s.bus.subscribers[s.topic] = append(subs[:i], subs[i+1:]...)
			break
		}
	}
	return nil
}
