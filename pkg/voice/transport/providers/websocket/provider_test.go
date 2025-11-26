package websocket

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWebSocketTransport(t *testing.T) {
	tests := []struct {
		config  *transport.Config
		name    string
		wantErr bool
	}{
		{
			name: "valid config",
			config: &transport.Config{
				Provider: "websocket",
				URL:      "ws://example.com",
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "config with defaults",
			config: &transport.Config{
				Provider: "websocket",
				URL:      "ws://example.com",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewWebSocketTransport(tt.config)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, provider)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, provider)
			}
		})
	}
}

func TestWebSocketTransport_SendAudio(t *testing.T) {
	config := &transport.Config{
		Provider: "websocket",
		URL:      "ws://example.com",
	}

	transport, err := NewWebSocketTransport(config)
	require.NoError(t, err)
	require.NotNil(t, transport)

	wsTransport := transport.(*WebSocketTransport)
	wsTransport.connected = true

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	// Test sending audio
	err = wsTransport.SendAudio(ctx, audio)
	require.NoError(t, err)

	// Test sending empty audio
	err = wsTransport.SendAudio(ctx, []byte{})
	require.Error(t, err)
}

func TestWebSocketTransport_ReceiveAudio(t *testing.T) {
	config := &transport.Config{
		Provider: "websocket",
		URL:      "ws://example.com",
	}

	transport, err := NewWebSocketTransport(config)
	require.NoError(t, err)
	require.NotNil(t, transport)

	wsTransport := transport.(*WebSocketTransport)

	// Test receiving audio
	audioCh := wsTransport.ReceiveAudio()
	assert.NotNil(t, audioCh)
}

func TestWebSocketTransport_Close(t *testing.T) {
	config := &transport.Config{
		Provider: "websocket",
		URL:      "ws://example.com",
	}

	transport, err := NewWebSocketTransport(config)
	require.NoError(t, err)
	require.NotNil(t, transport)

	wsTransport := transport.(*WebSocketTransport)
	wsTransport.connected = true

	// Test closing
	err = wsTransport.Close()
	require.NoError(t, err)
	assert.False(t, wsTransport.connected)
}

func TestWebSocketTransport_Connect(t *testing.T) {
	config := &transport.Config{
		Provider: "websocket",
		URL:      "ws://example.com",
	}

	transport, err := NewWebSocketTransport(config)
	require.NoError(t, err)
	require.NotNil(t, transport)

	wsTransport := transport.(*WebSocketTransport)

	ctx := context.Background()
	err = wsTransport.Connect(ctx)
	require.NoError(t, err)
	assert.True(t, wsTransport.connected)
}

func TestDefaultWebSocketConfig(t *testing.T) {
	config := DefaultWebSocketConfig()
	assert.NotNil(t, config)
	assert.Equal(t, 4096, config.ReadBufferSize)
	assert.Equal(t, 4096, config.WriteBufferSize)
	assert.Equal(t, 10*time.Second, config.HandshakeTimeout)
	assert.False(t, config.EnableCompression)
	assert.Equal(t, 30*time.Second, config.PingInterval)
}
