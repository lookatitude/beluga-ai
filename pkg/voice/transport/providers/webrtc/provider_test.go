package webrtc

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWebRTCTransport(t *testing.T) {
	tests := []struct {
		name    string
		config  *transport.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &transport.Config{
				Provider: "webrtc",
				URL:      "wss://example.com",
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
				Provider: "webrtc",
				URL:      "wss://example.com",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewWebRTCTransport(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, provider)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
			}
		})
	}
}

func TestWebRTCTransport_SendAudio(t *testing.T) {
	config := &transport.Config{
		Provider: "webrtc",
		URL:      "wss://example.com",
	}

	transport, err := NewWebRTCTransport(config)
	require.NoError(t, err)
	require.NotNil(t, transport)

	webrtcTransport := transport.(*WebRTCTransport)
	webrtcTransport.connected = true

	ctx := context.Background()
	audio := []byte{1, 2, 3, 4, 5}

	// Test sending audio
	err = webrtcTransport.SendAudio(ctx, audio)
	assert.NoError(t, err)

	// Test sending empty audio
	err = webrtcTransport.SendAudio(ctx, []byte{})
	assert.Error(t, err)
}

func TestWebRTCTransport_ReceiveAudio(t *testing.T) {
	config := &transport.Config{
		Provider: "webrtc",
		URL:      "wss://example.com",
	}

	transport, err := NewWebRTCTransport(config)
	require.NoError(t, err)
	require.NotNil(t, transport)

	webrtcTransport := transport.(*WebRTCTransport)

	// Test receiving audio
	audioCh := webrtcTransport.ReceiveAudio()
	assert.NotNil(t, audioCh)
}

func TestWebRTCTransport_Close(t *testing.T) {
	config := &transport.Config{
		Provider: "webrtc",
		URL:      "wss://example.com",
	}

	transport, err := NewWebRTCTransport(config)
	require.NoError(t, err)
	require.NotNil(t, transport)

	webrtcTransport := transport.(*WebRTCTransport)
	webrtcTransport.connected = true

	// Test closing
	err = webrtcTransport.Close()
	assert.NoError(t, err)
	assert.False(t, webrtcTransport.connected)
}

func TestWebRTCTransport_Connect(t *testing.T) {
	config := &transport.Config{
		Provider: "webrtc",
		URL:      "wss://example.com",
	}

	transport, err := NewWebRTCTransport(config)
	require.NoError(t, err)
	require.NotNil(t, transport)

	webrtcTransport := transport.(*WebRTCTransport)

	ctx := context.Background()
	err = webrtcTransport.Connect(ctx)
	assert.NoError(t, err)
	assert.True(t, webrtcTransport.connected)
}

func TestDefaultWebRTCConfig(t *testing.T) {
	config := DefaultWebRTCConfig()
	assert.NotNil(t, config)
	assert.Equal(t, "opus", config.AudioCodec)
	assert.Equal(t, "balanced", config.BundlePolicy)
	assert.Equal(t, "require", config.RTCPMuxPolicy)
	assert.True(t, config.EnableDTLS)
	assert.True(t, config.EnableSRTP)
	assert.Equal(t, 30*time.Second, config.ICEConnectionTimeout)
}
