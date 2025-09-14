package vectorstores

// Functional options for configuring VectorStore operations.
// These follow the functional options pattern for flexible and type-safe configuration.

// WithEmbedder sets the embedder to use for generating embeddings.
func WithEmbedder(embedder Embedder) Option {
	return func(c *Config) {
		c.Embedder = embedder
	}
}

// WithSearchK sets the number of similar documents to return in search operations.
func WithSearchK(k int) Option {
	return func(c *Config) {
		c.SearchK = k
	}
}

// WithScoreThreshold sets the minimum similarity score threshold for search results.
// Documents with scores below this threshold will be filtered out.
func WithScoreThreshold(threshold float32) Option {
	return func(c *Config) {
		c.ScoreThreshold = threshold
	}
}

// WithMetadataFilter adds a metadata filter for search operations.
// Only documents matching the filter criteria will be considered.
func WithMetadataFilter(key string, value interface{}) Option {
	return func(c *Config) {
		if c.MetadataFilters == nil {
			c.MetadataFilters = make(map[string]interface{})
		}
		c.MetadataFilters[key] = value
	}
}

// WithMetadataFilters sets multiple metadata filters for search operations.
func WithMetadataFilters(filters map[string]interface{}) Option {
	return func(c *Config) {
		if c.MetadataFilters == nil {
			c.MetadataFilters = make(map[string]interface{})
		}
		for k, v := range filters {
			c.MetadataFilters[k] = v
		}
	}
}

// WithProviderConfig sets provider-specific configuration options.
func WithProviderConfig(key string, value interface{}) Option {
	return func(c *Config) {
		if c.ProviderConfig == nil {
			c.ProviderConfig = make(map[string]interface{})
		}
		c.ProviderConfig[key] = value
	}
}

// WithProviderConfigs sets multiple provider-specific configuration options.
func WithProviderConfigs(config map[string]interface{}) Option {
	return func(c *Config) {
		if c.ProviderConfig == nil {
			c.ProviderConfig = make(map[string]interface{})
		}
		for k, v := range config {
			c.ProviderConfig[k] = v
		}
	}
}

// NewDefaultConfig creates a new Config with default values.
func NewDefaultConfig() *Config {
	return &Config{
		SearchK:        5,
		ScoreThreshold: 0.0,
	}
}

// ApplyOptions applies a slice of options to a Config.
func ApplyOptions(config *Config, opts ...Option) {
	for _, opt := range opts {
		opt(config)
	}
}

// CloneConfig creates a deep copy of a Config.
func CloneConfig(original *Config) *Config {
	if original == nil {
		return NewDefaultConfig()
	}

	clone := &Config{
		Embedder:       original.Embedder,
		SearchK:        original.SearchK,
		ScoreThreshold: original.ScoreThreshold,
	}

	// Deep copy metadata filters
	if original.MetadataFilters != nil {
		clone.MetadataFilters = make(map[string]interface{})
		for k, v := range original.MetadataFilters {
			clone.MetadataFilters[k] = v
		}
	}

	// Deep copy provider config
	if original.ProviderConfig != nil {
		clone.ProviderConfig = make(map[string]interface{})
		for k, v := range original.ProviderConfig {
			clone.ProviderConfig[k] = v
		}
	}

	return clone
}
