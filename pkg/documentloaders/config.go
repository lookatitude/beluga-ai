package documentloaders

import (
	"fmt"
	"runtime"

	"github.com/go-playground/validator/v10"
)

// LoaderConfig contains common configuration for all loaders.
type LoaderConfig struct {
	// MaxFileSize is the maximum file size in bytes (default: 100MB).
	MaxFileSize int64 `mapstructure:"max_file_size" yaml:"max_file_size" env:"LOADER_MAX_FILE_SIZE" validate:"min=1"`
}

// DefaultLoaderConfig returns a LoaderConfig with sensible defaults.
func DefaultLoaderConfig() *LoaderConfig {
	return &LoaderConfig{
		MaxFileSize: 100 * 1024 * 1024, // 100MB
	}
}

// DirectoryConfig contains configuration for RecursiveDirectoryLoader.
type DirectoryConfig struct {
	Path           string   `mapstructure:"path" yaml:"path" env:"LOADER_PATH" validate:"required"`
	Extensions     []string `mapstructure:"extensions" yaml:"extensions" env:"LOADER_EXTENSIONS"`
	LoaderConfig   `mapstructure:",squash"`
	MaxDepth       int  `mapstructure:"max_depth" yaml:"max_depth" env:"LOADER_MAX_DEPTH" validate:"min=0"`
	Concurrency    int  `mapstructure:"concurrency" yaml:"concurrency" env:"LOADER_CONCURRENCY" validate:"min=1"`
	FollowSymlinks bool `mapstructure:"follow_symlinks" yaml:"follow_symlinks" env:"LOADER_FOLLOW_SYMLINKS"`
}

// DefaultDirectoryConfig returns a DirectoryConfig with sensible defaults.
func DefaultDirectoryConfig() *DirectoryConfig {
	return &DirectoryConfig{
		LoaderConfig:   *DefaultLoaderConfig(),
		MaxDepth:       10,
		Concurrency:    runtime.GOMAXPROCS(0),
		FollowSymlinks: true,
	}
}

// Validate validates the DirectoryConfig.
func (c *DirectoryConfig) Validate() error {
	validate := validator.New()
	if err := validate.Struct(c); err != nil {
		return NewLoaderError("Validate", ErrCodeInvalidConfig, "", fmt.Sprintf("configuration validation failed: %v", err), err)
	}
	return nil
}

// Option configures a loader.
type Option func(*LoaderConfig)

// WithMaxFileSize sets the maximum file size.
func WithMaxFileSize(size int64) Option {
	return func(c *LoaderConfig) {
		c.MaxFileSize = size
	}
}

// DirectoryOption configures a directory loader.
type DirectoryOption func(*DirectoryConfig)

// WithMaxDepth sets the maximum recursion depth.
func WithMaxDepth(depth int) DirectoryOption {
	return func(c *DirectoryConfig) {
		c.MaxDepth = depth
	}
}

// WithExtensions sets the file extension filter.
func WithExtensions(exts ...string) DirectoryOption {
	return func(c *DirectoryConfig) {
		c.Extensions = exts
	}
}

// WithConcurrency sets the number of concurrent workers.
func WithConcurrency(n int) DirectoryOption {
	return func(c *DirectoryConfig) {
		c.Concurrency = n
	}
}

// WithFollowSymlinks enables or disables symlink following.
func WithFollowSymlinks(follow bool) DirectoryOption {
	return func(c *DirectoryConfig) {
		c.FollowSymlinks = follow
	}
}

// WithDirectoryMaxFileSize sets the maximum file size for directory loader.
func WithDirectoryMaxFileSize(size int64) DirectoryOption {
	return func(c *DirectoryConfig) {
		c.MaxFileSize = size
	}
}
