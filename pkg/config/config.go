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
func NewLoader(options iface.LoaderOptions) (iface.Loader, error) {
	return loader.NewLoader(options)
}

// NewProvider creates a new configuration provider.
// Currently supports Viper-based providers for YAML, JSON, TOML and environment variable configuration.
func NewProvider(configName string, configPaths []string, envPrefix, format string) (iface.Provider, error) {
	return viper.NewViperProvider(configName, configPaths, envPrefix, format)
}

// NewYAMLProvider creates a new YAML configuration provider.
func NewYAMLProvider(configName string, configPaths []string, envPrefix string) (iface.Provider, error) {
	return viper.NewViperProvider(configName, configPaths, envPrefix, "yaml")
}

// NewJSONProvider creates a new JSON configuration provider.
func NewJSONProvider(configName string, configPaths []string, envPrefix string) (iface.Provider, error) {
	return viper.NewViperProvider(configName, configPaths, envPrefix, "json")
}

// NewTOMLProvider creates a new TOML configuration provider.
func NewTOMLProvider(configName string, configPaths []string, envPrefix string) (iface.Provider, error) {
	return viper.NewViperProvider(configName, configPaths, envPrefix, "toml")
}

// NewAutoDetectProvider creates a provider that auto-detects format from file extension.
func NewAutoDetectProvider(configName string, configPaths []string, envPrefix string) (iface.Provider, error) {
	return viper.NewViperProvider(configName, configPaths, envPrefix, "")
}

// NewCompositeProvider creates a composite provider that tries multiple providers in order.
// This allows for fallback configurations (e.g., file -> environment -> defaults).
func NewCompositeProvider(providers ...iface.Provider) iface.Provider {
	return composite.NewCompositeProvider(providers...)
}

// LoadConfig loads configuration using default settings.
// This is a convenience function for simple use cases.
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
func MustLoadConfig() *iface.Config {
	cfg, err := LoadConfig()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}

// ValidateConfig validates the entire configuration structure.
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
