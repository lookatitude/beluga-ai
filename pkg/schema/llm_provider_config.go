package schema

// LLMProviderConfig defines the configuration for a specific LLM provider instance.
// It allows for common parameters and a flexible way to include provider-specific settings.
type LLMProviderConfig struct {
	Name string `yaml:"name" json:"name"` // Unique name for this configuration instance (e.g., "openai_gpt4_turbo", "anthropic_claude3_opus")
	// Provider identifies the type of LLM provider (e.g., "openai", "anthropic", "gemini", "ollama").
	// This will be used by the LLMProviderFactory to instantiate the correct client.
	Provider string `yaml:"provider" json:"provider"`

	// ModelName specifies the exact model to be used (e.g., "gpt-4-turbo-preview", "claude-3-opus-20240229").
	ModelName string `yaml:"model_name" json:"model_name"`

	// APIKey is the API key for the LLM provider, if required.
	// It is recommended to manage this securely, e.g., via environment variables or a secrets manager,
	// and have the configuration loader resolve it.
	APIKey string `yaml:"api_key,omitempty" json:"api_key,omitempty"`

	// BaseURL can be used to specify a custom API endpoint, e.g., for self-hosted models or proxies.
	BaseURL string `yaml:"base_url,omitempty" json:"base_url,omitempty"`

	// DefaultCallOptions holds common LLM call parameters that can be overridden at runtime.
	DefaultCallOptions map[string]interface{} `yaml:"default_call_options,omitempty" json:"default_call_options,omitempty"`
	// Example DefaultCallOptions:
	// "temperature": 0.7
	// "max_tokens": 1024
	// "top_p": 1.0

	// ProviderSpecific holds any additional configuration parameters unique to the LLM provider.
	// This allows for flexibility in supporting diverse provider APIs without cluttering the main struct.
	// For example, for Ollama, this might include "keep_alive" or "num_ctx".
	// For OpenAI, it might include "organization_id".
	ProviderSpecific map[string]interface{} `yaml:"provider_specific,omitempty" json:"provider_specific,omitempty"`
}




// LLMOption defines a function type for LLM call options.
// It allows for flexible configuration of LLM calls.
type LLMOption func(options *CallOptions)

// CallOptions holds parameters for an LLM call.
// This struct can be expanded with more common options.
type CallOptions struct {
	Temperature      *float64
	MaxTokens        *int
	TopP             *float64
	FrequencyPenalty *float64
	PresencePenalty  *float64
	Stop             []string
	Streaming        bool
	// ProviderSpecificArgs allows for passing through any other provider-specific options.
	ProviderSpecificArgs map[string]interface{}
}

// NewCallOptions creates a new CallOptions with default values.
func NewCallOptions() *CallOptions {
	return &CallOptions{
		ProviderSpecificArgs: make(map[string]interface{}),
	}
}

// WithTemperature sets the temperature for the LLM call.
func WithTemperature(temp float64) LLMOption {
	return func(o *CallOptions) {
		o.Temperature = &temp
	}
}

// WithMaxTokens sets the max tokens for the LLM call.
func WithMaxTokens(maxTokens int) LLMOption {
	return func(o *CallOptions) {
		o.MaxTokens = &maxTokens
	}
}

// WithTopP sets the TopP for the LLM call.
func WithTopP(topP float64) LLMOption {
	return func(o *CallOptions) {
		o.TopP = &topP
	}
}

// WithFrequencyPenalty sets the frequency penalty for the LLM call.
func WithFrequencyPenalty(penalty float64) LLMOption {
	return func(o *CallOptions) {
		o.FrequencyPenalty = &penalty
	}
}

// WithPresencePenalty sets the presence penalty for the LLM call.
func WithPresencePenalty(penalty float64) LLMOption {
	return func(o *CallOptions) {
		o.PresencePenalty = &penalty
	}
}

// WithStopSequences sets the stop sequences for the LLM call.
func WithStopSequences(stop []string) LLMOption {
	return func(o *CallOptions) {
		o.Stop = stop
	}
}

// WithStreaming enables or disables streaming for the LLM call.
func WithStreaming(streaming bool) LLMOption {
	return func(o *CallOptions) {
		o.Streaming = streaming
	}
}

// WithProviderSpecificArg adds a provider-specific argument.
func WithProviderSpecificArg(key string, value interface{}) LLMOption {
	return func(o *CallOptions) {
		if o.ProviderSpecificArgs == nil {
			o.ProviderSpecificArgs = make(map[string]interface{})
		}
		o.ProviderSpecificArgs[key] = value
	}
}

