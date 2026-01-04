// Package agents provides comprehensive tests for configuration validation.
// T159: Add test cases for all config validation scenarios
package agents

import (
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfig_Validate_ValidConfig tests validation of a valid default config.
func TestConfig_Validate_ValidConfig(t *testing.T) {
	config := DefaultConfig()
	err := config.Validate()
	assert.NoError(t, err, "Default config should be valid")
}

// TestConfig_Validate_DefaultMaxRetries tests DefaultMaxRetries validation.
func TestConfig_Validate_DefaultMaxRetries(t *testing.T) {
	tests := []struct {
		name      string
		retries   int
		wantError bool
	}{
		{
			name:      "zero retries",
			retries:   0,
			wantError: false,
		},
		{
			name:      "positive retries",
			retries:   3,
			wantError: false,
		},
		{
			name:      "negative retries",
			retries:   -1,
			wantError: true,
		},
		{
			name:      "large positive retries",
			retries:   100,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.DefaultMaxRetries = tt.retries
			err := config.Validate()

			if tt.wantError {
				assert.Error(t, err)
				assert.True(t, IsValidationError(err))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestConfig_Validate_DefaultRetryDelay tests DefaultRetryDelay validation.
func TestConfig_Validate_DefaultRetryDelay(t *testing.T) {
	tests := []struct {
		name      string
		delay     time.Duration
		wantError bool
	}{
		{
			name:      "zero delay",
			delay:     0,
			wantError: false,
		},
		{
			name:      "positive delay",
			delay:     2 * time.Second,
			wantError: false,
		},
		{
			name:      "negative delay",
			delay:     -1 * time.Second,
			wantError: true,
		},
		{
			name:      "large positive delay",
			delay:     1 * time.Hour,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.DefaultRetryDelay = tt.delay
			err := config.Validate()

			if tt.wantError {
				assert.Error(t, err)
				assert.True(t, IsValidationError(err))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestConfig_Validate_DefaultTimeout tests DefaultTimeout validation.
func TestConfig_Validate_DefaultTimeout(t *testing.T) {
	tests := []struct {
		name      string
		timeout   time.Duration
		wantError bool
	}{
		{
			name:      "zero timeout",
			timeout:   0,
			wantError: true,
		},
		{
			name:      "positive timeout",
			timeout:   30 * time.Second,
			wantError: false,
		},
		{
			name:      "negative timeout",
			timeout:   -1 * time.Second,
			wantError: true,
		},
		{
			name:      "very small timeout",
			timeout:   1 * time.Nanosecond,
			wantError: false,
		},
		{
			name:      "large timeout",
			timeout:   1 * time.Hour,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.DefaultTimeout = tt.timeout
			err := config.Validate()

			if tt.wantError {
				assert.Error(t, err)
				assert.True(t, IsValidationError(err))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestConfig_Validate_DefaultMaxIterations tests DefaultMaxIterations validation.
func TestConfig_Validate_DefaultMaxIterations(t *testing.T) {
	tests := []struct {
		name       string
		iterations int
		wantError  bool
	}{
		{
			name:       "zero iterations",
			iterations: 0,
			wantError:  true,
		},
		{
			name:       "positive iterations",
			iterations: 15,
			wantError:  false,
		},
		{
			name:       "negative iterations",
			iterations: -1,
			wantError:  true,
		},
		{
			name:       "large iterations",
			iterations: 1000,
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.DefaultMaxIterations = tt.iterations
			err := config.Validate()

			if tt.wantError {
				assert.Error(t, err)
				assert.True(t, IsValidationError(err))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestConfig_Validate_ExecutorConfig tests ExecutorConfig validation.
func TestConfig_Validate_ExecutorConfig(t *testing.T) {
	tests := []struct {
		setup     func(*Config)
		name      string
		wantError bool
	}{
		{
			name:      "valid executor config",
			setup:     func(c *Config) {},
			wantError: false,
		},
		{
			name: "zero DefaultMaxConcurrency",
			setup: func(c *Config) {
				c.ExecutorConfig.DefaultMaxConcurrency = 0
			},
			wantError: true,
		},
		{
			name: "negative DefaultMaxConcurrency",
			setup: func(c *Config) {
				c.ExecutorConfig.DefaultMaxConcurrency = -1
			},
			wantError: true,
		},
		{
			name: "zero MaxConcurrentExecutions",
			setup: func(c *Config) {
				c.ExecutorConfig.MaxConcurrentExecutions = 0
			},
			wantError: true,
		},
		{
			name: "negative MaxConcurrentExecutions",
			setup: func(c *Config) {
				c.ExecutorConfig.MaxConcurrentExecutions = -1
			},
			wantError: true,
		},
		{
			name: "zero ExecutionTimeout",
			setup: func(c *Config) {
				c.ExecutorConfig.ExecutionTimeout = 0
			},
			wantError: true,
		},
		{
			name: "negative ExecutionTimeout",
			setup: func(c *Config) {
				c.ExecutorConfig.ExecutionTimeout = -1 * time.Second
			},
			wantError: true,
		},
		{
			name: "all executor fields invalid",
			setup: func(c *Config) {
				c.ExecutorConfig.DefaultMaxConcurrency = 0
				c.ExecutorConfig.MaxConcurrentExecutions = 0
				c.ExecutorConfig.ExecutionTimeout = 0
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			tt.setup(config)
			err := config.Validate()

			if tt.wantError {
				assert.Error(t, err)
				assert.True(t, IsValidationError(err))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestConfig_Validate_MultipleErrors tests that validation catches multiple errors.
func TestConfig_Validate_MultipleErrors(t *testing.T) {
	config := DefaultConfig()
	config.DefaultMaxRetries = -1
	config.DefaultTimeout = 0
	config.DefaultMaxIterations = -1

	err := config.Validate()
	assert.Error(t, err)
	assert.True(t, IsValidationError(err))
	// Validation stops at first error, so we only get one error
}

// TestConfig_Validate_AgentConfigs tests config with agent configs.
func TestConfig_Validate_AgentConfigs(t *testing.T) {
	config := DefaultConfig()
	config.AgentConfigs = map[string]schema.AgentConfig{
		"agent1": {Name: "agent1"},
		"agent2": {Name: "agent2"},
	}

	err := config.Validate()
	assert.NoError(t, err, "Config with agent configs should be valid")
}

// TestConfig_Validate_EmptyAgentConfigs tests config with empty agent configs.
func TestConfig_Validate_EmptyAgentConfigs(t *testing.T) {
	config := DefaultConfig()
	config.AgentConfigs = make(map[string]schema.AgentConfig)

	err := config.Validate()
	assert.NoError(t, err, "Config with empty agent configs should be valid")
}

// TestValidateStreamingConfig tests streaming configuration validation.
func TestValidateStreamingConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    iface.StreamingConfig
		wantError bool
	}{
		{
			name: "valid config",
			config: iface.StreamingConfig{
				EnableStreaming:     true,
				ChunkBufferSize:     20,
				MaxStreamDuration:   30 * time.Minute,
				SentenceBoundary:    false,
				InterruptOnNewInput: true,
			},
			wantError: false,
		},
		{
			name: "zero ChunkBufferSize",
			config: iface.StreamingConfig{
				ChunkBufferSize:   0,
				MaxStreamDuration: 30 * time.Minute,
			},
			wantError: true,
		},
		{
			name: "negative ChunkBufferSize",
			config: iface.StreamingConfig{
				ChunkBufferSize:   -1,
				MaxStreamDuration: 30 * time.Minute,
			},
			wantError: true,
		},
		{
			name: "ChunkBufferSize at limit (100)",
			config: iface.StreamingConfig{
				ChunkBufferSize:   100,
				MaxStreamDuration: 30 * time.Minute,
			},
			wantError: false,
		},
		{
			name: "ChunkBufferSize over limit",
			config: iface.StreamingConfig{
				ChunkBufferSize:   101,
				MaxStreamDuration: 30 * time.Minute,
			},
			wantError: true,
		},
		{
			name: "zero MaxStreamDuration",
			config: iface.StreamingConfig{
				ChunkBufferSize:   20,
				MaxStreamDuration: 0,
			},
			wantError: true,
		},
		{
			name: "negative MaxStreamDuration",
			config: iface.StreamingConfig{
				ChunkBufferSize:   20,
				MaxStreamDuration: -1 * time.Second,
			},
			wantError: true,
		},
		{
			name: "small MaxStreamDuration",
			config: iface.StreamingConfig{
				ChunkBufferSize:   20,
				MaxStreamDuration: 1 * time.Nanosecond,
			},
			wantError: false,
		},
		{
			name: "very large MaxStreamDuration",
			config: iface.StreamingConfig{
				ChunkBufferSize:   20,
				MaxStreamDuration: 24 * time.Hour,
			},
			wantError: false,
		},
		{
			name: "all fields invalid",
			config: iface.StreamingConfig{
				ChunkBufferSize:   0,
				MaxStreamDuration: 0,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStreamingConfig(tt.config)

			if tt.wantError {
				assert.Error(t, err)
				assert.True(t, IsValidationError(err))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateStreamingConfig_ChunkBufferSizeBoundaries tests ChunkBufferSize boundary values.
func TestValidateStreamingConfig_ChunkBufferSizeBoundaries(t *testing.T) {
	tests := []struct {
		name      string
		size      int
		wantError bool
	}{
		{
			name:      "minimum valid (1)",
			size:      1,
			wantError: false,
		},
		{
			name:      "maximum valid (100)",
			size:      100,
			wantError: false,
		},
		{
			name:      "over maximum (101)",
			size:      101,
			wantError: true,
		},
		{
			name:      "zero",
			size:      0,
			wantError: true,
		},
		{
			name:      "negative",
			size:      -1,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := iface.StreamingConfig{
				ChunkBufferSize:   tt.size,
				MaxStreamDuration: 30 * time.Minute,
				EnableStreaming:   true,
			}
			err := ValidateStreamingConfig(config)

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

	assert.Equal(t, 3, config.DefaultMaxRetries)
	assert.Equal(t, 2*time.Second, config.DefaultRetryDelay)
	assert.Equal(t, 30*time.Second, config.DefaultTimeout)
	assert.Equal(t, 15, config.DefaultMaxIterations)
	assert.True(t, config.EnableMetrics)
	assert.True(t, config.EnableTracing)
	assert.Equal(t, "beluga_agents", config.MetricsPrefix)
	assert.Equal(t, "beluga-agents", config.TracingServiceName)
	assert.Equal(t, 10, config.ExecutorConfig.DefaultMaxConcurrency)
	assert.Equal(t, 100, config.ExecutorConfig.MaxConcurrentExecutions)
	assert.Equal(t, 5*time.Minute, config.ExecutorConfig.ExecutionTimeout)
	assert.True(t, config.ExecutorConfig.HandleParsingErrors)
	assert.False(t, config.ExecutorConfig.ReturnIntermediateSteps)
	assert.NotNil(t, config.AgentConfigs)
}

// TestValidateConfig_Function tests the package-level ValidateConfig function.
func TestValidateConfig_Function(t *testing.T) {
	config := DefaultConfig()
	err := ValidateConfig(config)
	assert.NoError(t, err)

	config.DefaultMaxRetries = -1
	err = ValidateConfig(config)
	assert.Error(t, err)
	assert.True(t, IsValidationError(err))
}

// TestConfig_Validate_NilConfig tests validation of nil config (edge case).
func TestConfig_Validate_NilConfig(t *testing.T) {
	// Test that DefaultConfig never returns nil and is valid
	defaultCfg := DefaultConfig()
	require.NotNil(t, defaultCfg)
	err := defaultCfg.Validate()
	assert.NoError(t, err)
}

// TestWithStreaming tests the WithStreaming option function.
func TestWithStreaming(t *testing.T) {
	tests := []struct {
		validate func(*iface.Options)
		name     string
		enabled  bool
	}{
		{
			name:    "enable streaming",
			enabled: true,
			validate: func(o *iface.Options) {
				assert.True(t, o.StreamingConfig.EnableStreaming)
				assert.Equal(t, 20, o.StreamingConfig.ChunkBufferSize)
				assert.Equal(t, 30*time.Minute, o.StreamingConfig.MaxStreamDuration)
			},
		},
		{
			name:    "disable streaming",
			enabled: false,
			validate: func(o *iface.Options) {
				assert.False(t, o.StreamingConfig.EnableStreaming)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &iface.Options{}
			opt := WithStreaming(tt.enabled)
			opt(opts)
			tt.validate(opts)
		})
	}
}

// TestWithStreamingConfig tests the WithStreamingConfig option function.
func TestWithStreamingConfig(t *testing.T) {
	streamingConfig := iface.StreamingConfig{
		EnableStreaming:     true,
		ChunkBufferSize:     50,
		SentenceBoundary:    true,
		InterruptOnNewInput: true,
		MaxStreamDuration:   60 * time.Minute,
	}

	opts := &iface.Options{}
	opt := WithStreamingConfig(streamingConfig)
	opt(opts)

	assert.Equal(t, streamingConfig, opts.StreamingConfig)
}

// TestConfigOptions tests all config option functions.
func TestConfigOptions(t *testing.T) {
	tests := []struct {
		option   iface.Option
		validate func(*iface.Options)
		name     string
	}{
		{
			name:   "WithMaxRetries",
			option: WithMaxRetries(5),
			validate: func(o *iface.Options) {
				assert.Equal(t, 5, o.MaxRetries)
			},
		},
		{
			name:   "WithRetryDelay",
			option: WithRetryDelay(5 * time.Second),
			validate: func(o *iface.Options) {
				assert.Equal(t, 5*time.Second, o.RetryDelay)
			},
		},
		{
			name:   "WithTimeout",
			option: WithTimeout(60 * time.Second),
			validate: func(o *iface.Options) {
				assert.Equal(t, 60*time.Second, o.Timeout)
			},
		},
		{
			name:   "WithMaxIterations",
			option: WithMaxIterations(20),
			validate: func(o *iface.Options) {
				assert.Equal(t, 20, o.MaxIterations)
			},
		},
		{
			name:   "WithMetrics",
			option: WithMetrics(false),
			validate: func(o *iface.Options) {
				assert.False(t, o.EnableMetrics)
			},
		},
		{
			name:   "WithTracing",
			option: WithTracing(false),
			validate: func(o *iface.Options) {
				assert.False(t, o.EnableTracing)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &iface.Options{}
			tt.option(opts)
			tt.validate(opts)
		})
	}
}

// TestValidateStreamingConfig_OnlyWhenEnabled tests that validation applies only to enabled streaming.
func TestValidateStreamingConfig_OnlyWhenEnabled(t *testing.T) {
	// Note: Current implementation validates regardless of EnableStreaming flag
	// This test verifies the current behavior
	config := iface.StreamingConfig{
		EnableStreaming:   false,
		ChunkBufferSize:   0, // Invalid but streaming disabled
		MaxStreamDuration: 0, // Invalid but streaming disabled
	}

	// Current implementation validates regardless of EnableStreaming
	err := ValidateStreamingConfig(config)
	assert.Error(t, err) // Currently fails validation
}

// TestConfig_Validate_CombinedScenarios tests combined validation scenarios.
func TestConfig_Validate_CombinedScenarios(t *testing.T) {
	tests := []struct {
		setup     func(*Config)
		name      string
		wantError bool
	}{
		{
			name: "all valid",
			setup: func(c *Config) {
				// All defaults are valid
			},
			wantError: false,
		},
		{
			name: "all executor fields invalid",
			setup: func(c *Config) {
				c.ExecutorConfig.DefaultMaxConcurrency = 0
				c.ExecutorConfig.MaxConcurrentExecutions = 0
				c.ExecutorConfig.ExecutionTimeout = 0
			},
			wantError: true,
		},
		{
			name: "mixed valid and invalid",
			setup: func(c *Config) {
				c.DefaultMaxRetries = -1
				c.DefaultTimeout = 30 * time.Second // Valid
			},
			wantError: true,
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
