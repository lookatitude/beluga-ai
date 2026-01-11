package s2s

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		config  *Config
		name    string
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Provider:              "amazon_nova",
				APIKey:                "test-key",
				Timeout:               30 * time.Second,
				SampleRate:            24000,
				Channels:              1,
				MaxRetries:            3,
				RetryDelay:            1 * time.Second,
				RetryBackoff:          2.0,
				LatencyTarget:         "medium",
				ReasoningMode:         "built-in",
				MaxConcurrentSessions: 50,
			},
			wantErr: false,
		},
		{
			name: "missing provider",
			config: &Config{
				APIKey: "test-key",
			},
			wantErr: true,
		},
		{
			name: "invalid provider",
			config: &Config{
				Provider: "invalid",
				APIKey:   "test-key",
			},
			wantErr: true,
		},
		{
			name: "invalid sample rate",
			config: &Config{
				Provider:   "amazon_nova",
				APIKey:     "test-key",
				SampleRate: 44100, // Not in allowed list
			},
			wantErr: true,
		},
		{
			name: "invalid channels",
			config: &Config{
				Provider: "amazon_nova",
				APIKey:   "test-key",
				Channels: 3, // Invalid
			},
			wantErr: true,
		},
		{
			name: "invalid timeout",
			config: &Config{
				Provider: "amazon_nova",
				APIKey:   "test-key",
				Timeout:  10 * time.Minute, // Too long
			},
			wantErr: true,
		},
		{
			name: "invalid retry count",
			config: &Config{
				Provider:   "amazon_nova",
				APIKey:     "test-key",
				MaxRetries: 15, // Too many
			},
			wantErr: true,
		},
		{
			name: "invalid latency target",
			config: &Config{
				Provider:      "amazon_nova",
				APIKey:        "test-key",
				LatencyTarget: "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid reasoning mode",
			config: &Config{
				Provider:      "amazon_nova",
				APIKey:        "test-key",
				ReasoningMode: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfig_DefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.NotNil(t, config)
	assert.Equal(t, "mock", config.Provider)
	assert.Equal(t, 24000, config.SampleRate)
	assert.Equal(t, 1, config.Channels)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, "medium", config.LatencyTarget)
	assert.Equal(t, "built-in", config.ReasoningMode)
}

func TestConfig_ConfigOptions(t *testing.T) {
	config := DefaultConfig()

	WithProvider("amazon_nova")(config)
	assert.Equal(t, "amazon_nova", config.Provider)

	WithSampleRate(16000)(config)
	assert.Equal(t, 16000, config.SampleRate)

	WithChannels(2)(config)
	assert.Equal(t, 2, config.Channels)

	WithLanguage("en-US")(config)
	assert.Equal(t, "en-US", config.Language)

	timeout := 60 * time.Second
	WithTimeout(timeout)(config)
	assert.Equal(t, timeout, config.Timeout)

	WithMaxRetries(5)(config)
	assert.Equal(t, 5, config.MaxRetries)

	WithLatencyTarget("low")(config)
	assert.Equal(t, "low", config.LatencyTarget)

	WithReasoningMode("external")(config)
	assert.Equal(t, "external", config.ReasoningMode)

	WithFallbackProviders("grok", "gemini")(config)
	assert.Equal(t, []string{"grok", "gemini"}, config.FallbackProviders)
}
