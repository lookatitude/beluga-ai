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

