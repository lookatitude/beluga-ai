package transport

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.NotNil(t, config)
	assert.Equal(t, "webrtc", config.Provider)
	assert.Equal(t, 16000, config.SampleRate)
	assert.Equal(t, 1, config.Channels)
	assert.Equal(t, 16, config.BitDepth)
	assert.Equal(t, "pcm", config.Codec)
	assert.Equal(t, 10*time.Second, config.ConnectTimeout)
	assert.Equal(t, 3, config.ReconnectAttempts)
	assert.Equal(t, 1*time.Second, config.ReconnectDelay)
	assert.Equal(t, 4096, config.SendBufferSize)
	assert.Equal(t, 4096, config.ReceiveBufferSize)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.True(t, config.EnableTracing)
	assert.True(t, config.EnableMetrics)
	assert.True(t, config.EnableStructuredLogging)
}

func TestConfig_WithProvider(t *testing.T) {
	config := DefaultConfig()
	WithProvider("websocket")(config)
	assert.Equal(t, "websocket", config.Provider)
}

func TestConfig_WithURL(t *testing.T) {
	config := DefaultConfig()
	WithURL("ws://example.com")(config)
	assert.Equal(t, "ws://example.com", config.URL)
}

func TestConfig_WithSampleRate(t *testing.T) {
	config := DefaultConfig()
	WithSampleRate(48000)(config)
	assert.Equal(t, 48000, config.SampleRate)
}

func TestConfig_WithChannels(t *testing.T) {
	config := DefaultConfig()
	WithChannels(2)(config)
	assert.Equal(t, 2, config.Channels)
}

func TestConfig_WithCodec(t *testing.T) {
	config := DefaultConfig()
	WithCodec("opus")(config)
	assert.Equal(t, "opus", config.Codec)
}

func TestConfig_WithConnectTimeout(t *testing.T) {
	config := DefaultConfig()
	timeout := 5 * time.Second
	WithConnectTimeout(timeout)(config)
	assert.Equal(t, timeout, config.ConnectTimeout)
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Provider:          "webrtc",
				URL:               "ws://example.com",
				SampleRate:        16000,
				Channels:          1,
				BitDepth:          16,
				Codec:             "pcm",
				ConnectTimeout:    10 * time.Second,
				ReconnectAttempts: 3,
				ReconnectDelay:    1 * time.Second,
				SendBufferSize:    4096,
				ReceiveBufferSize: 4096,
				Timeout:           30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "missing provider",
			config: &Config{
				URL: "ws://example.com",
			},
			wantErr: true,
		},
		{
			name: "missing URL",
			config: &Config{
				Provider: "webrtc",
			},
			wantErr: true,
		},
		{
			name: "invalid sample rate",
			config: &Config{
				Provider:   "webrtc",
				URL:        "ws://example.com",
				SampleRate: 9999,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
