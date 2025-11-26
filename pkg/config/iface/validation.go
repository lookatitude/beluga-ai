package iface

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// ValidateConfig validates the entire configuration structure.
func ValidateConfig(cfg *Config) error {
	var errs ValidationErrors

	// Validate LLM providers
	for i, provider := range cfg.LLMProviders {
		if err := validateLLMProvider(provider); err != nil {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("llm_providers[%d]", i),
				Message: err.Error(),
			})
		}
	}

	// Validate embedding providers
	for i, provider := range cfg.EmbeddingProviders {
		if err := validateEmbeddingProvider(provider); err != nil {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("embedding_providers[%d]", i),
				Message: err.Error(),
			})
		}
	}

	// Validate vector stores
	for i, store := range cfg.VectorStores {
		if err := validateVectorStore(store); err != nil {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("vector_stores[%d]", i),
				Message: err.Error(),
			})
		}
	}

	// Validate tools
	for i, tool := range cfg.Tools {
		if err := validateTool(tool); err != nil {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("tools[%d]", i),
				Message: err.Error(),
			})
		}
	}

	// Validate agents
	for i, agent := range cfg.Agents {
		if err := validateAgent(agent); err != nil {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("agents[%d]", i),
				Message: err.Error(),
			})
		}
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

func validateLLMProvider(provider schema.LLMProviderConfig) error {
	return provider.Validate()
}

func validateEmbeddingProvider(provider schema.EmbeddingProviderConfig) error {
	return provider.Validate()
}

func validateVectorStore(store schema.VectorStoreConfig) error {
	return store.Validate()
}

func validateTool(tool ToolConfig) error {
	if tool.Name == "" {
		return errors.New("name is required")
	}
	if tool.Provider == "" {
		return errors.New("provider is required")
	}
	return nil
}

func validateAgent(agent schema.AgentConfig) error {
	return agent.Validate()
}

// SetDefaults sets default values for configuration fields.
func SetDefaults(cfg *Config) {
	// Set default values for LLM providers
	for i := range cfg.LLMProviders {
		setLLMProviderDefaults(&cfg.LLMProviders[i])
	}

	// Set default values for embedding providers
	for i := range cfg.EmbeddingProviders {
		setEmbeddingProviderDefaults(&cfg.EmbeddingProviders[i])
	}
}

func setLLMProviderDefaults(provider *schema.LLMProviderConfig) {
	if provider.DefaultCallOptions == nil {
		provider.DefaultCallOptions = make(map[string]any)
	}

	// Set default values in DefaultCallOptions map
	if _, exists := provider.DefaultCallOptions["temperature"]; !exists {
		provider.DefaultCallOptions["temperature"] = 0.7
	}
	if _, exists := provider.DefaultCallOptions["max_tokens"]; !exists {
		provider.DefaultCallOptions["max_tokens"] = 1000
	}
}

func setEmbeddingProviderDefaults(provider *schema.EmbeddingProviderConfig) {
	// Embedding providers don't have a direct timeout field in the schema
	// The timeout would be handled at the provider implementation level
	// or through ProviderSpecific configuration
}

// ValidateProvider validates a configuration provider.
func ValidateProvider(provider Provider) error {
	// Try to load the main configuration and validate it
	var cfg Config
	if err := provider.Load(&cfg); err != nil {
		return fmt.Errorf("failed to load config for validation: %w", err)
	}
	return ValidateConfig(&cfg)
}

// IsRequiredField checks if a struct field has a "required" tag.
func IsRequiredField(field reflect.StructField) bool {
	tag := field.Tag.Get("validate")
	return strings.Contains(tag, "required")
}

// GetFieldName gets the field name from struct tags or field name.
func GetFieldName(field reflect.StructField) string {
	if jsonTag := field.Tag.Get("json"); jsonTag != "" {
		if commaIdx := strings.Index(jsonTag, ","); commaIdx != -1 {
			return jsonTag[:commaIdx]
		}
		return jsonTag
	}
	if yamlTag := field.Tag.Get("yaml"); yamlTag != "" {
		if commaIdx := strings.Index(yamlTag, ","); commaIdx != -1 {
			return yamlTag[:commaIdx]
		}
		return yamlTag
	}
	return field.Name
}
