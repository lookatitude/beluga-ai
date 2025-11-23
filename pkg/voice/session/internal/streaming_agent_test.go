package internal

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStreamingAgent(t *testing.T) {
	callback := func(ctx context.Context, transcript string) (string, error) {
		return "response", nil
	}

	sa := NewStreamingAgent(callback)
	assert.NotNil(t, sa)
	assert.False(t, sa.IsStreaming())
}

func TestNewStreamingAgent_NilCallback(t *testing.T) {
	sa := NewStreamingAgent(nil)
	assert.NotNil(t, sa)
	assert.False(t, sa.IsStreaming())
}

func TestStreamingAgent_StartStreaming_Success(t *testing.T) {
	called := false
	var receivedTranscript string
	callback := func(ctx context.Context, transcript string) (string, error) {
		called = true
		receivedTranscript = transcript
		return "Hello, how can I help?", nil
	}

	sa := NewStreamingAgent(callback)

	ctx := context.Background()
	ch, err := sa.StartStreaming(ctx, "Hello")
	require.NoError(t, err)
	require.NotNil(t, ch)
	assert.True(t, sa.IsStreaming())

	// Receive response
	select {
	case response := <-ch:
		assert.Equal(t, "Hello, how can I help?", response)
		assert.True(t, called)
		assert.Equal(t, "Hello", receivedTranscript)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for response")
	}

	// Streaming should be false after completion
	time.Sleep(100 * time.Millisecond)
	assert.False(t, sa.IsStreaming())
}

func TestStreamingAgent_StartStreaming_Error(t *testing.T) {
	expectedErr := errors.New("agent error")
	callback := func(ctx context.Context, transcript string) (string, error) {
		return "", expectedErr
	}

	sa := NewStreamingAgent(callback)

	ctx := context.Background()
	ch, err := sa.StartStreaming(ctx, "Hello")
	require.NoError(t, err)
	require.NotNil(t, ch)

	// Channel should close without sending response
	select {
	case response, ok := <-ch:
		if ok {
			t.Fatalf("Expected channel to close, got response: %s", response)
		}
		// Channel closed, which is expected
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for channel to close")
	}

	// Streaming should be false after error
	time.Sleep(100 * time.Millisecond)
	assert.False(t, sa.IsStreaming())
}

func TestStreamingAgent_StartStreaming_ContextCancellation(t *testing.T) {
	callback := func(ctx context.Context, transcript string) (string, error) {
		<-ctx.Done()
		return "", ctx.Err()
	}

	sa := NewStreamingAgent(callback)

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := sa.StartStreaming(ctx, "Hello")
	require.NoError(t, err)
	require.NotNil(t, ch)

	// Cancel context
	cancel()

	// Channel should close
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("Expected channel to close")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for channel to close")
	}

	// Streaming should be false after cancellation
	time.Sleep(100 * time.Millisecond)
	assert.False(t, sa.IsStreaming())
}

func TestStreamingAgent_StartStreaming_AlreadyActive(t *testing.T) {
	callback := func(ctx context.Context, transcript string) (string, error) {
		time.Sleep(100 * time.Millisecond)
		return "response", nil
	}

	sa := NewStreamingAgent(callback)

	ctx := context.Background()
	ch1, err := sa.StartStreaming(ctx, "First")
	require.NoError(t, err)
	require.NotNil(t, ch1)
	assert.True(t, sa.IsStreaming())

	// Try to start again while active
	ch2, err := sa.StartStreaming(ctx, "Second")
	assert.Error(t, err)
	assert.Nil(t, ch2)
	assert.Contains(t, err.Error(), "streaming already active")

	// Wait for first to complete
	<-ch1
	time.Sleep(100 * time.Millisecond)
	assert.False(t, sa.IsStreaming())
}

func TestStreamingAgent_StartStreaming_NoCallback(t *testing.T) {
	sa := NewStreamingAgent(nil)

	ctx := context.Background()
	ch, err := sa.StartStreaming(ctx, "Hello")
	assert.Error(t, err)
	assert.Nil(t, ch)
	assert.Contains(t, err.Error(), "agent callback not set")
}

func TestStreamingAgent_StopStreaming(t *testing.T) {
	callback := func(ctx context.Context, transcript string) (string, error) {
		time.Sleep(200 * time.Millisecond)
		return "response", nil
	}

	sa := NewStreamingAgent(callback)

	ctx := context.Background()
	ch, err := sa.StartStreaming(ctx, "Hello")
	require.NoError(t, err)
	require.NotNil(t, ch)
	assert.True(t, sa.IsStreaming())

	// Stop streaming
	sa.StopStreaming()
	assert.False(t, sa.IsStreaming())

	// Can start again after stop
	ch2, err := sa.StartStreaming(ctx, "Hello again")
	require.NoError(t, err)
	require.NotNil(t, ch2)
	assert.True(t, sa.IsStreaming())

	// Clean up
	<-ch2
}

func TestStreamingAgent_IsStreaming(t *testing.T) {
	callback := func(ctx context.Context, transcript string) (string, error) {
		return "response", nil
	}

	sa := NewStreamingAgent(callback)
	assert.False(t, sa.IsStreaming())

	ctx := context.Background()
	ch, err := sa.StartStreaming(ctx, "Hello")
	require.NoError(t, err)
	assert.True(t, sa.IsStreaming())

	// Wait for completion
	<-ch
	time.Sleep(100 * time.Millisecond)
	assert.False(t, sa.IsStreaming())
}

func TestStreamingAgent_ConcurrentAccess(t *testing.T) {
	callback := func(ctx context.Context, transcript string) (string, error) {
		return "response", nil
	}

	sa := NewStreamingAgent(callback)

	// Concurrent access
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			sa.IsStreaming()
			sa.StopStreaming()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

