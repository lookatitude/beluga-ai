package noise

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
				Provider:            "rnnoise",
				NoiseReductionLevel: 0.5,
				SampleRate:          16000,
				FrameSize:           480,
				Timeout:             1 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "missing provider",
			config: &Config{
				NoiseReductionLevel: 0.5,
			},
			wantErr: true,
		},
		{
			name: "invalid provider",
			config: &Config{
				Provider:            "invalid",
				NoiseReductionLevel: 0.5,
			},
			wantErr: true,
		},
		{
			name: "invalid noise reduction level",
			config: &Config{
				Provider:            "rnnoise",
				NoiseReductionLevel: 1.5, // Too high
			},
			wantErr: true,
		},
		{
			name: "invalid frame size",
			config: &Config{
				Provider:  "rnnoise",
				FrameSize: 10000, // Too large
			},
			wantErr: true,
		},
		{
			name: "invalid sample rate",
			config: &Config{
				Provider:   "rnnoise",
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
	assert.Equal(t, "rnnoise", config.Provider)
	assert.InEpsilon(t, 0.5, config.NoiseReductionLevel, 0.0001)
	assert.Equal(t, 16000, config.SampleRate)
	assert.Equal(t, 480, config.FrameSize)
	assert.True(t, config.EnableAdaptiveProcessing)
}

func TestConfigOption(t *testing.T) {
	config := DefaultConfig()

	WithProvider("webrtc")(config)
	assert.Equal(t, "webrtc", config.Provider)

	WithNoiseReductionLevel(0.7)(config)
	assert.InEpsilon(t, 0.7, config.NoiseReductionLevel, 0.0001)

	WithFrameSize(960)(config)
	assert.Equal(t, 960, config.FrameSize)

	WithSampleRate(48000)(config)
	assert.Equal(t, 48000, config.SampleRate)

	WithEnableAdaptiveProcessing(false)(config)
	assert.False(t, config.EnableAdaptiveProcessing)

	WithModelPath("/path/to/model")(config)
	assert.Equal(t, "/path/to/model", config.ModelPath)
}
