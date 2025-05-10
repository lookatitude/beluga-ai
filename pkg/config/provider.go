package config

import "github.com/lookatitude/beluga-ai/pkg/schema"

// Provider defines the interface for a configuration provider.
// It is responsible for loading configuration data from a source (e.g., file, environment variables)
// into a given struct.
type Provider interface {
	// Load populates the given configStruct with configuration values.
	// The configStruct should be a pointer to a struct that can be unmarshalled into.
	Load(configStruct interface{}) error

	// UnmarshalKey decodes the configuration at a specific key into a struct.
	// rawVal should be a pointer to a struct.
	UnmarshalKey(key string, rawVal interface{}) error

	// GetString retrieves a string configuration value by key.
	GetString(key string) string
	// GetInt retrieves an integer configuration value by key.
	GetInt(key string) int
	// GetBool retrieves a boolean configuration value by key.
	GetBool(key string) bool
	// GetFloat64 retrieves a float64 configuration value by key.
	GetFloat64(key string) float64
	// GetStringMapString retrieves a map[string]string configuration value by key.
	GetStringMapString(key string) map[string]string
	// IsSet checks if a key is set in the configuration.
	IsSet(key string) bool

	// GetLLMProviderConfig retrieves a specific LLMProviderConfig by name.
	// This is a more specific getter for convenience.
	GetLLMProviderConfig(name string) (schema.LLMProviderConfig, error)

	// GetLLMProvidersConfig retrieves all LLMProviderConfig.
	GetLLMProvidersConfig() ([]schema.LLMProviderConfig, error) // Added this method

	// GetEmbeddingProvidersConfig retrieves all EmbeddingProviderConfig.
	// Assuming this method exists or needs to be added for consistency with GetLLMProvidersConfig
	GetEmbeddingProvidersConfig() ([]schema.EmbeddingProviderConfig, error) // Ensuring this is present

	// GetAgentConfig retrieves a specific AgentConfig by name.
	GetAgentConfig(name string) (schema.AgentConfig, error) // Assuming AgentConfig is in schema

	// GetToolConfig retrieves a specific ToolConfig by name from the main config.
	GetToolConfig(name string) (ToolConfig, error) // Using local ToolConfig struct

	// GetMemoryProviderConfig retrieves a specific MemoryProviderConfig by name.
	// This is a more specific getter for convenience.
	// GetMemoryProviderConfig(name string) (schema.MemoryProviderConfig, error) // Placeholder for now
}

