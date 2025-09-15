package iface

import (
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// Config is the root configuration structure for the application.
// It holds configurations for various providers and components.
type Config struct {
	LLMProviders       []schema.LLMProviderConfig       `mapstructure:"llm_providers" yaml:"llm_providers"`
	EmbeddingProviders []schema.EmbeddingProviderConfig `mapstructure:"embedding_providers" yaml:"embedding_providers"`
	VectorStores       []schema.VectorStoreConfig       `mapstructure:"vector_stores" yaml:"vector_stores"`
	Tools              []ToolConfig                     `mapstructure:"tools" yaml:"tools"`
	Agents             []schema.AgentConfig             `mapstructure:"agents" yaml:"agents"`
	// Add other global or component-specific configurations here
}

// ToolConfig defines the configuration for a specific tool instance.
type ToolConfig struct {
	Name        string                 `mapstructure:"name" yaml:"name" validate:"required"`         // Unique name for this tool instance
	Description string                 `mapstructure:"description" yaml:"description"`               // Description of what the tool does
	Provider    string                 `mapstructure:"provider" yaml:"provider" validate:"required"` // The provider for this tool (e.g., "echo", "calculator")
	Enabled     bool                   `mapstructure:"enabled" yaml:"enabled" default:"true"`        // Whether the tool is enabled
	Config      map[string]interface{} `mapstructure:"config" yaml:"config,omitempty"`               // Provider-specific configuration for the tool
}

// LoaderOptions contains options for configuring the loader
type LoaderOptions struct {
	ConfigName  string
	ConfigPaths []string
	EnvPrefix   string
	Validate    bool
	SetDefaults bool
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	var msgs []string
	for _, err := range e {
		msgs = append(msgs, err.Error())
	}
	return fmt.Sprintf("configuration validation failed: %s", strings.Join(msgs, "; "))
}
