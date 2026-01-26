package backend

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	require.NotNil(t, cfg)
	assert.Equal(t, vbiface.PipelineTypeSTTTTS, cfg.PipelineType)
	assert.Equal(t, 500*time.Millisecond, cfg.LatencyTarget)
	assert.Equal(t, 30*time.Second, cfg.Timeout)
	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, time.Second, cfg.RetryDelay)
	assert.Equal(t, 100, cfg.MaxConcurrentSessions)
	assert.True(t, cfg.EnableTracing)
	assert.True(t, cfg.EnableMetrics)
	assert.True(t, cfg.EnableStructuredLogging)
	assert.NotNil(t, cfg.ProviderConfig)
	assert.NotNil(t, cfg.CustomProcessors)
}

func TestValidateConfig(t *testing.T) {
	// Helper to create a valid base config
	validBaseConfig := func() *vbiface.Config {
		return &vbiface.Config{
			Provider:      "mock",
			PipelineType:  vbiface.PipelineTypeSTTTTS,
			STTProvider:   "openai",
			TTSProvider:   "openai",
			LatencyTarget: 500 * time.Millisecond,
			Timeout:       30 * time.Second,
			RetryDelay:    time.Second,
		}
	}

	tests := []struct {
		name      string
		config    *vbiface.Config
		wantError bool
		errMsg    string
	}{
		{
			name:      "nil config",
			config:    nil,
			wantError: true,
			errMsg:    "cannot be nil",
		},
		{
			name:      "valid STT_TTS config",
			config:    validBaseConfig(),
			wantError: false,
		},
		{
			name: "valid S2S config",
			config: func() *vbiface.Config {
				cfg := validBaseConfig()
				cfg.PipelineType = vbiface.PipelineTypeS2S
				cfg.S2SProvider = "openai_realtime"
				cfg.STTProvider = ""
				cfg.TTSProvider = ""
				return cfg
			}(),
			wantError: false,
		},
		{
			name: "STT_TTS missing STT provider",
			config: func() *vbiface.Config {
				cfg := validBaseConfig()
				cfg.STTProvider = ""
				return cfg
			}(),
			wantError: true,
			errMsg:    "STTProvider",
		},
		{
			name: "STT_TTS missing TTS provider",
			config: func() *vbiface.Config {
				cfg := validBaseConfig()
				cfg.TTSProvider = ""
				return cfg
			}(),
			wantError: true,
			errMsg:    "TTSProvider",
		},
		{
			name: "S2S missing S2S provider",
			config: func() *vbiface.Config {
				cfg := validBaseConfig()
				cfg.PipelineType = vbiface.PipelineTypeS2S
				cfg.STTProvider = ""
				cfg.TTSProvider = ""
				return cfg
			}(),
			wantError: true,
			errMsg:    "S2SProvider",
		},
		{
			name: "missing provider",
			config: func() *vbiface.Config {
				cfg := validBaseConfig()
				cfg.Provider = ""
				return cfg
			}(),
			wantError: true,
			errMsg:    "Provider",
		},
		{
			name: "latency target too low",
			config: func() *vbiface.Config {
				cfg := validBaseConfig()
				cfg.LatencyTarget = 50 * time.Millisecond
				return cfg
			}(),
			wantError: true,
			errMsg:    "LatencyTarget",
		},
		{
			name: "latency target too high",
			config: func() *vbiface.Config {
				cfg := validBaseConfig()
				cfg.LatencyTarget = 10 * time.Second
				return cfg
			}(),
			wantError: true,
			errMsg:    "LatencyTarget",
		},
		{
			name: "timeout too low",
			config: func() *vbiface.Config {
				cfg := validBaseConfig()
				cfg.Timeout = 500 * time.Millisecond
				return cfg
			}(),
			wantError: true,
			errMsg:    "Timeout",
		},
		{
			name: "timeout too high",
			config: func() *vbiface.Config {
				cfg := validBaseConfig()
				cfg.Timeout = 10 * time.Minute
				return cfg
			}(),
			wantError: true,
			errMsg:    "Timeout",
		},
		{
			name: "retry delay too low",
			config: func() *vbiface.Config {
				cfg := validBaseConfig()
				cfg.RetryDelay = 50 * time.Millisecond
				return cfg
			}(),
			wantError: true,
			errMsg:    "RetryDelay",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfigOptions(t *testing.T) {
	cfg := DefaultConfig()

	// Test WithProvider
	WithProvider("livekit")(cfg)
	assert.Equal(t, "livekit", cfg.Provider)

	// Test WithPipelineType
	WithPipelineType(vbiface.PipelineTypeS2S)(cfg)
	assert.Equal(t, vbiface.PipelineTypeS2S, cfg.PipelineType)

	// Test WithLatencyTarget
	WithLatencyTarget(200 * time.Millisecond)(cfg)
	assert.Equal(t, 200*time.Millisecond, cfg.LatencyTarget)

	// Test WithMaxConcurrentSessions
	WithMaxConcurrentSessions(50)(cfg)
	assert.Equal(t, 50, cfg.MaxConcurrentSessions)
}

func TestValidateSessionConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *vbiface.SessionConfig
		wantError bool
		errMsg    string
	}{
		{
			name:      "nil config",
			config:    nil,
			wantError: true,
			errMsg:    "cannot be nil",
		},
		{
			name: "valid with callback",
			config: &vbiface.SessionConfig{
				AgentCallback: func(ctx context.Context, transcript string) (string, error) { return "response", nil },
				UserID:        "user-123",
				Transport:     "webrtc",
				ConnectionURL: "https://example.com",
				PipelineType:  vbiface.PipelineTypeSTTTTS,
			},
			wantError: false,
		},
		{
			name: "missing both callback and instance",
			config: &vbiface.SessionConfig{
				UserID:        "user-123",
				Transport:     "webrtc",
				ConnectionURL: "https://example.com",
				PipelineType:  vbiface.PipelineTypeSTTTTS,
			},
			wantError: true,
			errMsg:    "either agent_callback or agent_instance must be set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSessionConfig(tt.config)
			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
