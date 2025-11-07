package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/config/iface"
	"github.com/lookatitude/beluga-ai/pkg/config/providers/viper"
)

// Loader provides a high-level interface for loading configuration
type Loader struct {
	options iface.LoaderOptions
}

// NewLoader creates a new configuration loader with the given options
func NewLoader(options iface.LoaderOptions) (*Loader, error) {
	return &Loader{options: options}, nil
}

// LoadConfig loads the main application configuration
func (l *Loader) LoadConfig() (*iface.Config, error) {
	provider, err := viper.NewViperProvider(
		l.options.ConfigName,
		l.options.ConfigPaths,
		l.options.EnvPrefix,
		"yaml", // default to yaml for backward compatibility
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create config provider: %w", err)
	}

	var cfg iface.Config
	if err := provider.Load(&cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if l.options.SetDefaults {
		// Set defaults directly on the loaded config struct
		iface.SetDefaults(&cfg)
	}

	if l.options.Validate {
		if err := provider.Validate(); err != nil {
			return nil, fmt.Errorf("config validation failed: %w", err)
		}
	}

	return &cfg, nil
}

// LoadFromEnv loads configuration from environment variables only
func LoadFromEnv(prefix string) (*iface.Config, error) {
	provider, err := viper.NewViperProvider("", nil, prefix, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create env config provider: %w", err)
	}

	var cfg iface.Config

	// Viper's Unmarshal doesn't always parse array indices from env vars correctly
	// Use UnmarshalKey for each section to ensure env vars are parsed
	// Always try to unmarshal (IsSet may not detect array indices from env vars)
	_ = provider.UnmarshalKey("llm_providers", &cfg.LLMProviders)
	_ = provider.UnmarshalKey("embedding_providers", &cfg.EmbeddingProviders)
	_ = provider.UnmarshalKey("vector_stores", &cfg.VectorStores)
	_ = provider.UnmarshalKey("agents", &cfg.Agents)
	_ = provider.UnmarshalKey("tools", &cfg.Tools)

	// Load other fields using standard Load (this won't overwrite arrays we just set)
	if err := provider.Load(&cfg); err != nil {
		return nil, fmt.Errorf("failed to load config from env: %w", err)
	}

	iface.SetDefaults(&cfg)

	// Only validate if config has content (env vars might not always be set)
	// Empty configs are valid (no providers configured)
	hasContent := len(cfg.LLMProviders) > 0 || len(cfg.EmbeddingProviders) > 0 ||
		len(cfg.VectorStores) > 0 || len(cfg.Agents) > 0 || len(cfg.Tools) > 0

	if hasContent {
		if err := iface.ValidateConfig(&cfg); err != nil {
			return nil, fmt.Errorf("env config validation failed: %w", err)
		}
	}

	return &cfg, nil
}

// LoadFromFile loads configuration from a specific file
func LoadFromFile(filePath string) (*iface.Config, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file does not exist: %s", filePath)
	}

	dir := filepath.Dir(filePath)
	name := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))

	provider, err := viper.NewViperProvider(name, []string{dir}, "", "")
	if err != nil {
		return nil, fmt.Errorf("failed to create file config provider: %w", err)
	}

	var cfg iface.Config
	if err := provider.Load(&cfg); err != nil {
		return nil, fmt.Errorf("failed to load config from file: %w", err)
	}

	// Set defaults on the loaded config (modifies struct in place)
	iface.SetDefaults(&cfg)

	if err := iface.ValidateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("file config validation failed: %w", err)
	}

	return &cfg, nil
}

// MustLoadConfig loads configuration and panics on error
func (l *Loader) MustLoadConfig() *iface.Config {
	cfg, err := l.LoadConfig()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}

// WithConfigName sets the configuration file name (without extension)
func (l *Loader) WithConfigName(name string) *Loader {
	l.options.ConfigName = name
	return l
}

// WithConfigPaths sets the paths to search for configuration files
func (l *Loader) WithConfigPaths(paths ...string) *Loader {
	l.options.ConfigPaths = paths
	return l
}

// WithEnvPrefix sets the environment variable prefix
func (l *Loader) WithEnvPrefix(prefix string) *Loader {
	l.options.EnvPrefix = prefix
	return l
}

// WithValidation enables or disables configuration validation
func (l *Loader) WithValidation(enabled bool) *Loader {
	l.options.Validate = enabled
	return l
}

// WithDefaults enables or disables setting default values
func (l *Loader) WithDefaults(enabled bool) *Loader {
	l.options.SetDefaults = enabled
	return l
}

// GetEnvConfigMap returns a map of all environment variables with the given prefix
func GetEnvConfigMap(prefix string) map[string]string {
	envMap := make(map[string]string)
	if prefix == "" {
		// Empty prefix means no matching vars
		return envMap
	}
	prefix = strings.ToUpper(prefix) + "_"

	for _, env := range os.Environ() {
		if strings.HasPrefix(env, prefix) {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				key := strings.ToLower(strings.TrimPrefix(parts[0], prefix))
				if key != "" { // Only add if key is not empty after trimming
					envMap[key] = parts[1]
				}
			}
		}
	}

	return envMap
}

// EnvVarName converts a config key to environment variable name
func EnvVarName(prefix, key string) string {
	return strings.ToUpper(prefix + "_" + strings.ReplaceAll(key, ".", "_"))
}

// ConfigKey converts an environment variable name to config key
func ConfigKey(prefix, envVar string) string {
	key := strings.ToLower(strings.TrimPrefix(envVar, strings.ToUpper(prefix+"_")))
	return strings.ReplaceAll(key, "_", ".")
}
