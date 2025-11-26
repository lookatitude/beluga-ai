package session

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
				SessionID:         "test-session",
				Timeout:           30 * time.Minute,
				KeepAliveInterval: 30 * time.Second,
				MaxRetries:        3,
				RetryDelay:        1 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "invalid timeout",
			config: &Config{
				Timeout: 25 * time.Hour, // Too long
			},
			wantErr: true,
		},
		{
			name: "invalid keep-alive interval",
			config: &Config{
				KeepAliveInterval: 10 * time.Minute, // Too long
			},
			wantErr: true,
		},
		{
			name: "invalid max retries",
			config: &Config{
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
	assert.Equal(t, 30*time.Minute, config.Timeout)
	assert.False(t, config.AutoStart)
	assert.True(t, config.EnableKeepAlive)
	assert.Equal(t, 30*time.Second, config.KeepAliveInterval)
	assert.Equal(t, 3, config.MaxRetries)
}

func TestConfigOption(t *testing.T) {
	config := DefaultConfig()

	WithSessionID("custom-id")(config)
	assert.Equal(t, "custom-id", config.SessionID)

	WithTimeout(1 * time.Hour)(config)
	assert.Equal(t, 1*time.Hour, config.Timeout)

	WithAutoStart(true)(config)
	assert.True(t, config.AutoStart)

	WithEnableKeepAlive(false)(config)
	assert.False(t, config.EnableKeepAlive)

	WithKeepAliveInterval(1 * time.Minute)(config)
	assert.Equal(t, 1*time.Minute, config.KeepAliveInterval)

	WithMaxRetries(5)(config)
	assert.Equal(t, 5, config.MaxRetries)
}
