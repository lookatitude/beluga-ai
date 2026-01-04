package session

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
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

// TestConfig_Validate_Timeout tests Timeout validation boundaries.
func TestConfig_Validate_Timeout(t *testing.T) {
	tests := []struct {
		name      string
		timeout   time.Duration
		wantError bool
	}{
		{
			name:      "minimum valid timeout (1 minute)",
			timeout:   1 * time.Minute,
			wantError: false,
		},
		{
			name:      "maximum valid timeout (24 hours)",
			timeout:   24 * time.Hour,
			wantError: false,
		},
		{
			name:      "timeout below minimum",
			timeout:   59 * time.Second,
			wantError: true,
		},
		{
			name:      "timeout above maximum",
			timeout:   25 * time.Hour,
			wantError: true,
		},
		{
			name:      "zero timeout",
			timeout:   0,
			wantError: true,
		},
		{
			name:      "valid timeout in middle range",
			timeout:   1 * time.Hour,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.Timeout = tt.timeout
			err := config.Validate()

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestConfig_Validate_KeepAliveInterval tests KeepAliveInterval validation boundaries.
func TestConfig_Validate_KeepAliveInterval(t *testing.T) {
	tests := []struct {
		name            string
		keepAliveInterval time.Duration
		wantError       bool
	}{
		{
			name:            "minimum valid keep-alive interval (5 seconds)",
			keepAliveInterval: 5 * time.Second,
			wantError:       false,
		},
		{
			name:            "maximum valid keep-alive interval (5 minutes)",
			keepAliveInterval: 5 * time.Minute,
			wantError:       false,
		},
		{
			name:            "keep-alive interval below minimum",
			keepAliveInterval: 4 * time.Second,
			wantError:       true,
		},
		{
			name:            "keep-alive interval above maximum",
			keepAliveInterval: 6 * time.Minute,
			wantError:       true,
		},
		{
			name:            "zero keep-alive interval",
			keepAliveInterval: 0,
			wantError:       true,
		},
		{
			name:            "valid keep-alive interval in middle range",
			keepAliveInterval: 1 * time.Minute,
			wantError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.KeepAliveInterval = tt.keepAliveInterval
			err := config.Validate()

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestConfig_Validate_MaxRetries tests MaxRetries validation boundaries.
func TestConfig_Validate_MaxRetries(t *testing.T) {
	tests := []struct {
		name      string
		maxRetries int
		wantError bool
	}{
		{
			name:      "minimum valid max retries (0)",
			maxRetries: 0,
			wantError: false,
		},
		{
			name:      "maximum valid max retries (10)",
			maxRetries: 10,
			wantError: false,
		},
		{
			name:      "max retries below minimum",
			maxRetries: -1,
			wantError: true,
		},
		{
			name:      "max retries above maximum",
			maxRetries: 11,
			wantError: true,
		},
		{
			name:      "valid max retries in middle range",
			maxRetries: 5,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.MaxRetries = tt.maxRetries
			err := config.Validate()

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestConfig_Validate_RetryDelay tests RetryDelay validation boundaries.
func TestConfig_Validate_RetryDelay(t *testing.T) {
	tests := []struct {
		name      string
		retryDelay time.Duration
		wantError bool
	}{
		{
			name:      "minimum valid retry delay (100ms)",
			retryDelay: 100 * time.Millisecond,
			wantError: false,
		},
		{
			name:      "maximum valid retry delay (10 seconds)",
			retryDelay: 10 * time.Second,
			wantError: false,
		},
		{
			name:      "retry delay below minimum",
			retryDelay: 99 * time.Millisecond,
			wantError: true,
		},
		{
			name:      "retry delay above maximum",
			retryDelay: 11 * time.Second,
			wantError: true,
		},
		{
			name:      "zero retry delay",
			retryDelay: 0,
			wantError: true,
		},
		{
			name:      "valid retry delay in middle range",
			retryDelay: 1 * time.Second,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.RetryDelay = tt.retryDelay
			err := config.Validate()

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestConfig_Validate_CombinedScenarios tests multiple validation failures.
func TestConfig_Validate_CombinedScenarios(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*Config)
		wantError bool
	}{
		{
			name: "all fields valid",
			setup: func(c *Config) {
				// Default config is valid
			},
			wantError: false,
		},
		{
			name: "multiple invalid fields",
			setup: func(c *Config) {
				c.Timeout = 0
				c.MaxRetries = -1
				c.KeepAliveInterval = 0
			},
			wantError: true,
		},
		{
			name: "boundary values at limits",
			setup: func(c *Config) {
				c.Timeout = 24 * time.Hour
				c.KeepAliveInterval = 5 * time.Minute
				c.MaxRetries = 10
				c.RetryDelay = 10 * time.Second
			},
			wantError: false,
		},
		{
			name: "boundary values at minimums",
			setup: func(c *Config) {
				c.Timeout = 1 * time.Minute
				c.KeepAliveInterval = 5 * time.Second
				c.MaxRetries = 0
				c.RetryDelay = 100 * time.Millisecond
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			tt.setup(config)
			err := config.Validate()

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestDefaultConfig_AllFields tests that DefaultConfig sets all fields correctly.
func TestDefaultConfig_AllFields(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, "", config.SessionID)
	assert.Equal(t, 30*time.Minute, config.Timeout)
	assert.Equal(t, 30*time.Second, config.KeepAliveInterval)
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 1*time.Second, config.RetryDelay)
	assert.False(t, config.AutoStart)
	assert.True(t, config.EnableKeepAlive)
	assert.True(t, config.EnableTracing)
	assert.True(t, config.EnableMetrics)
	assert.True(t, config.EnableStructuredLogging)
}

// TestVoiceOptions_WithAgentInstance tests that WithAgentInstance sets agent instance and config correctly.
func TestVoiceOptions_WithAgentInstance(t *testing.T) {
	// Create a mock streaming agent
	mockAgent := &mockStreamingAgentForConfig{}

	agentConfig := &schema.AgentConfig{
		Name:            "test-agent",
		LLMProviderName: "mock",
	}

	opts := &VoiceOptions{}
	WithAgentInstance(mockAgent, agentConfig)(opts)

	assert.Equal(t, mockAgent, opts.AgentInstance)
	assert.Equal(t, agentConfig, opts.AgentConfig)
}

// TestVoiceOptions_WithAgentInstance_NilConfig tests that WithAgentInstance works with nil config.
func TestVoiceOptions_WithAgentInstance_NilConfig(t *testing.T) {
	mockAgent := &mockStreamingAgentForConfig{}

	opts := &VoiceOptions{}
	WithAgentInstance(mockAgent, nil)(opts)

	assert.Equal(t, mockAgent, opts.AgentInstance)
	assert.Nil(t, opts.AgentConfig)
}

// TestVoiceOptions_AgentInstanceValidation tests that AgentInstance validation works correctly.
// Note: Actual validation happens in session_impl.go during session creation.
func TestVoiceOptions_AgentInstanceValidation(t *testing.T) {
	t.Run("valid agent instance", func(t *testing.T) {
		mockAgent := &mockStreamingAgentForConfig{}
		agentConfig := &schema.AgentConfig{Name: "test"}

		opts := &VoiceOptions{}
		WithAgentInstance(mockAgent, agentConfig)(opts)

		// Validation happens in NewVoiceSessionImpl, but we can verify the option was set
		assert.NotNil(t, opts.AgentInstance)
		assert.NotNil(t, opts.AgentConfig)
	})

	t.Run("nil agent instance", func(t *testing.T) {
		opts := &VoiceOptions{}
		WithAgentInstance(nil, nil)(opts)

		assert.Nil(t, opts.AgentInstance)
		assert.Nil(t, opts.AgentConfig)
	})
}

// TestVoiceOptions_DefaultAgentConfig tests default agent configuration values.
func TestVoiceOptions_DefaultAgentConfig(t *testing.T) {
	t.Run("with provided config", func(t *testing.T) {
		mockAgent := &mockStreamingAgentForConfig{}
		agentConfig := &schema.AgentConfig{
			Name:            "custom-agent",
			LLMProviderName: "custom-llm",
		}

		opts := &VoiceOptions{}
		WithAgentInstance(mockAgent, agentConfig)(opts)

		assert.Equal(t, "custom-agent", opts.AgentConfig.Name)
		assert.Equal(t, "custom-llm", opts.AgentConfig.LLMProviderName)
	})

	t.Run("with nil config uses defaults in session_impl", func(t *testing.T) {
		mockAgent := &mockStreamingAgentForConfig{}

		opts := &VoiceOptions{}
		WithAgentInstance(mockAgent, nil)(opts)

		// Config is nil, but session_impl.go will create a default config
		assert.Nil(t, opts.AgentConfig)
	})
}

// mockStreamingAgentForConfig is a minimal mock for testing config validation.
type mockStreamingAgentForConfig struct{}

func (m *mockStreamingAgentForConfig) StreamExecute(ctx context.Context, inputs map[string]any) (<-chan iface.AgentStreamChunk, error) {
	return nil, nil
}

func (m *mockStreamingAgentForConfig) StreamPlan(ctx context.Context, intermediateSteps []iface.IntermediateStep, inputs map[string]any) (<-chan iface.AgentStreamChunk, error) {
	return nil, nil
}

// Add other required methods to satisfy the interface
func (m *mockStreamingAgentForConfig) GetConfig() schema.AgentConfig {
	return schema.AgentConfig{Name: "mock"}
}

func (m *mockStreamingAgentForConfig) GetTools() []tools.Tool {
	return nil
}

func (m *mockStreamingAgentForConfig) GetMetrics() iface.MetricsRecorder {
	return nil
}

func (m *mockStreamingAgentForConfig) GetLLM() llmsiface.LLM {
	return nil
}

func (m *mockStreamingAgentForConfig) InputVariables() []string {
	return []string{"input"}
}

func (m *mockStreamingAgentForConfig) OutputVariables() []string {
	return []string{"output"}
}

func (m *mockStreamingAgentForConfig) Execute(ctx context.Context, inputs map[string]any, options ...iface.Option) (any, error) {
	return nil, nil
}

func (m *mockStreamingAgentForConfig) Plan(ctx context.Context, intermediateSteps []iface.IntermediateStep, inputs map[string]any) (iface.AgentAction, iface.AgentFinish, error) {
	return iface.AgentAction{}, iface.AgentFinish{}, nil
}
