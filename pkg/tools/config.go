// Package tools provides configuration for tool execution.
package tools

import (
	"errors"
	"fmt"
	"time"

	validator "github.com/go-playground/validator/v10"
)

// Option is a functional option for configuring tool execution.
type Option func(*optionConfig)

// optionConfig holds the internal configuration options.
type optionConfig struct {
	timeout        time.Duration
	maxRetries     int
	maxConcurrency int
	enableMetrics  bool
}

// WithTimeout sets the timeout for tool execution.
func WithTimeout(timeout time.Duration) Option {
	return func(c *optionConfig) {
		c.timeout = timeout
	}
}

// WithMaxRetries sets the maximum number of retries for failed operations.
func WithMaxRetries(maxRetries int) Option {
	return func(c *optionConfig) {
		c.maxRetries = maxRetries
	}
}

// WithMaxConcurrency sets the maximum concurrency for batch operations.
func WithMaxConcurrency(n int) Option {
	return func(c *optionConfig) {
		c.maxConcurrency = n
	}
}

// WithMetrics enables or disables metrics collection.
func WithMetrics(enabled bool) Option {
	return func(c *optionConfig) {
		c.enableMetrics = enabled
	}
}

// defaultOptionConfig returns the default option configuration.
func defaultOptionConfig() *optionConfig {
	return &optionConfig{
		timeout:        30 * time.Second,
		maxRetries:     3,
		maxConcurrency: 10,
		enableMetrics:  true,
	}
}

// Config holds configuration for the tools package.
type Config struct {
	// Global tool settings
	Global *GlobalConfig `mapstructure:"global" yaml:"global"`

	// API tool settings
	API *APIConfig `mapstructure:"api" yaml:"api"`

	// Shell tool settings
	Shell *ShellConfig `mapstructure:"shell" yaml:"shell"`

	// MCP tool settings
	MCP *MCPConfig `mapstructure:"mcp" yaml:"mcp"`
}

// GlobalConfig holds global configuration for all tools.
type GlobalConfig struct {
	Enabled        bool          `mapstructure:"enabled" yaml:"enabled" env:"TOOLS_ENABLED" default:"true"`
	Timeout        time.Duration `mapstructure:"timeout" yaml:"timeout" env:"TOOLS_TIMEOUT" default:"30s"`
	MaxRetries     int           `mapstructure:"max_retries" yaml:"max_retries" env:"TOOLS_MAX_RETRIES" default:"3" validate:"min=0,max=10"`
	MaxConcurrency int           `mapstructure:"max_concurrency" yaml:"max_concurrency" env:"TOOLS_MAX_CONCURRENCY" default:"10" validate:"min=1,max=100"`
	EnableMetrics  bool          `mapstructure:"enable_metrics" yaml:"enable_metrics" env:"TOOLS_ENABLE_METRICS" default:"true"`
}

// APIConfig holds configuration for API tools.
type APIConfig struct {
	Enabled       bool          `mapstructure:"enabled" yaml:"enabled" env:"TOOLS_API_ENABLED" default:"true"`
	Timeout       time.Duration `mapstructure:"timeout" yaml:"timeout" env:"TOOLS_API_TIMEOUT" default:"30s"`
	MaxRetries    int           `mapstructure:"max_retries" yaml:"max_retries" env:"TOOLS_API_MAX_RETRIES" default:"3"`
	AllowInsecure bool          `mapstructure:"allow_insecure" yaml:"allow_insecure" env:"TOOLS_API_ALLOW_INSECURE" default:"false"`
}

// ShellConfig holds configuration for shell tools.
type ShellConfig struct {
	Enabled           bool          `mapstructure:"enabled" yaml:"enabled" env:"TOOLS_SHELL_ENABLED" default:"true"`
	Timeout           time.Duration `mapstructure:"timeout" yaml:"timeout" env:"TOOLS_SHELL_TIMEOUT" default:"30s"`
	AllowedCommands   []string      `mapstructure:"allowed_commands" yaml:"allowed_commands"`
	BlockedCommands   []string      `mapstructure:"blocked_commands" yaml:"blocked_commands"`
	WorkingDirectory  string        `mapstructure:"working_directory" yaml:"working_directory" env:"TOOLS_SHELL_WORKDIR"`
	UseRestrictedMode bool          `mapstructure:"use_restricted_mode" yaml:"use_restricted_mode" env:"TOOLS_SHELL_RESTRICTED" default:"false"`
}

// MCPConfig holds configuration for MCP (Minecraft Protocol) tools.
type MCPConfig struct {
	Enabled          bool          `mapstructure:"enabled" yaml:"enabled" env:"TOOLS_MCP_ENABLED" default:"true"`
	PingTimeout      time.Duration `mapstructure:"ping_timeout" yaml:"ping_timeout" env:"TOOLS_MCP_PING_TIMEOUT" default:"10s"`
	RconTimeout      time.Duration `mapstructure:"rcon_timeout" yaml:"rcon_timeout" env:"TOOLS_MCP_RCON_TIMEOUT" default:"15s"`
	DefaultRconPort  int           `mapstructure:"default_rcon_port" yaml:"default_rcon_port" env:"TOOLS_MCP_RCON_PORT" default:"25575"`
	DefaultQueryPort int           `mapstructure:"default_query_port" yaml:"default_query_port" env:"TOOLS_MCP_QUERY_PORT" default:"25565"`
}

// ToolConfig holds configuration for individual tool providers.
// This is used when registering tools with the registry.
type ToolConfig struct {
	Name        string         `mapstructure:"name" yaml:"name" validate:"required"`
	Description string         `mapstructure:"description" yaml:"description" validate:"required"`
	Type        string         `mapstructure:"type" yaml:"type" validate:"required,oneof=api shell gofunc mcp calculator echo custom"`
	Enabled     bool           `mapstructure:"enabled" yaml:"enabled" default:"true"`
	Timeout     time.Duration  `mapstructure:"timeout" yaml:"timeout" default:"30s"`
	MaxRetries  int            `mapstructure:"max_retries" yaml:"max_retries" default:"3"`
	Options     map[string]any `mapstructure:"options" yaml:"options"`
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	validate := validator.New()
	if err := validate.Struct(c); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	if c.Global != nil {
		if err := c.Global.Validate(); err != nil {
			return err
		}
	}
	if c.API != nil {
		if err := c.API.Validate(); err != nil {
			return err
		}
	}
	if c.Shell != nil {
		if err := c.Shell.Validate(); err != nil {
			return err
		}
	}
	if c.MCP != nil {
		if err := c.MCP.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// SetDefaults sets default values for the configuration.
func (c *Config) SetDefaults() {
	if c.Global == nil {
		c.Global = &GlobalConfig{}
	}
	c.Global.SetDefaults()

	if c.API == nil {
		c.API = &APIConfig{}
	}
	c.API.SetDefaults()

	if c.Shell == nil {
		c.Shell = &ShellConfig{}
	}
	c.Shell.SetDefaults()

	if c.MCP == nil {
		c.MCP = &MCPConfig{}
	}
	c.MCP.SetDefaults()
}

// Validate validates GlobalConfig.
func (c *GlobalConfig) Validate() error {
	if c.MaxRetries < 0 || c.MaxRetries > 10 {
		return errors.New("max_retries must be between 0 and 10")
	}
	if c.MaxConcurrency < 1 || c.MaxConcurrency > 100 {
		return errors.New("max_concurrency must be between 1 and 100")
	}
	return nil
}

// SetDefaults sets default values for GlobalConfig.
func (c *GlobalConfig) SetDefaults() {
	c.Enabled = true
	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}
	if c.MaxRetries == 0 {
		c.MaxRetries = 3
	}
	if c.MaxConcurrency == 0 {
		c.MaxConcurrency = 10
	}
	c.EnableMetrics = true
}

// Validate validates APIConfig.
func (c *APIConfig) Validate() error {
	return nil
}

// SetDefaults sets default values for APIConfig.
func (c *APIConfig) SetDefaults() {
	c.Enabled = true
	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}
	if c.MaxRetries == 0 {
		c.MaxRetries = 3
	}
}

// Validate validates ShellConfig.
func (c *ShellConfig) Validate() error {
	return nil
}

// SetDefaults sets default values for ShellConfig.
func (c *ShellConfig) SetDefaults() {
	c.Enabled = true
	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}
	if c.BlockedCommands == nil {
		c.BlockedCommands = []string{"rm -rf", "dd", "mkfs", ":(){:|:&};:"}
	}
}

// Validate validates MCPConfig.
func (c *MCPConfig) Validate() error {
	return nil
}

// SetDefaults sets default values for MCPConfig.
func (c *MCPConfig) SetDefaults() {
	c.Enabled = true
	if c.PingTimeout == 0 {
		c.PingTimeout = 10 * time.Second
	}
	if c.RconTimeout == 0 {
		c.RconTimeout = 15 * time.Second
	}
	if c.DefaultRconPort == 0 {
		c.DefaultRconPort = 25575
	}
	if c.DefaultQueryPort == 0 {
		c.DefaultQueryPort = 25565
	}
}

// DefaultConfig returns a new Config with default values.
func DefaultConfig() *Config {
	c := &Config{}
	c.SetDefaults()
	return c
}

// Validate validates ToolConfig.
func (c *ToolConfig) Validate() error {
	if c.Name == "" {
		return errors.New("tool name is required")
	}
	if c.Description == "" {
		return errors.New("tool description is required")
	}
	if c.Type == "" {
		return errors.New("tool type is required")
	}
	return nil
}

// SetDefaults sets default values for ToolConfig.
func (c *ToolConfig) SetDefaults() {
	c.Enabled = true
	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}
	if c.MaxRetries == 0 {
		c.MaxRetries = 3
	}
	if c.Options == nil {
		c.Options = make(map[string]any)
	}
}
