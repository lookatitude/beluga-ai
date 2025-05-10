package config

// GlobalConfig is a placeholder for a global application configuration structure.
// Specific components can define their own configuration structs that can be loaded
// by a config.Provider.
// For example, an LLM provider might have its own LLMConfig struct.
type GlobalConfig struct {
	AppName    string `mapstructure:"app_name"`
	LogLevel   string `mapstructure:"log_level"`
	ServerPort int    `mapstructure:"server_port"`
	// Other global settings can be added here.
}

// ComponentConfig is an example of how a specific component might define its configuration.
// This would typically be defined within the component's own package, but is shown here for illustration.
type ComponentConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	APIKey  string `mapstructure:"api_key"`
	Timeout int    `mapstructure:"timeout_seconds"`
}



// OpenAIEmbedderConfig holds configuration for the OpenAI Embedder.
type OpenAIEmbedderConfig struct {
	APIKey      string `mapstructure:"api_key"`
	Model       string `mapstructure:"model"`         // e.g., "text-embedding-ada-002"
	APIVersion  string `mapstructure:"api_version"`   // Optional: For Azure OpenAI
	APIEndpoint string `mapstructure:"api_endpoint"` // Optional: For Azure OpenAI or other proxies
	Timeout     int    `mapstructure:"timeout_seconds"`
}



// MockEmbedderConfig holds configuration for the Mock Embedder.
type MockEmbedderConfig struct {
	Dimension    int   `mapstructure:"dimension"`
	Seed         int64 `mapstructure:"seed"`
	RandomizeNil bool  `mapstructure:"randomize_nil"`
}




// ToolConfig defines the configuration for a specific tool instance.
type ToolConfig struct {
	Name        string                 `mapstructure:"name" yaml:"name"`                 // Unique name for this tool instance
	Description string                 `mapstructure:"description" yaml:"description"`     // Description of what the tool does
	Provider    string                 `mapstructure:"provider" yaml:"provider"`           // The provider for this tool (e.g., "echo", "calculator")
	Enabled     bool                   `mapstructure:"enabled" yaml:"enabled"`             // Whether the tool is enabled
	Config      map[string]interface{} `mapstructure:"config" yaml:"config,omitempty"` // Provider-specific configuration for the tool
}
