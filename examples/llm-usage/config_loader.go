package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/llms"
	"gopkg.in/yaml.v3"
)

// ConfigFile represents the structure of the YAML configuration file
type ConfigFile struct {
	LLMs         LLMsConfig           `yaml:"llms"`
	Global       GlobalConfig         `yaml:"global,omitempty"`
	Environments map[string]EnvConfig `yaml:"environments,omitempty"`
}

type LLMsConfig struct {
	PrimaryProvider string           `yaml:"primary_provider"`
	Providers       []ProviderConfig `yaml:"providers"`
}

type ProviderConfig struct {
	Name                 string                 `yaml:"name"`
	Provider             string                 `yaml:"provider"`
	ModelName            string                 `yaml:"model_name"`
	APIKey               string                 `yaml:"api_key"`
	BaseURL              string                 `yaml:"base_url,omitempty"`
	Temperature          *float32               `yaml:"temperature,omitempty"`
	TopP                 *float32               `yaml:"top_p,omitempty"`
	MaxTokens            *int                   `yaml:"max_tokens,omitempty"`
	StopSequences        []string               `yaml:"stop_sequences,omitempty"`
	FrequencyPenalty     *float32               `yaml:"frequency_penalty,omitempty"`
	PresencePenalty      *float32               `yaml:"presence_penalty,omitempty"`
	MaxConcurrentBatches int                    `yaml:"max_concurrent_batches,omitempty"`
	RetryConfig          RetryConfig            `yaml:"retry_config,omitempty"`
	Observability        ObservabilityConfig    `yaml:"observability,omitempty"`
	ProviderSpecific     map[string]interface{} `yaml:"provider_specific,omitempty"`
}

type RetryConfig struct {
	MaxRetries int     `yaml:"max_retries,omitempty"`
	Delay      string  `yaml:"delay,omitempty"`
	Backoff    float64 `yaml:"backoff,omitempty"`
}

type ObservabilityConfig struct {
	Tracing           *bool `yaml:"tracing,omitempty"`
	Metrics           *bool `yaml:"metrics,omitempty"`
	StructuredLogging *bool `yaml:"structured_logging,omitempty"`
}

type GlobalConfig struct {
	Defaults      DefaultsConfig      `yaml:"defaults,omitempty"`
	Observability ObservabilityConfig `yaml:"observability,omitempty"`
	ErrorHandling ErrorHandlingConfig `yaml:"error_handling,omitempty"`
	Failover      FailoverConfig      `yaml:"failover,omitempty"`
}

type DefaultsConfig struct {
	Timeout           string `yaml:"timeout,omitempty"`
	EnableStreaming   *bool  `yaml:"enable_streaming,omitempty"`
	EnableToolCalling *bool  `yaml:"enable_tool_calling,omitempty"`
}

type ErrorHandlingConfig struct {
	EnableRetry             *bool  `yaml:"enable_retry,omitempty"`
	EnableCircuitBreaker    *bool  `yaml:"enable_circuit_breaker,omitempty"`
	CircuitBreakerThreshold int    `yaml:"circuit_breaker_threshold,omitempty"`
	CircuitBreakerTimeout   string `yaml:"circuit_breaker_timeout,omitempty"`
}

type FailoverConfig struct {
	Enabled             *bool  `yaml:"enabled,omitempty"`
	Strategy            string `yaml:"strategy,omitempty"`
	HealthCheckInterval string `yaml:"health_check_interval,omitempty"`
	UnhealthyThreshold  int    `yaml:"unhealthy_threshold,omitempty"`
}

type EnvConfig struct {
	LLMs   LLMsConfig   `yaml:"llms,omitempty"`
	Global GlobalConfig `yaml:"global,omitempty"`
}

// ConfigLoader provides functionality to load and parse LLM configurations
type ConfigLoader struct {
	configFile  string
	environment string
}

// NewConfigLoader creates a new configuration loader
func NewConfigLoader(configFile, environment string) *ConfigLoader {
	return &ConfigLoader{
		configFile:  configFile,
		environment: environment,
	}
}

// LoadConfig loads and parses the configuration file
func (cl *ConfigLoader) LoadConfig() (*ConfigFile, error) {
	// Read the configuration file
	data, err := os.ReadFile(cl.configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config ConfigFile
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment-specific overrides
	if envConfig, exists := config.Environments[cl.environment]; exists {
		cl.applyEnvironmentOverrides(&config, &envConfig)
	}

	// Apply global defaults
	cl.applyGlobalDefaults(&config)

	return &config, nil
}

// applyEnvironmentOverrides applies environment-specific configuration
func (cl *ConfigLoader) applyEnvironmentOverrides(config *ConfigFile, envConfig *EnvConfig) {
	if envConfig.LLMs.PrimaryProvider != "" {
		config.LLMs.PrimaryProvider = envConfig.LLMs.PrimaryProvider
	}

	// Override providers if specified in environment
	if len(envConfig.LLMs.Providers) > 0 {
		config.LLMs.Providers = envConfig.LLMs.Providers
	}

	// Override global config if specified
	if envConfig.Global.Defaults.Timeout != "" {
		config.Global.Defaults = envConfig.Global.Defaults
	}
}

// applyGlobalDefaults applies global defaults to all providers
func (cl *ConfigLoader) applyGlobalDefaults(config *ConfigFile) {
	for i := range config.LLMs.Providers {
		provider := &config.LLMs.Providers[i]

		// Apply default timeout if not set
		if config.Global.Defaults.Timeout != "" && provider.RetryConfig.Delay == "" {
			// Note: In a real implementation, you might want to set a default timeout
			// This is simplified for the example
		}

		// Apply default observability settings if not set
		if provider.Observability.Tracing == nil {
			provider.Observability.Tracing = config.Global.Observability.Tracing
		}
		if provider.Observability.Metrics == nil {
			provider.Observability.Metrics = config.Global.Observability.Metrics
		}
		if provider.Observability.StructuredLogging == nil {
			provider.Observability.StructuredLogging = config.Global.Observability.StructuredLogging
		}
	}
}

// CreateLLMConfigs creates LLM configurations from the parsed config
func (cl *ConfigLoader) CreateLLMConfigs(config *ConfigFile) (map[string]*llms.Config, error) {
	llmConfigs := make(map[string]*llms.Config)

	for _, providerConfig := range config.LLMs.Providers {
		llmConfig, err := cl.createLLMConfig(&providerConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create config for provider %s: %w", providerConfig.Name, err)
		}

		llmConfigs[providerConfig.Name] = llmConfig
	}

	return llmConfigs, nil
}

// createLLMConfig creates a single LLM configuration from provider config
func (cl *ConfigLoader) createLLMConfig(providerConfig *ProviderConfig) (*llms.Config, error) {
	// Expand environment variables in API key
	apiKey := cl.expandEnvironmentVariables(providerConfig.APIKey)

	// Build configuration with functional options
	configOpts := []llms.ConfigOption{
		llms.WithProvider(providerConfig.Provider),
		llms.WithModelName(providerConfig.ModelName),
		llms.WithAPIKey(apiKey),
	}

	// Add optional configurations
	if providerConfig.BaseURL != "" {
		configOpts = append(configOpts, llms.WithBaseURL(providerConfig.BaseURL))
	}

	if providerConfig.Temperature != nil {
		configOpts = append(configOpts, llms.WithTemperatureConfig(*providerConfig.Temperature))
	}

	if providerConfig.TopP != nil {
		configOpts = append(configOpts, llms.WithTopPConfig(*providerConfig.TopP))
	}

	if providerConfig.MaxTokens != nil {
		configOpts = append(configOpts, llms.WithMaxTokensConfig(*providerConfig.MaxTokens))
	}

	if len(providerConfig.StopSequences) > 0 {
		configOpts = append(configOpts, llms.WithStopSequences(providerConfig.StopSequences))
	}

	if providerConfig.FrequencyPenalty != nil {
		// Note: FrequencyPenalty would need to be added to ConfigOption if not already present
	}

	if providerConfig.MaxConcurrentBatches > 0 {
		configOpts = append(configOpts, llms.WithMaxConcurrentBatches(providerConfig.MaxConcurrentBatches))
	}

	// Add retry configuration
	if providerConfig.RetryConfig.MaxRetries > 0 {
		delay := cl.parseDuration(providerConfig.RetryConfig.Delay, time.Second)
		configOpts = append(configOpts, llms.WithRetryConfig(
			providerConfig.RetryConfig.MaxRetries,
			delay,
			providerConfig.RetryConfig.Backoff,
		))
	}

	// Add observability configuration
	tracing := true
	metrics := true
	logging := true

	if providerConfig.Observability.Tracing != nil {
		tracing = *providerConfig.Observability.Tracing
	}
	if providerConfig.Observability.Metrics != nil {
		metrics = *providerConfig.Observability.Metrics
	}
	if providerConfig.Observability.StructuredLogging != nil {
		logging = *providerConfig.Observability.StructuredLogging
	}

	configOpts = append(configOpts, llms.WithObservability(tracing, metrics, logging))

	// Add provider-specific configuration
	if len(providerConfig.ProviderSpecific) > 0 {
		for key, value := range providerConfig.ProviderSpecific {
			if strValue, ok := value.(string); ok {
				configOpts = append(configOpts, llms.WithProviderSpecific(key, cl.expandEnvironmentVariables(strValue)))
			} else {
				configOpts = append(configOpts, llms.WithProviderSpecific(key, value))
			}
		}
	}

	// Create the configuration
	llmConfig := llms.NewConfig(configOpts...)

	return llmConfig, nil
}

// expandEnvironmentVariables expands environment variables in a string
func (cl *ConfigLoader) expandEnvironmentVariables(input string) string {
	if strings.HasPrefix(input, "${") && strings.HasSuffix(input, "}") {
		envVar := strings.TrimSuffix(strings.TrimPrefix(input, "${"), "}")
		if value := os.Getenv(envVar); value != "" {
			return value
		}
	}
	return input
}

// parseDuration parses a duration string with a default fallback
func (cl *ConfigLoader) parseDuration(durationStr string, defaultDuration time.Duration) time.Duration {
	if durationStr == "" {
		return defaultDuration
	}

	if duration, err := time.ParseDuration(durationStr); err == nil {
		return duration
	}

	return defaultDuration
}

// DemonstrateConfigLoading shows how to use the configuration loader
func DemonstrateConfigLoading() {
	fmt.Println("\n📄 Example: Configuration File Loading")

	loader := NewConfigLoader("config_example.yaml", "development")

	// Load and parse configuration
	config, err := loader.LoadConfig()
	if err != nil {
		fmt.Printf("❌ Failed to load configuration: %v\n", err)
		fmt.Printf("💡 Make sure config_example.yaml exists in the current directory\n")
		return
	}

	fmt.Printf("✅ Configuration loaded successfully\n")
	fmt.Printf("📋 Primary provider: %s\n", config.LLMs.PrimaryProvider)
	fmt.Printf("📊 Number of providers: %d\n", len(config.LLMs.Providers))

	// Create LLM configurations
	llmConfigs, err := loader.CreateLLMConfigs(config)
	if err != nil {
		fmt.Printf("❌ Failed to create LLM configurations: %v\n", err)
		return
	}

	fmt.Printf("✅ Created %d LLM configurations\n", len(llmConfigs))

	// Validate configurations
	ctx := context.Background()
	for name, llmConfig := range llmConfigs {
		if err := llms.ValidateProviderConfig(ctx, llmConfig); err != nil {
			fmt.Printf("⚠️  %s config validation failed: %v\n", name, err)
		} else {
			fmt.Printf("✅ %s configuration validated\n", name)
		}
	}

	fmt.Printf("💡 Configuration loading benefits:\n")
	fmt.Printf("   - Environment-specific configurations\n")
	fmt.Printf("   - Environment variable substitution\n")
	fmt.Printf("   - Centralized configuration management\n")
	fmt.Printf("   - Easy deployment configuration\n")
	fmt.Printf("   - Validation and error checking\n")
}

// Integration with main example
func init() {
	// Add config loading demonstration to the main execution flow
	go func() {
		time.Sleep(300 * time.Millisecond)
		DemonstrateConfigLoading()
	}()
}
