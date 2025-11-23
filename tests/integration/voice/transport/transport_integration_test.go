package transport

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/transport"
	// Import providers to trigger init() registration
	_ "github.com/lookatitude/beluga-ai/pkg/voice/transport/providers/webrtc"
	_ "github.com/lookatitude/beluga-ai/pkg/voice/transport/providers/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransport_Integration(t *testing.T) {
	// Integration test for Transport provider creation and basic operations
	// This test uses mock providers to avoid requiring real connections

	t.Run("provider creation", func(t *testing.T) {
		config := transport.DefaultConfig()
		config.Provider = "webrtc"
		config.URL = "wss://example.com"

		// Test that provider can be created via registry
		registry := transport.GetRegistry()
		provider, err := registry.GetProvider("webrtc", config)
		require.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("mock transport send", func(t *testing.T) {
		mockTransport := transport.NewAdvancedMockTransport("test",
			transport.WithConnected(true),
		)

		ctx := context.Background()
		audio := []byte{1, 2, 3, 4, 5}

		err := mockTransport.SendAudio(ctx, audio)
		require.NoError(t, err)
	})

	t.Run("mock transport receive", func(t *testing.T) {
		mockTransport := transport.NewAdvancedMockTransport("test",
			transport.WithConnected(true),
			transport.WithAudioData([]byte{1, 2, 3}, []byte{4, 5, 6}),
			transport.WithProcessingDelay(10*time.Millisecond),
		)

		audioCh := mockTransport.ReceiveAudio()
		require.NotNil(t, audioCh)

		// Receive audio data
		timeout := time.After(2 * time.Second)
		received := 0
		for {
			select {
			case audio, ok := <-audioCh:
				if !ok {
					if received == 0 {
						t.Fatal("expected data but received none")
					}
					return
				}
				assert.NotNil(t, audio)
				received++
			case <-timeout:
				if received == 0 {
					t.Fatal("timeout waiting for audio data")
				}
				return
			}
		}
	})
}

func TestTransport_ErrorHandling(t *testing.T) {
	t.Run("not connected error", func(t *testing.T) {
		mockTransport := transport.NewAdvancedMockTransport("test",
			transport.WithConnected(false),
		)

		ctx := context.Background()
		audio := []byte{1, 2, 3, 4, 5}

		err := mockTransport.SendAudio(ctx, audio)
		assert.Error(t, err)
		assert.False(t, transport.IsRetryableError(err))
	})

	t.Run("connection failed error retry", func(t *testing.T) {
		mockTransport := transport.NewAdvancedMockTransport("test",
			transport.WithError(transport.NewTransportError("Connect", transport.ErrCodeConnectionFailed, nil)),
		)

		ctx := context.Background()
		err := mockTransport.Connect(ctx)
		assert.Error(t, err)
		assert.True(t, transport.IsRetryableError(err))
	})
}

func TestTransport_ConcurrentOperations(t *testing.T) {
	mockTransport := transport.NewAdvancedMockTransport("test",
		transport.WithConnected(true),
	)

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	// Test concurrent sends
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			err := mockTransport.SendAudio(ctx, audio)
			if err != nil {
				errors <- err
			} else {
				done <- true
			}
		}()
	}

	// Collect results
	successCount := 0
	errorCount := 0
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
			successCount++
		case <-errors:
			errorCount++
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for results")
		}
	}

	assert.Equal(t, numGoroutines, successCount)
	assert.Equal(t, 0, errorCount)
}
