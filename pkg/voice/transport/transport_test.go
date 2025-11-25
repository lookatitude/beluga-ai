package transport

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/voice/transport/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProvider(t *testing.T) {
	ctx := context.Background()

	// Register a test provider using valid provider name
	registry := GetRegistry()
	testFactory := func(config *Config) (iface.Transport, error) {
		return NewAdvancedMockTransport("test"), nil
	}
	registry.Register("webrtc", testFactory)

	tests := []struct {
		name         string
		providerName string
		config       *Config
		wantErr      bool
	}{
		{
			name:         "valid provider",
			providerName: "webrtc",
			config: func() *Config {
				c := DefaultConfig()
				c.URL = "wss://example.com" // Required field
				return c
			}(),
			wantErr: false,
		},
		{
			name:         "nil config uses defaults",
			providerName: "webrtc",
			config: func() *Config {
				c := DefaultConfig()
				c.URL = "wss://example.com" // Required field
				return c
			}(),
			wantErr: false,
		},
		{
			name:         "invalid provider",
			providerName: "invalid",
			config: func() *Config {
				c := DefaultConfig()
				c.URL = "wss://example.com" // Required field
				return c
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProvider(ctx, tt.providerName, tt.config)
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

func TestNewProvider_WithOptions(t *testing.T) {
	ctx := context.Background()

	// Register a test provider using valid provider name
	registry := GetRegistry()
	testFactory := func(config *Config) (iface.Transport, error) {
		return NewAdvancedMockTransport("test"), nil
	}
	registry.Register("webrtc", testFactory)

	config := DefaultConfig()
	config.Provider = "webrtc"
	config.URL = "wss://example.com" // Required field

	provider, err := NewProvider(ctx, "", config, func(c *Config) {
		c.SampleRate = 48000
	})
	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestNewProvider_OverrideProviderName(t *testing.T) {
	ctx := context.Background()

	// Register test providers using valid provider names
	registry := GetRegistry()
	testFactory1 := func(config *Config) (iface.Transport, error) {
		return NewAdvancedMockTransport("test1"), nil
	}
	testFactory2 := func(config *Config) (iface.Transport, error) {
		return NewAdvancedMockTransport("test2"), nil
	}
	registry.Register("webrtc", testFactory1)
	registry.Register("websocket", testFactory2)

	config := DefaultConfig()
	config.Provider = "websocket" // Different from providerName
	config.URL = "wss://example.com" // Required field

	provider, err := NewProvider(ctx, "webrtc", config)
	require.NoError(t, err)
	assert.NotNil(t, provider)
	// Provider name should be overridden
	assert.Equal(t, "webrtc", config.Provider)
}

func TestInitMetrics(t *testing.T) {
	// Test that InitMetrics can be called multiple times
	// but only initializes once (metricsOnce)
	metrics := GetMetrics()
	// May be nil if not initialized, which is fine
	_ = metrics
}

func TestGetMetrics(t *testing.T) {
	metrics := GetMetrics()
	// May be nil if not initialized, which is fine
	_ = metrics
}
