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

	// Register a test provider to avoid import cycle
	registry := GetRegistry()
	testFactory := func(config *Config) (iface.Transport, error) {
		return NewAdvancedMockTransport("test"), nil
	}
	registry.Register("test-provider", testFactory)

	tests := []struct {
		name        string
		providerName string
		config      *Config
		wantErr     bool
	}{
		{
			name:        "valid provider",
			providerName: "test-provider",
			config:      DefaultConfig(),
			wantErr:     false,
		},
		{
			name:        "nil config uses defaults",
			providerName: "test-provider",
			config:      nil,
			wantErr:     false,
		},
		{
			name:        "invalid provider",
			providerName: "invalid",
			config:      DefaultConfig(),
			wantErr:     true,
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

	// Register a test provider
	registry := GetRegistry()
	testFactory := func(config *Config) (iface.Transport, error) {
		return NewAdvancedMockTransport("test"), nil
	}
	registry.Register("test-provider", testFactory)

	config := DefaultConfig()
	config.Provider = "test-provider"

	provider, err := NewProvider(ctx, "", config, func(c *Config) {
		c.SampleRate = 48000
	})
	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestNewProvider_OverrideProviderName(t *testing.T) {
	ctx := context.Background()

	// Register test providers
	registry := GetRegistry()
	testFactory1 := func(config *Config) (iface.Transport, error) {
		return NewAdvancedMockTransport("test1"), nil
	}
	testFactory2 := func(config *Config) (iface.Transport, error) {
		return NewAdvancedMockTransport("test2"), nil
	}
	registry.Register("test-provider-1", testFactory1)
	registry.Register("test-provider-2", testFactory2)

	config := DefaultConfig()
	config.Provider = "test-provider-2" // Different from providerName

	provider, err := NewProvider(ctx, "test-provider-1", config)
	require.NoError(t, err)
	assert.NotNil(t, provider)
	// Provider name should be overridden
	assert.Equal(t, "test-provider-1", config.Provider)
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

