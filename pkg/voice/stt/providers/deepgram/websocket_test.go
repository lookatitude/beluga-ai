package deepgram

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDeepgramStreamingSession_Error(t *testing.T) {
	config := DefaultDeepgramConfig()
	config.APIKey = "test-key"
	config.WebSocketURL = "wss://invalid-url-that-does-not-exist.com/v1/listen"

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	session, err := NewDeepgramStreamingSession(ctx, config)
	// Should fail without valid WebSocket connection
	assert.Error(t, err)
	assert.Nil(t, session)
}

func TestNewDeepgramStreamingSession_WithHTTPResponse(t *testing.T) {
	// Test error path when dial returns HTTP response
	config := DefaultDeepgramConfig()
	config.APIKey = "test-key"
	config.WebSocketURL = "wss://httpbin.org/status/401" // Returns HTTP 401

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	session, err := NewDeepgramStreamingSession(ctx, config)
	// Should fail with HTTP error
	assert.Error(t, err)
	assert.Nil(t, session)
}

func TestNewDeepgramStreamingSession_WithOptionalParams(t *testing.T) {
	config := DefaultDeepgramConfig()
	config.APIKey = "test-key"
	config.WebSocketURL = "wss://invalid-url"
	config.InterimResults = false
	config.Endpointing = 500
	config.VADEvents = true

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	session, err := NewDeepgramStreamingSession(ctx, config)
	assert.Error(t, err)
	assert.Nil(t, session)
}

func TestNewDeepgramStreamingSession_WithMetrics(t *testing.T) {
	config := DefaultDeepgramConfig()
	config.APIKey = "test-key"
	config.WebSocketURL = "wss://invalid-url"
	config.EnableMetrics = true

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	session, err := NewDeepgramStreamingSession(ctx, config)
	assert.Error(t, err)
	assert.Nil(t, session)
}

func TestDeepgramStreamingSession_Close(t *testing.T) {
	// Create a session that will fail to connect
	config := DefaultDeepgramConfig()
	config.APIKey = "invalid-key"
	config.WebSocketURL = "wss://invalid-url"

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	session, err := NewDeepgramStreamingSession(ctx, config)
	// We expect an error
	if err == nil && session != nil {
		// If somehow it connected, test close
		err := session.Close()
		assert.NoError(t, err)
	}
}

func TestDeepgramStreamingSession_SendAudio_Closed(t *testing.T) {
	// Test SendAudio on a closed session
	config := DefaultDeepgramConfig()
	config.APIKey = "test-key"
	config.WebSocketURL = "wss://invalid-url"

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	session, err := NewDeepgramStreamingSession(ctx, config)
	// Should fail to connect
	if err == nil && session != nil {
		// Close the session first
		_ = session.Close()

		// Try to send audio to closed session
		audio := []byte{1, 2, 3, 4, 5}
		err = session.SendAudio(ctx, audio)
		assert.Error(t, err)
	}
}

func TestDeepgramStreamingSession_ReceiveTranscript(t *testing.T) {
	// Test ReceiveTranscript returns a channel
	config := DefaultDeepgramConfig()
	config.APIKey = "test-key"
	config.WebSocketURL = "wss://invalid-url"

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	session, err := NewDeepgramStreamingSession(ctx, config)
	// Should fail to connect
	if err == nil && session != nil {
		ch := session.ReceiveTranscript()
		assert.NotNil(t, ch)
		_ = session.Close()
	}
}

func TestDeepgramStreamingSession_SendAudio_Success(t *testing.T) {
	mockServer := testutils.NewMockWebSocketServer()
	defer mockServer.Close()

	config := DefaultDeepgramConfig()
	config.APIKey = "test-key"
	config.WebSocketURL = mockServer.URL()

	ctx := context.Background()
	session, err := NewDeepgramStreamingSession(ctx, config)
	require.NoError(t, err)
	require.NotNil(t, session)
	defer session.Close()

	// Wait for connection
	require.True(t, mockServer.WaitForConnection(1*time.Second))

	// Send audio
	audio := []byte{1, 2, 3, 4, 5}
	err = session.SendAudio(ctx, audio)
	assert.NoError(t, err)

	// Give server time to receive message
	time.Sleep(100 * time.Millisecond)

	// Verify message was received
	messages := mockServer.GetMessages()
	if len(messages) == 0 {
		// Try again after a bit more time
		time.Sleep(200 * time.Millisecond)
		messages = mockServer.GetMessages()
	}
	assert.Greater(t, len(messages), 0, "Expected at least one message")
	if len(messages) > 0 {
		assert.Equal(t, audio, messages[0])
	}
}

func TestDeepgramStreamingSession_ReceiveTranscript_Success(t *testing.T) {
	mockServer := testutils.NewMockWebSocketServer()
	defer mockServer.Close()

	// Set up handler to send transcript response
	mockServer.SetOnMessage(func([]byte) []byte {
		response := map[string]interface{}{
			"type": "Results",
			"channel": map[string]interface{}{
				"alternatives": []map[string]interface{}{
					{
						"transcript": "Hello world",
						"confidence": 0.95,
					},
				},
			},
			"is_final": true,
		}
		data, _ := json.Marshal(response)
		return data
	})

	config := DefaultDeepgramConfig()
	config.APIKey = "test-key"
	config.WebSocketURL = mockServer.URL()

	ctx := context.Background()
	session, err := NewDeepgramStreamingSession(ctx, config)
	require.NoError(t, err)
	require.NotNil(t, session)
	defer session.Close()

	// Wait for connection
	require.True(t, mockServer.WaitForConnection(1*time.Second))

	// Send audio to trigger response
	audio := []byte{1, 2, 3, 4, 5}
	err = session.SendAudio(ctx, audio)
	require.NoError(t, err)

	// Receive transcript
	ch := session.ReceiveTranscript()
	select {
	case result := <-ch:
		assert.NoError(t, result.Error)
		assert.Equal(t, "Hello world", result.Text)
		assert.True(t, result.IsFinal)
		assert.Equal(t, 0.95, result.Confidence)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for transcript")
	}
}

func TestDeepgramStreamingSession_Close_WithConnection(t *testing.T) {
	mockServer := testutils.NewMockWebSocketServer()
	defer mockServer.Close()

	config := DefaultDeepgramConfig()
	config.APIKey = "test-key"
	config.WebSocketURL = mockServer.URL()

	ctx := context.Background()
	session, err := NewDeepgramStreamingSession(ctx, config)
	require.NoError(t, err)
	require.NotNil(t, session)

	// Wait for connection
	require.True(t, mockServer.WaitForConnection(1*time.Second))

	// Close session
	err = session.Close()
	assert.NoError(t, err)

	// Try to send after close
	err = session.SendAudio(ctx, []byte{1, 2, 3})
	assert.Error(t, err)
}

func TestDeepgramStreamingSession_ReceiveTranscript_Interim(t *testing.T) {
	mockServer := testutils.NewMockWebSocketServer()
	defer mockServer.Close()

	// Set up handler to send interim transcript
	mockServer.SetOnMessage(func([]byte) []byte {
		response := map[string]interface{}{
			"type": "Results",
			"channel": map[string]interface{}{
				"alternatives": []map[string]interface{}{
					{
						"transcript": "Hello",
						"confidence": 0.8,
					},
				},
			},
			"is_final": false,
		}
		data, _ := json.Marshal(response)
		return data
	})

	config := DefaultDeepgramConfig()
	config.APIKey = "test-key"
	config.WebSocketURL = mockServer.URL()

	ctx := context.Background()
	session, err := NewDeepgramStreamingSession(ctx, config)
	require.NoError(t, err)
	require.NotNil(t, session)
	defer session.Close()

	// Wait for connection
	require.True(t, mockServer.WaitForConnection(1*time.Second))

	// Send audio
	err = session.SendAudio(ctx, []byte{1, 2, 3})
	require.NoError(t, err)

	// Receive interim transcript
	ch := session.ReceiveTranscript()
	select {
	case result := <-ch:
		assert.NoError(t, result.Error)
		assert.Equal(t, "Hello", result.Text)
		assert.False(t, result.IsFinal)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for transcript")
	}
}

func TestDeepgramStreamingSession_ReceiveTranscript_MalformedResponse(t *testing.T) {
	mockServer := testutils.NewMockWebSocketServer()
	defer mockServer.Close()

	// Set up handler to send malformed JSON
	mockServer.SetOnMessage(func([]byte) []byte {
		return []byte("invalid json")
	})

	config := DefaultDeepgramConfig()
	config.APIKey = "test-key"
	config.WebSocketURL = mockServer.URL()

	ctx := context.Background()
	session, err := NewDeepgramStreamingSession(ctx, config)
	require.NoError(t, err)
	require.NotNil(t, session)
	defer session.Close()

	// Wait for connection
	require.True(t, mockServer.WaitForConnection(1*time.Second))

	// Send audio
	err = session.SendAudio(ctx, []byte{1, 2, 3})
	require.NoError(t, err)

	// Receive error
	ch := session.ReceiveTranscript()
	select {
	case result := <-ch:
		assert.Error(t, result.Error)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for error")
	}
}
