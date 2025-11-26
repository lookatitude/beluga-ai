package noise

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/voice/noise/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProvider(t *testing.T) {
	ctx := context.Background()
	// Register a test provider to avoid import cycle
	registry := GetRegistry()
	testFactory := func(config *Config) (iface.NoiseCancellation, error) {
		return NewAdvancedMockNoiseCancellation("test"), nil
	}
	registry.Register("test-provider", testFactory)

	tests := []struct {
		config       *Config
		name         string
		providerName string
		wantErr      bool
	}{
		{
			name:         "valid provider",
			providerName: "test-provider",
			config:       DefaultConfig(),
			wantErr:      false,
		},
		{
			name:         "nil config uses defaults",
			providerName: "test-provider",
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
	// Register a test provider
	registry := GetRegistry()
	testFactory := func(config *Config) (iface.NoiseCancellation, error) {
		return NewAdvancedMockNoiseCancellation("test"), nil
	}
	registry.Register("rnnoise", testFactory) // Use valid provider name

	config := DefaultConfig()
	// Don't set Provider in config - it will be validated and must be one of: rnnoise, webrtc, spectral
	// The providerName parameter in NewProvider will override it if provided

	provider, err := NewProvider(ctx, "rnnoise", config, func(c *Config) {
		c.FrameSize = 1024
	})
	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestInitMetrics(t *testing.T) {
	metrics := GetMetrics()
	_ = metrics
}

func TestGetMetrics(t *testing.T) {
	metrics := GetMetrics()
	_ = metrics
}
