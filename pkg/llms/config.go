package llms

import (
	"context"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/core"
)

// Config represents the configuration for LLM providers.
// It includes common settings that apply to all LLM providers.
type Config struct {
	// Provider specifies the LLM provider (e.g., "openai", "anthropic", "bedrock")
	Provider string `mapstructure:"provider" yaml:"provider" validate:"required,oneof=openai anthropic bedrock gemini ollama mock"`

	// ModelName specifies the model to use (e.g., "gpt-4", "claude-3-sonnet")
	ModelName string `mapstructure:"model_name" yaml:"model_name" validate:"required"`

	// APIKey for authentication (required for most providers)
	APIKey string `mapstructure:"api_key" yaml:"api_key" validate:"required_unless=Provider mock"`

	// BaseURL for custom API endpoints (optional)
	BaseURL string `mapstructure:"base_url" yaml:"base_url"`

	// Timeout for API calls
	Timeout time.Duration `mapstructure:"timeout" yaml:"timeout" default:"30s" validate:"min=1s,max=5m"`

	// Default generation parameters
	Temperature      *float32 `mapstructure:"temperature" yaml:"temperature" validate:"omitempty,gte=0,lte=2"`
	TopP             *float32 `mapstructure:"top_p" yaml:"top_p" validate:"omitempty,gte=0,lte=1"`
	TopK             *int     `mapstructure:"top_k" yaml:"top_k" validate:"omitempty,gte=1,lte=100"`
	MaxTokens        *int     `mapstructure:"max_tokens" yaml:"max_tokens" validate:"omitempty,gte=1,lte=32768"`
	StopSequences    []string `mapstructure:"stop_sequences" yaml:"stop_sequences"`
	FrequencyPenalty *float32 `mapstructure:"frequency_penalty" yaml:"frequency_penalty" validate:"omitempty,gte=-2,lte=2"`
	PresencePenalty  *float32 `mapstructure:"presence_penalty" yaml:"presence_penalty" validate:"omitempty,gte=-2,lte=2"`

	// Streaming configuration
	EnableStreaming bool `mapstructure:"enable_streaming" yaml:"enable_streaming" default:"true"`

	// Concurrency and batching
	MaxConcurrentBatches int `mapstructure:"max_concurrent_batches" yaml:"max_concurrent_batches" default:"5" validate:"gte=1,lte=100"`

	// Retry configuration
	MaxRetries   int           `mapstructure:"max_retries" yaml:"max_retries" default:"3" validate:"gte=0,lte=10"`
	RetryDelay   time.Duration `mapstructure:"retry_delay" yaml:"retry_delay" default:"1s" validate:"min=100ms,max=30s"`
	RetryBackoff float64       `mapstructure:"retry_backoff" yaml:"retry_backoff" default:"2.0" validate:"gte=1,lte=5"`

	// Provider-specific configuration
	ProviderSpecific map[string]interface{} `mapstructure:"provider_specific" yaml:"provider_specific"`

	// Observability settings
	EnableTracing           bool `mapstructure:"enable_tracing" yaml:"enable_tracing" default:"true"`
	EnableMetrics           bool `mapstructure:"enable_metrics" yaml:"enable_metrics" default:"true"`
	EnableStructuredLogging bool `mapstructure:"enable_structured_logging" yaml:"enable_structured_logging" default:"true"`

	// Tool calling configuration
	EnableToolCalling bool `mapstructure:"enable_tool_calling" yaml:"enable_tool_calling" default:"true"`
}

// ConfigOption is a functional option for configuring LLM instances
type ConfigOption func(*Config)

// WithProvider sets the LLM provider
func WithProvider(provider string) ConfigOption {
	return func(c *Config) {
		c.Provider = provider
	}
}

// WithModelName sets the model name
func WithModelName(modelName string) ConfigOption {
	return func(c *Config) {
		c.ModelName = modelName
	}
}

// WithAPIKey sets the API key
func WithAPIKey(apiKey string) ConfigOption {
	return func(c *Config) {
		c.APIKey = apiKey
	}
}

// WithBaseURL sets the base URL
func WithBaseURL(baseURL string) ConfigOption {
	return func(c *Config) {
		c.BaseURL = baseURL
	}
}

// WithTimeout sets the timeout
func WithTimeout(timeout time.Duration) ConfigOption {
	return func(c *Config) {
		c.Timeout = timeout
	}
}

// WithTemperatureConfig sets the temperature
func WithTemperatureConfig(temp float32) ConfigOption {
	return func(c *Config) {
		c.Temperature = &temp
	}
}

// WithTopPConfig sets the top-p value
func WithTopPConfig(topP float32) ConfigOption {
	return func(c *Config) {
		c.TopP = &topP
	}
}

// WithTopKConfig sets the top-k value
func WithTopKConfig(topK int) ConfigOption {
	return func(c *Config) {
		c.TopK = &topK
	}
}

// WithMaxTokensConfig sets the maximum tokens
func WithMaxTokensConfig(maxTokens int) ConfigOption {
	return func(c *Config) {
		c.MaxTokens = &maxTokens
	}
}

// WithStopSequences sets the stop sequences
func WithStopSequences(sequences []string) ConfigOption {
	return func(c *Config) {
		c.StopSequences = sequences
	}
}

// WithMaxConcurrentBatches sets the maximum concurrent batches
func WithMaxConcurrentBatches(n int) ConfigOption {
	return func(c *Config) {
		c.MaxConcurrentBatches = n
	}
}

// WithRetryConfig sets retry configuration
func WithRetryConfig(maxRetries int, delay time.Duration, backoff float64) ConfigOption {
	return func(c *Config) {
		c.MaxRetries = maxRetries
		c.RetryDelay = delay
		c.RetryBackoff = backoff
	}
}

// WithProviderSpecific sets provider-specific configuration
func WithProviderSpecific(key string, value interface{}) ConfigOption {
	return func(c *Config) {
		if c.ProviderSpecific == nil {
			c.ProviderSpecific = make(map[string]interface{})
		}
		c.ProviderSpecific[key] = value
	}
}

// WithObservability enables or disables observability features
func WithObservability(tracing, metrics, logging bool) ConfigOption {
	return func(c *Config) {
		c.EnableTracing = tracing
		c.EnableMetrics = metrics
		c.EnableStructuredLogging = logging
	}
}

// WithToolCalling enables or disables tool calling
func WithToolCalling(enabled bool) ConfigOption {
	return func(c *Config) {
		c.EnableToolCalling = enabled
	}
}

// NewDefaultConfig returns a default configuration
func NewDefaultConfig() *Config {
	return &Config{
		Provider:                "",
		ModelName:               "",
		APIKey:                  "",
		BaseURL:                 "",
		Timeout:                 30 * time.Second,
		Temperature:             nil,
		TopP:                    nil,
		TopK:                    nil,
		MaxTokens:               nil,
		StopSequences:           nil,
		FrequencyPenalty:        nil,
		PresencePenalty:         nil,
		EnableStreaming:         true,
		MaxConcurrentBatches:    5,
		MaxRetries:              3,
		RetryDelay:              time.Second,
		RetryBackoff:            2.0,
		ProviderSpecific:        make(map[string]interface{}),
		EnableTracing:           true,
		EnableMetrics:           true,
		EnableStructuredLogging: true,
		EnableToolCalling:       true,
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	validate := validator.New()
	if err := validate.Struct(c); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	return nil
}

// MergeOptions applies functional options to the configuration
func (c *Config) MergeOptions(opts ...ConfigOption) {
	for _, opt := range opts {
		opt(c)
	}
}

// NewConfig creates a new configuration with the given options
func NewConfig(opts ...ConfigOption) *Config {
	config := NewDefaultConfig()
	config.MergeOptions(opts...)
	return config
}

// ProviderConfig represents configuration for a specific provider
type ProviderConfig struct {
	Config
	// Additional provider-specific fields can be added here
}

// ValidateProviderConfig validates a provider configuration
func ValidateProviderConfig(ctx context.Context, config *Config) error {
	if config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	// Validate using the struct validator
	if err := config.Validate(); err != nil {
		return err
	}

	// Additional context-aware validation
	if config.Timeout < time.Second {
		return fmt.Errorf("timeout must be at least 1 second, got %v", config.Timeout)
	}

	// Provider-specific validation
	switch config.Provider {
	case "openai":
		if config.ModelName == "" {
			return fmt.Errorf("model_name is required for OpenAI provider")
		}
		if config.APIKey == "" {
			return fmt.Errorf("api_key is required for OpenAI provider")
		}
	case "anthropic":
		if config.ModelName == "" {
			return fmt.Errorf("model_name is required for Anthropic provider")
		}
		if config.APIKey == "" {
			return fmt.Errorf("api_key is required for Anthropic provider")
		}
	case "mock":
		// Mock provider has minimal requirements
		if config.ModelName == "" {
			config.ModelName = "mock-model"
		}
	default:
		// Allow other providers with minimal validation
	}

	return nil
}

// CallOptions represents runtime call options for LLM invocations
type CallOptions struct {
	Temperature      *float32               `json:"temperature,omitempty"`
	TopP             *float32               `json:"top_p,omitempty"`
	TopK             *int                   `json:"top_k,omitempty"`
	MaxTokens        *int                   `json:"max_tokens,omitempty"`
	StopSequences    []string               `json:"stop_sequences,omitempty"`
	FrequencyPenalty *float32               `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float32               `json:"presence_penalty,omitempty"`
	Tools            []interface{}          `json:"tools,omitempty"` // Generic tool interface
	ToolChoice       string                 `json:"tool_choice,omitempty"`
	AdditionalArgs   map[string]interface{} `json:"additional_args,omitempty"`
}

// NewCallOptions creates new call options
func NewCallOptions() *CallOptions {
	return &CallOptions{
		AdditionalArgs: make(map[string]interface{}),
	}
}

// ApplyCallOption applies a core.Option to CallOptions
func (co *CallOptions) ApplyCallOption(opt core.Option) {
	config := make(map[string]interface{})
	opt.Apply(&config)

	for key, value := range config {
		switch key {
		case "temperature":
			if temp, ok := value.(float32); ok {
				co.Temperature = &temp
			} else if temp, ok := value.(float64); ok {
				temp32 := float32(temp)
				co.Temperature = &temp32
			}
		case "top_p":
			if topP, ok := value.(float32); ok {
				co.TopP = &topP
			} else if topP, ok := value.(float64); ok {
				topP32 := float32(topP)
				co.TopP = &topP32
			}
		case "top_k":
			if topK, ok := value.(int); ok {
				co.TopK = &topK
			}
		case "max_tokens":
			if maxTokens, ok := value.(int); ok {
				co.MaxTokens = &maxTokens
			}
		case "stop_sequences":
			if stop, ok := value.([]string); ok {
				co.StopSequences = stop
			}
		case "frequency_penalty":
			if freq, ok := value.(float32); ok {
				co.FrequencyPenalty = &freq
			} else if freq, ok := value.(float64); ok {
				freq32 := float32(freq)
				co.FrequencyPenalty = &freq32
			}
		case "presence_penalty":
			if pres, ok := value.(float32); ok {
				co.PresencePenalty = &pres
			} else if pres, ok := value.(float64); ok {
				pres32 := float32(pres)
				co.PresencePenalty = &pres32
			}
		case "tools":
			if toolList, ok := value.([]tools.Tool); ok {
				co.Tools = make([]interface{}, len(toolList))
				for i, tool := range toolList {
					co.Tools[i] = tool
				}
			}
		case "tool_choice":
			if choice, ok := value.(string); ok {
				co.ToolChoice = choice
			}
		default:
			co.AdditionalArgs[key] = value
		}
	}
}
