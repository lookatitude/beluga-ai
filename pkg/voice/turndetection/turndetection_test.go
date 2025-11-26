package turndetection

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/turndetection/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProvider(t *testing.T) {
	ctx := context.Background()

	// Register a test provider using valid provider name
	registry := GetRegistry()
	testFactory := func(config *Config) (iface.TurnDetector, error) {
		return NewAdvancedMockTurnDetector("test"), nil
	}
	registry.Register("heuristic", testFactory)

	tests := []struct {
		config       *Config
		name         string
		providerName string
		wantErr      bool
	}{
		{
			name:         "valid provider",
			providerName: "heuristic",
			config:       DefaultConfig(),
			wantErr:      false,
		},
		{
			name:         "nil config uses defaults",
			providerName: "heuristic",
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
	testFactory := func(config *Config) (iface.TurnDetector, error) {
		return NewAdvancedMockTurnDetector("test"), nil
	}
	registry.Register("heuristic", testFactory)

	config := DefaultConfig()
	config.Provider = "heuristic"

	provider, err := NewProvider(ctx, "", config, func(c *Config) {
		c.MinSilenceDuration = 500 * time.Millisecond // Must be >= 100ms
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
