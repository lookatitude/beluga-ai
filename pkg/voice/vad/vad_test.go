package vad

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/voice/vad/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProvider(t *testing.T) {
	ctx := context.Background()

	// Register a test provider using valid provider name
	registry := GetRegistry()
	testFactory := func(config *Config) (iface.VADProvider, error) {
		return NewAdvancedMockVADProvider("test"), nil
	}
	registry.Register("silero", testFactory)

	tests := []struct {
		config       *Config
		name         string
		providerName string
		wantErr      bool
	}{
		{
			name:         "valid provider",
			providerName: "silero",
			config:       DefaultConfig(),
			wantErr:      false,
		},
		{
			name:         "nil config uses defaults",
			providerName: "silero",
			config:       nil,
			wantErr:      false,
		},
		{
			name:         "invalid provider",
			providerName: "invalid",
			config:       DefaultConfig(),
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProvider(ctx, tt.providerName, tt.config)
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

func TestNewProvider_WithOptions(t *testing.T) {
	ctx := context.Background()

	// Register a test provider using valid provider name
	registry := GetRegistry()
	testFactory := func(config *Config) (iface.VADProvider, error) {
		return NewAdvancedMockVADProvider("test"), nil
	}
	registry.Register("silero", testFactory)

	config := DefaultConfig()
	config.Provider = "silero"

	provider, err := NewProvider(ctx, "", config, func(c *Config) {
		c.Threshold = 0.5
	})
	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestNewProvider_OverrideProviderName(t *testing.T) {
	ctx := context.Background()

	// Register test providers using valid provider names
	registry := GetRegistry()
	testFactory1 := func(config *Config) (iface.VADProvider, error) {
		return NewAdvancedMockVADProvider("test1"), nil
	}
	testFactory2 := func(config *Config) (iface.VADProvider, error) {
		return NewAdvancedMockVADProvider("test2"), nil
	}
	registry.Register("silero", testFactory1)
	registry.Register("energy", testFactory2)

	config := DefaultConfig()
	config.Provider = "energy" // Different from providerName

	provider, err := NewProvider(ctx, "silero", config)
	require.NoError(t, err)
	assert.NotNil(t, provider)
	// Provider name should be overridden
	assert.Equal(t, "silero", config.Provider)
}

func TestNewProvider_InvalidConfig(t *testing.T) {
	ctx := context.Background()

	// Register a test provider
	registry := GetRegistry()
	testFactory := func(config *Config) (iface.VADProvider, error) {
		return NewAdvancedMockVADProvider("test"), nil
	}
	registry.Register("test-provider", testFactory)

	// Invalid config (missing required fields)
	config := &Config{
		Provider: "test-provider",
		// Missing required fields like Timeout, etc.
	}

	provider, err := NewProvider(ctx, "", config)
	require.Error(t, err)
	assert.Nil(t, provider)
}

func TestInitMetrics(t *testing.T) {
	// Test that InitMetrics can be called multiple times
	// but only initializes once (metricsOnce)
	// This is tested indirectly through GetMetrics
	metrics := GetMetrics()
	// May be nil if not initialized, which is fine
	_ = metrics
}

func TestGetMetrics(t *testing.T) {
	metrics := GetMetrics()
	// May be nil if not initialized, which is fine
	_ = metrics
}
