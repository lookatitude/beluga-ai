// Package config provides a flexible and extensible configuration management system
// for the Beluga AI Framework. It supports multiple configuration sources including
// files (YAML), environment variables, and programmatic configuration.
//
// The package follows the Beluga AI Framework design patterns with clear separation
// of interfaces, implementations, and providers.
package config

import (
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/config/iface"
	"github.com/lookatitude/beluga-ai/pkg/config/internal/loader"
	"github.com/lookatitude/beluga-ai/pkg/config/providers/viper"
)

// DefaultLoaderOptions returns default loader options for typical usage
func DefaultLoaderOptions() iface.LoaderOptions {
	return iface.LoaderOptions{
		ConfigName:  "config",
		ConfigPaths: []string{"./config", "."},
		EnvPrefix:   "BELUGA",
		Validate:    true,
		SetDefaults: true,
	}
}

// NewLoader creates a new configuration loader with the given options
func NewLoader(options iface.LoaderOptions) (iface.Loader, error) {
	return loader.NewLoader(options)
}

// NewProvider creates a new configuration provider.
// Currently supports Viper-based providers for YAML and environment variable configuration.
func NewProvider(configName string, configPaths []string, envPrefix string) (iface.Provider, error) {
	return viper.NewViperProvider(configName, configPaths, envPrefix)
}

// LoadConfig loads configuration using default settings.
// This is a convenience function for simple use cases.
func LoadConfig() (*iface.Config, error) {
	loader, err := NewLoader(DefaultLoaderOptions())
	if err != nil {
		return nil, fmt.Errorf("failed to create loader: %w", err)
	}
	return loader.LoadConfig()
}

// LoadFromEnv loads configuration from environment variables only
func LoadFromEnv(prefix string) (*iface.Config, error) {
	return loader.LoadFromEnv(prefix)
}

// LoadFromFile loads configuration from a specific file
func LoadFromFile(filePath string) (*iface.Config, error) {
	return loader.LoadFromFile(filePath)
}

// MustLoadConfig loads configuration and panics on error.
// Use this only in main() or initialization code where failure should stop the program.
func MustLoadConfig() *iface.Config {
	cfg, err := LoadConfig()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}

// ValidateConfig validates the entire configuration structure
func ValidateConfig(cfg *iface.Config) error {
	return iface.ValidateConfig(cfg)
}

// SetDefaults sets default values for configuration fields
func SetDefaults(cfg *iface.Config) {
	iface.SetDefaults(cfg)
}

// GetEnvConfigMap returns a map of all environment variables with the given prefix
func GetEnvConfigMap(prefix string) map[string]string {
	return loader.GetEnvConfigMap(prefix)
}

// EnvVarName converts a config key to environment variable name
func EnvVarName(prefix, key string) string {
	return loader.EnvVarName(prefix, key)
}

// ConfigKey converts an environment variable name to config key
func ConfigKey(prefix, envVar string) string {
	return loader.ConfigKey(prefix, envVar)
}

// Option is a functional option for configuring components
type Option func(*iface.Config)

// WithConfigName sets the configuration file name (without extension)
func WithConfigName(name string) Option {
	return func(c *iface.Config) {
		// This is a no-op for now as it's handled by the loader
		// Future enhancement could store this in config for reference
		_ = name
	}
}

// WithConfigPaths sets the paths to search for configuration files
func WithConfigPaths(paths ...string) Option {
	return func(c *iface.Config) {
		// This is a no-op for now as it's handled by the loader
		// Future enhancement could store this in config for reference
		_ = paths
	}
}

// WithEnvPrefix sets the environment variable prefix
func WithEnvPrefix(prefix string) Option {
	return func(c *iface.Config) {
		// This is a no-op for now as it's handled by the loader
		// Future enhancement could store this in config for reference
		_ = prefix
	}
}
