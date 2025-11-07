package schema

import (
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
)

// AgentConfig defines the configuration for an agent.
// This can include settings for the LLM, tools, memory, and other agent-specific parameters.
type AgentConfig struct {
	Name string `mapstructure:"name" yaml:"name" json:"name" validate:"required"` // Unique name for this agent configuration

	// LLMProviderName specifies the name of the LLMProviderConfig to use for this agent.
	// This should correspond to a name defined in the LLM provider configurations.
	LLMProviderName string `mapstructure:"llm_provider_name" yaml:"llm_provider_name" json:"llm_provider_name" validate:"required"`

	// LLMProviderConfigName is an alias or specific field for the LLM provider config name.
	// It seems the factory was expecting this exact name.
	LLMProviderConfigName string `mapstructure:"llm_provider_config_name" yaml:"llm_provider_config_name,omitempty" json:"llm_provider_config_name,omitempty"`

	// ToolNames lists the names of the tools available to this agent.
	// These names should correspond to tool configurations.
	ToolNames []string `mapstructure:"tool_names" yaml:"tool_names,omitempty" json:"tool_names,omitempty"`

	// MemoryProviderName specifies the name of the MemoryProviderConfig to use for this agent.
	// This should correspond to a name defined in the memory provider configurations.
	MemoryProviderName string `mapstructure:"memory_provider_name" yaml:"memory_provider_name,omitempty" json:"memory_provider_name,omitempty"`

	// MemoryType specifies the type of memory to use (e.g., "buffer", "vector").
	MemoryType string `mapstructure:"memory_type" yaml:"memory_type,omitempty" json:"memory_type,omitempty"`

	// MemoryConfigName specifies the name of the specific memory configuration to use.
	// This would correspond to a named configuration within the memory provider settings.
	MemoryConfigName string `mapstructure:"memory_config_name" yaml:"memory_config_name,omitempty" json:"memory_config_name,omitempty"`

	// MaxIterations defines the maximum number of steps the agent can take before stopping.
	// This is a safety measure to prevent infinite loops.
	MaxIterations int `mapstructure:"max_iterations" yaml:"max_iterations,omitempty" json:"max_iterations,omitempty" validate:"min=1"`

	Settings map[string]interface{} `mapstructure:"settings" yaml:"settings,omitempty" json:"settings,omitempty"`

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

// LLMProviderConfig defines the configuration for a specific LLM provider instance.
// It allows for common parameters and a flexible way to include provider-specific settings.
type LLMProviderConfig struct {
	Name string `mapstructure:"name" yaml:"name" json:"name" validate:"required"` // Unique name for this configuration instance (e.g., "openai_gpt4_turbo", "anthropic_claude3_opus")
	// Provider identifies the type of LLM provider (e.g., "openai", "anthropic", "gemini", "ollama").
	// This will be used by the LLMProviderFactory to instantiate the correct client.
	Provider string `mapstructure:"provider" yaml:"provider" json:"provider" validate:"required"`

	// ModelName specifies the exact model to be used (e.g., "gpt-4-turbo-preview", "claude-3-opus-20240229").
	ModelName string `mapstructure:"model_name" yaml:"model_name" json:"model_name" validate:"required"`

	// APIKey is the API key for the LLM provider, if required.
	// It is recommended to manage this securely, e.g., via environment variables or a secrets manager,
	// and have the configuration loader resolve it.
	APIKey string `mapstructure:"api_key" yaml:"api_key,omitempty" json:"api_key,omitempty"`

	// BaseURL can be used to specify a custom API endpoint, e.g., for self-hosted models or proxies.
	BaseURL string `mapstructure:"base_url" yaml:"base_url,omitempty" json:"base_url,omitempty"`

	// DefaultCallOptions holds common LLM call parameters that can be overridden at runtime.
	DefaultCallOptions map[string]interface{} `mapstructure:"default_call_options" yaml:"default_call_options,omitempty" json:"default_call_options,omitempty"`
	// Example DefaultCallOptions:
	// "temperature": 0.7
	// "max_tokens": 1024
	// "top_p": 1.0

	// ProviderSpecific holds any additional configuration parameters unique to the LLM provider.
	// This allows for flexibility in supporting diverse provider APIs without cluttering the main struct.
	// For example, for Ollama, this might include "keep_alive" or "num_ctx".
	// For OpenAI, it might include "organization_id".
	ProviderSpecific map[string]interface{} `mapstructure:"provider_specific" yaml:"provider_specific,omitempty" json:"provider_specific,omitempty"`
}

// EmbeddingProviderConfig defines the configuration for a specific embedding provider instance.
type EmbeddingProviderConfig struct {
	Name             string                 `mapstructure:"name" yaml:"name" json:"name" validate:"required"`
	Provider         string                 `mapstructure:"provider" yaml:"provider" json:"provider" validate:"required"`
	ModelName        string                 `mapstructure:"model_name" yaml:"model_name" json:"model_name" validate:"required"`
	APIKey           string                 `mapstructure:"api_key" yaml:"api_key" json:"api_key" validate:"required"`
	BaseURL          string                 `mapstructure:"base_url" yaml:"base_url,omitempty" json:"base_url,omitempty"`
	ProviderSpecific map[string]interface{} `mapstructure:"provider_specific" yaml:"provider_specific,omitempty" json:"provider_specific,omitempty"`
}

// VectorStoreConfig defines the configuration for a vector store provider.
type VectorStoreConfig struct {
	Name             string                 `mapstructure:"name" yaml:"name" validate:"required"`                           // Unique name for this vector store configuration
	Provider         string                 `mapstructure:"provider" yaml:"provider" validate:"required"`                   // e.g., "inmemory", "pgvector", "pinecone"
	ConnectionString string                 `mapstructure:"connection_string,omitempty" yaml:"connection_string,omitempty"` // Optional: Connection string if applicable
	ProviderSpecific map[string]interface{} `mapstructure:"provider_specific,omitempty" yaml:"provider_specific,omitempty"` // Provider-specific settings
}

// Validate validates the AgentConfig struct.
func (c *AgentConfig) Validate() error {
	validate := validator.New()
	return validate.Struct(c)
}

// Validate validates the LLMProviderConfig struct.
func (c *LLMProviderConfig) Validate() error {
	validate := validator.New()
	return validate.Struct(c)
}

// Validate validates the EmbeddingProviderConfig struct.
func (c *EmbeddingProviderConfig) Validate() error {
	validate := validator.New()
	return validate.Struct(c)
}

// Validate validates the VectorStoreConfig struct.
func (c *VectorStoreConfig) Validate() error {
	validate := validator.New()
	return validate.Struct(c)
}

// NewAgentConfig creates a new AgentConfig with validation.
func NewAgentConfig(name, llmProviderName string, opts ...AgentOption) (*AgentConfig, error) {
	config := &AgentConfig{
		Name:            name,
		LLMProviderName: llmProviderName,
		MaxIterations:   10, // sensible default
	}

	for _, opt := range opts {
		opt(config)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid agent config: %w", err)
	}

	return config, nil
}

// NewLLMProviderConfig creates a new LLMProviderConfig with validation.
func NewLLMProviderConfig(name, provider, modelName string, opts ...LLMProviderOption) (*LLMProviderConfig, error) {
	config := &LLMProviderConfig{
		Name:               name,
		Provider:           provider,
		ModelName:          modelName,
		DefaultCallOptions: make(map[string]interface{}),
		ProviderSpecific:   make(map[string]interface{}),
	}

	for _, opt := range opts {
		opt(config)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid LLM provider config: %w", err)
	}

	return config, nil
}

// NewEmbeddingProviderConfig creates a new EmbeddingProviderConfig with validation.
func NewEmbeddingProviderConfig(name, provider, modelName, apiKey string, opts ...EmbeddingOption) (*EmbeddingProviderConfig, error) {
	config := &EmbeddingProviderConfig{
		Name:             name,
		Provider:         provider,
		ModelName:        modelName,
		APIKey:           apiKey,
		ProviderSpecific: make(map[string]interface{}),
	}

	for _, opt := range opts {
		opt(config)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid embedding provider config: %w", err)
	}

	return config, nil
}

// NewVectorStoreConfig creates a new VectorStoreConfig with validation.
func NewVectorStoreConfig(name, provider string, opts ...VectorStoreOption) (*VectorStoreConfig, error) {
	config := &VectorStoreConfig{
		Name:             name,
		Provider:         provider,
		ProviderSpecific: make(map[string]interface{}),
	}

	for _, opt := range opts {
		opt(config)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid vector store config: %w", err)
	}

	return config, nil
}

// NewChatHistoryConfig creates a new ChatHistoryConfig with validation.
func NewChatHistoryConfig(opts ...ChatHistoryOption) (*ChatHistoryConfig, error) {
	config := &ChatHistoryConfig{
		MaxMessages: 100, // sensible default
		TTL:         0,   // no TTL by default
	}

	for _, opt := range opts {
		opt(config)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid chat history config: %w", err)
	}

	return config, nil
}

// SchemaValidationConfig defines configuration for schema validation rules.
type SchemaValidationConfig struct {
	// EnableStrictValidation enables strict validation of all schema constraints
	EnableStrictValidation bool `yaml:"enable_strict_validation" json:"enable_strict_validation"`

	// MaxMessageLength defines the maximum allowed length for message content
	MaxMessageLength int `yaml:"max_message_length" json:"max_message_length" validate:"min=1"`

	// MaxMetadataSize defines the maximum size of metadata maps
	MaxMetadataSize int `yaml:"max_metadata_size" json:"max_metadata_size" validate:"min=1"`

	// MaxToolCalls defines the maximum number of tool calls allowed per message
	MaxToolCalls int `yaml:"max_tool_calls" json:"max_tool_calls" validate:"min=1"`

	// MaxEmbeddingDimensions defines the maximum dimensions for embedding vectors
	MaxEmbeddingDimensions int `yaml:"max_embedding_dimensions" json:"max_embedding_dimensions" validate:"min=1"`

	// AllowedMessageTypes defines the allowed message types for validation
	AllowedMessageTypes []string `yaml:"allowed_message_types" json:"allowed_message_types"`

	// RequiredMetadataFields defines fields that must be present in metadata
	RequiredMetadataFields []string `yaml:"required_metadata_fields" json:"required_metadata_fields"`

	// EnableContentValidation enables validation of message content format
	EnableContentValidation bool `yaml:"enable_content_validation" json:"enable_content_validation"`

	// CustomValidationRules holds custom validation rules as key-value pairs
	CustomValidationRules map[string]interface{} `yaml:"custom_validation_rules" json:"custom_validation_rules"`
}

// Validate validates the SchemaValidationConfig struct.
func (c *SchemaValidationConfig) Validate() error {
	validate := validator.New()
	return validate.Struct(c)
}

// NewSchemaValidationConfig creates a new SchemaValidationConfig with validation.
func NewSchemaValidationConfig(opts ...SchemaValidationOption) (*SchemaValidationConfig, error) {
	config := &SchemaValidationConfig{
		EnableStrictValidation:  true,
		MaxMessageLength:        10000, // sensible default
		MaxMetadataSize:         100,   // sensible default
		MaxToolCalls:            10,    // sensible default
		MaxEmbeddingDimensions:  1536,  // common embedding dimension
		AllowedMessageTypes:     []string{"human", "ai", "system", "tool", "function"},
		RequiredMetadataFields:  []string{},
		EnableContentValidation: true,
		CustomValidationRules:   make(map[string]interface{}),
	}

	for _, opt := range opts {
		opt(config)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid schema validation config: %w", err)
	}

	return config, nil
}

// SchemaValidationOption defines a function type for SchemaValidationConfig options.
type SchemaValidationOption func(*SchemaValidationConfig)

// WithStrictValidation enables or disables strict validation.
func WithStrictValidation(enable bool) SchemaValidationOption {
	return func(c *SchemaValidationConfig) {
		c.EnableStrictValidation = enable
	}
}

// WithMaxMessageLength sets the maximum message length.
func WithMaxMessageLength(maxLength int) SchemaValidationOption {
	return func(c *SchemaValidationConfig) {
		c.MaxMessageLength = maxLength
	}
}

// WithMaxMetadataSize sets the maximum metadata size.
func WithMaxMetadataSize(maxSize int) SchemaValidationOption {
	return func(c *SchemaValidationConfig) {
		c.MaxMetadataSize = maxSize
	}
}

// WithMaxToolCalls sets the maximum number of tool calls.
func WithMaxToolCalls(maxCalls int) SchemaValidationOption {
	return func(c *SchemaValidationConfig) {
		c.MaxToolCalls = maxCalls
	}
}

// WithAllowedMessageTypes sets the allowed message types.
func WithAllowedMessageTypes(types []string) SchemaValidationOption {
	return func(c *SchemaValidationConfig) {
		c.AllowedMessageTypes = types
	}
}

// WithRequiredMetadataFields sets required metadata fields.
func WithRequiredMetadataFields(fields []string) SchemaValidationOption {
	return func(c *SchemaValidationConfig) {
		c.RequiredMetadataFields = fields
	}
}

// WithContentValidation enables or disables content validation.
func WithContentValidation(enable bool) SchemaValidationOption {
	return func(c *SchemaValidationConfig) {
		c.EnableContentValidation = enable
	}
}

// Functional options for configuration

// AgentOption defines a function type for AgentConfig options.
type AgentOption func(*AgentConfig)

// WithToolNames sets the tool names for the agent.
func WithToolNames(toolNames []string) AgentOption {
	return func(c *AgentConfig) {
		c.ToolNames = toolNames
	}
}

// WithMemoryProvider sets the memory provider for the agent.
func WithMemoryProvider(providerName, memoryType string) AgentOption {
	return func(c *AgentConfig) {
		c.MemoryProviderName = providerName
		c.MemoryType = memoryType
	}
}

// WithMaxIterations sets the maximum iterations for the agent.
func WithMaxIterations(maxIterations int) AgentOption {
	return func(c *AgentConfig) {
		c.MaxIterations = maxIterations
	}
}

// WithPromptTemplate sets the prompt template for the agent.
func WithPromptTemplate(template string) AgentOption {
	return func(c *AgentConfig) {
		c.PromptTemplate = template
	}
}

// WithAgentType sets the agent type.
func WithAgentType(agentType string) AgentOption {
	return func(c *AgentConfig) {
		c.AgentType = agentType
	}
}

// LLMProviderOption defines a function type for LLMProviderConfig options.
type LLMProviderOption func(*LLMProviderConfig)

// WithAPIKey sets the API key for the LLM provider.
func WithAPIKey(apiKey string) LLMProviderOption {
	return func(c *LLMProviderConfig) {
		c.APIKey = apiKey
	}
}

// WithBaseURL sets the base URL for the LLM provider.
func WithBaseURL(baseURL string) LLMProviderOption {
	return func(c *LLMProviderConfig) {
		c.BaseURL = baseURL
	}
}

// WithDefaultCallOptions sets the default call options for the LLM provider.
func WithDefaultCallOptions(options map[string]interface{}) LLMProviderOption {
	return func(c *LLMProviderConfig) {
		c.DefaultCallOptions = options
	}
}

// WithProviderSpecific sets provider-specific configuration.
func WithProviderSpecific(specific map[string]interface{}) LLMProviderOption {
	return func(c *LLMProviderConfig) {
		c.ProviderSpecific = specific
	}
}

// EmbeddingOption defines a function type for EmbeddingProviderConfig options.
type EmbeddingOption func(*EmbeddingProviderConfig)

// WithEmbeddingBaseURL sets the base URL for the embedding provider.
func WithEmbeddingBaseURL(baseURL string) EmbeddingOption {
	return func(c *EmbeddingProviderConfig) {
		c.BaseURL = baseURL
	}
}

// WithEmbeddingProviderSpecific sets provider-specific configuration for embeddings.
func WithEmbeddingProviderSpecific(specific map[string]interface{}) EmbeddingOption {
	return func(c *EmbeddingProviderConfig) {
		c.ProviderSpecific = specific
	}
}

// VectorStoreOption defines a function type for VectorStoreConfig options.
type VectorStoreOption func(*VectorStoreConfig)

// WithConnectionString sets the connection string for the vector store.
func WithConnectionString(connectionString string) VectorStoreOption {
	return func(c *VectorStoreConfig) {
		c.ConnectionString = connectionString
	}
}

// WithVectorStoreProviderSpecific sets provider-specific configuration for vector store.
func WithVectorStoreProviderSpecific(specific map[string]interface{}) VectorStoreOption {
	return func(c *VectorStoreConfig) {
		c.ProviderSpecific = specific
	}
}

// ChatHistoryOption defines a function type for ChatHistoryConfig options.
type ChatHistoryOption func(*ChatHistoryConfig)

// WithMaxMessages sets the maximum number of messages to keep in history.
func WithMaxMessages(maxMessages int) ChatHistoryOption {
	return func(c *ChatHistoryConfig) {
		c.MaxMessages = maxMessages
	}
}

// WithTTL sets the time-to-live for messages in history.
func WithTTL(ttl time.Duration) ChatHistoryOption {
	return func(c *ChatHistoryConfig) {
		c.TTL = ttl
	}
}

// WithPersistence enables or disables persistence for chat history.
func WithPersistence(enable bool) ChatHistoryOption {
	return func(c *ChatHistoryConfig) {
		c.EnablePersistence = enable
	}
}
