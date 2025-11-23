package turndetection

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
				Provider:           "heuristic",
				MinSilenceDuration: 500 * time.Millisecond,
				MinTurnLength:      10,
				MaxTurnLength:      5000,
				Threshold:          0.5,
				Timeout:            1 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "missing provider",
			config: &Config{
				MinSilenceDuration: 500 * time.Millisecond,
			},
			wantErr: true,
		},
		{
			name: "invalid provider",
			config: &Config{
				Provider:           "invalid",
				MinSilenceDuration: 500 * time.Millisecond,
			},
			wantErr: true,
		},
		{
			name: "invalid threshold",
			config: &Config{
				Provider:  "heuristic",
				Threshold: 1.5, // Too high
			},
			wantErr: true,
		},
		{
			name: "invalid min turn length",
			config: &Config{
				Provider:      "heuristic",
				MinTurnLength: 0, // Too small
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
	assert.Equal(t, "heuristic", config.Provider)
	assert.Equal(t, 500*time.Millisecond, config.MinSilenceDuration)
	assert.Equal(t, 10, config.MinTurnLength)
	assert.Equal(t, 5000, config.MaxTurnLength)
	assert.Equal(t, ".!?", config.SentenceEndMarkers)
}

func TestConfigOption(t *testing.T) {
	config := DefaultConfig()

	WithProvider("onnx")(config)
	assert.Equal(t, "onnx", config.Provider)

	WithMinSilenceDuration(1 * time.Second)(config)
	assert.Equal(t, 1*time.Second, config.MinSilenceDuration)

	WithMinTurnLength(20)(config)
	assert.Equal(t, 20, config.MinTurnLength)

	WithMaxTurnLength(10000)(config)
	assert.Equal(t, 10000, config.MaxTurnLength)

	WithSentenceEndMarkers(".,!?")(config)
	assert.Equal(t, ".,!?", config.SentenceEndMarkers)

	WithThreshold(0.7)(config)
	assert.Equal(t, 0.7, config.Threshold)

	WithModelPath("/path/to/model")(config)
	assert.Equal(t, "/path/to/model", config.ModelPath)
}
