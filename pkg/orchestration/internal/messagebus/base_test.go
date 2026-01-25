package messagebus

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChannelMessageBus(t *testing.T) {
	bus := NewChannelMessageBus()

	assert.NotNil(t, bus)
	assert.NotNil(t, bus.subs)
	assert.False(t, bus.closed)
	assert.Equal(t, "channel", bus.GetName())
}

func TestChannelMessageBus_Publish_NoSubscribers(t *testing.T) {
	bus := NewChannelMessageBus()

	ctx := context.Background()
	err := bus.Publish(ctx, "test.topic", "test_payload", nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no subscribers for topic test.topic")
}

func TestChannelMessageBus_Publish_Success(t *testing.T) {
	ctx := context.Background()
	bus := NewChannelMessageBus()

	// Subscribe first
	subID, err := bus.Subscribe(ctx, "test.topic", func(ctx context.Context, msg Message) error {
		return nil
	})
	require.NoError(t, err)
	assert.NotEmpty(t, subID)
	// Now publish
	err = bus.Publish(ctx, "test.topic", "test_payload", map[string]any{"key": "value"})

	require.NoError(t, err)
}

func TestChannelMessageBus_Publish_AfterClose(t *testing.T) {
	bus := NewChannelMessageBus()

	ctx := context.Background()
	// Subscribe first
	_, err := bus.Subscribe(ctx, "test.topic", func(ctx context.Context, msg Message) error {
		return nil
	})
	require.NoError(t, err)

	// Close the bus
	err = bus.Close()
	require.NoError(t, err)

	// Try to publish after close
	err = bus.Publish(ctx, "test.topic", "test_payload", nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "message bus is closed")
}

func TestChannelMessageBus_Publish_Timeout(t *testing.T) {
	bus := NewChannelMessageBus()

	ctx := context.Background()
	handlerBlock := make(chan struct{})

	// Subscribe with a handler that blocks until we signal
	_, err := bus.Subscribe(ctx, "test.topic", func(ctx context.Context, msg Message) error {
		<-handlerBlock // Block until signaled
		return nil
	})
	require.NoError(t, err)

	// Get the channel
	bus.mu.RLock()
	ch, exists := bus.subs["test.topic"]
	bus.mu.RUnlock()
	if !exists || ch == nil {
		t.Fatal("channel not found or nil")
	}

	// Fill the channel buffer completely to cause blocking
	// The buffer is 100, so we need to fill it
	filled := 0
FillLoop:
	for i := 0; i < 150; i++ { // Try more than buffer size
		select {
		case ch <- Message{ID: fmt.Sprintf("dummy-%d", i), Topic: "test.topic", Payload: "dummy"}:
			filled++
		default:
			// Buffer is full
			break FillLoop
		}
	}
	// Verify the buffer is actually full (default case was hit)
	if filled == 150 {
		t.Skip("Buffer never filled - handler may be consuming messages")
	}

	// Create a very short context that's already expired
	shortCtx, cancel := context.WithCancel(ctx)
	cancel() // Cancel immediately

	// This should fail because context is already canceled and buffer is full
	err = bus.Publish(shortCtx, "test.topic", "test_payload", nil)

	// Cleanup: unblock the handler
	close(handlerBlock)

	require.Error(t, err)
	// Should get context canceled error
	assert.True(t, errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "canceled") || strings.Contains(err.Error(), "context canceled"),
		"Expected context canceled error, got: %v", err)
}

func TestChannelMessageBus_Subscribe_Success(t *testing.T) {
	bus := NewChannelMessageBus()

	ctx := context.Background()
	handler := func(ctx context.Context, msg Message) error {
		return nil
	}

	subID, err := bus.Subscribe(ctx, "test.topic", handler)

	require.NoError(t, err)
	assert.NotEmpty(t, subID)
	assert.Contains(t, subID, "sub-")

	// Verify channel was created
	bus.mu.RLock()
	_, exists := bus.subs["test.topic"]
	bus.mu.RUnlock()
	assert.True(t, exists)
}

func TestChannelMessageBus_Subscribe_AfterClose(t *testing.T) {
	bus := NewChannelMessageBus()

	// Close the bus first
	err := bus.Close()
	require.NoError(t, err)

	ctx := context.Background()
	// Try to subscribe after close
	_, err = bus.Subscribe(ctx, "test.topic", func(ctx context.Context, msg Message) error {
		return nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "message bus is closed")
}

func TestChannelMessageBus_Subscribe_MultipleTopics(t *testing.T) {
	bus := NewChannelMessageBus()

	ctx := context.Background()
	// Subscribe to multiple topics
	_, err := bus.Subscribe(ctx, "topic1", func(ctx context.Context, msg Message) error {
		return nil
	})
	require.NoError(t, err)

	_, err = bus.Subscribe(ctx, "topic2", func(ctx context.Context, msg Message) error {
		return nil
	})
	require.NoError(t, err)

	// Verify both topics have channels
	bus.mu.RLock()
	_, exists1 := bus.subs["topic1"]
	_, exists2 := bus.subs["topic2"]
	bus.mu.RUnlock()

	assert.True(t, exists1)
	assert.True(t, exists2)
}

func TestChannelMessageBus_Unsubscribe(t *testing.T) {
	bus := NewChannelMessageBus()

	ctx := context.Background()
	// Subscribe first
	subID, err := bus.Subscribe(ctx, "test.topic", func(ctx context.Context, msg Message) error {
		return nil
	})
	require.NoError(t, err)

	// Try to unsubscribe
	err = bus.Unsubscribe(ctx, "test.topic", subID)

	require.NoError(t, err)

	// Try to unsubscribe again (should fail)
	err = bus.Unsubscribe(ctx, "test.topic", subID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "subscriber")
}

func TestChannelMessageBus_Start(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	bus := NewChannelMessageBus()

	err := bus.Start(ctx)

	require.NoError(t, err) // Should be a no-op
}

func TestChannelMessageBus_Stop(t *testing.T) {
	bus := NewChannelMessageBus()

	ctx := context.Background()
	err := bus.Stop(ctx)

	require.NoError(t, err)
	assert.True(t, bus.closed)
}

func TestChannelMessageBus_GetName(t *testing.T) {
	bus := NewChannelMessageBus()

	assert.Equal(t, "channel", bus.GetName())
}

func TestChannelMessageBus_Close(t *testing.T) {
	bus := NewChannelMessageBus()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Subscribe to create some channels
	_, err := bus.Subscribe(ctx, "topic1", func(ctx context.Context, msg Message) error {
		return nil
	})
	require.NoError(t, err)

	_, err = bus.Subscribe(ctx, "topic2", func(ctx context.Context, msg Message) error {
		return nil
	})
	require.NoError(t, err)

	// Close the bus
	err = bus.Close()

	require.NoError(t, err)
	assert.True(t, bus.closed)

	// Verify channels are closed
	bus.mu.RLock()
	assert.Nil(t, bus.subs["topic1"])
	assert.Nil(t, bus.subs["topic2"])
	bus.mu.RUnlock()
}

func TestChannelMessageBus_Close_AlreadyClosed(t *testing.T) {
	bus := NewChannelMessageBus()

	// Close once
	err := bus.Close()
	require.NoError(t, err)

	// Try to close again
	err = bus.Close()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "message bus already closed")
}

func TestChannelMessageBus_ConcurrentOperations(t *testing.T) {
	bus := NewChannelMessageBus()
	ctx := context.Background()
	var wg sync.WaitGroup

	// Start multiple goroutines publishing
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_ = bus.Publish(ctx, "concurrent.topic", "payload", nil)
			}
		}(i)
	}

	// Subscribe concurrently
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, _ = bus.Subscribe(ctx, "concurrent.topic", func(ctx context.Context, msg Message) error {
				return nil
			})
		}(i)
	}

	wg.Wait()

	// Bus should still be functional
	assert.False(t, bus.closed)
}

func TestChannelMessageBus_MessageStructure(t *testing.T) {
	bus := NewChannelMessageBus()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var mu sync.Mutex
	var receivedMsg Message
	var msgReceived bool

	// Subscribe with a handler that captures the message
	_, err := bus.Subscribe(ctx, "test.topic", func(ctx context.Context, msg Message) error {
		mu.Lock()
		receivedMsg = msg
		msgReceived = true
		mu.Unlock()
		return nil
	})
	require.NoError(t, err)

	payload := map[string]any{"data": "test"}
	metadata := map[string]any{"source": "test", "priority": "high"}

	// Publish a message
	err = bus.Publish(ctx, "test.topic", payload, metadata)
	require.NoError(t, err)

	// Give some time for the message to be processed
	time.Sleep(10 * time.Millisecond)

	// Verify message structure
	mu.Lock()
	wasReceived := msgReceived
	msg := receivedMsg
	mu.Unlock()
	assert.True(t, wasReceived)
	assert.NotEmpty(t, msg.ID)
	assert.Equal(t, "test.topic", msg.Topic)
	assert.Equal(t, payload, msg.Payload)
	assert.Equal(t, metadata, msg.Metadata)
	assert.Contains(t, msg.ID, "msg-")
}

func TestChannelMessageBus_BufferSize(t *testing.T) {
	bus := NewChannelMessageBus()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Subscribe to create a channel
	_, err := bus.Subscribe(ctx, "buffer.topic", func(ctx context.Context, msg Message) error {
		time.Sleep(100 * time.Millisecond) // Slow handler to test buffering
		return nil
	})
	require.NoError(t, err)

	bus.mu.RLock()
	ch := bus.subs["buffer.topic"]
	bus.mu.RUnlock()

	// Check channel buffer size
	assert.Equal(t, 100, cap(ch))
}

func TestChannelMessageBus_MultipleSubscribers(t *testing.T) {
	bus := NewChannelMessageBus()

	ctx := context.Background()
	var receivedCount int
	var mu sync.Mutex

	handler := func(ctx context.Context, msg Message) error {
		mu.Lock()
		receivedCount++
		mu.Unlock()
		return nil
	}

	// Subscribe multiple times to the same topic
	for i := 0; i < 3; i++ {
		_, err := bus.Subscribe(ctx, "multi.topic", handler)
		require.NoError(t, err)
	}

	// Publish a message
	err := bus.Publish(ctx, "multi.topic", "test", nil)
	require.NoError(t, err)

	// Give time for processing
	time.Sleep(50 * time.Millisecond)

	// Since this implementation only supports one channel per topic,
	// only one message should be sent
	mu.Lock()
	count := receivedCount
	mu.Unlock()
	assert.Equal(t, 1, count)
}

func TestChannelMessageBus_TopicIsolation(t *testing.T) {
	bus := NewChannelMessageBus()

	ctx := context.Background()
	var mu sync.Mutex
	var topic1Received, topic2Received bool

	// Subscribe to different topics
	_, err := bus.Subscribe(ctx, "topic1", func(ctx context.Context, msg Message) error {
		mu.Lock()
		topic1Received = true
		mu.Unlock()
		return nil
	})
	require.NoError(t, err)

	_, err = bus.Subscribe(ctx, "topic2", func(ctx context.Context, msg Message) error {
		mu.Lock()
		topic2Received = true
		mu.Unlock()
		return nil
	})
	require.NoError(t, err)

	// Publish to topic1 only
	err = bus.Publish(ctx, "topic1", "test", nil)
	require.NoError(t, err)

	// Give time for processing
	time.Sleep(10 * time.Millisecond)

	// Only topic1 should have received the message
	mu.Lock()
	topic1 := topic1Received
	topic2 := topic2Received
	mu.Unlock()
	assert.True(t, topic1)
	assert.False(t, topic2)
}
