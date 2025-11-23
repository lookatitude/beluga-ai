package turndetection

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/voice/turndetection/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProvider(t *testing.T) {
	ctx := context.Background()

	// Register a test provider to avoid import cycle
	registry := GetRegistry()
	testFactory := func(config *Config) (iface.TurnDetector, error) {
		return NewAdvancedMockTurnDetector("test"), nil
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
	testFactory := func(config *Config) (iface.TurnDetector, error) {
		return NewAdvancedMockTurnDetector("test"), nil
	}
	registry.Register("test-provider", testFactory)

	config := DefaultConfig()
	config.Provider = "test-provider"

	provider, err := NewProvider(ctx, "", config, func(c *Config) {
		c.MinSilenceDuration = 500
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

