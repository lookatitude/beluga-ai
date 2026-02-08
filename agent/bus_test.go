package agent

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNewInMemoryBus(t *testing.T) {
	bus := NewInMemoryBus()
	if bus == nil {
		t.Fatal("expected non-nil bus")
	}
	if bus.subscribers == nil {
		t.Fatal("expected non-nil subscribers map")
	}
}

func TestInMemoryBus_PublishSubscribe(t *testing.T) {
	bus := NewInMemoryBus()
	ctx := context.Background()

	var received []AgentEvent
	var mu sync.Mutex

	sub, err := bus.Subscribe(ctx, "test-topic", func(event AgentEvent) {
		mu.Lock()
		received = append(received, event)
		mu.Unlock()
	})
	if err != nil {
		t.Fatalf("Subscribe error: %v", err)
	}
	if sub == nil {
		t.Fatal("expected non-nil subscription")
	}

	event := AgentEvent{
		Type:     "test",
		SourceID: "agent-1",
		Payload:  "hello",
	}
	if err := bus.Publish(ctx, "test-topic", event); err != nil {
		t.Fatalf("Publish error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("expected 1 event, got %d", len(received))
	}
	if received[0].Type != "test" {
		t.Errorf("event type = %q, want %q", received[0].Type, "test")
	}
	if received[0].SourceID != "agent-1" {
		t.Errorf("source = %q, want %q", received[0].SourceID, "agent-1")
	}
	if received[0].Payload != "hello" {
		t.Errorf("payload = %v, want %q", received[0].Payload, "hello")
	}
}

func TestInMemoryBus_Publish_SetsTimestamp(t *testing.T) {
	bus := NewInMemoryBus()
	ctx := context.Background()

	var received AgentEvent
	_, _ = bus.Subscribe(ctx, "topic", func(event AgentEvent) {
		received = event
	})

	before := time.Now()
	_ = bus.Publish(ctx, "topic", AgentEvent{Type: "test"})
	after := time.Now()

	if received.Timestamp.Before(before) || received.Timestamp.After(after) {
		t.Errorf("timestamp %v not between %v and %v", received.Timestamp, before, after)
	}
}

func TestInMemoryBus_Publish_PreservesExistingTimestamp(t *testing.T) {
	bus := NewInMemoryBus()
	ctx := context.Background()

	var received AgentEvent
	_, _ = bus.Subscribe(ctx, "topic", func(event AgentEvent) {
		received = event
	})

	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	_ = bus.Publish(ctx, "topic", AgentEvent{Type: "test", Timestamp: ts})

	if !received.Timestamp.Equal(ts) {
		t.Errorf("timestamp = %v, want %v", received.Timestamp, ts)
	}
}

func TestInMemoryBus_Publish_CancelledContext(t *testing.T) {
	bus := NewInMemoryBus()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := bus.Publish(ctx, "topic", AgentEvent{Type: "test"})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestInMemoryBus_Subscribe_CancelledContext(t *testing.T) {
	bus := NewInMemoryBus()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := bus.Subscribe(ctx, "topic", func(event AgentEvent) {})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestInMemoryBus_Publish_NoSubscribers(t *testing.T) {
	bus := NewInMemoryBus()
	ctx := context.Background()

	err := bus.Publish(ctx, "empty-topic", AgentEvent{Type: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInMemoryBus_MultipleSubscribers(t *testing.T) {
	bus := NewInMemoryBus()
	ctx := context.Background()

	count1 := 0
	count2 := 0

	_, _ = bus.Subscribe(ctx, "topic", func(event AgentEvent) {
		count1++
	})
	_, _ = bus.Subscribe(ctx, "topic", func(event AgentEvent) {
		count2++
	})

	_ = bus.Publish(ctx, "topic", AgentEvent{Type: "test"})

	if count1 != 1 {
		t.Errorf("subscriber 1 count = %d, want 1", count1)
	}
	if count2 != 1 {
		t.Errorf("subscriber 2 count = %d, want 1", count2)
	}
}

func TestInMemoryBus_DifferentTopics(t *testing.T) {
	bus := NewInMemoryBus()
	ctx := context.Background()

	topicACalls := 0
	topicBCalls := 0

	_, _ = bus.Subscribe(ctx, "topic-a", func(event AgentEvent) {
		topicACalls++
	})
	_, _ = bus.Subscribe(ctx, "topic-b", func(event AgentEvent) {
		topicBCalls++
	})

	_ = bus.Publish(ctx, "topic-a", AgentEvent{Type: "test"})

	if topicACalls != 1 {
		t.Errorf("topic-a calls = %d, want 1", topicACalls)
	}
	if topicBCalls != 0 {
		t.Errorf("topic-b calls = %d, want 0", topicBCalls)
	}
}

func TestInMemoryBus_Unsubscribe(t *testing.T) {
	bus := NewInMemoryBus()
	ctx := context.Background()

	calls := 0
	sub, _ := bus.Subscribe(ctx, "topic", func(event AgentEvent) {
		calls++
	})

	_ = bus.Publish(ctx, "topic", AgentEvent{Type: "before"})
	if calls != 1 {
		t.Fatalf("expected 1 call before unsubscribe, got %d", calls)
	}

	if err := sub.Unsubscribe(); err != nil {
		t.Fatalf("Unsubscribe error: %v", err)
	}

	_ = bus.Publish(ctx, "topic", AgentEvent{Type: "after"})
	if calls != 1 {
		t.Errorf("expected 1 call after unsubscribe, got %d", calls)
	}
}

func TestInMemoryBus_Unsubscribe_OnlyAffectsTarget(t *testing.T) {
	bus := NewInMemoryBus()
	ctx := context.Background()

	calls1 := 0
	calls2 := 0

	sub1, _ := bus.Subscribe(ctx, "topic", func(event AgentEvent) {
		calls1++
	})
	_, _ = bus.Subscribe(ctx, "topic", func(event AgentEvent) {
		calls2++
	})

	_ = sub1.Unsubscribe()

	_ = bus.Publish(ctx, "topic", AgentEvent{Type: "test"})

	if calls1 != 0 {
		t.Errorf("sub1 calls = %d, want 0 (unsubscribed)", calls1)
	}
	if calls2 != 1 {
		t.Errorf("sub2 calls = %d, want 1", calls2)
	}
}

func TestInMemoryBus_ImplementsEventBus(t *testing.T) {
	var _ EventBus = (*InMemoryBus)(nil)
}
