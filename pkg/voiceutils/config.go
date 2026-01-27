// Package voiceutils provides shared utilities for voice processing packages.
package voiceutils

import (
	"errors"
	"fmt"
	"time"

	validator "github.com/go-playground/validator/v10"
)

// Option is a functional option for configuring voice utils components.
type Option func(*optionConfig)

// optionConfig holds the internal configuration options.
type optionConfig struct {
	timeout       time.Duration
	maxRetries    int
	bufferSize    int
	sampleRate    int
	channels      int
	bitDepth      int
	encoding      string
	enableMetrics bool
}

// WithTimeout sets the timeout for voice operations.
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

// WithBufferSize sets the buffer size for audio processing.
func WithBufferSize(size int) Option {
	return func(c *optionConfig) {
		c.bufferSize = size
	}
}

// WithSampleRate sets the audio sample rate.
func WithSampleRate(rate int) Option {
	return func(c *optionConfig) {
		c.sampleRate = rate
	}
}

// WithChannels sets the number of audio channels.
func WithChannels(channels int) Option {
	return func(c *optionConfig) {
		c.channels = channels
	}
}

// WithBitDepth sets the audio bit depth.
func WithBitDepth(depth int) Option {
	return func(c *optionConfig) {
		c.bitDepth = depth
	}
}

// WithEncoding sets the audio encoding format.
func WithEncoding(encoding string) Option {
	return func(c *optionConfig) {
		c.encoding = encoding
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
		timeout:       30 * time.Second,
		maxRetries:    3,
		bufferSize:    4096,
		sampleRate:    16000,
		channels:      1,
		bitDepth:      16,
		encoding:      "pcm",
		enableMetrics: true,
	}
}

// Config holds configuration for the voiceutils package.
type Config struct {
	// Audio format settings
	Audio *AudioConfig `mapstructure:"audio" yaml:"audio"`

	// Buffer pool settings
	BufferPool *BufferPoolConfig `mapstructure:"buffer_pool" yaml:"buffer_pool"`

	// Retry settings
	Retry *RetryConfig `mapstructure:"retry" yaml:"retry"`

	// Rate limit settings
	RateLimit *RateLimitConfig `mapstructure:"rate_limit" yaml:"rate_limit"`

	// Circuit breaker settings
	CircuitBreaker *CircuitBreakerConfig `mapstructure:"circuit_breaker" yaml:"circuit_breaker"`
}

// AudioConfig holds audio format configuration.
type AudioConfig struct {
	SampleRate int    `mapstructure:"sample_rate" yaml:"sample_rate" env:"VOICE_SAMPLE_RATE" default:"16000" validate:"min=8000,max=48000"`
	Channels   int    `mapstructure:"channels" yaml:"channels" env:"VOICE_CHANNELS" default:"1" validate:"min=1,max=2"`
	BitDepth   int    `mapstructure:"bit_depth" yaml:"bit_depth" env:"VOICE_BIT_DEPTH" default:"16" validate:"oneof=16 24 32"`
	Encoding   string `mapstructure:"encoding" yaml:"encoding" env:"VOICE_ENCODING" default:"pcm"`
}

// BufferPoolConfig holds buffer pool configuration.
type BufferPoolConfig struct {
	Enabled  bool  `mapstructure:"enabled" yaml:"enabled" env:"VOICE_BUFFER_POOL_ENABLED" default:"true"`
	Sizes    []int `mapstructure:"sizes" yaml:"sizes"`
	MaxPools int   `mapstructure:"max_pools" yaml:"max_pools" env:"VOICE_MAX_POOLS" default:"10"`
}

// RetryConfig holds retry configuration.
type RetryConfig struct {
	MaxRetries int           `mapstructure:"max_retries" yaml:"max_retries" env:"VOICE_MAX_RETRIES" default:"3" validate:"min=0,max=10"`
	BaseDelay  time.Duration `mapstructure:"base_delay" yaml:"base_delay" env:"VOICE_RETRY_BASE_DELAY" default:"100ms"`
	MaxDelay   time.Duration `mapstructure:"max_delay" yaml:"max_delay" env:"VOICE_RETRY_MAX_DELAY" default:"5s"`
	Multiplier float64       `mapstructure:"multiplier" yaml:"multiplier" env:"VOICE_RETRY_MULTIPLIER" default:"2.0" validate:"min=1.0,max=5.0"`
}

// RateLimitConfig holds rate limit configuration.
type RateLimitConfig struct {
	Enabled     bool          `mapstructure:"enabled" yaml:"enabled" env:"VOICE_RATE_LIMIT_ENABLED" default:"true"`
	MaxRequests int           `mapstructure:"max_requests" yaml:"max_requests" env:"VOICE_RATE_LIMIT_MAX_REQUESTS" default:"100"`
	Window      time.Duration `mapstructure:"window" yaml:"window" env:"VOICE_RATE_LIMIT_WINDOW" default:"1m"`
}

// CircuitBreakerConfig holds circuit breaker configuration.
type CircuitBreakerConfig struct {
	Enabled          bool          `mapstructure:"enabled" yaml:"enabled" env:"VOICE_CB_ENABLED" default:"true"`
	FailureThreshold int           `mapstructure:"failure_threshold" yaml:"failure_threshold" env:"VOICE_CB_FAILURE_THRESHOLD" default:"5" validate:"min=1"`
	SuccessThreshold int           `mapstructure:"success_threshold" yaml:"success_threshold" env:"VOICE_CB_SUCCESS_THRESHOLD" default:"2" validate:"min=1"`
	Timeout          time.Duration `mapstructure:"timeout" yaml:"timeout" env:"VOICE_CB_TIMEOUT" default:"30s"`
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	validate := validator.New()
	if err := validate.Struct(c); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Validate individual configs
	if c.Audio != nil {
		if err := c.Audio.Validate(); err != nil {
			return err
		}
	}
	if c.BufferPool != nil {
		if err := c.BufferPool.Validate(); err != nil {
			return err
		}
	}
	if c.Retry != nil {
		if err := c.Retry.Validate(); err != nil {
			return err
		}
	}
	if c.RateLimit != nil {
		if err := c.RateLimit.Validate(); err != nil {
			return err
		}
	}
	if c.CircuitBreaker != nil {
		if err := c.CircuitBreaker.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// SetDefaults sets default values for the configuration.
func (c *Config) SetDefaults() {
	if c.Audio == nil {
		c.Audio = &AudioConfig{}
	}
	c.Audio.SetDefaults()

	if c.BufferPool == nil {
		c.BufferPool = &BufferPoolConfig{}
	}
	c.BufferPool.SetDefaults()

	if c.Retry == nil {
		c.Retry = &RetryConfig{}
	}
	c.Retry.SetDefaults()

	if c.RateLimit == nil {
		c.RateLimit = &RateLimitConfig{}
	}
	c.RateLimit.SetDefaults()

	if c.CircuitBreaker == nil {
		c.CircuitBreaker = &CircuitBreakerConfig{}
	}
	c.CircuitBreaker.SetDefaults()
}

// Validate validates AudioConfig.
func (c *AudioConfig) Validate() error {
	if c.SampleRate < 8000 || c.SampleRate > 48000 {
		return errors.New("sample rate must be between 8000 and 48000")
	}
	if c.Channels < 1 || c.Channels > 2 {
		return errors.New("channels must be 1 (mono) or 2 (stereo)")
	}
	if c.BitDepth != 16 && c.BitDepth != 24 && c.BitDepth != 32 {
		return errors.New("bit depth must be 16, 24, or 32")
	}
	if c.Encoding == "" {
		return errors.New("encoding is required")
	}
	return nil
}

// SetDefaults sets default values for AudioConfig.
func (c *AudioConfig) SetDefaults() {
	if c.SampleRate == 0 {
		c.SampleRate = 16000
	}
	if c.Channels == 0 {
		c.Channels = 1
	}
	if c.BitDepth == 0 {
		c.BitDepth = 16
	}
	if c.Encoding == "" {
		c.Encoding = "pcm"
	}
}

// Validate validates BufferPoolConfig.
func (c *BufferPoolConfig) Validate() error {
	if c.MaxPools < 1 {
		return errors.New("max_pools must be at least 1")
	}
	return nil
}

// SetDefaults sets default values for BufferPoolConfig.
func (c *BufferPoolConfig) SetDefaults() {
	c.Enabled = true
	if len(c.Sizes) == 0 {
		c.Sizes = []int{512, 1024, 2048, 4096, 8192, 16384, 32768}
	}
	if c.MaxPools == 0 {
		c.MaxPools = 10
	}
}

// Validate validates RetryConfig.
func (c *RetryConfig) Validate() error {
	if c.MaxRetries < 0 || c.MaxRetries > 10 {
		return errors.New("max_retries must be between 0 and 10")
	}
	if c.Multiplier < 1.0 || c.Multiplier > 5.0 {
		return errors.New("multiplier must be between 1.0 and 5.0")
	}
	return nil
}

// SetDefaults sets default values for RetryConfig.
func (c *RetryConfig) SetDefaults() {
	if c.MaxRetries == 0 {
		c.MaxRetries = 3
	}
	if c.BaseDelay == 0 {
		c.BaseDelay = 100 * time.Millisecond
	}
	if c.MaxDelay == 0 {
		c.MaxDelay = 5 * time.Second
	}
	if c.Multiplier == 0 {
		c.Multiplier = 2.0
	}
}

// Validate validates RateLimitConfig.
func (c *RateLimitConfig) Validate() error {
	if c.Enabled && c.MaxRequests < 1 {
		return errors.New("max_requests must be at least 1 when rate limiting is enabled")
	}
	return nil
}

// SetDefaults sets default values for RateLimitConfig.
func (c *RateLimitConfig) SetDefaults() {
	c.Enabled = true
	if c.MaxRequests == 0 {
		c.MaxRequests = 100
	}
	if c.Window == 0 {
		c.Window = time.Minute
	}
}

// Validate validates CircuitBreakerConfig.
func (c *CircuitBreakerConfig) Validate() error {
	if c.Enabled {
		if c.FailureThreshold < 1 {
			return errors.New("failure_threshold must be at least 1 when circuit breaker is enabled")
		}
		if c.SuccessThreshold < 1 {
			return errors.New("success_threshold must be at least 1 when circuit breaker is enabled")
		}
	}
	return nil
}

// SetDefaults sets default values for CircuitBreakerConfig.
func (c *CircuitBreakerConfig) SetDefaults() {
	c.Enabled = true
	if c.FailureThreshold == 0 {
		c.FailureThreshold = 5
	}
	if c.SuccessThreshold == 0 {
		c.SuccessThreshold = 2
	}
	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}
}

// DefaultConfig returns a new Config with default values.
func DefaultConfig() *Config {
	c := &Config{}
	c.SetDefaults()
	return c
}
