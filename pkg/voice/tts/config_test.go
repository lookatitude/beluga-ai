package tts

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Provider:     "openai",
				APIKey:       "test-key",
				Model:        "tts-1",
				Voice:        "nova",
				Language:     "en",
				Timeout:      30 * time.Second,
				SampleRate:   24000,
				BitDepth:     16,
				Speed:        1.0,
				Pitch:        0.0,
				Volume:       1.0,
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
			name: "invalid speed",
			config: &Config{
				Provider: "openai",
				APIKey:   "test-key",
				Speed:    5.0, // Too high
			},
			wantErr: true,
		},
		{
			name: "invalid pitch",
			config: &Config{
				Provider: "openai",
				APIKey:   "test-key",
				Pitch:    25.0, // Too high
			},
			wantErr: true,
		},
		{
			name: "invalid volume",
			config: &Config{
				Provider: "openai",
				APIKey:   "test-key",
				Volume:   2.0, // Too high
			},
			wantErr: true,
		},
		{
			name: "invalid sample rate",
			config: &Config{
				Provider:   "openai",
				APIKey:     "test-key",
				SampleRate: 32000, // Not in allowed list
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

func TestConfig_DefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.NotNil(t, config)
	assert.Equal(t, "openai", config.Provider)
	assert.Equal(t, 24000, config.SampleRate)
	assert.Equal(t, 16, config.BitDepth)
	assert.Equal(t, 1.0, config.Speed)
	assert.True(t, config.EnableStreaming)
	assert.Equal(t, 3, config.MaxRetries)
}

func TestConfigOption(t *testing.T) {
	config := DefaultConfig()

	WithProvider("google")(config)
	assert.Equal(t, "google", config.Provider)

	WithAPIKey("new-key")(config)
	assert.Equal(t, "new-key", config.APIKey)

	WithModel("tts-1-hd")(config)
	assert.Equal(t, "tts-1-hd", config.Model)

	WithVoice("alloy")(config)
	assert.Equal(t, "alloy", config.Voice)

	WithLanguage("es")(config)
	assert.Equal(t, "es", config.Language)

	WithSpeed(1.5)(config)
	assert.Equal(t, 1.5, config.Speed)

	WithPitch(2.0)(config)
	assert.Equal(t, 2.0, config.Pitch)

	WithVolume(0.8)(config)
	assert.Equal(t, 0.8, config.Volume)

	WithTimeout(60 * time.Second)(config)
	assert.Equal(t, 60*time.Second, config.Timeout)

	WithSampleRate(48000)(config)
	assert.Equal(t, 48000, config.SampleRate)

	WithEnableStreaming(false)(config)
	assert.False(t, config.EnableStreaming)
}
