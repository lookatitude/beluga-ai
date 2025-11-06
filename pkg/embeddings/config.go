package embeddings

import (
	"fmt"
	"time"

	validator "github.com/go-playground/validator/v10"
)

// Option is a functional option for configuring embedders
type Option func(*optionConfig)

// optionConfig holds the internal configuration options
type optionConfig struct {
	timeout    time.Duration
	maxRetries int
	model      string
}

// WithTimeout sets the timeout for embedding operations
func WithTimeout(timeout time.Duration) Option {
	return func(c *optionConfig) {
		c.timeout = timeout
	}
}

// WithMaxRetries sets the maximum number of retries for failed operations
func WithMaxRetries(maxRetries int) Option {
	return func(c *optionConfig) {
		c.maxRetries = maxRetries
	}
}

// WithModel sets the model to use for embeddings
func WithModel(model string) Option {
	return func(c *optionConfig) {
		c.model = model
	}
}

// defaultOptionConfig returns the default option configuration
func defaultOptionConfig() *optionConfig {
	return &optionConfig{
		timeout:    30 * time.Second,
		maxRetries: 3,
		model:      "",
	}
}

// Config holds configuration for the embeddings package
type Config struct {
	// Provider-specific configurations
	OpenAI *OpenAIConfig `mapstructure:"openai" yaml:"openai"`
	Ollama *OllamaConfig `mapstructure:"ollama" yaml:"ollama"`
	Mock   *MockConfig   `mapstructure:"mock" yaml:"mock"`
}

// OpenAIConfig holds configuration for OpenAI embedding provider
type OpenAIConfig struct {
	APIKey     string        `mapstructure:"api_key" yaml:"api_key" env:"OPENAI_API_KEY" validate:"required"`
	Model      string        `mapstructure:"model" yaml:"model" env:"OPENAI_MODEL" default:"text-embedding-ada-002"`
	BaseURL    string        `mapstructure:"base_url" yaml:"base_url" env:"OPENAI_BASE_URL"`
	APIVersion string        `mapstructure:"api_version" yaml:"api_version" env:"OPENAI_API_VERSION"`
	Timeout    time.Duration `mapstructure:"timeout" yaml:"timeout" env:"OPENAI_TIMEOUT" default:"30s"`
	MaxRetries int           `mapstructure:"max_retries" yaml:"max_retries" env:"OPENAI_MAX_RETRIES" default:"3"`
	Enabled    bool          `mapstructure:"enabled" yaml:"enabled" env:"OPENAI_ENABLED" default:"true"`
}

// OllamaConfig holds configuration for Ollama embedding provider
type OllamaConfig struct {
	ServerURL  string        `mapstructure:"server_url" yaml:"server_url" env:"OLLAMA_SERVER_URL" default:"http://localhost:11434"`
	Model      string        `mapstructure:"model" yaml:"model" env:"OLLAMA_MODEL" validate:"required"`
	Timeout    time.Duration `mapstructure:"timeout" yaml:"timeout" env:"OLLAMA_TIMEOUT" default:"30s"`
	MaxRetries int           `mapstructure:"max_retries" yaml:"max_retries" env:"OLLAMA_MAX_RETRIES" default:"3"`
	KeepAlive  string        `mapstructure:"keep_alive" yaml:"keep_alive" env:"OLLAMA_KEEP_ALIVE" default:"5m"`
	Enabled    bool          `mapstructure:"enabled" yaml:"enabled" env:"OLLAMA_ENABLED" default:"true"`
}

// MockConfig holds configuration for mock embedding provider
type MockConfig struct {
	Dimension    int   `mapstructure:"dimension" yaml:"dimension" env:"MOCK_EMBEDDING_DIMENSION" default:"128"`
	Seed         int64 `mapstructure:"seed" yaml:"seed" env:"MOCK_EMBEDDING_SEED" default:"0"`
	RandomizeNil bool  `mapstructure:"randomize_nil" yaml:"randomize_nil" env:"MOCK_EMBEDDING_RANDOMIZE_NIL" default:"false"`
	Enabled      bool  `mapstructure:"enabled" yaml:"enabled" env:"MOCK_EMBEDDING_ENABLED" default:"true"`
}

// Validate validates the configuration
func (c *Config) Validate() error {
	validate := validator.New()
	if err := validate.Struct(c); err != nil {
		return err
	}

	// Validate individual provider configs
	if c.OpenAI != nil {
		if err := c.OpenAI.Validate(); err != nil {
			return err
		}
	}
	if c.Ollama != nil {
		if err := c.Ollama.Validate(); err != nil {
			return err
		}
	}
	if c.Mock != nil {
		if err := c.Mock.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// SetDefaults sets default values for the configuration
func (c *Config) SetDefaults() {
	if c.OpenAI == nil {
		c.OpenAI = &OpenAIConfig{}
	}
	if c.OpenAI.Model == "" {
		c.OpenAI.Model = "text-embedding-ada-002"
	}
	if c.OpenAI.Timeout == 0 {
		c.OpenAI.Timeout = 30 * time.Second
	}
	if c.OpenAI.MaxRetries == 0 {
		c.OpenAI.MaxRetries = 3
	}
	c.OpenAI.Enabled = true // Default to enabled if not specified

	if c.Ollama == nil {
		c.Ollama = &OllamaConfig{}
	}
	if c.Ollama.ServerURL == "" {
		c.Ollama.ServerURL = "http://localhost:11434"
	}
	if c.Ollama.Timeout == 0 {
		c.Ollama.Timeout = 30 * time.Second
	}
	if c.Ollama.MaxRetries == 0 {
		c.Ollama.MaxRetries = 3
	}
	if c.Ollama.KeepAlive == "" {
		c.Ollama.KeepAlive = "5m"
	}
	c.Ollama.Enabled = true

	if c.Mock == nil {
		c.Mock = &MockConfig{}
	}
	if c.Mock.Dimension == 0 {
		c.Mock.Dimension = 128
	}
	c.Mock.Enabled = true
}

// ValidateOpenAI validates OpenAI configuration
func (c *OpenAIConfig) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("openai API key is required")
	}
	if c.Model == "" {
		return fmt.Errorf("openai model is required")
	}
	return nil
}

// ValidateOllama validates Ollama configuration
func (c *OllamaConfig) Validate() error {
	if c.Model == "" {
		return fmt.Errorf("ollama model is required")
	}
	return nil
}

// ValidateMock validates Mock configuration
func (c *MockConfig) Validate() error {
	if c.Dimension <= 0 {
		return fmt.Errorf("mock dimension must be positive")
	}
	return nil
}
