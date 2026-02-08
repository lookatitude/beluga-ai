package config

import "time"

// ProviderConfig holds common configuration for any external provider
// (LLM, embedding, vector store, etc.). Provider-specific options live
// in the Options map.
//
// Example JSON:
//
//	{
//	  "provider": "openai",
//	  "api_key": "sk-...",
//	  "model": "gpt-4o",
//	  "base_url": "https://api.openai.com/v1",
//	  "timeout": 30000000000,
//	  "options": {"temperature": 0.7}
//	}
type ProviderConfig struct {
	// Provider is the registered provider name (e.g. "openai", "anthropic").
	Provider string `json:"provider" required:"true"`

	// APIKey is the authentication key for the provider.
	APIKey string `json:"api_key"`

	// Model is the model identifier (e.g. "gpt-4o", "claude-sonnet-4-5-20250514").
	Model string `json:"model"`

	// BaseURL overrides the default API endpoint.
	BaseURL string `json:"base_url"`

	// Timeout is the maximum duration for a single request.
	Timeout time.Duration `json:"timeout" default:"30000000000"`

	// Options holds provider-specific key-value configuration.
	Options map[string]any `json:"options"`
}

// GetOption retrieves a typed value from the provider's Options map.
// It returns the value and true if the key exists and the type assertion
// succeeds, or the zero value of T and false otherwise.
//
// Usage:
//
//	temp, ok := config.GetOption[float64](cfg, "temperature")
func GetOption[T any](cfg ProviderConfig, key string) (T, bool) {
	var zero T
	if cfg.Options == nil {
		return zero, false
	}
	v, ok := cfg.Options[key]
	if !ok {
		return zero, false
	}
	typed, ok := v.(T)
	if !ok {
		return zero, false
	}
	return typed, true
}
