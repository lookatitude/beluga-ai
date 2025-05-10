package schema

// VectorStoreConfig defines the configuration for a vector store provider.
type VectorStoreConfig struct {
	Name             string                 `mapstructure:"name" yaml:"name"`                         // Unique name for this vector store configuration
	Provider         string                 `mapstructure:"provider" yaml:"provider"`                   // e.g., "inmemory", "pgvector", "pinecone"
	ConnectionString string                 `mapstructure:"connection_string,omitempty" yaml:"connection_string,omitempty"` // Optional: Connection string if applicable
	ProviderSpecific map[string]interface{} `mapstructure:"provider_specific,omitempty" yaml:"provider_specific,omitempty"` // Provider-specific settings
}

