package config

// Provider defines the interface for a configuration provider.
// It is responsible for loading configuration data from a source (e.g., file, environment variables)
// into a given struct.
type Provider interface {
	// Load populates the given configStruct with configuration values.
	// The configStruct should be a pointer to a struct that can be unmarshalled into.
	Load(configStruct interface{}) error

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
}

