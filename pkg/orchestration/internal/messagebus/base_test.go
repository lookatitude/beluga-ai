package messagebus

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
func TestNewChannelMessageBus(t *testing.T) {
	bus := NewChannelMessageBus()

	assert.NotNil(t, bus)
	assert.NotNil(t, bus.subs)
	assert.False(t, bus.closed)
	assert.Equal(t, "channel", bus.GetName())
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
}

func TestChannelMessageBus_Publish_NoSubscribers(t *testing.T) {
	bus := NewChannelMessageBus()
	ctx := context.Background()

	err := bus.Publish(ctx, "test.topic", "test_payload", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no subscribers for topic test.topic")
}

func TestChannelMessageBus_Publish_Success(t *testing.T) {
	bus := NewChannelMessageBus()
	ctx := context.Background()

	// Subscribe first
	subID, err := bus.Subscribe(ctx, "test.topic", func(ctx context.Context, msg Message) error {
		return nil
	})
	require.NoError(t, err)
	assert.NotEmpty(t, subID)

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	// Now publish
	err = bus.Publish(ctx, "test.topic", "test_payload", map[string]interface{}{"key": "value"})

	assert.NoError(t, err)
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
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

	// Try to publish after close
	err = bus.Publish(ctx, "test.topic", "test_payload", nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message bus is closed")
}

func TestChannelMessageBus_Publish_Timeout(t *testing.T) {
	bus := NewChannelMessageBus()
	ctx := context.Background()

	// Subscribe with a handler
	_, err := bus.Subscribe(ctx, "test.topic", func(ctx context.Context, msg Message) error {
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
	for i := 0; i < 100; i++ {
		select {
		case ch <- Message{ID: fmt.Sprintf("dummy-%d", i), Topic: "test.topic", Payload: "dummy"}:
		default:
			// Channel might be getting consumed, try a few more times
			if i < 50 {
				time.Sleep(1 * time.Millisecond)
				continue
			}
		}
	}

	// Create a very short context that's already expired
	shortCtx, cancel := context.WithCancel(ctx)
	cancel() // Cancel immediately
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

	// This should fail because context is already cancelled
	err = bus.Publish(shortCtx, "test.topic", "test_payload", nil)

	assert.Error(t, err)
	// Should get context cancelled error
	assert.True(t, err == context.Canceled || strings.Contains(err.Error(), "canceled") || strings.Contains(err.Error(), "context canceled"),
		"Expected context canceled error, got: %v", err)
}

func TestChannelMessageBus_Subscribe_Success(t *testing.T) {
	bus := NewChannelMessageBus()
	ctx := context.Background()

	handler := func(ctx context.Context, msg Message) error {
		return nil
	}

	subID, err := bus.Subscribe(ctx, "test.topic", handler)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

	assert.NoError(t, err)
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
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

	// Close the bus first
	err := bus.Close()
	require.NoError(t, err)

	// Try to subscribe after close
	_, err = bus.Subscribe(ctx, "test.topic", func(ctx context.Context, msg Message) error {
		return nil
	})

	assert.Error(t, err)
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
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

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
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

	// Subscribe first
	subID, err := bus.Subscribe(ctx, "test.topic", func(ctx context.Context, msg Message) error {
		return nil
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	// Try to unsubscribe
	err = bus.Unsubscribe(ctx, "test.topic", subID)

	assert.NoError(t, err)

	// Try to unsubscribe again (should fail)
	err = bus.Unsubscribe(ctx, "test.topic", subID)
	assert.Error(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	assert.Contains(t, err.Error(), "subscriber")
}

func TestChannelMessageBus_Start(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	bus := NewChannelMessageBus()
	ctx := context.Background()

	err := bus.Start(ctx)

	assert.NoError(t, err) // Should be a no-op
}

func TestChannelMessageBus_Stop(t *testing.T) {
	bus := NewChannelMessageBus()
	ctx := context.Background()

	err := bus.Stop(ctx)

	assert.NoError(t, err)
	assert.True(t, bus.closed)
}

func TestChannelMessageBus_GetName(t *testing.T) {
	bus := NewChannelMessageBus()

	assert.Equal(t, "channel", bus.GetName())
}

func TestChannelMessageBus_Close(t *testing.T) {
	bus := NewChannelMessageBus()
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	ctx := context.Background()

	// Subscribe to create some channels
	_, err := bus.Subscribe(ctx, "topic1", func(ctx context.Context, msg Message) error {
		return nil
	})
	require.NoError(t, err)

	_, err = bus.Subscribe(ctx, "topic2", func(ctx context.Context, msg Message) error {
		return nil
	})
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

	// Close the bus
	err = bus.Close()

	assert.NoError(t, err)
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
	assert.NoError(t, err)

	// Try to close again
	err = bus.Close()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message bus already closed")
}

func TestChannelMessageBus_ConcurrentOperations(t *testing.T) {
	bus := NewChannelMessageBus()
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	var wg sync.WaitGroup

	// Start multiple goroutines publishing
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				bus.Publish(ctx, "concurrent.topic", "payload", nil)
			}
		}(i)
	}

	// Subscribe concurrently
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			bus.Subscribe(ctx, "concurrent.topic", func(ctx context.Context, msg Message) error {
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
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	ctx := context.Background()

	var receivedMsg Message
	var msgReceived bool

	// Subscribe with a handler that captures the message
	_, err := bus.Subscribe(ctx, "test.topic", func(ctx context.Context, msg Message) error {
		receivedMsg = msg
		msgReceived = true
		return nil
	})
	require.NoError(t, err)

	payload := map[string]interface{}{"data": "test"}
	metadata := map[string]interface{}{"source": "test", "priority": "high"}

	// Publish a message
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	err = bus.Publish(ctx, "test.topic", payload, metadata)
	require.NoError(t, err)

	// Give some time for the message to be processed
	time.Sleep(10 * time.Millisecond)

	// Verify message structure
	assert.True(t, msgReceived)
	assert.NotEmpty(t, receivedMsg.ID)
	assert.Equal(t, "test.topic", receivedMsg.Topic)
	assert.Equal(t, payload, receivedMsg.Payload)
	assert.Equal(t, metadata, receivedMsg.Metadata)
	assert.Contains(t, receivedMsg.ID, "msg-")
}

func TestChannelMessageBus_BufferSize(t *testing.T) {
	bus := NewChannelMessageBus()
	ctx := context.Background()

	// Subscribe to create a channel
	_, err := bus.Subscribe(ctx, "buffer.topic", func(ctx context.Context, msg Message) error {
		time.Sleep(100 * time.Millisecond) // Slow handler to test buffering
		return nil
	})
	require.NoError(t, err)

	bus.mu.RLock()
	ch := bus.subs["buffer.topic"]
	bus.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
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
	assert.Equal(t, 1, receivedCount)
}

func TestChannelMessageBus_TopicIsolation(t *testing.T) {
	bus := NewChannelMessageBus()
	ctx := context.Background()

	var topic1Received, topic2Received bool

	// Subscribe to different topics
	_, err := bus.Subscribe(ctx, "topic1", func(ctx context.Context, msg Message) error {
		topic1Received = true
		return nil
	})
	require.NoError(t, err)

	_, err = bus.Subscribe(ctx, "topic2", func(ctx context.Context, msg Message) error {
		topic2Received = true
		return nil
	})
	require.NoError(t, err)

	// Publish to topic1 only
	err = bus.Publish(ctx, "topic1", "test", nil)
	require.NoError(t, err)

	// Give time for processing
	time.Sleep(10 * time.Millisecond)

	// Only topic1 should have received the message
	assert.True(t, topic1Received)
	assert.False(t, topic2Received)
}
