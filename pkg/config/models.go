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



// OpenAILLMConfig holds configuration for the OpenAI LLM provider.
type OpenAILLMConfig struct {
	Model            string   `mapstructure:"model_name" yaml:"model_name"`
	APIKey           string   `mapstructure:"api_key" yaml:"api_key"`
	Temperature      float64  `mapstructure:"temperature" yaml:"temperature,omitempty"`
	MaxTokens        int      `mapstructure:"max_tokens" yaml:"max_tokens,omitempty"`
	TopP             float64  `mapstructure:"top_p" yaml:"top_p,omitempty"`
	FrequencyPenalty float64  `mapstructure:"frequency_penalty" yaml:"frequency_penalty,omitempty"`
	PresencePenalty  float64  `mapstructure:"presence_penalty" yaml:"presence_penalty,omitempty"`
	Stop             []string `mapstructure:"stop" yaml:"stop,omitempty"`
	Streaming        bool     `mapstructure:"streaming" yaml:"streaming,omitempty"`
	APIVersion       string   `mapstructure:"api_version" yaml:"api_version,omitempty"`   // Optional: For Azure OpenAI
	APIEndpoint      string   `mapstructure:"api_endpoint" yaml:"api_endpoint,omitempty"` // Optional: For Azure OpenAI or other proxies
	Timeout          int      `mapstructure:"timeout_seconds" yaml:"timeout_seconds,omitempty"`
}

// MockLLMConfig holds configuration for the Mock LLM provider.
type MockLLMConfig struct {
	ModelName     string                 `mapstructure:"model_name" yaml:"model_name"`
	ExpectedError string                 `mapstructure:"expected_error" yaml:"expected_error,omitempty"`
	Responses     []string               `mapstructure:"responses" yaml:"responses,omitempty"`
	ToolCalls     []map[string]interface{} `mapstructure:"tool_calls" yaml:"tool_calls,omitempty"` // Simplified for now
}
