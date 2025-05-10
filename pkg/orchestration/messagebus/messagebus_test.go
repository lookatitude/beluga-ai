package messagebus

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInMemoryMessageBus(t *testing.T) {
	bus := NewInMemoryMessageBus()
	require.NotNil(t, bus, "NewInMemoryMessageBus should not return nil")
	assert.NotNil(t, bus.subscribers, "Subscribers map should be initialized")
	assert.Equal(t, "inmemory", bus.GetName(), "Bus name should be 'inmemory'")
	assert.NotNil(t, bus.stopChan, "stopChan should be initialized")
}

func TestInMemoryMessageBus_PublishAndSubscribe(t *testing.T) {
	bus := NewInMemoryMessageBus()
	ctx := context.Background()
	topic := "test.topic"
	payload := "test_payload"
	var receivedPayload interface{}
	var handlerCalled bool
	var wg sync.WaitGroup

	handler := func(ctx context.Context, msg Message) error {
		receivedPayload = msg.Payload
		handlerCalled = true
		wg.Done()
		return nil
	}

	wg.Add(1)
	subID, err := bus.Subscribe(ctx, topic, handler)
	require.NoError(t, err, "Subscribe should not return an error")
	require.NotEmpty(t, subID, "Subscriber ID should not be empty")

	err = bus.Publish(ctx, topic, payload, nil)
	assert.NoError(t, err, "Publish should not return an error")

	// Wait for handler to be called (with a timeout)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Handler was called
	case <-time.After(1 * time.Second):
		t.Fatal("Handler was not called within timeout")
	}

	assert.True(t, handlerCalled, "Handler should have been called")
	assert.Equal(t, payload, receivedPayload, "Received payload mismatch")
}

func TestInMemoryMessageBus_MultipleSubscribers(t *testing.T) {
	bus := NewInMemoryMessageBus()
	ctx := context.Background()
	topic := "multi.sub.topic"
	payload := "multi_payload"
	var handler1Called, handler2Called bool
	var wg sync.WaitGroup

	handler1 := func(ctx context.Context, msg Message) error {
		assert.Equal(t, payload, msg.Payload)
		handler1Called = true
		wg.Done()
		return nil
	}

	handler2 := func(ctx context.Context, msg Message) error {
		assert.Equal(t, payload, msg.Payload)
		handler2Called = true
		wg.Done()
		return nil
	}

	wg.Add(2)
	_, err := bus.Subscribe(ctx, topic, handler1)
	require.NoError(t, err)
	_, err = bus.Subscribe(ctx, topic, handler2)
	require.NoError(t, err)

	err = bus.Publish(ctx, topic, payload, nil)
	assert.NoError(t, err)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("Handlers were not called within timeout")
	}

	assert.True(t, handler1Called, "Handler 1 should have been called")
	assert.True(t, handler2Called, "Handler 2 should have been called")
}

func TestInMemoryMessageBus_Unsubscribe(t *testing.T) {
	bus := NewInMemoryMessageBus()
	ctx := context.Background()
	topic := "unsubscribe.topic"
	payload := "unsubscribe_payload"
	var handlerCalled bool
	var wg sync.WaitGroup

	handler := func(ctx context.Context, msg Message) error {
		handlerCalled = true // This should not be called after unsubscribe
		wg.Done()
		return nil
	}

	subID, err := bus.Subscribe(ctx, topic, handler)
	require.NoError(t, err)

	err = bus.Unsubscribe(ctx, topic, subID)
	assert.NoError(t, err, "Unsubscribe should not return an error")

	// Ensure subscriber map is cleaned up if it becomes empty
	bus.mu.RLock()
	assert.Nil(t, bus.subscribers[topic], "Topic should be removed from subscribers map if no subscribers are left")
	bus.mu.RUnlock()

	err = bus.Publish(ctx, topic, payload, nil)
	assert.NoError(t, err)

	// Give some time for the goroutine (if any) to potentially execute
	time.Sleep(100 * time.Millisecond)

	assert.False(t, handlerCalled, "Handler should not have been called after unsubscribe")

	// Test unsubscribe non-existent ID
	err = bus.Unsubscribe(ctx, topic, "non-existent-sub-id")
	assert.NoError(t, err, "Unsubscribing a non-existent ID should not error")

	// Test unsubscribe from non-existent topic
	err = bus.Unsubscribe(ctx, "non-existent-topic", subID)
	assert.NoError(t, err, "Unsubscribing from a non-existent topic should not error")
}

func TestInMemoryMessageBus_PublishToTopicWithNoSubscribers(t *testing.T) {
	bus := NewInMemoryMessageBus()
	ctx := context.Background()
	topic := "no.subscribers.topic"
	payload := "no_sub_payload"

	err := bus.Publish(ctx, topic, payload, nil)
	assert.NoError(t, err, "Publish to a topic with no subscribers should not error")
}

func TestInMemoryMessageBus_StartAndStop(t *testing.T) {
	bus := NewInMemoryMessageBus()
	ctx := context.Background()

	err := bus.Start(ctx)
	assert.NoError(t, err, "Start should not return an error for InMemoryMessageBus")

	err = bus.Stop(ctx)
	assert.NoError(t, err, "Stop should not return an error for InMemoryMessageBus")

	// Check if stopChan is closed
	select {
	case <-bus.stopChan:
		// Expected, channel is closed
	default:
		t.Error("stopChan should be closed after Stop() is called")
	}
}

func TestInMemoryMessageBus_GetName(t *testing.T) {
	bus := NewInMemoryMessageBus()
	assert.Equal(t, "inmemory", bus.GetName(), "GetName should return 'inmemory'")
}

func TestInMemoryMessageBus_MessageIDGeneration(t *testing.T) {
	bus := NewInMemoryMessageBus()
	ctx := context.Background()
	topic := "id.test.topic"
	var firstMsgID string
	var wg sync.WaitGroup

	handler := func(ctx context.Context, msg Message) error {
		assert.NotEmpty(t, msg.ID, "Message ID should not be empty")
		if firstMsgID == "" {
			firstMsgID = msg.ID
		} else {
			assert.NotEqual(t, firstMsgID, msg.ID, "Message IDs should be unique for subsequent publishes")
		}
		wg.Done()
		return nil
	}

	wg.Add(2)
	_, err := bus.Subscribe(ctx, topic, handler)
	require.NoError(t, err)

	err = bus.Publish(ctx, topic, "payload1", nil)
	require.NoError(t, err)

	err = bus.Publish(ctx, topic, "payload2", nil)
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("Handlers were not called within timeout for ID test")
	}
}

