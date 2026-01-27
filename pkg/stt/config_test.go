package stt

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
				Provider:     "deepgram",
				APIKey:       "test-key",
				Model:        "nova-3",
				Language:     "en",
				Timeout:      30 * time.Second,
				SampleRate:   16000,
				Channels:     1,
				MaxRetries:   3,
				RetryDelay:   1 * time.Second,
				RetryBackoff: 2.0,
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
				Provider:   "deepgram",
				APIKey:     "test-key",
				SampleRate: 44100, // Not in allowed list
			},
			wantErr: true,
		},
		{
			name: "invalid channels",
			config: &Config{
				Provider: "deepgram",
				APIKey:   "test-key",
				Channels: 3, // Invalid
			},
			wantErr: true,
		},
		{
			name: "invalid timeout",
			config: &Config{
				Provider: "deepgram",
				APIKey:   "test-key",
				Timeout:  10 * time.Minute, // Too long
			},
			wantErr: true,
		},
		{
			name: "invalid retry count",
			config: &Config{
				Provider:   "deepgram",
				APIKey:     "test-key",
				MaxRetries: 15, // Too many
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
	assert.Equal(t, "deepgram", config.Provider)
	assert.Equal(t, 16000, config.SampleRate)
	assert.Equal(t, 1, config.Channels)
	assert.True(t, config.EnableStreaming)
	assert.Equal(t, 3, config.MaxRetries)
}

func TestConfigOption(t *testing.T) {
	config := DefaultConfig()

	WithProvider("google")(config)
	assert.Equal(t, "google", config.Provider)

	WithAPIKey("new-key")(config)
	assert.Equal(t, "new-key", config.APIKey)

	WithModel("whisper")(config)
	assert.Equal(t, "whisper", config.Model)

	WithLanguage("es")(config)
	assert.Equal(t, "es", config.Language)

	WithTimeout(60 * time.Second)(config)
	assert.Equal(t, 60*time.Second, config.Timeout)

	WithSampleRate(48000)(config)
	assert.Equal(t, 48000, config.SampleRate)

	WithChannels(2)(config)
	assert.Equal(t, 2, config.Channels)

	WithEnableStreaming(false)(config)
	assert.False(t, config.EnableStreaming)
}
