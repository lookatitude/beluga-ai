package schema

import (
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
)

// AgentConfig defines the configuration for an agent.
// This can include settings for the LLM, tools, memory, and other agent-specific parameters.
type AgentConfig struct {
	Settings              map[string]any `mapstructure:"settings" yaml:"settings,omitempty" json:"settings,omitempty"`
	AgentSpecificConfig   map[string]any `yaml:"agent_specific_config,omitempty" json:"agent_specific_config,omitempty"`
	PromptTemplate        string         `yaml:"prompt_template,omitempty" json:"prompt_template,omitempty"`
	MemoryProviderName    string         `mapstructure:"memory_provider_name" yaml:"memory_provider_name,omitempty" json:"memory_provider_name,omitempty"`
	MemoryType            string         `mapstructure:"memory_type" yaml:"memory_type,omitempty" json:"memory_type,omitempty"`
	MemoryConfigName      string         `mapstructure:"memory_config_name" yaml:"memory_config_name,omitempty" json:"memory_config_name,omitempty"`
	LLMProviderConfigName string         `mapstructure:"llm_provider_config_name" yaml:"llm_provider_config_name,omitempty" json:"llm_provider_config_name,omitempty"`
	Name                  string         `mapstructure:"name" yaml:"name" json:"name" validate:"required"`
	OutputParser          string         `yaml:"output_parser,omitempty" json:"output_parser,omitempty"`
	AgentType             string         `yaml:"agent_type,omitempty" json:"agent_type,omitempty"`
	LLMProviderName       string         `mapstructure:"llm_provider_name" yaml:"llm_provider_name" json:"llm_provider_name" validate:"required"`
	ToolNames             []string       `mapstructure:"tool_names" yaml:"tool_names,omitempty" json:"tool_names,omitempty"`
	MaxIterations         int            `mapstructure:"max_iterations" yaml:"max_iterations,omitempty" json:"max_iterations,omitempty" validate:"min=1"`
}

// LLMProviderConfig defines the configuration for a specific LLM provider instance.
// It allows for common parameters and a flexible way to include provider-specific settings.
type LLMProviderConfig struct {
	DefaultCallOptions map[string]any `mapstructure:"default_call_options" yaml:"default_call_options,omitempty" json:"default_call_options,omitempty"`
	ProviderSpecific   map[string]any `mapstructure:"provider_specific" yaml:"provider_specific,omitempty" json:"provider_specific,omitempty"`
	Name               string         `mapstructure:"name" yaml:"name" json:"name" validate:"required"`
	Provider           string         `mapstructure:"provider" yaml:"provider" json:"provider" validate:"required"`
	ModelName          string         `mapstructure:"model_name" yaml:"model_name" json:"model_name" validate:"required"`
	APIKey             string         `mapstructure:"api_key" yaml:"api_key,omitempty" json:"api_key,omitempty"`
	BaseURL            string         `mapstructure:"base_url" yaml:"base_url,omitempty" json:"base_url,omitempty"`
}

// EmbeddingProviderConfig defines the configuration for a specific embedding provider instance.
type EmbeddingProviderConfig struct {
	ProviderSpecific map[string]any `mapstructure:"provider_specific" yaml:"provider_specific,omitempty" json:"provider_specific,omitempty"`
	Name             string         `mapstructure:"name" yaml:"name" json:"name" validate:"required"`
	Provider         string         `mapstructure:"provider" yaml:"provider" json:"provider" validate:"required"`
	ModelName        string         `mapstructure:"model_name" yaml:"model_name" json:"model_name" validate:"required"`
	APIKey           string         `mapstructure:"api_key" yaml:"api_key" json:"api_key" validate:"required"`
	BaseURL          string         `mapstructure:"base_url" yaml:"base_url,omitempty" json:"base_url,omitempty"`
}

// VectorStoreConfig defines the configuration for a vector store provider.
type VectorStoreConfig struct {
	ProviderSpecific map[string]any `mapstructure:"provider_specific,omitempty" yaml:"provider_specific,omitempty"`
	Name             string         `mapstructure:"name" yaml:"name" validate:"required"`
	Provider         string         `mapstructure:"provider" yaml:"provider" validate:"required"`
	ConnectionString string         `mapstructure:"connection_string,omitempty" yaml:"connection_string,omitempty"`
}

// Validate validates the AgentConfig struct.
func (c *AgentConfig) Validate() error {
	validate := validator.New()
	if err := validate.Struct(c); err != nil {
		return fmt.Errorf("agent config validation failed: %w", err)
	}
	return nil
}

// Validate validates the LLMProviderConfig struct.
func (c *LLMProviderConfig) Validate() error {
	validate := validator.New()
	if err := validate.Struct(c); err != nil {
		return fmt.Errorf("LLM provider config validation failed: %w", err)
	}
	return nil
}

// Validate validates the EmbeddingProviderConfig struct.
func (c *EmbeddingProviderConfig) Validate() error {
	validate := validator.New()
	if err := validate.Struct(c); err != nil {
		return fmt.Errorf("embedding provider config validation failed: %w", err)
	}
	return nil
}

// Validate validates the VectorStoreConfig struct.
func (c *VectorStoreConfig) Validate() error {
	validate := validator.New()
	if err := validate.Struct(c); err != nil {
		return fmt.Errorf("vector store config validation failed: %w", err)
	}
	return nil
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
		DefaultCallOptions: make(map[string]any),
		ProviderSpecific:   make(map[string]any),
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
		ProviderSpecific: make(map[string]any),
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
		ProviderSpecific: make(map[string]any),
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
	CustomValidationRules   map[string]any `yaml:"custom_validation_rules" json:"custom_validation_rules"`
	AllowedMessageTypes     []string       `yaml:"allowed_message_types" json:"allowed_message_types"`
	RequiredMetadataFields  []string       `yaml:"required_metadata_fields" json:"required_metadata_fields"`
	MaxMessageLength        int            `yaml:"max_message_length" json:"max_message_length" validate:"min=1"`
	MaxMetadataSize         int            `yaml:"max_metadata_size" json:"max_metadata_size" validate:"min=1"`
	MaxToolCalls            int            `yaml:"max_tool_calls" json:"max_tool_calls" validate:"min=1"`
	MaxEmbeddingDimensions  int            `yaml:"max_embedding_dimensions" json:"max_embedding_dimensions" validate:"min=1"`
	EnableStrictValidation  bool           `yaml:"enable_strict_validation" json:"enable_strict_validation"`
	EnableContentValidation bool           `yaml:"enable_content_validation" json:"enable_content_validation"`
}

// Validate validates the SchemaValidationConfig struct.
func (c *SchemaValidationConfig) Validate() error {
	validate := validator.New()
	if err := validate.Struct(c); err != nil {
		return fmt.Errorf("schema validation config validation failed: %w", err)
	}
	return nil
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
		CustomValidationRules:   make(map[string]any),
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
func WithDefaultCallOptions(options map[string]any) LLMProviderOption {
	return func(c *LLMProviderConfig) {
		c.DefaultCallOptions = options
	}
}

// WithProviderSpecific sets provider-specific configuration.
func WithProviderSpecific(specific map[string]any) LLMProviderOption {
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
func WithEmbeddingProviderSpecific(specific map[string]any) EmbeddingOption {
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
func WithVectorStoreProviderSpecific(specific map[string]any) VectorStoreOption {
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
