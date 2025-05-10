package schema

// AgentConfig defines the configuration for an agent.
// It includes parameters for LLM provider, tools, memory, and agent metadata.
type AgentConfig struct {
	Name         string `yaml:"name" json:"name"`
	Role         string `yaml:"role" json:"role"`
	Goal         string `yaml:"goal" json:"goal"`
	Backstory    string `yaml:"backstory" json:"backstory"`

	LLMProviderConfigName string `yaml:"llm_provider_config_name" json:"llm_provider_config_name"` // Name of the LLMProviderConfig to use from global config
	ToolNames             []string `yaml:"tool_names" json:"tool_names"`                         // Names of tools to be loaded from the ToolRegistry
	MemoryType            string `yaml:"memory_type" json:"memory_type"`                         // Type of memory to use (e.g., "buffer", "vector")
	MemoryConfigName      string `yaml:"memory_config_name" json:"memory_config_name"`           // Name of the specific memory configuration (e.g., for a specific VectorStore provider)

	// MaxIterations limits the number of steps an agent can take.
	MaxIterations int `yaml:"max_iterations" json:"max_iterations"`
	// MaxRetries limits the number of retries for recoverable errors.
	MaxRetries int `yaml:"max_retries" json:"max_retries"`
	// Verbose enables detailed logging for the agent's operations.
	Verbose bool `yaml:"verbose" json:"verbose"`
	// AllowDelegation indicates if the agent can delegate tasks to other agents.
	AllowDelegation bool `yaml:"allow_delegation" json:"allow_delegation"`

	// ProviderSpecificConfig can hold any additional configuration specific to a custom agent type or its providers.
	ProviderSpecificConfig map[string]interface{} `yaml:"provider_specific_config,omitempty" json:"provider_specific_config,omitempty"`
}

