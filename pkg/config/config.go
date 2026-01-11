// Package config provides a flexible and extensible configuration management system
// for the Beluga AI Framework. It supports multiple configuration sources including
// files (YAML), environment variables, and programmatic configuration.
//
// The package follows the Beluga AI Framework design patterns with clear separation
// of interfaces, implementations, and providers.
package config

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/lookatitude/beluga-ai/pkg/config/iface"
	"github.com/lookatitude/beluga-ai/pkg/config/internal/loader"
	"github.com/lookatitude/beluga-ai/pkg/config/providers/composite"
	"github.com/lookatitude/beluga-ai/pkg/config/providers/viper"
)

// DefaultLoaderOptions returns default loader options for typical usage.
// These defaults provide sensible configuration for most applications:
// - Config name: "config"
// - Search paths: "./config" and "."
// - Environment prefix: "BELUGA"
// - Validation and defaults enabled
//
// Returns:
//   - iface.LoaderOptions: Default loader options
//
// Example:
//
//	options := config.DefaultLoaderOptions()
//	options.ConfigName = "myapp"
//	loader, err := config.NewLoader(options)
//
// Example usage can be found in examples/config/basic/main.go
func DefaultLoaderOptions() iface.LoaderOptions {
	return iface.LoaderOptions{
		ConfigName:  "config",
		ConfigPaths: []string{"./config", "."},
		EnvPrefix:   "BELUGA",
		Validate:    true,
		SetDefaults: true,
	}
}

// NewLoader creates a new configuration loader with the given options.
// The loader handles loading configuration from multiple sources and merging them.
//
// Parameters:
//   - options: Loader options specifying config name, paths, environment prefix, etc.
//
// Returns:
//   - iface.Loader: A new loader instance
//   - error: Configuration errors if options are invalid
//
// Example:
//
//	options := config.DefaultLoaderOptions()
//	loader, err := config.NewLoader(options)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Example usage can be found in examples/config/basic/main.go
func NewLoader(options iface.LoaderOptions) (iface.Loader, error) {
	return loader.NewLoader(options)
}

// NewProvider creates a new configuration provider.
// Currently supports Viper-based providers for YAML, JSON, TOML and environment variable configuration.
//
// Parameters:
//   - configName: Base name of the configuration file (without extension)
//   - configPaths: Directories to search for configuration files
//   - envPrefix: Prefix for environment variables (e.g., "BELUGA")
//   - format: Configuration format ("yaml", "json", "toml", or "" for auto-detect)
//
// Returns:
//   - iface.Provider: A new provider instance
//   - error: Provider creation errors
//
// Example:
//
//	provider, err := config.NewProvider("config", []string{"./config"}, "BELUGA", "yaml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Example usage can be found in examples/config/basic/main.go
func NewProvider(configName string, configPaths []string, envPrefix, format string) (iface.Provider, error) {
	return viper.NewViperProvider(configName, configPaths, envPrefix, format)
}

// NewYAMLProvider creates a new YAML configuration provider.
// This is a convenience function that calls NewProvider with format "yaml".
//
// Parameters:
//   - configName: Base name of the YAML configuration file
//   - configPaths: Directories to search for the configuration file
//   - envPrefix: Prefix for environment variables
//
// Returns:
//   - iface.Provider: A new YAML provider instance
//   - error: Provider creation errors
//
// Example:
//
//	provider, err := config.NewYAMLProvider("config", []string{"./config"}, "BELUGA")
//
// Example usage can be found in examples/config/basic/main.go
func NewYAMLProvider(configName string, configPaths []string, envPrefix string) (iface.Provider, error) {
	return viper.NewViperProvider(configName, configPaths, envPrefix, "yaml")
}

// NewJSONProvider creates a new JSON configuration provider.
// This is a convenience function that calls NewProvider with format "json".
//
// Parameters:
//   - configName: Base name of the JSON configuration file
//   - configPaths: Directories to search for the configuration file
//   - envPrefix: Prefix for environment variables
//
// Returns:
//   - iface.Provider: A new JSON provider instance
//   - error: Provider creation errors
//
// Example:
//
//	provider, err := config.NewJSONProvider("config", []string{"./config"}, "BELUGA")
//
// Example usage can be found in examples/config/basic/main.go
func NewJSONProvider(configName string, configPaths []string, envPrefix string) (iface.Provider, error) {
	return viper.NewViperProvider(configName, configPaths, envPrefix, "json")
}

// NewTOMLProvider creates a new TOML configuration provider.
// This is a convenience function that calls NewProvider with format "toml".
//
// Parameters:
//   - configName: Base name of the TOML configuration file
//   - configPaths: Directories to search for the configuration file
//   - envPrefix: Prefix for environment variables
//
// Returns:
//   - iface.Provider: A new TOML provider instance
//   - error: Provider creation errors
//
// Example:
//
//	provider, err := config.NewTOMLProvider("config", []string{"./config"}, "BELUGA")
//
// Example usage can be found in examples/config/basic/main.go
func NewTOMLProvider(configName string, configPaths []string, envPrefix string) (iface.Provider, error) {
	return viper.NewViperProvider(configName, configPaths, envPrefix, "toml")
}

// NewAutoDetectProvider creates a provider that auto-detects format from file extension.
// Supported formats: .yaml, .yml, .json, .toml
//
// Parameters:
//   - configName: Base name of the configuration file
//   - configPaths: Directories to search for the configuration file
//   - envPrefix: Prefix for environment variables
//
// Returns:
//   - iface.Provider: A new auto-detect provider instance
//   - error: Provider creation errors
//
// Example:
//
//	provider, err := config.NewAutoDetectProvider("config", []string{"./config"}, "BELUGA")
//
// Example usage can be found in examples/config/basic/main.go
func NewAutoDetectProvider(configName string, configPaths []string, envPrefix string) (iface.Provider, error) {
	return viper.NewViperProvider(configName, configPaths, envPrefix, "")
}

// NewCompositeProvider creates a composite provider that tries multiple providers in order.
// This allows for fallback configurations (e.g., file -> environment -> defaults).
// The first provider that successfully loads configuration is used.
//
// Parameters:
//   - providers: One or more providers to try in order
//
// Returns:
//   - iface.Provider: A composite provider that tries each provider in sequence
//
// Example:
//
//	fileProvider, _ := config.NewYAMLProvider("config", []string{"./config"}, "")
//	envProvider, _ := config.NewProvider("", nil, "BELUGA", "")
//	composite := config.NewCompositeProvider(fileProvider, envProvider)
//
// Example usage can be found in examples/config/basic/main.go
func NewCompositeProvider(providers ...iface.Provider) iface.Provider {
	return composite.NewCompositeProvider(providers...)
}

// LoadConfig loads configuration using default settings.
// This is a convenience function for simple use cases. It uses DefaultLoaderOptions()
// and searches for configuration files in standard locations.
//
// Returns:
//   - *iface.Config: Loaded configuration
//   - error: Loading errors (file not found, invalid format, etc.)
//
// Example:
//
//	cfg, err := config.LoadConfig()
//	if err != nil {
//	    log.Fatal("Failed to load config:", err)
//	}
//
// Example usage can be found in examples/config/basic/main.go
func LoadConfig() (*iface.Config, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/config")
	ctx, span := tracer.Start(context.Background(), "LoadConfig")
	defer span.End()

	start := time.Now()
	success := false
	defer func() {
		GetGlobalMetrics().RecordConfigLoad(ctx, time.Since(start), success, "loader")
	}()

	loader, err := NewLoader(DefaultLoaderOptions())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create loader")
		span.SetAttributes(attribute.String("error", err.Error()))
		logWithOTELContext(ctx, slog.LevelError, "Failed to create config loader", "error", err)
		return nil, fmt.Errorf("failed to create loader: %w", err)
	}

	cfg, err := loader.LoadConfig()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to load config")
		span.SetAttributes(attribute.String("error", err.Error()))
		logWithOTELContext(ctx, slog.LevelError, "Failed to load config", "error", err)
		return nil, err
	}

	span.SetStatus(codes.Ok, "config loaded successfully")
	logWithOTELContext(ctx, slog.LevelInfo, "Config loaded successfully", "source", "loader")
	success = true
	return cfg, nil
}

// LoadFromEnv loads configuration from environment variables only.
// All environment variables with the given prefix are loaded and converted
// to configuration keys by removing the prefix and converting to lowercase.
//
// Parameters:
//   - prefix: Environment variable prefix (e.g., "BELUGA" for BELUGA_API_KEY)
//
// Returns:
//   - *iface.Config: Configuration loaded from environment variables
//   - error: Loading or parsing errors
//
// Example:
//
//	// Load from BELUGA_* environment variables
//	cfg, err := config.LoadFromEnv("BELUGA")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Example usage can be found in examples/config/basic/main.go
func LoadFromEnv(prefix string) (*iface.Config, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/config")
	ctx, span := tracer.Start(context.Background(), "LoadFromEnv")
	defer span.End()

	span.SetAttributes(attribute.String("prefix", prefix))

	start := time.Now()
	success := false
	defer func() {
		GetGlobalMetrics().RecordConfigLoad(ctx, time.Since(start), success, "env")
	}()

	cfg, err := loader.LoadFromEnv(prefix)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to load config from env")
		span.SetAttributes(attribute.String("error", err.Error()))
		logWithOTELContext(ctx, slog.LevelError, "Failed to load config from env", "error", err, "prefix", prefix)
		return nil, err
	}

	span.SetStatus(codes.Ok, "config loaded from env successfully")
	logWithOTELContext(ctx, slog.LevelInfo, "Config loaded from env successfully", "prefix", prefix)
	success = true
	return cfg, nil
}

// LoadFromFile loads configuration from a specific file.
// The file format is auto-detected from the extension (.yaml, .json, .toml).
//
// Parameters:
//   - filePath: Path to the configuration file
//
// Returns:
//   - *iface.Config: Configuration loaded from the file
//   - error: File not found, parsing errors, or format errors
//
// Example:
//
//	cfg, err := config.LoadFromFile("./config/app.yaml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Example usage can be found in examples/config/basic/main.go
func LoadFromFile(filePath string) (*iface.Config, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/config")
	ctx, span := tracer.Start(context.Background(), "LoadFromFile")
	defer span.End()

	span.SetAttributes(attribute.String("file_path", filePath))

	start := time.Now()
	success := false
	defer func() {
		GetGlobalMetrics().RecordConfigLoad(ctx, time.Since(start), success, "file")
	}()

	cfg, err := loader.LoadFromFile(filePath)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to load config from file")
		span.SetAttributes(attribute.String("error", err.Error()))
		logWithOTELContext(ctx, slog.LevelError, "Failed to load config from file", "error", err, "file_path", filePath)
		return nil, err
	}

	span.SetStatus(codes.Ok, "config loaded from file successfully")
	logWithOTELContext(ctx, slog.LevelInfo, "Config loaded from file successfully", "file_path", filePath)
	success = true
	return cfg, nil
}

// MustLoadConfig loads configuration and panics on error.
// Use this only in main() or initialization code where failure should stop the program.
// For production code, prefer LoadConfig() with proper error handling.
//
// Returns:
//   - *iface.Config: Loaded configuration (panics on error)
//
// Example:
//
//	cfg := config.MustLoadConfig()
//	// Use cfg...
//
// Example usage can be found in examples/config/basic/main.go
func MustLoadConfig() *iface.Config {
	cfg, err := LoadConfig()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}

// ValidateConfig validates the entire configuration structure.
// It checks that all required fields are present and that values are within
// acceptable ranges. Validation rules are defined in the Config struct tags.
//
// Parameters:
//   - cfg: Configuration to validate
//
// Returns:
//   - error: Validation errors if configuration is invalid, nil otherwise
//
// Example:
//
//	cfg, _ := config.LoadConfig()
//	if err := config.ValidateConfig(cfg); err != nil {
//	    log.Fatal("Invalid configuration:", err)
//	}
//
// Example usage can be found in examples/config/basic/main.go
func ValidateConfig(cfg *iface.Config) error {
	ctx := context.Background()
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/config")
	ctx, span := tracer.Start(ctx, "config.ValidateConfig")
	defer span.End()

	start := time.Now()
	success := false
	defer func() {
		GetGlobalMetrics().RecordValidation(ctx, time.Since(start), success)
	}()

	err := iface.ValidateConfig(cfg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "config validation failed")
		span.SetAttributes(attribute.String("error", err.Error()))
		logWithOTELContext(ctx, slog.LevelError, "Config validation failed", "error", err)
		return err
	}

	span.SetStatus(codes.Ok, "config validation succeeded")
	logWithOTELContext(ctx, slog.LevelInfo, "Config validation succeeded")
	success = true
	return nil
}

// logWithOTELContext extracts OTEL trace/span IDs from context and logs with structured logging.
func logWithOTELContext(ctx context.Context, level slog.Level, msg string, attrs ...any) {
	// Extract OTEL context
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		otelAttrs := []any{
			"trace_id", spanCtx.TraceID().String(),
			"span_id", spanCtx.SpanID().String(),
		}
		attrs = append(otelAttrs, attrs...)
	}

	// Use slog for structured logging
	logger := slog.Default()
	logger.Log(ctx, level, msg, attrs...)
}

// SetDefaults sets default values for configuration fields.
func SetDefaults(cfg *iface.Config) {
	iface.SetDefaults(cfg)
}

// GetEnvConfigMap returns a map of all environment variables with the given prefix.
func GetEnvConfigMap(prefix string) map[string]string {
	return loader.GetEnvConfigMap(prefix)
}

// EnvVarName converts a config key to environment variable name.
func EnvVarName(prefix, key string) string {
	return loader.EnvVarName(prefix, key)
}

// ConfigKey converts an environment variable name to config key.
func ConfigKey(prefix, envVar string) string {
	return loader.ConfigKey(prefix, envVar)
}

// Option is a functional option for configuring components.
type Option func(*iface.Config)

// WithConfigName sets the configuration file name (without extension).
func WithConfigName(name string) Option {
	return func(c *iface.Config) {
		// This is a no-op for now as it's handled by the loader
		// Future enhancement could store this in config for reference
		_ = name
	}
}

// WithConfigPaths sets the paths to search for configuration files.
func WithConfigPaths(paths ...string) Option {
	return func(c *iface.Config) {
		// This is a no-op for now as it's handled by the loader
		// Future enhancement could store this in config for reference
		_ = paths
	}
}

// WithEnvPrefix sets the environment variable prefix.
func WithEnvPrefix(prefix string) Option {
	return func(c *iface.Config) {
		// This is a no-op for now as it's handled by the loader
		// Future enhancement could store this in config for reference
		_ = prefix
	}
}
