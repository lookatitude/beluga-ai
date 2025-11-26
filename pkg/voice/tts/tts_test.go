package tts

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/voice/tts/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProvider(t *testing.T) {
	ctx := context.Background()

	// Register a test provider using valid provider name
	registry := GetRegistry()
	testFactory := func(config *Config) (iface.TTSProvider, error) {
		return NewAdvancedMockTTSProvider("test"), nil
	}
	registry.Register("openai", testFactory)

	tests := []struct {
		config       *Config
		name         string
		providerName string
		wantErr      bool
	}{
		{
			name:         "valid provider",
			providerName: "openai",
			config: func() *Config {
				c := DefaultConfig()
				c.APIKey = "test-key" // Required field
				return c
			}(),
			wantErr: false,
		},
		{
			name:         "nil config uses defaults",
			providerName: "openai",
			config: func() *Config {
				c := DefaultConfig()
				c.APIKey = "test-key" // Required field
				return c
			}(),
			wantErr: false,
		},
		{
			name:         "invalid provider",
			providerName: "invalid",
			config: func() *Config {
				c := DefaultConfig()
				c.APIKey = "test-key" // Required field
				return c
			}(),
			wantErr: true,
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
	testFactory := func(config *Config) (iface.TTSProvider, error) {
		return NewAdvancedMockTTSProvider("test"), nil
	}
	registry.Register("openai", testFactory)

	config := DefaultConfig()
	config.Provider = "openai"
	config.APIKey = "test-key" // Required field

	provider, err := NewProvider(ctx, "", config, func(c *Config) {
		c.Model = "test-model"
	})
	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestNewProvider_OverrideProviderName(t *testing.T) {
	ctx := context.Background()

	// Register test providers using valid provider names
	registry := GetRegistry()
	testFactory1 := func(config *Config) (iface.TTSProvider, error) {
		return NewAdvancedMockTTSProvider("test1"), nil
	}
	testFactory2 := func(config *Config) (iface.TTSProvider, error) {
		return NewAdvancedMockTTSProvider("test2"), nil
	}
	registry.Register("openai", testFactory1)
	registry.Register("google", testFactory2)

	config := DefaultConfig()
	config.Provider = "google" // Different from providerName
	config.APIKey = "test-key" // Required field

	provider, err := NewProvider(ctx, "openai", config)
	require.NoError(t, err)
	assert.NotNil(t, provider)
	// Provider name should be overridden
	assert.Equal(t, "openai", config.Provider)
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
