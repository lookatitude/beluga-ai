package vad

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
				Provider:           "silero",
				Threshold:          0.5,
				FrameSize:          512,
				SampleRate:         16000,
				MinSpeechDuration:  250 * time.Millisecond,
				MaxSilenceDuration: 500 * time.Millisecond,
				Timeout:            1 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "missing provider",
			config: &Config{
				Threshold: 0.5,
			},
			wantErr: true,
		},
		{
			name: "invalid provider",
			config: &Config{
				Provider:  "invalid",
				Threshold: 0.5,
			},
			wantErr: true,
		},
		{
			name: "invalid threshold",
			config: &Config{
				Provider:  "silero",
				Threshold: 1.5, // Too high
			},
			wantErr: true,
		},
		{
			name: "invalid frame size",
			config: &Config{
				Provider:  "silero",
				FrameSize: 10000, // Too large
			},
			wantErr: true,
		},
		{
			name: "invalid sample rate",
			config: &Config{
				Provider:   "silero",
				SampleRate: 12000, // Not in allowed list
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
	assert.Equal(t, "silero", config.Provider)
	assert.InEpsilon(t, 0.5, config.Threshold, 0.0001)
	assert.Equal(t, 512, config.FrameSize)
	assert.Equal(t, 16000, config.SampleRate)
	assert.Equal(t, 250*time.Millisecond, config.MinSpeechDuration)
	assert.True(t, config.EnablePreprocessing)
}

func TestConfigOption(t *testing.T) {
	config := DefaultConfig()

	WithProvider("energy")(config)
	assert.Equal(t, "energy", config.Provider)

	WithThreshold(0.7)(config)
	assert.InEpsilon(t, 0.7, config.Threshold, 0.0001)

	WithFrameSize(1024)(config)
	assert.Equal(t, 1024, config.FrameSize)

	WithSampleRate(48000)(config)
	assert.Equal(t, 48000, config.SampleRate)

	WithMinSpeechDuration(500 * time.Millisecond)(config)
	assert.Equal(t, 500*time.Millisecond, config.MinSpeechDuration)

	WithMaxSilenceDuration(1 * time.Second)(config)
	assert.Equal(t, 1*time.Second, config.MaxSilenceDuration)

	WithEnablePreprocessing(false)(config)
	assert.False(t, config.EnablePreprocessing)

	WithModelPath("/path/to/model")(config)
	assert.Equal(t, "/path/to/model", config.ModelPath)
}
