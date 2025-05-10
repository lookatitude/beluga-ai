package schema

// AgentConfig defines the configuration for an agent.
// This can include settings for the LLM, tools, memory, and other agent-specific parameters.
type AgentConfig struct {
	Name string `yaml:"name" json:"name"` // Unique name for this agent configuration

	// LLMProviderName specifies the name of the LLMProviderConfig to use for this agent.
	// This should correspond to a name defined in the LLM provider configurations.
	LLMProviderName string `yaml:"llm_provider_name" json:"llm_provider_name"`

	// LLMProviderConfigName is an alias or specific field for the LLM provider config name.
	// It seems the factory was expecting this exact name.
	LLMProviderConfigName string `yaml:"llm_provider_config_name,omitempty" json:"llm_provider_config_name,omitempty"`

	// ToolNames lists the names of the tools available to this agent.
	// These names should correspond to tool configurations.
	ToolNames []string `yaml:"tool_names,omitempty" json:"tool_names,omitempty"`

	// MemoryProviderName specifies the name of the MemoryProviderConfig to use for this agent.
	// This should correspond to a name defined in the memory provider configurations.
	MemoryProviderName string `yaml:"memory_provider_name,omitempty" json:"memory_provider_name,omitempty"`

	// MemoryType specifies the type of memory to use (e.g., "buffer", "vector").
	MemoryType string `yaml:"memory_type,omitempty" json:"memory_type,omitempty"`

	// MemoryConfigName specifies the name of the specific memory configuration to use.
	// This would correspond to a named configuration within the memory provider settings.
	MemoryConfigName string `yaml:"memory_config_name,omitempty" json:"memory_config_name,omitempty"`

	// MaxIterations defines the maximum number of steps the agent can take before stopping.
	// This is a safety measure to prevent infinite loops.
	MaxIterations int `yaml:"max_iterations,omitempty" json:"max_iterations,omitempty"`

	// PromptTemplate is the main prompt template used by the agent.
	// It can be a string or a path to a template file.
	PromptTemplate string `yaml:"prompt_template,omitempty" json:"prompt_template,omitempty"`

	// OutputParser defines how the agent's output is parsed.
	// This could be the name of a registered output parser or configuration for one.
	OutputParser string `yaml:"output_parser,omitempty" json:"output_parser,omitempty"`

	// AgentType specifies the type of agent (e.g., "react", "openai_tools").
	AgentType string `yaml:"agent_type,omitempty" json:"agent_type,omitempty"`

	// Additional agent-specific configuration can be added here or in a map.
	// For example, specific settings for a ReAct agent or an OpenAI tools agent.
	AgentSpecificConfig map[string]interface{} `yaml:"agent_specific_config,omitempty" json:"agent_specific_config,omitempty"`
}

