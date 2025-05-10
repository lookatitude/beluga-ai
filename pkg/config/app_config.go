package config

import "github.com/lookatitude/beluga-ai/pkg/schema"

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

