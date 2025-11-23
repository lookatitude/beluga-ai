package azure

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAzureStreamingSession_Error(t *testing.T) {
	config := DefaultAzureConfig()
	config.APIKey = "test-key"
	config.WebSocketURL = "wss://invalid-url-that-does-not-exist.com"

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	session, err := NewAzureStreamingSession(ctx, config)
	// Should fail without valid WebSocket connection
	assert.Error(t, err)
	assert.Nil(t, session)
}

func TestNewAzureStreamingSession_WithHTTPResponse(t *testing.T) {
	// Test error path when dial returns HTTP response
	config := DefaultAzureConfig()
	config.APIKey = "test-key"
	config.WebSocketURL = "wss://httpbin.org/status/401" // Returns HTTP 401

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	session, err := NewAzureStreamingSession(ctx, config)
	// Should fail with HTTP error
	assert.Error(t, err)
	assert.Nil(t, session)
}

func TestNewAzureStreamingSession_WithOptionalParams(t *testing.T) {
	config := DefaultAzureConfig()
	config.APIKey = "test-key"
	config.WebSocketURL = "wss://invalid-url"
	config.EnablePunctuation = true
	config.EnableWordLevelTimestamps = true
	config.EnableSpeakerDiarization = true
	config.EndpointID = "test-endpoint-id"

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	session, err := NewAzureStreamingSession(ctx, config)
	assert.Error(t, err)
	assert.Nil(t, session)
}

func TestNewAzureStreamingSession_WithMetrics(t *testing.T) {
	config := DefaultAzureConfig()
	config.APIKey = "test-key"
	config.WebSocketURL = "wss://invalid-url"
	config.EnableMetrics = true

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	session, err := NewAzureStreamingSession(ctx, config)
	assert.Error(t, err)
	assert.Nil(t, session)
}

func TestAzureStreamingSession_Close(t *testing.T) {
	// Create a session that will fail to connect
	config := DefaultAzureConfig()
	config.APIKey = "invalid-key"
	config.WebSocketURL = "wss://invalid-url"

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	session, err := NewAzureStreamingSession(ctx, config)
	// We expect an error
	if err == nil && session != nil {
		// If somehow it connected, test close
		err := session.Close()
		assert.NoError(t, err)
	}
}

func TestAzureStreamingSession_SendAudio_Closed(t *testing.T) {
	// Test SendAudio on a closed session
	config := DefaultAzureConfig()
	config.APIKey = "test-key"
	config.WebSocketURL = "wss://invalid-url"

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	session, err := NewAzureStreamingSession(ctx, config)
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

func TestAzureStreamingSession_ReceiveTranscript(t *testing.T) {
	// Test ReceiveTranscript returns a channel
	config := DefaultAzureConfig()
	config.APIKey = "test-key"
	config.WebSocketURL = "wss://invalid-url"

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	session, err := NewAzureStreamingSession(ctx, config)
	// Should fail to connect
	if err == nil && session != nil {
		ch := session.ReceiveTranscript()
		assert.NotNil(t, ch)
		_ = session.Close()
	}
}

func TestGenerateConnectionID(t *testing.T) {
	// Test that generateConnectionID returns a non-empty string
	id1 := generateConnectionID()
	assert.NotEmpty(t, id1)

	// Test that it generates different IDs
	time.Sleep(1 * time.Millisecond)
	id2 := generateConnectionID()
	assert.NotEqual(t, id1, id2)
}

func TestAzureStreamingSession_SendAudio_Success(t *testing.T) {
	mockServer := testutils.NewMockWebSocketServer()
	defer mockServer.Close()

	config := DefaultAzureConfig()
	config.APIKey = "test-key"
	config.WebSocketURL = mockServer.URL()

	ctx := context.Background()
	session, err := NewAzureStreamingSession(ctx, config)
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

func TestAzureStreamingSession_ReceiveTranscript_Success(t *testing.T) {
	mockServer := testutils.NewMockWebSocketServer()
	defer mockServer.Close()

	// Set up handler to send transcript response
	mockServer.SetOnMessage(func([]byte) []byte {
		response := map[string]interface{}{
			"RecognitionStatus": "Success",
			"DisplayText":        "Hello world",
			"Offset":             0,
			"Duration":            1000,
		}
		data, _ := json.Marshal(response)
		return data
	})

	config := DefaultAzureConfig()
	config.APIKey = "test-key"
	config.WebSocketURL = mockServer.URL()

	ctx := context.Background()
	session, err := NewAzureStreamingSession(ctx, config)
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
		assert.Equal(t, 1.0, result.Confidence)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for transcript")
	}
}

func TestAzureStreamingSession_Close_WithConnection(t *testing.T) {
	mockServer := testutils.NewMockWebSocketServer()
	defer mockServer.Close()

	config := DefaultAzureConfig()
	config.APIKey = "test-key"
	config.WebSocketURL = mockServer.URL()

	ctx := context.Background()
	session, err := NewAzureStreamingSession(ctx, config)
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

func TestAzureStreamingSession_ReceiveTranscript_NonSuccess(t *testing.T) {
	mockServer := testutils.NewMockWebSocketServer()
	defer mockServer.Close()

	// Set up handler to send non-success response
	mockServer.SetOnMessage(func([]byte) []byte {
		response := map[string]interface{}{
			"RecognitionStatus": "NoMatch",
			"DisplayText":       "",
		}
		data, _ := json.Marshal(response)
		return data
	})

	config := DefaultAzureConfig()
	config.APIKey = "test-key"
	config.WebSocketURL = mockServer.URL()

	ctx := context.Background()
	session, err := NewAzureStreamingSession(ctx, config)
	require.NoError(t, err)
	require.NotNil(t, session)
	defer session.Close()

	// Wait for connection
	require.True(t, mockServer.WaitForConnection(1*time.Second))

	// Send audio
	err = session.SendAudio(ctx, []byte{1, 2, 3})
	require.NoError(t, err)

	// Should not receive a result (NoMatch status is ignored)
	ch := session.ReceiveTranscript()
	select {
	case <-ch:
		t.Fatal("Should not receive result for NoMatch status")
	case <-time.After(500 * time.Millisecond):
		// Expected - no result should be sent
	}
}

func TestAzureStreamingSession_ReceiveTranscript_MalformedResponse(t *testing.T) {
	mockServer := testutils.NewMockWebSocketServer()
	defer mockServer.Close()

	// Set up handler to send malformed JSON
	mockServer.SetOnMessage(func([]byte) []byte {
		return []byte("invalid json")
	})

	config := DefaultAzureConfig()
	config.APIKey = "test-key"
	config.WebSocketURL = mockServer.URL()

	ctx := context.Background()
	session, err := NewAzureStreamingSession(ctx, config)
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

