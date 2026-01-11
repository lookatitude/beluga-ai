// Package multimodal provides configuration structures for multimodal operations.
package multimodal

import (
	"context"
	"fmt"
	"time"

	validator "github.com/go-playground/validator/v10"
)

// Config holds configuration for multimodal operations.
type Config struct {
	// Provider name (e.g., "openai", "google", "anthropic")
	Provider string `mapstructure:"provider" yaml:"provider" env:"MULTIMODAL_PROVIDER" validate:"required"`

	// Model name (e.g., "gpt-4o", "gemini-pro")
	Model string `mapstructure:"model" yaml:"model" env:"MULTIMODAL_MODEL" validate:"required"`

	// API key for the provider
	APIKey string `mapstructure:"api_key" yaml:"api_key" env:"MULTIMODAL_API_KEY"`

	// Base URL for the API (optional, uses provider default if not set)
	BaseURL string `mapstructure:"base_url" yaml:"base_url" env:"MULTIMODAL_BASE_URL"`

	// Request timeout
	Timeout time.Duration `mapstructure:"timeout" yaml:"timeout" env:"MULTIMODAL_TIMEOUT" default:"30s"`

	// Maximum retry attempts
	MaxRetries int `mapstructure:"max_retries" yaml:"max_retries" env:"MULTIMODAL_MAX_RETRIES" default:"3"`

	// Delay between retries
	RetryDelay time.Duration `mapstructure:"retry_delay" yaml:"retry_delay" env:"MULTIMODAL_RETRY_DELAY" default:"1s"`

	// Enable streaming support
	EnableStreaming bool `mapstructure:"enable_streaming" yaml:"enable_streaming" env:"MULTIMODAL_ENABLE_STREAMING" default:"false"`

	// Chunk size for streaming operations
	StreamChunkSize int64 `mapstructure:"stream_chunk_size" yaml:"stream_chunk_size" env:"MULTIMODAL_STREAM_CHUNK_SIZE" default:"1048576"` // 1MB

	// Provider-specific settings
	ProviderSpecific map[string]any `mapstructure:"provider_specific" yaml:"provider_specific"`
}

// ModalityCapabilities represents the capabilities of a provider or model for different modalities.
type ModalityCapabilities struct {
	// Text processing support
	Text bool `mapstructure:"text" yaml:"text" default:"true"`

	// Image processing support
	Image bool `mapstructure:"image" yaml:"image" default:"false"`

	// Audio processing support
	Audio bool `mapstructure:"audio" yaml:"audio" default:"false"`

	// Video processing support
	Video bool `mapstructure:"video" yaml:"video" default:"false"`

	// Maximum image size in bytes
	MaxImageSize int64 `mapstructure:"max_image_size" yaml:"max_image_size" default:"10485760"` // 10MB

	// Maximum audio size in bytes
	MaxAudioSize int64 `mapstructure:"max_audio_size" yaml:"max_audio_size" default:"10485760"` // 10MB

	// Maximum video size in bytes
	MaxVideoSize int64 `mapstructure:"max_video_size" yaml:"max_video_size" default:"104857600"` // 100MB

	// Supported image formats
	SupportedImageFormats []string `mapstructure:"supported_image_formats" yaml:"supported_image_formats"`

	// Supported audio formats
	SupportedAudioFormats []string `mapstructure:"supported_audio_formats" yaml:"supported_audio_formats"`

	// Supported video formats
	SupportedVideoFormats []string `mapstructure:"supported_video_formats" yaml:"supported_video_formats"`
}

// RoutingConfig represents routing instructions for content blocks.
type RoutingConfig struct {
	// Routing strategy: "auto", "manual", "fallback"
	Strategy string `mapstructure:"strategy" yaml:"strategy" default:"auto" validate:"oneof=auto manual fallback"`

	// Provider for text content (optional, uses auto-routing if not set)
	TextProvider string `mapstructure:"text_provider" yaml:"text_provider"`

	// Provider for image content (optional, uses auto-routing if not set)
	ImageProvider string `mapstructure:"image_provider" yaml:"image_provider"`

	// Provider for audio content (optional, uses auto-routing if not set)
	AudioProvider string `mapstructure:"audio_provider" yaml:"audio_provider"`

	// Provider for video content (optional, uses auto-routing if not set)
	VideoProvider string `mapstructure:"video_provider" yaml:"video_provider"`

	// Fallback to text-only processing if modality not supported
	FallbackToText bool `mapstructure:"fallback_to_text" yaml:"fallback_to_text" default:"true"`
}

// Option is a functional option for configuring multimodal operations.
// Options can be chained together to build a complete configuration.
type Option func(*Config)

// WithProvider sets the provider name (e.g., "openai", "gemini", "anthropic").
func WithProvider(provider string) Option {
	return func(c *Config) {
		c.Provider = provider
	}
}

// WithModel sets the model name.
func WithModel(model string) Option {
	return func(c *Config) {
		c.Model = model
	}
}

// WithAPIKey sets the API key.
func WithAPIKey(apiKey string) Option {
	return func(c *Config) {
		c.APIKey = apiKey
	}
}

// WithTimeout sets the request timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.Timeout = timeout
	}
}

// WithStreaming enables streaming support.
func WithStreaming(enabled bool) Option {
	return func(c *Config) {
		c.EnableStreaming = enabled
	}
}

// WithMaxRetries sets the maximum number of retries.
func WithMaxRetries(maxRetries int) Option {
	return func(c *Config) {
		c.MaxRetries = maxRetries
	}
}

// WithRetryDelay sets the delay between retries.
func WithRetryDelay(delay time.Duration) Option {
	return func(c *Config) {
		c.RetryDelay = delay
	}
}

// WithStreamChunkSize sets the chunk size for streaming operations.
func WithStreamChunkSize(size int64) Option {
	return func(c *Config) {
		c.StreamChunkSize = size
	}
}

// WithBaseURL sets the base URL for the API.
func WithBaseURL(baseURL string) Option {
	return func(c *Config) {
		c.BaseURL = baseURL
	}
}

// WithProviderSpecific sets provider-specific settings.
func WithProviderSpecific(settings map[string]any) Option {
	return func(c *Config) {
		c.ProviderSpecific = settings
	}
}

// Validate validates the configuration and returns an error if invalid.
// Checks that required fields are set and that numeric values are within valid ranges.
func (c *Config) Validate() error {
	validate := validator.New()
	if err := validate.Struct(c); err != nil {
		return NewMultimodalErrorWithMessage("Validate", ErrCodeInvalidConfig,
			fmt.Sprintf("configuration validation failed: %v", err), err)
	}

	// Additional validation
	if c.Timeout <= 0 {
		return NewMultimodalErrorWithMessage("Validate", ErrCodeInvalidConfig,
			"timeout must be greater than 0", nil)
	}

	if c.MaxRetries < 0 {
		return NewMultimodalErrorWithMessage("Validate", ErrCodeInvalidConfig,
			"max_retries must be >= 0", nil)
	}

	if c.RetryDelay < 0 {
		return NewMultimodalErrorWithMessage("Validate", ErrCodeInvalidConfig,
			"retry_delay must be >= 0", nil)
	}

	if c.EnableStreaming && c.StreamChunkSize <= 0 {
		return NewMultimodalErrorWithMessage("Validate", ErrCodeInvalidConfig,
			"stream_chunk_size must be > 0 when streaming is enabled", nil)
	}

	return nil
}

// Validate validates the routing configuration and returns an error if invalid.
// For manual strategy, at least one provider must be specified.
func (r *RoutingConfig) Validate(ctx context.Context) error {
	validate := validator.New()
	if err := validate.Struct(r); err != nil {
		return NewMultimodalErrorWithMessage("ValidateRouting", ErrCodeInvalidConfig,
			fmt.Sprintf("routing configuration validation failed: %v", err), err)
	}

	// Additional validation for manual strategy
	if r.Strategy == "manual" {
		hasProvider := r.TextProvider != "" || r.ImageProvider != "" ||
			r.AudioProvider != "" || r.VideoProvider != ""
		if !hasProvider {
			return NewMultimodalErrorWithMessage("ValidateRouting", ErrCodeInvalidConfig,
				"manual strategy requires at least one provider to be specified", nil)
		}
	}

	return nil
}
